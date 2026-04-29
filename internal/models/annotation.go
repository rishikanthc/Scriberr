package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnnotationKind string

const (
	AnnotationKindHighlight AnnotationKind = "highlight"
	AnnotationKindNote      AnnotationKind = "note"
)

const (
	AnnotationStatusActive = "active"
	AnnotationStatusStale  = "stale"
)

// TranscriptAnnotation stores a user-owned highlight or note anchored to a transcript range.
type TranscriptAnnotation struct {
	ID              string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID          uint           `json:"user_id" gorm:"not null;index;default:1"`
	TranscriptionID string         `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	Kind            AnnotationKind `json:"kind" gorm:"type:varchar(20);not null;index"`
	Content         *string        `json:"content,omitempty" gorm:"type:text"`
	Color           *string        `json:"color,omitempty" gorm:"type:varchar(32)"`
	Quote           string         `json:"quote" gorm:"type:text;not null"`
	AnchorStartMS   int64          `json:"anchor_start_ms" gorm:"type:integer;not null"`
	AnchorEndMS     int64          `json:"anchor_end_ms" gorm:"type:integer;not null"`
	AnchorStartWord *int           `json:"anchor_start_word,omitempty" gorm:"type:integer"`
	AnchorEndWord   *int           `json:"anchor_end_word,omitempty" gorm:"type:integer"`
	AnchorStartChar *int           `json:"anchor_start_char,omitempty" gorm:"type:integer"`
	AnchorEndChar   *int           `json:"anchor_end_char,omitempty" gorm:"type:integer"`
	AnchorTextHash  *string        `json:"anchor_text_hash,omitempty" gorm:"type:varchar(128)"`
	Status          string         `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	MetadataJSON    string         `json:"-" gorm:"column:metadata_json;type:json;not null;default:'{}'"`
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	Transcription TranscriptionJob            `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;references:ID;constraint:OnDelete:CASCADE"`
	Entries       []TranscriptAnnotationEntry `json:"entries,omitempty" gorm:"foreignKey:AnnotationID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TranscriptAnnotation) TableName() string { return "transcript_annotations" }

func (a *TranscriptAnnotation) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return a.BeforeSave(tx)
}

func (a *TranscriptAnnotation) BeforeSave(tx *gorm.DB) error {
	if a.UserID == 0 {
		a.UserID = primaryUserID
	}
	if a.Status == "" {
		a.Status = AnnotationStatusActive
	}
	if a.MetadataJSON == "" {
		a.MetadataJSON = "{}"
	}
	if !validAnnotationKind(a.Kind) {
		return fmt.Errorf("transcript annotation kind is invalid")
	}
	if a.AnchorEndMS < a.AnchorStartMS {
		return fmt.Errorf("transcript annotation anchor end must be greater than or equal to start")
	}
	return nil
}

func validAnnotationKind(kind AnnotationKind) bool {
	switch kind {
	case AnnotationKindHighlight, AnnotationKindNote:
		return true
	default:
		return false
	}
}

// TranscriptAnnotationEntry stores one note bubble inside a root note annotation thread.
type TranscriptAnnotationEntry struct {
	ID           string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	AnnotationID string         `json:"annotation_id" gorm:"type:varchar(36);not null;index"`
	UserID       uint           `json:"user_id" gorm:"not null;index;default:1"`
	Content      string         `json:"content" gorm:"type:text;not null"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	Annotation TranscriptAnnotation `json:"annotation,omitempty" gorm:"foreignKey:AnnotationID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TranscriptAnnotationEntry) TableName() string { return "transcript_annotation_entries" }

func (e *TranscriptAnnotationEntry) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return e.BeforeSave(tx)
}

func (e *TranscriptAnnotationEntry) BeforeSave(tx *gorm.DB) error {
	if e.UserID == 0 {
		e.UserID = primaryUserID
	}
	e.Content = strings.TrimSpace(e.Content)
	if e.Content == "" {
		return fmt.Errorf("transcript annotation entry content is required")
	}
	return nil
}
