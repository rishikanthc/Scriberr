package engineprovider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"scriberr/internal/transcription/asrcontract"

	speechengine "scriberr-engine/speech/engine"
	"scriberr-engine/speech/runtime"
)

type fakeSpeechEngine struct {
	transcriptionReq  speechengine.TranscriptionRequest
	diarizationReq    speechengine.DiarizationRequest
	transcriptionOut  *speechengine.TranscriptionResult
	diarizationOut    *speechengine.DiarizationResult
	err               error
	info              *speechengine.ProviderInfo
	models            []speechengine.ModelCard
	status            *speechengine.ProviderStatus
	loaded            []speechengine.LoadedModel
	loadedRequested   string
	unloadedRequested string
	closed            bool
}

type captureProgressSink struct {
	events []asrcontract.ProviderProgress
}

func (s *captureProgressSink) Report(ctx context.Context, event asrcontract.ProviderProgress) {
	s.events = append(s.events, event)
}

func (e *fakeSpeechEngine) Inspect(ctx context.Context) (*speechengine.ProviderInfo, error) {
	if e.err != nil {
		return nil, e.err
	}
	if e.info != nil {
		return e.info, nil
	}
	return &speechengine.ProviderInfo{
		ContractVersion: speechengine.ContractVersionV1,
		Provider: speechengine.ProviderIdentity{
			ID:     "local",
			Name:   "Sherpa ONNX",
			Vendor: "scriberr",
		},
		Runtime: speechengine.RuntimeInfo{
			DeviceBackends:       []string{"cpu", "cuda"},
			ActiveBackend:        "cpu",
			SupportsConcurrent:   false,
			MaxConcurrentJobs:    1,
			ProviderCapabilities: []speechengine.Capability{speechengine.CapabilityTranscription, speechengine.CapabilityDiarization},
		},
		AudioInput: speechengine.AudioInputSpec{
			RequiredSampleRate: 16000,
			RequiredChannels:   1,
			Formats:            []string{"wav"},
			PathMode:           speechengine.PathModeMountedFile,
		},
	}, nil
}

func (e *fakeSpeechEngine) Models(ctx context.Context) ([]speechengine.ModelCard, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.models, nil
}

func (e *fakeSpeechEngine) Status(ctx context.Context) (*speechengine.ProviderStatus, error) {
	if e.err != nil {
		return nil, e.err
	}
	if e.status != nil {
		return e.status, nil
	}
	return &speechengine.ProviderStatus{
		State:        speechengine.ProviderStateIdle,
		LoadedModels: e.loaded,
		Capacity: speechengine.ProviderCapacity{
			MaxConcurrentJobs: 1,
			AvailableSlots:    1,
		},
	}, nil
}

func (e *fakeSpeechEngine) Transcribe(ctx context.Context, req speechengine.TranscriptionRequest) (*speechengine.TranscriptionResult, error) {
	e.transcriptionReq = req
	if e.err != nil {
		return nil, e.err
	}
	return e.transcriptionOut, nil
}

func (e *fakeSpeechEngine) Diarize(ctx context.Context, req speechengine.DiarizationRequest) (*speechengine.DiarizationResult, error) {
	e.diarizationReq = req
	if e.err != nil {
		return nil, e.err
	}
	return e.diarizationOut, nil
}

func (e *fakeSpeechEngine) Close() error {
	e.closed = true
	return nil
}

func (e *fakeSpeechEngine) LoadModel(ctx context.Context, modelID string) error {
	e.loadedRequested = modelID
	return e.err
}

func (e *fakeSpeechEngine) UnloadModel(modelID string) error {
	e.unloadedRequested = modelID
	return e.err
}

func (e *fakeSpeechEngine) LoadedModels() []speechengine.LoadedModel {
	return e.loaded
}

