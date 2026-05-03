package api

import (
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/models"
	profiledomain "scriberr/internal/profile"
	"scriberr/internal/transcription/engineprovider"

	"github.com/gin-gonic/gin"
	speechmodels "scriberr-engine/speech/models"
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
	if options.Language != nil && strings.TrimSpace(*options.Language) != "" && !validLanguage(strings.TrimSpace(*options.Language)) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "language is invalid", stringPtr("options.language"))
		return false
	}
	if strings.TrimSpace(options.Model) != "" {
		if _, ok := speechmodels.DefaultModelRegistry().Resolve(strings.TrimSpace(options.Model)); !ok {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "model is invalid", stringPtr("options.model"))
			return false
		}
	}
	if task := strings.TrimSpace(options.Task); task != "" && task != "transcribe" && task != "translate" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "task is invalid", stringPtr("options.task"))
		return false
	}
	if method := strings.TrimSpace(options.DecodingMethod); method != "" && method != "greedy_search" && method != "modified_beam_search" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "decoding method is invalid", stringPtr("options.decoding_method"))
		return false
	}
	if chunking := strings.ToLower(strings.TrimSpace(options.ChunkingStrategy)); chunking != "" && chunking != "fixed" && chunking != "vad" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "chunking strategy is invalid", stringPtr("options.chunking_strategy"))
		return false
	}
	if options.Threads < 0 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "threads must be zero or greater", stringPtr("options.threads"))
		return false
	}
	if options.TailPaddings != nil && (*options.TailPaddings < -1 || *options.TailPaddings > 16) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "tail paddings is invalid", stringPtr("options.tail_paddings"))
		return false
	}
	if options.NumSpeakers < 0 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "number of speakers must be zero or greater", stringPtr("options.num_speakers"))
		return false
	}
	if options.DiarizationThreshold < 0 || options.DiarizationThreshold > 1 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "diarization threshold is invalid", stringPtr("options.diarization_threshold"))
		return false
	}
	if options.MinDurationOn < 0 || options.MinDurationOn > 2 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "minimum speech duration is invalid", stringPtr("options.min_duration_on"))
		return false
	}
	if options.MinDurationOff < 0 || options.MinDurationOff > 2 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "minimum silence duration is invalid", stringPtr("options.min_duration_off"))
		return false
	}
	return true
}
func profileParams(options profileOptionsRequest) models.WhisperXParams {
	params := supportedProfileParams(models.WhisperXParams{
		Model:                options.Model,
		Language:             options.Language,
		Task:                 options.Task,
		Threads:              options.Threads,
		TailPaddings:         options.TailPaddings,
		CanarySourceLanguage: options.CanarySourceLanguage,
		CanaryTargetLanguage: options.CanaryTargetLanguage,
		CanaryUsePunctuation: options.CanaryUsePunctuation,
		DecodingMethod:       options.DecodingMethod,
		ChunkingStrategy:     options.ChunkingStrategy,
		Diarize:              options.Diarize,
		DiarizeModel:         options.DiarizeModel,
		NumSpeakers:          options.NumSpeakers,
		DiarizationThreshold: options.DiarizationThreshold,
		MinDurationOn:        options.MinDurationOn,
		MinDurationOff:       options.MinDurationOff,
	})
	if options.Diarization != nil {
		params.Diarize = *options.Diarization
	}
	return params
}

func supportedProfileParams(input models.WhisperXParams) models.WhisperXParams {
	model := strings.TrimSpace(input.Model)
	if spec, ok := speechmodels.DefaultModelRegistry().ResolveOrDefault(model, speechmodels.ModelDefaultTranscription); ok {
		model = string(spec.ID)
	} else {
		model = engineprovider.DefaultTranscriptionModel
	}
	task := strings.TrimSpace(input.Task)
	if task == "" {
		task = "transcribe"
	}
	decodingMethod := strings.TrimSpace(input.DecodingMethod)
	if decodingMethod == "" {
		decodingMethod = "greedy_search"
	}
	if familyForModel(model) == "whisper" {
		decodingMethod = "greedy_search"
	}
	chunkingStrategy := strings.ToLower(strings.TrimSpace(input.ChunkingStrategy))
	if chunkingStrategy == "" {
		chunkingStrategy = "fixed"
	}
	var language *string
	if input.Language != nil {
		trimmed := strings.TrimSpace(*input.Language)
		if trimmed != "" && trimmed != "auto" {
			language = &trimmed
		}
	}
	diarizationThreshold := input.DiarizationThreshold
	if diarizationThreshold == 0 {
		diarizationThreshold = 0.5
	}
	minDurationOn := input.MinDurationOn
	if minDurationOn == 0 {
		minDurationOn = 0.2
	}
	minDurationOff := input.MinDurationOff
	if minDurationOff == 0 {
		minDurationOff = 0.3
	}
	return models.WhisperXParams{
		ModelFamily:             familyForModel(model),
		Model:                   model,
		Language:                language,
		Task:                    task,
		Threads:                 input.Threads,
		TailPaddings:            input.TailPaddings,
		EnableTokenTimestamps:   boolPtr(true),
		EnableSegmentTimestamps: boolPtr(true),
		CanarySourceLanguage:    strings.TrimSpace(input.CanarySourceLanguage),
		CanaryTargetLanguage:    strings.TrimSpace(input.CanaryTargetLanguage),
		CanaryUsePunctuation:    input.CanaryUsePunctuation,
		DecodingMethod:          decodingMethod,
		ChunkingStrategy:        chunkingStrategy,
		Diarize:                 input.Diarize,
		DiarizeModel:            engineprovider.DefaultDiarizationModel,
		NumSpeakers:             input.NumSpeakers,
		DiarizationThreshold:    diarizationThreshold,
		MinDurationOn:           minDurationOn,
		MinDurationOff:          minDurationOff,
	}
}

func familyForModel(modelID string) string {
	spec, ok := speechmodels.DefaultModelRegistry().Resolve(modelID)
	if !ok {
		return "transcription"
	}
	switch spec.Family {
	case speechmodels.FamilyWhisper:
		return "whisper"
	case speechmodels.FamilyNemo:
		return "nemo_transducer"
	case speechmodels.FamilyCanary:
		return "canary"
	default:
		return string(spec.Family)
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
