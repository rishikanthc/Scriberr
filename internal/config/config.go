package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
	UploadDir string

	// Python/WhisperX configuration
	UVPath      string
	WhisperXEnv string
}

// Load loads configuration from environment variables and .env file
func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		Port:         getEnv("PORT", "8080"),
		Host:         getEnv("HOST", "localhost"),
		DatabasePath: getEnv("DATABASE_PATH", "data/scriberr.db"),
		JWTSecret:    getJWTSecret(),
		UploadDir:    getEnv("UPLOAD_DIR", "data/uploads"),
		UVPath:       findUVPath(),
		WhisperXEnv:  getEnv("WHISPERX_ENV", "data/whisperx-env"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int with a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as bool with a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
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
		log.Printf("Warning: Could not generate secure JWT secret, using fallback: %v", err)
		return "fallback-jwt-secret-please-set-JWT_SECRET-env-var"
	}
	secret := hex.EncodeToString(bytes)
	// Ensure dir exists and write file (best-effort)
	_ = os.MkdirAll(filepath.Dir(secretFile), 0755)
	_ = os.WriteFile(secretFile, []byte(secret), 0600)
	log.Println("Generated persistent JWT secret at", secretFile)
	return secret
}

// findUVPath finds UV package manager in common locations
func findUVPath() string {
	if uvPath := os.Getenv("UV_PATH"); uvPath != "" {
		return uvPath
	}

	if path, err := exec.LookPath("uv"); err == nil {
		log.Printf("Found UV at: %s", path)
		return path
	}

	log.Println("Warning: UV package manager not found in PATH, using 'uv' as fallback")
	return "uv"
}