func TestLocalProviderTranscribeMapsRequestAndWords(t *testing.T) {
	fake := &fakeSpeechEngine{
		transcriptionOut: &speechengine.TranscriptionResult{
			Text:     "hello world",
			Language: "en",
			Words: []speechengine.TranscriptWord{
				{Text: "hello", StartSec: 0.1, EndSec: 0.4},
				{Text: "world", StartSec: 0.5, EndSec: 0.9},
			},
			Segments: []speechengine.TranscriptSegment{
				{Text: "hello world", StartSec: 0.1, EndSec: 0.9},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 4}, runtime.ProviderCPU, fake)
	progress := &captureProgressSink{}

	result, err := provider.Transcribe(context.Background(), TranscriptionRequest{
		JobID:            "job-1",
		UserID:           7,
		AudioPath:        "/tmp/audio.wav",
		Progress:         progress,
		ModelID:          "whisper-tiny",
		Language:         "en",
		Task:             "translate",
		Threads:          2,
		Chunking:         "vad",
		ChunkDurationSec: 25,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}

	if fake.transcriptionReq.ModelID != "whisper-tiny" {
		t.Fatalf("ModelID = %q", fake.transcriptionReq.ModelID)
	}
	if fake.transcriptionReq.RequestID != "job-1" {
		t.Fatalf("RequestID = %q", fake.transcriptionReq.RequestID)
	}
	if fake.transcriptionReq.Language != "en" {
		t.Fatalf("Language = %q", fake.transcriptionReq.Language)
	}
	if fake.transcriptionReq.Task != "translate" {
		t.Fatalf("Task = %q", fake.transcriptionReq.Task)
	}
	if fake.transcriptionReq.NumThreads != 2 {
		t.Fatalf("NumThreads = %d", fake.transcriptionReq.NumThreads)
	}
	if fake.transcriptionReq.Provider != runtime.ProviderCPU {
		t.Fatalf("Provider = %q", fake.transcriptionReq.Provider)
	}
	if fake.transcriptionReq.Chunking != "vad" || fake.transcriptionReq.ChunkDurationSec != 25 {
		t.Fatalf("unexpected chunking request: %#v", fake.transcriptionReq)
	}
	if fake.transcriptionReq.EnableTokenTimestamps == nil || !*fake.transcriptionReq.EnableTokenTimestamps {
		t.Fatalf("EnableTokenTimestamps was not forced on")
	}
	if fake.transcriptionReq.EnableSegmentTimestamps == nil || !*fake.transcriptionReq.EnableSegmentTimestamps {
		t.Fatalf("EnableSegmentTimestamps was not forced on")
	}
	fake.transcriptionReq.Progress.Report(context.Background(), speechengine.Progress{
		Stage:     speechengine.StageTranscribing,
		Operation: speechengine.OperationTranscription,
		Model:     "whisper-tiny",
	})
	if len(progress.events) != 1 || progress.events[0].Stage != asrcontract.StageTranscribing {
		t.Fatalf("progress was not bridged: %#v", progress.events)
	}
	if result.Text != "hello world" || result.Language != "en" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(result.Words) != 2 || result.Words[0].Word != "hello" || result.Words[1].End != 0.9 {
		t.Fatalf("unexpected words: %#v", result.Words)
	}
	if len(result.Segments) != 1 || result.Segments[0].Text != "hello world" || result.Segments[0].ID != "seg_0000" {
		t.Fatalf("unexpected segments: %#v", result.Segments)
	}
	if result.ModelID != "whisper-tiny" || result.EngineID != "local" {
		t.Fatalf("unexpected model/engine ids: %#v", result)
	}
}

func TestLocalProviderTranscribeDefaultsAndEmptyWords(t *testing.T) {
	fake := &fakeSpeechEngine{
		transcriptionOut: &speechengine.TranscriptionResult{Text: "text"},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 4}, runtime.ProviderCPU, fake)

	result, err := provider.Transcribe(context.Background(), TranscriptionRequest{})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if fake.transcriptionReq.ModelID != DefaultTranscriptionModel {
		t.Fatalf("ModelID = %q, want %q", fake.transcriptionReq.ModelID, DefaultTranscriptionModel)
	}
	if fake.transcriptionReq.Task != "transcribe" {
		t.Fatalf("Task = %q, want transcribe", fake.transcriptionReq.Task)
	}
	if fake.transcriptionReq.NumThreads != 4 {
		t.Fatalf("NumThreads = %d, want 4", fake.transcriptionReq.NumThreads)
	}
	if result.Words == nil {
		t.Fatalf("Words is nil, want empty array")
	}
}

func TestLocalProviderTranscribeRejectsNilEngineResult(t *testing.T) {
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, &fakeSpeechEngine{})

	_, err := provider.Transcribe(context.Background(), TranscriptionRequest{})
	if err == nil {
		t.Fatalf("Transcribe returned nil error")
	}
	if strings.Contains(err.Error(), "/") {
		t.Fatalf("error should be public-safe: %q", err.Error())
	}
}

