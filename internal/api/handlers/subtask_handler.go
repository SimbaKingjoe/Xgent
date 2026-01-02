package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// SubtaskHandler handles subtask-related requests
type SubtaskHandler struct {
	storage *storage.Storage
	logger  *zap.Logger
}

// NewSubtaskHandler creates a new subtask handler
func NewSubtaskHandler(storage *storage.Storage, logger *zap.Logger) *SubtaskHandler {
	return &SubtaskHandler{
		storage: storage,
		logger:  logger,
	}
}

// ListByTask retrieves subtasks for a task
func (h *SubtaskHandler) ListByTask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Verify task ownership
	task, err := h.storage.Tasks().GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get subtasks (loaded via Preload in GetByID)
	c.JSON(http.StatusOK, gin.H{
		"subtasks": task.SubTasks,
		"total":    len(task.SubTasks),
	})
}

// Get retrieves a subtask by ID
func (h *SubtaskHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subtaskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	// Get subtask
	var subtask models.SubTask
	if err := h.storage.DB().First(&subtask, subtaskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subtask not found"})
		return
	}

	// Verify task ownership
	task, err := h.storage.Tasks().GetByID(subtask.TaskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, subtask)
}

// UpdateStatus updates subtask status
func (h *SubtaskHandler) UpdateStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subtaskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	var req struct {
		Status   models.TaskStatus `json:"status" binding:"required"`
		Progress int               `json:"progress"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get subtask
	var subtask models.SubTask
	if err := h.storage.DB().First(&subtask, subtaskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subtask not found"})
		return
	}

	// Verify task ownership
	task, err := h.storage.Tasks().GetByID(subtask.TaskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update subtask
	subtask.Status = req.Status
	if req.Progress > 0 {
		subtask.Progress = req.Progress
	}

	if err := h.storage.DB().Save(&subtask).Error; err != nil {
		h.logger.Error("Failed to update subtask", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subtask"})
		return
	}

	c.JSON(http.StatusOK, subtask)
}

// GetLogs retrieves logs for a subtask
func (h *SubtaskHandler) GetLogs(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subtaskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	// Get subtask
	var subtask models.SubTask
	if err := h.storage.DB().First(&subtask, subtaskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subtask not found"})
		return
	}

	// Verify task ownership
	task, err := h.storage.Tasks().GetByID(subtask.TaskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

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

	// Get logs for this subtask
	var logs []*models.TaskLog
	query := h.storage.DB().Where("task_id = ? AND subtask_id = ?", subtask.TaskID, subtaskID).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		h.logger.Error("Failed to get logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
