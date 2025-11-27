package models

import (
	"time"
)

// Note represents an annotation attached to a transcription
type Note struct {
	ID              string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TranscriptionID string `json:"transcription_id" gorm:"type:varchar(36);not null;index"`

	// Indexed selection into transcript by word positions
	StartWordIndex int `json:"start_word_index" gorm:"type:int;not null"`
	EndWordIndex   int `json:"end_word_index" gorm:"type:int;not null"`

	// Time bounds for the selection (in seconds)
	StartTime float64 `json:"start_time" gorm:"type:real;not null"`
	EndTime   float64 `json:"end_time" gorm:"type:real;not null"`

	// The exact quoted text chosen by the user
	Quote string `json:"quote" gorm:"type:text;not null"`

	// The user's note content (markdown/plain)
	Content string `json:"content" gorm:"type:text;not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Transcription TranscriptionJob `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;constraint:OnDelete:CASCADE"`
}
