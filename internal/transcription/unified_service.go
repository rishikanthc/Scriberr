package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

// UnifiedTranscriptionService provides a unified interface for all transcription and diarization models
type UnifiedTranscriptionService struct {
	registry          *registry.ModelRegistry
	preprocessors     map[string]interfaces.Preprocessor
	postprocessors    map[string]interfaces.Postprocessor
	tempDirectory     string
	outputDirectory   string
	defaultModelIDs   map[string]string // Default model IDs for each task type
}

// NewUnifiedTranscriptionService creates a new unified transcription service
func NewUnifiedTranscriptionService() *UnifiedTranscriptionService {
	return &UnifiedTranscriptionService{
		registry:        registry.GetRegistry(),
		preprocessors:   make(map[string]interfaces.Preprocessor),
		postprocessors:  make(map[string]interfaces.Postprocessor),
		tempDirectory:   "data/temp",
		outputDirectory: "data/transcripts",
		defaultModelIDs: map[string]string{
			"transcription": "whisperx",
			"diarization":   "pyannote",
		},
	}
}

// Initialize prepares all registered models for use
func (u *UnifiedTranscriptionService) Initialize(ctx context.Context) error {
	logger.Info("Initializing unified transcription service")

	// Create necessary directories
	if err := os.MkdirAll(u.tempDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	if err := os.MkdirAll(u.outputDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Initialize all registered models
	if err := u.registry.InitializeModels(ctx); err != nil {
		return fmt.Errorf("failed to initialize models: %w", err)
	}

	logger.Info("Unified transcription service initialized successfully")
	return nil
}

// ProcessJob processes a transcription job using the new adapter architecture
func (u *UnifiedTranscriptionService) ProcessJob(ctx context.Context, jobID string) error {
	startTime := time.Now()
	logger.Info("Processing job with unified service", "job_id", jobID)

	// Get the job from database
	var job models.TranscriptionJob
	if err := database.DB.Preload("MultiTrackFiles").Where("id = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Create execution record
	execution := &models.TranscriptionJobExecution{
		TranscriptionJobID: jobID,
		StartedAt:          startTime,
		ActualParameters:   job.Parameters,
		Status:             models.StatusProcessing,
	}

	if err := database.DB.Create(execution).Error; err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	// Helper function to update execution status
	updateExecutionStatus := func(status models.JobStatus, errorMsg string) {
		completedAt := time.Now()
		execution.CompletedAt = &completedAt
		execution.Status = status
		execution.CalculateProcessingDuration()

		if errorMsg != "" {
			execution.ErrorMessage = &errorMsg
		}

		database.DB.Save(execution)
	}

	// Check for multi-track processing
	if job.IsMultiTrack && job.Parameters.IsMultiTrackEnabled {
		logger.Info("Processing multi-track job", "job_id", jobID)
		if err := u.processMultiTrackJob(ctx, &job); err != nil {
			errMsg := fmt.Sprintf("multi-track processing failed: %v", err)
			updateExecutionStatus(models.StatusFailed, errMsg)
			return fmt.Errorf(errMsg)
		}
	} else {
		// Process single track
		if err := u.processSingleTrackJob(ctx, &job); err != nil {
			errMsg := fmt.Sprintf("single-track processing failed: %v", err)
			updateExecutionStatus(models.StatusFailed, errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	// Success
	updateExecutionStatus(models.StatusCompleted, "")
	logger.Info("Job processed successfully", "job_id", jobID, "duration", time.Since(startTime))
	return nil
}

// processSingleTrackJob handles single audio file transcription
func (u *UnifiedTranscriptionService) processSingleTrackJob(ctx context.Context, job *models.TranscriptionJob) error {
	logger.Info("Processing single-track job", "job_id", job.ID, "model_family", job.Parameters.ModelFamily)

	// Create processing context
	procCtx := interfaces.ProcessingContext{
		JobID:           job.ID,
		OutputDirectory: filepath.Join(u.outputDirectory, job.ID),
		TempDirectory:   u.tempDirectory,
		Metadata:        map[string]string{},
	}

	// Create output directory
	if err := os.MkdirAll(procCtx.OutputDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create audio input
	audioInput, err := u.createAudioInput(job.AudioPath)
	if err != nil {
		return fmt.Errorf("failed to create audio input: %w", err)
	}

	// Determine models to use first
	transcriptionModelID, diarizationModelID, err := u.selectModels(job.Parameters)
	if err != nil {
		return fmt.Errorf("failed to select models: %w", err)
	}

	var transcriptResult *interfaces.TranscriptResult
	var diarizationResult *interfaces.DiarizationResult

	// Perform transcription
	if transcriptionModelID != "" {
		logger.Info("Running transcription", "model_id", transcriptionModelID)
		transcriptionAdapter, err := u.registry.GetTranscriptionAdapter(transcriptionModelID)
		if err != nil {
			return fmt.Errorf("failed to get transcription adapter: %w", err)
		}

		// Convert parameters for this specific model
		params := u.convertParametersForModel(job.Parameters, transcriptionModelID)

		transcriptResult, err = transcriptionAdapter.Transcribe(ctx, audioInput, params, procCtx)
		if err != nil {
			return fmt.Errorf("transcription failed: %w", err)
		}
	}

	// Perform diarization if requested and not already done by transcription
	if job.Parameters.Diarize && diarizationModelID != "" {
		// Convert parameters for diarization model
		diarizationParams := u.convertParametersForModel(job.Parameters, diarizationModelID)
		
		if !u.transcriptionIncludesDiarization(transcriptionModelID, diarizationParams) {
			logger.Info("Running separate diarization", "model_id", diarizationModelID)
			diarizationAdapter, err := u.registry.GetDiarizationAdapter(diarizationModelID)
			if err != nil {
				return fmt.Errorf("failed to get diarization adapter: %w", err)
			}

			diarizationResult, err = diarizationAdapter.Diarize(ctx, audioInput, diarizationParams, procCtx)
			if err != nil {
				return fmt.Errorf("diarization failed: %w", err)
			}

			// Merge diarization results with transcription
			if transcriptResult != nil && diarizationResult != nil {
				transcriptResult = u.mergeDiarizationWithTranscription(transcriptResult, diarizationResult)
			}
		}
	}

	// Save results to database
	if transcriptResult != nil {
		if err := u.saveTranscriptionResults(job.ID, transcriptResult); err != nil {
			return fmt.Errorf("failed to save transcription results: %w", err)
		}
	}

	return nil
}

// processMultiTrackJob handles multi-track audio processing
func (u *UnifiedTranscriptionService) processMultiTrackJob(ctx context.Context, job *models.TranscriptionJob) error {
	logger.Info("Processing multi-track job", "job_id", job.ID, "track_count", len(job.MultiTrackFiles))

	// Create unified processor for this service
	unifiedProcessor := &UnifiedJobProcessor{
		unifiedService: u,
	}
	
	// Create multi-track transcriber with unified processor
	transcriber := NewMultiTrackTranscriber(unifiedProcessor)
	
	// Process the multi-track transcription
	return transcriber.ProcessMultiTrackTranscription(ctx, job.ID)
}

// selectModels determines which models to use based on job parameters
func (u *UnifiedTranscriptionService) selectModels(params models.WhisperXParams) (transcriptionModelID, diarizationModelID string, err error) {
	// Determine transcription model
	switch params.ModelFamily {
	case "nvidia_parakeet":
		transcriptionModelID = "parakeet"
	case "nvidia_canary":
		transcriptionModelID = "canary"
	case "whisper":
		transcriptionModelID = "whisperx"
	default:
		transcriptionModelID = "whisperx" // Default fallback
	}

	// Determine diarization model if needed
	if params.Diarize {
		switch params.DiarizeModel {
		case "nvidia_sortformer":
			diarizationModelID = "sortformer"
		case "pyannote", "pyannote/speaker-diarization-3.1":
			diarizationModelID = "pyannote"
		default:
			diarizationModelID = "pyannote" // Default fallback
		}
	}

	logger.Info("Selected models", 
		"transcription", transcriptionModelID, 
		"diarization", diarizationModelID,
		"original_family", params.ModelFamily,
		"original_diarize_model", params.DiarizeModel)

	return transcriptionModelID, diarizationModelID, nil
}

// transcriptionIncludesDiarization checks if the transcription model already includes diarization
func (u *UnifiedTranscriptionService) transcriptionIncludesDiarization(modelID string, params map[string]interface{}) bool {
	// WhisperX includes diarization when enabled
	if modelID == "whisperx" {
		if diarize, ok := params["diarize"].(bool); ok && diarize {
			// Check if it's using nvidia_sortformer (which requires separate processing)
			if diarizeModel, ok := params["diarize_model"].(string); ok {
				return diarizeModel != "nvidia_sortformer"
			}
			return true
		}
	}
	
	return false
}

// createAudioInput creates an AudioInput from a file path
func (u *UnifiedTranscriptionService) createAudioInput(audioPath string) (interfaces.AudioInput, error) {
	// Get file info
	fileInfo, err := os.Stat(audioPath)
	if err != nil {
		return interfaces.AudioInput{}, fmt.Errorf("failed to stat audio file: %w", err)
	}

	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(audioPath))
	format := strings.TrimPrefix(ext, ".")

	// Create audio input (basic implementation)
	audioInput := interfaces.AudioInput{
		FilePath: audioPath,
		Format:   format,
		Size:     fileInfo.Size(),
		Metadata: map[string]string{},
	}

	// TODO: Extract actual audio metadata (sample rate, channels, duration)
	// For now, use defaults
	audioInput.SampleRate = 16000
	audioInput.Channels = 1
	audioInput.Duration = time.Duration(float64(fileInfo.Size()/32000)) * time.Second // Rough estimate

	return audioInput, nil
}

// parametersToMap converts WhisperXParams to a generic parameter map
// convertParametersForModel converts WhisperX parameters to model-specific parameters
func (u *UnifiedTranscriptionService) convertParametersForModel(params models.WhisperXParams, modelID string) map[string]interface{} {
	switch modelID {
	case "parakeet":
		return u.convertToParakeetParams(params)
	case "canary":
		return u.convertToCanaryParams(params)
	case "whisperx":
		return u.convertToWhisperXParams(params)
	case "pyannote":
		return u.convertToPyannoteParams(params)
	case "sortformer":
		return u.convertToSortformerParams(params)
	default:
		// Fallback to legacy conversion
		return u.parametersToMap(params)
	}
}

// convertToParakeetParams converts to Parakeet-specific parameters
func (u *UnifiedTranscriptionService) convertToParakeetParams(params models.WhisperXParams) map[string]interface{} {
	return map[string]interface{}{
		"timestamps":     true,
		"context_left":   params.AttentionContextLeft,
		"context_right":  params.AttentionContextRight,
		"output_format":  "json",
		"auto_convert_audio": true,
	}
}

// convertToCanaryParams converts to Canary-specific parameters
func (u *UnifiedTranscriptionService) convertToCanaryParams(params models.WhisperXParams) map[string]interface{} {
	paramMap := map[string]interface{}{
		"timestamps":     true,
		"output_format":  "json",
		"auto_convert_audio": true,
		"task":           params.Task,
	}
	
	// Set source language
	if params.Language != nil {
		paramMap["source_lang"] = *params.Language
	} else {
		paramMap["source_lang"] = "en"
	}
	
	// Set target language for translation
	if params.Task == "translate" {
		paramMap["target_lang"] = "en"
	}
	
	return paramMap
}

// convertToWhisperXParams converts to WhisperX-specific parameters
func (u *UnifiedTranscriptionService) convertToWhisperXParams(params models.WhisperXParams) map[string]interface{} {
	// For WhisperX, we use the standard WhisperX parameters (no NVIDIA-specific ones)
	paramMap := map[string]interface{}{
		// Core parameters
		"model":         params.Model,
		"device":        params.Device,
		"device_index":  params.DeviceIndex,
		"batch_size":    params.BatchSize,
		"compute_type":  params.ComputeType,
		"threads":       params.Threads,
		
		// Task and language
		"task":          params.Task,
		
		// Diarization
		"diarize":       params.Diarize,
		"diarize_model": params.DiarizeModel,
		
		// Quality settings
		"temperature":    params.Temperature,
		"best_of":        params.BestOf,
		"beam_size":      params.BeamSize,
		"patience":       params.Patience,
		
		// VAD settings
		"vad_method":     params.VadMethod,
		"vad_onset":      params.VadOnset,
		"vad_offset":     params.VadOffset,
	}
	
	// Handle pointer fields - only add if not nil
	if params.Language != nil {
		paramMap["language"] = *params.Language
	}
	if params.MinSpeakers != nil {
		paramMap["min_speakers"] = *params.MinSpeakers
	}
	if params.MaxSpeakers != nil {
		paramMap["max_speakers"] = *params.MaxSpeakers
	}
	if params.HfToken != nil {
		paramMap["hf_token"] = *params.HfToken
	}
	if params.ModelDir != nil {
		paramMap["model_dir"] = *params.ModelDir
	}
	if params.AlignModel != nil {
		paramMap["align_model"] = *params.AlignModel
	}
	if params.SuppressTokens != nil {
		paramMap["suppress_tokens"] = *params.SuppressTokens
	}
	if params.InitialPrompt != nil {
		paramMap["initial_prompt"] = *params.InitialPrompt
	}
	
	return paramMap
}

// convertToPyannoteParams converts to PyAnnote-specific parameters
func (u *UnifiedTranscriptionService) convertToPyannoteParams(params models.WhisperXParams) map[string]interface{} {
	paramMap := map[string]interface{}{
		"output_format": "json",
		"auto_convert_audio": true,
	}
	
	if params.MinSpeakers != nil {
		paramMap["min_speakers"] = *params.MinSpeakers
	}
	if params.MaxSpeakers != nil {
		paramMap["max_speakers"] = *params.MaxSpeakers
	}
	if params.HfToken != nil {
		paramMap["hf_token"] = *params.HfToken
	}
	
	return paramMap
}

// convertToSortformerParams converts to Sortformer-specific parameters
func (u *UnifiedTranscriptionService) convertToSortformerParams(params models.WhisperXParams) map[string]interface{} {
	return map[string]interface{}{
		"output_format": "json",
		"auto_convert_audio": true,
		// Sortformer is optimized for 4 speakers, no additional config needed
	}
}

func (u *UnifiedTranscriptionService) parametersToMap(params models.WhisperXParams) map[string]interface{} {
	paramMap := map[string]interface{}{
		// Core parameters
		"model":         params.Model,
		"device":        params.Device,
		"device_index":  params.DeviceIndex,
		"batch_size":    params.BatchSize,
		"compute_type":  params.ComputeType,
		"threads":       params.Threads,
		
		// Language and task
		"task":          params.Task,
		
		// Diarization
		"diarize":       params.Diarize,
		"diarize_model": params.DiarizeModel,
	}
	
	// Handle pointer fields - only add if not nil
	if params.Language != nil {
		paramMap["language"] = *params.Language
	}
	if params.MinSpeakers != nil {
		paramMap["min_speakers"] = *params.MinSpeakers
	}
	if params.MaxSpeakers != nil {
		paramMap["max_speakers"] = *params.MaxSpeakers
	}
	if params.HfToken != nil {
		paramMap["hf_token"] = *params.HfToken
	}
	if params.ModelDir != nil {
		paramMap["model_dir"] = *params.ModelDir
	}
	if params.AlignModel != nil {
		paramMap["align_model"] = *params.AlignModel
	}
	if params.SuppressTokens != nil {
		paramMap["suppress_tokens"] = *params.SuppressTokens
	}
	if params.InitialPrompt != nil {
		paramMap["initial_prompt"] = *params.InitialPrompt
	}
	
	// Add remaining non-pointer fields
	paramMap["temperature"] = params.Temperature
	paramMap["best_of"] = params.BestOf
	paramMap["beam_size"] = params.BeamSize
	paramMap["patience"] = params.Patience
	paramMap["vad_method"] = params.VadMethod
	paramMap["vad_onset"] = params.VadOnset
	paramMap["vad_offset"] = params.VadOffset
	paramMap["context_left"] = params.AttentionContextLeft
	paramMap["context_right"] = params.AttentionContextRight
	paramMap["timestamps"] = true
	paramMap["output_format"] = "json"
	paramMap["auto_convert_audio"] = true

	// For Canary model, set source and target languages
	if params.ModelFamily == "nvidia_canary" {
		if params.Language != nil {
			paramMap["source_lang"] = *params.Language
		} else {
			paramMap["source_lang"] = "en"
		}
		
		if params.Task == "translate" {
			paramMap["target_lang"] = "en" // Default target for translation
		} else {
			paramMap["target_lang"] = paramMap["source_lang"]
		}
	}

	return paramMap
}

// mergeDiarizationWithTranscription combines diarization results with transcription
func (u *UnifiedTranscriptionService) mergeDiarizationWithTranscription(transcript *interfaces.TranscriptResult, diarization *interfaces.DiarizationResult) *interfaces.TranscriptResult {
	logger.Info("Merging diarization with transcription", 
		"transcript_segments", len(transcript.Segments),
		"diarization_segments", len(diarization.Segments))

	// Create a copy of the transcript to avoid modifying the original
	mergedTranscript := *transcript
	mergedTranscript.Segments = make([]interfaces.TranscriptSegment, len(transcript.Segments))
	copy(mergedTranscript.Segments, transcript.Segments)

	// Assign speakers to transcript segments based on timing overlap
	for i := range mergedTranscript.Segments {
		segment := &mergedTranscript.Segments[i]
		bestSpeaker := u.findBestSpeakerForSegment(segment.Start, segment.End, diarization.Segments)
		if bestSpeaker != "" {
			segment.Speaker = &bestSpeaker
		}
	}

	// Also assign speakers to words if available
	if len(transcript.Words) > 0 {
		mergedTranscript.Words = make([]interfaces.TranscriptWord, len(transcript.Words))
		copy(mergedTranscript.Words, transcript.Words)
		
		for i := range mergedTranscript.Words {
			word := &mergedTranscript.Words[i]
			bestSpeaker := u.findBestSpeakerForSegment(word.Start, word.End, diarization.Segments)
			if bestSpeaker != "" {
				word.Speaker = &bestSpeaker
			}
		}
	}

	return &mergedTranscript
}

// findBestSpeakerForSegment finds the speaker with maximum overlap for a given time segment
func (u *UnifiedTranscriptionService) findBestSpeakerForSegment(start, end float64, diarizationSegments []interfaces.DiarizationSegment) string {
	maxOverlap := 0.0
	bestSpeaker := ""

	for _, diarSeg := range diarizationSegments {
		// Calculate overlap
		overlapStart := max(start, diarSeg.Start)
		overlapEnd := min(end, diarSeg.End)
		overlap := max(0, overlapEnd-overlapStart)

		if overlap > maxOverlap {
			maxOverlap = overlap
			bestSpeaker = diarSeg.Speaker
		}
	}

	return bestSpeaker
}

// saveTranscriptionResults saves the transcription results to the database
func (u *UnifiedTranscriptionService) saveTranscriptionResults(jobID string, result *interfaces.TranscriptResult) error {
	// Convert result to JSON string for database storage
	resultJSON, err := u.convertTranscriptResultToJSON(result)
	if err != nil {
		return fmt.Errorf("failed to convert result to JSON: %w", err)
	}

	// Update the job in the database
	if err := database.DB.Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Update("transcript", resultJSON).Error; err != nil {
		return fmt.Errorf("failed to update job transcript: %w", err)
	}

	logger.Info("Saved transcription results", "job_id", jobID, "text_length", len(result.Text))
	return nil
}

// convertTranscriptResultToJSON converts the interface result to the expected JSON format
func (u *UnifiedTranscriptionService) convertTranscriptResultToJSON(result *interfaces.TranscriptResult) (string, error) {
	// Convert to the format expected by the existing database schema
	legacyFormat := struct {
		Segments []struct {
			Start   float64 `json:"start"`
			End     float64 `json:"end"`
			Text    string  `json:"text"`
			Speaker *string `json:"speaker,omitempty"`
		} `json:"segments"`
		Word []struct {
			Start   float64 `json:"start"`
			End     float64 `json:"end"`
			Word    string  `json:"word"`
			Score   float64 `json:"score"`
			Speaker *string `json:"speaker,omitempty"`
		} `json:"word_segments,omitempty"`
		Language string `json:"language"`
		Text     string `json:"text"`
	}{
		Language: result.Language,
		Text:     result.Text,
	}

	// Convert segments
	legacyFormat.Segments = make([]struct {
		Start   float64 `json:"start"`
		End     float64 `json:"end"`
		Text    string  `json:"text"`
		Speaker *string `json:"speaker,omitempty"`
	}, len(result.Segments))

	for i, seg := range result.Segments {
		legacyFormat.Segments[i] = struct {
			Start   float64 `json:"start"`
			End     float64 `json:"end"`
			Text    string  `json:"text"`
			Speaker *string `json:"speaker,omitempty"`
		}{
			Start:   seg.Start,
			End:     seg.End,
			Text:    seg.Text,
			Speaker: seg.Speaker,
		}
	}

	// Convert words
	if len(result.Words) > 0 {
		legacyFormat.Word = make([]struct {
			Start   float64 `json:"start"`
			End     float64 `json:"end"`
			Word    string  `json:"word"`
			Score   float64 `json:"score"`
			Speaker *string `json:"speaker,omitempty"`
		}, len(result.Words))

		for i, word := range result.Words {
			legacyFormat.Word[i] = struct {
				Start   float64 `json:"start"`
				End     float64 `json:"end"`
				Word    string  `json:"word"`
				Score   float64 `json:"score"`
				Speaker *string `json:"speaker,omitempty"`
			}{
				Start:   word.Start,
				End:     word.End,
				Word:    word.Word,
				Score:   word.Score,
				Speaker: word.Speaker,
			}
		}
	}

	// Convert to JSON string
	jsonBytes, err := json.Marshal(legacyFormat)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// GetSupportedModels returns all supported models through the new architecture
func (u *UnifiedTranscriptionService) GetSupportedModels() map[string]interfaces.ModelCapabilities {
	return u.registry.GetAllCapabilities()
}

// GetModelStatus returns the status of all models
func (u *UnifiedTranscriptionService) GetModelStatus(ctx context.Context) map[string]bool {
	return u.registry.GetModelStatus(ctx)
}

// ValidateModelParameters validates parameters for a specific model
func (u *UnifiedTranscriptionService) ValidateModelParameters(modelID string, params map[string]interface{}) error {
	return u.registry.ValidateModelParameters(modelID, params)
}

// Helper functions
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}