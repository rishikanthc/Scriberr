package api

import (
	"net/http"
	"strings"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) listProfiles(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var profiles []models.TranscriptionProfile
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&profiles).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list profiles", nil)
		return
	}
	items := make([]gin.H, 0, len(profiles))
	for i := range profiles {
		items = append(items, profileResponse(&profiles[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}
func (h *Handler) createProfile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req createProfileRequest
	if !bindJSON(c, &req) {
		return
	}
	if !validateProfileInput(c, req.Name, req.Options) {
		return
	}
	description := strings.TrimSpace(req.Description)
	profile := models.TranscriptionProfile{
		ID:          randomHex(16),
		UserID:      userID,
		Name:        strings.TrimSpace(req.Name),
		Description: &description,
		IsDefault:   req.IsDefault,
		Parameters:  profileParams(req.Options),
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}
		if profile.IsDefault {
			return saveUserDefaultProfile(tx, userID, &profile.ID)
		}
		return nil
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create profile", nil)
		return
	}
	response := profileResponse(&profile)
	h.publishEvent("profile.updated", gin.H{"id": response["id"]})
	c.JSON(http.StatusCreated, response)
}
func (h *Handler) getProfile(c *gin.Context) {
	profile, ok := h.profileByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	response := profileResponse(profile)
	h.publishEvent("profile.updated", gin.H{"id": response["id"]})
	c.JSON(http.StatusOK, response)
}
func (h *Handler) updateProfile(c *gin.Context) {
	profile, ok := h.profileByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	var req updateProfileRequest
	if !bindJSON(c, &req) {
		return
	}
	if !validateProfileInput(c, req.Name, req.Options) {
		return
	}
	description := strings.TrimSpace(req.Description)
	profile.Name = strings.TrimSpace(req.Name)
	profile.Description = &description
	profile.Parameters = profileParams(req.Options)
	if req.IsDefault != nil {
		profile.IsDefault = *req.IsDefault
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(profile).Error; err != nil {
			return err
		}
		if req.IsDefault != nil {
			if *req.IsDefault {
				return saveUserDefaultProfile(tx, profile.UserID, &profile.ID)
			}
			return clearUserDefaultProfileIfMatches(tx, profile.UserID, profile.ID)
		}
		return nil
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update profile", nil)
		return
	}
	c.JSON(http.StatusOK, profileResponse(profile))
}
func (h *Handler) deleteProfile(c *gin.Context) {
	profile, ok := h.profileByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	if err := database.DB.Delete(&models.TranscriptionProfile{}, "id = ? AND user_id = ?", profile.ID, profile.UserID).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete profile", nil)
		return
	}
	var user models.User
	if err := database.DB.First(&user, profile.UserID).Error; err == nil && user.DefaultProfileID != nil && *user.DefaultProfileID == profile.ID {
		user.DefaultProfileID = nil
		_ = database.DB.Save(&user).Error
	}
	h.publishEvent("profile.updated", gin.H{"id": publicIDForProfile(profile.ID), "deleted": true})
	c.Status(http.StatusNoContent)
}
func (h *Handler) setDefaultProfile(c *gin.Context, publicID string) {
	profile, ok := h.profileByPublicID(c, publicID)
	if !ok {
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TranscriptionProfile{}).
			Where("user_id = ? AND id <> ?", profile.UserID, profile.ID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TranscriptionProfile{}).
			Where("id = ? AND user_id = ?", profile.ID, profile.UserID).
			Update("is_default", true).Error; err != nil {
			return err
		}
		return saveUserDefaultProfile(tx, profile.UserID, &profile.ID)
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not set default profile", nil)
		return
	}
	response := gin.H{"id": publicIDForProfile(profile.ID), "is_default": true}
	h.publishEvent("profile.updated", response)
	c.JSON(http.StatusOK, response)
}
func (h *Handler) profileCommand(c *gin.Context) {
	if strings.HasSuffix(c.Param("idAction"), ":set-default") {
		h.setDefaultProfile(c, strings.TrimSuffix(c.Param("idAction"), ":set-default"))
		return
	}
	writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
}
func validateProfileInput(c *gin.Context, name string, options profileOptionsRequest) bool {
	if strings.TrimSpace(name) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", stringPtr("name"))
		return false
	}
	if options.Language != "" && !validLanguage(options.Language) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "language is invalid", stringPtr("options.language"))
		return false
	}
	return true
}
func profileParams(options profileOptionsRequest) models.WhisperXParams {
	params := models.WhisperXParams{
		Model:   strings.TrimSpace(options.Model),
		Device:  strings.TrimSpace(options.Device),
		Diarize: options.Diarization,
	}
	if params.Device == "" {
		params.Device = "auto"
	}
	if options.Language != "" {
		language := strings.TrimSpace(options.Language)
		params.Language = &language
	}
	return params
}
func (h *Handler) profileByPublicID(c *gin.Context, publicID string) (*models.TranscriptionProfile, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return nil, false
	}
	id, ok := parseProfileID(publicID)
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "profile not found", nil)
		return nil, false
	}
	var profile models.TranscriptionProfile
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&profile).Error; err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "profile not found", nil)
		return nil, false
	}
	return &profile, true
}
func parseProfileID(publicID string) (string, bool) {
	id := strings.TrimPrefix(publicID, "profile_")
	if id == publicID || id == "" {
		return "", false
	}
	return id, true
}
func publicIDForProfile(id string) string {
	return "profile_" + id
}
func profileExistsForUser(userID uint, profileID string) bool {
	var count int64
	return database.DB.Model(&models.TranscriptionProfile{}).
		Where("id = ? AND user_id = ?", profileID, userID).
		Count(&count).Error == nil && count == 1
}
func saveUserDefaultProfile(tx *gorm.DB, userID uint, profileID *string) error {
	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		return err
	}
	user.DefaultProfileID = profileID
	return tx.Save(&user).Error
}
func clearUserDefaultProfileIfMatches(tx *gorm.DB, userID uint, profileID string) error {
	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		return err
	}
	if user.DefaultProfileID == nil || *user.DefaultProfileID != profileID {
		return nil
	}
	user.DefaultProfileID = nil
	return tx.Save(&user).Error
}
