package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
)

type fakeTransport struct {
	mu       sync.Mutex
	requests []recordedRequest
	handler  func(*http.Request, []byte) (*http.Response, error)
}

type recordedRequest struct {
	Method string
	Path   string
	Body   []byte
}

func (t *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
	}
	t.mu.Lock()
	t.requests = append(t.requests, recordedRequest{Method: req.Method, Path: req.URL.Path, Body: body})
	t.mu.Unlock()
	if t.handler == nil {
		return jsonResponse(http.StatusOK, map[string]any{}), nil
	}
	return t.handler(req, body)
}

func (t *fakeTransport) seen() []recordedRequest {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]recordedRequest, len(t.requests))
	copy(out, t.requests)
	return out
}

func TestClientControlPlaneEndpoints(t *testing.T) {
	transport := &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/health":
			return jsonResponse(http.StatusOK, map[string]any{"ok": true}), nil
		case "GET /api/v1/provider":
			return jsonResponse(http.StatusOK, asrcontract.ProviderInfo{
				ContractVersion: asrcontract.ContractVersionV1,
				Provider:        asrcontract.ProviderIdentity{Name: "Remote"},
			}), nil
		case "GET /api/v1/models":
			return jsonResponse(http.StatusOK, []asrcontract.ModelCard{{
				ID:           "remote-transcriber",
				DisplayName:  "Remote Transcriber",
				Installed:    true,
				Default:      true,
				Capabilities: asrcontract.Capabilities{Transcription: true, WordTimestamps: true},
			}}), nil
		case "GET /api/v1/status":
			return jsonResponse(http.StatusOK, asrcontract.ProviderStatus{
				State: asrcontract.ProviderStateIdle,
				Capacity: asrcontract.ProviderCapacity{
					MaxConcurrentJobs: 1,
					AvailableSlots:    1,
				},
			}), nil
		case "GET /api/v1/models/loaded":
			return jsonResponse(http.StatusOK, []asrcontract.LoadedModel{{ID: "remote-transcriber"}}), nil
		case "POST /api/v1/models/remote-transcriber:load":
			return jsonResponse(http.StatusOK, map[string]any{}), nil
		case "POST /api/v1/models/remote-transcriber:unload":
			return jsonResponse(http.StatusOK, map[string]any{}), nil
		default:
			return jsonResponse(http.StatusNotFound, providerError(asrcontract.CodeInvalidRequest, "unknown")), nil
		}
	}}
	client := newTestClient(t, transport)

	if err := client.Prepare(context.Background()); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	info, err := client.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if info.Provider.ID != "remote-a" {
		t.Fatalf("provider id = %q, want remote-a", info.Provider.ID)
	}
	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if models[0].Provider != "remote-a" {
		t.Fatalf("model provider = %q, want remote-a", models[0].Provider)
	}
	capabilities, err := client.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if len(capabilities) != 1 || capabilities[0].Capabilities[0] != "transcription" {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
	if _, err := client.Status(context.Background()); err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if _, err := client.LoadedModels(context.Background()); err != nil {
		t.Fatalf("LoadedModels returned error: %v", err)
	}
	if err := client.LoadModel(context.Background(), asrcontract.LoadModelRequest{Model: "remote-transcriber"}); err != nil {
		t.Fatalf("LoadModel returned error: %v", err)
	}
	if err := client.UnloadModel(context.Background(), asrcontract.UnloadModelRequest{Model: "remote-transcriber"}); err != nil {
		t.Fatalf("UnloadModel returned error: %v", err)
	}
}

