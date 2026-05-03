package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type apiEvent struct {
	Name            string
	Data            gin.H
	TranscriptionID string
	UserID          uint
}

type eventSubscriber struct {
	id              string
	transcriptionID string
	userID          uint
	ch              chan apiEvent
}

type eventBroker struct {
	mu          sync.Mutex
	subscribers map[string]*eventSubscriber
}

func newEventBroker() *eventBroker {
	return &eventBroker{subscribers: map[string]*eventSubscriber{}}
}

func (b *eventBroker) subscribe(userID uint, transcriptionID string) (*eventSubscriber, func()) {
	sub := &eventSubscriber{
		id:              randomHex(8),
		transcriptionID: transcriptionID,
		userID:          userID,
		ch:              make(chan apiEvent, 16),
	}
	b.mu.Lock()
	b.subscribers[sub.id] = sub
	b.mu.Unlock()
	return sub, func() {
		b.mu.Lock()
		delete(b.subscribers, sub.id)
		b.mu.Unlock()
	}
}

func (b *eventBroker) publish(event apiEvent) {
	b.mu.Lock()
	subscribers := make([]*eventSubscriber, 0, len(b.subscribers))
	for _, sub := range b.subscribers {
		if sub.matches(event) {
			subscribers = append(subscribers, sub)
		}
	}
	b.mu.Unlock()

	for _, sub := range subscribers {
		select {
		case sub.ch <- event:
		default:
		}
	}
}

func (s *eventSubscriber) matches(event apiEvent) bool {
	if s.transcriptionID != "" && s.transcriptionID != event.TranscriptionID {
		return false
	}
	if event.UserID == 0 {
		return true
	}
	return s.userID == event.UserID
}

func (b *eventBroker) subscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subscribers)
}

func (h *Handler) streamEvents(c *gin.Context) {
	h.streamEventChannel(c, "")
}

func (h *Handler) streamTranscriptionEvents(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	h.streamEventChannel(c, "tr_"+job.ID)
}

func (h *Handler) streamEventChannel(c *gin.Context, transcriptionID string) {
	principal, ok := h.currentPrincipal(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "streaming is not supported", nil)
		return
	}
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Vary", "Authorization")
	c.Status(http.StatusOK)

	sub, unsubscribe := h.events.subscribe(principal.UserID, transcriptionID)
	defer unsubscribe()

	_, _ = c.Writer.Write([]byte(": connected\n\n"))
	flusher.Flush()

	heartbeatInterval := h.eventHeartbeat
	if heartbeatInterval <= 0 {
		heartbeatInterval = 25 * time.Second
	}
	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			_, _ = c.Writer.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		case event := <-sub.ch:
			writeSSE(c, flusher, event)
		}
	}
}

func writeSSE(c *gin.Context, flusher http.Flusher, event apiEvent) {
	payload, err := json.Marshal(event.Data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(c.Writer, "event: %s\n", event.Name)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
	flusher.Flush()
}

func (h *Handler) publishEvent(name string, data gin.H) {
	h.publishEventForUser(name, data, 0)
}

func (h *Handler) publishEventForUser(name string, data gin.H, userID uint) {
	h.events.publish(apiEvent{Name: name, Data: data, UserID: userID})
}

func (h *Handler) PublishFileEvent(_ context.Context, name string, payload map[string]any) {
	if h == nil {
		return
	}
	h.publishEventForUser(name, sanitizeEventPayload(payload), eventPayloadUserID(payload))
}

func (h *Handler) PublishTranscriptionEvent(_ context.Context, name string, transcriptionID string, payload map[string]any) {
	if h == nil {
		return
	}
	sanitized := sanitizeEventPayload(payload)
	userID := eventPayloadUserID(payload)
	h.publishTranscriptionEvent(name, transcriptionID, sanitized, userID)
	h.publishEventForUser(name, sanitized, userID)
}

func (h *Handler) publishTranscriptionEvent(name, transcriptionID string, data gin.H, userID uint) {
	h.events.publish(apiEvent{Name: name, Data: data, TranscriptionID: transcriptionID, UserID: userID})
}

func sanitizeEventPayload(payload map[string]any) gin.H {
	out := gin.H{}
	for key, value := range payload {
		if eventPayloadKeyIsInternal(key) {
			continue
		}
		out[key] = value
	}
	return out
}

func eventPayloadKeyIsInternal(key string) bool {
	switch key {
	case "user_id", "path", "audio_path", "source_file_path", "output_json_path", "output_srt_path", "output_vtt_path", "logs_path":
		return true
	default:
		return false
	}
}

func eventPayloadUserID(payload map[string]any) uint {
	if payload == nil {
		return 0
	}
	switch value := payload["user_id"].(type) {
	case uint:
		return value
	case int:
		if value > 0 {
			return uint(value)
		}
	case int64:
		if value > 0 {
			return uint(value)
		}
	case float64:
		if value > 0 {
			return uint(value)
		}
	}
	return 0
}
