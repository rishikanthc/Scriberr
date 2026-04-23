package database

import (
	"fmt"

	"scriberr/internal/models"

	"gorm.io/gorm"
)

const latestSchemaVersion = 1

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
	if err := tx.AutoMigrate(schemaModels...); err != nil {
		return fmt.Errorf("auto migrate target schema: %w", err)
	}
	if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("auto migrate schema state: %w", err)
	}

	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_speaker_mappings_unique ON speaker_mappings(transcription_id, original_speaker)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_tracks_unique ON transcription_tracks(transcription_id, track_index)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_executions_unique ON transcription_executions(transcription_id, execution_number)`,
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