func TestLocalProviderDiarizeMapsRequestAndSpeakers(t *testing.T) {
	fake := &fakeSpeechEngine{
		diarizationOut: &speechengine.DiarizationResult{
			Segments: []speechengine.DiarizationSegment{
				{Speaker: 0, Start: 0, End: 1.5},
				{Speaker: 12, Start: 1.6, End: 3.2},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 3}, runtime.ProviderCPU, fake)

	result, err := provider.Diarize(context.Background(), DiarizationRequest{
		JobID:       "job-2",
		AudioPath:   "/tmp/audio.wav",
		ModelID:     "diarization-default",
		NumSpeakers: 2,
	})
	if err != nil {
		t.Fatalf("Diarize returned error: %v", err)
	}
	if fake.diarizationReq.ModelID != "diarization-default" {
		t.Fatalf("ModelID = %q", fake.diarizationReq.ModelID)
	}
	if fake.diarizationReq.RequestID != "job-2" {
		t.Fatalf("RequestID = %q", fake.diarizationReq.RequestID)
	}
	if fake.diarizationReq.NumClusters != 2 {
		t.Fatalf("NumClusters = %d", fake.diarizationReq.NumClusters)
	}
	if fake.diarizationReq.NumThreads != 3 {
		t.Fatalf("NumThreads = %d", fake.diarizationReq.NumThreads)
	}
	if len(result.Segments) != 2 || result.Segments[0].Speaker != "SPEAKER_00" || result.Segments[1].Speaker != "SPEAKER_12" {
		t.Fatalf("unexpected segments: %#v", result.Segments)
	}
}

func TestLocalProviderDiarizeRejectsNilEngineResult(t *testing.T) {
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, &fakeSpeechEngine{})

	_, err := provider.Diarize(context.Background(), DiarizationRequest{})
	if err == nil {
		t.Fatalf("Diarize returned nil error")
	}
	if strings.Contains(err.Error(), "/") {
		t.Fatalf("error should be public-safe: %q", err.Error())
	}
}

func TestLocalProviderCapabilitiesUseEngineModelCards(t *testing.T) {
	fake := &fakeSpeechEngine{models: []speechengine.ModelCard{
		{
			ID:          "whisper-base",
			DisplayName: "Whisper Base",
			Provider:    "local",
			Family:      "whisper",
			Installed:   true,
			Default:     true,
			Tasks:       []speechengine.Task{speechengine.TaskTranscribe},
			Capabilities: speechengine.Capabilities{
				Transcription:  true,
				WordTimestamps: true,
			},
		},
		{
			ID:          "diarization-default",
			DisplayName: "Diarization",
			Provider:    "local",
			Family:      "pyannote",
			Default:     true,
			Capabilities: speechengine.Capabilities{
				Diarization: true,
			},
		},
	}}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake)

	capabilities, err := provider.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if len(capabilities) != 2 {
		t.Fatalf("capabilities length = %d", len(capabilities))
	}
	if !capabilities[0].Installed || !capabilities[0].Default {
		t.Fatalf("whisper-base capability missing installed/default: %#v", capabilities[0])
	}
	if strings.Join(capabilities[0].Capabilities, ",") != "transcription,word_timestamps" {
		t.Fatalf("whisper capabilities = %#v", capabilities[0].Capabilities)
	}
	if capabilities[1].Installed || !capabilities[1].Default {
		t.Fatalf("diarization capability installed/default mismatch: %#v", capabilities[1])
	}
	if strings.Join(capabilities[1].Capabilities, ",") != "diarization" {
		t.Fatalf("diarization capabilities = %#v", capabilities[1].Capabilities)
	}
}

