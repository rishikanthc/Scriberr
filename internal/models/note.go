package models

import (
	"time"

	"gorm.io/gorm"
)

// Note represents an annotation attached to a transcription.
type Note struct {
	ID              string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID          uint           `json:"user_id" gorm:"not null;index;default:1"`
	TranscriptionID string         `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	Content         string         `json:"content" gorm:"type:text;not null"`
	StartMS         int64          `json:"start_ms" gorm:"column:start_ms;type:integer;not null;default:0"`
	EndMS           int64          `json:"end_ms" gorm:"column:end_ms;type:integer;not null;default:0"`
	MetadataJSON    string         `json:"-" gorm:"column:metadata_json;type:json"`
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	StartWordIndex int     `json:"start_word_index" gorm:"-"`
	EndWordIndex   int     `json:"end_word_index" gorm:"-"`
	StartTime      float64 `json:"start_time" gorm:"-"`
	EndTime        float64 `json:"end_time" gorm:"-"`
	Quote          string  `json:"quote" gorm:"-"`

	Transcription TranscriptionJob `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Note) TableName() string { return "notes" }

func (n *Note) BeforeCreate(tx *gorm.DB) error {
	if n.UserID == 0 {
		n.UserID = primaryUserID
	}
	return n.syncColumnsFromCompat()
}

func (n *Note) BeforeSave(tx *gorm.DB) error {
	return n.syncColumnsFromCompat()
}

func (n *Note) AfterFind(tx *gorm.DB) error {
	n.StartTime = float64(n.StartMS) / 1000
	n.EndTime = float64(n.EndMS) / 1000
	if n.MetadataJSON == "" {
		return nil
	}
	var metadata struct {
		StartWordIndex int    `json:"start_word_index,omitempty"`
		EndWordIndex   int    `json:"end_word_index,omitempty"`
		Quote          string `json:"quote,omitempty"`
	}
	if err := unmarshalJSONColumn("notes.metadata_json", n.MetadataJSON, &metadata); err != nil {
		return err
	}
	n.StartWordIndex = metadata.StartWordIndex
	n.EndWordIndex = metadata.EndWordIndex
	n.Quote = metadata.Quote
	return nil
}

func (n *Note) syncColumnsFromCompat() error {
	n.StartMS = int64(n.StartTime * 1000)
	n.EndMS = int64(n.EndTime * 1000)
	metadataJSON, err := marshalJSONColumn("notes.metadata_json", map[string]any{
		"start_word_index": n.StartWordIndex,
		"end_word_index":   n.EndWordIndex,
		"quote":            n.Quote,
	})
	if err != nil {
		return err
	}
	n.MetadataJSON = metadataJSON
	return nil
}
