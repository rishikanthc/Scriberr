package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	*zap.SugaredLogger
}

var (
	defaultLogger *Logger
	currentLevel  = LevelInfo
	atomicLevel   = zap.NewAtomicLevelAt(zapcore.InfoLevel)
)

func Init(level string) {
	zapLevel := parseLevel(level)
	atomicLevel.SetLevel(zapLevel)
	currentLevel = toLocalLevel(zapLevel)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeDuration = zapcore.MillisDurationEncoder

	cfg := zap.Config{
		Level:             atomicLevel,
		Development:       zapLevel == zapcore.DebugLevel,
		DisableCaller:     true,
		DisableStacktrace: zapLevel != zapcore.DebugLevel,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
	}

	log, err := cfg.Build()
	if err != nil {
		log = zap.NewNop()
	}
	defaultLogger = &Logger{SugaredLogger: log.Sugar()}
}

func Get() *Logger {
	if defaultLogger == nil {
		Init(os.Getenv("LOG_LEVEL"))
	}
	return defaultLogger
}

func GetLevel() LogLevel {
	return currentLevel
}

func SetGinOutput() {
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
}

func Debug(msg string, args ...any) {
	Get().Debugw(msg, args...)
}

func Info(msg string, args ...any) {
	Get().Infow(msg, args...)
}

func Warn(msg string, args ...any) {
	Get().Warnw(msg, args...)
}

func Error(msg string, args ...any) {
	Get().Errorw(msg, args...)
}

func WithContext(key string, value any) *Logger {
	return &Logger{SugaredLogger: Get().With(key, value)}
}

func Startup(step, message string, args ...any) {
	if currentLevel <= LevelInfo {
		fmt.Printf("[+] %s\n", message)
	}
	Debug("startup step", append([]any{"step", step, "message", message}, args...)...)
}

func JobStarted(jobID, filename, model string, params map[string]any) {
	Info("transcription started", "file", filename)
	Debug("job started", "job_id", jobID, "file", filename, "model", model, "params", params)
}

func JobCompleted(jobID string, duration time.Duration, result any) {
	Info("transcription completed", "duration", duration.String())
	Debug("job completed", "job_id", jobID, "duration", duration.String(), "result", result)
}

func JobFailed(jobID string, duration time.Duration, err error) {
	Error("transcription failed", "error", err.Error())
	Debug("job failed", "job_id", jobID, "duration", duration.String(), "error", err.Error())
}

func HTTPRequest(method, path string, status int, duration time.Duration, userAgent string) {
	fields := []any{
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", float64(duration.Microseconds()) / 1000,
		"user_agent", userAgent,
	}
	if currentLevel <= LevelDebug {
		Debug("api request", fields...)
		return
	}
	if status >= 500 {
		Error("api request failed", fields...)
		return
	}
	if status >= 400 {
		Warn("api request rejected", fields...)
	}
}

func AuthEvent(event, username, ip string, success bool, details ...any) {
	fields := append([]any{"event", event, "username", username, "ip", ip, "success", success}, details...)
	if success {
		Info("auth event", fields...)
		return
	}
	Warn("auth event", fields...)
}

func WorkerOperation(workerID int, jobID string, operation string, args ...any) {
	Debug("worker operation", append([]any{"worker_id", workerID, "job_id", jobID, "operation", operation}, args...)...)
}

func Performance(operation string, duration time.Duration, details ...any) {
	Debug("performance", append([]any{"operation", operation, "duration", duration.String()}, details...)...)
}

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		fields := []any{
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", float64(time.Since(start).Microseconds()) / 1000,
			"auth_type", c.GetString("auth_type"),
		}

		status := c.Writer.Status()
		switch {
		case currentLevel <= LevelDebug:
			Debug("api request", fields...)
		case status >= 500:
			Error("api request failed", fields...)
		case status >= 400:
			Warn("api request rejected", fields...)
		}
	}
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "info", "":
		return zapcore.InfoLevel
	default:
		return zapcore.InfoLevel
	}
}

func toLocalLevel(level zapcore.Level) LogLevel {
	switch level {
	case zapcore.DebugLevel:
		return LevelDebug
	case zapcore.WarnLevel:
		return LevelWarn
	case zapcore.ErrorLevel:
		return LevelError
	default:
		return LevelInfo
	}
}
