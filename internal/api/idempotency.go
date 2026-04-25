package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"

	"github.com/gin-gonic/gin"
)

const maxIdempotencyBodyBytes int64 = 64 << 20

var validIdempotencyKey = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type idempotencyStore struct {
	mu      sync.Mutex
	entries map[string]*idempotencyEntry
}

type idempotencyEntry struct {
	fingerprint string
	status      int
	header      http.Header
	body        []byte
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{entries: map[string]*idempotencyEntry{}}
}

func (h *Handler) idempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.handleIdempotency(c, func() { c.Next() }) {
			c.Abort()
			return
		}
	}
}

func (h *Handler) runIdempotent(c *gin.Context, handler gin.HandlerFunc) {
	if !h.handleIdempotency(c, func() { handler(c) }) {
		return
	}
	handler(c)
}

func (h *Handler) handleIdempotency(c *gin.Context, next func()) bool {
	key := c.GetHeader("Idempotency-Key")
	if key == "" {
		return true
	}
	if !validIdempotencyKey.MatchString(key) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Idempotency-Key is invalid", stringPtr("Idempotency-Key"))
		return false
	}
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxIdempotencyBodyBytes+1))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body could not be read", nil)
		return false
	}
	if int64(len(body)) > maxIdempotencyBodyBytes {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large for idempotent retry", nil)
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	storeKey := idempotencyStoreKey(userID, c.Request.Method, c.Request.URL.Path, key)
	fingerprint := idempotencyFingerprint(c, body)
	if entry := h.idempotency.get(storeKey); entry != nil {
		if entry.fingerprint != fingerprint {
			writeError(c, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key was already used with a different request", stringPtr("Idempotency-Key"))
			return false
		}
		writeCachedIdempotencyResponse(c, entry)
		return false
	}

	recorder := &idempotencyResponseWriter{ResponseWriter: c.Writer, body: bytes.Buffer{}}
	c.Writer = recorder
	next()
	if recorder.status == 0 {
		recorder.status = http.StatusOK
	}
	if recorder.status >= 200 && recorder.status < 300 {
		h.idempotency.put(storeKey, &idempotencyEntry{
			fingerprint: fingerprint,
			status:      recorder.status,
			header:      recorder.Header().Clone(),
			body:        append([]byte(nil), recorder.body.Bytes()...),
		})
	}
	return false
}

type idempotencyResponseWriter struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *idempotencyResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *idempotencyResponseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *idempotencyResponseWriter) WriteString(data string) (int, error) {
	w.body.WriteString(data)
	return w.ResponseWriter.WriteString(data)
}

func (s *idempotencyStore) get(key string) *idempotencyEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.entries[key]
}

func (s *idempotencyStore) put(key string, entry *idempotencyEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = entry
}

func idempotencyStoreKey(userID uint, method, path, key string) string {
	return hex.EncodeToString([]byte(fmt.Sprintf("%d %s %s %s", userID, method, path, key)))
}

func idempotencyFingerprint(c *gin.Context, body []byte) string {
	hash := sha256.New()
	hash.Write([]byte(c.Request.Method))
	hash.Write([]byte{0})
	hash.Write([]byte(c.Request.URL.Path))
	hash.Write([]byte{0})
	hash.Write([]byte(c.GetHeader("Content-Type")))
	hash.Write([]byte{0})
	hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil))
}

func writeCachedIdempotencyResponse(c *gin.Context, entry *idempotencyEntry) {
	for key, values := range entry.header {
		if key == "X-Request-Id" {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Writer.WriteHeader(entry.status)
	if len(entry.body) > 0 {
		_, _ = c.Writer.Write(entry.body)
	}
}
