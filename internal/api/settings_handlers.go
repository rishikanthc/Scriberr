package api

import (
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/account"

	"github.com/gin-gonic/gin"
)

func (h *Handler) getSettings(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	c.JSON(http.StatusOK, settingsResponse(h, user))
}
func (h *Handler) updateSettings(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req updateSettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	update := account.SettingsUpdate{}
	if req.DefaultProfileID != nil {
		update.DefaultProfileIDSet = true
		rawProfileID := strings.TrimSpace(*req.DefaultProfileID)
		if rawProfileID == "" {
			update.DefaultProfileID = nil
		} else {
			parsedID, ok := parseProfileID(rawProfileID)
			if !ok {
				writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "default_profile_id is invalid", stringPtr("default_profile_id"))
				return
			}
			update.DefaultProfileID = &parsedID
		}
	}
	update.AutoTranscriptionEnabled = req.AutoTranscriptionEnabled
	update.AutoRenameEnabled = req.AutoRenameEnabled
	updated, err := h.account.UpdateSettings(c.Request.Context(), user.ID, update)
	if errors.Is(err, account.ErrInvalidDefaultProfile) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "default_profile_id is invalid", stringPtr("default_profile_id"))
		return
	}
	if errors.Is(err, account.ErrDefaultProfileRequired) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "default_profile_id is required before enabling auto transcription", stringPtr("default_profile_id"))
		return
	}
	if errors.Is(err, account.ErrSmallLLMRequired) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "small LLM model is required before enabling auto rename", stringPtr("auto_rename_enabled"))
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update settings", nil)
		return
	}
	response := settingsResponse(h, updated)
	h.publishEventForUser("settings.updated", gin.H{
		"auto_transcription_enabled": response.AutoTranscriptionEnabled,
		"auto_rename_enabled":        response.AutoRenameEnabled,
		"default_profile_id":         response.DefaultProfileID,
	}, user.ID)
	c.JSON(http.StatusOK, response)
}
