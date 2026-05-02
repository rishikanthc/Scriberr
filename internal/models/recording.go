package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RecordingStatus string

const (
	RecordingStatusRecording  RecordingStatus = "recording"
	RecordingStatusStopping   RecordingStatus = "stopping"
	RecordingStatusFinalizing RecordingStatus = "finalizing"
	RecordingStatusReady      RecordingStatus = "ready"
	RecordingStatusFailed     RecordingStatus = "failed"
	RecordingStatusCanceled   RecordingStatus = "canceled"
	RecordingStatusExpired    RecordingStatus = "expired"
)

type RecordingSourceKind string

const (
	RecordingSourceKindMicrophone RecordingSourceKind = "microphone"
	RecordingSourceKindTab        RecordingSourceKind = "tab"
	RecordingSourceKindSystem     RecordingSourceKind = "system"
)

// RecordingSession stores the durable lifecycle for browser-captured audio.
type RecordingSession struct {
	ID                       string              `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID                   uint                `json:"user_id" gorm:"not null;index;default:1"`
	Title                    *string             `json:"title,omitempty" gorm:"type:text"`
	Status                   RecordingStatus     `json:"status" gorm:"type:varchar(20);not null;default:'recording';index"`
	SourceKind               RecordingSourceKind `json:"source_kind" gorm:"type:varchar(20);not null;default:'microphone'"`
	MimeType                 string              `json:"mime_type" gorm:"type:varchar(120);not null"`
	Codec                    *string             `json:"codec,omitempty" gorm:"type:varchar(60)"`
	ChunkDurationMs          *int64              `json:"chunk_duration_ms,omitempty" gorm:"type:integer"`
	ExpectedFinalIndex       *int                `json:"expected_final_index,omitempty" gorm:"type:integer"`
	ReceivedChunks           int                 `json:"received_chunks" gorm:"not null;default:0"`
	ReceivedBytes            int64               `json:"received_bytes" gorm:"not null;default:0"`
	DurationMs               *int64              `json:"duration_ms,omitempty" gorm:"type:integer"`
	FileID                   *string             `json:"file_id,omitempty" gorm:"type:varchar(36);index"`
	TranscriptionID          *string             `json:"transcription_id,omitempty" gorm:"type:varchar(36);index"`
	AutoTranscribe           bool                `json:"auto_transcribe" gorm:"not null;default:false"`
	ProfileID                *string             `json:"profile_id,omitempty" gorm:"type:varchar(36);index"`
	TranscriptionOptionsJSON string              `json:"-" gorm:"column:transcription_options_json;type:json;not null;default:'{}'"`
	StartedAt                time.Time           `json:"started_at" gorm:"not null"`
	StoppedAt                *time.Time          `json:"stopped_at,omitempty"`
	FinalizeQueuedAt         *time.Time          `json:"finalize_queued_at,omitempty"`
	FinalizeStartedAt        *time.Time          `json:"finalize_started_at,omitempty"`
	CompletedAt              *time.Time          `json:"completed_at,omitempty"`
	FailedAt                 *time.Time          `json:"failed_at,omitempty"`
	ExpiresAt                *time.Time          `json:"expires_at,omitempty" gorm:"index"`
	LastError                *string             `json:"last_error,omitempty" gorm:"type:text"`
	Progress                 float64             `json:"progress" gorm:"not null;default:0"`
	ProgressStage            string              `json:"progress_stage,omitempty" gorm:"type:varchar(50)"`
	ClaimedBy                *string             `json:"claimed_by,omitempty" gorm:"type:varchar(128)"`
	ClaimExpiresAt           *time.Time          `json:"claim_expires_at,omitempty" gorm:"index"`
	MetadataJSON             string              `json:"-" gorm:"column:metadata_json;type:json;not null;default:'{}'"`
	CreatedAt                time.Time           `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                time.Time           `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt                gorm.DeletedAt      `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	User          User                  `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	File          *TranscriptionJob     `json:"file,omitempty" gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:SET NULL"`
	Transcription *TranscriptionJob     `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;references:ID;constraint:OnDelete:SET NULL"`
	Profile       *TranscriptionProfile `json:"profile,omitempty" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:SET NULL"`
	Chunks        []RecordingChunk      `json:"chunks,omitempty" gorm:"foreignKey:SessionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (RecordingSession) TableName() string { return "recording_sessions" }

func (s *RecordingSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return s.BeforeSave(tx)
}

func (s *RecordingSession) BeforeSave(tx *gorm.DB) error {
	if s.UserID == 0 {
		s.UserID = primaryUserID
	}
	if s.Status == "" {
		s.Status = RecordingStatusRecording
	}
	if !validRecordingStatus(s.Status) {
		return fmt.Errorf("recording session status is invalid")
	}
	if s.SourceKind == "" {
		s.SourceKind = RecordingSourceKindMicrophone
	}
	if !validRecordingSourceKind(s.SourceKind) {
		return fmt.Errorf("recording session source_kind is invalid")
	}
	s.MimeType = strings.TrimSpace(s.MimeType)
	if s.MimeType == "" {
		return fmt.Errorf("recording session mime_type is required")
	}
	if s.StartedAt.IsZero() {
		s.StartedAt = time.Now()
	}
	if s.TranscriptionOptionsJSON == "" {
		s.TranscriptionOptionsJSON = "{}"
	}
	if s.MetadataJSON == "" {
		s.MetadataJSON = "{}"
	}
	if s.ReceivedChunks < 0 {
		return fmt.Errorf("recording session received_chunks cannot be negative")
	}
	if s.ReceivedBytes < 0 {
		return fmt.Errorf("recording session received_bytes cannot be negative")
	}
	if s.Progress < 0 || s.Progress > 1 {
		return fmt.Errorf("recording session progress must be between 0 and 1")
	}
	return nil
}

func validRecordingStatus(status RecordingStatus) bool {
	switch status {
	case RecordingStatusRecording, RecordingStatusStopping, RecordingStatusFinalizing, RecordingStatusReady, RecordingStatusFailed, RecordingStatusCanceled, RecordingStatusExpired:
		return true
	default:
		return false
	}
}

func validRecordingSourceKind(kind RecordingSourceKind) bool {
	switch kind {
	case RecordingSourceKindMicrophone, RecordingSourceKindTab, RecordingSourceKindSystem:
		return true
	default:
		return false
	}
}

// RecordingChunk stores one persisted MediaRecorder chunk for a session.
type RecordingChunk struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SessionID  string    `json:"session_id" gorm:"type:varchar(36);not null;index"`
	UserID     uint      `json:"user_id" gorm:"not null;index;default:1"`
	ChunkIndex int       `json:"chunk_index" gorm:"not null"`
	Path       string    `json:"-" gorm:"type:text;not null"`
	MimeType   string    `json:"mime_type" gorm:"type:varchar(120);not null"`
	SHA256     *string   `json:"sha256,omitempty" gorm:"type:varchar(64)"`
	SizeBytes  int64     `json:"size_bytes" gorm:"not null"`
	DurationMs *int64    `json:"duration_ms,omitempty" gorm:"type:integer"`
	ReceivedAt time.Time `json:"received_at" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`

	User    User             `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Session RecordingSession `json:"session,omitempty" gorm:"foreignKey:SessionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (RecordingChunk) TableName() string { return "recording_chunks" }

func (c *RecordingChunk) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.UserID == 0 {
		c.UserID = primaryUserID
	}
	c.SessionID = strings.TrimSpace(c.SessionID)
	if c.SessionID == "" {
		return fmt.Errorf("recording chunk session_id is required")
	}
	if c.ChunkIndex < 0 {
		return fmt.Errorf("recording chunk index cannot be negative")
	}
	c.Path = strings.TrimSpace(c.Path)
	if c.Path == "" {
		return fmt.Errorf("recording chunk path is required")
	}
	c.MimeType = strings.TrimSpace(c.MimeType)
	if c.MimeType == "" {
		return fmt.Errorf("recording chunk mime_type is required")
	}
	if c.SizeBytes < 0 {
		return fmt.Errorf("recording chunk size_bytes cannot be negative")
	}
	if c.ReceivedAt.IsZero() {
		c.ReceivedAt = time.Now()
	}
	return nil
}
