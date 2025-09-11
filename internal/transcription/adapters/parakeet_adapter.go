package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

// ParakeetAdapter implements the TranscriptionAdapter interface for NVIDIA Parakeet
type ParakeetAdapter struct {
	*BaseAdapter
	envPath string
}

// NewParakeetAdapter creates a new Parakeet adapter
func NewParakeetAdapter() *ParakeetAdapter {
	envPath := "whisperx-env/parakeet"
	
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "parakeet",
		ModelFamily:        "nvidia_parakeet",
		DisplayName:        "NVIDIA Parakeet TDT 0.6B v3",
		Description:        "NVIDIA's Parakeet model for English transcription with timestamps",
		Version:            "0.6.3",
		SupportedLanguages: []string{"en"}, // English only
		SupportedFormats:   []string{"wav", "flac"},
		RequiresGPU:        false, // Can run on CPU but GPU recommended
		MemoryRequirement:  4096,  // 4GB recommended
		Features: map[string]bool{
			"timestamps":         true,
			"word_level":         true,
			"long_form":          true,
			"attention_context":  true,
			"high_quality":       true,
		},
		Metadata: map[string]string{
			"engine":      "nvidia_nemo",
			"framework":   "nemo_toolkit",
			"license":     "CC-BY-4.0",
			"language":    "english_only",
			"sample_rate": "16000",
			"format":      "16khz_mono_wav",
		},
	}

	schema := []interfaces.ParameterSchema{
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

		// Long-form audio settings (Parakeet specific)
		{
			Name:        "context_left",
			Type:        "int",
			Required:    false,
			Default:     256,
			Min:         &[]float64{0}[0],
			Max:         &[]float64{1024}[0],
			Description: "Left attention context size for long-form audio",
			Group:       "advanced",
		},
		{
			Name:        "context_right",
			Type:        "int",
			Required:    false,
			Default:     256,
			Min:         &[]float64{0}[0],
			Max:         &[]float64{1024}[0],
			Description: "Right attention context size for long-form audio",
			Group:       "advanced",
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

		// Note: include_confidence removed as it's not supported by Parakeet script
	}

	baseAdapter := NewBaseAdapter("parakeet", envPath, capabilities, schema)
	
	adapter := &ParakeetAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetSupportedModels returns the specific Parakeet model available
func (p *ParakeetAdapter) GetSupportedModels() []string {
	return []string{"parakeet-tdt-0.6b-v3"}
}

// PrepareEnvironment sets up the Parakeet environment
func (p *ParakeetAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Parakeet environment", "env_path", p.envPath)

	// Check if environment is already ready
	testCmd := exec.Command("uv", "run", "--native-tls", "--project", p.envPath, "python", "-c", "import nemo.collections.asr")
	if testCmd.Run() == nil {
		modelPath := filepath.Join(p.envPath, "parakeet-tdt-0.6b-v3.nemo")
		scriptPath := filepath.Join(p.envPath, "transcribe.py")
		
		// Check both model and script exist
		if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
			if _, err := os.Stat(scriptPath); err == nil {
				logger.Info("Parakeet environment already ready")
				p.initialized = true
				return nil
			} else {
				logger.Info("Parakeet model exists but script missing, recreating script")
			}
		} else {
			logger.Info("Parakeet model file missing or incomplete, redownloading")
		}
	} else {
		logger.Info("Parakeet environment not ready, setting up")
	}

	// Setup environment
	if err := p.setupParakeetEnvironment(); err != nil {
		return fmt.Errorf("failed to setup Parakeet environment: %w", err)
	}

	// Download model
	if err := p.downloadParakeetModel(); err != nil {
		return fmt.Errorf("failed to download Parakeet model: %w", err)
	}

	// Create transcription script
	if err := p.createTranscriptionScript(); err != nil {
		return fmt.Errorf("failed to create transcription script: %w", err)
	}

	p.initialized = true
	logger.Info("Parakeet environment prepared successfully")
	return nil
}

