package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/pkg/logger"

	"github.com/joho/godotenv"
)

// Config holds all configuration values
type Config struct {
	// Server configuration
	Port string
	Host string

	// Database configuration
	DatabasePath string

	// JWT configuration
	JWTSecret           string
	LLMCredentialSecret string

	// File storage
	UploadDir      string
	TranscriptsDir string
	TempDir        string
	Recordings     RecordingConfig

	// Local speech engine configuration
	Engine EngineConfig

	// Durable transcription worker configuration
	Worker WorkerConfig

	// Environment configuration
	Environment    string
	AllowedOrigins []string
	SecureCookies  bool // Explicit control over Secure flag (for HTTPS deployments)
	// OpenAI configuration
	OpenAIAPIKey string

	// Hugging Face configuration
	HFToken string
}

type EngineConfig struct {
	CacheDir     string
	Provider     string
	Threads      int
	MaxLoaded    int
	AutoDownload bool
}

type WorkerConfig struct {
	Workers      int
	PollInterval time.Duration
	LeaseTimeout time.Duration
}

type RecordingConfig struct {
	Dir                   string
	MaxChunkBytes         int64
	MaxSessionBytes       int64
	MaxDuration           time.Duration
	SessionTTL            time.Duration
	FinalizerWorkers      int
	FinalizerPollInterval time.Duration
	FinalizerLeaseTimeout time.Duration
	CleanupInterval       time.Duration
	FailedRetention       time.Duration
	AllowedMimeTypes      []string
}

// Load loads configuration from environment variables and .env file.
// Prefer LoadWithError for new startup paths so invalid configuration fails clearly.
func Load() *Config {
	cfg, err := LoadWithError()
	if err != nil {
		logger.Error("Invalid configuration; falling back to defaults where possible", "error", err)
		return loadUnchecked()
	}
	return cfg
}

// LoadWithError loads and validates configuration from environment variables and .env file.
func LoadWithError() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found, using system environment variables")
	}

	cfg := loadUnchecked()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadUnchecked() *Config {
	// Default SecureCookies to true in production, false otherwise
	defaultSecure := "false"
	if strings.ToLower(getEnv("APP_ENV", "development")) == "production" {
		defaultSecure = "true"
	}

	jwtSecret := getJWTSecret()
	return &Config{
		Port:                getEnv("PORT", "8080"),
		Host:                getEnv("HOST", "0.0.0.0"),
		Environment:         getEnv("APP_ENV", "development"),
		AllowedOrigins:      strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:8080"), ","),
		DatabasePath:        getEnv("DATABASE_PATH", "data/scriberr.db"),
		JWTSecret:           jwtSecret,
		LLMCredentialSecret: getEnv("LLM_CREDENTIAL_SECRET", jwtSecret),
		UploadDir:           getEnv("UPLOAD_DIR", "data/uploads"),
		TranscriptsDir:      getEnv("TRANSCRIPTS_DIR", "data/transcripts"),
		TempDir:             getEnv("TEMP_DIR", "data/temp"),
		Recordings: RecordingConfig{
			Dir:                   getEnv("RECORDINGS_DIR", "data/recordings"),
			MaxChunkBytes:         getEnvInt64Unchecked("RECORDING_MAX_CHUNK_BYTES", 25<<20),
			MaxSessionBytes:       getEnvInt64Unchecked("RECORDING_MAX_SESSION_BYTES", 2<<30),
			MaxDuration:           getEnvDurationUnchecked("RECORDING_MAX_DURATION", 8*time.Hour),
			SessionTTL:            getEnvDurationUnchecked("RECORDING_SESSION_TTL", 12*time.Hour),
			FinalizerWorkers:      getEnvIntUnchecked("RECORDING_FINALIZER_WORKERS", 1),
			FinalizerPollInterval: getEnvDurationUnchecked("RECORDING_FINALIZER_POLL_INTERVAL", 2*time.Second),
			FinalizerLeaseTimeout: getEnvDurationUnchecked("RECORDING_FINALIZER_LEASE_TIMEOUT", 10*time.Minute),
			CleanupInterval:       getEnvDurationUnchecked("RECORDING_CLEANUP_INTERVAL", 10*time.Minute),
			FailedRetention:       getEnvDurationUnchecked("RECORDING_FAILED_RETENTION", 24*time.Hour),
			AllowedMimeTypes:      splitCSVEnv("RECORDING_ALLOWED_MIME_TYPES", []string{"audio/webm;codecs=opus", "audio/webm"}),
		},
		Engine: EngineConfig{
			CacheDir:     getEnv("SPEECH_ENGINE_CACHE_DIR", "data/models"),
			Provider:     strings.ToLower(strings.TrimSpace(getEnv("SPEECH_ENGINE_PROVIDER", "auto"))),
			Threads:      getEnvIntUnchecked("SPEECH_ENGINE_THREADS", 0),
			MaxLoaded:    getEnvIntUnchecked("SPEECH_ENGINE_MAX_LOADED", 2),
			AutoDownload: getEnvBoolUnchecked("SPEECH_ENGINE_AUTO_DOWNLOAD", true),
		},
		Worker: WorkerConfig{
			Workers:      getEnvIntUnchecked("TRANSCRIPTION_WORKERS", 1),
			PollInterval: getEnvDurationUnchecked("TRANSCRIPTION_QUEUE_POLL_INTERVAL", 2*time.Second),
			LeaseTimeout: getEnvDurationUnchecked("TRANSCRIPTION_LEASE_TIMEOUT", 10*time.Minute),
		},
		SecureCookies: getEnv("SECURE_COOKIES", defaultSecure) == "true",
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
		HFToken:       getEnv("HF_TOKEN", ""),
	}
}

