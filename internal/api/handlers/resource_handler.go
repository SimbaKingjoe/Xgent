package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/crd"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// ResourceHandler handles CRD resource requests
type ResourceHandler struct {
	storage *storage.Storage
	logger  *zap.Logger
}

// NewResourceHandler creates a new resource handler
func NewResourceHandler(storage *storage.Storage, logger *zap.Logger) *ResourceHandler {
	return &ResourceHandler{
		storage: storage,
		logger:  logger,
	}
}

// CreateResourceRequest represents resource creation request
type CreateResourceRequest struct {
	WorkspaceID uint                `json:"workspace_id"`
	Type        models.ResourceType `json:"type" binding:"required"`
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	Spec        string              `json:"spec" binding:"required"`
}

// Create creates a new resource
func (h *ResourceHandler) Create(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateResourceRequest
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

	// Validate YAML spec
	parser := crd.NewParser()
	if _, err := parser.Parse([]byte(req.Spec)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid YAML spec: %v", err)})
		return
	}

	// Check if resource already exists
	exists, _ := h.storage.Resources().Exists(req.WorkspaceID, req.Name, req.Type)
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Resource already exists"})
		return
	}

	// Create resource
	resource := &models.Resource{
		WorkspaceID: req.WorkspaceID,
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		Spec:        req.Spec,
		Status:      "active",
	}

	if err := h.storage.Resources().Create(resource); err != nil {
		h.logger.Error("Failed to create resource", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create resource"})
		return
	}

	c.JSON(http.StatusCreated, resource)
}

// Get retrieves a resource by ID
func (h *ResourceHandler) Get(c *gin.Context) {
	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	resource, err := h.storage.Resources().GetByID(uint(resourceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	c.JSON(http.StatusOK, resource)
}

// List retrieves resources
func (h *ResourceHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Get workspace ID from query
	workspaceID := uint(0)
	if wsIDStr := c.Query("workspace_id"); wsIDStr != "" {
		if wsID, err := strconv.ParseUint(wsIDStr, 10, 32); err == nil {
			workspaceID = uint(wsID)
		}
	}

	// Use default workspace if not specified
	if workspaceID == 0 {
		workspaces, err := h.storage.Workspaces().ListByUser(userID)
		if err != nil || len(workspaces) == 0 {
			c.JSON(http.StatusOK, gin.H{"resources": []models.Resource{}})
			return
		}
		workspaceID = workspaces[0].ID
	}

	// Get resource type filter
	resourceType := models.ResourceType(c.Query("type"))

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

	resources, err := h.storage.Resources().List(workspaceID, resourceType, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list resources", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list resources"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resources": resources,
		"limit":     limit,
		"offset":    offset,
	})
}

// Update updates a resource
func (h *ResourceHandler) Update(c *gin.Context) {
	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	resource, err := h.storage.Resources().GetByID(uint(resourceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	var req CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate YAML spec
	parser := crd.NewParser()
	if _, err := parser.Parse([]byte(req.Spec)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid YAML spec: %v", err)})
		return
	}

	// Update fields
	if req.Name != "" {
		resource.Name = req.Name
	}
	if req.Description != "" {
		resource.Description = req.Description
	}
	if req.Spec != "" {
		resource.Spec = req.Spec
	}

	if err := h.storage.Resources().Update(resource); err != nil {
		h.logger.Error("Failed to update resource", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update resource"})
		return
	}

	c.JSON(http.StatusOK, resource)
}

// Delete deletes a resource
func (h *ResourceHandler) Delete(c *gin.Context) {
	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	if err := h.storage.Resources().Delete(uint(resourceID)); err != nil {
		h.logger.Error("Failed to delete resource", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete resource"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Resource deleted"})
}

// Apply applies resources from YAML
func (h *ResourceHandler) Apply(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Get workspace ID from query
	workspaceID := uint(0)
	if wsIDStr := c.Query("workspace_id"); wsIDStr != "" {
		if wsID, err := strconv.ParseUint(wsIDStr, 10, 32); err == nil {
			workspaceID = uint(wsID)
		}
	}

	// Use default workspace if not specified
	if workspaceID == 0 {
		workspaces, err := h.storage.Workspaces().ListByUser(userID)
		if err != nil || len(workspaces) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No workspace found"})
			return
		}
		workspaceID = workspaces[0].ID
	}

	// Read YAML content
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Parse YAML
	parser := crd.NewParser()
	resource, err := parser.Parse(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse YAML: %v", err)})
		return
	}

	// Map CRD kind to resource type
	var resourceType models.ResourceType
	switch resource.GetKind() {
	case crd.KindSoul:
		resourceType = models.ResourceTypeSoul
	case crd.KindMind:
		resourceType = models.ResourceTypeMind
	case crd.KindCraft:
		resourceType = models.ResourceTypeCraft
	case crd.KindRobot:
		resourceType = models.ResourceTypeRobot
	case crd.KindTeam:
		resourceType = models.ResourceTypeTeam
	case crd.KindCollaboration:
		resourceType = models.ResourceTypeCollaboration
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown resource kind"})
		return
	}

	metadata := resource.GetMetadata()

	// Check if resource exists
	existingResource, err := h.storage.Resources().GetByName(workspaceID, metadata.Name, resourceType)
	if err == nil {
		// Update existing resource
		existingResource.Spec = string(body)
		existingResource.Description = metadata.Description
		if err := h.storage.Resources().Update(existingResource); err != nil {
			h.logger.Error("Failed to update resource", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update resource"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"action":   "updated",
			"resource": existingResource,
		})
		return
	}

	// Create new resource
	newResource := &models.Resource{
		WorkspaceID: workspaceID,
		Type:        resourceType,
		Name:        metadata.Name,
		Description: metadata.Description,
		Spec:        string(body),
		Status:      "active",
	}

	if err := h.storage.Resources().Create(newResource); err != nil {
		h.logger.Error("Failed to create resource", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create resource"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"action":   "created",
		"resource": newResource,
	})
}
