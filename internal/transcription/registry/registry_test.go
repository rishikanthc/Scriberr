package registry

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"scriberr/internal/transcription/interfaces"
)

// stubAdapter is a minimal transcription adapter that counts PrepareEnvironment calls
type stubAdapter struct {
	modelID  string
	prepared int32
}

func (s *stubAdapter) GetCapabilities() interfaces.ModelCapabilities {
	return interfaces.ModelCapabilities{ModelID: s.modelID, ModelFamily: "stub"}
}

func (s *stubAdapter) GetParameterSchema() []interfaces.ParameterSchema { return nil }

func (s *stubAdapter) ValidateParameters(map[string]interface{}) error { return nil }

func (s *stubAdapter) PrepareEnvironment(context.Context) error {
	atomic.AddInt32(&s.prepared, 1)
	return nil
}

func (s *stubAdapter) GetModelPath() string { return "/tmp/" + s.modelID }

func (s *stubAdapter) IsReady(context.Context) bool { return true }

func (s *stubAdapter) GetEstimatedProcessingTime(interfaces.AudioInput) time.Duration {
	return time.Second
}

func (s *stubAdapter) Transcribe(context.Context, interfaces.AudioInput, map[string]interface{}, interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	return &interfaces.TranscriptResult{}, nil
}

func (s *stubAdapter) GetSupportedModels() []string { return []string{s.modelID} }

func (s *stubAdapter) prepareCount() int32 { return atomic.LoadInt32(&s.prepared) }

func TestInitializeModelsPreparesAllByDefault(t *testing.T) {
	ClearRegistry()
	t.Cleanup(ClearRegistry)

	enabled := &stubAdapter{modelID: "enabled"}
	disabled := &stubAdapter{modelID: "disabled"}
	RegisterTranscriptionAdapter("enabled", enabled)
	RegisterTranscriptionAdapter("disabled", disabled)

	if err := GetRegistry().InitializeModels(context.Background()); err != nil {
		t.Fatalf("InitializeModels returned error: %v", err)
	}

	if got := enabled.prepareCount(); got != 1 {
		t.Errorf("expected enabled model to be prepared once, got %d", got)
	}
	if got := disabled.prepareCount(); got != 1 {
		t.Errorf("expected all models to be prepared when no filter is set, got %d", got)
	}
}

func TestInitializeModelsSkipsDisabledModels(t *testing.T) {
	ClearRegistry()
	t.Cleanup(ClearRegistry)

	enabled := &stubAdapter{modelID: "enabled"}
	disabled := &stubAdapter{modelID: "disabled"}
	RegisterTranscriptionAdapter("enabled", enabled)
	RegisterTranscriptionAdapter("disabled", disabled)

	r := GetRegistry()
	r.SetEnabledModels([]string{"enabled"})

	if err := r.InitializeModels(context.Background()); err != nil {
		t.Fatalf("InitializeModels returned error: %v", err)
	}

	if got := enabled.prepareCount(); got != 1 {
		t.Errorf("expected enabled model to be prepared once, got %d", got)
	}
	if got := disabled.prepareCount(); got != 0 {
		t.Errorf("expected disabled model to be skipped, got %d prepare calls", got)
	}

	// Disabled adapters stay registered and resolvable
	if _, err := r.GetTranscriptionAdapter("disabled"); err != nil {
		t.Errorf("disabled model should still be registered: %v", err)
	}

	// ... and get prepared on demand when a job actually needs them
	if err := r.EnsureModelReady(context.Background(), "disabled"); err != nil {
		t.Fatalf("EnsureModelReady returned error: %v", err)
	}
	if got := disabled.prepareCount(); got != 1 {
		t.Errorf("expected disabled model to be prepared on demand, got %d", got)
	}

	// Repeated calls do not prepare again
	if err := r.EnsureModelReady(context.Background(), "disabled"); err != nil {
		t.Fatalf("EnsureModelReady returned error: %v", err)
	}
	if got := disabled.prepareCount(); got != 1 {
		t.Errorf("expected on-demand preparation to happen once, got %d", got)
	}

	// Enabled models are already prepared, so this is a no-op
	if err := r.EnsureModelReady(context.Background(), "enabled"); err != nil {
		t.Fatalf("EnsureModelReady returned error: %v", err)
	}
	if got := enabled.prepareCount(); got != 1 {
		t.Errorf("expected enabled model not to be re-prepared, got %d", got)
	}
}
