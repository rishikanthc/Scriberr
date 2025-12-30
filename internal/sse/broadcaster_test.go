package sse

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBroadcaster(t *testing.T) {
	b := NewBroadcaster()

	// 1. Test ServeHTTP connection
	req := httptest.NewRequest("GET", "/events?job_id=test-job-1", nil)
	w := httptest.NewRecorder()

	// Use a context with timeout to simulate a client disconnecting after receiving messages
	// In a real scenario, the ServeHTTP blocks until client disconnects
	// We need to run ServeHTTP in a goroutine and consume the response body
	// However, httptest.Recorder buffers everything, so it wont work well out of the box for streaming if we wait for it to return.
	// Instead we can use a custom writer or just test the Broadcast logic separately?
	// Actually, we can test that connecting establishes the headers correctly.

	// Let's test headers first without blocking
	go b.ServeHTTP(w, req)
	time.Sleep(100 * time.Millisecond) // Wait for connection

	// Check headers
	if contentType := w.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", contentType)
	}

	// 2. Test Broadcasting
	jobID := "test-job-1"
	eventType := "status_update"
	testPayload := map[string]string{"status": "completed"}
	b.Broadcast(jobID, eventType, testPayload)

	time.Sleep(100 * time.Millisecond) // Allow processing

	// The recorder body should contain the data
	body := w.Body.String()

	// Check for connected message
	if !strings.Contains(body, "{\"type\":\"connected\", \"job_id\":\"test-job-1\"}") {
		t.Errorf("Expected connected message not found, got: %s", body)
	}

	// Check for broadcasted message
	expectedJSON, _ := json.Marshal(Event{Type: "status_update", Payload: testPayload})
	if !strings.Contains(body, string(expectedJSON)) {
		t.Errorf("Expected message %s not found in body: %s", string(expectedJSON), body)
	}
}
