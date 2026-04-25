package api

import (
	"net/http"
	"strings"

	"scriberr/internal/database"

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
	defaultProfileID := user.DefaultProfileID
	if req.DefaultProfileID != nil {
		rawProfileID := strings.TrimSpace(*req.DefaultProfileID)
		if rawProfileID == "" {
			defaultProfileID = nil
		} else {
			parsedID, ok := parseProfileID(rawProfileID)
			if !ok || !profileExistsForUser(user.ID, parsedID) {
				writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "default_profile_id is invalid", stringPtr("default_profile_id"))
				return
			}
			defaultProfileID = &parsedID
		}
	}
	autoTranscription := user.AutoTranscriptionEnabled
	if req.AutoTranscriptionEnabled != nil {
		autoTranscription = *req.AutoTranscriptionEnabled
	}
	user.DefaultProfileID = defaultProfileID
	user.AutoTranscriptionEnabled = autoTranscription
	if err := database.DB.Save(user).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update settings", nil)
		return
	}
	response := settingsResponse(h, user)
	h.publishEvent("settings.updated", gin.H{"auto_transcription_enabled": response["auto_transcription_enabled"], "default_profile_id": response["default_profile_id"]})
	c.JSON(http.StatusOK, response)
}
