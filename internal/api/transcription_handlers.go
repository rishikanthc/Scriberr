package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/tags"
	transcriptiondomain "scriberr/internal/transcription"
	"scriberr/internal/transcription/orchestrator"
	"scriberr/internal/transcription/worker"

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
	profileID, ok := parseOptionalTranscriptionProfileID(c, req.ProfileID)
	if !ok {
		return
	}
	sourceID := strings.TrimPrefix(req.FileID, "file_")
	if sourceID == req.FileID || sourceID == "" {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return
	}
	job, err := h.transcriptions.Create(c.Request.Context(), transcriptiondomain.CreateCommand{
		UserID:      userID,
		FileID:      sourceID,
		Title:       req.Title,
		ProfileID:   profileID,
		Language:    req.Options.Language,
		Diarization: req.Options.Diarization,
	})
	if !h.writeTranscriptionServiceError(c, err, "could not create transcription") {
		return
	}
	response := transcriptionResponse(job)
	h.publishTranscriptionEvent("transcription.created", response["id"].(string), gin.H{"id": response["id"], "file_id": response["file_id"], "status": response["status"]})
	h.publishEvent("transcription.created", gin.H{"id": response["id"], "file_id": response["file_id"], "status": response["status"]})
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
		Diarization *bool  `json:"diarization"`
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
	profileID, ok := parseOptionalTranscriptionProfileID(c, c.PostForm("profile_id"))
	if !ok {
		return
	}

	upload, ok := h.storeUploadedFile(c, userID)
	if !ok {
		return
	}
	source := upload.Job

	job, err := h.transcriptions.Submit(c.Request.Context(), transcriptiondomain.SubmitCommand{
		UserID:      userID,
		File:        source,
		Title:       c.PostForm("title"),
		ProfileID:   profileID,
		Language:    options.Language,
		Diarization: options.Diarization,
	})
	if !h.writeTranscriptionServiceError(c, err, "could not create transcription") {
		return
	}
	response := gin.H{
		"id":      "tr_" + job.ID,
		"file_id": fileIDForTranscription(job),
		"status":  string(job.Status),
	}
	h.publishTranscriptionEvent("transcription.created", response["id"].(string), response)
	h.publishEvent("transcription.created", response)
	c.JSON(http.StatusAccepted, response)
}

func parseOptionalTranscriptionProfileID(c *gin.Context, publicProfileID string) (string, bool) {
	profileID := strings.TrimSpace(publicProfileID)
	if profileID == "" {
		return "", true
	}
	parsedID, ok := parseProfileID(profileID)
	if !ok {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "profile_id is invalid", stringPtr("profile_id"))
		return "", false
	}
	return parsedID, true
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
	status := strings.TrimSpace(c.Query("status"))
	if status != "" {
		if !validTranscriptionStatus(status) {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "status is invalid", stringPtr("status"))
			return
		}
	}
	listOptions := transcriptionListOptions(status, opts)
	if tagRefs := transcriptionTagFilters(c); len(tagRefs) > 0 {
		if h.tags == nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
			return
		}
		matchAll, ok := parseTagMatch(c)
		if !ok {
			return
		}
		ids, err := h.tags.TranscriptionIDsByTags(c.Request.Context(), tags.FilterRequest{
			UserID:   userID,
			TagRefs:  tagRefs,
			MatchAll: matchAll,
		})
		if err != nil {
			writeTagServiceError(c, err)
			return
		}
		if len(ids) == 0 {
			listOptions.ForceEmpty = true
		} else {
			listOptions.IDs = ids
		}
	}
	jobs, err := h.transcriptions.List(c.Request.Context(), userID, listOptions)
	if err != nil {
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
	userID, id, ok := h.transcriptionRequestIdentity(c, c.Param("id"))
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
	job, err := h.transcriptions.UpdateTitle(c.Request.Context(), userID, id, title)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update transcription", nil)
		return
	}
	response := transcriptionResponse(job)
	h.publishTranscriptionEvent("transcription.updated", response["id"].(string), gin.H{"id": response["id"], "status": response["status"]})
	h.publishEvent("transcription.updated", gin.H{"id": response["id"], "status": response["status"]})
	c.JSON(http.StatusOK, response)
}
func (h *Handler) deleteTranscription(c *gin.Context) {
	userID, id, ok := h.transcriptionRequestIdentity(c, c.Param("id"))
	if !ok {
		return
	}
	if err := h.transcriptions.Delete(c.Request.Context(), userID, id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete transcription", nil)
		return
	}
	h.publishTranscriptionEvent("transcription.deleted", "tr_"+id, gin.H{"id": "tr_" + id})
	h.publishEvent("transcription.deleted", gin.H{"id": "tr_" + id})
	c.Status(http.StatusNoContent)
}
func (h *Handler) cancelTranscription(c *gin.Context, publicID string) {
	userID, id, ok := h.transcriptionRequestIdentity(c, publicID)
	if !ok {
		return
	}
	job, err := h.transcriptions.Cancel(c.Request.Context(), userID, id)
	if errors.Is(err, transcriptiondomain.ErrStateConflict) || errors.Is(err, worker.ErrStateConflict) {
		writeError(c, http.StatusConflict, "CONFLICT", "transcription cannot be stopped", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not stop transcription", nil)
		return
	}
	response := gin.H{"id": "tr_" + job.ID, "file_id": fileIDForTranscription(job), "status": string(models.StatusStopped), "stage": "stopped"}
	h.publishTranscriptionEvent("transcription.stopped", response["id"].(string), response)
	h.publishEvent("transcription.stopped", response)
	c.JSON(http.StatusOK, response)
}
func (h *Handler) retryTranscription(c *gin.Context, publicID string) {
	userID, id, ok := h.transcriptionRequestIdentity(c, publicID)
	if !ok {
		return
	}
	retry, err := h.transcriptions.Retry(c.Request.Context(), userID, id)
	if !h.writeTranscriptionServiceError(c, err, "could not retry transcription") {
		return
	}
	response := gin.H{
		"id":                      "tr_" + retry.ID,
		"file_id":                 fileIDForTranscription(retry),
		"source_transcription_id": "tr_" + id,
		"status":                  string(retry.Status),
	}
	h.publishTranscriptionEvent("transcription.created", response["id"].(string), response)
	h.publishEvent("transcription.created", response)
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
	transcript, err := orchestrator.ParseStoredTranscript(text)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read transcript", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"transcription_id": "tr_" + job.ID,
		"text":             transcript.Text,
		"segments":         transcript.Segments,
		"words":            transcript.Words,
	})
}
func (h *Handler) streamTranscriptionAudio(c *gin.Context) {
	userID, id, ok := h.transcriptionRequestIdentity(c, c.Param("id"))
	if !ok {
		return
	}
	file, job, err := h.transcriptions.OpenAudio(c.Request.Context(), userID, id)
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

func (h *Handler) writeEnqueueError(c *gin.Context, enqueueErr error) {
	if errors.Is(enqueueErr, worker.ErrQueueStopped) {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "transcription queue is unavailable", nil)
		return
	}
	writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not enqueue transcription", nil)
}

