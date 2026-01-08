package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BatchStatus represents the processing status of a CSV batch
type BatchStatus string

const (
	BatchPending    BatchStatus = "pending"
	BatchProcessing BatchStatus = "processing"
	BatchCompleted  BatchStatus = "completed"
	BatchFailed     BatchStatus = "failed"
	BatchCancelled  BatchStatus = "cancelled"
)

// RowStatus represents the processing status of a single URL
type RowStatus string

const (
	RowPending    RowStatus = "pending"
	RowProcessing RowStatus = "processing"
	RowCompleted  RowStatus = "completed"
	RowFailed     RowStatus = "failed"
)

// CSVBatch represents a batch job for processing YouTube URLs from a CSV file
type CSVBatch struct {
	ID        string      `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string      `json:"name" gorm:"type:varchar(255);not null"`
	Status    BatchStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"`
	OutputDir string      `json:"output_dir" gorm:"type:text;not null"`

	// Progress tracking
	TotalRows   int `json:"total_rows" gorm:"type:int;not null;default:0"`
	CurrentRow  int `json:"current_row" gorm:"type:int;not null;default:0"`
	SuccessRows int `json:"success_rows" gorm:"type:int;not null;default:0"`
	FailedRows  int `json:"failed_rows" gorm:"type:int;not null;default:0"`

	// Transcription configuration
	ProfileID  *string        `json:"profile_id,omitempty" gorm:"type:varchar(36)"`
	Parameters WhisperXParams `json:"parameters" gorm:"embedded;embeddedPrefix:param_"`

	// Metadata
	ErrorMessage *string    `json:"error_message,omitempty" gorm:"type:text"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	Rows []CSVBatchRow `json:"rows,omitempty" gorm:"foreignKey:BatchID;references:ID"`
}

// BeforeCreate generates UUID if not set
func (b *CSVBatch) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// Progress returns the completion percentage
func (b *CSVBatch) Progress() float64 {
	if b.TotalRows == 0 {
		return 0
	}
	return float64(b.SuccessRows+b.FailedRows) / float64(b.TotalRows) * 100
}

// CSVBatchRow represents a single URL entry in a batch
type CSVBatchRow struct {
	ID      uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	BatchID string `json:"batch_id" gorm:"type:varchar(36);not null;index;uniqueIndex:idx_batch_row"`
	RowNum  int    `json:"row_num" gorm:"type:int;not null;uniqueIndex:idx_batch_row"` // 1-indexed position in CSV

	// Video information
	URL        string  `json:"url" gorm:"type:text;not null"`
	Title      *string `json:"title,omitempty" gorm:"type:text"`
	Filename   *string `json:"filename,omitempty" gorm:"type:varchar(255)"`
	AudioPath  *string `json:"audio_path,omitempty" gorm:"type:text"`
	OutputPath *string `json:"output_path,omitempty" gorm:"type:text"`

	// Status
	Status       RowStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"`
	ErrorMessage *string   `json:"error_message,omitempty" gorm:"type:text"`

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}
