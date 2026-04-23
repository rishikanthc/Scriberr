package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

func marshalJSONColumn(column string, value any) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", column, err)
	}
	return string(bytes), nil
}

func unmarshalJSONColumn(column string, raw string, dest any) error {
	if raw == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("unmarshal %s: %w", column, err)
	}
	return nil
}

func clearOtherDefaultsForUser(tx *gorm.DB, model any, userID uint, currentID any) error {
	if userID == 0 {
		return fmt.Errorf("clear defaults: missing user_id")
	}
	return tx.Model(model).
		Where("user_id = ? AND id <> ?", userID, currentID).
		Update("is_default", false).Error
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
