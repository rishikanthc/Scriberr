package registry

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

// ModelRegistry manages all available model adapters with auto-discovery
type ModelRegistry struct {
	mu                    sync.RWMutex
	transcriptionAdapters map[string]interfaces.TranscriptionAdapter
	diarizationAdapters   map[string]interfaces.DiarizationAdapter
	compositeAdapters     map[string]interfaces.CompositeAdapter
	capabilities          map[string]interfaces.ModelCapabilities
	initialized           bool
}

// Global registry instance
var globalRegistry *ModelRegistry
var registryOnce sync.Once

// GetRegistry returns the global model registry instance
func GetRegistry() *ModelRegistry {
	registryOnce.Do(func() {
		globalRegistry = &ModelRegistry{
			transcriptionAdapters: make(map[string]interfaces.TranscriptionAdapter),
			diarizationAdapters:   make(map[string]interfaces.DiarizationAdapter),
			compositeAdapters:     make(map[string]interfaces.CompositeAdapter),
			capabilities:          make(map[string]interfaces.ModelCapabilities),
		}
	})
	return globalRegistry
}

// RegisterTranscriptionAdapter registers a transcription model adapter
func RegisterTranscriptionAdapter(modelID string, adapter interfaces.TranscriptionAdapter) {
	registry := GetRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.transcriptionAdapters[modelID] = adapter
	registry.capabilities[modelID] = adapter.GetCapabilities()

	logger.Debug("Registered transcription adapter",
		"model_id", modelID,
		"family", adapter.GetCapabilities().ModelFamily,
		"display_name", adapter.GetCapabilities().DisplayName)
}

// RegisterDiarizationAdapter registers a diarization model adapter
func RegisterDiarizationAdapter(modelID string, adapter interfaces.DiarizationAdapter) {
	registry := GetRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.diarizationAdapters[modelID] = adapter
	registry.capabilities[modelID] = adapter.GetCapabilities()

	logger.Debug("Registered diarization adapter",
		"model_id", modelID,
		"family", adapter.GetCapabilities().ModelFamily,
		"display_name", adapter.GetCapabilities().DisplayName)
}

// RegisterCompositeAdapter registers a composite (transcription + diarization) adapter
func RegisterCompositeAdapter(modelID string, adapter interfaces.CompositeAdapter) {
	registry := GetRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.compositeAdapters[modelID] = adapter
	registry.capabilities[modelID] = adapter.GetCapabilities()

	logger.Debug("Registered composite adapter",
		"model_id", modelID,
		"family", adapter.GetCapabilities().ModelFamily,
		"display_name", adapter.GetCapabilities().DisplayName)
}

// GetTranscriptionAdapter retrieves a transcription adapter by ID
func (r *ModelRegistry) GetTranscriptionAdapter(modelID string) (interfaces.TranscriptionAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if adapter, exists := r.transcriptionAdapters[modelID]; exists {
		return adapter, nil
	}

	// Check if it's available as a composite adapter
	if adapter, exists := r.compositeAdapters[modelID]; exists {
		return adapter, nil
	}

	return nil, fmt.Errorf("transcription adapter not found: %s", modelID)
}

// GetDiarizationAdapter retrieves a diarization adapter by ID
func (r *ModelRegistry) GetDiarizationAdapter(modelID string) (interfaces.DiarizationAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if adapter, exists := r.diarizationAdapters[modelID]; exists {
		return adapter, nil
	}

	// Check if it's available as a composite adapter
	if adapter, exists := r.compositeAdapters[modelID]; exists {
		return adapter, nil
	}

	return nil, fmt.Errorf("diarization adapter not found: %s", modelID)
}

// GetCompositeAdapter retrieves a composite adapter by ID
func (r *ModelRegistry) GetCompositeAdapter(modelID string) (interfaces.CompositeAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if adapter, exists := r.compositeAdapters[modelID]; exists {
		return adapter, nil
	}

	return nil, fmt.Errorf("composite adapter not found: %s", modelID)
}

// GetCapabilities returns the capabilities of a model
func (r *ModelRegistry) GetCapabilities(modelID string) (interfaces.ModelCapabilities, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if capabilities, exists := r.capabilities[modelID]; exists {
		return capabilities, nil
	}

	return interfaces.ModelCapabilities{}, fmt.Errorf("model not found: %s", modelID)
}

