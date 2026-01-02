package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"go.uber.org/zap"
)

// SessionHandler handles session-related requests
type SessionHandler struct {
	storage *storage.Storage
	logger  *zap.Logger
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(storage *storage.Storage, logger *zap.Logger) *SessionHandler {
	return &SessionHandler{
		storage: storage,
		logger:  logger,
	}
}

// List retrieves sessions for the current user
func (h *SessionHandler) List(c *gin.Context) {
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

	sessions, err := h.storage.Sessions().ListByUser(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"limit":    limit,
		"offset":   offset,
	})
}

// Get retrieves a session by ID
func (h *SessionHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	sessionID := c.Param("id")

	session, err := h.storage.Sessions().GetBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Check ownership
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// Delete deletes a session
func (h *SessionHandler) Delete(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	sessionID := c.Param("id")

	session, err := h.storage.Sessions().GetBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Check ownership
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.storage.Sessions().Delete(sessionID); err != nil {
		h.logger.Error("Failed to delete session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted"})
}

// GetMessages retrieves messages for a session
func (h *SessionHandler) GetMessages(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	sessionID := c.Param("id")

	session, err := h.storage.Sessions().GetBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Check ownership
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	messages, err := h.storage.Sessions().GetMessages(sessionID, limit)
	if err != nil {
		h.logger.Error("Failed to get messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}