// setupParakeetEnvironment creates the Python environment for Parakeet
func (p *ParakeetAdapter) setupParakeetEnvironment() error {
	if err := os.MkdirAll(p.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create parakeet directory: %w", err)
	}

	// Create pyproject.toml
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
	pyprojectPath := filepath.Join(p.envPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing Parakeet dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = p.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// downloadParakeetModel downloads the Parakeet model file
func (p *ParakeetAdapter) downloadParakeetModel() error {
	modelFileName := "parakeet-tdt-0.6b-v3.nemo"
	modelPath := filepath.Join(p.envPath, modelFileName)

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		logger.Info("Parakeet model already exists", "path", modelPath, "size", stat.Size())
		return nil
	}

	logger.Info("Downloading Parakeet model", "path", modelPath)
	
	modelURL := "https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3/resolve/main/parakeet-tdt-0.6b-v3.nemo?download=true"
	
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
		return fmt.Errorf("failed to download Parakeet model: %w: %s", err, strings.TrimSpace(string(out)))
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

	logger.Info("Successfully downloaded Parakeet model", "size", stat.Size())
	return nil
}

// createTranscriptionScript creates the Python script for Parakeet transcription
func (p *ParakeetAdapter) createTranscriptionScript() error {
	scriptContent := `#!/usr/bin/env python3
"""
NVIDIA Parakeet transcription script with timestamp support.
"""

import argparse
import json
import sys
import os
from pathlib import Path
import nemo.collections.asr as nemo_asr


def transcribe_audio(
    audio_path: str,
    timestamps: bool = True,
    output_file: str = None,
    context_left: int = 256,
    context_right: int = 256,
    include_confidence: bool = True,
):
    """
    Transcribe audio using NVIDIA Parakeet model.
    """
    # Get the directory where this script is located
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(script_dir, "parakeet-tdt-0.6b-v3.nemo")
    
    print(f"Script directory: {script_dir}")
    print(f"Looking for model at: {model_path}")
    
    if not os.path.exists(model_path):
        print(f"Error during transcription: Can't find {model_path}")
        # List files in the directory to help debug
        try:
            files = os.listdir(script_dir)
            print(f"Files in {script_dir}: {files}")
        except Exception as e:
            print(f"Could not list directory: {e}")
        sys.exit(1)
    
    print(f"Loading NVIDIA Parakeet model from: {model_path}")
    asr_model = nemo_asr.models.ASRModel.restore_from(model_path)
    
    # Configure for long-form audio if context sizes are not default
    if context_left != 256 or context_right != 256:
        print(f"Configuring attention context: left={context_left}, right={context_right}")
        try:
            asr_model.change_attention_model(
                self_attention_model="rel_pos_local_attn",
                att_context_size=[context_left, context_right]
            )
            print("Long-form audio mode enabled")
        except Exception as e:
            print(f"Warning: Failed to configure attention model: {e}")
            print("Continuing with default attention settings")
    
    print(f"Transcribing: {audio_path}")
    
    if timestamps:
        output = asr_model.transcribe([audio_path], timestamps=True)
        
        # Extract text and timestamps
        result_data = output[0]
        text = result_data.text
        word_timestamps = result_data.timestamp.get("word", [])
        segment_timestamps = result_data.timestamp.get("segment", [])
        
        print(f"Transcription: {text}")
        
        # Prepare output data
        output_data = {
            "transcription": text,
            "language": "en",
            "word_timestamps": word_timestamps,
            "segment_timestamps": segment_timestamps,
            "audio_file": audio_path,
            "model": "parakeet-tdt-0.6b-v3",
            "context": {
                "left": context_left,
                "right": context_right
            }
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
        # Simple transcription without timestamps
        output = asr_model.transcribe([audio_path])
        text = output[0].text
        
        output_data = {
            "transcription": text,
            "language": "en", 
            "audio_file": audio_path,
            "model": "parakeet-tdt-0.6b-v3"
        }
        
        if output_file:
            with open(output_file, 'w', encoding='utf-8') as f:
                json.dump(output_data, f, indent=2, ensure_ascii=False)
            print(f"Results saved to: {output_file}")
        else:
            print(json.dumps(output_data, indent=2, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe audio using NVIDIA Parakeet model"
    )
    parser.add_argument("audio_file", help="Path to audio file")
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
        "--context-left", type=int, default=256,
        help="Left attention context size (default: 256)"
    )
    parser.add_argument(
        "--context-right", type=int, default=256,
        help="Right attention context size (default: 256)"
    )
    parser.add_argument(
        "--include-confidence", action="store_true", default=True,
        help="Include confidence scores"
    )
    parser.add_argument(
        "--no-confidence", dest="include_confidence", action="store_false",
        help="Exclude confidence scores"
    )
    
    args = parser.parse_args()
    
    # Validate input file
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)
    
    try:
        transcribe_audio(
            audio_path=args.audio_file,
            timestamps=args.timestamps,
            output_file=args.output,
            context_left=args.context_left,
            context_right=args.context_right,
            include_confidence=args.include_confidence,
        )
    except Exception as e:
        print(f"Error during transcription: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
`

	scriptPath := filepath.Join(p.envPath, "transcribe.py")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write transcription script: %w", err)
	}

	return nil
}

