package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) createTranscription(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req createTranscriptionRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.FileID == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "file_id is required", stringPtr("file_id"))
		return
	}
	if req.Options.Language != "" && !validLanguage(req.Options.Language) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "language is invalid", stringPtr("options.language"))
		return
	}
	if !validateTranscriptionProfile(c, userID, req.ProfileID) {
		return
	}
	sourceID := strings.TrimPrefix(req.FileID, "file_")
	var source models.TranscriptionJob
	if sourceID == req.FileID || database.DB.Where("id = ? AND user_id = ? AND source_file_hash IS NULL", sourceID, userID).First(&source).Error != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" && source.Title != nil {
		title = *source.Title
	}
	sourceFileID := source.ID
	job := models.TranscriptionJob{
		ID:             randomHex(16),
		UserID:         userID,
		Title:          &title,
		Status:         models.StatusPending,
		AudioPath:      source.AudioPath,
		SourceFileName: source.SourceFileName,
		SourceFileHash: &sourceFileID,
		Language:       nil,
		Diarization:    req.Options.Diarization,
	}
	if req.Options.Language != "" {
		job.Language = &req.Options.Language
		job.Parameters.Language = &req.Options.Language
	}
	if err := database.DB.Create(&job).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create transcription", nil)
		return
	}
	response := transcriptionResponse(&job)
	h.publishTranscriptionEvent("transcription.created", response["id"].(string), gin.H{"id": response["id"], "status": response["status"]})
	h.publishEvent("transcription.created", gin.H{"id": response["id"], "status": response["status"]})
	c.JSON(http.StatusAccepted, response)
}
func (h *Handler) submitTranscription(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var options struct {
		Language    string `json:"language"`
		Diarization bool   `json:"diarization"`
	}
	if rawOptions := strings.TrimSpace(c.PostForm("options")); rawOptions != "" {
		if err := json.Unmarshal([]byte(rawOptions), &options); err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "options must be valid JSON", stringPtr("options"))
			return
		}
	}
	if options.Language != "" && !validLanguage(options.Language) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "language is invalid", stringPtr("options.language"))
		return
	}
	if !validateTranscriptionProfile(c, userID, c.PostForm("profile_id")) {
		return
	}

	source, _, _, ok := h.storeUploadedFile(c, userID)
	if !ok {
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" && source.Title != nil {
		title = *source.Title
	}
	sourceFileID := source.ID
	job := models.TranscriptionJob{
		ID:             randomHex(16),
		UserID:         userID,
		Title:          &title,
		Status:         models.StatusPending,
		AudioPath:      source.AudioPath,
		SourceFileName: source.SourceFileName,
		SourceFileHash: &sourceFileID,
		Diarization:    options.Diarization,
	}
	if options.Language != "" {
		job.Language = &options.Language
		job.Parameters.Language = &options.Language
	}
	if err := database.DB.Create(&job).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create transcription", nil)
		return
	}
	response := gin.H{
		"id":      "tr_" + job.ID,
		"file_id": "file_" + source.ID,
		"status":  string(job.Status),
	}
	h.publishTranscriptionEvent("transcription.created", response["id"].(string), gin.H{"id": response["id"], "status": response["status"]})
	h.publishEvent("transcription.created", gin.H{"id": response["id"], "status": response["status"]})
	c.JSON(http.StatusAccepted, response)
}

func validateTranscriptionProfile(c *gin.Context, userID uint, publicProfileID string) bool {
	profileID := strings.TrimSpace(publicProfileID)
	if profileID == "" {
		return true
	}
	parsedID, ok := parseProfileID(profileID)
	if !ok || !profileExistsForUser(userID, parsedID) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "profile_id is invalid", stringPtr("profile_id"))
		return false
	}
	return true
}

