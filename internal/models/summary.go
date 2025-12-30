package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SummaryTemplate represents a saved summarization prompt/template
type SummaryTemplate struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Description *string   `json:"description,omitempty" gorm:"type:text"`
	Model       string    `json:"model" gorm:"type:varchar(255);not null;default:''"`
	Prompt      string    `json:"prompt" gorm:"type:text;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (st *SummaryTemplate) BeforeCreate(tx *gorm.DB) error {
	if st.ID == "" {
		st.ID = uuid.New().String()
	}
	return nil
}

// SummarySetting stores global settings for summarization (single row)
type SummarySetting struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	DefaultModel string    `json:"default_model" gorm:"type:varchar(255);not null;default:''"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Summary stores a generated summary linked to a transcription
type Summary struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TranscriptionID string    `json:"transcription_id" gorm:"type:varchar(36);index;not null"`
	TemplateID      *string   `json:"template_id,omitempty" gorm:"type:varchar(36)"`
	Model           string    `json:"model" gorm:"type:varchar(255);not null"`
	Content         string    `json:"content" gorm:"type:text;not null"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Transcription TranscriptionJob `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;constraint:OnDelete:CASCADE"`
}

// BeforeCreate ensures Summary has a UUID primary key
func (s *Summary) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
