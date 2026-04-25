package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, status int, code, message string, field *string) {
	if c.Writer.Written() {
		return
	}
	c.JSON(status, ErrorBody{Error: APIError{
		Code:      code,
		Message:   message,
		Field:     field,
		RequestID: requestID(c),
	}})
}
func requestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req_fallback"
	}
	return "req_" + hex.EncodeToString(b[:])
}
func bindJSON(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON", nil)
		return false
	}
	return true
}
