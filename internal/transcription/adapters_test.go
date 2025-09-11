package transcription

import (
	"context"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/registry"
)

func TestModelRegistry(t *testing.T) {
	// Get the global registry
	reg := registry.GetRegistry()

	// Test that models are auto-registered
	transcriptionModels := reg.GetTranscriptionModels()
	if len(transcriptionModels) == 0 {
		t.Error("No transcription models registered")
	}

	diarizationModels := reg.GetDiarizationModels()
	if len(diarizationModels) == 0 {
		t.Error("No diarization models registered")
	}

	t.Logf("Registered transcription models: %v", transcriptionModels)
	t.Logf("Registered diarization models: %v", diarizationModels)
}

func TestWhisperXAdapter(t *testing.T) {
	reg := registry.GetRegistry()

	// Get WhisperX adapter
	adapter, err := reg.GetTranscriptionAdapter("whisperx")
	if err != nil {
		t.Fatalf("Failed to get WhisperX adapter: %v", err)
	}

	// Test capabilities
	capabilities := adapter.GetCapabilities()
	if capabilities.ModelID != "whisperx" {
		t.Errorf("Expected model ID 'whisperx', got '%s'", capabilities.ModelID)
	}

	if capabilities.ModelFamily != "whisper" {
		t.Errorf("Expected model family 'whisper', got '%s'", capabilities.ModelFamily)
	}

	// Test parameter schema
	schema := adapter.GetParameterSchema()
	if len(schema) == 0 {
		t.Error("No parameter schema returned")
	}

	// Test parameter validation
	validParams := map[string]interface{}{
		"model":      "small",
		"device":     "cpu",
		"batch_size": 8,
		"diarize":    false,
	}

	if err := adapter.ValidateParameters(validParams); err != nil {
		t.Errorf("Valid parameters failed validation: %v", err)
	}

	// Test invalid parameters
	invalidParams := map[string]interface{}{
		"model":      "invalid_model",
		"batch_size": -1,
	}

	if err := adapter.ValidateParameters(invalidParams); err == nil {
		t.Error("Invalid parameters should have failed validation")
	}
}

func TestParakeetAdapter(t *testing.T) {
	reg := registry.GetRegistry()

	// Get Parakeet adapter
	adapter, err := reg.GetTranscriptionAdapter("parakeet")
	if err != nil {
		t.Fatalf("Failed to get Parakeet adapter: %v", err)
	}

	// Test capabilities
	capabilities := adapter.GetCapabilities()
	if capabilities.ModelFamily != "nvidia_parakeet" {
		t.Errorf("Expected model family 'nvidia_parakeet', got '%s'", capabilities.ModelFamily)
	}

	// Test that it only supports English
	if len(capabilities.SupportedLanguages) != 1 || capabilities.SupportedLanguages[0] != "en" {
		t.Errorf("Expected only English support, got: %v", capabilities.SupportedLanguages)
	}

	// Test parameter validation with context settings
	validParams := map[string]interface{}{
		"timestamps":    true,
		"context_left":  512,
		"context_right": 512,
	}

	if err := adapter.ValidateParameters(validParams); err != nil {
		t.Errorf("Valid parameters failed validation: %v", err)
	}
}

func TestCanaryAdapter(t *testing.T) {
	reg := registry.GetRegistry()

	// Get Canary adapter
	adapter, err := reg.GetTranscriptionAdapter("canary")
	if err != nil {
		t.Fatalf("Failed to get Canary adapter: %v", err)
	}

	// Test capabilities
	capabilities := adapter.GetCapabilities()
	if capabilities.ModelFamily != "nvidia_canary" {
		t.Errorf("Expected model family 'nvidia_canary', got '%s'", capabilities.ModelFamily)
	}

	// Test that it supports multiple languages
	if len(capabilities.SupportedLanguages) <= 1 {
		t.Errorf("Expected multiple language support, got: %v", capabilities.SupportedLanguages)
	}

	// Test translation features
	if !capabilities.Features["translation"] {
		t.Error("Expected translation feature to be supported")
	}

	// Test parameter validation with language settings
	validParams := map[string]interface{}{
		"source_lang": "en",
		"target_lang": "es",
		"task":        "translate",
	}

	if err := adapter.ValidateParameters(validParams); err != nil {
		t.Errorf("Valid parameters failed validation: %v", err)
	}
}

