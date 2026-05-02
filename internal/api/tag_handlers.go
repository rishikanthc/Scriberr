package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/tags"

	"github.com/gin-gonic/gin"
)

type createTagRequest struct {
	Name        string  `json:"name"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

type updateTagRequest struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

type replaceTranscriptionTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

func (h *Handler) listTags(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	opts, ok := parseTagListQuery(c)
	if !ok {
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	items, _, err := h.tags.ListTags(c.Request.Context(), tags.ListRequest{
		UserID: userID,
		Search: opts.search,
		Offset: opts.offset,
		Limit:  opts.limit + 1,
	})
	if err != nil {
		writeTagServiceError(c, err)
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
		responseItems = append(responseItems, tagResponse(&items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": responseItems, "next_cursor": nextCursor})
}

func (h *Handler) createTag(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req createTagRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	created, err := h.tags.CreateTag(c.Request.Context(), tags.CreateRequest{
		UserID:      userID,
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
	})
	if err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, tagResponse(created))
}

func (h *Handler) getTag(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	tag, err := h.tags.GetTag(c.Request.Context(), userID, c.Param("tag_id"))
	if err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, tagResponse(tag))
}

func (h *Handler) updateTag(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req updateTagRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	updated, err := h.tags.UpdateTag(c.Request.Context(), tags.UpdateRequest{
		UserID:      userID,
		TagID:       c.Param("tag_id"),
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
	})
	if err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, tagResponse(updated))
}

func (h *Handler) deleteTag(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	if err := h.tags.DeleteTag(c.Request.Context(), userID, c.Param("tag_id")); err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listTranscriptionTags(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	items, err := h.tags.ListTranscriptionTags(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tagListResponse(items), "next_cursor": nil})
}

func (h *Handler) replaceTranscriptionTags(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req replaceTranscriptionTagsRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	items, err := h.tags.ReplaceTranscriptionTags(c.Request.Context(), tags.ReplaceTranscriptionTagsRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		TagIDs:          req.TagIDs,
	})
	if err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tagListResponse(items), "next_cursor": nil})
}

func (h *Handler) addTranscriptionTag(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	items, err := h.tags.AddTagToTranscription(c.Request.Context(), tags.TranscriptionTagRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		TagID:           c.Param("tag_id"),
	})
	if err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tagListResponse(items), "next_cursor": nil})
}

func (h *Handler) removeTranscriptionTag(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.tags == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag service is not configured", nil)
		return
	}
	if err := h.tags.RemoveTagFromTranscription(c.Request.Context(), tags.TranscriptionTagRequest{
		UserID:          userID,
		TranscriptionID: c.Param("id"),
		TagID:           c.Param("tag_id"),
	}); err != nil {
		writeTagServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type tagListOptions struct {
	limit  int
	offset int
	search string
}

func parseTagListQuery(c *gin.Context) (tagListOptions, bool) {
	limit := defaultListLimit
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > maxListLimit {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit is invalid", stringPtr("limit"))
			return tagListOptions{}, false
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
			return tagListOptions{}, false
		}
		offset = parsedOffset
	}
	return tagListOptions{limit: limit, offset: offset, search: strings.TrimSpace(c.Query("q"))}, true
}

func tagListResponse(items []models.AudioTag) []gin.H {
	response := make([]gin.H, 0, len(items))
	for i := range items {
		response = append(response, tagResponse(&items[i]))
	}
	return response
}

func tagResponse(tag *models.AudioTag) gin.H {
	color := any(nil)
	if tag.Color != nil {
		color = *tag.Color
	}
	description := any(nil)
	if tag.Description != nil {
		description = *tag.Description
	}
	return gin.H{
		"id":          tags.PublicTagID(tag.ID),
		"name":        tag.Name,
		"color":       color,
		"description": description,
		"created_at":  tag.CreatedAt,
		"updated_at":  tag.UpdatedAt,
	}
}

func writeTagServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, tags.ErrValidation):
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", cleanTagServiceError(err), nil)
	case errors.Is(err, tags.ErrConflict):
		writeError(c, http.StatusConflict, "CONFLICT", "tag already exists", nil)
	case errors.Is(err, tags.ErrNotFound):
		writeError(c, http.StatusNotFound, "NOT_FOUND", "tag or transcription not found", nil)
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "tag operation failed", nil)
	}
}

func cleanTagServiceError(err error) string {
	message := strings.TrimSpace(strings.TrimPrefix(err.Error(), tags.ErrValidation.Error()+":"))
	if message == "" {
		return "tag request is invalid"
	}
	return message
}

func transcriptionTagFilters(c *gin.Context) []string {
	var refs []string
	for _, value := range c.QueryArray("tag") {
		refs = appendTagFilterValues(refs, value)
	}
	refs = appendTagFilterValues(refs, c.Query("tags"))
	return refs
}

func appendTagFilterValues(refs []string, raw string) []string {
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			refs = append(refs, trimmed)
		}
	}
	return refs
}

func parseTagMatch(c *gin.Context) (bool, bool) {
	switch strings.TrimSpace(c.DefaultQuery("tag_match", "any")) {
	case "", "any":
		return false, true
	case "all":
		return true, true
	default:
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "tag_match is invalid", stringPtr("tag_match"))
		return false, false
	}
}
