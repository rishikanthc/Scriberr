package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"scriberr/internal/models"
	recordingdomain "scriberr/internal/recording"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const statusClientClosedRequest = 499

func (h *Handler) createRecording(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.recordings == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording service is not configured", nil)
		return
	}
	var req createRecordingRequest
	if !bindJSON(c, &req) {
		return
	}
	profileID, ok := h.validateRecordingProfile(c, userID, req.ProfileID)
	if !ok {
		return
	}
	optionsJSON, ok := recordingOptionsJSON(c, req.Options.Language, req.Options.Diarization)
	if !ok {
		return
	}
	session, err := h.recordings.CreateSession(c.Request.Context(), recordingdomain.CreateSessionRequest{
		UserID:                   userID,
		Title:                    req.Title,
		SourceKind:               req.SourceKind,
		MimeType:                 req.MimeType,
		Codec:                    req.Codec,
		ChunkDurationMs:          req.ChunkDurationMs,
		AutoTranscribe:           req.AutoTranscribe,
		ProfileID:                profileID,
		TranscriptionOptionsJSON: optionsJSON,
	})
	if err != nil {
		writeRecordingServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, recordingResponse(session))
}

func (h *Handler) listRecordings(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.recordings == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording service is not configured", nil)
		return
	}
	opts, ok := parseListQuery(c, allowedResourceSorts())
	if !ok {
		return
	}
	items, _, err := h.recordings.ListSessions(c.Request.Context(), recordingdomain.ListSessionsRequest{
		UserID: userID,
		Limit:  opts.Limit,
	})
	if err != nil {
		writeRecordingServiceError(c, err)
		return
	}
	responses := make([]RecordingResponse, 0, len(items))
	for i := range items {
		responses = append(responses, recordingResponse(&items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": responses, "next_cursor": nil})
}

func (h *Handler) getRecording(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.recordings == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording service is not configured", nil)
		return
	}
	session, err := h.recordings.GetSession(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeRecordingServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, recordingResponse(session))
}

func (h *Handler) uploadRecordingChunk(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.recordings == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording service is not configured", nil)
		return
	}
	chunkIndex, err := strconv.Atoi(strings.TrimSpace(c.Param("chunk_index")))
	if err != nil || chunkIndex < 0 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "chunk_index is invalid", stringPtr("chunk_index"))
		return
	}
	limit := recordingChunkSizeLimit(h)
	if c.Request.ContentLength > limit {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "chunk is too large", nil)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	durationMs, ok := parseOptionalInt64Header(c, "X-Chunk-Duration-Ms")
	if !ok {
		return
	}
	checksum := strings.TrimSpace(c.GetHeader("X-Chunk-SHA256"))
	var checksumPtr *string
	if checksum != "" {
		checksumPtr = &checksum
	}
	result, err := h.recordings.AppendChunk(c.Request.Context(), recordingdomain.AppendChunkRequest{
		UserID:      userID,
		RecordingID: c.Param("id"),
		ChunkIndex:  chunkIndex,
		MimeType:    c.GetHeader("Content-Type"),
		SHA256:      checksumPtr,
		DurationMs:  durationMs,
		Body:        c.Request.Body,
	})
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "chunk is too large", nil)
			return
		}
		writeRecordingServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, recordingChunkResponse(result))
}

func (h *Handler) stopRecording(c *gin.Context, publicID string) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.recordings == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording service is not configured", nil)
		return
	}
	var req stopRecordingRequest
	if !bindJSON(c, &req) {
		return
	}
	session, err := h.recordings.StopSession(c.Request.Context(), recordingdomain.StopSessionRequest{
		UserID:          userID,
		RecordingID:     publicID,
		FinalChunkIndex: req.FinalChunkIndex,
		DurationMs:      req.DurationMs,
		AutoTranscribe:  req.AutoTranscribe,
	})
	if err != nil {
		writeRecordingServiceError(c, err)
		return
	}
	h.notifyRecordingFinalizer()
	c.JSON(http.StatusAccepted, recordingResponse(session))
}

func (h *Handler) cancelRecording(c *gin.Context, publicID string) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.recordings == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording service is not configured", nil)
		return
	}
	if err := h.recordings.CancelSession(c.Request.Context(), userID, publicID); err != nil {
		writeRecordingServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": publicID, "status": string(models.RecordingStatusCanceled)})
}

func (h *Handler) retryFinalizeRecording(c *gin.Context, publicID string) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.recordings == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording service is not configured", nil)
		return
	}
	session, err := h.recordings.RetryFinalization(c.Request.Context(), userID, publicID)
	if err != nil {
		writeRecordingServiceError(c, err)
		return
	}
	h.notifyRecordingFinalizer()
	c.JSON(http.StatusAccepted, recordingResponse(session))
}

func (h *Handler) validateRecordingProfile(c *gin.Context, userID uint, publicProfileID *string) (*string, bool) {
	if publicProfileID == nil || strings.TrimSpace(*publicProfileID) == "" {
		return nil, true
	}
	profileID, ok := parseProfileID(strings.TrimSpace(*publicProfileID))
	if !ok {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "profile_id is invalid", stringPtr("profile_id"))
		return nil, false
	}
	exists, err := h.profiles.Exists(c.Request.Context(), userID, profileID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not validate profile", nil)
		return nil, false
	}
	if !exists {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "profile_id is invalid", stringPtr("profile_id"))
		return nil, false
	}
	return &profileID, true
}

func recordingOptionsJSON(c *gin.Context, language string, diarization *bool) (string, bool) {
	language = strings.TrimSpace(language)
	if language != "" && !validLanguage(language) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "language is invalid", stringPtr("options.language"))
		return "", false
	}
	options := map[string]any{}
	if language != "" {
		options["language"] = language
	}
	if diarization != nil {
		options["diarization"] = *diarization
	}
	bytes, err := json.Marshal(options)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not encode recording options", nil)
		return "", false
	}
	return string(bytes), true
}

func parseOptionalInt64Header(c *gin.Context, header string) (*int64, bool) {
	raw := strings.TrimSpace(c.GetHeader(header))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", strings.ToLower(header)+" is invalid", stringPtr(header))
		return nil, false
	}
	return &value, true
}

func recordingChunkSizeLimit(h *Handler) int64 {
	if h != nil && h.config != nil && h.config.Recordings.MaxChunkBytes > 0 {
		return h.config.Recordings.MaxChunkBytes
	}
	return 25 << 20
}

func writeRecordingServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, recordingdomain.ErrValidation):
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", validationMessage(err), nil)
	case errors.Is(err, recordingdomain.ErrNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		writeError(c, http.StatusNotFound, "NOT_FOUND", "recording not found", nil)
	case errors.Is(err, recordingdomain.ErrConflict):
		writeError(c, http.StatusConflict, "CONFLICT", "recording state conflict", nil)
	case errors.Is(err, context.Canceled):
		writeError(c, statusClientClosedRequest, "REQUEST_CANCELED", "request was canceled", nil)
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "recording operation failed", nil)
	}
}

func (h *Handler) notifyRecordingFinalizer() {
	if h != nil && h.finalizer != nil {
		h.finalizer.Notify()
	}
}

func validationMessage(err error) string {
	message := err.Error()
	if _, detail, ok := strings.Cut(message, ": "); ok {
		return detail
	}
	return "request is invalid"
}
