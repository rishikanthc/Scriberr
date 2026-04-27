package models

import (
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
	configJSON, err := marshalJSONColumn("summary_templates.config_json", map[string]any{
		"model":                st.Model,
		"include_speaker_info": st.IncludeSpeakerInfo,
	})
	if err != nil {
		return err
	}
	st.ConfigJSON = configJSON
	if st.IsDefault {
		if err := clearOtherDefaultsForUser(tx, &SummaryTemplate{}, st.UserID, st.ID); err != nil {
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
	if err := unmarshalJSONColumn("summary_templates.config_json", st.ConfigJSON, &cfg); err != nil {
		return err
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
	ID                  string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TranscriptionID     string     `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	UserID              uint       `json:"user_id" gorm:"not null;index;default:1"`
	TemplateID          *string    `json:"template_id,omitempty" gorm:"type:varchar(36);index"`
	Title               *string    `json:"title,omitempty" gorm:"type:text"`
	Content             string     `json:"content" gorm:"type:text;not null"`
	Model               string     `json:"model,omitempty" gorm:"column:model_name;type:varchar(255)"`
	Provider            string     `json:"provider,omitempty" gorm:"type:varchar(50)"`
	Status              string     `json:"status,omitempty" gorm:"type:varchar(20);not null;default:'completed'"`
	ErrorMessage        *string    `json:"error_message,omitempty" gorm:"type:text"`
	TranscriptTruncated bool       `json:"transcript_truncated" gorm:"not null;default:false"`
	ContextWindow       int        `json:"context_window" gorm:"not null;default:0"`
	InputCharacters     int        `json:"input_characters" gorm:"not null;default:0"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	FailedAt            *time.Time `json:"failed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

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

// SummaryWidget stores a user-defined extraction widget for generated summaries.
type SummaryWidget struct {
	ID             string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID         uint           `json:"user_id" gorm:"not null;index;default:1"`
	Name           string         `json:"name" gorm:"type:varchar(120);not null"`
	Description    *string        `json:"description,omitempty" gorm:"type:text"`
	AlwaysEnabled  bool           `json:"always_enabled" gorm:"not null;default:false"`
	WhenToUse      *string        `json:"when_to_use,omitempty" gorm:"type:text"`
	ContextSource  string         `json:"context_source" gorm:"type:varchar(20);not null;default:'summary'"`
	Prompt         string         `json:"prompt" gorm:"type:text;not null"`
	RenderMarkdown bool           `json:"render_markdown" gorm:"not null;default:false"`
	DisplayTitle   string         `json:"display_title" gorm:"type:varchar(160);not null"`
	Enabled        bool           `json:"enabled" gorm:"not null;default:true;index"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`
}

func (SummaryWidget) TableName() string { return "summary_widgets" }

func (w *SummaryWidget) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	if w.UserID == 0 {
		w.UserID = primaryUserID
	}
	if w.ContextSource == "" {
		w.ContextSource = "summary"
	}
	return nil
}

// SummaryWidgetRun stores one durable execution of a summary widget.
type SummaryWidgetRun struct {
	ID               string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SummaryID        string     `json:"summary_id" gorm:"type:varchar(36);not null;index"`
	TranscriptionID  string     `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	WidgetID         string     `json:"widget_id" gorm:"type:varchar(36);not null;index"`
	UserID           uint       `json:"user_id" gorm:"not null;index;default:1"`
	WidgetName       string     `json:"widget_name" gorm:"type:varchar(120);not null"`
	DisplayTitle     string     `json:"display_title" gorm:"type:varchar(160);not null"`
	ContextSource    string     `json:"context_source" gorm:"type:varchar(20);not null"`
	RenderMarkdown   bool       `json:"render_markdown" gorm:"not null;default:false"`
	Model            string     `json:"model" gorm:"column:model_name;type:varchar(255)"`
	Provider         string     `json:"provider" gorm:"type:varchar(50)"`
	Status           string     `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	Output           string     `json:"output" gorm:"type:text;not null"`
	ErrorMessage     *string    `json:"error_message,omitempty" gorm:"type:text"`
	ContextTruncated bool       `json:"context_truncated" gorm:"not null;default:false"`
	ContextWindow    int        `json:"context_window" gorm:"not null;default:0"`
	InputCharacters  int        `json:"input_characters" gorm:"not null;default:0"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	FailedAt         *time.Time `json:"failed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	Summary Summary       `json:"summary,omitempty" gorm:"foreignKey:SummaryID;references:ID;constraint:OnDelete:CASCADE"`
	Widget  SummaryWidget `json:"widget,omitempty" gorm:"foreignKey:WidgetID;references:ID"`
}

func (SummaryWidgetRun) TableName() string { return "summary_widget_runs" }

func (r *SummaryWidgetRun) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.UserID == 0 {
		r.UserID = primaryUserID
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	if r.ContextSource == "" {
		r.ContextSource = "summary"
	}
	return nil
}
