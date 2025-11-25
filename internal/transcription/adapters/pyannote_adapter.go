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
	"scriberr/pkg/logger"
)

// PyAnnoteAdapter implements the DiarizationAdapter interface for PyAnnote
type PyAnnoteAdapter struct {
	*BaseAdapter
	envPath string
}

// NewPyAnnoteAdapter creates a new PyAnnote diarization adapter
func NewPyAnnoteAdapter(envPath string) *PyAnnoteAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "pyannote",
		ModelFamily:        "pyannote",
		DisplayName:        "PyAnnote Speaker Diarization 3.1",
		Description:        "PyAnnote audio speaker diarization with configurable speaker constraints",
		Version:            "3.1.0",
		SupportedLanguages: []string{"*"}, // Language-agnostic
		SupportedFormats:   []string{"wav", "mp3", "flac", "m4a", "ogg"},
		RequiresGPU:        false, // Optional GPU support
		MemoryRequirement:  2048,  // 2GB recommended
		Features: map[string]bool{
			"speaker_detection":   true,
			"speaker_constraints": true,
			"confidence_scores":   true,
			"rttm_output":         true,
			"flexible_speakers":   true,
		},
		Metadata: map[string]string{
			"engine":    "pyannote_audio",
			"framework": "pytorch",
			"license":   "MIT",
			"requires":  "huggingface_token",
			"model_hub": "huggingface",
		},
	}

	schema := []interfaces.ParameterSchema{
		// Core settings
		{
			Name:        "hf_token",
			Type:        "string",
			Required:    true,
			Default:     nil,
			Description: "HuggingFace token for model access (required)",
			Group:       "basic",
		},
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     "pyannote/speaker-diarization-3.1",
			Options:     []string{"pyannote/speaker-diarization-3.1", "pyannote/speaker-diarization-3.0"},
			Description: "PyAnnote model to use",
			Group:       "basic",
		},

		// Speaker constraints
		{
			Name:        "min_speakers",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{20}[0],
			Description: "Minimum number of speakers",
			Group:       "basic",
		},
		{
			Name:        "max_speakers",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{20}[0],
			Description: "Maximum number of speakers",
			Group:       "basic",
		},

		// Output settings
		{
			Name:        "output_format",
			Type:        "string",
			Required:    false,
			Default:     "rttm",
			Options:     []string{"rttm", "json"},
			Description: "Output format for diarization results",
			Group:       "advanced",
		},

		// Performance settings
		{
			Name:        "use_auth_token",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Use authentication token for model access",
			Group:       "advanced",
		},
		{
			Name:        "device",
			Type:        "string",
			Required:    false,
			Default:     "cpu",
			Options:     []string{"cpu", "cuda", "mps"},
			Description: "Device to use for computation (cpu, cuda for NVIDIA GPUs, mps for Apple Silicon)",
			Group:       "advanced",
		},

		// Quality settings
		{
			Name:        "segmentation_onset",
			Type:        "float",
			Required:    false,
			Default:     0.5,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "Voice activity detection onset threshold",
			Group:       "advanced",
		},
		{
			Name:        "segmentation_offset",
			Type:        "float",
			Required:    false,
			Default:     0.363,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "Voice activity detection offset threshold",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("pyannote", envPath, capabilities, schema)

	adapter := &PyAnnoteAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetMaxSpeakers returns the maximum number of speakers PyAnnote can handle
func (p *PyAnnoteAdapter) GetMaxSpeakers() int {
	return 20 // Practical limit for PyAnnote
}

// GetMinSpeakers returns the minimum number of speakers PyAnnote requires
func (p *PyAnnoteAdapter) GetMinSpeakers() int {
	return 1
}

// PrepareEnvironment sets up the PyAnnote environment (shared with NVIDIA models)
func (p *PyAnnoteAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing PyAnnote environment", "env_path", p.envPath)

	// Check if PyAnnote is already available (using cache to speed up repeated checks)
	if CheckEnvironmentReady(p.envPath, "from pyannote.audio import Pipeline") {
		logger.Info("PyAnnote already available in environment")
		// Still ensure script exists
		if err := p.createDiarizationScript(); err != nil {
			return fmt.Errorf("failed to create diarization script: %w", err)
		}
		p.initialized = true
		return nil
	}

	// Check if the shared environment exists (created by NVIDIA adapters)
	pyprojectPath := filepath.Join(p.envPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err != nil {
		// Create environment if it doesn't exist
		if err := p.setupPyAnnoteEnvironment(); err != nil {
			return fmt.Errorf("failed to setup PyAnnote environment: %w", err)
		}
	} else {
		// Environment exists but PyAnnote not available - add PyAnnote to existing environment
		logger.Info("Adding PyAnnote to existing environment")
		if err := p.addPyAnnoteToEnvironment(); err != nil {
			return fmt.Errorf("failed to add PyAnnote to environment: %w", err)
		}
	}

	// Always ensure diarization script exists
	if err := p.createDiarizationScript(); err != nil {
		return fmt.Errorf("failed to create diarization script: %w", err)
	}

	// Verify PyAnnote is now available
	testCmd := exec.Command("uv", "run", "--native-tls", "--project", p.envPath, "python", "-c", "from pyannote.audio import Pipeline")
	if testCmd.Run() != nil {
		logger.Warn("PyAnnote environment test still failed after setup")
	}

	p.initialized = true
	logger.Info("PyAnnote environment prepared successfully")
	return nil
}

// setupPyAnnoteEnvironment creates the Python environment if it doesn't exist
func (p *PyAnnoteAdapter) setupPyAnnoteEnvironment() error {
	if err := os.MkdirAll(p.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create pyannote directory: %w", err)
	}

	// Create pyproject.toml (same as NVIDIA models but with pyannote.audio added)
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
    "pyannote.audio"
]

[tool.uv.sources]
nemo-toolkit = { git = "https://github.com/NVIDIA/NeMo.git" }
`
	pyprojectPath := filepath.Join(p.envPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing PyAnnote dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = p.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// addPyAnnoteToEnvironment adds pyannote.audio to an existing environment
func (p *PyAnnoteAdapter) addPyAnnoteToEnvironment() error {
	// Read existing pyproject.toml
	pyprojectPath := filepath.Join(p.envPath, "pyproject.toml")
	data, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return fmt.Errorf("failed to read pyproject.toml: %w", err)
	}

	content := string(data)
	logger.Info("Current pyproject.toml content", "content", content)

	// Check if pyannote.audio is already in dependencies
	if strings.Contains(content, "pyannote.audio") {
		logger.Info("pyannote.audio already in dependencies, running sync")
	} else {
		// Instead of complex string manipulation, let's recreate the file with pyannote.audio
		logger.Info("Adding pyannote.audio to dependencies by recreating pyproject.toml")

		// Create updated pyproject.toml with pyannote.audio included
		updatedContent := `[project]
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
    "pyannote.audio"
]

[tool.uv.sources]
nemo-toolkit = { git = "https://github.com/NVIDIA/NeMo.git" }
`

		// Write updated pyproject.toml
		if err := os.WriteFile(pyprojectPath, []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to write updated pyproject.toml: %w", err)
		}
		logger.Info("Updated pyproject.toml with pyannote.audio")
	}

	// Run uv sync to install pyannote.audio
	logger.Info("Installing PyAnnote dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = p.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// createDiarizationScript creates the Python script for PyAnnote diarization
func (p *PyAnnoteAdapter) createDiarizationScript() error {
	scriptPath := filepath.Join(p.envPath, "pyannote_diarize.py")

	// Check if script already exists
	if _, err := os.Stat(scriptPath); err == nil {
		return nil
	}

	scriptContent := `#!/usr/bin/env python3
"""
PyAnnote speaker diarization script.
Processes audio files to identify and separate different speakers.
"""

import argparse
import json
import sys
import os
from pathlib import Path
from pyannote.audio import Pipeline


def diarize_audio(
    audio_path: str,
    output_file: str,
    hf_token: str,
    model: str = "pyannote/speaker-diarization-3.1",
    min_speakers: int = None,
    max_speakers: int = None,
    output_format: str = "rttm",
    device: str = "cpu"
):
    """
    Perform speaker diarization on audio file using PyAnnote.
    """
    print(f"Loading PyAnnote speaker diarization pipeline: {model}")
    
    try:
        # Initialize the diarization pipeline
        pipeline = Pipeline.from_pretrained(
            model,
            use_auth_token=hf_token
        )
        
        # Move to specified device
        if device == "cuda":
            try:
                import torch
                if torch.cuda.is_available():
                    pipeline = pipeline.to(torch.device("cuda"))
                    print("Using CUDA for diarization")
                else:
                    print("CUDA not available, falling back to CPU")
            except ImportError:
                print("PyTorch not available for CUDA, using CPU")
        elif device == "mps":
            try:
                import torch
                if torch.backends.mps.is_available():
                    pipeline = pipeline.to(torch.device("mps"))
                    print("Using MPS (Apple Silicon) for diarization")
                else:
                    print("MPS not available, falling back to CPU")
            except (ImportError, AttributeError):
                print("PyTorch MPS not available, using CPU")
        
        print("Pipeline loaded successfully")
    except Exception as e:
        print(f"Error loading pipeline: {e}")
        print("Make sure you have a valid Hugging Face token and have accepted the model's license")
        sys.exit(1)

    print(f"Processing audio file: {audio_path}")
    
    try:
        # Run diarization
        diarization_params = {}
        if min_speakers is not None:
            diarization_params["min_speakers"] = min_speakers
        if max_speakers is not None:
            diarization_params["max_speakers"] = max_speakers
            
        if diarization_params:
            print(f"Using speaker constraints: {diarization_params}")
            diarization = pipeline(audio_path, **diarization_params)
        else:
            print("Using automatic speaker detection")
            diarization = pipeline(audio_path)
        
        print(f"Diarization completed. Saving results to: {output_file}")
        
        if output_format == "rttm":
            # Save the diarization output to RTTM format
            with open(output_file, "w") as rttm:
                diarization.write_rttm(rttm)
        else:
            # Save as JSON format
            save_json_format(diarization, output_file, audio_path)
        
        # Print summary
        speakers = set()
        total_speech_time = 0.0
        
        for segment, track, speaker in diarization.itertracks(yield_label=True):
            speakers.add(speaker)
            total_speech_time += segment.duration
        
        print(f"\nDiarization Summary:")
        print(f"  Speakers detected: {len(speakers)}")
        print(f"  Speaker labels: {sorted(speakers)}")
        print(f"  Total speech time: {total_speech_time:.2f} seconds")
        print(f"  Output file saved: {output_file}")
        
    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


def save_json_format(diarization, output_file: str, audio_path: str):
    """Save diarization results in JSON format."""
    segments = []
    speakers = set()
    
    for segment, track, speaker in diarization.itertracks(yield_label=True):
        segments.append({
            "start": segment.start,
            "end": segment.end,
            "speaker": speaker,
            "confidence": 1.0,  # PyAnnote doesn't provide per-segment confidence
            "duration": segment.duration
        })
        speakers.add(speaker)
    
    # Sort segments by start time
    segments.sort(key=lambda x: x["start"])
    
    results = {
        "audio_file": audio_path,
        "model": "pyannote/speaker-diarization-3.1",
        "segments": segments,
        "speakers": sorted(speakers),
        "speaker_count": len(speakers),
        "total_duration": max(seg["end"] for seg in segments) if segments else 0,
        "processing_info": {
            "total_segments": len(segments),
            "total_speech_time": sum(seg["duration"] for seg in segments)
        }
    }
    
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)


