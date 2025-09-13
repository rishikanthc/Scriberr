package interfaces

import (
	"context"
	"time"

	"scriberr/internal/models"
)

// ModelCapabilities describes what a model can do and its requirements
type ModelCapabilities struct {
	ModelID            string            `json:"model_id"`
	ModelFamily        string            `json:"model_family"`
	DisplayName        string            `json:"display_name"`
	Description        string            `json:"description"`
	Version            string            `json:"version"`
	SupportedLanguages []string          `json:"supported_languages"`
	SupportedFormats   []string          `json:"supported_formats"`
	RequiresGPU        bool              `json:"requires_gpu"`
	MemoryRequirement  int               `json:"memory_requirement_mb"`
	Features           map[string]bool   `json:"features"`
	Metadata           map[string]string `json:"metadata"`
}

// ParameterSchema defines a parameter that a model accepts
type ParameterSchema struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // "int", "float", "string", "bool", "[]string"
	Required    bool        `json:"required"`
	Default     interface{} `json:"default"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
	Options     []string    `json:"options,omitempty"` // For enum-like parameters
	Description string      `json:"description"`
	Group       string      `json:"group"` // For UI grouping: "basic", "advanced", "quality"
}

// ValidationRule defines parameter validation logic
type ValidationRule struct {
	Pattern     string `json:"pattern,omitempty"`     // Regex pattern for strings
	MinLength   *int   `json:"min_length,omitempty"`  // For strings and arrays
	MaxLength   *int   `json:"max_length,omitempty"`  // For strings and arrays
	CustomCheck string `json:"custom_check,omitempty"` // Custom validation function name
}

// AudioInput represents input audio data and metadata
type AudioInput struct {
	FilePath     string            `json:"file_path"`
	Format       string            `json:"format"`
	SampleRate   int               `json:"sample_rate"`
	Channels     int               `json:"channels"`
	Duration     time.Duration     `json:"duration"`
	Size         int64             `json:"size"`
	Metadata     map[string]string `json:"metadata"`
	TempFilePath string            `json:"temp_file_path,omitempty"` // For converted files
}

// TranscriptSegment represents a segment of transcribed audio
type TranscriptSegment struct {
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Text     string  `json:"text"`
	Speaker  *string `json:"speaker,omitempty"`
	Language *string `json:"language,omitempty"`
}

// TranscriptWord represents word-level timing information
type TranscriptWord struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Word    string  `json:"word"`
	Score   float64 `json:"score"`
	Speaker *string `json:"speaker,omitempty"`
}

// TranscriptResult represents the output of transcription
type TranscriptResult struct {
	Text         string             `json:"text"`
	Language     string             `json:"language"`
	Segments     []TranscriptSegment `json:"segments"`
	WordSegments []TranscriptWord   `json:"word_segments,omitempty"`
	Confidence   float64            `json:"confidence"`
	ProcessingTime time.Duration    `json:"processing_time"`
	ModelUsed    string             `json:"model_used"`
	Metadata     map[string]string  `json:"metadata"`
}

// DiarizationSegment represents speaker diarization information
type DiarizationSegment struct {
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Speaker  string  `json:"speaker"`
	Confidence float64 `json:"confidence"`
}

// DiarizationResult represents the output of speaker diarization
type DiarizationResult struct {
	Segments       []DiarizationSegment `json:"segments"`
	SpeakerCount   int                  `json:"speaker_count"`
	Speakers       []string             `json:"speakers"`
	ProcessingTime time.Duration        `json:"processing_time"`
	ModelUsed      string               `json:"model_used"`
	Metadata       map[string]string    `json:"metadata"`
}

// ProcessingContext contains context information for processing
type ProcessingContext struct {
	JobID           string            `json:"job_id"`
	UserID          *string           `json:"user_id,omitempty"`
	OutputDirectory string            `json:"output_directory"`
	TempDirectory   string            `json:"temp_directory"`
	Metadata        map[string]string `json:"metadata"`
}

// ModelAdapter is the base interface that all model adapters must implement
type ModelAdapter interface {
	// GetCapabilities returns what this model can do
	GetCapabilities() ModelCapabilities

	// GetParameterSchema returns the parameters this model accepts
	GetParameterSchema() []ParameterSchema

	// ValidateParameters checks if the provided parameters are valid for this model
	ValidateParameters(params map[string]interface{}) error

	// PrepareEnvironment ensures the model environment is ready (downloads, installs, etc.)
	PrepareEnvironment(ctx context.Context) error

	// GetModelPath returns the path where the model files are stored
	GetModelPath() string

	// IsReady checks if the model is ready to process jobs
	IsReady(ctx context.Context) bool

	// GetEstimatedProcessingTime estimates how long processing will take
	GetEstimatedProcessingTime(input AudioInput) time.Duration
}

// TranscriptionAdapter handles audio transcription
type TranscriptionAdapter interface {
	ModelAdapter

	// Transcribe processes audio and returns transcription
	Transcribe(ctx context.Context, input AudioInput, params map[string]interface{}, procCtx ProcessingContext) (*TranscriptResult, error)

	// GetSupportedModels returns the list of specific model variants this adapter supports
	GetSupportedModels() []string
}

// DiarizationAdapter handles speaker diarization
type DiarizationAdapter interface {
	ModelAdapter

	// Diarize processes audio and returns speaker diarization
	Diarize(ctx context.Context, input AudioInput, params map[string]interface{}, procCtx ProcessingContext) (*DiarizationResult, error)

	// GetMaxSpeakers returns the maximum number of speakers this model can handle
	GetMaxSpeakers() int

	// GetMinSpeakers returns the minimum number of speakers this model requires
	GetMinSpeakers() int
}

// CompositeAdapter can combine transcription and diarization
type CompositeAdapter interface {
	TranscriptionAdapter
	DiarizationAdapter

	// ProcessCombined performs both transcription and diarization in an optimized way
	ProcessCombined(ctx context.Context, input AudioInput, params map[string]interface{}, procCtx ProcessingContext) (*TranscriptResult, *DiarizationResult, error)
}

// ModelRequirements specifies what capabilities are needed for a job
type ModelRequirements struct {
	Language         string            `json:"language"`
	Features         []string          `json:"features"`         // "timestamps", "diarization", "translation"
	Quality          string            `json:"quality"`          // "fast", "good", "best"
	MaxMemoryMB      int               `json:"max_memory_mb"`
	RequireGPU       *bool             `json:"require_gpu,omitempty"`
	PreferredFamily  *string           `json:"preferred_family,omitempty"`
	MinConfidence    float64           `json:"min_confidence"`
	MaxProcessingTime *time.Duration   `json:"max_processing_time,omitempty"`
	Constraints      map[string]string `json:"constraints"`
}

// AdapterFactory creates model adapters
type AdapterFactory interface {
	// CreateTranscriptionAdapter creates a transcription adapter by ID
	CreateTranscriptionAdapter(modelID string) (TranscriptionAdapter, error)

	// CreateDiarizationAdapter creates a diarization adapter by ID
	CreateDiarizationAdapter(modelID string) (DiarizationAdapter, error)

	// CreateCompositeAdapter creates a composite adapter by ID
	CreateCompositeAdapter(modelID string) (CompositeAdapter, error)

	// GetAvailableModels returns all available model IDs and their capabilities
	GetAvailableModels() map[string]ModelCapabilities

	// SelectBestModel selects the best model for the given requirements
	SelectBestModel(requirements ModelRequirements) (string, error)
}

// ProcessingPipeline handles the full processing workflow
type ProcessingPipeline interface {
	// Process handles the complete transcription/diarization workflow
	Process(ctx context.Context, job *models.TranscriptionJob) error

	// GetSupportedModels returns all models available through this pipeline
	GetSupportedModels() map[string]ModelCapabilities
}

// Preprocessor handles audio preprocessing before model execution
type Preprocessor interface {
	// Process transforms the audio input
	Process(ctx context.Context, input AudioInput) (AudioInput, error)

	// AppliesTo determines if this preprocessor should be used for the given model
	AppliesTo(capabilities ModelCapabilities) bool

	// GetRequiredFormats returns the output formats this preprocessor can produce
	GetRequiredFormats() []string
}

// Postprocessor handles result processing after model execution
type Postprocessor interface {
	// ProcessTranscript processes transcription results
	ProcessTranscript(ctx context.Context, result *TranscriptResult, params map[string]interface{}) (*TranscriptResult, error)

	// ProcessDiarization processes diarization results
	ProcessDiarization(ctx context.Context, result *DiarizationResult, params map[string]interface{}) (*DiarizationResult, error)

	// AppliesTo determines if this postprocessor should be used
	AppliesTo(capabilities ModelCapabilities, params map[string]interface{}) bool
}

// Legacy type aliases for backward compatibility
type Segment = TranscriptSegment
type Word = TranscriptWord