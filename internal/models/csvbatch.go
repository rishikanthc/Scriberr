package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CSVBatchStatus represents the status of a CSV batch job
type CSVBatchStatus string

const (
	CSVBatchStatusPending    CSVBatchStatus = "pending"
	CSVBatchStatusProcessing CSVBatchStatus = "processing"
	CSVBatchStatusCompleted  CSVBatchStatus = "completed"
	CSVBatchStatusFailed     CSVBatchStatus = "failed"
	CSVBatchStatusCancelled  CSVBatchStatus = "cancelled"
)

// CSVRowStatus represents the status of an individual CSV row
type CSVRowStatus string

const (
	CSVRowStatusPending    CSVRowStatus = "pending"
	CSVRowStatusProcessing CSVRowStatus = "processing"
	CSVRowStatusCompleted  CSVRowStatus = "completed"
	CSVRowStatusFailed     CSVRowStatus = "failed"
	CSVRowStatusSkipped    CSVRowStatus = "skipped"
)

// CSVBatch represents a batch job for processing multiple YouTube URLs from a CSV file
type CSVBatch struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string         `json:"name" gorm:"type:varchar(255)"`
	Status      CSVBatchStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	TotalRows   int            `json:"total_rows" gorm:"type:int;not null;default:0"`
	CurrentRow  int            `json:"current_row" gorm:"type:int;not null;default:0"`
	SuccessRows int            `json:"success_rows" gorm:"type:int;not null;default:0"`
	FailedRows  int            `json:"failed_rows" gorm:"type:int;not null;default:0"`
	OutputDir   string         `json:"output_dir" gorm:"type:text;not null"`
	CSVFilePath string         `json:"csv_file_path" gorm:"type:text;not null"`

	// Transcription parameters (from profile or custom)
	ProfileID  *string        `json:"profile_id,omitempty" gorm:"type:varchar(36)"`
	Parameters WhisperXParams `json:"parameters" gorm:"embedded;embeddedPrefix:param_"`

	// Error tracking
	ErrorMessage *string `json:"error_message,omitempty" gorm:"type:text"`

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty" gorm:"type:datetime"`
	CompletedAt *time.Time `json:"completed_at,omitempty" gorm:"type:datetime"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Rows []CSVBatchRow `json:"rows,omitempty" gorm:"foreignKey:BatchID"`
}

// BeforeCreate sets the ID if not already set
func (cb *CSVBatch) BeforeCreate(tx *gorm.DB) error {
	if cb.ID == "" {
		cb.ID = uuid.New().String()
	}
	return nil
}

// CSVBatchRow represents a single row from the CSV file
type CSVBatchRow struct {
	ID      uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	BatchID string `json:"batch_id" gorm:"type:varchar(36);not null;index"`
	RowID   int    `json:"row_id" gorm:"type:int;not null"` // 1-indexed row number from CSV

	// URL and video info
	URL            string  `json:"url" gorm:"type:text;not null"`
	VideoTitle     *string `json:"video_title,omitempty" gorm:"type:text"`
	VideoFilename  *string `json:"video_filename,omitempty" gorm:"type:varchar(255)"`
	AudioFilePath  *string `json:"audio_file_path,omitempty" gorm:"type:text"`
	OutputFilePath *string `json:"output_file_path,omitempty" gorm:"type:text"` // JSON output path

	// Status and progress
	Status       CSVRowStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	ErrorMessage *string      `json:"error_message,omitempty" gorm:"type:text"`

	// Linked transcription job (if created)
	TranscriptionJobID *string `json:"transcription_job_id,omitempty" gorm:"type:varchar(36)"`

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty" gorm:"type:datetime"`
	CompletedAt *time.Time `json:"completed_at,omitempty" gorm:"type:datetime"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Batch            CSVBatch          `json:"batch,omitempty" gorm:"foreignKey:BatchID"`
	TranscriptionJob *TranscriptionJob `json:"transcription_job,omitempty" gorm:"foreignKey:TranscriptionJobID"`
}

// GetProgressPercentage returns the progress percentage of the batch
func (cb *CSVBatch) GetProgressPercentage() float64 {
	if cb.TotalRows == 0 {
		return 0
	}
	return float64(cb.SuccessRows+cb.FailedRows) / float64(cb.TotalRows) * 100
}

// IsComplete returns true if all rows have been processed
func (cb *CSVBatch) IsComplete() bool {
	return cb.SuccessRows+cb.FailedRows >= cb.TotalRows
}
