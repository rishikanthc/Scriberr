package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"scriberr/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// Initialize initializes the database connection with optimized settings
func Initialize(dbPath string) error {
	var err error

	// Create database directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// SQLite connection string with performance optimizations
	dsn := fmt.Sprintf("%s?"+
		"_pragma=foreign_keys(1)&"+ // Enable foreign keys
		"_pragma=journal_mode(WAL)&"+ // Use WAL mode for better concurrency
		"_pragma=synchronous(NORMAL)&"+ // Balance between safety and performance
		"_pragma=cache_size(-64000)&"+ // 64MB cache size
		"_pragma=temp_store(MEMORY)&"+ // Store temp tables in memory
		"_pragma=mmap_size(268435456)&"+ // 256MB mmap size
		"_timeout=30000", // 30 second timeout
		dbPath)

	// Open database connection with optimized config
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logger.Warn), // Reduce logging overhead
		CreateBatchSize: 100,                                 // Optimize batch inserts
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

	// Configure connection pool for optimal performance
	sqlDB.SetMaxOpenConns(10)                  // SQLite generally works well with lower connection counts
	sqlDB.SetMaxIdleConns(5)                   // Keep some connections idle
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // Reset connections every 30 minutes
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)  // Close idle connections after 5 minutes

	// Auto migrate the schema
	if err := DB.AutoMigrate(
		&models.TranscriptionJob{},
		&models.TranscriptionJobExecution{},
		&models.SpeakerMapping{},
		&models.MultiTrackFile{},
		&models.User{},
		&models.APIKey{},
		&models.TranscriptionProfile{},
		&models.LLMConfig{},
		&models.ChatSession{},
		&models.ChatMessage{},
		&models.SummaryTemplate{},
		&models.SummarySetting{},
		&models.Summary{},
		&models.Note{},
		&models.RefreshToken{},
		&models.CSVBatch{},
		&models.CSVBatchRow{},
	); err != nil {
		return fmt.Errorf("failed to auto migrate: %v", err)
	}

	// Cleanup duplicate speaker mappings before creating unique index (for backward compatibility)
	// Keep the latest mapping for each (job_id, original_speaker) pair
	cleanupQuery := `
		DELETE FROM speaker_mappings 
		WHERE id NOT IN (
			SELECT MAX(id) 
			FROM speaker_mappings 
			GROUP BY transcription_job_id, original_speaker
		)
	`
	if err := DB.Exec(cleanupQuery).Error; err != nil {
		// Log warning but continue, as table might not exist yet or query might fail for other reasons
		// We don't want to block startup if this fails, but index creation might fail next.
		fmt.Printf("Warning: Failed to cleanup duplicate speaker mappings: %v\n", err)
	}

	// Add unique constraint for speaker mappings (transcription_job_id + original_speaker)
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_speaker_mappings_unique ON speaker_mappings(transcription_job_id, original_speaker)").Error; err != nil {
		return fmt.Errorf("failed to create unique constraint for speaker mappings: %v", err)
	}

	return nil
}

// Close closes the database connection gracefully
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Close()
	DB = nil // Set to nil after closing
	return err
}

// HealthCheck performs a health check on the database connection
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

	// Test the connection with a ping
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %v", err)
	}

	return nil
}

// GetConnectionStats returns database connection pool statistics
func GetConnectionStats() sql.DBStats {
	if DB == nil {
		return sql.DBStats{}
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return sql.DBStats{}
	}

	return sqlDB.Stats()
}
