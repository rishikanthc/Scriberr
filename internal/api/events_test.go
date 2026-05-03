package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scriberr/internal/models"
	recordingdomain "scriberr/internal/recording"
	"scriberr/internal/summarization"
	"scriberr/internal/transcription/orchestrator"
	"scriberr/internal/transcription/worker"

	"github.com/stretchr/testify/require"
)

func waitForSubscribers(t *testing.T, s *authTestServer, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if s.handler.events.subscriberCount() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.Equal(t, want, s.handler.events.subscriberCount())
}

func startEventStream(t *testing.T, s *authTestServer, token, path string) (*httptest.ResponseRecorder, context.CancelFunc, chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.router.ServeHTTP(recorder, req)
	}()
	waitForSubscribers(t, s, 1)
	return recorder, cancel, done
}

func stopEventStream(t *testing.T, cancel context.CancelFunc, done chan struct{}) {
	t.Helper()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event stream did not stop")
	}
}

func TestGlobalSSEReceivesFileEventsAndCleansUp(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/events")
	resp, body := uploadMultipart(t, s, token, "file", "meeting.wav", "audio/wav", []byte("RIFF----WAVEfmt data"), "Team sync")
	require.Equal(t, http.StatusCreated, resp.Code)
	fileID := body["id"].(string)

	stopEventStream(t, cancel, done)
	require.Equal(t, 0, s.handler.events.subscriberCount())
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
	require.Equal(t, "no-cache, no-transform", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no", recorder.Header().Get("X-Accel-Buffering"))
	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	stream := recorder.Body.String()
	require.Contains(t, stream, "event: file.ready")
	require.Contains(t, stream, `"id":"`+fileID+`"`)
	require.NotContains(t, stream, s.uploadDir)
}

func TestTranscriptionSSEFiltersByTranscription(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "First transcript",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	firstID := body["id"].(string)

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/transcriptions/"+firstID+"/events")
	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Second transcript",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	secondID := body["id"].(string)

	resp, _ = s.request(t, http.MethodPatch, "/api/v1/transcriptions/"+firstID, map[string]any{
		"title": "First transcript renamed",
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)

	stopEventStream(t, cancel, done)
	stream := recorder.Body.String()
	require.Contains(t, stream, "event: transcription.updated")
	require.Contains(t, stream, `"id":"`+firstID+`"`)
	require.False(t, strings.Contains(stream, secondID))
	require.NotContains(t, stream, s.uploadDir)
}

func TestGlobalSSEReceivesTranscriptionProgressOnce(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/events")
	s.handler.Publish(context.Background(), progressEventForTest("job-1", "transcription.progress"))

	require.Eventually(t, func() bool {
		return strings.Count(recorder.Body.String(), "event: transcription.progress") >= 1
	}, time.Second, 10*time.Millisecond)
	stopEventStream(t, cancel, done)

	require.Equal(t, 1, strings.Count(recorder.Body.String(), "event: transcription.progress"))
}

func TestSSEReceivesAnnotationEvents(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	transcriptionID := createTranscriptionForAnnotationTest(t, s, token)

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/transcriptions/"+transcriptionID+"/events")
	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/annotations", map[string]any{
		"kind":    "note",
		"content": "cache this",
		"quote":   "quoted text",
		"anchor": map[string]any{
			"start_ms": 10,
			"end_ms":   20,
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	annotationID := body["id"].(string)

	require.Eventually(t, func() bool {
		stream := recorder.Body.String()
		return strings.Contains(stream, "event: annotation.created") && strings.Contains(stream, `"id":"`+annotationID+`"`)
	}, time.Second, 10*time.Millisecond)
	stopEventStream(t, cancel, done)

	stream := recorder.Body.String()
	require.Contains(t, stream, `"transcription_id":"`+transcriptionID+`"`)
	require.Contains(t, stream, `"kind":"note"`)
	require.Contains(t, stream, `"status":"active"`)
	require.NotContains(t, stream, "cache this")
	require.NotContains(t, stream, "quoted text")
}

func TestSSEReceivesTagEvents(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	transcriptionID := createTranscriptionForTagTest(t, s, token, "Tagged stream")

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/transcriptions/"+transcriptionID+"/events")
	resp, body := s.request(t, http.MethodPost, "/api/v1/tags", map[string]any{
		"name":  "Stream Tag",
		"color": "#E87539",
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	tagID := body["id"].(string)

	resp, _ = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/tags/"+tagID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)

	require.Eventually(t, func() bool {
		stream := recorder.Body.String()
		return strings.Contains(stream, "event: transcription.tags.updated") &&
			strings.Contains(stream, `"transcription_id":"`+transcriptionID+`"`)
	}, time.Second, 10*time.Millisecond)
	stopEventStream(t, cancel, done)

	stream := recorder.Body.String()
	require.NotContains(t, stream, "Stream Tag")
	require.NotContains(t, stream, "#E87539")

	globalRecorder, globalCancel, globalDone := startEventStream(t, s, token, "/api/v1/events")
	resp, _ = s.request(t, http.MethodPatch, "/api/v1/tags/"+tagID, map[string]any{
		"name": "Renamed Stream Tag",
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Eventually(t, func() bool {
		stream := globalRecorder.Body.String()
		return strings.Contains(stream, "event: tag.updated") && strings.Contains(stream, `"id":"`+tagID+`"`)
	}, time.Second, 10*time.Millisecond)
	stopEventStream(t, globalCancel, globalDone)

	require.NotContains(t, globalRecorder.Body.String(), "Renamed Stream Tag")
}

func progressEventForTest(jobID, name string) orchestrator.ProgressEvent {
	return orchestrator.ProgressEvent{
		Name:     name,
		JobID:    jobID,
		UserID:   1,
		Stage:    "transcribing",
		Progress: 0.42,
		Status:   models.StatusProcessing,
	}
}

func TestSSERequiresAuthentication(t *testing.T) {
	s := newAuthTestServer(t)

	resp, body := s.request(t, http.MethodGet, "/api/v1/events", nil, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "UNAUTHORIZED", errBody["code"])
}

func TestEventBrokerUnsubscribeDoesNotCloseChannel(t *testing.T) {
	broker := newEventBroker()
	sub, unsubscribe := broker.subscribe("")

	unsubscribe()
	broker.publish(apiEvent{Name: "file.ready"})

	select {
	case _, ok := <-sub.ch:
		require.True(t, ok)
	default:
	}
	require.Equal(t, 0, broker.subscriberCount())
}

func TestSSESendsHeartbeat(t *testing.T) {
	s := newAuthTestServer(t)
	s.handler.eventHeartbeat = 10 * time.Millisecond
	token := registerForFileTests(t, s)

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/events")
	require.Eventually(t, func() bool {
		return strings.Contains(recorder.Body.String(), ": heartbeat")
	}, time.Second, 10*time.Millisecond)
	stopEventStream(t, cancel, done)
}

func TestEventAdaptersEmitPublicIDsAndOmitInternalDetails(t *testing.T) {
	handler := NewHandler(nil, nil, HandlerDependencies{})
	sub, unsubscribe := handler.events.subscribe("")
	defer unsubscribe()

	handler.PublishRecordingEvent(context.Background(), recordingdomain.Event{
		Name:            "recording.ready",
		RecordingID:     "rec_recording-1",
		Status:          models.RecordingStatusReady,
		Stage:           "ready",
		Progress:        1,
		FileID:          "file_file-1",
		TranscriptionID: "tr_transcription-1",
	})
	handler.PublishSummaryStatus(context.Background(), summarization.StatusEvent{
		Name:            "summary.completed",
		SummaryID:       "summary-1",
		TranscriptionID: "transcription-1",
		Status:          "completed",
		Truncated:       true,
	})
	handler.PublishStatus(context.Background(), worker.StatusEvent{
		Name:     "transcription.completed",
		JobID:    "transcription-1",
		FileID:   "file_file-1",
		Stage:    "completed",
		Progress: 1,
		Status:   models.StatusCompleted,
	})
	handler.PublishFileEvent(context.Background(), "file.ready", map[string]any{
		"id":     "file_file-1",
		"kind":   "audio",
		"status": "ready",
		"path":   "/tmp/private/audio.wav",
	})

	events := drainPublishedEvents(sub)
	require.Len(t, events, 5)
	body := eventsJSON(t, events)
	require.Contains(t, body, `"id":"rec_recording-1"`)
	require.Contains(t, body, `"file_id":"file_file-1"`)
	require.Contains(t, body, `"transcription_id":"tr_transcription-1"`)
	require.Contains(t, body, `"id":"summary-1"`)
	require.Contains(t, body, `"transcription_id":"tr_transcription-1"`)
	require.Contains(t, body, `"id":"tr_transcription-1"`)
	require.NotContains(t, body, "/tmp/private")
	require.NotContains(t, body, "audio.wav")
	require.NotContains(t, body, `"path"`)
}

func drainPublishedEvents(sub *eventSubscriber) []apiEvent {
	var events []apiEvent
	for {
		select {
		case event := <-sub.ch:
			events = append(events, event)
		default:
			return events
		}
	}
}

func eventsJSON(t *testing.T, events []apiEvent) string {
	t.Helper()
	var out strings.Builder
	for _, event := range events {
		payload, err := json.Marshal(event.Data)
		require.NoError(t, err)
		out.WriteString(event.Name)
		out.Write(payload)
	}
	return out.String()
}
