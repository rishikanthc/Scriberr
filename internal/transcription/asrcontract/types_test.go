package asrcontract

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestModelCardSupportsCapabilities(t *testing.T) {
	card := ModelCard{
		ID:       "parakeet-v3",
		Provider: "local-sherpa",
		Capabilities: Capabilities{
			Transcription:     true,
			WordTimestamps:    true,
			SegmentTimestamps: true,
		},
	}

	if !card.Supports(CapabilityTranscription, CapabilityWordTimestamps) {
		t.Fatal("expected card to support transcription and word timestamps")
	}
	if card.Supports(CapabilityDiarization) {
		t.Fatal("did not expect card to support diarization")
	}
	if card.Supports(Capability("custom_extension")) {
		t.Fatal("unknown capabilities should not be supported unless present in extensions")
	}

	card.Capabilities.Extensions = map[string]bool{"custom_extension": true}
	if !card.Supports(Capability("custom_extension")) {
		t.Fatal("expected extension capability to be supported")
	}
}

func TestParameterSchemaValidationAndProfileValues(t *testing.T) {
	schema := ParameterSchema{
		{
			Key:            CommonParameterRuntimeNumThreads,
			Label:          "Threads",
			Type:           ParameterTypeInteger,
			Default:        float64(4),
			Min:            floatPtr(1),
			Max:            floatPtr(16),
			Step:           floatPtr(1),
			Scope:          ParameterScopeRuntime,
			RequiresReload: true,
		},
		{
			Key:     CommonParameterDecodingMethod,
			Label:   "Decoding",
			Type:    ParameterTypeEnum,
			Default: "greedy_search",
			Options: []ParameterOption{
				{Value: "greedy_search", Label: "Greedy"},
				{Value: "modified_beam_search", Label: "Beam"},
			},
			Scope: ParameterScopeDecoding,
		},
		{
			Key:      "sherpa.whisper.tail_paddings",
			Label:    "Tail paddings",
			Type:     ParameterTypeInteger,
			Default:  float64(-1),
			Min:      floatPtr(-1),
			Max:      floatPtr(16),
			Scope:    ParameterScopeDecoding,
			Advanced: true,
		},
	}

	if err := ValidateParameterSchema(schema); err != nil {
		t.Fatalf("ValidateParameterSchema returned error: %v", err)
	}
	values, err := ValidateParameterValues(schema, map[string]any{
		CommonParameterRuntimeNumThreads: float64(8),
		CommonParameterDecodingMethod:    "modified_beam_search",
		"sherpa.whisper.tail_paddings":   float64(2),
	})
	if err != nil {
		t.Fatalf("ValidateParameterValues returned error: %v", err)
	}
	if values[CommonParameterRuntimeNumThreads] != int64(8) {
		t.Fatalf("integer parameter was not normalized: %#v", values)
	}

	if err := ValidateParameterSchema(ParameterSchema{{Key: "tail_paddings", Type: ParameterTypeInteger, Scope: ParameterScopeDecoding}}); err == nil {
		t.Fatal("expected unnamespaced provider-specific key to fail schema validation")
	}
	if _, err := ValidateParameterValues(schema, map[string]any{CommonParameterRuntimeNumThreads: float64(99)}); err == nil {
		t.Fatal("expected numeric bound violation")
	} else {
		var parameterErr *ParameterValueError
		if !errors.As(err, &parameterErr) || parameterErr.Parameter != CommonParameterRuntimeNumThreads {
			t.Fatalf("expected ParameterValueError for %q, got %T %[2]v", CommonParameterRuntimeNumThreads, err)
		}
	}
	if _, err := ValidateParameterValues(schema, map[string]any{CommonParameterDecodingMethod: "unknown"}); err == nil {
		t.Fatal("expected enum value violation")
	}
	if _, err := ValidateParameterValues(schema, map[string]any{"unknown.option": true}); err == nil {
		t.Fatal("expected unknown parameter rejection")
	}
}

func TestValidateParameterValuesRejectsChangedReadOnlyParameter(t *testing.T) {
	schema := ParameterSchema{{
		Key:      "sherpa.model_type",
		Label:    "Sherpa model type",
		Type:     ParameterTypeString,
		Default:  "nemo_transducer",
		Scope:    ParameterScopeModel,
		ReadOnly: true,
	}}

	if _, err := ValidateParameterValues(schema, nil); err != nil {
		t.Fatalf("omitted read-only value should validate: %v", err)
	}
	values, err := ValidateParameterValues(schema, map[string]any{"sherpa.model_type": "nemo_transducer"})
	if err != nil {
		t.Fatalf("default read-only value should validate: %v", err)
	}
	if values["sherpa.model_type"] != "nemo_transducer" {
		t.Fatalf("read-only default was not preserved: %#v", values)
	}
	if _, err := ValidateParameterValues(schema, map[string]any{"sherpa.model_type": "whisper"}); err == nil {
		t.Fatal("expected changed read-only value to fail")
	}
}

