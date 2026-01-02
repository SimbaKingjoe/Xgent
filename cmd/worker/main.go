package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
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

	log.Info("Xgent-Go worker started",
		zap.Int("workers", cfg.Orchestrator.Workers),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down worker...")
	log.Info("Worker exited")
}

// AppConfig represents application configuration
type AppConfig struct {
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
}

func loadConfig() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("database.driver", "mysql")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("orchestrator.workers", 10)
	viper.SetDefault("orchestrator.queue_size", 100)
	viper.SetDefault("orchestrator.workspace_dir", "/tmp/xgent-workspaces")

	// Read environment variables
	viper.AutomaticEnv()

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