func TestPyAnnoteAdapter(t *testing.T) {
	reg := registry.GetRegistry()

	// Get PyAnnote adapter
	adapter, err := reg.GetDiarizationAdapter("pyannote")
	if err != nil {
		t.Fatalf("Failed to get PyAnnote adapter: %v", err)
	}

	// Test capabilities
	capabilities := adapter.GetCapabilities()
	if capabilities.ModelFamily != "pyannote" {
		t.Errorf("Expected model family 'pyannote', got '%s'", capabilities.ModelFamily)
	}

	// Test speaker constraints
	maxSpeakers := adapter.GetMaxSpeakers()
	if maxSpeakers <= 0 {
		t.Errorf("Expected positive max speakers, got: %d", maxSpeakers)
	}

	minSpeakers := adapter.GetMinSpeakers()
	if minSpeakers <= 0 {
		t.Errorf("Expected positive min speakers, got: %d", minSpeakers)
	}

	// Test parameter validation - should require HF token
	paramsWithoutToken := map[string]interface{}{
		"min_speakers": 2,
		"max_speakers": 4,
	}

	if err := adapter.ValidateParameters(paramsWithoutToken); err == nil {
		t.Error("Parameters without HF token should fail validation")
	}

	// Test with token
	paramsWithToken := map[string]interface{}{
		"hf_token":     "dummy_token",
		"min_speakers": 2,
		"max_speakers": 4,
	}

	if err := adapter.ValidateParameters(paramsWithToken); err != nil {
		t.Errorf("Valid parameters with token failed validation: %v", err)
	}
}

func TestSortformerAdapter(t *testing.T) {
	reg := registry.GetRegistry()

	// Get Sortformer adapter
	adapter, err := reg.GetDiarizationAdapter("sortformer")
	if err != nil {
		t.Fatalf("Failed to get Sortformer adapter: %v", err)
	}

	// Test capabilities
	capabilities := adapter.GetCapabilities()
	if capabilities.ModelFamily != "nvidia_sortformer" {
		t.Errorf("Expected model family 'nvidia_sortformer', got '%s'", capabilities.ModelFamily)
	}

	// Test that it doesn't require authentication
	if capabilities.Metadata["no_auth"] != "true" {
		t.Error("Expected Sortformer to not require authentication")
	}

	// Test parameter validation - should not require HF token
	validParams := map[string]interface{}{
		"max_speakers": 4,
		"batch_size":   1,
	}

	if err := adapter.ValidateParameters(validParams); err != nil {
		t.Errorf("Valid parameters failed validation: %v", err)
	}
}

func TestModelSelection(t *testing.T) {
	reg := registry.GetRegistry()

	// Test selecting best transcription model
	requirements := interfaces.ModelRequirements{
		Language: "en",
		Features: []string{"timestamps", "diarization"},
		Quality:  "good",
	}

	modelID, err := reg.SelectBestTranscriptionModel(requirements)
	if err != nil {
		t.Fatalf("Failed to select transcription model: %v", err)
	}

	t.Logf("Selected transcription model: %s", modelID)

	// Test selecting best diarization model
	diarRequirements := interfaces.ModelRequirements{
		Language: "en",
		Features: []string{"speaker_detection"},
	}

	diarModelID, err := reg.SelectBestDiarizationModel(diarRequirements)
	if err != nil {
		t.Fatalf("Failed to select diarization model: %v", err)
	}

	t.Logf("Selected diarization model: %s", diarModelID)
}

func TestUnifiedTranscriptionService(t *testing.T) {
	// Create unified service
	service := NewUnifiedTranscriptionService()

	// Test model discovery
	models := service.GetSupportedModels()
	if len(models) == 0 {
		t.Error("No models discovered by unified service")
	}

	t.Logf("Unified service discovered %d models", len(models))

	// Test model status check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := service.GetModelStatus(ctx)
	t.Logf("Model status: %+v", status)
}

func TestAudioInputCreation(t *testing.T) {
	service := NewUnifiedTranscriptionService()

	// Test creating audio input from a hypothetical file
	audioPath := "/tmp/test.wav"
	
	// This will fail since the file doesn't exist, but we can test the structure
	_, err := service.createAudioInput(audioPath)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParameterConversion(t *testing.T) {
	service := NewUnifiedTranscriptionService()

	// Test converting WhisperX parameters to generic map
	params := models.WhisperXParams{
		Model:       "small",
		Device:      "cpu",
		BatchSize:   8,
		Language:    stringPtr("en"),
		Task:        "transcribe",
		Diarize:     true,
		MinSpeakers: intPtr(2),
		MaxSpeakers: intPtr(4),
	}

	paramMap := service.parametersToMap(params)

	// Verify key parameters are present
	if paramMap["model"] != "small" {
		t.Errorf("Expected model 'small', got '%v'", paramMap["model"])
	}

	if paramMap["device"] != "cpu" {
		t.Errorf("Expected device 'cpu', got '%v'", paramMap["device"])
	}

	if paramMap["diarize"] != true {
		t.Errorf("Expected diarize true, got '%v'", paramMap["diarize"])
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// Benchmark tests
func BenchmarkModelRegistryLookup(b *testing.B) {
	reg := registry.GetRegistry()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reg.GetTranscriptionAdapter("whisperx")
		if err != nil {
			b.Fatalf("Failed to get adapter: %v", err)
		}
	}
}

func BenchmarkParameterValidation(b *testing.B) {
	reg := registry.GetRegistry()
	adapter, err := reg.GetTranscriptionAdapter("whisperx")
	if err != nil {
		b.Fatalf("Failed to get adapter: %v", err)
	}

	params := map[string]interface{}{
		"model":      "small",
		"device":     "cpu",
		"batch_size": 8,
		"diarize":    false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := adapter.ValidateParameters(params)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}