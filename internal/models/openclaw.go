package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OpenClawProfile stores remote delivery settings for OpenClaw automation.
type OpenClawProfile struct {
	ID        string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string         `json:"name" gorm:"type:varchar(255);not null"`
	IP        string         `json:"ip" gorm:"type:varchar(255);not null"`         // Accepts host or user@host
	SSHKey    string         `json:"ssh_key,omitempty" gorm:"type:text;not null"`  // Private key content
	HookKey   string         `json:"hook_key,omitempty" gorm:"type:text;not null"` // OpenClaw hook bearer token
	Message   string         `json:"message" gorm:"type:text;not null"`            // Default hook message
	HookName  string         `json:"hook_name" gorm:"type:varchar(255);not null"`  // OpenClaw agent name
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`
}

// BeforeCreate sets a UUID if not already set.
func (p *OpenClawProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}