def main():
    parser = argparse.ArgumentParser(
        description="Perform speaker diarization using PyAnnote.audio"
    )
    parser.add_argument(
        "audio_file",
        help="Path to audio file"
    )
    parser.add_argument(
        "--output", "-o",
        required=True,
        help="Output file path"
    )
    parser.add_argument(
        "--hf-token",
        required=True,
        help="Hugging Face access token"
    )
    parser.add_argument(
        "--model",
        default="pyannote/speaker-diarization-3.1",
        help="PyAnnote model to use"
    )
    parser.add_argument(
        "--min-speakers",
        type=int,
        help="Minimum number of speakers"
    )
    parser.add_argument(
        "--max-speakers",
        type=int,
        help="Maximum number of speakers"
    )
    parser.add_argument(
        "--output-format",
        choices=["rttm", "json"],
        default="rttm",
        help="Output format"
    )
    parser.add_argument(
        "--device",
        choices=["cpu", "cuda", "mps"],
        default="cpu",
        help="Device to use for computation"
    )

    args = parser.parse_args()

    # Validate input file
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

    # Validate speaker constraints
    if args.min_speakers is not None and args.min_speakers < 1:
        print("Error: min_speakers must be at least 1")
        sys.exit(1)
        
    if args.max_speakers is not None and args.max_speakers < 1:
        print("Error: max_speakers must be at least 1")
        sys.exit(1)
        
    if (args.min_speakers is not None and args.max_speakers is not None and 
        args.min_speakers > args.max_speakers):
        print("Error: min_speakers cannot be greater than max_speakers")
        sys.exit(1)

    # Create output directory if it doesn't exist
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    try:
        diarize_audio(
            audio_path=args.audio_file,
            output_file=args.output,
            hf_token=args.hf_token,
            model=args.model,
            min_speakers=args.min_speakers,
            max_speakers=args.max_speakers,
            output_format=args.output_format,
            device=args.device
        )
    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write diarization script: %w", err)
	}

	return nil
}

