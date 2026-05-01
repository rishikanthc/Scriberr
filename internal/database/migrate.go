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
		"llm_configs",
		"summary_settings",
	}
	for _, table := range legacyOnlyTables {
		if db.Migrator().HasTable(table) {
			return true, nil
		}
	}

	sameNameTables := []string{
		"users",
		"api_keys",
		"refresh_tokens",
		"transcription_profiles",
		"speaker_mappings",
		"summary_templates",
		"summaries",
		"chat_sessions",
		"chat_messages",
	}
	for _, table := range sameNameTables {
		if !db.Migrator().HasTable(table) {
			continue
		}
		legacyLike, err := isLegacySameNameTable(db, table)
		if err != nil {
			return false, err
		}
		if legacyLike {
			return true, nil
		}
	}

	return false, nil
}

func isLegacySameNameTable(db *gorm.DB, table string) (bool, error) {
	columns, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		return false, fmt.Errorf("inspect columns for %s: %w", table, err)
	}
	columnNames := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		columnNames[column.Name()] = struct{}{}
	}

	requiredCurrentColumns := map[string][]string{
		"users":                  {"settings_json", "password_hash"},
		"api_keys":               {"key_hash", "metadata_json"},
		"refresh_tokens":         {"token_hash", "revoked_at"},
		"transcription_profiles": {"config_json", "user_id"},
		"speaker_mappings":       {"display_name", "user_id", "transcription_id"},
		"summary_templates":      {"config_json", "user_id"},
		"summaries":              {"model_name", "user_id"},
		"chat_sessions":          {"parent_transcription_id", "context_policy_json", "user_id"},
		"chat_messages":          {"chat_session_id", "reasoning_content", "status", "user_id"},
	}

	required := requiredCurrentColumns[table]
	if len(required) == 0 {
		return false, nil
	}
	for _, column := range required {
		if _, ok := columnNames[column]; !ok {
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
		if !isIgnorableSQLiteDuplicateIndexError(tx, err) {
			return fmt.Errorf("ensure schema migrations table: %w", err)
		}
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&schemaMigration{
		Version:   version,
		AppliedAt: time.Now().Unix(),
	}).Error
}
