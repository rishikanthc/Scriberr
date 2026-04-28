package database

import (
	"fmt"
	"time"

	"scriberr/internal/models"

	"gorm.io/gorm"
)

type migrationStep func(*gorm.DB) error

var schemaSteps = map[int]migrationStep{
	2: migrateStepV1ToV2,
}

func runSchemaSteps(tx *gorm.DB, currentVersion int) error {
	for version := currentVersion + 1; version <= latestSchemaVersion; version++ {
		step := schemaSteps[version]
		if step != nil {
			if err := step(tx); err != nil {
				return fmt.Errorf("apply schema migration v%d: %w", version, err)
			}
		}
		if err := recordSchemaVersion(tx, version); err != nil {
			return fmt.Errorf("record schema version %d: %w", version, err)
		}
	}
	return nil
}

func migrateStepV1ToV2(tx *gorm.DB) error {
	if err := ensureSingleDefaultPerUser(tx); err != nil {
		return err
	}
	if err := backfillCompatibilityColumns(tx); err != nil {
		return err
	}
	return nil
}

func backfillCompatibilityColumns(tx *gorm.DB) error {
	if err := backfillUsers(tx); err != nil {
		return err
	}
	if err := backfillAPIKeys(tx); err != nil {
		return err
	}
	if err := backfillTranscriptionProfiles(tx); err != nil {
		return err
	}
	if err := backfillTranscriptionJobs(tx); err != nil {
		return err
	}
	if err := backfillTranscriptionExecutions(tx); err != nil {
		return err
	}
	if err := backfillSummaryTemplates(tx); err != nil {
		return err
	}
	if err := backfillLLMConfigs(tx); err != nil {
		return err
	}
	return nil
}

func backfillUsers(tx *gorm.DB) error {
	var rows []models.User
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := row.BeforeSave(tx); err != nil {
			return err
		}
		updates := map[string]any{
			"settings_json": row.SettingsJSON,
		}
		if err := withPreservedUpdatedAt(tx.Model(&models.User{}).Where("id = ?", row.ID), updates, row.UpdatedAt).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillAPIKeys(tx *gorm.DB) error {
	var rows []models.APIKey
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := row.BeforeSave(tx); err != nil {
			return err
		}
		updates := map[string]any{
			"key_hash":      row.KeyHash,
			"key_prefix":    row.KeyPrefix,
			"metadata_json": row.MetadataJSON,
		}
		if err := withPreservedUpdatedAt(tx.Model(&models.APIKey{}).Where("id = ?", row.ID), updates, time.Time{}).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillTranscriptionProfiles(tx *gorm.DB) error {
	var rows []models.TranscriptionProfile
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := row.BeforeSave(tx); err != nil {
			return err
		}
		updates := map[string]any{
			"user_id":             row.UserID,
			"model_name":          row.ModelName,
			"model_family":        row.ModelFamily,
			"language":            row.Language,
			"diarization_enabled": row.DiarizationEnabled,
			"device":              row.Device,
			"compute_type":        row.ComputeType,
			"config_json":         row.ConfigJSON,
			"is_default":          row.IsDefault,
		}
		if err := withPreservedUpdatedAt(tx.Model(&models.TranscriptionProfile{}).Where("id = ?", row.ID), updates, row.UpdatedAt).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillTranscriptionJobs(tx *gorm.DB) error {
	var rows []models.TranscriptionJob
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := row.SyncColumnsForMigration(); err != nil {
			return err
		}
		updates := map[string]any{
			"user_id":             row.UserID,
			"source_file_path":    row.AudioPath,
			"source_file_name":    row.SourceFileName,
			"source_file_hash":    row.SourceFileHash,
			"source_duration_ms":  row.SourceDurationMs,
			"language":            row.Language,
			"transcript_text":     row.Transcript,
			"output_json_path":    row.OutputJSONPath,
			"output_srt_path":     row.OutputSRTPath,
			"output_vtt_path":     row.OutputVTTPath,
			"latest_execution_id": row.LatestExecutionID,
			"last_error":          row.ErrorMessage,
			"metadata_json":       row.MetadataJSON,
			"completed_at":        row.CompletedAt,
			"status":              row.Status,
			"title":               row.Title,
		}
		if err := withPreservedUpdatedAt(tx.Model(&models.TranscriptionJob{}).Where("id = ?", row.ID), updates, row.UpdatedAt).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillTranscriptionExecutions(tx *gorm.DB) error {
	var rows []models.TranscriptionJobExecution
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := row.SyncColumnsForMigration(); err != nil {
			return err
		}
		updates := map[string]any{
			"user_id":          row.UserID,
			"execution_number": row.ExecutionNumber,
			"trigger_type":     row.TriggerType,
			"status":           row.Status,
			"profile_id":       row.ProfileID,
			"model_name":       row.ModelName,
			"model_family":     row.ModelFamily,
			"provider":         row.Provider,
			"device":           row.Device,
			"compute_type":     row.ComputeType,
			"request_json":     row.RequestJSON,
			"config_json":      row.ConfigJSON,
			"started_at":       row.StartedAt,
			"completed_at":     row.CompletedAt,
			"failed_at":        row.FailedAt,
			"error_message":    row.ErrorMessage,
			"logs_path":        row.LogsPath,
			"output_json_path": row.OutputJSONPath,
		}
		if err := withPreservedUpdatedAt(tx.Model(&models.TranscriptionJobExecution{}).Where("id = ?", row.ID), updates, time.Time{}).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillSummaryTemplates(tx *gorm.DB) error {
	var rows []models.SummaryTemplate
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := row.BeforeSave(tx); err != nil {
			return err
		}
		updates := map[string]any{
			"user_id":     row.UserID,
			"name":        row.Name,
			"prompt":      row.Prompt,
			"description": row.Description,
			"config_json": row.ConfigJSON,
			"is_default":  row.IsDefault,
		}
		if err := withPreservedUpdatedAt(tx.Model(&models.SummaryTemplate{}).Where("id = ?", row.ID), updates, row.UpdatedAt).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillLLMConfigs(tx *gorm.DB) error {
	var rows []models.LLMConfig
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := row.BeforeSave(tx); err != nil {
			return err
		}
		updates := map[string]any{
			"user_id":     row.UserID,
			"name":        row.Name,
			"provider":    row.Provider,
			"model_name":  row.ModelName,
			"base_url":    row.BaseURL,
			"config_json": row.ConfigJSON,
			"is_default":  row.IsDefault,
		}
		if err := withPreservedUpdatedAt(tx.Model(&models.LLMConfig{}).Where("id = ?", row.ID), updates, row.UpdatedAt).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func withPreservedUpdatedAt(db *gorm.DB, updates map[string]any, updatedAt time.Time) *gorm.DB {
	if !updatedAt.IsZero() {
		updates["updated_at"] = updatedAt
	}
	return db.Session(&gorm.Session{SkipHooks: true}).Set("gorm:update_track_time", false)
}
