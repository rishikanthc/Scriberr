package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/engineprovider/contracttest"
)

func TestExampleProviderServerSatisfiesContract(t *testing.T) {
	server := newExampleProviderServer(t)
	defer server.Close()
	client, err := NewClient(Config{
		ID:           "example",
		BaseURL:      server.URL,
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	contracttest.RunProviderContract(t, client, contracttest.Options{
		RequiredModel:        "example-transcriber",
		RequiredCapabilities: []asrcontract.Capability{asrcontract.CapabilityTranscription},
	})

	result, err := client.Transcribe(t.Context(), engineprovider.TranscriptionRequest{
		JobID:     "contract-job",
		AudioPath: "/provider-input/audio/contract.wav",
		ModelID:   "example-transcriber",
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if result.Text != "hello from example provider" || result.EngineID != "example" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func newExampleProviderServer(t *testing.T) *httptest.Server {
	t.Helper()
	loadedAt := time.Now().UTC().Truncate(time.Millisecond)
	memory := 256
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/v1/provider", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, asrcontract.ProviderInfo{
			ContractVersion: asrcontract.ContractVersionV1,
			Provider:        asrcontract.ProviderIdentity{ID: "example", Name: "Example ASR Provider", Version: "0.1.0"},
			Runtime: asrcontract.RuntimeInfo{
				DeviceBackends:       []string{"cpu"},
				ActiveBackend:        "cpu",
				SupportsConcurrent:   false,
				MaxConcurrentJobs:    1,
				ProviderCapabilities: []asrcontract.Capability{asrcontract.CapabilityTranscription},
			},
			AudioInput: asrcontract.AudioInputSpec{
				RequiredSampleRate: 16000,
				RequiredChannels:   1,
				Formats:            []string{"wav"},
				PathMode:           asrcontract.PathModeMountedFile,
			},
		})
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		writeExampleJSON(t, w, []asrcontract.ModelCard{exampleModelCard()})
	})
	mux.HandleFunc("/v1/models/loaded", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, []asrcontract.LoadedModel{{ID: "example-transcriber", LoadedAt: &loadedAt, MemoryMB: &memory}})
	})
	mux.HandleFunc("/v1/models/example-transcriber:load", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, map[string]string{"status": "loaded"})
	})
	mux.HandleFunc("/v1/models/example-transcriber:unload", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, map[string]string{"status": "unloaded"})
	})
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, asrcontract.ProviderStatus{
			State:        asrcontract.ProviderStateIdle,
			LoadedModels: []asrcontract.LoadedModel{{ID: "example-transcriber", LoadedAt: &loadedAt, MemoryMB: &memory}},
			Capacity:     asrcontract.ProviderCapacity{MaxConcurrentJobs: 1, AvailableSlots: 1},
		})
	})
	mux.HandleFunc("/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		var req jobCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeExampleError(t, w, http.StatusBadRequest, asrcontract.CodeInvalidRequest, "invalid job request")
			return
		}
		if req.Operation != asrcontract.OperationTranscription || req.Transcription == nil {
			writeExampleError(t, w, http.StatusBadRequest, asrcontract.CodeUnsupportedOperation, "only transcription is implemented")
			return
		}
		writeExampleJSON(t, w, jobCreateResponse{JobID: "example-job"})
	})
	mux.HandleFunc("/v1/jobs/example-job", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, jobStatusResponse{
			JobID:  "example-job",
			Status: "completed",
			Transcription: &asrcontract.TranscriptionResult{
				Model:    "example-transcriber",
				Language: "en",
				Text:     "hello from example provider",
				Segments: []asrcontract.TranscriptSegment{{ID: "seg_000001", Start: 0, End: 1, Text: "hello from example provider"}},
				Words:    []asrcontract.TranscriptWord{{Start: 0, End: 0.5, Word: "hello"}},
			},
		})
	})
	mux.HandleFunc("/v1/jobs/example-job/events", func(w http.ResponseWriter, r *http.Request) {
		writeExampleJSON(t, w, eventsResponse{Events: []asrcontract.ProviderProgress{{
			Stage:     asrcontract.StageCompleted,
			Progress:  floatPtr(1),
			Operation: asrcontract.OperationTranscription,
			Model:     "example-transcriber",
			Timestamp: time.Now().UTC(),
		}}})
	})
	return httptest.NewServer(mux)
}

func exampleModelCard() asrcontract.ModelCard {
	return asrcontract.ModelCard{
		ID:          "example-transcriber",
		DisplayName: "Example Transcriber",
		Provider:    "example",
		ModelType:   "example",
		Installed:   true,
		Loaded:      true,
		Default:     true,
		Tasks:       []asrcontract.Task{asrcontract.TaskTranscribe},
		Languages:   []string{"en"},
		Capabilities: asrcontract.Capabilities{
			Transcription:     true,
			WordTimestamps:    true,
			SegmentTimestamps: true,
		},
		LanguageSupport: &asrcontract.LanguageSupport{
			Languages: []string{"en"},
			Mode:      "fixed",
		},
		Chunking: &asrcontract.ChunkingCapabilities{
			SupportsEngineChunking:   true,
			SupportsProviderChunking: false,
			PreferredMode:            "fixed",
			RecommendedChunkSeconds:  floatPtr(30),
			MaxChunkSeconds:          floatPtr(120),
			SupportsBatching:         false,
			RecommendedBatchSize:     intPtr(1),
			MaxBatchSize:             intPtr(1),
		},
		ParameterSchema: asrcontract.ParameterSchema{
			{
				Key:            asrcontract.CommonParameterRuntimeNumThreads,
				Label:          "Threads",
				Type:           asrcontract.ParameterTypeInteger,
				Default:        float64(1),
				Min:            floatPtr(1),
				Max:            floatPtr(8),
				Step:           floatPtr(1),
				Scope:          asrcontract.ParameterScopeRuntime,
				RequiresReload: true,
			},
			{
				Key:     asrcontract.CommonParameterOutputWordTimestamps,
				Label:   "Word timestamps",
				Type:    asrcontract.ParameterTypeBoolean,
				Default: true,
				Scope:   asrcontract.ParameterScopeOutput,
			},
		},
	}
}

func writeExampleJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func writeExampleError(t *testing.T, w http.ResponseWriter, status int, code asrcontract.ErrorCode, message string) {
	t.Helper()
	w.WriteHeader(status)
	writeExampleJSON(t, w, map[string]asrcontract.ProviderError{
		"error": {Code: code, Message: message},
	})
}

func floatPtr(v float64) *float64 { return &v }

func intPtr(v int) *int { return &v }