func TestProviderErrorClassification(t *testing.T) {
	err := NewProviderError(CodeProviderBusy, "provider is busy", true)

	if !IsCode(err, CodeProviderBusy) {
		t.Fatal("expected provider busy code")
	}
	if !Retryable(err) {
		t.Fatal("expected retryable provider error")
	}
	if IsCode(err, CodeModelNotInstalled) {
		t.Fatal("did not expect model-not-installed code")
	}
	if Retryable(errors.New("plain error")) {
		t.Fatal("plain errors should not be classified retryable")
	}
}

func TestContractJSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	payload := struct {
		Provider ProviderInfo        `json:"provider"`
		Model    ModelCard           `json:"model"`
		Status   ProviderStatus      `json:"status"`
		Progress ProviderProgress    `json:"progress"`
		Result   TranscriptionResult `json:"result"`
	}{
		Provider: ProviderInfo{
			ContractVersion: ContractVersionV1,
			Provider: ProviderIdentity{
				ID:      "local-sherpa",
				Name:    "Sherpa ONNX",
				Version: "2.0.0",
				Vendor:  "scriberr",
			},
			Runtime: RuntimeInfo{
				DeviceBackends:       []string{"cpu", "cuda"},
				ActiveBackend:        "cpu",
				SupportsConcurrent:   false,
				MaxConcurrentJobs:    1,
				ProviderCapabilities: []Capability{CapabilityTranscription},
			},
			AudioInput: AudioInputSpec{
				RequiredSampleRate: 16000,
				RequiredChannels:   1,
				Formats:            []string{"wav"},
				PathMode:           PathModeMountedFile,
			},
		},
		Model: ModelCard{
			ID:          "whisper-base",
			DisplayName: "Whisper Base",
			Provider:    "local-sherpa",
			ModelType:   "whisper",
			Version:     "base",
			Installed:   true,
			Default:     true,
			Tasks:       []Task{TaskTranscribe},
			Languages:   []string{"en"},
			Capabilities: Capabilities{
				Transcription:     true,
				WordTimestamps:    true,
				SegmentTimestamps: true,
			},
			Limits: ModelLimits{RecommendedChunkSec: floatPtr(30)},
			ResourceRequirements: ResourceRequirements{
				Backends: []string{"cpu"},
			},
			ParameterSchema: ParameterSchema{{
				Key:     CommonParameterDecodingMethod,
				Label:   "Decoding",
				Type:    ParameterTypeEnum,
				Default: "greedy_search",
				Options: []ParameterOption{
					{Value: "greedy_search", Label: "Greedy"},
					{Value: "modified_beam_search", Label: "Beam"},
				},
				Scope: ParameterScopeDecoding,
			}},
		},
		Status: ProviderStatus{
			State: ProviderStateBusy,
			ActiveJob: &ActiveJob{
				ID:        "job_123",
				Operation: OperationTranscription,
				Model:     "whisper-base",
				Stage:     StageRunning,
				Progress:  floatPtr(0.5),
			},
			LoadedModels: []LoadedModel{{ID: "whisper-base", LoadedAt: &now, MemoryMB: intPtr(512)}},
			Capacity: ProviderCapacity{
				MaxConcurrentJobs: 1,
				AvailableSlots:    0,
			},
		},
		Progress: ProviderProgress{
			Stage:     StageLoadingModel,
			Progress:  floatPtr(0.2),
			Message:   "loading",
			Operation: OperationTranscription,
			Model:     "whisper-base",
			Timestamp: now,
		},
		Result: TranscriptionResult{
			Model:    "whisper-base",
			Language: "en",
			Text:     "hello world",
			Segments: []TranscriptSegment{{ID: "seg_0001", Start: 0, End: 1.2, Text: "hello world"}},
			Words:    []TranscriptWord{{Start: 0, End: 0.5, Word: "hello", Confidence: floatPtr(0.9)}},
			Metadata: map[string]any{"processing_time_sec": 1.25},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal contract payload: %v", err)
	}

	var decoded struct {
		Provider ProviderInfo        `json:"provider"`
		Model    ModelCard           `json:"model"`
		Status   ProviderStatus      `json:"status"`
		Progress ProviderProgress    `json:"progress"`
		Result   TranscriptionResult `json:"result"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal contract payload: %v", err)
	}
	if decoded.Provider.ContractVersion != ContractVersionV1 {
		t.Fatalf("contract version mismatch: %q", decoded.Provider.ContractVersion)
	}
	if !decoded.Model.Supports(CapabilityTranscription, CapabilityWordTimestamps) {
		t.Fatal("decoded model lost capabilities")
	}
	if decoded.Status.State != ProviderStateBusy || decoded.Status.ActiveJob == nil {
		t.Fatalf("decoded status mismatch: %+v", decoded.Status)
	}
	if decoded.Progress.Stage != StageLoadingModel {
		t.Fatalf("decoded progress mismatch: %+v", decoded.Progress)
	}
	if len(decoded.Result.Words) != 1 || decoded.Result.Words[0].Word != "hello" {
		t.Fatalf("decoded result mismatch: %+v", decoded.Result)
	}
}

func floatPtr(v float64) *float64 { return &v }

func intPtr(v int) *int { return &v }
