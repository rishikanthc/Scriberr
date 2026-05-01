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
}

type eventSubscriber struct {
	id              string
	transcriptionID string
	ch              chan apiEvent
}

type eventBroker struct {
	mu          sync.Mutex
	subscribers map[string]*eventSubscriber
}

func newEventBroker() *eventBroker {
	return &eventBroker{subscribers: map[string]*eventSubscriber{}}
}

func (b *eventBroker) subscribe(transcriptionID string) (*eventSubscriber, func()) {
	sub := &eventSubscriber{
		id:              randomHex(8),
		transcriptionID: transcriptionID,
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
		if sub.transcriptionID == "" || sub.transcriptionID == event.TranscriptionID {
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

	sub, unsubscribe := h.events.subscribe(transcriptionID)
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
	h.events.publish(apiEvent{Name: name, Data: data})
}

func (h *Handler) PublishFileEvent(_ context.Context, name string, payload map[string]any) {
	if h == nil {
		return
	}
	h.publishEvent(name, gin.H(payload))
}

func (h *Handler) publishTranscriptionEvent(name, transcriptionID string, data gin.H) {
	h.events.publish(apiEvent{Name: name, Data: data, TranscriptionID: transcriptionID})
}
