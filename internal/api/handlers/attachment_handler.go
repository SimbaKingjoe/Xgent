package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/services/attachment"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"go.uber.org/zap"
)

// AttachmentHandler handles attachment-related requests
type AttachmentHandler struct {
	storage           *storage.Storage
	attachmentService *attachment.Service
	logger            *zap.Logger
}

// NewAttachmentHandler creates a new attachment handler
func NewAttachmentHandler(storage *storage.Storage, attachmentService *attachment.Service, logger *zap.Logger) *AttachmentHandler {
	return &AttachmentHandler{
		storage:           storage,
		attachmentService: attachmentService,
		logger:            logger,
	}
}

// Upload handles file upload
func (h *AttachmentHandler) Upload(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Upload file
	attachment, err := h.attachmentService.Upload(file, userID)
	if err != nil {
		h.logger.Error("Failed to upload file", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, attachment)
}

// Get retrieves an attachment by ID
func (h *AttachmentHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	attachment, err := h.storage.Attachments().GetByID(uint(attachmentID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}

	// Check ownership
	if attachment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, attachment)
}

// Download downloads an attachment file
func (h *AttachmentHandler) Download(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	data, filename, err := h.attachmentService.GetFile(uint(attachmentID), userID)
	if err != nil {
		h.logger.Error("Failed to get file", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// GetContent retrieves extracted text content
func (h *AttachmentHandler) GetContent(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	attachment, err := h.storage.Attachments().GetByID(uint(attachmentID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}

	// Check ownership
	if attachment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           attachment.ID,
		"filename":     attachment.Filename,
		"text_content": attachment.TextContent,
		"text_length":  attachment.TextLength,
		"status":       attachment.Status,
	})
}

// List retrieves attachments for the current user
func (h *AttachmentHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	limit := 50
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

	attachments, err := h.storage.Attachments().ListByUser(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list attachments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list attachments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attachments": attachments,
		"limit":       limit,
		"offset":      offset,
	})
}

// Delete deletes an attachment
func (h *AttachmentHandler) Delete(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	if err := h.attachmentService.Delete(uint(attachmentID), userID); err != nil {
		h.logger.Error("Failed to delete attachment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attachment deleted"})
}

// AttachToTask attaches a file to a task
func (h *AttachmentHandler) AttachToTask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	var req struct {
		TaskID uint `json:"task_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.attachmentService.AttachToTask(uint(attachmentID), req.TaskID, userID); err != nil {
		h.logger.Error("Failed to attach to task", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attachment linked to task"})
}