// Diarize processes audio using PyAnnote
func (p *PyAnnoteAdapter) Diarize(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
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

	// Check for required HF token
	hfToken := p.GetStringParameter(params, "hf_token")
	if hfToken == "" {
		return nil, fmt.Errorf("HuggingFace token is required for PyAnnote diarization")
	}

	// Create temporary directory
	tempDir, err := p.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer p.CleanupTempDirectory(tempDir)

	// Build command arguments
	args, err := p.buildPyAnnoteArgs(input, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute PyAnnote
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	// Setup log file
	logFile, err := os.OpenFile(filepath.Join(procCtx.OutputDirectory, "transcription.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Warn("Failed to create log file", "error", err)
	} else {
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	logger.Info("Executing PyAnnote command", "args", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("diarization was cancelled")
		}

		// Read tail of log file for context
		logPath := filepath.Join(procCtx.OutputDirectory, "transcription.log")
		logTail, readErr := p.ReadLogTail(logPath, 2048)
		if readErr != nil {
			logger.Warn("Failed to read log tail", "error", readErr)
		}

		logger.Error("PyAnnote execution failed", "error", err)
		return nil, fmt.Errorf("PyAnnote execution failed: %w\nLogs:\n%s", err, logTail)
	}

	// Parse result
	result, err := p.parseResult(tempDir, input, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = p.GetStringParameter(params, "model")
	result.Metadata = p.CreateDefaultMetadata(params)

	logger.Info("PyAnnote diarization completed",
		"segments", len(result.Segments),
		"speakers", result.SpeakerCount,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// buildPyAnnoteArgs builds the command arguments for PyAnnote
func (p *PyAnnoteAdapter) buildPyAnnoteArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFormat := p.GetStringParameter(params, "output_format")
	var outputFile string
	if outputFormat == "json" {
		outputFile = filepath.Join(tempDir, "result.json")
	} else {
		outputFile = filepath.Join(tempDir, "result.rttm")
	}

	scriptPath := filepath.Join(p.envPath, "pyannote_diarize.py")
	args := []string{
		"run", "--native-tls", "--project", p.envPath, "python", scriptPath,
		input.FilePath,
		"--output", outputFile,
		"--hf-token", p.GetStringParameter(params, "hf_token"),
	}

	// Add model
	if model := p.GetStringParameter(params, "model"); model != "" {
		args = append(args, "--model", model)
	}

	// Add speaker constraints
	if minSpeakers := p.GetIntParameter(params, "min_speakers"); minSpeakers > 0 {
		args = append(args, "--min-speakers", strconv.Itoa(minSpeakers))
	}
	if maxSpeakers := p.GetIntParameter(params, "max_speakers"); maxSpeakers > 0 {
		args = append(args, "--max-speakers", strconv.Itoa(maxSpeakers))
	}

	// Add output format
	args = append(args, "--output-format", outputFormat)

	// Add device
	if device := p.GetStringParameter(params, "device"); device != "" {
		args = append(args, "--device", device)
	}

	return args, nil
}

// parseResult parses the PyAnnote output
func (p *PyAnnoteAdapter) parseResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.DiarizationResult, error) {
	outputFormat := p.GetStringParameter(params, "output_format")

	if outputFormat == "json" {
		return p.parseJSONResult(tempDir)
	} else {
		return p.parseRTTMResult(tempDir, input)
	}
}

// parseJSONResult parses JSON format output
func (p *PyAnnoteAdapter) parseJSONResult(tempDir string) (*interfaces.DiarizationResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var pyannoteResult struct {
		AudioFile string `json:"audio_file"`
		Model     string `json:"model"`
		Segments  []struct {
			Start      float64 `json:"start"`
			End        float64 `json:"end"`
			Speaker    string  `json:"speaker"`
			Confidence float64 `json:"confidence"`
			Duration   float64 `json:"duration"`
		} `json:"segments"`
		Speakers      []string `json:"speakers"`
		SpeakerCount  int      `json:"speaker_count"`
		TotalDuration float64  `json:"total_duration"`
	}

	if err := json.Unmarshal(data, &pyannoteResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Convert to standard format
	result := &interfaces.DiarizationResult{
		Segments:     make([]interfaces.DiarizationSegment, len(pyannoteResult.Segments)),
		SpeakerCount: pyannoteResult.SpeakerCount,
		Speakers:     pyannoteResult.Speakers,
	}

	for i, seg := range pyannoteResult.Segments {
		result.Segments[i] = interfaces.DiarizationSegment{
			Start:      seg.Start,
			End:        seg.End,
			Speaker:    seg.Speaker,
			Confidence: seg.Confidence,
		}
	}

	return result, nil
}

// parseRTTMResult parses RTTM format output
func (p *PyAnnoteAdapter) parseRTTMResult(tempDir string, input interfaces.AudioInput) (*interfaces.DiarizationResult, error) {
	resultFile := filepath.Join(tempDir, "result.rttm")

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var segments []interfaces.DiarizationSegment
	speakers := make(map[string]bool)

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "SPEAKER") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}

		start, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			continue
		}

		duration, err := strconv.ParseFloat(parts[4], 64)
		if err != nil {
			continue
		}

		end := start + duration
		speaker := parts[7]
		speakers[speaker] = true

		segments = append(segments, interfaces.DiarizationSegment{
			Start:      start,
			End:        end,
			Speaker:    speaker,
			Confidence: 1.0, // RTTM doesn't include confidence scores
		})
	}

	// Convert speakers map to slice
	speakerList := make([]string, 0, len(speakers))
	for speaker := range speakers {
		speakerList = append(speakerList, speaker)
	}

	result := &interfaces.DiarizationResult{
		Segments:     segments,
		SpeakerCount: len(speakers),
		Speakers:     speakerList,
	}

	return result, nil
}

// GetEstimatedProcessingTime provides PyAnnote-specific time estimation
func (p *PyAnnoteAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// PyAnnote is typically faster than real-time for diarization
	baseTime := p.BaseAdapter.GetEstimatedProcessingTime(input)

	// PyAnnote typically processes at about 10-15% of audio duration
	return time.Duration(float64(baseTime) * 0.5)
}
