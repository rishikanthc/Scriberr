package models

import "fmt"

func requireUserID(model string, userID uint) error {
	if userID == 0 {
		return fmt.Errorf("%s user_id is required", model)
	}
	return nil
}

func requireUserIDForIdentifiedSave(model string, userID uint, hasID bool) error {
	if !hasID {
		return nil
	}
	return requireUserID(model, userID)
}
