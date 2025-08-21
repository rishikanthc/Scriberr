package config

import (
	"log"
	"os"
	"strconv"

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
	PythonPath  string
	UVPath      string
	WhisperXEnv string
	
	// Default API key for testing
	DefaultAPIKey string
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
		JWTSecret:    getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		UploadDir:    getEnv("UPLOAD_DIR", "data/uploads"),
		PythonPath:   getEnv("PYTHON_PATH", "python3"),
		UVPath:       getEnv("UV_PATH", "uv"),
		WhisperXEnv:  getEnv("WHISPERX_ENV", "data/whisperx-env"),
		DefaultAPIKey: getEnv("DEFAULT_API_KEY", "dev-api-key-123"),
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