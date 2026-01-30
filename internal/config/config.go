package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

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
	JWTSecret string

	// File storage
	UploadDir      string
	TranscriptsDir string
	TempDir        string

	// Python diarization configuration
	ModelEnv string

	// Environment configuration
	Environment       string
	AllowedOrigins    []string
	SecureCookiesMode string // "auto" (default), "true"/"false"
	TrustProxyHeaders bool   // Whether to trust X-Forwarded-Proto/Forwarded
	// OpenAI configuration
	OpenAIAPIKey string

	// Hugging Face configuration
	HFToken string
}

// Load loads configuration from environment variables and .env file
func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found, using system environment variables")
	}

	defaultSecureMode := strings.ToLower(getEnv("SECURE_COOKIES", "auto"))
	if defaultSecureMode == "" {
		defaultSecureMode = "auto"
	}

	return &Config{
		Port:              getEnv("PORT", "8080"),
		Host:              getEnv("HOST", "0.0.0.0"),
		Environment:       getEnv("APP_ENV", "development"),
		AllowedOrigins:    strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:8080"), ","),
		DatabasePath:      getEnv("DATABASE_PATH", "data/scriberr.db"),
		JWTSecret:         getJWTSecret(),
		UploadDir:         getEnv("UPLOAD_DIR", "data/uploads"),
		TranscriptsDir:    getEnv("TRANSCRIPTS_DIR", "data/transcripts"),
		TempDir:           getEnv("TEMP_DIR", "data/temp"),
		ModelEnv:          resolveModelEnv(),
		SecureCookiesMode: defaultSecureMode,
		TrustProxyHeaders: strings.ToLower(getEnv("TRUST_PROXY_HEADERS", "true")) == "true",
		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		HFToken:           getEnv("HF_TOKEN", ""),
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

func resolveModelEnv() string {
	if value := os.Getenv("MODEL_ENV"); value != "" {
		return value
	}
	if value := os.Getenv("WHISPERX_ENV"); value != "" {
		logger.Warn("WHISPERX_ENV is deprecated; use MODEL_ENV instead")
		return value
	}
	legacy := filepath.Join("data", "whisperx-env")
	if _, err := os.Stat(legacy); err == nil {
		logger.Warn("Using legacy model env path; set MODEL_ENV to override", "path", legacy)
		return legacy
	}
	return filepath.Join("data", "model-env")
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