func (c *Config) validate() error {
	if err := validateProvider(c.Engine.Provider); err != nil {
		return err
	}
	if _, err := getEnvInt("SPEECH_ENGINE_THREADS", 0, 0); err != nil {
		return err
	}
	if _, err := getEnvInt("SPEECH_ENGINE_MAX_LOADED", 2, 1); err != nil {
		return err
	}
	if _, err := getEnvBool("SPEECH_ENGINE_AUTO_DOWNLOAD", true); err != nil {
		return err
	}
	if _, err := getEnvInt("TRANSCRIPTION_WORKERS", 1, 1); err != nil {
		return err
	}
	if _, err := getEnvDuration("TRANSCRIPTION_QUEUE_POLL_INTERVAL", 2*time.Second); err != nil {
		return err
	}
	if _, err := getEnvDuration("TRANSCRIPTION_LEASE_TIMEOUT", 10*time.Minute); err != nil {
		return err
	}
	if _, err := getEnvInt64("RECORDING_MAX_CHUNK_BYTES", 25<<20, 1); err != nil {
		return err
	}
	if _, err := getEnvInt64("RECORDING_MAX_SESSION_BYTES", 2<<30, 1); err != nil {
		return err
	}
	if _, err := getEnvDuration("RECORDING_MAX_DURATION", 8*time.Hour); err != nil {
		return err
	}
	if _, err := getEnvDuration("RECORDING_SESSION_TTL", 12*time.Hour); err != nil {
		return err
	}
	if _, err := getEnvInt("RECORDING_FINALIZER_WORKERS", 1, 1); err != nil {
		return err
	}
	if _, err := getEnvDuration("RECORDING_FINALIZER_POLL_INTERVAL", 2*time.Second); err != nil {
		return err
	}
	if _, err := getEnvDuration("RECORDING_FINALIZER_LEASE_TIMEOUT", 10*time.Minute); err != nil {
		return err
	}
	if _, err := getEnvDuration("RECORDING_CLEANUP_INTERVAL", 10*time.Minute); err != nil {
		return err
	}
	if _, err := getEnvDuration("RECORDING_FAILED_RETENTION", 24*time.Hour); err != nil {
		return err
	}
	if len(c.Recordings.AllowedMimeTypes) == 0 {
		return fmt.Errorf("RECORDING_ALLOWED_MIME_TYPES must include at least one MIME type")
	}
	for _, mimeType := range c.Recordings.AllowedMimeTypes {
		if !validRecordingMimeType(mimeType) {
			return fmt.Errorf("RECORDING_ALLOWED_MIME_TYPES contains invalid MIME type %q", mimeType)
		}
	}
	return nil
}

func validateProvider(provider string) error {
	switch provider {
	case "auto", "cpu", "cuda":
		return nil
	default:
		return fmt.Errorf("SPEECH_ENGINE_PROVIDER must be one of auto, cpu, or cuda; got %q", provider)
	}
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue, minValue int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer; got %q", key, raw)
	}
	if value < minValue {
		return 0, fmt.Errorf("%s must be greater than or equal to %d; got %d", key, minValue, value)
	}
	return value, nil
}

func getEnvIntUnchecked(key string, defaultValue int) int {
	value, err := getEnvInt(key, defaultValue, -1<<31)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvInt64(key string, defaultValue, minValue int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer; got %q", key, raw)
	}
	if value < minValue {
		return 0, fmt.Errorf("%s must be greater than or equal to %d; got %d", key, minValue, value)
	}
	return value, nil
}

func getEnvInt64Unchecked(key string, defaultValue int64) int64 {
	value, err := getEnvInt64(key, defaultValue, -1<<63)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvBool(key string, defaultValue bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean; got %q", key, raw)
	}
	return value, nil
}

func getEnvBoolUnchecked(key string, defaultValue bool) bool {
	value, err := getEnvBool(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration; got %q", key, raw)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0; got %s", key, value)
	}
	return value, nil
}

func getEnvDurationUnchecked(key string, defaultValue time.Duration) time.Duration {
	value, err := getEnvDuration(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func splitCSVEnv(key string, defaultValues []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return append([]string(nil), defaultValues...)
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func validRecordingMimeType(mimeType string) bool {
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" || strings.ContainsAny(mimeType, "\r\n") {
		return false
	}
	parts := strings.SplitN(mimeType, "/", 2)
	return len(parts) == 2 && parts[0] == "audio" && strings.TrimSpace(parts[1]) != ""
}

// getJWTSecret gets JWT secret from env or generates a secure random one
func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	// Persist a dev secret across restarts to avoid invalidating tokens
	secretFile := getEnv("JWT_SECRET_FILE", "data/jwt_secret")
	if data, err := os.ReadFile(secretFile); err == nil && len(data) > 0 {
		return strings.TrimSpace(string(data))
	}
	// Generate a secure random JWT secret and persist it
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		logger.Warn("Could not generate secure JWT secret, using fallback", "error", err)
		return "fallback-jwt-secret-please-set-JWT_SECRET-env-var"
	}
	secret := hex.EncodeToString(bytes)
	// Ensure dir exists and write file (best-effort)
	_ = os.MkdirAll(filepath.Dir(secretFile), 0755)
	_ = os.WriteFile(secretFile, []byte(secret), 0600)
	logger.Debug("Generated persistent JWT secret", "path", secretFile)
	return secret
}
