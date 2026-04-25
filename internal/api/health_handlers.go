package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func (h *Handler) ready(c *gin.Context) {
	if h.readinessCheck != nil {
		if err := h.readinessCheck(); err != nil {
			writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service is not ready", nil)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "ok"})
}
func (h *Handler) notImplemented(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		writeError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", feature+" is not implemented yet", nil)
	}
}
func (h *Handler) bindJSONPlaceholder(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength != 0 {
			var body map[string]any
			if err := c.ShouldBindJSON(&body); err != nil {
				writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON", nil)
				return
			}
		}
		writeError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", feature+" is not implemented yet", nil)
	}
}
