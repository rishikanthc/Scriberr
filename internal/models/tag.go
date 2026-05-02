package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AudioTag is a user-owned label that can be assigned to audio/transcription records.
type AudioTag struct {
	ID             string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID         uint           `json:"user_id" gorm:"not null;index;default:1"`
	Name           string         `json:"name" gorm:"type:varchar(120);not null"`
	NormalizedName string         `json:"normalized_name" gorm:"type:varchar(120);not null;index"`
	Color          *string        `json:"color,omitempty" gorm:"type:varchar(32)"`
	Description    *string        `json:"description,omitempty" gorm:"type:text"`
	MetadataJSON   string         `json:"-" gorm:"column:metadata_json;type:json;not null;default:'{}'"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	Assignments []AudioTagAssignment `json:"assignments,omitempty" gorm:"foreignKey:TagID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AudioTag) TableName() string { return "audio_tags" }

func (t *AudioTag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return t.BeforeSave(tx)
}

func (t *AudioTag) BeforeSave(tx *gorm.DB) error {
	if t.UserID == 0 {
		t.UserID = primaryUserID
	}
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return fmt.Errorf("audio tag name is required")
	}
	if t.NormalizedName == "" {
		t.NormalizedName = NormalizeAudioTagName(t.Name)
	}
	if t.NormalizedName == "" {
		return fmt.Errorf("audio tag normalized name is required")
	}
	if t.MetadataJSON == "" {
		t.MetadataJSON = "{}"
	}
	return nil
}

// NormalizeAudioTagName returns the canonical comparison key for a tag display name.
func NormalizeAudioTagName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(name))), " ")
}

// AudioTagAssignment links one tag to one audio/transcription record.
type AudioTagAssignment struct {
	ID              string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID          uint           `json:"user_id" gorm:"not null;index;default:1"`
	TagID           string         `json:"tag_id" gorm:"type:varchar(36);not null;index"`
	TranscriptionID string         `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	Tag           AudioTag         `json:"tag,omitempty" gorm:"foreignKey:TagID;references:ID;constraint:OnDelete:CASCADE"`
	Transcription TranscriptionJob `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AudioTagAssignment) TableName() string { return "audio_tag_assignments" }

func (a *AudioTagAssignment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.UserID == 0 {
		a.UserID = primaryUserID
	}
	if strings.TrimSpace(a.TagID) == "" {
		return fmt.Errorf("audio tag assignment tag_id is required")
	}
	if strings.TrimSpace(a.TranscriptionID) == "" {
		return fmt.Errorf("audio tag assignment transcription_id is required")
	}
	return nil
}
