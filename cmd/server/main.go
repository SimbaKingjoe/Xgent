package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"github.com/xcode-ai/xgent-go/internal/api"
	"github.com/xcode-ai/xgent-go/internal/orchestrator"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:      "info",
		OutputPath: "stdout",
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer log.Sync()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	// Initialize storage
	store, err := storage.New(&storage.Config{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Database: cfg.Database.Database,
		Username: cfg.Database.Username,
		Password: cfg.Database.Password,
	}, log)
	if err != nil {
		log.Fatal("failed to initialize storage", zap.Error(err))
	}

	// Run migrations
	if err := store.AutoMigrate(); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}

	// Initialize orchestrator
	orch := orchestrator.New(&orchestrator.Config{
		Workers:      cfg.Orchestrator.Workers,
		QueueSize:    cfg.Orchestrator.QueueSize,
		WorkspaceDir: cfg.Orchestrator.WorkspaceDir,
	}, store, log)

	// Start orchestrator
	if err := orch.Start(); err != nil {
		log.Fatal("failed to start orchestrator", zap.Error(err))
	}
	defer orch.Stop()

	// Initialize API server
	server := api.NewServer(&api.Config{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		Mode:         cfg.Server.Mode,
		JWTSecret:    cfg.Server.JWTSecret,
		AllowOrigins: cfg.Server.AllowOrigins,
	}, store, orch, log)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	log.Info("Xgent-Go server started",
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Error("server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited")
}

// AppConfig represents application configuration
type AppConfig struct {
	Server struct {
		Host         string   `mapstructure:"host"`
		Port         int      `mapstructure:"port"`
		Mode         string   `mapstructure:"mode"`
		JWTSecret    string   `mapstructure:"jwt_secret"`
		AllowOrigins []string `mapstructure:"allow_origins"`
	} `mapstructure:"server"`

	Database struct {
		Driver   string `mapstructure:"driver"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Database string `mapstructure:"database"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`

	Orchestrator struct {
		Workers      int    `mapstructure:"workers"`
		QueueSize    int    `mapstructure:"queue_size"`
		WorkspaceDir string `mapstructure:"workspace_dir"`
	} `mapstructure:"orchestrator"`

	Agno struct {
		OpenAIKey    string `mapstructure:"openai_key"`
		AnthropicKey string `mapstructure:"anthropic_key"`
	} `mapstructure:"agno"`
}

func loadConfig() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.allow_origins", []string{"*"})
	viper.SetDefault("database.driver", "mysql")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("orchestrator.workers", 10)
	viper.SetDefault("orchestrator.queue_size", 100)
	viper.SetDefault("orchestrator.workspace_dir", "/tmp/xgent-workspaces")

	// Read environment variables
	viper.AutomaticEnv()

	// Bind environment variables for docker-compose compatibility
	viper.BindEnv("database.driver", "DATABASE_DRIVER")
	viper.BindEnv("database.host", "DATABASE_HOST")
	viper.BindEnv("database.port", "DATABASE_PORT")
	viper.BindEnv("database.database", "DATABASE_NAME")
	viper.BindEnv("database.username", "DATABASE_USER")
	viper.BindEnv("database.password", "DATABASE_PASSWORD")
	viper.BindEnv("server.jwt_secret", "JWT_SECRET")
	viper.BindEnv("agno.openai_key", "OPENAI_API_KEY")
	viper.BindEnv("agno.anthropic_key", "ANTHROPIC_API_KEY")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
