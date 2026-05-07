package engineprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"scriberr/internal/transcription/asrcontract"

	speechengine "scriberr-engine/speech/engine"
	speechmodels "scriberr-engine/speech/models"
	speechproviders "scriberr-engine/speech/providers"
	"scriberr-engine/speech/providers/sherpa/catalog"
	engresults "scriberr-engine/speech/results"
	"scriberr-engine/speech/runtime"
)

type fakeSpeechEngine struct {
	transcriptionReq  speechengine.TranscriptionRequest
	diarizationReq    speechengine.DiarizationRequest
	transcriptionOut  *speechengine.TranscriptionResult
	diarizationOut    *speechengine.DiarizationResult
	err               error
	info              *speechengine.ProviderInfo
	models            []speechproviders.ModelDescriptor
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
			ProviderCapabilities: []speechproviders.TaskKind{speechproviders.TaskTranscription, speechproviders.TaskDiarization},
		},
		AudioInput: speechengine.AudioInputSpec{
			RequiredSampleRate: 16000,
			RequiredChannels:   1,
			Formats:            []string{"wav"},
			PathMode:           speechengine.PathModeMountedFile,
		},
	}, nil
}

func (e *fakeSpeechEngine) Models(ctx context.Context) ([]speechproviders.ModelDescriptor, error) {
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
		JobID:     "job-1",
		UserID:    7,
		AudioPath: "/tmp/audio.wav",
		Progress:  progress,
		ModelID:   "whisper-tiny",
		Parameters: map[string]any{
			"language": "en",
			"task":     "translate",
			asrcontract.CommonParameterRuntimeNumThreads:      2,
			asrcontract.CommonParameterChunkingMode:           "vad",
			asrcontract.CommonParameterChunkingChunkSeconds:   float64(25),
			asrcontract.CommonParameterOutputWordTimestamps:   true,
			asrcontract.CommonParameterOutputTimestamps:       true,
			asrcontract.CommonParameterOutputTokenTimestamps:  true,
			asrcontract.CommonParameterChunkingOverlapSeconds: float64(0),
		},
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
	if fake.transcriptionReq.Parameters["language"] != "en" {
		t.Fatalf("language parameter = %#v", fake.transcriptionReq.Parameters["language"])
	}
	if fake.transcriptionReq.Parameters["task"] != "translate" {
		t.Fatalf("task parameter = %#v", fake.transcriptionReq.Parameters["task"])
	}
	if fake.transcriptionReq.Parameters[asrcontract.CommonParameterRuntimeNumThreads] != 2 {
		t.Fatalf("runtime parameter = %#v", fake.transcriptionReq.Parameters[asrcontract.CommonParameterRuntimeNumThreads])
	}
	if fake.transcriptionReq.Parameters[asrcontract.CommonParameterChunkingMode] != "vad" ||
		fake.transcriptionReq.Parameters[asrcontract.CommonParameterChunkingChunkSeconds] != float64(25) {
		t.Fatalf("unexpected chunking parameters: %#v", fake.transcriptionReq.Parameters)
	}
	fake.transcriptionReq.Progress.Report(context.Background(), speechengine.Progress{
		Stage:     speechengine.StageRunning,
		Operation: speechengine.OperationTranscription,
		Model:     "whisper-tiny",
	})
	if len(progress.events) != 1 || progress.events[0].Stage != asrcontract.Stage("running") {
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

func TestLocalProviderTranscribeUsesEnginePlanAndMetricsMetadata(t *testing.T) {
	fake := &fakeSpeechEngine{
		transcriptionOut: &speechengine.TranscriptionResult{
			Text: "hello parakeet",
			Words: []speechengine.TranscriptWord{
				{Text: "hello", StartSec: 0.1, EndSec: 0.4},
				{Text: "parakeet", StartSec: 0.5, EndSec: 1.1},
			},
			Segments: []speechengine.TranscriptSegment{
				{Text: "hello parakeet", StartSec: 0.1, EndSec: 1.1},
			},
			Metrics: engresults.Metrics{
				AudioDurationSec: 10,
				DecodeDuration:   2 * time.Second,
				ChunkCount:       1,
				BatchSize:        1,
				HypothesisWords:  2,
			},
			Plan: engresults.PlanSummary{
				ChunkingMode: "fixed",
				Task:         "transcribe",
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 4}, runtime.ProviderCPU, fake)

	result, err := provider.Transcribe(context.Background(), TranscriptionRequest{
		JobID:     "job-parakeet",
		AudioPath: "/provider/audio.wav",
		ModelID:   "parakeet-v3",
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}

	if fake.transcriptionReq.Parameters != nil {
		t.Fatalf("local provider synthesized parameters: %#v", fake.transcriptionReq.Parameters)
	}
	metrics, ok := result.Metadata["metrics"].(engresults.Metrics)
	if !ok {
		t.Fatalf("metadata missing metrics: %#v", result.Metadata)
	}
	if metrics.BatchSize != 1 || metrics.HypothesisWords != 2 {
		t.Fatalf("metadata metrics mismatch: %#v", metrics)
	}
	plan, ok := result.Metadata["plan"].(engresults.PlanSummary)
	if !ok || plan.ChunkingMode != "fixed" {
		t.Fatalf("metadata missing plan: %#v", result.Metadata)
	}
	if strings.Contains(fmt.Sprint(result.Metadata), "/provider/audio.wav") {
		t.Fatalf("metadata leaked audio path: %#v", result.Metadata)
	}
}

func TestLocalProviderTranscribePreservesExplicitParakeetVAD(t *testing.T) {
	fake := &fakeSpeechEngine{transcriptionOut: &speechengine.TranscriptionResult{Text: "hello"}}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 4}, runtime.ProviderCPU, fake)

	_, err := provider.Transcribe(context.Background(), TranscriptionRequest{
		ModelID: "parakeet-v3",
		Parameters: map[string]any{
			asrcontract.CommonParameterChunkingMode:         "vad",
			asrcontract.CommonParameterChunkingChunkSeconds: float64(12),
			asrcontract.CommonParameterBatchingBatchSize:    1,
		},
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if fake.transcriptionReq.Parameters[asrcontract.CommonParameterChunkingMode] != "vad" {
		t.Fatalf("chunking parameter = %#v, want vad", fake.transcriptionReq.Parameters[asrcontract.CommonParameterChunkingMode])
	}
	if fake.transcriptionReq.Parameters[asrcontract.CommonParameterChunkingChunkSeconds] != float64(12) {
		t.Fatalf("chunk seconds parameter = %#v, want 12", fake.transcriptionReq.Parameters[asrcontract.CommonParameterChunkingChunkSeconds])
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
	if fake.transcriptionReq.Parameters != nil {
		t.Fatalf("local provider synthesized parameters: %#v", fake.transcriptionReq.Parameters)
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
			SpeakerSegments: []speechengine.SpeakerSegment{
				{SpeakerID: "SPEAKER_00", StartSec: 0, EndSec: 1.5},
				{SpeakerID: "SPEAKER_12", StartSec: 1.6, EndSec: 3.2},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 3}, runtime.ProviderCPU, fake)

	result, err := provider.Diarize(context.Background(), DiarizationRequest{
		JobID:     "job-2",
		AudioPath: "/tmp/audio.wav",
		ModelID:   "diarization-default",
		Parameters: map[string]any{
			"num_speakers": 2,
		},
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
	if fake.diarizationReq.Parameters["num_speakers"] != 2 {
		t.Fatalf("num_speakers parameter = %#v", fake.diarizationReq.Parameters["num_speakers"])
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

func TestLocalProviderCapabilitiesUseEngineDescriptors(t *testing.T) {
	fake := &fakeSpeechEngine{models: []speechproviders.ModelDescriptor{
		descriptorForModelWith(t, catalog.ModelWhisperBase, func(desc *speechproviders.ModelDescriptor) {
			desc.Installed = true
			desc.Default = true
		}),
		descriptorForModelWith(t, catalog.ModelDiarizationDefault, func(desc *speechproviders.ModelDescriptor) {
			desc.Default = true
		}),
	}}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake)
	registry, err := NewRegistry("local", provider)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	capabilities, err := registry.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if len(capabilities) != 2 {
		t.Fatalf("capabilities length = %d", len(capabilities))
	}
	if !capabilities[0].Installed || !capabilities[0].Default {
		t.Fatalf("whisper-base capability missing installed/default: %#v", capabilities[0])
	}
	if strings.Join(capabilities[0].Capabilities, ",") != "transcription,translation,word_timestamps,segment_timestamps,token_timestamps" {
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
		models: []speechproviders.ModelDescriptor{
			descriptorForModelWith(t, catalog.ModelWhisperBase, func(desc *speechproviders.ModelDescriptor) {
				desc.Installed = true
				desc.Loaded = true
				desc.Default = true
			}),
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
		models: []speechproviders.ModelDescriptor{
			descriptorForModelWith(t, catalog.ModelWhisperBase, func(desc *speechproviders.ModelDescriptor) {
				desc.Installed = true
			}),
			descriptorForModelWith(t, catalog.ModelParakeetV3, func(desc *speechproviders.ModelDescriptor) {
				desc.Installed = true
			}),
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
	requireParameter(t, whisper.ParameterSchema, "sherpa.whisper.enable_token_timestamps")
	if hasParameter(parakeet.ParameterSchema, "sherpa.whisper.language") {
		t.Fatalf("parakeet descriptor should not expose whisper language parameter: %#v", parakeet.ParameterSchema)
	}
	requireReloadParameter(t, parakeet.ParameterSchema, "sherpa.model_type")
	requireReloadParameter(t, parakeet.ParameterSchema, "runtime.provider")
	requireReloadParameter(t, parakeet.ParameterSchema, asrcontract.CommonParameterRuntimeNumThreads)

	if got := parakeet.RecommendedDefaults[asrcontract.CommonParameterChunkingMode]; got != "fixed" {
		t.Fatalf("parakeet chunking default = %#v, want fixed", got)
	}
	if got := parakeet.RecommendedDefaults[asrcontract.CommonParameterRuntimeNumThreads]; got != float64(4) {
		t.Fatalf("parakeet threads default = %#v, want 4", got)
	}
	if got := parakeet.RecommendedDefaults[asrcontract.CommonParameterBatchingBatchSize]; got != float64(1) {
		t.Fatalf("parakeet batch default = %#v, want 1", got)
	}
	if parakeet.Chunking == nil || parakeet.Chunking.RecommendedChunkSeconds == nil || *parakeet.Chunking.RecommendedChunkSeconds != 30 {
		t.Fatalf("parakeet chunking metadata missing fixed 30s recommendation: %#v", parakeet.Chunking)
	}
	requireArtifactRequirement(t, parakeet, "encoder")
	requireArtifactRequirement(t, parakeet, "decoder")
	requireArtifactRequirement(t, parakeet, "joiner")
	requireArtifactRequirement(t, parakeet, "tokens")

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
		models: []speechproviders.ModelDescriptor{
			descriptorForModel(t, catalog.ModelWhisperBase),
			descriptorForModel(t, catalog.ModelParakeetV3),
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

func requireReloadParameter(t *testing.T, schema asrcontract.ParameterSchema, key string) {
	t.Helper()
	for _, parameter := range schema {
		if parameter.Key == key {
			if !parameter.RequiresReload {
				t.Fatalf("parameter %q should require reload: %#v", key, parameter)
			}
			return
		}
	}
	t.Fatalf("parameter %q not found in %#v", key, schema)
}

func requireArtifactRequirement(t *testing.T, model asrcontract.ModelCard, requirement string) {
	t.Helper()
	raw, ok := model.Extensions["artifacts"]
	if !ok {
		t.Fatalf("model %q missing artifact requirements", model.ID)
	}
	items, ok := raw.([]map[string]any)
	if !ok {
		t.Fatalf("artifact requirements should be []map[string]any: %#v", raw)
	}
	for _, item := range items {
		if item["key"] == requirement {
			return
		}
	}
	t.Fatalf("artifact requirement %q not found in %#v", requirement, items)
}

func descriptorForModel(t *testing.T, id speechmodels.ModelID) speechmodels.Descriptor {
	t.Helper()
	descriptor, ok := catalog.DefaultModelRegistry().ResolveDescriptor(string(id))
	if !ok {
		t.Fatalf("descriptor %q not found", id)
	}
	return descriptor
}

func descriptorForModelWith(t *testing.T, id speechmodels.ModelID, edit func(*speechproviders.ModelDescriptor)) speechmodels.Descriptor {
	t.Helper()
	descriptor := descriptorForModel(t, id)
	edit(&descriptor)
	return descriptor
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
