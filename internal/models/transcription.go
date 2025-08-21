package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TranscriptionJob represents a transcription job record
type TranscriptionJob struct {
	ID           string          `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Title        *string         `json:"title,omitempty" gorm:"type:text"`
	Status       JobStatus       `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	AudioPath    string          `json:"audio_path" gorm:"type:text;not null"`
	Transcript   *string         `json:"transcript,omitempty" gorm:"type:text"`
	Diarization  bool            `json:"diarization" gorm:"type:boolean;default:false"`
	Summary      *string         `json:"summary,omitempty" gorm:"type:text"`
	ErrorMessage *string         `json:"error_message,omitempty" gorm:"type:text"`
	CreatedAt    time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	
	// WhisperX parameters
	Parameters WhisperXParams `json:"parameters" gorm:"embedded"`
}

// JobStatus represents the status of a transcription job
type JobStatus string

const (
	StatusPending     JobStatus = "pending"
	StatusProcessing  JobStatus = "processing"
	StatusCompleted   JobStatus = "completed"
	StatusFailed      JobStatus = "failed"
)

// WhisperXParams contains parameters for WhisperX transcription
type WhisperXParams struct {
	Model           string  `json:"model" gorm:"type:varchar(50);default:'base'"`
	Language        *string `json:"language,omitempty" gorm:"type:varchar(10)"`
	BatchSize       int     `json:"batch_size" gorm:"type:int;default:16"`
	ComputeType     string  `json:"compute_type" gorm:"type:varchar(20);default:'int8'"`
	Device          string  `json:"device" gorm:"type:varchar(20);default:'cpu'"`
	VadFilter       bool    `json:"vad_filter" gorm:"type:boolean;default:false"`
	VadOnset        float64 `json:"vad_onset" gorm:"type:real;default:0.500"`
	VadOffset       float64 `json:"vad_offset" gorm:"type:real;default:0.363"`
	MinSpeakers     *int    `json:"min_speakers,omitempty" gorm:"type:int"`
	MaxSpeakers     *int    `json:"max_speakers,omitempty" gorm:"type:int"`
}

// BeforeCreate sets the ID if not already set
func (tj *TranscriptionJob) BeforeCreate(tx *gorm.DB) error {
	if tj.ID == "" {
		tj.ID = uuid.New().String()
	}
	return nil
}

// User represents a user for authentication
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null;type:varchar(50)"`
	Password  string    `json:"-" gorm:"not null;type:varchar(255)"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// APIKey represents an API key for external authentication
type APIKey struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Key         string    `json:"key" gorm:"uniqueIndex;not null;type:varchar(255)"`
	Name        string    `json:"name" gorm:"not null;type:varchar(100)"`
	Description *string   `json:"description,omitempty" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"type:boolean;default:true"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate sets the API key if not already set
func (ak *APIKey) BeforeCreate(tx *gorm.DB) error {
	if ak.Key == "" {
		ak.Key = uuid.New().String()
	}
	return nil
}