func (h *Handler) writeTranscriptionServiceError(c *gin.Context, err error, fallbackMessage string) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, transcriptiondomain.ErrFileNotFound), errors.Is(err, transcriptiondomain.ErrNotFound):
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
	case errors.Is(err, transcriptiondomain.ErrInvalidProfile):
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "profile_id is invalid", stringPtr("profile_id"))
	case errors.Is(err, worker.ErrQueueStopped):
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "transcription queue is unavailable", nil)
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", fallbackMessage, nil)
	}
	return false
}

func (h *Handler) transcriptionByPublicID(c *gin.Context, publicID string) (*models.TranscriptionJob, bool) {
	userID, id, ok := h.transcriptionRequestIdentity(c, publicID)
	if !ok {
		return nil, false
	}
	job, err := h.transcriptions.Get(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription not found", nil)
		return nil, false
	}
	return job, true
}

func (h *Handler) transcriptionRequestIdentity(c *gin.Context, publicID string) (uint, string, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return 0, "", false
	}
	id := strings.TrimPrefix(publicID, "tr_")
	if id == publicID || id == "" {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription not found", nil)
		return 0, "", false
	}
	if h.transcriptions == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "transcription service is not configured", nil)
		return 0, "", false
	}
	return userID, id, true
}

func transcriptionListOptions(status string, opts *listQuery) transcriptiondomain.ListOptions {
	var cursor *transcriptiondomain.ListCursor
	if opts.Cursor != nil {
		cursor = &transcriptiondomain.ListCursor{Value: opts.Cursor.Value, ID: opts.Cursor.ID}
	}
	return transcriptiondomain.ListOptions{
		Status:       status,
		Query:        opts.Query,
		UpdatedAfter: opts.UpdatedAfter,
		Limit:        opts.Limit,
		SortColumn:   opts.SortColumn,
		SortDesc:     opts.SortDesc,
		Cursor:       cursor,
	}
}
func (h *Handler) transcriptionCommand(c *gin.Context) {
	action := c.Param("idAction")
	switch {
	case strings.HasSuffix(action, ":stop"):
		h.cancelTranscription(c, strings.TrimSuffix(action, ":stop"))
	case strings.HasSuffix(action, ":cancel"):
		h.cancelTranscription(c, strings.TrimSuffix(action, ":cancel"))
	case strings.HasSuffix(action, ":retry"):
		h.retryTranscription(c, strings.TrimSuffix(action, ":retry"))
	default:
		writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
	}
}
