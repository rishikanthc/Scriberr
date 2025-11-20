package database

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "scriberr/internal/models"

    "github.com/glebarez/sqlite"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// Initialize initializes the database connection with optimized settings
func Initialize(dbPath string) error {
    var err error

    // Choose driver
    driver := strings.ToLower(os.Getenv("DB_DRIVER"))
    dbURL := os.Getenv("DATABASE_URL")

    if driver == "postgres" || (dbURL != "" && driver == "") {
        // Open Postgres
        if dbURL == "" {
            return fmt.Errorf("DATABASE_URL is required for postgres driver")
        }
        DB, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{
            Logger:          logger.Default.LogMode(logger.Warn),
            CreateBatchSize: 100,
        })
        if err != nil {
            return fmt.Errorf("failed to connect to postgres: %v", err)
        }
    } else {
        // Ensure directory for sqlite file exists
        dir := filepath.Dir(dbPath)
        if dir != "." && dir != "" {
            if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
                return fmt.Errorf("failed to create data directory: %v", mkErr)
            }
        }

        // SQLite connection string with performance optimizations
        dsn := fmt.Sprintf("%s?"+
            "_pragma=foreign_keys(1)&"+          // Enable foreign keys
            "_pragma=journal_mode(WAL)&"+        // Use WAL mode for better concurrency
            "_pragma=synchronous(NORMAL)&"+      // Balance between safety and performance
            "_pragma=cache_size(-64000)&"+       // 64MB cache size
            "_pragma=temp_store(MEMORY)&"+       // Store temp tables in memory
            "_pragma=mmap_size(268435456)&"+     // 256MB mmap size
            "_timeout=30000",                     // 30 second timeout
            dbPath)

        // Open SQLite database connection with optimized config
        DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
            Logger:          logger.Default.LogMode(logger.Warn), // Reduce logging overhead
            CreateBatchSize: 100,                                 // Optimize batch inserts
        })
        if err != nil {
            return fmt.Errorf("failed to connect to sqlite: %v", err)
        }
    }

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

    // Configure connection pool for optimal performance
    if driver == "postgres" || (dbURL != "" && driver == "") {
        // Higher concurrency for Postgres
        sqlDB.SetMaxOpenConns(25)
        sqlDB.SetMaxIdleConns(10)
    } else {
        // SQLite generally works well with lower connection counts
        sqlDB.SetMaxOpenConns(10)
        sqlDB.SetMaxIdleConns(5)
    }
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
    ); err != nil {
        return fmt.Errorf("failed to auto migrate: %v", err)
    }

	// Add unique constraint for speaker mappings (transcription_job_id + original_speaker)
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_speaker_mappings_unique ON speaker_mappings(transcription_job_id, original_speaker)").Error; err != nil {
		return fmt.Errorf("failed to create unique constraint for speaker mappings: %v", err)
	}

    // Backfill ownership: assign existing records without owner to the first user
    var firstUser models.User
    _ = DB.Order("id ASC").First(&firstUser).Error
    if firstUser.ID != 0 {
        // Tables with user ownership
        _ = DB.Model(&models.TranscriptionJob{}).Where("user_id = 0").Update("user_id", firstUser.ID).Error
        _ = DB.Model(&models.APIKey{}).Where("user_id = 0").Update("user_id", firstUser.ID).Error
        _ = DB.Model(&models.TranscriptionProfile{}).Where("user_id = 0").Update("user_id", firstUser.ID).Error
        _ = DB.Model(&models.ChatSession{}).Where("user_id = 0").Update("user_id", firstUser.ID).Error
        _ = DB.Model(&models.SummaryTemplate{}).Where("user_id = 0").Update("user_id", firstUser.ID).Error
    }

    // Ensure at least one admin: make the first user admin if none exist
    {
        var adminCount int64
        if err := DB.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err == nil {
            if adminCount == 0 && firstUser.ID != 0 && !firstUser.IsAdmin {
                firstUser.IsAdmin = true
                _ = DB.Save(&firstUser).Error
            }
        }
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