func (h *Handler) listTranscriptions(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	opts, ok := parseListQuery(c, allowedResourceSorts())
	if !ok {
		return
	}
	query := database.DB.Where("user_id = ? AND source_file_hash IS NOT NULL", userID)
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if !validTranscriptionStatus(status) {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "status is invalid", stringPtr("status"))
			return
		}
		query = query.Where("status = ?", status)
	}
	var jobs []models.TranscriptionJob
	if err := applyListQuery(query, opts).Find(&jobs).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list transcriptions", nil)
		return
	}
	jobs, nextCursor := trimListPage(jobs, opts)
	items := make([]gin.H, 0, len(jobs))
	for i := range jobs {
		items = append(items, transcriptionListResponse(&jobs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nextCursor})
}
func (h *Handler) getTranscription(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	c.JSON(http.StatusOK, transcriptionResponse(job))
}
func (h *Handler) updateTranscription(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	var req updateTranscriptionRequest
	if !bindJSON(c, &req) {
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "title is required", stringPtr("title"))
		return
	}
	if err := database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("title", title).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update transcription", nil)
		return
	}
	job.Title = &title
	response := transcriptionResponse(job)
	h.publishTranscriptionEvent("transcription.updated", response["id"].(string), gin.H{"id": response["id"], "status": response["status"]})
	h.publishEvent("transcription.updated", gin.H{"id": response["id"], "status": response["status"]})
	c.JSON(http.StatusOK, response)
}
func (h *Handler) deleteTranscription(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	if err := database.DB.Delete(&models.TranscriptionJob{}, "id = ?", job.ID).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete transcription", nil)
		return
	}
	h.publishTranscriptionEvent("transcription.deleted", "tr_"+job.ID, gin.H{"id": "tr_" + job.ID})
	h.publishEvent("transcription.deleted", gin.H{"id": "tr_" + job.ID})
	c.Status(http.StatusNoContent)
}
func (h *Handler) cancelTranscription(c *gin.Context, publicID string) {
	job, ok := h.transcriptionByPublicID(c, publicID)
	if !ok {
		return
	}
	if job.Status == models.StatusCompleted || job.Status == models.StatusFailed || job.Status == models.StatusCanceled {
		writeError(c, http.StatusConflict, "CONFLICT", "transcription cannot be canceled", nil)
		return
	}
	if err := database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("status", models.StatusCanceled).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not cancel transcription", nil)
		return
	}
	response := gin.H{"id": "tr_" + job.ID, "status": string(models.StatusCanceled)}
	h.publishTranscriptionEvent("transcription.canceled", response["id"].(string), response)
	h.publishEvent("transcription.canceled", response)
	c.JSON(http.StatusOK, response)
}
func (h *Handler) retryTranscription(c *gin.Context, publicID string) {
	job, ok := h.transcriptionByPublicID(c, publicID)
	if !ok {
		return
	}
	sourceFileID := ""
	if job.SourceFileHash != nil {
		sourceFileID = *job.SourceFileHash
	}
	retry := models.TranscriptionJob{
		ID:             randomHex(16),
		UserID:         job.UserID,
		Title:          job.Title,
		Status:         models.StatusPending,
		AudioPath:      job.AudioPath,
		SourceFileName: job.SourceFileName,
		SourceFileHash: &sourceFileID,
		Language:       job.Language,
		Diarization:    job.Diarization,
		Parameters:     job.Parameters,
	}
	if err := database.DB.Create(&retry).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not retry transcription", nil)
		return
	}
	response := gin.H{
		"id":                      "tr_" + retry.ID,
		"source_transcription_id": "tr_" + job.ID,
		"status":                  string(retry.Status),
	}
	h.publishTranscriptionEvent("transcription.created", response["id"].(string), gin.H{"id": response["id"], "status": response["status"]})
	h.publishEvent("transcription.created", gin.H{"id": response["id"], "status": response["status"]})
	c.JSON(http.StatusAccepted, response)
}
func (h *Handler) getTranscript(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	text := ""
	if job.Transcript != nil {
		text = *job.Transcript
	}
	c.JSON(http.StatusOK, gin.H{
		"transcription_id": "tr_" + job.ID,
		"text":             text,
		"segments":         []any{},
		"words":            []any{},
	})
}
func (h *Handler) streamTranscriptionAudio(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	file, err := os.Open(job.AudioPath)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription audio not found", nil)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription audio not found", nil)
		return
	}
	mimeType := mediaType("", job.SourceFileName)
	c.Header("Content-Type", mimeType)
	c.Header("Accept-Ranges", "bytes")
	http.ServeContent(c.Writer, c.Request, job.SourceFileName, stat.ModTime(), file)
}
func (h *Handler) transcriptionByPublicID(c *gin.Context, publicID string) (*models.TranscriptionJob, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return nil, false
	}
	id := strings.TrimPrefix(publicID, "tr_")
	if id == publicID || id == "" {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription not found", nil)
		return nil, false
	}
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ? AND user_id = ? AND source_file_hash IS NOT NULL", id, userID).First(&job).Error; err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription not found", nil)
		return nil, false
	}
	return &job, true
}
func (h *Handler) transcriptionCommand(c *gin.Context) {
	action := c.Param("idAction")
	switch {
	case strings.HasSuffix(action, ":cancel"):
		h.cancelTranscription(c, strings.TrimSuffix(action, ":cancel"))
	case strings.HasSuffix(action, ":retry"):
		h.retryTranscription(c, strings.TrimSuffix(action, ":retry"))
	default:
		writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
	}
}
