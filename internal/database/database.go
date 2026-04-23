package database

import (
	"database/sql"
	"fmt"

	"gorm.io/gorm"
)

// DB is the global database handle used by the application.
var DB *gorm.DB

// Initialize opens the SQLite database, configures it, and migrates it to the latest schema.
func Initialize(dbPath string) error {
	db, err := Open(dbPath)
	if err != nil {
		return err
	}
	if err := Migrate(db); err != nil {
		_ = closeDB(db)
		return err
	}
	DB = db
	return nil
}

// Close closes the global database connection gracefully.
func Close() error {
	if DB == nil {
		return nil
	}
	err := closeDB(DB)
	DB = nil
	return err
}

// HealthCheck verifies the current database connection.
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

// GetConnectionStats returns the current connection pool statistics.
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

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