func TestClientTranscribeSubmitsPollsAndReplaysProgress(t *testing.T) {
	progress := 0.45
	transport := &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/jobs":
			var create jobCreateRequest
			if err := json.Unmarshal(body, &create); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if create.Operation != asrcontract.OperationTranscription || create.Transcription == nil {
				t.Fatalf("unexpected create request: %#v", create)
			}
			if create.Transcription.Audio.Path != "/mnt/audio/job.wav" || create.Transcription.Audio.SampleRate != 16000 || create.Transcription.Audio.Channels != 1 {
				t.Fatalf("unexpected audio input: %#v", create.Transcription.Audio)
			}
			return jsonResponse(http.StatusOK, jobCreateResponse{JobID: "job-remote-1"}), nil
		case "GET /api/v1/jobs/job-remote-1":
			return jsonResponse(http.StatusOK, jobStatusResponse{
				JobID:  "job-remote-1",
				Status: "completed",
				Progress: &asrcontract.ProviderProgress{
					Stage:     asrcontract.StageTranscribing,
					Progress:  &progress,
					Operation: asrcontract.OperationTranscription,
					Timestamp: time.Now(),
				},
				Transcription: &asrcontract.TranscriptionResult{
					Model:    "remote-transcriber",
					Language: "en",
					Text:     "hello world",
					Segments: []asrcontract.TranscriptSegment{{ID: "s1", Start: 0, End: 1, Speaker: "SPEAKER_00", Text: "hello world"}},
					Words:    []asrcontract.TranscriptWord{{Start: 0, End: 0.5, Word: "hello", Speaker: "SPEAKER_00"}},
				},
			}), nil
		case "GET /api/v1/jobs/job-remote-1/events":
			return jsonResponse(http.StatusOK, eventsResponse{Events: []asrcontract.ProviderProgress{{
				Stage:     asrcontract.StagePostprocessing,
				Operation: asrcontract.OperationTranscription,
				Timestamp: time.Now(),
			}}}), nil
		default:
			return jsonResponse(http.StatusNotFound, providerError(asrcontract.CodeInvalidRequest, "unknown")), nil
		}
	}}
	client := newTestClient(t, transport)
	sink := &recordingSink{}

	result, err := client.Transcribe(context.Background(), engineprovider.TranscriptionRequest{
		JobID:     "local-job",
		AudioPath: "/mnt/audio/job.wav",
		ModelID:   "remote-transcriber",
		Language:  "en",
		Progress:  sink,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if result.Text != "hello world" || result.ModelID != "remote-transcriber" || result.EngineID != "remote-a" {
		t.Fatalf("unexpected transcription result: %#v", result)
	}
	if len(sink.events) != 2 {
		t.Fatalf("progress events = %d, want 2", len(sink.events))
	}
}

func TestClientDiarizeAndSpeakerIdentification(t *testing.T) {
	transport := &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/jobs":
			var create jobCreateRequest
			if err := json.Unmarshal(body, &create); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if create.Operation == asrcontract.OperationDiarization {
				return jsonResponse(http.StatusOK, jobCreateResponse{JobID: "diarize-job"}), nil
			}
			if create.Operation == asrcontract.OperationSpeakerIdentification {
				return jsonResponse(http.StatusOK, jobCreateResponse{JobID: "speaker-job"}), nil
			}
			return jsonResponse(http.StatusBadRequest, providerError(asrcontract.CodeInvalidRequest, "bad operation")), nil
		case "GET /api/v1/jobs/diarize-job":
			return jsonResponse(http.StatusOK, jobStatusResponse{
				JobID:       "diarize-job",
				Status:      "completed",
				Diarization: &asrcontract.DiarizationResult{Model: "diarizer", Segments: []asrcontract.DiarizationSegment{{Start: 0, End: 1, Speaker: "SPEAKER_00"}}},
			}), nil
		case "GET /api/v1/jobs/speaker-job":
			confidence := 0.9
			return jsonResponse(http.StatusOK, jobStatusResponse{
				JobID:                 "speaker-job",
				Status:                "completed",
				SpeakerIdentification: &asrcontract.SpeakerIDResult{Model: "speaker-id", Speakers: []asrcontract.SpeakerIdentity{{Speaker: "SPEAKER_00", Label: "Ada", Confidence: &confidence}}},
			}), nil
		case "GET /api/v1/jobs/diarize-job/events", "GET /api/v1/jobs/speaker-job/events":
			return jsonResponse(http.StatusOK, eventsResponse{}), nil
		default:
			return jsonResponse(http.StatusNotFound, providerError(asrcontract.CodeInvalidRequest, "unknown")), nil
		}
	}}
	client := newTestClient(t, transport)

	diarization, err := client.Diarize(context.Background(), engineprovider.DiarizationRequest{JobID: "job", AudioPath: "/mnt/audio.wav", ModelID: "diarizer"})
	if err != nil {
		t.Fatalf("Diarize returned error: %v", err)
	}
	if len(diarization.Segments) != 1 || diarization.EngineID != "remote-a" {
		t.Fatalf("unexpected diarization result: %#v", diarization)
	}
	speakers, err := client.IdentifySpeakers(context.Background(), asrcontract.SpeakerIDRequest{RequestID: "job", Model: "speaker-id"})
	if err != nil {
		t.Fatalf("IdentifySpeakers returned error: %v", err)
	}
	if len(speakers.Speakers) != 1 || speakers.Speakers[0].Label != "Ada" {
		t.Fatalf("unexpected speaker result: %#v", speakers)
	}
}

func TestClientMapsProviderErrorsAndSanitizesDetails(t *testing.T) {
	transport := &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
		return jsonResponse(http.StatusConflict, map[string]any{
			"error": asrcontract.ProviderError{
				Code:      asrcontract.CodeProviderBusy,
				Message:   "busy",
				Retryable: true,
				Details: map[string]any{
					"provider_url": "http://secret-provider",
					"audio_path":   "/mnt/private.wav",
					"queue_depth":  float64(1),
				},
			},
		}), nil
	}}
	client := newTestClient(t, transport)

	_, err := client.Models(context.Background())
	var providerErr *asrcontract.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T, want ProviderError", err)
	}
	if providerErr.Code != asrcontract.CodeProviderBusy || !providerErr.Retryable {
		t.Fatalf("unexpected provider error: %#v", providerErr)
	}
	if _, ok := providerErr.Details["provider_url"]; ok {
		t.Fatalf("provider url leaked in details: %#v", providerErr.Details)
	}
	if _, ok := providerErr.Details["audio_path"]; ok {
		t.Fatalf("audio path leaked in details: %#v", providerErr.Details)
	}
	if providerErr.Details["queue_depth"] != float64(1) {
		t.Fatalf("safe detail missing: %#v", providerErr.Details)
	}
}

