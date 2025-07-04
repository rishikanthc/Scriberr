package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
	"golang.org/x/crypto/bcrypt"
)

const dbFile = "./storage/scriberr.db"

var db *sql.DB

// InitDB initializes the database connection and creates tables if they don't exist.
func InitDB() (*sql.DB, error) {
	// Ensure the storage directory exists before creating the database file.
	if _, err := os.Stat("./storage"); os.IsNotExist(err) {
		log.Println("Storage directory not found, creating it...")
		if mkdirErr := os.MkdirAll("./storage", 0755); mkdirErr != nil {
			return nil, mkdirErr
		}
	}

	// Open the SQLite database file. It will be created if it doesn't exist.
	database, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify the connection is alive.
	if err := database.Ping(); err != nil {
		return nil, err
	}

	log.Println("Database connection established.")

	// Run migrations to create the necessary tables.
	if err := runMigrations(database); err != nil {
		return nil, err
	}

	// Create a default admin user if no users exist.
	if err := createDefaultUser(database); err != nil {
		return nil, err
	}

	db = database
	return db, nil
}

// migrateAddSummaryColumn checks if the 'summary' column exists in 'audio_records' and adds it if not.
// This ensures backward compatibility for existing databases.
func migrateAddSummaryColumn(tx *sql.Tx) error {
	// Check if the 'summary' column already exists.
	rows, err := tx.Query(`PRAGMA table_info(audio_records)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var summaryColumnExists bool
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt_value *string // Use pointer to handle NULL
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk); err != nil {
			return err
		}
		if name == "summary" {
			summaryColumnExists = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// If the column does not exist, add it.
	if !summaryColumnExists {
		log.Println("Adding 'summary' column to 'audio_records' table...")
		_, err := tx.Exec(`ALTER TABLE audio_records ADD COLUMN summary TEXT`)
		if err != nil {
			return err
		}
		log.Println("Column 'summary' added successfully.")
	}

	return nil
}

// runMigrations creates the database schema if it doesn't already exist.
func runMigrations(db *sql.DB) error {
	createAudioTableSQL := `
	CREATE TABLE IF NOT EXISTS audio_records (
		"id" TEXT NOT NULL PRIMARY KEY,
		"title" TEXT,
		"transcript" TEXT,
		"speaker_map" TEXT,
		"summary" TEXT,
		"created_at" TIMESTAMP NOT NULL
	);`

	createSummaryTemplatesTableSQL := `
	CREATE TABLE IF NOT EXISTS summary_templates (
		"id" TEXT NOT NULL PRIMARY KEY,
		"title" TEXT NOT NULL,
		"prompt" TEXT NOT NULL,
		"created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	createChatSessionsTableSQL := `
	CREATE TABLE IF NOT EXISTS chat_sessions (
		"id" TEXT NOT NULL PRIMARY KEY,
		"audio_id" TEXT NOT NULL,
		"title" TEXT NOT NULL,
		"model" TEXT NOT NULL,
		"created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		"updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (audio_id) REFERENCES audio_records(id) ON DELETE CASCADE
	);`

	createChatMessagesTableSQL := `
	CREATE TABLE IF NOT EXISTS chat_messages (
		"id" TEXT NOT NULL PRIMARY KEY,
		"session_id" TEXT NOT NULL,
		"role" TEXT NOT NULL,
		"content" TEXT NOT NULL,
		"created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
	);`

	createUsersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"username" TEXT NOT NULL UNIQUE,
		"password_hash" TEXT NOT NULL,
		"created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	log.Println("Running database migrations...")

	// Execute migrations within a transaction for safety
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(createAudioTableSQL); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec(createUsersTableSQL); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec(createSummaryTemplatesTableSQL); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec(createChatSessionsTableSQL); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec(createChatMessagesTableSQL); err != nil {
		tx.Rollback()
		return err
	}

	if err := migrateAddSummaryColumn(tx); err != nil {
		tx.Rollback()
		return err
	}

	// --- Start Migration: Add summary column if not exists ---
	var summaryColumnExists bool
	rows, err := tx.Query("PRAGMA table_info(audio_records)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to check table info for migration: %w", err)
	}
	// This defer is crucial to ensure rows are closed even if errors occur mid-loop.
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notnull    int
			dflt_value *string
			pk         int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to scan table info row: %w", err)
		}
		if name == "summary" {
			summaryColumnExists = true
			break
		}
	}
	// Check for errors from rows.Next() that terminated the loop.
	if err = rows.Err(); err != nil {
		tx.Rollback()
		return fmt.Errorf("error during table info scan: %w", err)
	}

	if !summaryColumnExists {
		log.Println("Migration: 'summary' column not found in 'audio_records' table, adding it...")
		if _, err := tx.Exec("ALTER TABLE audio_records ADD COLUMN summary TEXT"); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to add 'summary' column: %w", err)
		}
		log.Println("Migration: 'summary' column added successfully.")
	}
	// --- End Migration ---

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("Migrations completed successfully.")
	return nil
}

// createDefaultUser checks if any user exists and creates a default admin if not.
func createDefaultUser(db *sql.DB) error {
	var userCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return err
	}

	if userCount == 0 {
		log.Println("No users found. Creating default admin user...")
		username := "admin"
		// This password is for demonstration only. In a real application,
		// this should be handled more securely, perhaps via an initial setup process.
		password := "password"

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		query := "INSERT INTO users (username, password_hash) VALUES (?, ?)"
		stmt, err := db.Prepare(query)
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err = stmt.Exec(username, string(hashedPassword))
		if err != nil {
			return err
		}
		log.Println("Default user 'admin' with password 'password' created successfully.")
	}
	return nil
}

// GetDB returns the singleton database connection pool.
func GetDB() *sql.DB {
	if db == nil {
		log.Fatal("Database has not been initialized. Call InitDB() first.")
	}
	return db
}
