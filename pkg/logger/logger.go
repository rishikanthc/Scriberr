package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with convenience methods
type Logger struct {
	*slog.Logger
}

// LogLevel represents logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	// Default logger instance
	defaultLogger *Logger
	// Current log level
	currentLevel = LevelInfo
)

// Init initializes the global logger with specified level
func Init(level string) {
	// Parse log level from environment or parameter
	switch strings.ToLower(level) {
	case "debug":
		currentLevel = LevelDebug
	case "info", "":
		currentLevel = LevelInfo
	case "warn", "warning":
		currentLevel = LevelWarn
	case "error":
		currentLevel = LevelError
	default:
		currentLevel = LevelInfo
	}

	// Configure slog level
	var slogLevel slog.Level
	switch currentLevel {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	}

	// Create handler with optimized settings
	opts := &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: currentLevel == LevelDebug, // Only add source in debug mode
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Optimize timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format("15:04:05.000")),
				}
			}
			return a
		},
	}

	// Use text handler for better performance than JSON in most cases
	handler := slog.NewTextHandler(os.Stdout, opts)
	defaultLogger = &Logger{slog.New(handler)}
}

// Get returns the default logger instance
func Get() *Logger {
	if defaultLogger == nil {
		Init(os.Getenv("LOG_LEVEL"))
	}
	return defaultLogger
}

// Convenience methods for common logging patterns

func Debug(msg string, args ...any) {
	if currentLevel <= LevelDebug {
		Get().Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if currentLevel <= LevelInfo {
		Get().Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if currentLevel <= LevelWarn {
		Get().Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if currentLevel <= LevelError {
		Get().Error(msg, args...)
	}
}

// WithContext creates a logger with additional context
func WithContext(key string, value any) *Logger {
	return &Logger{Get().With(key, value)}
}

// Performance optimized logging for hot paths
func DebugIf(condition bool, msg string, args ...any) {
	if condition && currentLevel <= LevelDebug {
		Get().Debug(msg, args...)
	}
}

func InfoIf(condition bool, msg string, args ...any) {
	if condition && currentLevel <= LevelInfo {
		Get().Info(msg, args...)
	}
}

// Database query logger for development
func QueryDebug(query string, duration float64, args ...any) {
	if currentLevel <= LevelDebug {
		Get().Debug("database query",
			"query", query,
			"duration_ms", duration,
			"args", args)
	}
}

// HTTP request logger
func HTTPInfo(method, path string, status int, duration float64) {
	if currentLevel <= LevelInfo {
		Get().Info("http request",
			"method", method,
			"path", path,
			"status", status,
			"duration_ms", duration)
	}
}

// Worker operation logger
func WorkerInfo(workerID int, jobID string, operation string, args ...any) {
	if currentLevel <= LevelInfo {
		logger := Get().With("worker_id", workerID, "job_id", jobID, "operation", operation)
		logger.Info("worker operation", args...)
	}
}