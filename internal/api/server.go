package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/xcode-ai/xgent-go/internal/api/handlers"
	"github.com/xcode-ai/xgent-go/internal/api/middleware"
	"github.com/xcode-ai/xgent-go/internal/orchestrator"
	"github.com/xcode-ai/xgent-go/internal/services/attachment"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"go.uber.org/zap"
)

// Server represents the API server
type Server struct {
	router       *gin.Engine
	httpServer   *http.Server
	storage      *storage.Storage
	orchestrator *orchestrator.Orchestrator
	logger       *zap.Logger
	config       *Config
}

// Config contains server configuration
type Config struct {
	Host         string
	Port         int
	Mode         string // debug, release
	JWTSecret    string
	AllowOrigins []string
}

// NewServer creates a new API server
func NewServer(cfg *Config, storage *storage.Storage, orch *orchestrator.Orchestrator, logger *zap.Logger) *Server {
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	server := &Server{
		router:       router,
		storage:      storage,
		orchestrator: orch,
		logger:       logger,
		config:       cfg,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware configures global middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(middleware.Logger(s.logger))

	// CORS middleware
	corsConfig := cors.Config{
		AllowOrigins:     s.config.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	s.router.Use(cors.New(corsConfig))

	// Request ID middleware
	s.router.Use(middleware.RequestID())
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Root endpoint
	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "Xgent-Go API",
			"version": "1.0.0",
			"docs":    "/api/v1",
			"health":  "/health",
		})
	})

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Public routes
		auth := v1.Group("/auth")
		{
			authHandler := handlers.NewAuthHandler(s.storage, s.config.JWTSecret, s.logger)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.Auth(s.config.JWTSecret))
		{
			// Workspaces
			workspaceHandler := handlers.NewWorkspaceHandler(s.storage, s.logger)
			workspaces := protected.Group("/workspaces")
			{
				workspaces.GET("", workspaceHandler.List)
				workspaces.POST("", workspaceHandler.Create)
				workspaces.GET("/:id", workspaceHandler.Get)
				workspaces.PUT("/:id", workspaceHandler.Update)
				workspaces.DELETE("/:id", workspaceHandler.Delete)
			}

			// Resources (CRD)
			resourceHandler := handlers.NewResourceHandler(s.storage, s.logger)
			resources := protected.Group("/resources")
			{
				resources.GET("", resourceHandler.List)
				resources.POST("", resourceHandler.Create)
				resources.GET("/:id", resourceHandler.Get)
				resources.PUT("/:id", resourceHandler.Update)
				resources.DELETE("/:id", resourceHandler.Delete)
				resources.POST("/apply", resourceHandler.Apply) // Apply YAML
			}

			// Tasks
			taskHandler := handlers.NewTaskHandler(s.storage, s.orchestrator, s.logger)
			tasks := protected.Group("/tasks")
			{
				tasks.POST("", taskHandler.Create)
				tasks.GET("", taskHandler.List)
				tasks.GET("/:id", taskHandler.Get)
				tasks.DELETE("/:id", taskHandler.Delete)
				tasks.POST("/:id/cancel", taskHandler.Cancel)
				tasks.GET("/:id/logs", taskHandler.GetLogs)
				tasks.GET("/:id/stream", taskHandler.Stream)
			}

			// Subtasks
			subtaskHandler := handlers.NewSubtaskHandler(s.storage, s.logger)
			tasks.GET("/:id/subtasks", subtaskHandler.ListByTask)
			subtasks := protected.Group("/subtasks")
			{
				subtasks.GET("/:id", subtaskHandler.Get)
				subtasks.PATCH("/:id/status", subtaskHandler.UpdateStatus)
				subtasks.GET("/:id/logs", subtaskHandler.GetLogs)
			}

			// Bots
			botHandler := handlers.NewBotHandler(s.storage, s.logger)
			bots := protected.Group("/bots")
			{
				bots.GET("", botHandler.List)
				bots.GET("/:name", botHandler.Get)
			}

			// Teams
			teamHandler := handlers.NewTeamHandler(s.storage, s.logger)
			teams := protected.Group("/teams")
			{
				teams.GET("", teamHandler.List)
				teams.GET("/:name", teamHandler.Get)
			}

			// Sessions
			sessionHandler := handlers.NewSessionHandler(s.storage, s.logger)
			sessions := protected.Group("/sessions")
			{
				sessions.GET("", sessionHandler.List)
				sessions.GET("/:id", sessionHandler.Get)
				sessions.DELETE("/:id", sessionHandler.Delete)
				sessions.GET("/:id/messages", sessionHandler.GetMessages)
			}

			// Attachments
			attachmentService := attachment.NewService(s.storage, "/tmp/xgent-uploads", s.logger)
			attachmentHandler := handlers.NewAttachmentHandler(s.storage, attachmentService, s.logger)
			attachments := protected.Group("/attachments")
			{
				attachments.POST("/upload", attachmentHandler.Upload)
				attachments.GET("", attachmentHandler.List)
				attachments.GET("/:id", attachmentHandler.Get)
				attachments.GET("/:id/download", attachmentHandler.Download)
				attachments.GET("/:id/content", attachmentHandler.GetContent)
				attachments.DELETE("/:id", attachmentHandler.Delete)
				attachments.POST("/:id/attach", attachmentHandler.AttachToTask)
			}
		}
	}

	// Swagger documentation (optional)
	// s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.logger.Info("Starting API server", zap.String("addr", addr))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping API server")

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}
