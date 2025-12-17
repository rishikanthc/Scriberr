package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"scriberr/pkg/logger"
)

// Event represents a Server-Sent Event
type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Subscription represents a client subscription to a specific job
type Subscription struct {
	JobID   string
	Channel chan Event
}

// Message represents an internal broadcast message
type Message struct {
	JobID string
	Event Event
}

// Broadcaster manages SSE connections and broadcasting
type Broadcaster struct {
	subscribers map[string]map[chan Event]bool // JobID -> Set of Clients
	register    chan Subscription
	unregister  chan Subscription
	broadcast   chan Message
	shutdown    chan struct{}
	mutex       sync.RWMutex
}

// NewBroadcaster creates a new Broadcaster
func NewBroadcaster() *Broadcaster {
	b := &Broadcaster{
		subscribers: make(map[string]map[chan Event]bool),
		register:    make(chan Subscription),
		unregister:  make(chan Subscription),
		broadcast:   make(chan Message),
		shutdown:    make(chan struct{}),
	}

	go b.listen()
	return b
}

// listen handles the addition and removal of clients and broadcasting of messages
func (b *Broadcaster) listen() {
	for {
		select {
		case sub := <-b.register:
			b.mutex.Lock()
			if b.subscribers[sub.JobID] == nil {
				b.subscribers[sub.JobID] = make(map[chan Event]bool)
			}
			b.subscribers[sub.JobID][sub.Channel] = true
			b.mutex.Unlock()
			logger.Debug("New SSE client registered", "job_id", sub.JobID)

		case sub := <-b.unregister:
			b.mutex.Lock()
			if clients, ok := b.subscribers[sub.JobID]; ok {
				delete(clients, sub.Channel)
				close(sub.Channel)
				if len(clients) == 0 {
					delete(b.subscribers, sub.JobID)
				}
			}
			b.mutex.Unlock()
			logger.Debug("SSE client unregistered", "job_id", sub.JobID)

		case msg := <-b.broadcast:
			b.mutex.RLock()
			// Send only to subscribers of this job
			if clients, ok := b.subscribers[msg.JobID]; ok {
				for s := range clients {
					// Send non-blocking
					select {
					case s <- msg.Event:
					default:
						logger.Warn("Skipping slow SSE client", "job_id", msg.JobID)
					}
				}
			}
			b.mutex.RUnlock()

		case <-b.shutdown:
			b.mutex.Lock()
			logger.Info("Broadcaster shutting down")
			for _, clients := range b.subscribers {
				for s := range clients {
					close(s)
				}
			}
			b.subscribers = nil
			b.mutex.Unlock()
			return
		}
	}
}

// Shutdown stops the broadcaster and closes all client connections
func (b *Broadcaster) Shutdown() {
	close(b.shutdown)
}

// ServeHTTP handles the SSE connection
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Require Job ID
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	// Set headers for SSE
	// Note: CORS headers are handled by the router middleware
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Create a channel for this client
	messageChan := make(chan Event)
	subscription := Subscription{JobID: jobID, Channel: messageChan}

	// Register subscription
	b.register <- subscription

	// Ensure cleanup on exit
	defer func() {
		// Use select to avoid blocking if the broadcaster has already shut down
		select {
		case b.unregister <- subscription:
		case <-b.shutdown:
			// Broadcaster is shutting down/stopped, no need to deregister
			logger.Debug("Skipping SSE client deregistration (shutdown)")
		}
	}()

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\", \"job_id\":\"%s\"}\n\n", jobID)
	flusher.Flush()

	// Keep connection open and push events
	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-messageChan:
			if !ok {
				return // Channel closed, exit handler
			}
			data, err := json.Marshal(msg)
			if err != nil {
				logger.Error("Failed to marshal SSE message", "error", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			// Keep-alive heartbeat
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// Broadcast sends an event to clients subscribed to the specific job
func (b *Broadcaster) Broadcast(jobID string, eventType string, payload interface{}) {
	b.broadcast <- Message{
		JobID: jobID,
		Event: Event{
			Type:    eventType,
			Payload: payload,
		},
	}
}
