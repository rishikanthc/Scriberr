package transcription

import (
	"context"
	"os/exec"

	"scriberr/internal/repository"
	"scriberr/pkg/logger"
)

// UnifiedJobProcessor implements the existing JobProcessor interface using the new unified service
type UnifiedJobProcessor struct {
	unifiedService *UnifiedTranscriptionService
}

// NewUnifiedJobProcessor creates a new job processor using the unified service
func NewUnifiedJobProcessor(jobRepo repository.JobRepository, tempDir, outputDir string) *UnifiedJobProcessor {
	return &UnifiedJobProcessor{
		unifiedService: NewUnifiedTranscriptionService(jobRepo, tempDir, outputDir),
	}
}

// Initialize prepares the job processor
func (u *UnifiedJobProcessor) Initialize(ctx context.Context) error {
	return u.unifiedService.Initialize(ctx)
}

// ProcessJob implements the legacy JobProcessor interface
func (u *UnifiedJobProcessor) ProcessJob(ctx context.Context, jobID string) error {
	logger.Info("Processing job with unified processor", "job_id", jobID)
	return u.unifiedService.ProcessJob(ctx, jobID)
}

// ProcessJobWithProcess implements the enhanced JobProcessor interface with process registration
func (u *UnifiedJobProcessor) ProcessJobWithProcess(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error {
	// Note: The new adapter architecture doesn't expose the underlying process in the same way
	// For backward compatibility, we'll call the registerProcess function with nil
	// In the future, we could modify adapters to support process registration if needed

	logger.Info("Processing job with unified processor (with process registration)", "job_id", jobID)

	// Register a nil process for backward compatibility
	registerProcess(nil)

	return u.unifiedService.ProcessJob(ctx, jobID)
}

// GetUnifiedService returns the underlying unified service for direct access to new features
func (u *UnifiedJobProcessor) GetUnifiedService() *UnifiedTranscriptionService {
	return u.unifiedService
}

// GetSupportedModels returns all supported models through the new architecture
func (u *UnifiedJobProcessor) GetSupportedModels() map[string]interface{} {
	capabilities := u.unifiedService.GetSupportedModels()

	// Convert to the format expected by existing APIs
	result := make(map[string]interface{})
	for modelID, cap := range capabilities {
		result[modelID] = map[string]interface{}{
			"id":           cap.ModelID,
			"family":       cap.ModelFamily,
			"name":         cap.DisplayName,
			"description":  cap.Description,
			"version":      cap.Version,
			"languages":    cap.SupportedLanguages,
			"formats":      cap.SupportedFormats,
			"features":     cap.Features,
			"memory_mb":    cap.MemoryRequirement,
			"requires_gpu": cap.RequiresGPU,
		}
	}

	return result
}

// GetModelStatus returns the status of all models
func (u *UnifiedJobProcessor) GetModelStatus(ctx context.Context) map[string]bool {
	return u.unifiedService.GetModelStatus(ctx)
}

// ValidateModelParameters validates parameters for a specific model
func (u *UnifiedJobProcessor) ValidateModelParameters(modelID string, params map[string]interface{}) error {
	return u.unifiedService.ValidateModelParameters(modelID, params)
}

// InitEmbeddedPythonEnv initializes the Python environment for all adapters
func (u *UnifiedJobProcessor) InitEmbeddedPythonEnv() error {
	ctx := context.Background()
	return u.unifiedService.Initialize(ctx)
}

// GetSupportedLanguages returns supported languages from all models
func (u *UnifiedJobProcessor) GetSupportedLanguages() []string {
	// Aggregate unique languages from all models
	languageSet := make(map[string]bool)

	capabilities := u.unifiedService.GetSupportedModels()
	for _, cap := range capabilities {
		for _, lang := range cap.SupportedLanguages {
			languageSet[lang] = true
		}
	}

	// Convert to sorted slice
	languages := make([]string, 0, len(languageSet))
	for lang := range languageSet {
		languages = append(languages, lang)
	}

	// Sort for consistent output
	sort := func(slice []string) {
		for i := 0; i < len(slice)-1; i++ {
			for j := i + 1; j < len(slice); j++ {
				if slice[i] > slice[j] {
					slice[i], slice[j] = slice[j], slice[i]
				}
			}
		}
	}
	sort(languages)

	return languages
}

// ensurePythonEnv ensures Python environment is ready (for compatibility)
func (u *UnifiedJobProcessor) ensurePythonEnv() error {
	ctx := context.Background()
	return u.unifiedService.Initialize(ctx)
}

// TerminateMultiTrackJob terminates a multi-track job and all its individual track jobs
func (u *UnifiedJobProcessor) TerminateMultiTrackJob(jobID string) error {
	return u.unifiedService.TerminateMultiTrackJob(jobID)
}

// IsMultiTrackJob checks if a job is a multi-track job
func (u *UnifiedJobProcessor) IsMultiTrackJob(jobID string) bool {
	return u.unifiedService.IsMultiTrackJob(jobID)
}