// GetAllCapabilities returns capabilities for all registered models
func (r *ModelRegistry) GetAllCapabilities() map[string]interfaces.ModelCapabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	result := make(map[string]interfaces.ModelCapabilities)
	for id, cap := range r.capabilities {
		result[id] = cap
	}
	return result
}

// GetTranscriptionModels returns all available transcription model IDs
func (r *ModelRegistry) GetTranscriptionModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []string
	for id := range r.transcriptionAdapters {
		models = append(models, id)
	}
	for id := range r.compositeAdapters {
		models = append(models, id)
	}

	sort.Strings(models)
	return models
}

// GetDiarizationModels returns all available diarization model IDs
func (r *ModelRegistry) GetDiarizationModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []string
	for id := range r.diarizationAdapters {
		models = append(models, id)
	}
	for id := range r.compositeAdapters {
		models = append(models, id)
	}

	sort.Strings(models)
	return models
}

// ModelScore represents a model's suitability score for given requirements
type ModelScore struct {
	ModelID string
	Score   float64
	Reasons []string
}

// SelectBestTranscriptionModel finds the best transcription model for given requirements
func (r *ModelRegistry) SelectBestTranscriptionModel(requirements interfaces.ModelRequirements) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []ModelScore

	// Score transcription adapters
	for modelID, adapter := range r.transcriptionAdapters {
		if score, reasons := r.scoreModel(adapter.GetCapabilities(), requirements); score > 0 {
			candidates = append(candidates, ModelScore{
				ModelID: modelID,
				Score:   score,
				Reasons: reasons,
			})
		}
	}

	// Score composite adapters
	for modelID, adapter := range r.compositeAdapters {
		if score, reasons := r.scoreModel(adapter.GetCapabilities(), requirements); score > 0 {
			candidates = append(candidates, ModelScore{
				ModelID: modelID,
				Score:   score,
				Reasons: reasons,
			})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no suitable transcription model found for requirements: %+v", requirements)
	}

	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	bestModel := candidates[0]
	logger.Info("Selected best transcription model",
		"model_id", bestModel.ModelID,
		"score", bestModel.Score,
		"reasons", strings.Join(bestModel.Reasons, ", "))

	return bestModel.ModelID, nil
}

// SelectBestDiarizationModel finds the best diarization model for given requirements
func (r *ModelRegistry) SelectBestDiarizationModel(requirements interfaces.ModelRequirements) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []ModelScore

	// Score diarization adapters
	for modelID, adapter := range r.diarizationAdapters {
		if score, reasons := r.scoreModel(adapter.GetCapabilities(), requirements); score > 0 {
			candidates = append(candidates, ModelScore{
				ModelID: modelID,
				Score:   score,
				Reasons: reasons,
			})
		}
	}

	// Score composite adapters
	for modelID, adapter := range r.compositeAdapters {
		if score, reasons := r.scoreModel(adapter.GetCapabilities(), requirements); score > 0 {
			candidates = append(candidates, ModelScore{
				ModelID: modelID,
				Score:   score,
				Reasons: reasons,
			})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no suitable diarization model found for requirements: %+v", requirements)
	}

	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	bestModel := candidates[0]
	logger.Info("Selected best diarization model",
		"model_id", bestModel.ModelID,
		"score", bestModel.Score,
		"reasons", strings.Join(bestModel.Reasons, ", "))

	return bestModel.ModelID, nil
}

