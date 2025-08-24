package database

import (
	"fmt"
	"os"

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