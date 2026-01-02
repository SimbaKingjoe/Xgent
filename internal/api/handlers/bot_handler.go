package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// BotHandler handles bot-related requests
type BotHandler struct {
	storage *storage.Storage
	logger  *zap.Logger
}

// NewBotHandler creates a new bot handler
func NewBotHandler(storage *storage.Storage, logger *zap.Logger) *BotHandler {
	return &BotHandler{
		storage: storage,
		logger:  logger,
	}
}

// List retrieves all bots in the workspace
func (h *BotHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Get default workspace
	workspaces, err := h.storage.Workspaces().ListByUser(userID)
	if err != nil || len(workspaces) == 0 {
		c.JSON(http.StatusOK, gin.H{"bots": []models.Resource{}})
		return
	}

	// Get all bot resources
	bots, err := h.storage.Resources().List(workspaces[0].ID, models.ResourceTypeRobot, 100, 0)
	if err != nil {
		h.logger.Error("Failed to list bots", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list bots"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bots": bots})
}

// Get retrieves a bot by name
func (h *BotHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	botName := c.Param("name")

	// Get default workspace
	workspaces, err := h.storage.Workspaces().ListByUser(userID)
	if err != nil || len(workspaces) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No workspace found"})
		return
	}

	// Get bot resource
	bot, err := h.storage.Resources().GetByName(workspaces[0].ID, botName, models.ResourceTypeRobot)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
		return
	}

	c.JSON(http.StatusOK, bot)
}
