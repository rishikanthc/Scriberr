package models

import "time"

type SystemSetting struct {
	Key       string    `json:"key" gorm:"primaryKey;type:varchar(100);not null"`
	ValueJSON string    `json:"-" gorm:"column:value_json;type:json;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (SystemSetting) TableName() string { return "system_settings" }
