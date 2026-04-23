package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SummaryTemplate represents a saved summarization template.
type SummaryTemplate struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID      uint           `json:"user_id" gorm:"not null;index;default:1"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	Prompt      string         `json:"prompt" gorm:"type:text;not null"`
	Description *string        `json:"description,omitempty" gorm:"type:text"`
	ConfigJSON  string         `json:"-" gorm:"column:config_json;type:json"`
	IsDefault   bool           `json:"is_default" gorm:"not null;default:false;index"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	Model              string `json:"model,omitempty" gorm:"-"`
	IncludeSpeakerInfo bool   `json:"include_speaker_info,omitempty" gorm:"-"`
}

func (SummaryTemplate) TableName() string { return "summary_templates" }

func (st *SummaryTemplate) BeforeCreate(tx *gorm.DB) error {
	if st.ID == "" {
		st.ID = uuid.New().String()
	}
	return st.BeforeSave(tx)
}

func (st *SummaryTemplate) BeforeSave(tx *gorm.DB) error {
	if st.UserID == 0 {
		st.UserID = primaryUserID
	}
	bytes, _ := json.Marshal(map[string]any{
		"model":                st.Model,
		"include_speaker_info": st.IncludeSpeakerInfo,
	})
	st.ConfigJSON = string(bytes)
	if st.IsDefault {
		if err := tx.Model(&SummaryTemplate{}).Where("id != ?", st.ID).Update("is_default", false).Error; err != nil {
			return err
		}
	}
	return nil
}

func (st *SummaryTemplate) AfterFind(tx *gorm.DB) error {
	if st.ConfigJSON == "" {
		return nil
	}
	var cfg struct {
		Model              string `json:"model,omitempty"`
		IncludeSpeakerInfo bool   `json:"include_speaker_info,omitempty"`
	}
	if err := json.Unmarshal([]byte(st.ConfigJSON), &cfg); err != nil {
		return nil
	}
	st.Model = cfg.Model
	st.IncludeSpeakerInfo = cfg.IncludeSpeakerInfo
	return nil
}

// SummarySetting stores summary preferences in repository-managed user settings.
type SummarySetting struct {
	DefaultModel string `json:"default_model"`
}

// Summary stores a generated summary linked to a transcription.
type Summary struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TranscriptionID string    `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	UserID          uint      `json:"user_id" gorm:"not null;index;default:1"`
	TemplateID      *string   `json:"template_id,omitempty" gorm:"type:varchar(36);index"`
	Title           *string   `json:"title,omitempty" gorm:"type:text"`
	Content         string    `json:"content" gorm:"type:text;not null"`
	Model           string    `json:"model,omitempty" gorm:"column:model_name;type:varchar(255)"`
	Provider        string    `json:"provider,omitempty" gorm:"type:varchar(50)"`
	Status          string    `json:"status,omitempty" gorm:"type:varchar(20);not null;default:'completed'"`
	ErrorMessage    *string   `json:"error_message,omitempty" gorm:"type:text"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Transcription TranscriptionJob `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Summary) TableName() string { return "summaries" }

func (s *Summary) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.UserID == 0 {
		s.UserID = primaryUserID
	}
	if s.Status == "" {
		s.Status = "completed"
	}
	return nil
}
