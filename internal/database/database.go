package database

import (
	"fmt"
	"os"

	"scriberr/internal/auth"
	"scriberr/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// Initialize initializes the database connection
func Initialize(dbPath string) error {
	var err error
	
	// Create database directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}
	
	// Open database connection
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto migrate the schema
	if err := DB.AutoMigrate(
		&models.TranscriptionJob{},
		&models.User{},
		&models.APIKey{},
		&models.TranscriptionProfile{},
	); err != nil {
		return fmt.Errorf("failed to auto migrate: %v", err)
	}

	// Create default user and API key if they don't exist
	if err := createDefaultCredentials(); err != nil {
		return fmt.Errorf("failed to create default credentials: %v", err)
	}

	return nil
}

// createDefaultCredentials creates default user and API key if they don't exist
func createDefaultCredentials() error {
	// Check if any users exist
	var userCount int64
	if err := DB.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return err
	}

	// Create default user if no users exist
	if userCount == 0 {
		// Use environment variable for default password or generate a secure default
		defaultPassword := os.Getenv("SCRIBERR_DEFAULT_PASSWORD")
		if defaultPassword == "" {
			defaultPassword = "admin123" // This should be changed on first login
		}

		hashedPassword, err := auth.HashPassword(defaultPassword)
		if err != nil {
			return fmt.Errorf("failed to hash default password: %v", err)
		}

		defaultUser := models.User{
			Username: "admin",
			Password: hashedPassword,
		}

		if err := DB.Create(&defaultUser).Error; err != nil {
			return fmt.Errorf("failed to create default user: %v", err)
		}

		fmt.Printf("✓ Default user created: username='admin', password='%s'\n", defaultPassword)
		fmt.Println("⚠️  SECURITY WARNING: Please change the default password after first login!")
	}

	// Check if any API keys exist
	var apiKeyCount int64
	if err := DB.Model(&models.APIKey{}).Count(&apiKeyCount).Error; err != nil {
		return err
	}

	// Create default API key if no API keys exist
	if apiKeyCount == 0 {
		defaultAPIKey := models.APIKey{
			Key:         "dev-api-key-123", // For backward compatibility with existing setups
			Name:        "Default Development Key",
			Description: new(string),
			IsActive:    true,
		}
		*defaultAPIKey.Description = "Default API key for development and backward compatibility"

		if err := DB.Create(&defaultAPIKey).Error; err != nil {
			return fmt.Errorf("failed to create default API key: %v", err)
		}

		fmt.Printf("✓ Default API key created: %s\n", defaultAPIKey.Key)
		fmt.Println("⚠️  SECURITY WARNING: This is a development key. Create secure API keys for production!")
	}

	return nil
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}