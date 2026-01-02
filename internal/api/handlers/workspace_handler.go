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

// WorkspaceHandler handles workspace-related requests
type WorkspaceHandler struct {
	storage *storage.Storage
	logger  *zap.Logger
}

// NewWorkspaceHandler creates a new workspace handler
func NewWorkspaceHandler(storage *storage.Storage, logger *zap.Logger) *WorkspaceHandler {
	return &WorkspaceHandler{
		storage: storage,
		logger:  logger,
	}
}

// CreateWorkspaceRequest represents workspace creation request
type CreateWorkspaceRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
}

// Create creates a new workspace
func (h *WorkspaceHandler) Create(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspace := &models.Workspace{
		Name:        req.Name,
		Description: req.Description,
		UserID:      userID,
	}

	if err := h.storage.Workspaces().Create(workspace); err != nil {
		h.logger.Error("Failed to create workspace", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workspace"})
		return
	}

	c.JSON(http.StatusCreated, workspace)
}

// Get retrieves a workspace by ID
func (h *WorkspaceHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	workspace, err := h.storage.Workspaces().GetByID(uint(workspaceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
		return
	}

	// Check ownership
	if workspace.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, workspace)
}

// List retrieves workspaces for the current user
func (h *WorkspaceHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	workspaces, err := h.storage.Workspaces().ListByUser(userID)
	if err != nil {
		h.logger.Error("Failed to list workspaces", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workspaces"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspaces": workspaces})
}

// Update updates a workspace
func (h *WorkspaceHandler) Update(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	workspace, err := h.storage.Workspaces().GetByID(uint(workspaceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
		return
	}

	// Check ownership
	if workspace.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspace.Name = req.Name
	workspace.Description = req.Description

	if err := h.storage.Workspaces().Update(workspace); err != nil {
		h.logger.Error("Failed to update workspace", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workspace"})
		return
	}

	c.JSON(http.StatusOK, workspace)
}

// Delete deletes a workspace
func (h *WorkspaceHandler) Delete(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	workspace, err := h.storage.Workspaces().GetByID(uint(workspaceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
		return
	}

	// Check ownership
	if workspace.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.storage.Workspaces().Delete(uint(workspaceID)); err != nil {
		h.logger.Error("Failed to delete workspace", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete workspace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workspace deleted"})
}
