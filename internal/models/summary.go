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

