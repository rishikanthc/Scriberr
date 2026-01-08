package transcription

import (
	"context"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/transcription/adapters"
	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/registry"

	"github.com/stretchr/testify/mock"
)

// MockJobRepository is a mock implementation of JobRepository
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) Create(ctx context.Context, entity *models.TranscriptionJob) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockJobRepository) FindByID(ctx context.Context, id interface{}) (*models.TranscriptionJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) Update(ctx context.Context, entity *models.TranscriptionJob) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockJobRepository) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockJobRepository) List(ctx context.Context, offset, limit int) ([]models.TranscriptionJob, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]models.TranscriptionJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) FindWithAssociations(ctx context.Context, id string) (*models.TranscriptionJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionJob, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]models.TranscriptionJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) UpdateTranscript(ctx context.Context, jobID string, transcript string) error {
	args := m.Called(ctx, jobID, transcript)
	return args.Error(0)
}

func (m *MockJobRepository) CreateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockJobRepository) UpdateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockJobRepository) DeleteExecutionsByJobID(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) DeleteMultiTrackFilesByJobID(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) ListWithParams(ctx context.Context, offset, limit int, sortBy, sortOrder, searchQuery string, updatedAfter *time.Time) ([]models.TranscriptionJob, int64, error) {
	args := m.Called(ctx, offset, limit, sortBy, sortOrder, searchQuery, updatedAfter)
	return args.Get(0).([]models.TranscriptionJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) FindActiveTrackJobs(ctx context.Context, parentJobID string) ([]models.TranscriptionJob, error) {
	args := m.Called(ctx, parentJobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) FindLatestCompletedExecution(ctx context.Context, jobID string) (*models.TranscriptionJobExecution, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TranscriptionJobExecution), args.Error(1)
}

func (m *MockJobRepository) UpdateStatus(ctx context.Context, jobID string, status models.JobStatus) error {
	args := m.Called(ctx, jobID, status)
	return args.Error(0)
}

func (m *MockJobRepository) UpdateError(ctx context.Context, jobID string, errorMsg string) error {
	args := m.Called(ctx, jobID, errorMsg)
	return args.Error(0)
}

func (m *MockJobRepository) FindByStatus(ctx context.Context, status models.JobStatus) ([]models.TranscriptionJob, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) CountByStatus(ctx context.Context, status models.JobStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) UpdateSummary(ctx context.Context, jobID string, summary string) error {
	args := m.Called(ctx, jobID, summary)
	return args.Error(0)
}

// MockTranscriptionAdapter is a mock implementation of TranscriptionAdapter
type MockTranscriptionAdapter struct {
	mock.Mock
}

func (m *MockTranscriptionAdapter) GetCapabilities() interfaces.ModelCapabilities {
	return interfaces.ModelCapabilities{
		ModelID:     "mock-model",
		ModelFamily: "mock",
		Features:    map[string]bool{"timestamps": true},
	}
}

func (m *MockTranscriptionAdapter) GetParameterSchema() []interfaces.ParameterSchema {
	return []interfaces.ParameterSchema{}
}

func (m *MockTranscriptionAdapter) ValidateParameters(params map[string]interface{}) error {
	return nil
}

func (m *MockTranscriptionAdapter) PrepareEnvironment(ctx context.Context) error {
	return nil
}

func (m *MockTranscriptionAdapter) GetModelPath() string {
	return "/tmp/mock-model"
}

func (m *MockTranscriptionAdapter) IsReady(ctx context.Context) bool {
	return true
}

func (m *MockTranscriptionAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	return time.Second
}

func (m *MockTranscriptionAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	return &interfaces.TranscriptResult{
		Text: "mock transcript",
	}, nil
}

func (m *MockTranscriptionAdapter) GetSupportedModels() []string {
	return []string{"mock-model"}
}

func (m *MockTranscriptionAdapter) Diarize(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	return &interfaces.DiarizationResult{
		SpeakerCount: 1,
		Speakers:     []string{"SPEAKER_00"},
	}, nil
}

func (m *MockTranscriptionAdapter) GetMaxSpeakers() int {
	return 10
}

func (m *MockTranscriptionAdapter) GetMinSpeakers() int {
	return 1
}

func TestModelRegistry(t *testing.T) {
	// Register mock models
	registry.RegisterTranscriptionAdapter("mock-transcription", new(MockTranscriptionAdapter))
	registry.RegisterDiarizationAdapter("mock-diarization", new(MockTranscriptionAdapter))
	// We need a mock diarization adapter too, but for now let's just use transcription one if interface matches?
	// No, interfaces are different.
	// I'll create a MockDiarizationAdapter too.
	// Or just reuse MockTranscriptionAdapter if I implement DiarizationAdapter interface on it.
	// Let's implement DiarizationAdapter on MockTranscriptionAdapter.

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
	registry.RegisterTranscriptionAdapter("whisperx", adapters.NewWhisperXAdapter("/tmp/whisperx"))

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
	registry.RegisterTranscriptionAdapter("parakeet", adapters.NewParakeetAdapter("/tmp/parakeet"))

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
	registry.RegisterTranscriptionAdapter("canary", adapters.NewCanaryAdapter("/tmp/canary"))

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
	registry.RegisterDiarizationAdapter("pyannote", adapters.NewPyAnnoteAdapter("/tmp/pyannote"))

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

	// Test parameter validation - hf_token is optional at validation time
	// (can be provided via HF_TOKEN environment variable at runtime)
	paramsWithoutToken := map[string]interface{}{
		"min_speakers": 2,
		"max_speakers": 4,
	}

	if err := adapter.ValidateParameters(paramsWithoutToken); err != nil {
		t.Errorf("Parameters without HF token should pass validation (token can come from env var): %v", err)
	}

	// Test with token explicitly provided
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
	registry.RegisterDiarizationAdapter("sortformer", adapters.NewSortformerAdapter("/tmp/sortformer"))

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
	registry.RegisterTranscriptionAdapter("whisperx", adapters.NewWhisperXAdapter("/tmp/whisperx"))
	registry.RegisterDiarizationAdapter("pyannote", adapters.NewPyAnnoteAdapter("/tmp/pyannote"))

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
	// Register mock adapter
	registry.ClearRegistry()
	mockAdapter := new(MockTranscriptionAdapter)
	registry.RegisterTranscriptionAdapter("mock-model", mockAdapter)
	defer registry.ClearRegistry()

	// Create unified service with mock repo
	mockRepo := new(MockJobRepository)
	service := NewUnifiedTranscriptionService(mockRepo, "data/temp", "data/transcripts")

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
	mockRepo := new(MockJobRepository)
	service := NewUnifiedTranscriptionService(mockRepo, "data/temp", "data/transcripts")

	// Test creating audio input from a hypothetical file
	audioPath := "/tmp/test.wav"

	// This will fail since the file doesn't exist, but we can test the structure
	_, err := service.createAudioInput(audioPath)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParameterConversion(t *testing.T) {
	mockRepo := new(MockJobRepository)
	service := NewUnifiedTranscriptionService(mockRepo, "data/temp", "data/transcripts")

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
