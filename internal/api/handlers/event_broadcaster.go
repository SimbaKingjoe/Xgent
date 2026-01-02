package handlers

import (
	"log"
	"sync"
	"time"
)

const (
	maxEventBuffer = 200 // Max events to buffer per task
	bufferTTL      = 10 * time.Minute
)

// TaskEvent represents a real-time task event
type TaskEvent struct {
	TaskID    uint                   `json:"task_id"`
	Type      string                 `json:"type"`
	Content   string                 `json:"content,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Progress  int                    `json:"progress,omitempty"`
	Status    string                 `json:"status,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// eventBuffer stores recent events for replay to late subscribers
type eventBuffer struct {
	events    []TaskEvent
	createdAt time.Time
}

// EventBroadcaster manages real-time event distribution to WebSocket clients
type EventBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[uint][]chan TaskEvent // taskID -> list of subscriber channels
	buffers     map[uint]*eventBuffer     // taskID -> buffered events for replay
}

// Global broadcaster instance
var globalBroadcaster = NewEventBroadcaster()

// GetBroadcaster returns the global event broadcaster
func GetBroadcaster() *EventBroadcaster {
	return globalBroadcaster
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster() *EventBroadcaster {
	b := &EventBroadcaster{
		subscribers: make(map[uint][]chan TaskEvent),
		buffers:     make(map[uint]*eventBuffer),
	}
	// Start cleanup goroutine
	go b.cleanupLoop()
	return b
}

// cleanupLoop periodically removes old event buffers
func (b *EventBroadcaster) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		for taskID, buf := range b.buffers {
			if now.Sub(buf.createdAt) > bufferTTL {
				delete(b.buffers, taskID)
			}
		}
		b.mu.Unlock()
	}
}

// Subscribe creates a new subscription channel for a task and replays buffered events
func (b *EventBroadcaster) Subscribe(taskID uint) chan TaskEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan TaskEvent, 200) // Buffer to prevent blocking
	b.subscribers[taskID] = append(b.subscribers[taskID], ch)

	// Replay buffered events to new subscriber (copy slice to avoid race)
	if buf, ok := b.buffers[taskID]; ok && len(buf.events) > 0 {
		eventsCopy := make([]TaskEvent, len(buf.events))
		copy(eventsCopy, buf.events)
		log.Printf("[Broadcaster] Replaying %d buffered events for task %d", len(eventsCopy), taskID)

		// Replay synchronously to ensure events are sent before returning
		for i, event := range eventsCopy {
			select {
			case ch <- event:
			default:
				log.Printf("[Broadcaster] Channel full during replay for task %d, skipped %d events", taskID, len(eventsCopy)-i)
				goto doneReplay
			}
		}
	doneReplay:
	} else {
		log.Printf("[Broadcaster] No buffered events for task %d", taskID)
	}

	log.Printf("[Broadcaster] New subscriber for task %d, total subscribers: %d", taskID, len(b.subscribers[taskID]))
	return ch
}

// Unsubscribe removes a subscription channel
func (b *EventBroadcaster) Unsubscribe(taskID uint, ch chan TaskEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[taskID]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(b.subscribers[taskID]) == 0 {
		delete(b.subscribers, taskID)
	}
}

// Broadcast sends an event to all subscribers and buffers it for late subscribers
func (b *EventBroadcaster) Broadcast(event TaskEvent) {
	event.Timestamp = time.Now()

	log.Printf("[Broadcaster] Event: taskID=%d, type=%s, eventType=%s, content=%s, subscribers=%d",
		event.TaskID, event.Type, event.EventType, event.Content[:min(len(event.Content), 50)], len(b.subscribers[event.TaskID]))

	b.mu.Lock()
	// Buffer the event for late subscribers
	buf, ok := b.buffers[event.TaskID]
	if !ok {
		buf = &eventBuffer{
			events:    make([]TaskEvent, 0, maxEventBuffer),
			createdAt: time.Now(),
		}
		b.buffers[event.TaskID] = buf
	}
	if len(buf.events) < maxEventBuffer {
		buf.events = append(buf.events, event)
	}

	// Get subscribers snapshot
	subs := make([]chan TaskEvent, len(b.subscribers[event.TaskID]))
	copy(subs, b.subscribers[event.TaskID])
	b.mu.Unlock()

	// Send to all subscribers
	for _, ch := range subs {
		select {
		case ch <- event:
			log.Printf("[Broadcaster] Sent event to subscriber for task %d", event.TaskID)
		default:
			log.Printf("[Broadcaster] Channel full, skipping event for task %d", event.TaskID)
		}
	}
}

// ClearBuffer removes buffered events for a task (call when task completes)
func (b *EventBroadcaster) ClearBuffer(taskID uint) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.buffers, taskID)
}

// BroadcastToTask is a convenience method
func BroadcastToTask(taskID uint, eventType, content string, details map[string]interface{}, progress int, status string) {
	GetBroadcaster().Broadcast(TaskEvent{
		TaskID:    taskID,
		Type:      "log",
		EventType: eventType,
		Content:   content,
		Details:   details,
		Progress:  progress,
		Status:    status,
	})
}
