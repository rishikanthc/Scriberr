package api

import (
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/account"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listAPIKeys(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	keys, err := h.account.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list api keys", nil)
		return
	}
	items := make([]gin.H, 0, len(keys))
	for _, key := range keys {
		description := ""
		if key.Description != nil {
			description = *key.Description
		}
		items = append(items, gin.H{
			"id":           publicAPIKeyID(key.ID),
			"name":         key.Name,
			"description":  description,
			"key_preview":  keyPreview(key.KeyPrefix),
			"is_active":    key.RevokedAt == nil,
			"last_used_at": key.LastUsed,
			"created_at":   key.CreatedAt,
			"updated_at":   key.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}
func (h *Handler) createAPIKey(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	var req createAPIKeyRequest
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", stringPtr("name"))
		return
	}
	key, rawKey, err := h.account.CreateAPIKey(c.Request.Context(), userID, req.Name, req.Description)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create api key", nil)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":          publicAPIKeyID(key.ID),
		"name":        key.Name,
		"description": req.Description,
		"key":         rawKey,
		"key_preview": keyPreview(key.KeyPrefix),
		"created_at":  key.CreatedAt,
	})
}
func (h *Handler) deleteAPIKey(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	id, ok := parseAPIKeyID(c.Param("id"))
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "api key not found", nil)
		return
	}
	if err := h.account.DeleteAPIKey(c.Request.Context(), userID, id); errors.Is(err, account.ErrAPIKeyNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "api key not found", nil)
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete api key", nil)
		return
	}
	c.Status(http.StatusNoContent)
}