func TestLocalProviderModelsStatusAndLifecycle(t *testing.T) {
	fake := &fakeSpeechEngine{
		loaded: []speechengine.LoadedModel{{ID: "whisper-base"}},
		models: []speechengine.ModelCard{
			{
				ID:          "whisper-base",
				DisplayName: "Whisper Base",
				Provider:    "local",
				Family:      "whisper",
				Version:     "whisper",
				Installed:   true,
				Loaded:      true,
				Default:     true,
				Tasks:       []speechengine.Task{speechengine.TaskTranscribe},
				Capabilities: speechengine.Capabilities{
					Transcription: true,
				},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake)

	info, err := provider.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if info.ContractVersion != asrcontract.ContractVersionV1 || info.Provider.ID != "local" {
		t.Fatalf("unexpected provider info: %#v", info)
	}

	models, err := provider.Models(context.Background())
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if len(models) != 1 || !models[0].Loaded || !models[0].Installed || !models[0].Supports(asrcontract.CapabilityTranscription) {
		t.Fatalf("unexpected model cards: %#v", models)
	}

	status, err := provider.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.State != asrcontract.ProviderStateIdle || len(status.LoadedModels) != 1 {
		t.Fatalf("unexpected status: %#v", status)
	}

	if err := provider.LoadModel(context.Background(), asrcontract.LoadModelRequest{Model: "whisper-base"}); err != nil {
		t.Fatalf("LoadModel returned error: %v", err)
	}
	if fake.loadedRequested != "whisper-base" {
		t.Fatalf("loaded model = %q", fake.loadedRequested)
	}
	if err := provider.UnloadModel(context.Background(), asrcontract.UnloadModelRequest{Model: "whisper-base"}); err != nil {
		t.Fatalf("UnloadModel returned error: %v", err)
	}
	if fake.unloadedRequested != "whisper-base" {
		t.Fatalf("unloaded model = %q", fake.unloadedRequested)
	}
}

func TestLocalProviderModelDescriptorsDistinguishWhisperAndParakeet(t *testing.T) {
	fake := &fakeSpeechEngine{
		models: []speechengine.ModelCard{
			{
				ID:          "whisper-base",
				DisplayName: "Whisper Base",
				Provider:    "local",
				Family:      "whisper",
				Version:     "base",
				Installed:   true,
				Tasks:       []speechengine.Task{speechengine.TaskTranscribe, speechengine.Task("translate")},
				Languages:   []string{"auto", "en", "es"},
				Capabilities: speechengine.Capabilities{
					Transcription:     true,
					WordTimestamps:    true,
					SegmentTimestamps: true,
					TokenTimestamps:   true,
					LanguageDetection: true,
				},
			},
			{
				ID:          "parakeet-v3",
				DisplayName: "Parakeet V3",
				Provider:    "local",
				Family:      "nemo_transducer",
				Version:     "v3",
				Installed:   true,
				Tasks:       []speechengine.Task{speechengine.TaskTranscribe},
				Languages:   []string{"en"},
				Capabilities: speechengine.Capabilities{
					Transcription:     true,
					WordTimestamps:    true,
					SegmentTimestamps: true,
				},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Provider: "cpu", Threads: 4, CacheDir: "/Users/zade/private/cache"}, runtime.ProviderCPU, fake)

	models, err := provider.Models(context.Background())
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	whisper := modelByID(t, models, "whisper-base")
	parakeet := modelByID(t, models, "parakeet-v3")

	requireParameter(t, whisper.ParameterSchema, "sherpa.whisper.language")
	requireParameter(t, whisper.ParameterSchema, "sherpa.whisper.task")
	requireParameter(t, whisper.ParameterSchema, "sherpa.whisper.tail_paddings")
	requireParameter(t, whisper.ParameterSchema, asrcontract.CommonParameterOutputTokenTimestamps)
	if hasParameter(parakeet.ParameterSchema, "sherpa.whisper.language") {
		t.Fatalf("parakeet descriptor should not expose whisper language parameter: %#v", parakeet.ParameterSchema)
	}
	requireParameter(t, parakeet.ParameterSchema, "sherpa.nemo_transducer.encoder")
	requireParameter(t, parakeet.ParameterSchema, "sherpa.nemo_transducer.decoder")
	requireParameter(t, parakeet.ParameterSchema, "sherpa.nemo_transducer.joiner")
	requireParameter(t, parakeet.ParameterSchema, "sherpa.tokens")

	if got := parakeet.RecommendedDefaults[asrcontract.CommonParameterChunkingMode]; got != "fixed" {
		t.Fatalf("parakeet chunking default = %#v, want fixed", got)
	}
	if got := parakeet.RecommendedDefaults[asrcontract.CommonParameterRuntimeNumThreads]; got != 4 {
		t.Fatalf("parakeet threads default = %#v, want 4", got)
	}
	if got := parakeet.RecommendedDefaults[asrcontract.CommonParameterBatchingBatchSize]; got != 1 {
		t.Fatalf("parakeet batch default = %#v, want 1", got)
	}
	if parakeet.Chunking == nil || parakeet.Chunking.RecommendedChunkSeconds == nil || *parakeet.Chunking.RecommendedChunkSeconds != 30 {
		t.Fatalf("parakeet chunking metadata missing fixed 30s recommendation: %#v", parakeet.Chunking)
	}

	data, err := json.Marshal(models)
	if err != nil {
		t.Fatalf("marshal model descriptors: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "/Users/") || strings.Contains(text, "private/cache") || strings.Contains(text, "CacheDir") {
		t.Fatalf("model descriptors leaked host/cache details: %s", text)
	}
}

func TestLocalProviderModelDescriptorParameterSchemasValidate(t *testing.T) {
	fake := &fakeSpeechEngine{
		models: []speechengine.ModelCard{
			{
				ID:     "whisper-base",
				Family: "whisper",
				Capabilities: speechengine.Capabilities{
					Transcription: true,
				},
			},
			{
				ID:     "parakeet-v3",
				Family: "nemo_transducer",
				Capabilities: speechengine.Capabilities{
					Transcription: true,
				},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Provider: "cpu", Threads: 4}, runtime.ProviderCPU, fake)

	models, err := provider.Models(context.Background())
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	for _, model := range models {
		if err := asrcontract.ValidateModelCard(model); err != nil {
			t.Fatalf("model %q descriptor did not validate: %v", model.ID, err)
		}
	}

	parakeet := modelByID(t, models, "parakeet-v3")
	_, err = asrcontract.ValidateParameterValues(parakeet.ParameterSchema, map[string]any{
		asrcontract.CommonParameterChunkingMode:         "fixed",
		asrcontract.CommonParameterChunkingChunkSeconds: float64(30),
		asrcontract.CommonParameterBatchingBatchSize:    float64(1),
	})
	if err != nil {
		t.Fatalf("parakeet measured defaults should validate: %v", err)
	}
	_, err = asrcontract.ValidateParameterValues(parakeet.ParameterSchema, map[string]any{"sherpa.whisper.language": "en"})
	if err == nil {
		t.Fatal("parakeet schema accepted whisper-specific parameter")
	}
}

func TestLocalProviderSanitizesErrors(t *testing.T) {
	fake := &fakeSpeechEngine{
		err: errors.New("load /Users/zade/Code/asr/Scriberr/data/uploads/audio.wav failed token=secret"),
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake)

	_, err := provider.Transcribe(context.Background(), TranscriptionRequest{})
	if err == nil {
		t.Fatalf("Transcribe returned nil error")
	}
	msg := err.Error()
	if strings.Contains(msg, "/Users/") || strings.Contains(msg, "secret") {
		t.Fatalf("error was not sanitized: %q", msg)
	}
	if !strings.Contains(msg, "[redacted-path]") || !strings.Contains(msg, "token=[redacted]") {
		t.Fatalf("error missing sanitized markers: %q", msg)
	}
}

func modelByID(t *testing.T, models []asrcontract.ModelCard, id string) asrcontract.ModelCard {
	t.Helper()
	for _, model := range models {
		if model.ID == id {
			return model
		}
	}
	t.Fatalf("model %q not found in %#v", id, models)
	return asrcontract.ModelCard{}
}

func requireParameter(t *testing.T, schema asrcontract.ParameterSchema, key string) {
	t.Helper()
	if !hasParameter(schema, key) {
		t.Fatalf("parameter %q not found in %#v", key, schema)
	}
}

func hasParameter(schema asrcontract.ParameterSchema, key string) bool {
	for _, parameter := range schema {
		if parameter.Key == key {
			return true
		}
	}
	return false
}

func TestLocalProviderCloseClosesEngine(t *testing.T) {
	fake := &fakeSpeechEngine{}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake)

	if err := provider.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !fake.closed {
		t.Fatalf("fake engine was not closed")
	}
}
