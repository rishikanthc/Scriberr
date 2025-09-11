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

// SortformerAdapter implements the DiarizationAdapter interface for NVIDIA Sortformer
type SortformerAdapter struct {
	*BaseAdapter
	envPath string
}

// NewSortformerAdapter creates a new NVIDIA Sortformer diarization adapter
func NewSortformerAdapter() *SortformerAdapter {
	envPath := "whisperx-env/parakeet" // Shares environment with NVIDIA models
	
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "sortformer",
		ModelFamily:        "nvidia_sortformer",
		DisplayName:        "NVIDIA Sortformer 4-Speaker v2",
		Description:        "NVIDIA's streaming Sortformer model optimized for 4-speaker diarization",
		Version:            "2.0.0",
		SupportedLanguages: []string{"*"}, // Language-agnostic
		SupportedFormats:   []string{"wav", "flac"},
		RequiresGPU:        false, // Optional GPU support
		MemoryRequirement:  3072,  // 3GB recommended
		Features: map[string]bool{
			"speaker_detection":    true,
			"streaming":            true,
			"optimized_4_speakers": true,
			"fast_processing":      true,
			"no_token_required":    true,
		},
		Metadata: map[string]string{
			"engine":        "nvidia_nemo",
			"framework":     "nemo_toolkit",
			"license":       "CC-BY-4.0",
			"optimization":  "4_speakers",
			"sample_rate":   "16000",
			"format":        "16khz_mono_wav",
			"no_auth":       "true",
		},
	}

	schema := []interfaces.ParameterSchema{
		// Core settings
		{
			Name:        "max_speakers",
			Type:        "int",
			Required:    false,
			Default:     4,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{8}[0],
			Description: "Maximum number of speakers (optimized for 4)",
			Group:       "basic",
		},
		{
			Name:        "batch_size",
			Type:        "int",
			Required:    false,
			Default:     1,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{4}[0],
			Description: "Batch size for processing",
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
			Name:        "device",
			Type:        "string",
			Required:    false,
			Default:     "auto",
			Options:     []string{"cpu", "cuda", "auto"},
			Description: "Device to use for computation",
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

		// Model-specific settings
		{
			Name:        "streaming_mode",
			Type:        "bool",
			Required:    false,
			Default:     false,
			Description: "Enable streaming processing mode",
			Group:       "advanced",
		},
		{
			Name:        "chunk_length_s",
			Type:        "float",
			Required:    false,
			Default:     30.0,
			Min:         &[]float64{5.0}[0],
			Max:         &[]float64{120.0}[0],
			Description: "Chunk length in seconds for streaming mode",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("sortformer", envPath, capabilities, schema)
	
	adapter := &SortformerAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetMaxSpeakers returns the maximum number of speakers Sortformer can handle
func (s *SortformerAdapter) GetMaxSpeakers() int {
	return 8 // Can handle more but optimized for 4
}

// GetMinSpeakers returns the minimum number of speakers Sortformer requires
func (s *SortformerAdapter) GetMinSpeakers() int {
	return 1
}

// PrepareEnvironment sets up the Sortformer environment (shared with NVIDIA models)
func (s *SortformerAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Sortformer environment", "env_path", s.envPath)

	// Check if environment is already ready
	testCmd := exec.Command("uv", "run", "--native-tls", "--project", s.envPath, "python", "-c", "from nemo.collections.asr.models import SortformerEncLabelModel")
	if testCmd.Run() == nil {
		modelPath := filepath.Join(s.envPath, "diar_streaming_sortformer_4spk-v2.nemo")
		if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
			scriptPath := filepath.Join(s.envPath, "sortformer_diarize.py")
			if _, err := os.Stat(scriptPath); err == nil {
				logger.Info("Sortformer environment already ready")
				s.initialized = true
				return nil
			}
		}
	}

	// Check if the shared environment exists (created by other NVIDIA adapters)
	pyprojectPath := filepath.Join(s.envPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err != nil {
		// Create environment if it doesn't exist
		if err := s.setupSortformerEnvironment(); err != nil {
			return fmt.Errorf("failed to setup Sortformer environment: %w", err)
		}
	}

	// Download model
	if err := s.downloadSortformerModel(); err != nil {
		return fmt.Errorf("failed to download Sortformer model: %w", err)
	}

	// Create diarization script
	if err := s.createDiarizationScript(); err != nil {
		return fmt.Errorf("failed to create diarization script: %w", err)
	}

	s.initialized = true
	logger.Info("Sortformer environment prepared successfully")
	return nil
}

// setupSortformerEnvironment creates the Python environment if it doesn't exist
func (s *SortformerAdapter) setupSortformerEnvironment() error {
	if err := os.MkdirAll(s.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create sortformer directory: %w", err)
	}

	// Create pyproject.toml (same as other NVIDIA models)
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
	pyprojectPath := filepath.Join(s.envPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing Sortformer dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = s.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// downloadSortformerModel downloads the Sortformer model file
func (s *SortformerAdapter) downloadSortformerModel() error {
	modelFileName := "diar_streaming_sortformer_4spk-v2.nemo"
	modelPath := filepath.Join(s.envPath, modelFileName)

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		logger.Info("Sortformer model already exists", "path", modelPath, "size", stat.Size())
		return nil
	}

	logger.Info("Downloading Sortformer model", "path", modelPath)
	
	modelURL := "https://huggingface.co/nvidia/diar_streaming_sortformer_4spk-v2/resolve/main/diar_streaming_sortformer_4spk-v2.nemo?download=true"
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	tempPath := modelPath + ".tmp"
	os.Remove(tempPath)

	cmd := exec.CommandContext(ctx, "curl",
		"-L", "-#", "--max-time", "1800",
		"-o", tempPath, modelURL)

	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to download Sortformer model: %w: %s", err, strings.TrimSpace(string(out)))
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

	logger.Info("Successfully downloaded Sortformer model", "size", stat.Size())
	return nil
}

// createDiarizationScript creates the Python script for Sortformer diarization
func (s *SortformerAdapter) createDiarizationScript() error {
	scriptPath := filepath.Join(s.envPath, "sortformer_diarize.py")
	
	// Check if script already exists
	if _, err := os.Stat(scriptPath); err == nil {
		return nil
	}

	scriptContent := `#!/usr/bin/env python3
"""
NVIDIA Sortformer speaker diarization script.
Uses diar_streaming_sortformer_4spk-v2 for optimized 4-speaker diarization.
"""

import argparse
import json
import sys
import os
from pathlib import Path
import torch

try:
    from nemo.collections.asr.models import SortformerEncLabelModel
except ImportError:
    print("Error: NeMo not found. Please install nemo_toolkit[asr]")
    sys.exit(1)


def diarize_audio(
    audio_path: str,
    output_file: str,
    batch_size: int = 1,
    device: str = None,
    max_speakers: int = 4,
    output_format: str = "rttm",
    streaming_mode: bool = False,
    chunk_length_s: float = 30.0,
):
    """
    Perform speaker diarization using NVIDIA's Sortformer model.
    """
    if device is None or device == "auto":
        device = "cuda" if torch.cuda.is_available() else "cpu"

    print(f"Using device: {device}")
    print(f"Loading NVIDIA Sortformer diarization model...")

    # Get the directory where this script is located
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(script_dir, "diar_streaming_sortformer_4spk-v2.nemo")

    try:
        if not os.path.exists(model_path):
            print(f"Error: Model file not found: {model_path}")
            print("Please ensure diar_streaming_sortformer_4spk-v2.nemo is in the same directory as this script")
            sys.exit(1)

        # Load from local file
        print(f"Loading model from local path: {model_path}")
        diar_model = SortformerEncLabelModel.restore_from(
            restore_path=model_path,
            map_location=device,
            strict=False,
        )

        # Switch to inference mode
        diar_model.eval()
        print("Model loaded successfully")

    except Exception as e:
        print(f"Error loading model: {e}")
        sys.exit(1)

    print(f"Processing audio file: {audio_path}")

    # Verify audio file exists
    if not os.path.exists(audio_path):
        print(f"Error: Audio file not found: {audio_path}")
        sys.exit(1)

    try:
        # Run diarization
        print(f"Running diarization with batch_size={batch_size}, max_speakers={max_speakers}")
        
        if streaming_mode:
            print(f"Using streaming mode with chunk_length_s={chunk_length_s}")
            # Note: Streaming mode implementation would go here
            # For now, use standard diarization
            predicted_segments = diar_model.diarize(audio=audio_path, batch_size=batch_size)
        else:
            predicted_segments = diar_model.diarize(audio=audio_path, batch_size=batch_size)

        print(f"Diarization completed. Found segments: {len(predicted_segments)}")

        # Process and save results
        save_results(predicted_segments, output_file, audio_path, output_format)

    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


def save_results(segments, output_file: str, audio_path: str, output_format: str):
    """
    Save diarization results to output file.
    Supports both JSON and RTTM formats based on output_format parameter.
    """
    output_path = Path(output_file)

    if output_format == "rttm":
        save_rttm_format(segments, output_file, audio_path)
    else:
        save_json_format(segments, output_file, audio_path)


def save_json_format(segments, output_file: str, audio_path: str):
    """Save results in JSON format."""
    results = {
        "audio_file": audio_path,
        "model": "nvidia/diar_streaming_sortformer_4spk-v2",
        "segments": [],
    }

    # Handle the case where segments is a list containing a single list of string entries
    if len(segments) == 1 and isinstance(segments[0], list):
        segments = segments[0]

    # Convert segments to JSON format
    speakers = set()
    for i, segment in enumerate(segments):
        try:
            # Handle different possible segment formats
            if isinstance(segment, str):
                # String format: "start end speaker_id"
                parts = segment.strip().split()
                if len(parts) >= 3:
                    segment_data = {
                        "start": float(parts[0]),
                        "end": float(parts[1]),
                        "speaker": str(parts[2]),
                        "duration": float(parts[1]) - float(parts[0]),
                        "confidence": 1.0,
                    }
                else:
                    print(f"Warning: Invalid string segment format: {segment}")
                    continue
            elif hasattr(segment, 'start') and hasattr(segment, 'end') and hasattr(segment, 'label'):
                # Standard pyannote-like format
                segment_data = {
                    "start": float(segment.start),
                    "end": float(segment.end),
                    "speaker": str(segment.label),
                    "duration": float(segment.end - segment.start),
                    "confidence": getattr(segment, 'confidence', 1.0),
                }
            elif isinstance(segment, (list, tuple)) and len(segment) >= 3:
                # List/tuple format: [start, end, speaker]
                segment_data = {
                    "start": float(segment[0]),
                    "end": float(segment[1]),
                    "speaker": str(segment[2]),
                    "duration": float(segment[1] - segment[0]),
                    "confidence": 1.0,
                }
            elif isinstance(segment, dict):
                # Dictionary format
                segment_data = {
                    "start": float(segment.get('start', 0)),
                    "end": float(segment.get('end', 0)),
                    "speaker": str(segment.get('speaker', segment.get('label', f'speaker_{i}'))),
                    "duration": float(segment.get('end', 0) - segment.get('start', 0)),
                    "confidence": float(segment.get('confidence', 1.0)),
                }
            else:
                # Fallback: try to extract attributes dynamically
                segment_data = {
                    "start": float(getattr(segment, 'start', 0)),
                    "end": float(getattr(segment, 'end', 0)),
                    "speaker": str(getattr(segment, 'label', getattr(segment, 'speaker', f'speaker_{i}'))),
                    "duration": float(getattr(segment, 'end', 0) - getattr(segment, 'start', 0)),
                    "confidence": float(getattr(segment, 'confidence', 1.0)),
                }

            results["segments"].append(segment_data)
            speakers.add(segment_data["speaker"])

        except Exception as e:
            print(f"Warning: Could not process segment {i}: {e}")
            print(f"Segment: {segment}")

    # Sort by start time
    if results["segments"]:
        results["segments"].sort(key=lambda x: x["start"])

    # Add summary statistics
    results["speakers"] = sorted(speakers)
    results["speaker_count"] = len(speakers)
    results["total_segments"] = len(results["segments"])
    results["total_duration"] = max(seg["end"] for seg in results["segments"]) if results["segments"] else 0

    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    print(f"Results saved to: {output_file}")
    print(f"Found {len(speakers)} speakers: {', '.join(sorted(speakers))}")


def save_rttm_format(segments, output_file: str, audio_path: str):
    """Save results in RTTM (Rich Transcription Time Marked) format."""
    audio_filename = Path(audio_path).stem
    speakers = set()

    # Handle the case where segments is a list containing a single list of string entries
    if len(segments) == 1 and isinstance(segments[0], list):
        segments = segments[0]

    with open(output_file, "w") as f:
        for i, segment in enumerate(segments):
            try:
                # Handle different possible segment formats
                if isinstance(segment, str):
                    # String format: "start end speaker_id"
                    parts = segment.strip().split()
                    if len(parts) >= 3:
                        start = float(parts[0])
                        end = float(parts[1])
                        speaker = str(parts[2])
                    else:
                        print(f"Warning: Invalid string segment format: {segment}")
                        continue
                elif hasattr(segment, 'start') and hasattr(segment, 'end') and hasattr(segment, 'label'):
                    # Standard pyannote-like format
                    start = float(segment.start)
                    end = float(segment.end)
                    speaker = str(segment.label)
                elif isinstance(segment, (list, tuple)) and len(segment) >= 3:
                    # List/tuple format: [start, end, speaker]
                    start = float(segment[0])
                    end = float(segment[1])
                    speaker = str(segment[2])
                elif isinstance(segment, dict):
                    # Dictionary format
                    start = float(segment.get('start', 0))
                    end = float(segment.get('end', 0))
                    speaker = str(segment.get('speaker', segment.get('label', f'speaker_{i}')))
                else:
                    # Fallback: try to extract attributes dynamically
                    start = float(getattr(segment, 'start', 0))
                    end = float(getattr(segment, 'end', 0))
                    speaker = str(getattr(segment, 'label', getattr(segment, 'speaker', f'speaker_{i}')))

                duration = end - start
                speakers.add(speaker)

                # RTTM format: SPEAKER <filename> <channel> <start> <duration> <NA> <NA> <speaker_id> <NA> <NA>
                line = f"SPEAKER {audio_filename} 1 {start:.3f} {duration:.3f} <NA> <NA> {speaker} <NA> <NA>\n"
                f.write(line)

            except Exception as e:
                print(f"Warning: Could not process segment {i} for RTTM: {e}")
                print(f"Segment: {segment}")

    print(f"RTTM results saved to: {output_file}")
    print(f"Found {len(speakers)} speakers: {', '.join(sorted(speakers))}")


def main():
    parser = argparse.ArgumentParser(
        description="Speaker diarization using NVIDIA Sortformer model (local model only)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    # Basic diarization with JSON output
    python sortformer_diarize.py samples/sample.wav output.json
    
    # Generate RTTM format output
    python sortformer_diarize.py samples/sample.wav output.rttm
    
    # Specify device and batch size
    python sortformer_diarize.py --device cuda --batch-size 2 samples/sample.wav output.json

Note: This script requires diar_streaming_sortformer_4spk-v2.nemo to be in the same directory.
        """,
    )

    parser.add_argument("audio_file", help="Path to input audio file (WAV, FLAC, etc.)")
    parser.add_argument("output_file", help="Path to output file (.json for JSON format, .rttm for RTTM format)")
    parser.add_argument("--batch-size", type=int, default=1, help="Batch size for processing (default: 1)")
    parser.add_argument("--device", choices=["cuda", "cpu", "auto"], default="auto", help="Device to use for inference (default: auto-detect)")
    parser.add_argument("--max-speakers", type=int, default=4, help="Maximum number of speakers (default: 4, optimized for this model)")
    parser.add_argument("--output-format", choices=["json", "rttm"], help="Output format (auto-detected from file extension if not specified)")
    parser.add_argument("--streaming", action="store_true", help="Enable streaming mode")
    parser.add_argument("--chunk-length-s", type=float, default=30.0, help="Chunk length in seconds for streaming mode (default: 30.0)")

    args = parser.parse_args()

    # Validate inputs
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

    # Auto-detect output format from file extension if not specified
    if args.output_format is None:
        if args.output_file.lower().endswith('.rttm'):
            output_format = "rttm"
        else:
            output_format = "json"
    else:
        output_format = args.output_format

    # Create output directory if it doesn't exist
    output_dir = Path(args.output_file).parent
    output_dir.mkdir(parents=True, exist_ok=True)

    device = None if args.device == "auto" else args.device

    # Run diarization
    diarize_audio(
        audio_path=args.audio_file,
        output_file=args.output_file,
        batch_size=args.batch_size,
        device=device,
        max_speakers=args.max_speakers,
        output_format=output_format,
        streaming_mode=args.streaming,
        chunk_length_s=args.chunk_length_s,
    )


if __name__ == "__main__":
    main()
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write diarization script: %w", err)
	}

	return nil
}

// Diarize processes audio using Sortformer
func (s *SortformerAdapter) Diarize(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	startTime := time.Now()
	s.LogProcessingStart(input, procCtx)
	defer func() {
		s.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := s.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := s.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := s.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer s.CleanupTempDirectory(tempDir)

	// Convert audio if needed (Sortformer requires 16kHz mono WAV)
	audioInput := input
	if s.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := s.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	// Build command arguments
	args, err := s.buildSortformerArgs(audioInput, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute Sortformer
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	logger.Info("Executing Sortformer command", "args", strings.Join(args, " "))
	
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("diarization was cancelled")
	}
	if err != nil {
		logger.Error("Sortformer execution failed", "output", string(output), "error", err)
		return nil, fmt.Errorf("Sortformer execution failed: %w", err)
	}

	// Parse result
	result, err := s.parseResult(tempDir, audioInput, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "diar_streaming_sortformer_4spk-v2"
	result.Metadata = s.CreateDefaultMetadata(params)

	logger.Info("Sortformer diarization completed", 
		"segments", len(result.Segments),
		"speakers", result.SpeakerCount,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// buildSortformerArgs builds the command arguments for Sortformer
func (s *SortformerAdapter) buildSortformerArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFormat := s.GetStringParameter(params, "output_format")
	var outputFile string
	if outputFormat == "json" {
		outputFile = filepath.Join(tempDir, "result.json")
	} else {
		outputFile = filepath.Join(tempDir, "result.rttm")
	}
	
	scriptPath := filepath.Join(s.envPath, "sortformer_diarize.py")
	args := []string{
		"run", "--native-tls", "--project", s.envPath, "python", scriptPath,
		input.FilePath,
		outputFile,
	}

	// Add batch size
	if batchSize := s.GetIntParameter(params, "batch_size"); batchSize > 0 {
		args = append(args, "--batch-size", strconv.Itoa(batchSize))
	}

	// Add device
	if device := s.GetStringParameter(params, "device"); device != "" {
		args = append(args, "--device", device)
	}

	// Add max speakers
	if maxSpeakers := s.GetIntParameter(params, "max_speakers"); maxSpeakers > 0 {
		args = append(args, "--max-speakers", strconv.Itoa(maxSpeakers))
	}

	// Add output format
	args = append(args, "--output-format", outputFormat)

	// Add streaming mode
	if s.GetBoolParameter(params, "streaming_mode") {
		args = append(args, "--streaming")
		if chunkLength := s.GetFloatParameter(params, "chunk_length_s"); chunkLength > 0 {
			args = append(args, "--chunk-length-s", fmt.Sprintf("%.1f", chunkLength))
		}
	}

	return args, nil
}

// parseResult parses the Sortformer output
func (s *SortformerAdapter) parseResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.DiarizationResult, error) {
	outputFormat := s.GetStringParameter(params, "output_format")
	
	if outputFormat == "json" {
		return s.parseJSONResult(tempDir)
	} else {
		return s.parseRTTMResult(tempDir, input)
	}
}

// parseJSONResult parses JSON format output
func (s *SortformerAdapter) parseJSONResult(tempDir string) (*interfaces.DiarizationResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")
	
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var sortformerResult struct {
		AudioFile     string `json:"audio_file"`
		Model         string `json:"model"`
		Segments      []struct {
			Start      float64 `json:"start"`
			End        float64 `json:"end"`
			Speaker    string  `json:"speaker"`
			Confidence float64 `json:"confidence"`
			Duration   float64 `json:"duration"`
		} `json:"segments"`
		Speakers      []string `json:"speakers"`
		SpeakerCount  int      `json:"speaker_count"`
		TotalSegments int      `json:"total_segments"`
		TotalDuration float64  `json:"total_duration"`
	}

	if err := json.Unmarshal(data, &sortformerResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Convert to standard format
	result := &interfaces.DiarizationResult{
		Segments:     make([]interfaces.DiarizationSegment, len(sortformerResult.Segments)),
		SpeakerCount: sortformerResult.SpeakerCount,
		Speakers:     sortformerResult.Speakers,
	}

	for i, seg := range sortformerResult.Segments {
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
func (s *SortformerAdapter) parseRTTMResult(tempDir string, input interfaces.AudioInput) (*interfaces.DiarizationResult, error) {
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

// GetEstimatedProcessingTime provides Sortformer-specific time estimation
func (s *SortformerAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Sortformer is typically very fast, often faster than real-time
	baseTime := s.BaseAdapter.GetEstimatedProcessingTime(input)
	
	// Sortformer typically processes at about 5-10% of audio duration
	return time.Duration(float64(baseTime) * 0.3)
}

// init registers the Sortformer adapter
func init() {
	registry.RegisterDiarizationAdapter("sortformer", NewSortformerAdapter())
}