package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/xcode-ai/xgent-go/internal/storage/models"
)

// TaskQueue manages task queuing and distribution
type TaskQueue struct {
	tasks   chan *TaskItem
	workers int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	active  map[uint]*TaskItem
}

// TaskItem wraps a task with execution context
type TaskItem struct {
	Task     *models.Task
	Context  context.Context
	Callback ProgressCallback
}

// ProgressCallback is an alias for models.ProgressCallback
type ProgressCallback = models.ProgressCallback

// NewTaskQueue creates a new task queue
func NewTaskQueue(workers int) *TaskQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskQueue{
		tasks:   make(chan *TaskItem, 100),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
		active:  make(map[uint]*TaskItem),
	}
}

// Start starts the task queue workers
func (q *TaskQueue) Start(executor TaskExecutor) {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i, executor)
	}
}

// Stop stops the task queue
func (q *TaskQueue) Stop() {
	q.cancel()
	close(q.tasks)
	q.wg.Wait()
}

// Enqueue adds a task to the queue
func (q *TaskQueue) Enqueue(task *models.Task, callback ProgressCallback) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	item := &TaskItem{
		Task:     task,
		Context:  q.ctx,
		Callback: callback,
	}

	select {
	case q.tasks <- item:
		q.active[task.ID] = item
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// Cancel cancels a task
func (q *TaskQueue) Cancel(taskID uint) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	item, exists := q.active[taskID]
	if !exists {
		return fmt.Errorf("task not found: %d", taskID)
	}

	// Cancel the task context (if we implement per-task context)
	// For now, just mark as cancelled in callback
	if item.Callback != nil {
		item.Callback(taskID, 0, models.TaskStatusCancelled, "Task cancelled by user", nil)
	}

	delete(q.active, taskID)
	return nil
}

// GetActive returns all active tasks
func (q *TaskQueue) GetActive() []*TaskItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	items := make([]*TaskItem, 0, len(q.active))
	for _, item := range q.active {
		items = append(items, item)
	}
	return items
}

// worker processes tasks from the queue
func (q *TaskQueue) worker(id int, executor TaskExecutor) {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case item, ok := <-q.tasks:
			if !ok {
				return
			}
			q.processTask(id, item, executor)
		}
	}
}

// processTask executes a single task
func (q *TaskQueue) processTask(workerID int, item *TaskItem, executor TaskExecutor) {
	defer func() {
		q.mu.Lock()
		delete(q.active, item.Task.ID)
		q.mu.Unlock()

		if r := recover(); r != nil {
			if item.Callback != nil {
				item.Callback(item.Task.ID, 0, models.TaskStatusFailed, 
					fmt.Sprintf("panic: %v", r), nil)
			}
		}
	}()

	// Execute task
	err := executor.Execute(item.Context, item.Task, item.Callback)
	if err != nil {
		if item.Callback != nil {
			item.Callback(item.Task.ID, 0, models.TaskStatusFailed, 
				fmt.Sprintf("execution failed: %v", err), nil)
		}
	}
}

// TaskExecutor defines the interface for task execution
type TaskExecutor interface {
	Execute(ctx context.Context, task *models.Task, callback ProgressCallback) error
}
