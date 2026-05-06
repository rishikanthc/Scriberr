package api

import (
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/models"
	profiledomain "scriberr/internal/profile"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listProfiles(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	profiles, err := h.profiles.List(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list profiles", nil)
		return
	}
	items := make([]ProfileResponse, 0, len(profiles))
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
	profile := &models.TranscriptionProfile{
		ID:          randomHex(16),
		UserID:      userID,
		Name:        strings.TrimSpace(req.Name),
		Description: &description,
		IsDefault:   req.IsDefault,
		Parameters:  profileParams(req.Options),
	}
	if err := h.profiles.Create(c.Request.Context(), profile); err != nil {
		if errors.Is(err, profiledomain.ErrInvalidPipeline) {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "pipeline is invalid", stringPtr("options.pipeline"))
			return
		}
		if errors.Is(err, profiledomain.ErrInvalidModel) {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "model is invalid", stringPtr("options.pipeline"))
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create profile", nil)
		return
	}
	response := profileResponse(profile)
	h.publishEventForUser("profile.updated", gin.H{"id": response.ID}, userID)
	c.JSON(http.StatusCreated, response)
}
func (h *Handler) getProfile(c *gin.Context) {
	profile, ok := h.profileByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	c.JSON(http.StatusOK, profileResponse(profile))
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
	if err := h.profiles.Update(c.Request.Context(), profile, req.IsDefault != nil); err != nil {
		if errors.Is(err, profiledomain.ErrInvalidPipeline) {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "pipeline is invalid", stringPtr("options.pipeline"))
			return
		}
		if errors.Is(err, profiledomain.ErrInvalidModel) {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "model is invalid", stringPtr("options.pipeline"))
			return
		}
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
	if err := h.profiles.Delete(c.Request.Context(), profile.UserID, profile.ID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete profile", nil)
		return
	}
	h.publishEventForUser("profile.updated", gin.H{"id": publicIDForProfile(profile.ID), "deleted": true}, profile.UserID)
	c.Status(http.StatusNoContent)
}
func (h *Handler) setDefaultProfile(c *gin.Context, publicID string) {
	profile, ok := h.profileByPublicID(c, publicID)
	if !ok {
		return
	}
	if err := h.profiles.SetDefault(c.Request.Context(), profile.UserID, profile.ID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not set default profile", nil)
		return
	}
	response := gin.H{"id": publicIDForProfile(profile.ID), "is_default": true}
	h.publishEventForUser("profile.updated", response, profile.UserID)
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
	if len(options.Pipeline) == 0 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "pipeline is required", stringPtr("options.pipeline"))
		return false
	}
	if field, ok := legacyProfileOptionField(options); ok {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "profile option must be configured on the owning pipeline step", stringPtr(field))
		return false
	}
	return true
}
func profileParams(options profileOptionsRequest) models.ASRParams {
	return models.ASRParams{Pipeline: options.Pipeline}
}

func legacyProfileOptionField(options profileOptionsRequest) (string, bool) {
	switch {
	case options.Language != nil:
		return "options.language", true
	case strings.TrimSpace(options.Task) != "":
		return "options.task", true
	case options.Threads != nil:
		return "options.threads", true
	case options.TailPaddings != nil:
		return "options.tail_paddings", true
	case strings.TrimSpace(options.DecodingMethod) != "":
		return "options.decoding_method", true
	case strings.TrimSpace(options.ChunkingStrategy) != "":
		return "options.chunking_strategy", true
	case options.NumSpeakers != nil:
		return "options.num_speakers", true
	case options.DiarizationThreshold != nil:
		return "options.diarization_threshold", true
	case options.MinDurationOn != nil:
		return "options.min_duration_on", true
	case options.MinDurationOff != nil:
		return "options.min_duration_off", true
	default:
		return "", false
	}
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
	profile, err := h.profiles.Get(c.Request.Context(), userID, id)
	if errors.Is(err, profiledomain.ErrNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "profile not found", nil)
		return nil, false
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load profile", nil)
		return nil, false
	}
	return profile, true
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
