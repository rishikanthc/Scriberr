package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open creates a configured SQLite connection without running migrations.
func Open(dbPath string) (*gorm.DB, error) {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?"+
		"_pragma=foreign_keys(1)&"+
		"_pragma=journal_mode(WAL)&"+
		"_pragma=synchronous(NORMAL)&"+
		"_pragma=cache_size(-64000)&"+
		"_pragma=temp_store(MEMORY)&"+
		"_pragma=mmap_size(268435456)&"+
		"_timeout=30000",
		dbPath,
	)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logger.Warn),
		CreateBatchSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return db, nil
}
