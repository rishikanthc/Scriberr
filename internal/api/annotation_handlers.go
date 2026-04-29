package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/annotations"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

type annotationAnchorRequest struct {
	StartMS   int64   `json:"start_ms"`
	EndMS     int64   `json:"end_ms"`
	StartWord *int    `json:"start_word,omitempty"`
	EndWord   *int    `json:"end_word,omitempty"`
	StartChar *int    `json:"start_char,omitempty"`
	EndChar   *int    `json:"end_char,omitempty"`
	TextHash  *string `json:"text_hash,omitempty"`
}

type createAnnotationRequest struct {
	Kind    string                   `json:"kind"`
	Content *string                  `json:"content,omitempty"`
	Color   *string                  `json:"color,omitempty"`
	Quote   string                   `json:"quote"`
	Anchor  *annotationAnchorRequest `json:"anchor"`
}

type updateAnnotationRequest struct {
	Content *string                  `json:"content,omitempty"`
	Color   *string                  `json:"color,omitempty"`
	Quote   *string                  `json:"quote,omitempty"`
	Anchor  *annotationAnchorRequest `json:"anchor,omitempty"`
}

type createAnnotationEntryRequest struct {
	Content string `json:"content"`
}

type updateAnnotationEntryRequest struct {
	Content string `json:"content"`
}

func (h *Handler) listAnnotations(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	opts, ok := parseAnnotationListQuery(c)
	if !ok {
		return
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}

	items, _, err := h.annotations.ListAnnotations(c.Request.Context(), annotations.ListRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		Kind:            opts.kind,
		UpdatedAfter:    opts.updatedAfter,
		Offset:          opts.offset,
		Limit:           opts.limit + 1,
	})
	if err != nil {
		writeAnnotationServiceError(c, err)
		return
	}

	nextCursor := any(nil)
	if len(items) > opts.limit {
		items = items[:opts.limit]
		nextCursor = encodeListCursor(listCursor{
			Sort:  "offset",
			Value: strconv.Itoa(opts.offset + opts.limit),
			ID:    items[len(items)-1].ID,
		})
	}
	responseItems := make([]gin.H, 0, len(items))
	for i := range items {
		responseItems = append(responseItems, annotationResponse(&items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": responseItems, "next_cursor": nextCursor})
}

func (h *Handler) createAnnotation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req createAnnotationRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.Anchor == nil {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "anchor is required", stringPtr("anchor"))
		return
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}
	created, err := h.annotations.CreateAnnotation(c.Request.Context(), annotations.CreateRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		Kind:            models.AnnotationKind(strings.TrimSpace(req.Kind)),
		Content:         req.Content,
		Color:           req.Color,
		Quote:           req.Quote,
		Anchor:          annotationAnchorCommand(*req.Anchor),
	})
	if err != nil {
		writeAnnotationServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, annotationResponse(created))
}

func (h *Handler) getAnnotation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}
	annotation, err := h.annotations.GetAnnotation(c.Request.Context(), userID, c.Param("id"), c.Param("annotation_id"))
	if err != nil {
		writeAnnotationServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, annotationResponse(annotation))
}

func (h *Handler) updateAnnotation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req updateAnnotationRequest
	if !bindJSON(c, &req) {
		return
	}
	var anchor *annotations.Anchor
	if req.Anchor != nil {
		value := annotationAnchorCommand(*req.Anchor)
		anchor = &value
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}
	updated, err := h.annotations.UpdateAnnotation(c.Request.Context(), annotations.UpdateRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		AnnotationID:    c.Param("annotation_id"),
		Content:         req.Content,
		Color:           req.Color,
		Quote:           req.Quote,
		Anchor:          anchor,
	})
	if err != nil {
		writeAnnotationServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, annotationResponse(updated))
}

func (h *Handler) deleteAnnotation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}
	if err := h.annotations.DeleteAnnotation(c.Request.Context(), userID, c.Param("id"), c.Param("annotation_id")); err != nil {
		writeAnnotationServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) createAnnotationEntry(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req createAnnotationEntryRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}
	entry, annotation, err := h.annotations.CreateAnnotationEntry(c.Request.Context(), annotations.CreateEntryRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		AnnotationID:    c.Param("annotation_id"),
		Content:         req.Content,
	})
	if err != nil {
		writeAnnotationServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, annotationEntryResponse(entry, annotation))
}

func (h *Handler) updateAnnotationEntry(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req updateAnnotationEntryRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}
	entry, annotation, err := h.annotations.UpdateAnnotationEntry(c.Request.Context(), annotations.UpdateEntryRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		AnnotationID:    c.Param("annotation_id"),
		EntryID:         c.Param("entry_id"),
		Content:         req.Content,
	})
	if err != nil {
		writeAnnotationServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, annotationEntryResponse(entry, annotation))
}

