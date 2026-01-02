package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config contains logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	OutputPath string // stdout, stderr, or file path
	Format     string // json or console
}

// New creates a new logger
func New(cfg Config) (*zap.Logger, error) {
	// Set defaults
	if cfg.Format == "" {
		cfg.Format = "console"
	}
	if cfg.OutputPath == "" {
		cfg.OutputPath = "stdout"
	}

	// Parse level
	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Build logger config
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      cfg.Level == "debug",
		Encoding:         cfg.Format,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{cfg.OutputPath},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}
