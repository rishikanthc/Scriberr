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

	// Python/WhisperX configuration
	WhisperXEnv string

	// Environment configuration
	Environment    string
	AllowedOrigins []string
	SecureCookies  bool // Explicit control over Secure flag (for HTTPS deployments)
	// OpenAI configuration
	OpenAIAPIKey string
}

// Load loads configuration from environment variables and .env file
func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found, using system environment variables")
	}

	// Default SecureCookies to true in production, false otherwise
	defaultSecure := "false"
	if strings.ToLower(getEnv("APP_ENV", "development")) == "production" {
		defaultSecure = "true"
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		Host:           getEnv("HOST", "0.0.0.0"),
		Environment:    getEnv("APP_ENV", "development"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:8080"), ","),
		DatabasePath:   getEnv("DATABASE_PATH", "data/scriberr.db"),
		JWTSecret:      getJWTSecret(),
		UploadDir:      getEnv("UPLOAD_DIR", "data/uploads"),
		TranscriptsDir: getEnv("TRANSCRIPTS_DIR", "data/transcripts"),
		WhisperXEnv:    getEnv("WHISPERX_ENV", "data/whisperx-env"),
		SecureCookies:  getEnv("SECURE_COOKIES", defaultSecure) == "true",
		OpenAIAPIKey:   getEnv("OPENAI_API_KEY", ""),
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
