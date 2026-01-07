package logging

import (
	"context"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger

// Config represents logging configuration
type Config struct {
	Level       string         `yaml:"level"`
	OutputPath  string         `yaml:"output_path"`
	Development bool           `yaml:"development"`
	Rotation    RotationConfig `yaml:"rotation"`
}

// RotationConfig represents log rotation settings
type RotationConfig struct {
	Enabled    bool `yaml:"enabled"`     // Enable log rotation
	MaxSize    int  `yaml:"max_size"`    // Max size in MB before rotation
	MaxBackups int  `yaml:"max_backups"` // Max number of old log files to keep
	MaxAge     int  `yaml:"max_age"`     // Max days to keep old log files
	Compress   bool `yaml:"compress"`    // Compress rotated log files
	LocalTime  bool `yaml:"local_time"`  // Use local time for rotation
}

// Init initializes the logging system
func Init(cfg *Config) error {
	var zapLevel zapcore.Level
	switch cfg.Level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	var config zap.Config
	if cfg.Development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Setup output paths with rotation support
	var outputPaths []string
	var errorOutputPaths []string

	if cfg.OutputPath == "stdout" || cfg.OutputPath == "stderr" {
		// Use stdout/stderr directly
		outputPaths = []string{cfg.OutputPath}
		errorOutputPaths = []string{cfg.OutputPath}
	} else {
		// File output with optional rotation
		// Ensure output directory exists
		logDir := filepath.Dir(cfg.OutputPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}

		if cfg.Rotation.Enabled {
			// Use lumberjack for rotation
			rotationWriter := &lumberjack.Logger{
				Filename:   cfg.OutputPath,
				MaxSize:    cfg.Rotation.MaxSize, // megabytes
				MaxBackups: cfg.Rotation.MaxBackups,
				MaxAge:     cfg.Rotation.MaxAge, // days
				Compress:   cfg.Rotation.Compress,
				LocalTime:  cfg.Rotation.LocalTime,
			}

			// Create a zapcore.WriteSyncer for the rotated writer
			writeSyncer := zapcore.AddSync(rotationWriter)
			config.OutputPaths = []string{}
			config.ErrorOutputPaths = []string{}

			// Build encoder
			var encoder zapcore.Encoder
			if cfg.Development {
				encoder = zapcore.NewConsoleEncoder(config.EncoderConfig)
			} else {
				encoder = zapcore.NewJSONEncoder(config.EncoderConfig)
			}

			// Create core with rotation
			core := zapcore.NewCore(encoder, writeSyncer, zapLevel)
			logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

			logger.Info("日志系统初始化完成（启用轮转）",
				zap.String("level", cfg.Level),
				zap.String("output", cfg.OutputPath),
				zap.Int("max_size_mb", cfg.Rotation.MaxSize),
				zap.Int("max_backups", cfg.Rotation.MaxBackups),
				zap.Int("max_age_days", cfg.Rotation.MaxAge),
				zap.Bool("compress", cfg.Rotation.Compress))

			return nil
		} else {
			// Regular file output without rotation
			outputPaths = []string{cfg.OutputPath}
			errorOutputPaths = []string{cfg.OutputPath}

			// Ensure output directory exists
			logDir := filepath.Dir(cfg.OutputPath)
			if err := os.MkdirAll(logDir, 0755); err != nil {
				return err
			}
		}
	}

	config.OutputPaths = outputPaths
	config.ErrorOutputPaths = errorOutputPaths

	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}

	logger.Info("日志系统初始化完成",
		zap.String("level", cfg.Level),
		zap.String("output", cfg.OutputPath))

	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if logger == nil {
		// Return a no-op logger if not initialized
		return zap.NewNop()
	}
	return logger
}

// WithComponent creates a logger with component name
func WithComponent(component string) *zap.Logger {
	return GetLogger().With(zap.String("component", component))
}

// WithContext creates a logger with context fields
func WithContext(ctx context.Context, fields ...zap.Field) *zap.Logger {
	l := GetLogger()

	// Extract tracing information from context
	if questionID := ctx.Value("question_id"); questionID != nil {
		fields = append(fields, zap.String("question_id", questionID.(string)))
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields = append(fields, zap.String("trace_id", traceID.(string)))
	}
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields = append(fields, zap.String("request_id", requestID.(string)))
	}

	return l.With(fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() {
	if logger != nil {
		logger.Sync()
	}
}
