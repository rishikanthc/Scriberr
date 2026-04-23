package database

import (
	"fmt"
	"strings"
	"time"

	"scriberr/internal/models"

	"gorm.io/gorm"
)

const latestSchemaVersion = 2

var schemaModels = []any{
	&models.User{},
	&models.APIKey{},
	&models.RefreshToken{},
	&models.TranscriptionProfile{},
	&models.TranscriptionJob{},
	&models.TranscriptionJobExecution{},
	&models.MultiTrackFile{},
	&models.SpeakerMapping{},
	&models.SummaryTemplate{},
	&models.Summary{},
	&models.Note{},
	&models.ChatSession{},
	&models.ChatMessage{},
	&models.LLMConfig{},
}

type schemaMigration struct {
	Version   int   `gorm:"primaryKey"`
	AppliedAt int64 `gorm:"not null"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

func createTargetSchema(tx *gorm.DB) error {
	for _, model := range schemaModels {
		if err := tx.AutoMigrate(model); err != nil {
			if !isIgnorableSQLiteDuplicateIndexError(err) {
				return fmt.Errorf("auto migrate target schema: %w", err)
			}
		}
	}
	if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
		if !isIgnorableSQLiteDuplicateIndexError(err) {
			return fmt.Errorf("auto migrate schema state: %w", err)
		}
	}
	if err := ensureSingleDefaultPerUser(tx); err != nil {
		return fmt.Errorf("enforce default-selection invariants: %w", err)
	}

	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_speaker_mappings_unique ON speaker_mappings(transcription_id, original_speaker)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_tracks_unique ON transcription_tracks(transcription_id, track_index)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_executions_unique ON transcription_executions(transcription_id, execution_number)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_profiles_user_default_unique ON transcription_profiles(user_id) WHERE is_default = 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_summary_templates_user_default_unique ON summary_templates(user_id) WHERE is_default = 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_profiles_user_default_unique ON llm_profiles(user_id) WHERE is_default = 1`,
		`CREATE INDEX IF NOT EXISTS idx_transcriptions_status_created_at ON transcriptions(status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_session_created_at ON chat_messages(chat_session_id, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_transcription_created_at ON summaries(transcription_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_transcription_created_at ON notes(transcription_id, created_at DESC)`,
	}
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("apply schema index: %w", err)
		}
	}
	return nil
}

func isIgnorableSQLiteDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "sql logic error") {
		return false
	}
	if !strings.Contains(errMsg, "already exists") {
		return false
	}
	if !strings.Contains(errMsg, "index idx_") {
		return false
	}
	return true
}

func ensureSingleDefaultPerUser(tx *gorm.DB) error {
	if err := enforceLatestDefaultPerUserForProfiles(tx); err != nil {
		return err
	}
	if err := enforceLatestDefaultPerUserForSummaryTemplates(tx); err != nil {
		return err
	}
	if err := enforceLatestDefaultPerUserForLLMProfiles(tx); err != nil {
		return err
	}
	return nil
}

type defaultProfileRow struct {
	ID        string    `gorm:"column:id"`
	UserID    uint      `gorm:"column:user_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type defaultStringRow struct {
	ID        string
	UserID    uint
	CreatedAt time.Time
}

type defaultUintRow struct {
	ID        uint
	UserID    uint
	CreatedAt time.Time
}

func enforceLatestDefaultPerUserForProfiles(tx *gorm.DB) error {
	var rows []defaultProfileRow
	if err := tx.Model(&models.TranscriptionProfile{}).
		Where("is_default = ?", true).
		Order("user_id ASC, created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load default transcription profiles: %w", err)
	}

	latestIDByUser := make(map[uint]string)
	for _, row := range rows {
		if _, ok := latestIDByUser[row.UserID]; ok {
			continue
		}
		latestIDByUser[row.UserID] = row.ID
	}

	idsToClear := make([]string, 0, len(rows))
	for _, row := range rows {
		if latestIDByUser[row.UserID] != row.ID {
			idsToClear = append(idsToClear, row.ID)
		}
	}
	if len(idsToClear) == 0 {
		return nil
	}
	if err := tx.Model(&models.TranscriptionProfile{}).
		Where("id IN ?", idsToClear).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("clear duplicate profile defaults: %w", err)
	}
	return nil
}

func enforceLatestDefaultPerUserForSummaryTemplates(tx *gorm.DB) error {
	var rows []defaultStringRow
	if err := tx.Model(&models.SummaryTemplate{}).
		Where("is_default = ?", true).
		Order("user_id ASC, created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load default summary templates: %w", err)
	}

	latestIDByUser := make(map[uint]string)
	for _, row := range rows {
		if _, ok := latestIDByUser[row.UserID]; ok {
			continue
		}
		latestIDByUser[row.UserID] = row.ID
	}

	idsToClear := make([]string, 0, len(rows))
	for _, row := range rows {
		if latestIDByUser[row.UserID] != row.ID {
			idsToClear = append(idsToClear, row.ID)
		}
	}
	if len(idsToClear) == 0 {
		return nil
	}
	if err := tx.Model(&models.SummaryTemplate{}).
		Where("id IN ?", idsToClear).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("clear duplicate summary template defaults: %w", err)
	}
	return nil
}

func enforceLatestDefaultPerUserForLLMProfiles(tx *gorm.DB) error {
	var rows []defaultUintRow
	if err := tx.Model(&models.LLMConfig{}).
		Where("is_default = ?", true).
		Order("user_id ASC, created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load default llm profiles: %w", err)
	}

	latestIDByUser := make(map[uint]uint)
	for _, row := range rows {
		if _, ok := latestIDByUser[row.UserID]; ok {
			continue
		}
		latestIDByUser[row.UserID] = row.ID
	}

	idsToClear := make([]uint, 0, len(rows))
	for _, row := range rows {
		if latestIDByUser[row.UserID] != row.ID {
			idsToClear = append(idsToClear, row.ID)
		}
	}
	if len(idsToClear) == 0 {
		return nil
	}
	if err := tx.Model(&models.LLMConfig{}).
		Where("id IN ?", idsToClear).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("clear duplicate llm defaults: %w", err)
	}
	return nil
}