func (h *Handler) deleteAnnotationEntry(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.annotations == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation service is not configured", nil)
		return
	}
	if err := h.annotations.DeleteAnnotationEntry(c.Request.Context(), userID, c.Param("id"), c.Param("annotation_id"), c.Param("entry_id")); err != nil {
		writeAnnotationServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type annotationListOptions struct {
	limit        int
	offset       int
	kind         *models.AnnotationKind
	updatedAfter *time.Time
}

func parseAnnotationListQuery(c *gin.Context) (annotationListOptions, bool) {
	limit := defaultListLimit
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > maxListLimit {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit is invalid", stringPtr("limit"))
			return annotationListOptions{}, false
		}
		limit = parsed
	}

	offset := 0
	if rawCursor := strings.TrimSpace(c.Query("cursor")); rawCursor != "" {
		cursor, err := decodeListCursor(rawCursor)
		parsedOffset := 0
		if err == nil && cursor.Sort == "offset" {
			parsedOffset, err = strconv.Atoi(cursor.Value)
		}
		if err != nil || cursor.Sort != "offset" || parsedOffset < 0 {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "cursor is invalid", stringPtr("cursor"))
			return annotationListOptions{}, false
		}
		offset = parsedOffset
	}

	var kind *models.AnnotationKind
	if rawKind := strings.TrimSpace(c.Query("kind")); rawKind != "" {
		parsedKind := models.AnnotationKind(rawKind)
		switch parsedKind {
		case models.AnnotationKindHighlight, models.AnnotationKindNote:
			kind = &parsedKind
		default:
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "kind is invalid", stringPtr("kind"))
			return annotationListOptions{}, false
		}
	}

	var updatedAfter *time.Time
	if rawUpdatedAfter := strings.TrimSpace(c.Query("updated_after")); rawUpdatedAfter != "" {
		parsed, err := time.Parse(time.RFC3339, rawUpdatedAfter)
		if err != nil {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "updated_after is invalid", stringPtr("updated_after"))
			return annotationListOptions{}, false
		}
		updatedAfter = &parsed
	}

	return annotationListOptions{limit: limit, offset: offset, kind: kind, updatedAfter: updatedAfter}, true
}

func annotationAnchorCommand(req annotationAnchorRequest) annotations.Anchor {
	return annotations.Anchor{
		StartMS:   req.StartMS,
		EndMS:     req.EndMS,
		StartWord: req.StartWord,
		EndWord:   req.EndWord,
		StartChar: req.StartChar,
		EndChar:   req.EndChar,
		TextHash:  req.TextHash,
	}
}

func annotationResponse(annotation *models.TranscriptAnnotation) gin.H {
	content := any(nil)
	if annotation.Content != nil {
		content = *annotation.Content
	}
	color := any(nil)
	if annotation.Color != nil {
		color = *annotation.Color
	}
	anchor := gin.H{
		"start_ms": annotation.AnchorStartMS,
		"end_ms":   annotation.AnchorEndMS,
	}
	if annotation.AnchorStartWord != nil {
		anchor["start_word"] = *annotation.AnchorStartWord
	}
	if annotation.AnchorEndWord != nil {
		anchor["end_word"] = *annotation.AnchorEndWord
	}
	if annotation.AnchorStartChar != nil {
		anchor["start_char"] = *annotation.AnchorStartChar
	}
	if annotation.AnchorEndChar != nil {
		anchor["end_char"] = *annotation.AnchorEndChar
	}
	if annotation.AnchorTextHash != nil {
		anchor["text_hash"] = *annotation.AnchorTextHash
	}
	response := gin.H{
		"id":               annotations.PublicAnnotationID(annotation.ID),
		"transcription_id": "tr_" + annotation.TranscriptionID,
		"kind":             string(annotation.Kind),
		"content":          content,
		"color":            color,
		"quote":            annotation.Quote,
		"anchor":           anchor,
		"status":           annotation.Status,
		"created_at":       annotation.CreatedAt,
		"updated_at":       annotation.UpdatedAt,
	}
	if annotation.Kind == models.AnnotationKindNote {
		response["content"] = nil
		entries := make([]gin.H, 0, len(annotation.Entries))
		for i := range annotation.Entries {
			entries = append(entries, annotationEntryResponse(&annotation.Entries[i], annotation))
		}
		response["entries"] = entries
	}
	return response
}

func annotationEntryResponse(entry *models.TranscriptAnnotationEntry, annotation *models.TranscriptAnnotation) gin.H {
	transcriptionID := ""
	annotationID := annotations.PublicAnnotationID(entry.AnnotationID)
	if annotation != nil {
		transcriptionID = "tr_" + annotation.TranscriptionID
		annotationID = annotations.PublicAnnotationID(annotation.ID)
	}
	response := gin.H{
		"id":            annotations.PublicAnnotationEntryID(entry.ID),
		"annotation_id": annotationID,
		"content":       entry.Content,
		"created_at":    entry.CreatedAt,
		"updated_at":    entry.UpdatedAt,
	}
	if transcriptionID != "" {
		response["transcription_id"] = transcriptionID
	}
	return response
}

func writeAnnotationServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, annotations.ErrNotFound):
		writeError(c, http.StatusNotFound, "NOT_FOUND", "annotation not found", nil)
	case errors.Is(err, annotations.ErrValidation):
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", annotationValidationMessage(err), nil)
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "annotation operation failed", nil)
	}
}

func annotationValidationMessage(err error) string {
	message := err.Error()
	if _, value, ok := strings.Cut(message, annotations.ErrValidation.Error()+": "); ok {
		return value
	}
	return "annotation request is invalid"
}
