package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// TeamHandler handles team-related requests
type TeamHandler struct {
	storage *storage.Storage
	logger  *zap.Logger
}

// NewTeamHandler creates a new team handler
func NewTeamHandler(storage *storage.Storage, logger *zap.Logger) *TeamHandler {
	return &TeamHandler{
		storage: storage,
		logger:  logger,
	}
}

// List retrieves all teams in the workspace
func (h *TeamHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Get default workspace
	workspaces, err := h.storage.Workspaces().ListByUser(userID)
	if err != nil || len(workspaces) == 0 {
		c.JSON(http.StatusOK, gin.H{"teams": []models.Resource{}})
		return
	}

	// Get all team resources
	teams, err := h.storage.Resources().List(workspaces[0].ID, models.ResourceTypeTeam, 100, 0)
	if err != nil {
		h.logger.Error("Failed to list teams", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list teams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// Get retrieves a team by name
func (h *TeamHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	teamName := c.Param("name")

	// Get default workspace
	workspaces, err := h.storage.Workspaces().ListByUser(userID)
	if err != nil || len(workspaces) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No workspace found"})
		return
	}

	// Get team resource
	team, err := h.storage.Resources().GetByName(workspaces[0].ID, teamName, models.ResourceTypeTeam)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}

	c.JSON(http.StatusOK, team)
}
