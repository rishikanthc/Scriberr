package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
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
	done        chan struct{}
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
	storeKey := idempotencyStoreKey(userID, c.Request.Method, c.Request.URL.Path, key)
	fingerprint, body, reusable, ok := idempotencyFingerprint(c)
	if !ok {
		return false
	}
	entry, owner, conflict := h.idempotency.reserve(storeKey, fingerprint)
	if conflict {
		writeError(c, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key was already used with a different request", stringPtr("Idempotency-Key"))
		return false
	}
	if !owner {
		<-entry.done
		if entry.cached() {
			writeCachedIdempotencyResponse(c, entry)
			return false
		}
		if reusable {
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		h.executeAndMaybeCacheIdempotent(c, next, storeKey, fingerprint)
		return false
	}

	h.executeAndMaybeCacheIdempotent(c, next, storeKey, fingerprint)
	return false
}

func (h *Handler) executeAndMaybeCacheIdempotent(c *gin.Context, next func(), storeKey, fingerprint string) {
	recorder := &idempotencyResponseWriter{ResponseWriter: c.Writer, body: bytes.Buffer{}}
	c.Writer = recorder
	next()
	if recorder.status == 0 {
		recorder.status = http.StatusOK
	}
	if recorder.status >= 200 && recorder.status < 300 {
		h.idempotency.complete(storeKey, &idempotencyEntry{
			fingerprint: fingerprint,
			status:      recorder.status,
			header:      recorder.Header().Clone(),
			body:        append([]byte(nil), recorder.body.Bytes()...),
		})
		return
	}
	h.idempotency.forget(storeKey)
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

func (e *idempotencyEntry) cached() bool {
	return e.status >= 200 && e.status < 300
}

func (s *idempotencyStore) reserve(key, fingerprint string) (*idempotencyEntry, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry := s.entries[key]; entry != nil {
		return entry, false, entry.fingerprint != fingerprint
	}
	entry := &idempotencyEntry{
		fingerprint: fingerprint,
		done:        make(chan struct{}),
	}
	s.entries[key] = entry
	return entry, true, false
}

func (s *idempotencyStore) complete(key string, completed *idempotencyEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entries[key]
	if entry == nil {
		return
	}
	entry.status = completed.status
	entry.header = completed.header
	entry.body = completed.body
	close(entry.done)
}

func (s *idempotencyStore) forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entries[key]
	if entry == nil {
		return
	}
	delete(s.entries, key)
	close(entry.done)
}

func idempotencyStoreKey(userID uint, method, path, key string) string {
	return hex.EncodeToString([]byte(fmt.Sprintf("%d %s %s %s", userID, method, path, key)))
}

func idempotencyFingerprint(c *gin.Context) (string, []byte, bool, bool) {
	if isMultipartRequest(c.Request) {
		return streamingIdempotencyFingerprint(c), nil, false, true
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxIdempotencyBodyBytes+1))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body could not be read", nil)
		return "", nil, false, false
	}
	if int64(len(body)) > maxIdempotencyBodyBytes {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large for idempotent retry", nil)
		return "", nil, false, false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	hash := sha256.New()
	hash.Write([]byte(c.Request.Method))
	hash.Write([]byte{0})
	hash.Write([]byte(c.Request.URL.Path))
	hash.Write([]byte{0})
	hash.Write([]byte(c.GetHeader("Content-Type")))
	hash.Write([]byte{0})
	hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil)), body, true, true
}

func isMultipartRequest(req *http.Request) bool {
	contentType := strings.ToLower(strings.TrimSpace(req.Header.Get("Content-Type")))
	return strings.HasPrefix(contentType, "multipart/form-data")
}

func streamingIdempotencyFingerprint(c *gin.Context) string {
	hash := sha256.New()
	hash.Write([]byte(c.Request.Method))
	hash.Write([]byte{0})
	hash.Write([]byte(c.Request.URL.Path))
	hash.Write([]byte{0})
	hash.Write([]byte(c.GetHeader("Content-Type")))
	hash.Write([]byte{0})
	hash.Write([]byte(fmt.Sprintf("content-length:%d", c.Request.ContentLength)))
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