// scoreModel calculates how well a model matches the requirements
func (r *ModelRegistry) scoreModel(capabilities interfaces.ModelCapabilities, requirements interfaces.ModelRequirements) (float64, []string) {
	score := 0.0
	var reasons []string

	// Language support (critical)
	if requirements.Language != "" {
		languageSupported := false
		for _, lang := range capabilities.SupportedLanguages {
			if lang == requirements.Language || lang == "auto" || lang == "*" {
				languageSupported = true
				break
			}
		}
		if !languageSupported {
			return 0, []string{"language not supported"}
		}
		score += 20
		reasons = append(reasons, "language supported")
	}

	// Required features (high priority)
	for _, feature := range requirements.Features {
		if supported, exists := capabilities.Features[feature]; exists && supported {
			score += 15
			reasons = append(reasons, fmt.Sprintf("supports %s", feature))
		} else {
			score -= 10
			reasons = append(reasons, fmt.Sprintf("missing %s", feature))
		}
	}

	// Memory requirements
	if requirements.MaxMemoryMB > 0 && capabilities.MemoryRequirement > requirements.MaxMemoryMB {
		score -= 20
		reasons = append(reasons, "exceeds memory limit")
	} else if requirements.MaxMemoryMB > 0 {
		score += 5
		reasons = append(reasons, "within memory limit")
	}

	// GPU requirements
	if requirements.RequireGPU != nil {
		if *requirements.RequireGPU && !capabilities.RequiresGPU {
			score -= 15
			reasons = append(reasons, "GPU required but not used")
		} else if !*requirements.RequireGPU && capabilities.RequiresGPU {
			score -= 10
			reasons = append(reasons, "GPU not preferred but required")
		} else {
			score += 10
			reasons = append(reasons, "GPU preference matched")
		}
	}

	// Preferred family
	if requirements.PreferredFamily != nil && capabilities.ModelFamily == *requirements.PreferredFamily {
		score += 15
		reasons = append(reasons, "preferred family")
	}

	// Quality preference
	switch requirements.Quality {
	case "fast":
		if strings.Contains(strings.ToLower(capabilities.ModelID), "fast") ||
			strings.Contains(strings.ToLower(capabilities.ModelID), "tiny") ||
			strings.Contains(strings.ToLower(capabilities.ModelID), "small") {
			score += 10
			reasons = append(reasons, "optimized for speed")
		}
	case "best":
		if strings.Contains(strings.ToLower(capabilities.ModelID), "large") ||
			strings.Contains(strings.ToLower(capabilities.ModelID), "xl") ||
			strings.Contains(strings.ToLower(capabilities.ModelID), "turbo") {
			score += 10
			reasons = append(reasons, "optimized for quality")
		}
	case "good":
		if strings.Contains(strings.ToLower(capabilities.ModelID), "medium") ||
			strings.Contains(strings.ToLower(capabilities.ModelID), "base") {
			score += 10
			reasons = append(reasons, "balanced quality/speed")
		}
	}

	// Apply constraints
	for key, value := range requirements.Constraints {
		if metaValue, exists := capabilities.Metadata[key]; exists && metaValue == value {
			score += 5
			reasons = append(reasons, fmt.Sprintf("constraint %s=%s met", key, value))
		}
	}

	return score, reasons
}

// InitializeModels ensures all registered models are ready to use (parallel)
func (r *ModelRegistry) InitializeModels(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return nil
	}

	logger.Info("Initializing registered models sequentially...")

	initErrors := make(chan error, 10) // Buffer for potential errors

	// Initialize transcription adapters sequentially
	for modelID, adapter := range r.transcriptionAdapters {
		logger.Debug("Initializing transcription model", "model_id", modelID)
		if err := adapter.PrepareEnvironment(ctx); err != nil {
			logger.Error("Failed to initialize transcription model",
				"model_id", modelID, "error", err)
			initErrors <- fmt.Errorf("transcription model %s: %w", modelID, err)
		} else {
			logger.Info("Transcription model initialized", "model_id", modelID)
		}
	}

	// Initialize diarization adapters sequentially
	for modelID, adapter := range r.diarizationAdapters {
		logger.Debug("Initializing diarization model", "model_id", modelID)
		if err := adapter.PrepareEnvironment(ctx); err != nil {
			logger.Error("Failed to initialize diarization model",
				"model_id", modelID, "error", err)
			initErrors <- fmt.Errorf("diarization model %s: %w", modelID, err)
		} else {
			logger.Info("Diarization model initialized", "model_id", modelID)
		}
	}

	// Initialize composite adapters sequentially
	for modelID, adapter := range r.compositeAdapters {
		logger.Debug("Initializing composite model", "model_id", modelID)
		if err := adapter.PrepareEnvironment(ctx); err != nil {
			logger.Error("Failed to initialize composite model",
				"model_id", modelID, "error", err)
			initErrors <- fmt.Errorf("composite model %s: %w", modelID, err)
		} else {
			logger.Info("Composite model initialized", "model_id", modelID)
		}
	}

	close(initErrors)

	// Collect any errors (but don't fail completely)
	var errorList []error
	for err := range initErrors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		logger.Warn("Some models failed to initialize", "error_count", len(errorList))
		for _, err := range errorList {
			logger.Warn("Model initialization error", "error", err)
		}
	}

	r.initialized = true
	logger.Info("Model initialization completed")
	return nil
}

