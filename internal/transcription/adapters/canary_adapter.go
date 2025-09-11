package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

// CanaryAdapter implements the TranscriptionAdapter interface for NVIDIA Canary
type CanaryAdapter struct {
	*BaseAdapter
	envPath string
}

// NewCanaryAdapter creates a new Canary adapter
func NewCanaryAdapter() *CanaryAdapter {
	envPath := "whisperx-env/parakeet" // Shares environment with Parakeet
	
	capabilities := interfaces.ModelCapabilities{
		ModelID:     "canary",
		ModelFamily: "nvidia_canary",
		DisplayName: "NVIDIA Canary 1B v2",
		Description: "NVIDIA's multilingual Canary model with translation capabilities",
		Version:     "1.2.0",
		SupportedLanguages: []string{
			"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh",
			// Canary supports many more languages
		},
		SupportedFormats:   []string{"wav", "flac"},
		RequiresGPU:        false, // Can run on CPU but GPU strongly recommended
		MemoryRequirement:  8192,  // 8GB+ recommended for Canary
		Features: map[string]bool{
			"timestamps":     true,
			"word_level":     true,
			"multilingual":   true,
			"translation":    true,
			"high_quality":   true,
			"code_switching": true,
		},
		Metadata: map[string]string{
			"engine":         "nvidia_nemo",
			"framework":      "nemo_toolkit",
			"license":        "CC-BY-4.0",
			"multilingual":   "true",
			"sample_rate":    "16000",
			"format":         "16khz_mono_wav",
			"memory_warning": "requires_8gb_plus",
		},
	}

	schema := []interfaces.ParameterSchema{
		// Language settings
		{
			Name:        "source_lang",
			Type:        "string",
			Required:    false,
			Default:     "en",
			Options:     []string{"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"},
			Description: "Source language of the audio",
			Group:       "basic",
		},
		{
			Name:        "target_lang",
			Type:        "string", 
			Required:    false,
			Default:     "en",
			Options:     []string{"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"},
			Description: "Target language for transcription/translation",
			Group:       "basic",
		},
		{
			Name:        "task",
			Type:        "string",
			Required:    false,
			Default:     "transcribe",
			Options:     []string{"transcribe", "translate"},
			Description: "Task to perform: transcribe (same language) or translate",
			Group:       "basic",
		},

		// Core settings
		{
			Name:        "timestamps",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Include word and segment level timestamps",
			Group:       "basic",
		},
		{
			Name:        "output_format",
			Type:        "string",
			Required:    false,
			Default:     "json",
			Options:     []string{"json", "text"},
			Description: "Output format for results",
			Group:       "basic",
		},

		// Audio preprocessing
		{
			Name:        "auto_convert_audio",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Automatically convert audio to 16kHz mono WAV",
			Group:       "advanced",
		},

		// Performance settings
		{
			Name:        "batch_size",
			Type:        "int",
			Required:    false,
			Default:     1,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{8}[0],
			Description: "Batch size for processing (higher uses more memory)",
			Group:       "advanced",
		},

		// Output settings
		{
			Name:        "include_confidence",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Include confidence scores in output",
			Group:       "advanced",
		},
		{
			Name:        "preserve_formatting",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Preserve punctuation and capitalization",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("canary", envPath, capabilities, schema)
	
	adapter := &CanaryAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetSupportedModels returns the specific Canary model available
func (c *CanaryAdapter) GetSupportedModels() []string {
	return []string{"canary-1b-v2"}
}

// PrepareEnvironment sets up the Canary environment (shared with Parakeet)
func (c *CanaryAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Canary environment", "env_path", c.envPath)

	// Check if environment is already ready
	testCmd := exec.Command("uv", "run", "--native-tls", "--project", c.envPath, "python", "-c", "import nemo.collections.asr")
	if testCmd.Run() == nil {
		modelPath := filepath.Join(c.envPath, "canary-1b-v2.nemo")
		if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
			logger.Info("Canary environment already ready")
			c.initialized = true
			return nil
		}
	}

	// Setup environment (reuse Parakeet setup since they share the same environment)
	if err := c.setupCanaryEnvironment(); err != nil {
		return fmt.Errorf("failed to setup Canary environment: %w", err)
	}

	// Download model
	if err := c.downloadCanaryModel(); err != nil {
		return fmt.Errorf("failed to download Canary model: %w", err)
	}

	// Create transcription script
	if err := c.createTranscriptionScript(); err != nil {
		return fmt.Errorf("failed to create transcription script: %w", err)
	}

	c.initialized = true
	logger.Info("Canary environment prepared successfully")
	return nil
}

// setupCanaryEnvironment creates the Python environment (shared with Parakeet)
func (c *CanaryAdapter) setupCanaryEnvironment() error {
	if err := os.MkdirAll(c.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create canary directory: %w", err)
	}

	// Check if pyproject.toml already exists from Parakeet setup
	pyprojectPath := filepath.Join(c.envPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		logger.Info("Environment already configured by Parakeet")
		return nil
	}

	// Create pyproject.toml (same as Parakeet since they share environment)
	pyprojectContent := `[project]
name = "parakeet-transcription"
version = "0.1.0"
description = "Audio transcription using NVIDIA Parakeet models"
requires-python = ">=3.11"
dependencies = [
    "nemo-toolkit[asr]",
    "torch",
    "torchaudio",
    "librosa",
    "soundfile",
    "ml-dtypes>=0.3.1,<0.5.0",
    "onnx>=1.15.0,<1.18.0",
]

[tool.uv.sources]
nemo-toolkit = { git = "https://github.com/NVIDIA/NeMo.git" }
`
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing Canary dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = c.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// downloadCanaryModel downloads the Canary model file
func (c *CanaryAdapter) downloadCanaryModel() error {
	modelFileName := "canary-1b-v2.nemo"
	modelPath := filepath.Join(c.envPath, modelFileName)

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		logger.Info("Canary model already exists", "path", modelPath, "size", stat.Size())
		return nil
	}

	logger.Info("Downloading Canary model", "path", modelPath)
	
	modelURL := "https://huggingface.co/nvidia/canary-1b-v2/resolve/main/canary-1b-v2.nemo?download=true"
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	tempPath := modelPath + ".tmp"
	os.Remove(tempPath)

	cmd := exec.CommandContext(ctx, "curl",
		"-L", "--progress-bar", "--create-dirs",
		"-o", tempPath, modelURL)

	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to download Canary model: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if err := os.Rename(tempPath, modelPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to move downloaded model: %w", err)
	}

	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("downloaded model file not found: %w", err)
	}
	if stat.Size() < 1024*1024 {
		return fmt.Errorf("downloaded model file appears incomplete (size: %d bytes)", stat.Size())
	}

	logger.Info("Successfully downloaded Canary model", "size", stat.Size())
	return nil
}

// createTranscriptionScript creates the Python script for Canary transcription
func (c *CanaryAdapter) createTranscriptionScript() error {
	scriptPath := filepath.Join(c.envPath, "canary_transcribe.py")
	
	// Check if script already exists
	if _, err := os.Stat(scriptPath); err == nil {
		return nil
	}

	scriptContent := `#!/usr/bin/env python3
"""
NVIDIA Canary multilingual transcription and translation script.
"""

import argparse
import json
import sys
import os
from pathlib import Path
import nemo.collections.asr as nemo_asr


def transcribe_audio(
    audio_path: str,
    source_lang: str = "en",
    target_lang: str = "en", 
    task: str = "transcribe",
    timestamps: bool = True,
    output_file: str = None,
    include_confidence: bool = True,
    preserve_formatting: bool = True,
):
    """
    Transcribe or translate audio using NVIDIA Canary model.
    """
    # Get the directory where this script is located
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(script_dir, "canary-1b-v2.nemo")
    
    if not os.path.exists(model_path):
        print(f"Error: Model file not found: {model_path}")
        sys.exit(1)
    
    print(f"Loading NVIDIA Canary model from: {model_path}")
    asr_model = nemo_asr.models.ASRModel.restore_from(model_path)
    
    print(f"Processing: {audio_path}")
    print(f"Task: {task}")
    print(f"Source language: {source_lang}")
    print(f"Target language: {target_lang}")
    
    if timestamps:
        if task == "translate" and source_lang != target_lang:
            # Translation with timestamps
            output = asr_model.transcribe(
                [audio_path], 
                source_lang=source_lang,
                target_lang=target_lang,
                timestamps=True
            )
        else:
            # Transcription with timestamps
            output = asr_model.transcribe(
                [audio_path],
                source_lang=source_lang,
                target_lang=target_lang,
                timestamps=True
            )
        
        # Extract text and timestamps
        result_data = output[0]
        text = result_data.text
        word_timestamps = result_data.timestamp.get("word", [])
        segment_timestamps = result_data.timestamp.get("segment", [])
        
        print(f"Result: {text}")
        
        # Prepare output data
        output_data = {
            "transcription": text,
            "source_language": source_lang,
            "target_language": target_lang,
            "task": task,
            "word_timestamps": word_timestamps,
            "segment_timestamps": segment_timestamps,
            "audio_file": audio_path,
            "model": "canary-1b-v2"
        }
        
        if include_confidence:
            # Add confidence scores if available
            if hasattr(result_data, 'confidence') and result_data.confidence:
                output_data["confidence"] = result_data.confidence
        
        # Save to file
        if output_file:
            with open(output_file, 'w', encoding='utf-8') as f:
                json.dump(output_data, f, indent=2, ensure_ascii=False)
            print(f"Results saved to: {output_file}")
        else:
            print(json.dumps(output_data, indent=2, ensure_ascii=False))
    
    else:
        # Simple transcription/translation without timestamps
        if task == "translate" and source_lang != target_lang:
            output = asr_model.transcribe(
                [audio_path],
                source_lang=source_lang,
                target_lang=target_lang
            )
        else:
            output = asr_model.transcribe(
                [audio_path],
                source_lang=source_lang,
                target_lang=target_lang
            )
        
        text = output[0].text
        
        output_data = {
            "transcription": text,
            "source_language": source_lang,
            "target_language": target_lang,
            "task": task,
            "audio_file": audio_path,
            "model": "canary-1b-v2"
        }
        
        if output_file:
            with open(output_file, 'w', encoding='utf-8') as f:
                json.dump(output_data, f, indent=2, ensure_ascii=False)
            print(f"Results saved to: {output_file}")
        else:
            print(json.dumps(output_data, indent=2, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe or translate audio using NVIDIA Canary model"
    )
    parser.add_argument("audio_file", help="Path to audio file")
    parser.add_argument(
        "--source-lang", default="en",
        choices=["en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"],
        help="Source language (default: en)"
    )
    parser.add_argument(
        "--target-lang", default="en",
        choices=["en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"],
        help="Target language (default: en)"
    )
    parser.add_argument(
        "--task", choices=["transcribe", "translate"], default="transcribe",
        help="Task to perform (default: transcribe)"
    )
    parser.add_argument(
        "--timestamps", action="store_true", default=True,
        help="Include word and segment level timestamps"
    )
    parser.add_argument(
        "--no-timestamps", dest="timestamps", action="store_false",
        help="Disable timestamps"
    )
    parser.add_argument(
        "--output", "-o", help="Output file path"
    )
    parser.add_argument(
        "--include-confidence", action="store_true", default=True,
        help="Include confidence scores"
    )
    parser.add_argument(
        "--no-confidence", dest="include_confidence", action="store_false",
        help="Exclude confidence scores"
    )
    parser.add_argument(
        "--preserve-formatting", action="store_true", default=True,
        help="Preserve punctuation and capitalization"
    )
    
    args = parser.parse_args()
    
    # Validate input file
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)
    
    try:
        transcribe_audio(
            audio_path=args.audio_file,
            source_lang=args.source_lang,
            target_lang=args.target_lang,
            task=args.task,
            timestamps=args.timestamps,
            output_file=args.output,
            include_confidence=args.include_confidence,
            preserve_formatting=args.preserve_formatting,
        )
    except Exception as e:
        print(f"Error during transcription: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write transcription script: %w", err)
	}

	return nil
}

// Transcribe processes audio using Canary
func (c *CanaryAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	c.LogProcessingStart(input, procCtx)
	defer func() {
		c.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := c.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := c.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := c.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer c.CleanupTempDirectory(tempDir)

	// Convert audio if needed (Canary requires 16kHz mono WAV)
	audioInput := input
	if c.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := c.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	// Build command arguments
	args, err := c.buildCanaryArgs(audioInput, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute Canary
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	logger.Info("Executing Canary command", "args", strings.Join(args, " "))
	
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("transcription was cancelled")
	}
	if err != nil {
		logger.Error("Canary execution failed", "output", string(output), "error", err)
		return nil, fmt.Errorf("Canary execution failed: %w", err)
	}

	// Parse result
	result, err := c.parseResult(tempDir, audioInput, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "canary-1b-v2"
	result.Metadata = c.CreateDefaultMetadata(params)

	logger.Info("Canary transcription completed", 
		"segments", len(result.Segments),
		"words", len(result.Words),
		"processing_time", result.ProcessingTime,
		"task", c.GetStringParameter(params, "task"))

	return result, nil
}

// buildCanaryArgs builds the command arguments for Canary
func (c *CanaryAdapter) buildCanaryArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFile := filepath.Join(tempDir, "result.json")
	
	scriptPath := filepath.Join(c.envPath, "canary_transcribe.py")
	args := []string{
		"run", "--native-tls", "--project", c.envPath, "python", scriptPath,
		input.FilePath,
		"--output", outputFile,
	}

	// Add language settings
	args = append(args, "--source-lang", c.GetStringParameter(params, "source_lang"))
	args = append(args, "--target-lang", c.GetStringParameter(params, "target_lang"))
	args = append(args, "--task", c.GetStringParameter(params, "task"))

	// Add timestamps flag
	if c.GetBoolParameter(params, "timestamps") {
		args = append(args, "--timestamps")
	} else {
		args = append(args, "--no-timestamps")
	}

	// Add confidence flag
	if c.GetBoolParameter(params, "include_confidence") {
		args = append(args, "--include-confidence")
	} else {
		args = append(args, "--no-confidence")
	}

	// Add formatting flag
	if c.GetBoolParameter(params, "preserve_formatting") {
		args = append(args, "--preserve-formatting")
	}

	return args, nil
}

// parseResult parses the Canary output
func (c *CanaryAdapter) parseResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.TranscriptResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")
	
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var canaryResult struct {
		Transcription     string `json:"transcription"`
		SourceLanguage    string `json:"source_language"`
		TargetLanguage    string `json:"target_language"`
		Task              string `json:"task"`
		WordTimestamps    []struct {
			Word        string  `json:"word"`
			StartOffset int     `json:"start_offset"`
			EndOffset   int     `json:"end_offset"`
			Start       float64 `json:"start"`
			End         float64 `json:"end"`
		} `json:"word_timestamps"`
		SegmentTimestamps []struct {
			Segment     string  `json:"segment"`
			StartOffset int     `json:"start_offset"`
			EndOffset   int     `json:"end_offset"`
			Start       float64 `json:"start"`
			End         float64 `json:"end"`
		} `json:"segment_timestamps"`
		Confidence interface{} `json:"confidence,omitempty"`
	}

	if err := json.Unmarshal(data, &canaryResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Determine the language for the result
	resultLanguage := canaryResult.TargetLanguage
	if canaryResult.Task == "transcribe" {
		resultLanguage = canaryResult.SourceLanguage
	}

	// Convert to standard format
	result := &interfaces.TranscriptResult{
		Text:       canaryResult.Transcription,
		Language:   resultLanguage,
		Segments:   make([]interfaces.TranscriptSegment, len(canaryResult.SegmentTimestamps)),
		Words:      make([]interfaces.TranscriptWord, len(canaryResult.WordTimestamps)),
		Confidence: 0.0, // Default confidence
	}

	// Convert segments
	for i, seg := range canaryResult.SegmentTimestamps {
		result.Segments[i] = interfaces.TranscriptSegment{
			Start:    seg.Start,
			End:      seg.End,
			Text:     seg.Segment,
			Language: &resultLanguage,
		}
	}

	// Convert words
	for i, word := range canaryResult.WordTimestamps {
		result.Words[i] = interfaces.TranscriptWord{
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
			Score: 1.0, // Canary doesn't provide word-level scores
		}
	}

	return result, nil
}

// GetEstimatedProcessingTime provides Canary-specific time estimation
func (c *CanaryAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Canary is typically slower than Parakeet due to its multilingual capabilities
	baseTime := c.BaseAdapter.GetEstimatedProcessingTime(input)
	
	// Canary typically processes at about 40-50% of audio duration
	return time.Duration(float64(baseTime) * 2.0)
}

// init registers the Canary adapter
func init() {
	registry.RegisterTranscriptionAdapter("canary", NewCanaryAdapter())
}