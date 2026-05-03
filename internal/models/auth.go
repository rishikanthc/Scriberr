package models

import (
	"time"

	"gorm.io/gorm"
)

type userSettings struct {
	DefaultProfileID         *string `json:"default_profile_id,omitempty"`
	AutoTranscriptionEnabled bool    `json:"auto_transcription_enabled,omitempty"`
	AutoRenameEnabled        bool    `json:"auto_rename_enabled,omitempty"`
	SummaryDefaultModel      string  `json:"summary_default_model,omitempty"`
}

// RefreshToken represents a persistent refresh token for rotating access.
type RefreshToken struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"user_id" gorm:"not null;index"`
	Hashed    string     `json:"-" gorm:"column:token_hash;not null;uniqueIndex;type:varchar(128)"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" gorm:"index"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`

	Revoked bool `json:"revoked" gorm:"-"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (rt *RefreshToken) AfterFind(tx *gorm.DB) error {
	rt.Revoked = rt.RevokedAt != nil
	return nil
}

// User represents an authenticated user.
type User struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Username     string         `json:"username" gorm:"uniqueIndex;not null;type:varchar(50)"`
	Password     string         `json:"-" gorm:"column:password_hash;not null;type:varchar(255)"`
	Email        *string        `json:"email,omitempty" gorm:"uniqueIndex;type:varchar(255)"`
	DisplayName  *string        `json:"display_name,omitempty" gorm:"type:varchar(255)"`
	Role         string         `json:"role" gorm:"type:varchar(20);not null;default:'admin'"`
	SettingsJSON string         `json:"-" gorm:"column:settings_json;type:json"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	DefaultProfileID         *string `json:"default_profile_id,omitempty" gorm:"-"`
	AutoTranscriptionEnabled bool    `json:"auto_transcription_enabled" gorm:"-"`
	AutoRenameEnabled        bool    `json:"auto_rename_enabled" gorm:"-"`
	SummaryDefaultModel      string  `json:"-" gorm:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error { return u.BeforeSave(tx) }

func (u *User) BeforeSave(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = "admin"
	}
	settingsJSON, err := marshalJSONColumn("users.settings_json", userSettings{
		DefaultProfileID:         u.DefaultProfileID,
		AutoTranscriptionEnabled: u.AutoTranscriptionEnabled,
		AutoRenameEnabled:        u.AutoRenameEnabled,
		SummaryDefaultModel:      u.SummaryDefaultModel,
	})
	if err != nil {
		return err
	}
	u.SettingsJSON = settingsJSON
	return nil
}

func (u *User) AfterFind(tx *gorm.DB) error {
	if u.SettingsJSON == "" {
		return nil
	}
	var settings userSettings
	if err := unmarshalJSONColumn("users.settings_json", u.SettingsJSON, &settings); err != nil {
		return err
	}
	u.DefaultProfileID = settings.DefaultProfileID
	u.AutoTranscriptionEnabled = settings.AutoTranscriptionEnabled
	u.AutoRenameEnabled = settings.AutoRenameEnabled
	u.SummaryDefaultModel = settings.SummaryDefaultModel
	return nil
}

// APIKey represents an API key for external authentication.
type APIKey struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	UserID       uint       `json:"user_id" gorm:"not null;index"`
	Name         string     `json:"name" gorm:"not null;type:varchar(100)"`
	KeyPrefix    string     `json:"key_prefix" gorm:"not null;type:varchar(16);index"`
	KeyHash      string     `json:"-" gorm:"column:key_hash;not null;uniqueIndex;type:varchar(128)"`
	MetadataJSON string     `json:"-" gorm:"column:metadata_json;type:json"`
	LastUsed     *time.Time `json:"last_used,omitempty" gorm:"column:last_used_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty" gorm:"index"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" gorm:"index"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`

	Key         string    `json:"key,omitempty" gorm:"-"`
	Description *string   `json:"description,omitempty" gorm:"-"`
	IsActive    bool      `json:"is_active" gorm:"-"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"-"`
}

func (APIKey) TableName() string { return "api_keys" }

func (ak *APIKey) BeforeCreate(tx *gorm.DB) error { return ak.BeforeSave(tx) }

func (ak *APIKey) BeforeSave(tx *gorm.DB) error {
	if ak.KeyHash == "" && ak.Key != "" {
		ak.KeyHash = sha256Hex(ak.Key)
	}
	if ak.KeyPrefix == "" && ak.Key != "" {
		if len(ak.Key) <= 8 {
			ak.KeyPrefix = ak.Key
		} else {
			ak.KeyPrefix = ak.Key[:8]
		}
	}
	metadataJSON, err := marshalJSONColumn("api_keys.metadata_json", map[string]any{
		"description": ak.Description,
	})
	if err != nil {
		return err
	}
	ak.MetadataJSON = metadataJSON
	return nil
}

func (ak *APIKey) AfterFind(tx *gorm.DB) error {
	ak.IsActive = ak.RevokedAt == nil
	ak.UpdatedAt = ak.CreatedAt
	if ak.MetadataJSON != "" {
		var metadata struct {
			Description *string `json:"description,omitempty"`
		}
		if err := unmarshalJSONColumn("api_keys.metadata_json", ak.MetadataJSON, &metadata); err != nil {
			return err
		}
		ak.Description = metadata.Description
	}
	return nil
}
