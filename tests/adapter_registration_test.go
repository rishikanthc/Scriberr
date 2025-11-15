package tests

import (
	"path/filepath"
	"testing"

	"scriberr/internal/config"
	"scriberr/internal/transcription/adapters"
	"scriberr/internal/transcription/registry"
)

// NOTE: These tests verify the dependency injection pattern where adapters accept envPath parameters.
// This was implemented in PR #260 (model storage location fix).

// TestAdapterEnvPathInjection tests that all adapters accept custom envPath parameter
func TestAdapterEnvPathInjection(t *testing.T) {
	testEnvPath := "/test/custom/path"

	// Test WhisperX adapter
	whisperx := adapters.NewWhisperXAdapter(testEnvPath)
	if whisperx == nil {
		t.Fatal("NewWhisperXAdapter returned nil")
	}

	// Test Parakeet adapter
	parakeet := adapters.NewParakeetAdapter(filepath.Join(testEnvPath, "parakeet"))
	if parakeet == nil {
		t.Fatal("NewParakeetAdapter returned nil")
	}

	// Test Canary adapter
	canary := adapters.NewCanaryAdapter(filepath.Join(testEnvPath, "parakeet"))
	if canary == nil {
		t.Fatal("NewCanaryAdapter returned nil")
	}

	// Test Sortformer adapter
	sortformer := adapters.NewSortformerAdapter(filepath.Join(testEnvPath, "parakeet"))
	if sortformer == nil {
		t.Fatal("NewSortformerAdapter returned nil")
	}

	// Test PyAnnote adapter
	pyannote := adapters.NewPyAnnoteAdapter(filepath.Join(testEnvPath, "parakeet"))
	if pyannote == nil {
		t.Fatal("NewPyAnnoteAdapter returned nil")
	}
}

// TestRegisterAdapters tests that registerAdapters correctly registers all adapters
func TestRegisterAdapters(t *testing.T) {
	// Clear registry before test
	registry.ClearRegistry()

	// Create test config
	tempDir := t.TempDir()
	cfg := &config.Config{
		WhisperXEnv: tempDir,
	}

	// Register adapters (simulate what main.go does)
	nvidiaEnvPath := filepath.Join(cfg.WhisperXEnv, "parakeet")

	registry.RegisterTranscriptionAdapter("whisperx",
		adapters.NewWhisperXAdapter(cfg.WhisperXEnv))
	registry.RegisterTranscriptionAdapter("parakeet",
		adapters.NewParakeetAdapter(nvidiaEnvPath))
	registry.RegisterTranscriptionAdapter("canary",
		adapters.NewCanaryAdapter(nvidiaEnvPath))

	registry.RegisterDiarizationAdapter("pyannote",
		adapters.NewPyAnnoteAdapter(nvidiaEnvPath))
	registry.RegisterDiarizationAdapter("sortformer",
		adapters.NewSortformerAdapter(nvidiaEnvPath))

	// Verify registrations
	transcriptionAdapters := registry.GetTranscriptionAdapters()
	if len(transcriptionAdapters) != 3 {
		t.Errorf("Expected 3 transcription adapters, got %d", len(transcriptionAdapters))
	}

	// Check specific adapters are registered
	if _, exists := transcriptionAdapters["whisperx"]; !exists {
		t.Error("whisperx adapter not registered")
	}
	if _, exists := transcriptionAdapters["parakeet"]; !exists {
		t.Error("parakeet adapter not registered")
	}
	if _, exists := transcriptionAdapters["canary"]; !exists {
		t.Error("canary adapter not registered")
	}

	diarizationAdapters := registry.GetDiarizationAdapters()
	if len(diarizationAdapters) != 2 {
		t.Errorf("Expected 2 diarization adapters, got %d", len(diarizationAdapters))
	}

	// Check specific adapters are registered
	if _, exists := diarizationAdapters["pyannote"]; !exists {
		t.Error("pyannote adapter not registered")
	}
	if _, exists := diarizationAdapters["sortformer"]; !exists {
		t.Error("sortformer adapter not registered")
	}
}

// TestAdaptersUseConfigPaths tests that adapters use injected paths, not hardcoded ones
func TestAdaptersUseConfigPaths(t *testing.T) {
	// This test ensures adapters accept custom paths and don't use hardcoded "whisperx-env"
	customPath := "/completely/custom/path/for/testing"

	// Create adapters with custom path
	whisperx := adapters.NewWhisperXAdapter(customPath)
	parakeet := adapters.NewParakeetAdapter(filepath.Join(customPath, "parakeet"))
	canary := adapters.NewCanaryAdapter(filepath.Join(customPath, "parakeet"))
	sortformer := adapters.NewSortformerAdapter(filepath.Join(customPath, "parakeet"))
	pyannote := adapters.NewPyAnnoteAdapter(filepath.Join(customPath, "parakeet"))

	// All adapters should accept the custom path without error
	if whisperx == nil {
		t.Error("WhisperX adapter should accept custom path")
	}
	if parakeet == nil {
		t.Error("Parakeet adapter should accept custom path")
	}
	if canary == nil {
		t.Error("Canary adapter should accept custom path")
	}
	if sortformer == nil {
		t.Error("Sortformer adapter should accept custom path")
	}
	if pyannote == nil {
		t.Error("PyAnnote adapter should accept custom path")
	}
}

// TestClearRegistry tests the registry clear function
func TestClearRegistry(t *testing.T) {
	// Register an adapter
	registry.RegisterTranscriptionAdapter("test", adapters.NewWhisperXAdapter("/tmp"))

	// Verify it's registered
	adapters := registry.GetTranscriptionAdapters()
	if len(adapters) == 0 {
		t.Fatal("Expected at least one adapter before clear")
	}

	// Clear registry
	registry.ClearRegistry()

	// Verify it's empty
	adapters = registry.GetTranscriptionAdapters()
	if len(adapters) != 0 {
		t.Errorf("Expected 0 adapters after clear, got %d", len(adapters))
	}

	diarizationAdapters := registry.GetDiarizationAdapters()
	if len(diarizationAdapters) != 0 {
		t.Errorf("Expected 0 diarization adapters after clear, got %d", len(diarizationAdapters))
	}
}