// Transcribe processes audio using Parakeet
func (p *ParakeetAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	p.LogProcessingStart(input, procCtx)
	defer func() {
		p.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := p.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := p.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer p.CleanupTempDirectory(tempDir)

	// Convert audio if needed (Parakeet requires 16kHz mono WAV)
	audioInput := input
	if p.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := p.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	// Build command arguments
	args, err := p.buildParakeetArgs(audioInput, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute Parakeet
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	logger.Info("Executing Parakeet command", "args", strings.Join(args, " "))
	
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("transcription was cancelled")
	}
	if err != nil {
		logger.Error("Parakeet execution failed", "output", string(output), "error", err)
		return nil, fmt.Errorf("Parakeet execution failed: %w", err)
	}

	// Parse result
	result, err := p.parseResult(tempDir, audioInput, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "parakeet-tdt-0.6b-v3"
	result.Metadata = p.CreateDefaultMetadata(params)

	logger.Info("Parakeet transcription completed", 
		"segments", len(result.Segments),
		"words", len(result.Words),
		"processing_time", result.ProcessingTime)

	return result, nil
}

// buildParakeetArgs builds the command arguments for Parakeet
func (p *ParakeetAdapter) buildParakeetArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFile := filepath.Join(tempDir, "result.json")
	
	scriptPath := filepath.Join(p.envPath, "transcribe.py")
	args := []string{
		"run", "--native-tls", "--project", p.envPath, "python", scriptPath,
		input.FilePath,
		"--output", outputFile,
	}

	// Add timestamps flag (Parakeet script supports --timestamps)
	if p.GetBoolParameter(params, "timestamps") {
		args = append(args, "--timestamps")
	}

	// Add context settings
	args = append(args, "--context-left", strconv.Itoa(p.GetIntParameter(params, "context_left")))
	args = append(args, "--context-right", strconv.Itoa(p.GetIntParameter(params, "context_right")))

	// Note: --include-confidence is not supported by Parakeet script, removed

	return args, nil
}

// parseResult parses the Parakeet output
func (p *ParakeetAdapter) parseResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.TranscriptResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")
	
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var parakeetResult struct {
		Transcription     string `json:"transcription"`
		Language          string `json:"language"`
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

	if err := json.Unmarshal(data, &parakeetResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Convert to standard format
	result := &interfaces.TranscriptResult{
		Text:       parakeetResult.Transcription,
		Language:   parakeetResult.Language,
		Segments:   make([]interfaces.TranscriptSegment, len(parakeetResult.SegmentTimestamps)),
		Words:      make([]interfaces.TranscriptWord, len(parakeetResult.WordTimestamps)),
		Confidence: 0.0, // Default confidence
	}

	// Convert segments
	for i, seg := range parakeetResult.SegmentTimestamps {
		result.Segments[i] = interfaces.TranscriptSegment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Segment,
		}
	}

	// Convert words
	for i, word := range parakeetResult.WordTimestamps {
		result.Words[i] = interfaces.TranscriptWord{
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
			Score: 1.0, // Parakeet doesn't provide word-level scores
		}
	}

	return result, nil
}

// GetEstimatedProcessingTime provides Parakeet-specific time estimation
func (p *ParakeetAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Parakeet is generally faster than WhisperX but slower than real-time
	baseTime := p.BaseAdapter.GetEstimatedProcessingTime(input)
	
	// Parakeet typically processes at about 30% of audio duration
	return time.Duration(float64(baseTime) * 1.5)
}

// init registers the Parakeet adapter
func init() {
	registry.RegisterTranscriptionAdapter("parakeet", NewParakeetAdapter())
}