func TestClientRejectsMalformedAndOversizedResponses(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		client := newTestClient(t, &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
			return textResponse(http.StatusOK, "{"), nil
		}})
		_, err := client.Models(context.Background())
		if !asrcontract.IsCode(err, asrcontract.CodeInvalidRequest) {
			t.Fatalf("error = %v, want INVALID_REQUEST", err)
		}
	})
	t.Run("oversized", func(t *testing.T) {
		client := newTestClient(t, &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
			return textResponse(http.StatusOK, strings.Repeat("x", 64)), nil
		}})
		client.maxResponseBytes = 8
		_, err := client.Models(context.Background())
		if !asrcontract.IsCode(err, asrcontract.CodeInvalidRequest) {
			t.Fatalf("error = %v, want INVALID_REQUEST", err)
		}
	})
}

func TestClientMapsRequestTimeout(t *testing.T) {
	client := newTestClient(t, &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}})
	client.timeout = time.Nanosecond
	_, err := client.Models(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline", err)
	}
}

func TestClientCancelsRemoteJobWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/jobs":
			return jsonResponse(http.StatusOK, jobCreateResponse{JobID: "job-cancel"}), nil
		case "GET /api/v1/jobs/job-cancel":
			return jsonResponse(http.StatusOK, jobStatusResponse{
				JobID:  "job-cancel",
				Status: "processing",
				Progress: &asrcontract.ProviderProgress{
					Stage:     asrcontract.StageTranscribing,
					Operation: asrcontract.OperationTranscription,
					Timestamp: time.Now(),
				},
			}), nil
		case "GET /api/v1/jobs/job-cancel/events":
			return jsonResponse(http.StatusOK, eventsResponse{}), nil
		case "DELETE /api/v1/jobs/job-cancel":
			return jsonResponse(http.StatusOK, map[string]any{}), nil
		default:
			return jsonResponse(http.StatusNotFound, providerError(asrcontract.CodeInvalidRequest, "unknown")), nil
		}
	}}
	client := newTestClient(t, transport)
	client.pollInterval = time.Hour
	_, err := client.Transcribe(ctx, engineprovider.TranscriptionRequest{
		JobID:    "local-job",
		ModelID:  "remote-transcriber",
		Progress: cancelingSink{cancel: cancel},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	var sawDelete bool
	for _, request := range transport.seen() {
		if request.Method == http.MethodDelete && request.Path == "/api/v1/jobs/job-cancel" {
			sawDelete = true
		}
	}
	if !sawDelete {
		t.Fatalf("remote cancel DELETE was not sent; requests=%#v", transport.seen())
	}
}

func TestClientSurfacesUnsupportedOperation(t *testing.T) {
	client := newTestClient(t, &fakeTransport{handler: func(req *http.Request, body []byte) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, providerError(asrcontract.CodeUnsupportedOperation, "speaker identification is unsupported")), nil
	}})
	_, err := client.IdentifySpeakers(context.Background(), asrcontract.SpeakerIDRequest{RequestID: "job", Model: "speaker-id"})
	if !asrcontract.IsCode(err, asrcontract.CodeUnsupportedOperation) {
		t.Fatalf("error = %v, want unsupported operation", err)
	}
}

type recordingSink struct {
	events []asrcontract.ProviderProgress
}

func (s *recordingSink) Report(ctx context.Context, event asrcontract.ProviderProgress) {
	s.events = append(s.events, event)
}

type cancelingSink struct {
	cancel context.CancelFunc
}

func (s cancelingSink) Report(ctx context.Context, event asrcontract.ProviderProgress) {
	s.cancel()
}

func newTestClient(t *testing.T, transport http.RoundTripper) *Client {
	t.Helper()
	client, err := NewClient(Config{
		ID:           "remote-a",
		BaseURL:      "https://provider.test/api",
		HTTPClient:   &http.Client{Transport: transport},
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client
}

func jsonResponse(status int, value any) *http.Response {
	payload, _ := json.Marshal(value)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(payload)),
	}
}

func textResponse(status int, value string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       io.NopCloser(strings.NewReader(value)),
	}
}

func providerError(code asrcontract.ErrorCode, message string) map[string]any {
	return map[string]any{"error": asrcontract.ProviderError{Code: code, Message: message}}
}
