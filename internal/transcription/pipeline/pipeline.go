package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

// ProcessingPipeline handles the full processing workflow with preprocessing
type ProcessingPipeline struct {
	preprocessors  []interfaces.Preprocessor
	postprocessors []interfaces.Postprocessor
}

// NewProcessingPipeline creates a new processing pipeline
func NewProcessingPipeline() *ProcessingPipeline {
	pipeline := &ProcessingPipeline{
		preprocessors:  make([]interfaces.Preprocessor, 0),
		postprocessors: make([]interfaces.Postprocessor, 0),
	}

	// Register default preprocessors
	pipeline.RegisterPreprocessor(&AudioFormatPreprocessor{})

	return pipeline
}

// RegisterPreprocessor adds a preprocessor to the pipeline
func (p *ProcessingPipeline) RegisterPreprocessor(preprocessor interfaces.Preprocessor) {
	p.preprocessors = append(p.preprocessors, preprocessor)
}

// RegisterPostprocessor adds a postprocessor to the pipeline
func (p *ProcessingPipeline) RegisterPostprocessor(postprocessor interfaces.Postprocessor) {
	p.postprocessors = append(p.postprocessors, postprocessor)
}

// ProcessAudio applies all applicable preprocessors to the audio input
func (p *ProcessingPipeline) ProcessAudio(ctx context.Context, input interfaces.AudioInput, capabilities interfaces.ModelCapabilities) (interfaces.AudioInput, error) {
	currentInput := input

	for _, preprocessor := range p.preprocessors {
		if preprocessor.AppliesTo(capabilities) {
			logger.Info("Applying preprocessor", "type", fmt.Sprintf("%T", preprocessor))
			processedInput, err := preprocessor.Process(ctx, currentInput)
			if err != nil {
				logger.Warn("Preprocessor failed, continuing with original input", "error", err)
				continue
			}
			currentInput = processedInput
		}
	}

	return currentInput, nil
}

// AudioFormatPreprocessor converts audio to required formats
type AudioFormatPreprocessor struct{}

// AppliesTo checks if this preprocessor should be used for the given model
func (a *AudioFormatPreprocessor) AppliesTo(capabilities interfaces.ModelCapabilities) bool {
	// Apply to all models for consistent audio format (mono 16kHz)
	return true
}

// GetRequiredFormats returns the output formats this preprocessor can produce
func (a *AudioFormatPreprocessor) GetRequiredFormats() []string {
	return []string{"wav"}
}

// Process converts audio to the required format
func (a *AudioFormatPreprocessor) Process(ctx context.Context, input interfaces.AudioInput) (interfaces.AudioInput, error) {
	// Check if conversion is needed
	requiredFormat := "wav"
	requiredSampleRate := 16000
	requiredChannels := 1

	if strings.ToLower(input.Format) == requiredFormat &&
		input.SampleRate == requiredSampleRate &&
		input.Channels == requiredChannels {
		// No conversion needed
		return input, nil
	}

	logger.Info("Converting audio format",
		"from_format", input.Format,
		"to_format", requiredFormat,
		"from_sample_rate", input.SampleRate,
		"to_sample_rate", requiredSampleRate,
		"from_channels", input.Channels,
		"to_channels", requiredChannels)

	// Create output path
	outputPath := strings.TrimSuffix(input.FilePath, filepath.Ext(input.FilePath)) + "_converted.wav"

	// Build FFmpeg command
	args := []string{
		"-i", input.FilePath,
		"-ar", strconv.Itoa(requiredSampleRate),
		"-ac", strconv.Itoa(requiredChannels),
		"-c:a", "pcm_s16le",
		"-y", // Overwrite output file
		outputPath,
	}

	// Execute FFmpeg
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("FFmpeg conversion failed", "output", string(output), "error", err)
		return input, fmt.Errorf("audio conversion failed: %w", err)
	}

	// Create new audio input
	convertedInput := interfaces.AudioInput{
		FilePath:     outputPath,
		Format:       requiredFormat,
		SampleRate:   requiredSampleRate,
		Channels:     requiredChannels,
		Duration:     input.Duration, // Preserve duration
		Size:         0,              // Will be set when file is read
		Metadata:     input.Metadata,
		TempFilePath: outputPath, // Mark as temporary
	}

	// Get file size
	if stat, err := os.Stat(outputPath); err == nil {
		convertedInput.Size = stat.Size()
	}

	logger.Info("Audio conversion completed",
		"output_path", outputPath,
		"output_size", convertedInput.Size)

	return convertedInput, nil
}

// VoiceActivityDetectionPreprocessor applies VAD preprocessing
type VoiceActivityDetectionPreprocessor struct{}

// AppliesTo checks if this preprocessor should be used
func (v *VoiceActivityDetectionPreprocessor) AppliesTo(capabilities interfaces.ModelCapabilities) bool {
	// Apply to models that benefit from VAD preprocessing
	return capabilities.Features["vad"]
}

// GetRequiredFormats returns the output formats this preprocessor can produce
func (v *VoiceActivityDetectionPreprocessor) GetRequiredFormats() []string {
	return []string{"wav", "mp3", "flac"}
}

// Process applies voice activity detection preprocessing
func (v *VoiceActivityDetectionPreprocessor) Process(ctx context.Context, input interfaces.AudioInput) (interfaces.AudioInput, error) {
	// For now, this is a placeholder
	// In a real implementation, this would apply VAD to remove silence
	logger.Info("VAD preprocessing (placeholder)", "file", input.FilePath)
	return input, nil
}

// NoiseReductionPreprocessor applies noise reduction
type NoiseReductionPreprocessor struct{}

// AppliesTo checks if this preprocessor should be used
func (n *NoiseReductionPreprocessor) AppliesTo(capabilities interfaces.ModelCapabilities) bool {
	// Apply to models that would benefit from noise reduction
	return capabilities.Features["high_quality"]
}

// GetRequiredFormats returns the output formats this preprocessor can produce
func (n *NoiseReductionPreprocessor) GetRequiredFormats() []string {
	return []string{"wav"}
}

// Process applies noise reduction preprocessing
func (n *NoiseReductionPreprocessor) Process(ctx context.Context, input interfaces.AudioInput) (interfaces.AudioInput, error) {
	// For now, this is a placeholder
	// In a real implementation, this would apply noise reduction using FFmpeg or other tools
	logger.Info("Noise reduction preprocessing (placeholder)", "file", input.FilePath)
	return input, nil
}

// TextPostprocessor handles transcription result post-processing
type TextPostprocessor struct{}

// ProcessTranscript processes transcription results
func (t *TextPostprocessor) ProcessTranscript(ctx context.Context, result *interfaces.TranscriptResult, params map[string]interface{}) (*interfaces.TranscriptResult, error) {
	// Apply text cleaning, formatting, etc.
	logger.Info("Post-processing transcript", "segments", len(result.Segments))

	// Example post-processing: trim whitespace from segments
	for i := range result.Segments {
		result.Segments[i].Text = strings.TrimSpace(result.Segments[i].Text)
	}

	return result, nil
}

// ProcessDiarization processes diarization results
func (t *TextPostprocessor) ProcessDiarization(ctx context.Context, result *interfaces.DiarizationResult, params map[string]interface{}) (*interfaces.DiarizationResult, error) {
	// Apply diarization result cleaning, speaker merging, etc.
	logger.Info("Post-processing diarization", "segments", len(result.Segments))
	return result, nil
}

// AppliesTo determines if this postprocessor should be used
func (t *TextPostprocessor) AppliesTo(capabilities interfaces.ModelCapabilities, params map[string]interface{}) bool {
	return true // Always apply text post-processing
}
