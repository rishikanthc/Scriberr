package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
		AddSource: false, // Clean logs without source info
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Clean timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format("15:04:05")),
				}
			}
			// Clean level names
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				switch level {
				case slog.LevelDebug:
					a.Value = slog.StringValue("DEBUG")
				case slog.LevelInfo:
					a.Value = slog.StringValue("INFO ")
				case slog.LevelWarn:
					a.Value = slog.StringValue("WARN ")
				case slog.LevelError:
					a.Value = slog.StringValue("ERROR")
				}
			}
			return a
		},
	}

	// Use text handler for clean, readable output
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

// GetLevel returns the current log level
func GetLevel() LogLevel {
	return currentLevel
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

// Startup logging for key initialization steps
func Startup(step, message string, args ...any) {
	// Simple message at INFO level, technical details at DEBUG
	if currentLevel <= LevelInfo {
		// Clean, user-friendly startup message
		// \033[36m is Cyan color for the [+] prefix
		fmt.Printf("\033[36m[+]\033[0m %s\n", message)
	}
	if currentLevel <= LevelDebug {
		Debug("Startup step", append([]any{"step", step, "message", message}, args...)...)
	}
}

// Job logging for transcription operations
func JobStarted(jobID, filename, model string, params map[string]any) {
	// Simple message at INFO, details at DEBUG
	Info("Transcription started", "file", filename)
	Debug("Job started with details",
		"job_id", jobID,
		"file", filename,
		"model", model,
		"params", params)
}

func JobCompleted(jobID string, duration time.Duration, result any) {
	Info("Transcription completed", "duration", duration.String())
	Debug("Job completed with details",
		"job_id", jobID,
		"duration", duration.String(),
		"result", result)
}

func JobFailed(jobID string, duration time.Duration, err error) {
	Error("Transcription failed", "error", err.Error())
	Debug("Job failed with details",
		"job_id", jobID,
		"duration", duration.String(),
		"error", err.Error())
}

// HTTP request logging - filtered for INFO level
func HTTPRequest(method, path string, status int, duration time.Duration, userAgent string) {
	// Skip noisy endpoints at INFO level
	if currentLevel <= LevelInfo {
		switch path {
		case "/api/v1/transcription/list", "/health":
			// Skip logging frequent status checks at INFO level
			return
		}
		if strings.HasSuffix(path, "/status") || strings.HasSuffix(path, "/track-progress") {
			// Skip job status polling at INFO level
			return
		}
	}

	// Log all requests at DEBUG level
	if currentLevel <= LevelDebug {
		Debug("API request",
			"method", method,
			"path", path,
			"status", status,
			"duration", fmt.Sprintf("%.2fms", float64(duration.Nanoseconds())/1e6),
			"user_agent", userAgent)
	}
}

// Authentication logging
func AuthEvent(event, username, ip string, success bool, details ...any) {
	if success {
		Info("User login successful", "username", username)
		Debug("Auth event details",
			append([]any{"event", event, "username", username, "ip", ip, "success", success}, details...)...)
	} else {
		Info("User login failed", "username", username, "reason", "invalid_credentials")
		Debug("Auth event details",
			append([]any{"event", event, "username", username, "ip", ip, "success", success}, details...)...)
	}
}

// Worker operation logger
func WorkerOperation(workerID int, jobID string, operation string, args ...any) {
	Debug("Worker operation",
		append([]any{"worker_id", workerID, "job_id", jobID, "operation", operation}, args...)...)
}

// Performance logging for debugging
func Performance(operation string, duration time.Duration, details ...any) {
	Debug("Performance",
		append([]any{"operation", operation, "duration", duration.String()}, details...)...)
}

// GIN middleware for clean HTTP logging
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Build path with query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Format log message based on level
		if currentLevel <= LevelInfo {
			// Clean format for INFO level, skip noisy endpoints
			switch {
			case strings.Contains(path, "/status") || strings.Contains(path, "/track-progress"):
				return // Skip status polling
			case path == "/api/v1/transcription/list" || path == "/health":
				return // Skip frequent list calls
			}
		}

		// Log request
		status := c.Writer.Status()
		statusColor := getStatusColor(status)

		if currentLevel <= LevelDebug {
			// Detailed logging for DEBUG
			Debug("API request",
				"method", c.Request.Method,
				"path", path,
				"status", status,
				"duration", fmt.Sprintf("%.2fms", float64(duration.Nanoseconds())/1e6),
				"ip", c.ClientIP(),
				"user_agent", c.Request.UserAgent())
		} else {
			// Clean format for INFO: "INFO  15:04:05 GET /api/v1/transcription/submit 200 5.13ms"
			fmt.Printf("INFO  %s %s %s %s%d%s %s\n",
				time.Now().Format("15:04:05"),
				c.Request.Method,
				path,
				statusColor,
				status,
				"\033[0m", // Reset color
				fmt.Sprintf("%.2fms", float64(duration.Nanoseconds())/1e6))
		}
	}
}

// getStatusColor returns ANSI color codes for HTTP status codes
func getStatusColor(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "\033[32m" // Green
	case status >= 300 && status < 400:
		return "\033[33m" // Yellow
	case status >= 400 && status < 500:
		return "\033[31m" // Red
	case status >= 500:
		return "\033[35m" // Magenta
	default:
		return "\033[37m" // White
	}
}

// SetGinOutput configures GIN to use a custom writer that suppresses default logs
func SetGinOutput() {
	// Set GIN to use a discard writer to suppress default logging
	gin.DefaultWriter = io.Discard
}