// GetModelStatus returns the status of all registered models
func (r *ModelRegistry) GetModelStatus(ctx context.Context) map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]bool)

	// Check transcription adapters
	for modelID, adapter := range r.transcriptionAdapters {
		status[modelID] = adapter.IsReady(ctx)
	}

	// Check diarization adapters
	for modelID, adapter := range r.diarizationAdapters {
		status[modelID] = adapter.IsReady(ctx)
	}

	// Check composite adapters
	for modelID, adapter := range r.compositeAdapters {
		status[modelID] = adapter.IsReady(ctx)
	}

	return status
}

// GetEstimatedProcessingTime estimates processing time for given input and model
func (r *ModelRegistry) GetEstimatedProcessingTime(modelID string, input interfaces.AudioInput) (time.Duration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check transcription adapters
	if adapter, exists := r.transcriptionAdapters[modelID]; exists {
		return adapter.GetEstimatedProcessingTime(input), nil
	}

	// Check diarization adapters
	if adapter, exists := r.diarizationAdapters[modelID]; exists {
		return adapter.GetEstimatedProcessingTime(input), nil
	}

	// Check composite adapters
	if adapter, exists := r.compositeAdapters[modelID]; exists {
		return adapter.GetEstimatedProcessingTime(input), nil
	}

	return 0, fmt.Errorf("model not found: %s", modelID)
}

// ValidateModelParameters validates parameters for a specific model
func (r *ModelRegistry) ValidateModelParameters(modelID string, params map[string]interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check transcription adapters
	if adapter, exists := r.transcriptionAdapters[modelID]; exists {
		return adapter.ValidateParameters(params)
	}

	// Check diarization adapters
	if adapter, exists := r.diarizationAdapters[modelID]; exists {
		return adapter.ValidateParameters(params)
	}

	// Check composite adapters
	if adapter, exists := r.compositeAdapters[modelID]; exists {
		return adapter.ValidateParameters(params)
	}

	return fmt.Errorf("model not found: %s", modelID)
}

// GetParameterSchema returns the parameter schema for a specific model
func (r *ModelRegistry) GetParameterSchema(modelID string) ([]interfaces.ParameterSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check transcription adapters
	if adapter, exists := r.transcriptionAdapters[modelID]; exists {
		return adapter.GetParameterSchema(), nil
	}

	// Check diarization adapters
	if adapter, exists := r.diarizationAdapters[modelID]; exists {
		return adapter.GetParameterSchema(), nil
	}

	// Check composite adapters
	if adapter, exists := r.compositeAdapters[modelID]; exists {
		return adapter.GetParameterSchema(), nil
	}

	return nil, fmt.Errorf("model not found: %s", modelID)
}

// Test helper functions

// ClearRegistry clears all registered adapters (for testing only)
func ClearRegistry() {
	registry := GetRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.transcriptionAdapters = make(map[string]interfaces.TranscriptionAdapter)
	registry.diarizationAdapters = make(map[string]interfaces.DiarizationAdapter)
	registry.compositeAdapters = make(map[string]interfaces.CompositeAdapter)
	registry.capabilities = make(map[string]interfaces.ModelCapabilities)
	registry.initialized = false
}

// GetTranscriptionAdapters returns all registered transcription adapters (for testing)
func GetTranscriptionAdapters() map[string]interfaces.TranscriptionAdapter {
	registry := GetRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]interfaces.TranscriptionAdapter)
	for id, adapter := range registry.transcriptionAdapters {
		result[id] = adapter
	}
	return result
}

// GetDiarizationAdapters returns all registered diarization adapters (for testing)
func GetDiarizationAdapters() map[string]interfaces.DiarizationAdapter {
	registry := GetRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]interfaces.DiarizationAdapter)
	for id, adapter := range registry.diarizationAdapters {
		result[id] = adapter
	}
	return result
}
