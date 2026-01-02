package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/orchestrator"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// TaskHandler handles task-related requests
type TaskHandler struct {
	storage      *storage.Storage
	orchestrator *orchestrator.Orchestrator
	logger       *zap.Logger
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(storage *storage.Storage, orch *orchestrator.Orchestrator, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		storage:      storage,
		orchestrator: orch,
		logger:       logger,
	}
}

// CreateTaskRequest represents task creation request
type CreateTaskRequest struct {
	Title        string `json:"title" binding:"required"`
	Description  string `json:"description"`
	Prompt       string `json:"prompt" binding:"required"`
	ResourceType string `json:"resource_type" binding:"required,oneof=robot team"`
	ResourceName string `json:"resource_name" binding:"required"`
	Mode         string `json:"mode,omitempty"`
	GitURL       string `json:"git_url,omitempty"`
	BranchName   string `json:"branch_name,omitempty"`
	WorkspaceID  uint   `json:"workspace_id"`
}

// Create creates a new task
func (h *TaskHandler) Create(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use default workspace if not specified
	if req.WorkspaceID == 0 {
		workspaces, err := h.storage.Workspaces().ListByUser(userID)
		if err != nil || len(workspaces) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No workspace found"})
			return
		}
		req.WorkspaceID = workspaces[0].ID
	}

	// Create task
	task := &models.Task{
		WorkspaceID:  req.WorkspaceID,
		UserID:       userID,
		Status:       models.TaskStatusPending,
		Title:        req.Title,
		Description:  req.Description,
		Prompt:       req.Prompt,
		ResourceType: req.ResourceType,
		ResourceName: req.ResourceName,
		Mode:         req.Mode,
		GitURL:       req.GitURL,
		BranchName:   req.BranchName,
		Progress:     0,
	}

	if err := h.storage.Tasks().Create(task); err != nil {
		h.logger.Error("Failed to create task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Submit task to orchestrator
	callback := func(taskID uint, progress int, status models.TaskStatus, message string, metadata map[string]interface{}) {
		// Determine event type
		eventType := "info"
		if metadata != nil {
			if t, ok := metadata["type"].(string); ok {
				eventType = t
			}
		}

		// Extract details from metadata
		var details map[string]interface{}
		if metadata != nil {
			if d, ok := metadata["details"].(map[string]interface{}); ok {
				details = d
			}
		}

		// Broadcast event in real-time to WebSocket subscribers
		GetBroadcaster().Broadcast(TaskEvent{
			TaskID:    taskID,
			Type:      "log",
			EventType: eventType,
			Content:   message,
			Details:   details,
			Progress:  progress,
			Status:    string(status),
		})

		// Update task in database
		if t, err := h.storage.Tasks().GetByID(taskID); err == nil {
			t.Progress = progress
			t.Status = status
			h.storage.Tasks().Update(t)

			// Build JSON message with type, content, and details for frontend parsing
			logMessage := map[string]interface{}{
				"type":    eventType,
				"content": message,
			}
			if details != nil {
				logMessage["details"] = details
			}
			msgJSON, _ := json.Marshal(logMessage)

			// Add log entry with full event data
			h.storage.Tasks().AddLog(&models.TaskLog{
				TaskID:    taskID,
				Level:     "info",
				Message:   string(msgJSON),
				EventType: eventType,
			})
		}
	}

	if err := h.orchestrator.SubmitTask(task, callback); err != nil {
		h.logger.Error("Failed to submit task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// Get retrieves a task by ID
func (h *TaskHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.storage.Tasks().GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check ownership
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// List retrieves tasks for the current user
func (h *TaskHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	limit := 20
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	tasks, err := h.storage.Tasks().ListByUser(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  tasks,
		"limit":  limit,
		"offset": offset,
	})
}

// Delete deletes a task
func (h *TaskHandler) Delete(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.storage.Tasks().GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check ownership
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.storage.Tasks().Delete(uint(taskID)); err != nil {
		h.logger.Error("Failed to delete task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

// Cancel cancels a running task
func (h *TaskHandler) Cancel(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.storage.Tasks().GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check ownership
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Cancel task in orchestrator
	if err := h.orchestrator.CancelTask(uint(taskID)); err != nil {
		h.logger.Error("Failed to cancel task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel task"})
		return
	}

	// Update task status
	task.Status = models.TaskStatusCancelled
	h.storage.Tasks().Update(task)

	c.JSON(http.StatusOK, gin.H{"message": "Task cancelled"})
}

// GetLogs retrieves task logs
func (h *TaskHandler) GetLogs(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.storage.Tasks().GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check ownership
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	logs, err := h.storage.Tasks().GetLogs(uint(taskID), limit)
	if err != nil {
		h.logger.Error("Failed to get logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// Stream handles WebSocket streaming for task execution with real-time events
func (h *TaskHandler) Stream(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.storage.Tasks().GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check ownership
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket", zap.Error(err))
		return
	}
	defer conn.Close()

	h.logger.Info("WebSocket connection established",
		zap.Uint("task_id", uint(taskID)),
		zap.Uint("user_id", userID),
	)

	// Send initial task status
	conn.WriteJSON(gin.H{
		"type":     "status",
		"task_id":  task.ID,
		"status":   task.Status,
		"progress": task.Progress,
	})

	// Send a connection confirmation event so frontend knows WebSocket is working
	conn.WriteJSON(gin.H{
		"type":       "log",
		"task_id":    task.ID,
		"event_type": "connected",
		"message":    `{"type":"connected","content":"WebSocket 连接成功，等待推理事件..."}`,
	})

	h.logger.Info("Sent connection confirmation to client", zap.Uint("task_id", task.ID))

	// Subscribe to real-time events for this task
	eventCh := GetBroadcaster().Subscribe(uint(taskID))
	defer GetBroadcaster().Unsubscribe(uint(taskID), eventCh)

	// Also check task status periodically for completion
	statusTicker := time.NewTicker(2 * time.Second)
	defer statusTicker.Stop()

	// Handle WebSocket close
	done := make(chan struct{})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				close(done)
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			h.logger.Info("WebSocket client disconnected", zap.Uint64("task_id", taskID))
			return

		case event := <-eventCh:
			// Send real-time event immediately
			h.logger.Info("Sending event via WebSocket",
				zap.Uint("task_id", event.TaskID),
				zap.String("event_type", event.EventType),
				zap.String("content_preview", event.Content[:min(len(event.Content), 50)]),
			)

			msg := gin.H{
				"type":       event.Type,
				"task_id":    event.TaskID,
				"event_type": event.EventType,
				"progress":   event.Progress,
				"status":     event.Status,
			}

			// Build message field as JSON for frontend parsing
			logMessage := map[string]interface{}{
				"type":    event.EventType,
				"content": event.Content,
			}
			if event.Details != nil {
				logMessage["details"] = event.Details
			}
			msgJSON, _ := json.Marshal(logMessage)
			msg["message"] = string(msgJSON)

			if err := conn.WriteJSON(msg); err != nil {
				h.logger.Error("Failed to write WebSocket message", zap.Error(err))
				return
			}

		case <-statusTicker.C:
			// Periodically check task completion status
			updatedTask, _ := h.storage.Tasks().GetByID(uint(taskID))
			if updatedTask != nil {
				// Send status update
				conn.WriteJSON(gin.H{
					"type":     "status",
					"task_id":  updatedTask.ID,
					"status":   updatedTask.Status,
					"progress": updatedTask.Progress,
				})

				// Close connection if task is completed
				if updatedTask.Status == models.TaskStatusCompleted ||
					updatedTask.Status == models.TaskStatusFailed ||
					updatedTask.Status == models.TaskStatusCancelled {
					conn.WriteJSON(gin.H{
						"type":    "complete",
						"task_id": taskID,
						"status":  updatedTask.Status,
						"result":  updatedTask.Result,
					})
					return
				}
			}
		}
	}
}
