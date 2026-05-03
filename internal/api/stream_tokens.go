package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	streamTokenTTL                   = 10 * time.Minute
	streamTokenResourceFileAudio     = "file_audio"
	streamTokenResourceTranscription = "transcription_audio"
)

type streamTokenPayload struct {
	UserID   uint   `json:"user_id"`
	Resource string `json:"resource"`
	ID       string `json:"id"`
	Expires  int64  `json:"expires"`
}

func (h *Handler) issueFileAudioToken(c *gin.Context) {
	userID, id, ok := h.fileRequestIdentity(c)
	if !ok {
		return
	}
	if _, err := h.files.Get(c.Request.Context(), userID, id); err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return
	}
	publicID := "file_" + id
	h.writeStreamToken(c, userID, streamTokenResourceFileAudio, publicID, "/api/v1/files/"+publicID+"/audio")
}

func (h *Handler) issueTranscriptionAudioToken(c *gin.Context) {
	userID, id, ok := h.transcriptionRequestIdentity(c, c.Param("id"))
	if !ok {
		return
	}
	if _, err := h.transcriptions.Get(c.Request.Context(), userID, id); err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription not found", nil)
		return
	}
	publicID := "tr_" + id
	h.writeStreamToken(c, userID, streamTokenResourceTranscription, publicID, "/api/v1/transcriptions/"+publicID+"/audio")
}

func (h *Handler) writeStreamToken(c *gin.Context, userID uint, resource, id, urlPath string) {
	expiresAt := time.Now().Add(streamTokenTTL)
	token, err := h.signStreamToken(streamTokenPayload{
		UserID:   userID,
		Resource: resource,
		ID:       id,
		Expires:  expiresAt.Unix(),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not issue stream token", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
		"url":        urlPath + "?stream_token=" + token,
	})
}

func (h *Handler) signStreamToken(payload streamTokenPayload) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encodedBody := base64.RawURLEncoding.EncodeToString(body)
	signature := h.streamTokenSignature(encodedBody)
	return encodedBody + "." + signature, nil
}

func (h *Handler) validateStreamToken(token, resource, id string) (streamTokenPayload, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return streamTokenPayload{}, false
	}
	expected := h.streamTokenSignature(parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return streamTokenPayload{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return streamTokenPayload{}, false
	}
	var payload streamTokenPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return streamTokenPayload{}, false
	}
	if payload.UserID == 0 || payload.Resource != resource || payload.ID != id || payload.Expires <= time.Now().Unix() {
		return streamTokenPayload{}, false
	}
	return payload, true
}

func (h *Handler) streamTokenSignature(encodedPayload string) string {
	secret := ""
	if h != nil && h.config != nil {
		secret = h.config.JWTSecret
	}
	mac := hmac.New(sha256.New, []byte("scriberr-stream-token:"+secret))
	mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (h *Handler) authenticateStreamToken(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}
	token := strings.TrimSpace(c.Query("stream_token"))
	if token == "" {
		return false
	}
	resource, id, ok := streamTokenRoute(c)
	if !ok {
		return false
	}
	payload, ok := h.validateStreamToken(token, resource, id)
	if !ok {
		return false
	}
	if h.account != nil {
		user, err := h.account.ValidateActiveUser(c.Request.Context(), payload.UserID)
		if err != nil {
			return false
		}
		c.Set("username", user.Username)
		c.Set("role", user.Role)
	} else {
		c.Set("role", "")
	}
	c.Set("auth_type", "stream_token")
	c.Set("user_id", payload.UserID)
	return true
}

func streamTokenRoute(c *gin.Context) (string, string, bool) {
	publicID := c.Param("id")
	if publicID == "" {
		return "", "", false
	}
	switch c.FullPath() {
	case "/api/v1/files/:id/audio":
		if !strings.HasPrefix(publicID, "file_") {
			return "", "", false
		}
		return streamTokenResourceFileAudio, publicID, true
	case "/api/v1/transcriptions/:id/audio":
		if !strings.HasPrefix(publicID, "tr_") {
			return "", "", false
		}
		return streamTokenResourceTranscription, publicID, true
	default:
		return "", "", false
	}
}
