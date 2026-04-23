package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Migrate upgrades the database to the latest supported schema.
func Migrate(db *gorm.DB) error {
	hasSchemaState, err := db.Migrator().HasTable(&schemaMigration{}), error(nil)
	if err != nil {
		return err
	}

	if hasSchemaState {
		return db.Transaction(func(tx *gorm.DB) error {
			if err := createTargetSchema(tx); err != nil {
				return err
			}
			version, err := currentSchemaVersion(tx)
			if err != nil {
				return err
			}
			return runSchemaSteps(tx, version)
		})
	}

	legacy, err := detectLegacySchema(db)
	if err != nil {
		return err
	}
	if legacy {
		return migrateLegacy(db)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := createTargetSchema(tx); err != nil {
			return err
		}
		return recordSchemaVersion(tx, latestSchemaVersion)
	})
}

func detectLegacySchema(db *gorm.DB) (bool, error) {
	currentTables := []string{
		"transcriptions",
		"transcription_executions",
		"transcription_tracks",
		"llm_profiles",
		"schema_migrations",
	}
	for _, table := range currentTables {
		if db.Migrator().HasTable(table) {
			return false, nil
		}
	}

	legacyOnlyTables := []string{
		"transcription_jobs",
		"transcription_job_executions",
		"multi_track_files",
		"llm_configs",
		"summary_settings",
	}
	for _, table := range legacyOnlyTables {
		if db.Migrator().HasTable(table) {
			return true, nil
		}
	}

	return false, nil
}

func currentSchemaVersion(tx *gorm.DB) (int, error) {
	var migration schemaMigration
	err := tx.Order("version DESC").First(&migration).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return migration.Version, nil
}

func recordSchemaVersion(tx *gorm.DB, version int) error {
	if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
		if !isIgnorableSQLiteDuplicateIndexError(err) {
			return fmt.Errorf("ensure schema migrations table: %w", err)
		}
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&schemaMigration{
		Version:   version,
		AppliedAt: time.Now().Unix(),
	}).Error
}
