package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/processing"
	"scriberr/pkg/logger"
)

// WhisperXService handles WhisperX transcription
type WhisperXService struct {
	multiTrackProcessor *processing.MultiTrackProcessor
}

// NewWhisperXService creates a new WhisperX service
func NewWhisperXService(cfg *config.Config) *WhisperXService {
	return &WhisperXService{
		multiTrackProcessor: processing.NewMultiTrackProcessor(),
	}
}

// TranscriptResult represents the WhisperX output format
type TranscriptResult struct {
	Segments []Segment `json:"segments"`
	Word     []Word    `json:"word_segments,omitempty"`
	Language string    `json:"language"`
	Text     string    `json:"text,omitempty"`
}

// Segment represents a transcript segment
type Segment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Speaker *string `json:"speaker,omitempty"`
}

// Word represents a word-level transcript
type Word struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Word    string  `json:"word"`
	Score   float64 `json:"score"`
	Speaker *string `json:"speaker,omitempty"`
}

// ProcessJob implements the JobProcessor interface
func (ws *WhisperXService) ProcessJob(ctx context.Context, jobID string) error {
	// Call the enhanced version with a no-op register function
	return ws.ProcessJobWithProcess(ctx, jobID, func(*exec.Cmd) {})
}

// ProcessJobWithProcess implements the enhanced JobProcessor interface
func (ws *WhisperXService) ProcessJobWithProcess(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error {
	// Get the job from database to check model family and multi-track status
	var job models.TranscriptionJob
	if err := database.DB.Preload("MultiTrackFiles").Where("id = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("failed to get job: %v", err)
	}

	// Validate single-track job doesn't have multi-track parameters enabled
	if !job.IsMultiTrack && job.Parameters.IsMultiTrackEnabled {
		return fmt.Errorf("single-track job cannot have multi-track transcription enabled")
	}

	// Check if this is a multi-track job with multi-track enabled parameters
	// This check now happens BEFORE model family routing
	if job.IsMultiTrack && job.Parameters.IsMultiTrackEnabled {
		logger.Info("Processing multi-track job", "job_id", jobID, "model_family", job.Parameters.ModelFamily, "merge_status", job.MergeStatus)

		// First, ensure audio files are merged if not already done
		if job.MergeStatus != "completed" {
			logger.Info("Starting merge processing for multi-track job", "job_id", jobID)
			if ws.multiTrackProcessor != nil {
				if err := ws.multiTrackProcessor.ProcessMultiTrackJob(ctx, jobID); err != nil {
					return fmt.Errorf("failed to merge multi-track audio: %w", err)
				}
			} else {
				logger.Warn("MultiTrackProcessor not available, skipping merge", "job_id", jobID)
			}
		}

		// Then transcribe each track individually and merge transcripts
		// MultiTrackTranscriber will handle model family routing internally
		logger.Info("Starting multi-track transcription", "job_id", jobID, "model_family", job.Parameters.ModelFamily)
		multiTrackTranscriber := NewMultiTrackTranscriber(ws)
		return multiTrackTranscriber.ProcessMultiTrackTranscription(ctx, jobID)
	}

	// Route single-track jobs to appropriate service based on model family
	logger.Info("Job routing", "job_id", jobID, "model_family", job.Parameters.ModelFamily, "is_multi_track", job.IsMultiTrack)
	if job.Parameters.ModelFamily == "nvidia_parakeet" {
		logger.Info("Routing job to Parakeet service", "job_id", jobID)
		parakeetService := NewParakeetService(nil)
		return parakeetService.ProcessJobWithProcess(ctx, jobID, registerProcess)
	} else if job.Parameters.ModelFamily == "nvidia_canary" {
		logger.Info("Routing job to Canary service", "job_id", jobID)
		canaryService := NewCanaryService(nil)
		return canaryService.ProcessJobWithProcess(ctx, jobID, registerProcess)
	}

	// Default to WhisperX processing
	logger.Info("Processing job with WhisperX", "job_id", jobID)
	return ws.processWhisperXJob(ctx, jobID, registerProcess)
}

// processWhisperXJob handles the original WhisperX processing logic
func (ws *WhisperXService) processWhisperXJob(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error {
	startTime := time.Now()

	// Get the job from database
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("failed to get job: %v", err)
	}

	// Create execution record to track this processing attempt
	execution := &models.TranscriptionJobExecution{
		TranscriptionJobID: jobID,
		StartedAt:          startTime,
		ActualParameters:   job.Parameters, // Copy the parameters used
		Status:             models.StatusProcessing,
	}

	if err := database.DB.Create(execution).Error; err != nil {
		return fmt.Errorf("failed to create execution record: %v", err)
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

	// Ensure Python environment is set up
	if err := ws.ensurePythonEnv(); err != nil {
		errMsg := fmt.Sprintf("failed to setup Python environment: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Check if audio file exists
	if _, err := os.Stat(job.AudioPath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("audio file not found: %s", job.AudioPath)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Prepare output directory
	outputDir := filepath.Join("data", "transcripts", jobID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		errMsg := fmt.Sprintf("failed to create output directory: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Build WhisperX command (handles both regular transcription and diarization)
	args, err := ws.buildWhisperXArgs(&job, outputDir)
	if err != nil {
		errMsg := fmt.Sprintf("failed to build command: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Create command with context for proper cancellation support
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	// Configure process attributes for cross-platform kill behavior
	configureCmdSysProcAttr(cmd)

	// Register the process for immediate termination capability
	registerProcess(cmd)

	// Execute WhisperX
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		errMsg := "job was cancelled"
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}
	if err != nil {
		fmt.Printf("DEBUG: WhisperX stderr/stdout: %s\n", string(output))
		errMsg := fmt.Sprintf("WhisperX execution failed: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Run separate diarization if nvidia_sortformer is selected
	if job.Parameters.Diarize && job.Parameters.DiarizeModel == "nvidia_sortformer" {
		fmt.Printf("DEBUG: Running separate NVIDIA Sortformer diarization for job %s\n", jobID)
		if err := ws.runSortformerDiarization(ctx, &job, outputDir); err != nil {
			errMsg := fmt.Sprintf("failed to run Sortformer diarization: %v", err)
			updateExecutionStatus(models.StatusFailed, errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	// Load and parse the result
	resultPath := filepath.Join(outputDir, "result.json")
	if err := ws.parseAndSaveResult(jobID, resultPath); err != nil {
		errMsg := fmt.Sprintf("failed to parse result: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Success! Update execution status
	updateExecutionStatus(models.StatusCompleted, "")

	return nil
}

// ensurePythonEnv ensures the Python environment is set up by cloning WhisperX from git and setting up NVidia models (Parakeet and Canary)
func (ws *WhisperXService) ensurePythonEnv() error {
	envPath := ws.getEnvPath()
	whisperxPath := filepath.Join(envPath, "WhisperX")
	nvidiaPath := filepath.Join(envPath, "parakeet") // Using parakeet directory for both models

	// Get absolute paths for debugging
	absEnvPath, _ := filepath.Abs(envPath)
	absWhisperxPath, _ := filepath.Abs(whisperxPath)
	absNvidiaPath, _ := filepath.Abs(nvidiaPath)
	workingDir, _ := os.Getwd()

	fmt.Printf("DEBUG: Current working directory: %s\n", workingDir)
	fmt.Printf("DEBUG: Relative WhisperX path: %s\n", whisperxPath)
	fmt.Printf("DEBUG: Absolute WhisperX path: %s\n", absWhisperxPath)
	fmt.Printf("DEBUG: Absolute NVIDIA path: %s\n", absNvidiaPath)
	fmt.Printf("DEBUG: Absolute env path: %s\n", absEnvPath)

	// Check WhisperX and NVIDIA environments independently
	whisperxCmd := exec.Command("uv", "run", "--native-tls", "--project", whisperxPath, "python", "-c", "import whisperx")
	nvidiaCmd := exec.Command("uv", "run", "--native-tls", "--project", nvidiaPath, "python", "-c", "import nemo.collections.asr")

	whisperxWorking := whisperxCmd.Run() == nil
	nvidiaEnvWorking := nvidiaCmd.Run() == nil

	// Check if both models exist
	parakeetModelPath := filepath.Join(nvidiaPath, "parakeet-tdt-0.6b-v3.nemo")
	canaryModelPath := filepath.Join(nvidiaPath, "canary-1b-v2.nemo")
	sortformerModelPath := filepath.Join(nvidiaPath, "diar_streaming_sortformer_4spk-v2.nemo")

	parakeetModelExists := false
	if stat, err := os.Stat(parakeetModelPath); err == nil && stat.Size() > 1024*1024 {
		parakeetModelExists = true
	}

	canaryModelExists := false
	if stat, err := os.Stat(canaryModelPath); err == nil && stat.Size() > 1024*1024 {
		canaryModelExists = true
	}

	sortformerModelExists := false
	if stat, err := os.Stat(sortformerModelPath); err == nil && stat.Size() > 1024*1024 {
		sortformerModelExists = true
	}

	// Check if nemo_diarize.py script exists
	nemoScriptPath := filepath.Join(nvidiaPath, "nemo_diarize.py")
	nemoScriptExists := false
	if _, err := os.Stat(nemoScriptPath); err == nil {
		nemoScriptExists = true
	}

	fmt.Printf("DEBUG: Environment status - WhisperX: %v, NVIDIA Env: %v, Parakeet Model: %v, Canary Model: %v, Sortformer Model: %v, NeMo Script: %v\n",
		whisperxWorking, nvidiaEnvWorking, parakeetModelExists, canaryModelExists, sortformerModelExists, nemoScriptExists)

	// If everything is working, we're done
	if whisperxWorking && nvidiaEnvWorking && parakeetModelExists && canaryModelExists && sortformerModelExists && nemoScriptExists {
		fmt.Printf("DEBUG: WhisperX and NVIDIA models fully set up and working\n")
		return nil
	}

	// Ensure base directory exists
	if err := os.MkdirAll(envPath, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %v", err)
	}

	// Setup WhisperX if needed
	if !whisperxWorking {
		fmt.Printf("DEBUG: Setting up WhisperX at: %s\n", whisperxPath)

		// Remove existing WhisperX directory if it exists
		if err := os.RemoveAll(whisperxPath); err != nil {
			return fmt.Errorf("failed to remove existing WhisperX environment: %v", err)
		}

		if err := ws.cloneWhisperX(envPath); err != nil {
			return fmt.Errorf("failed to clone WhisperX: %v", err)
		}

		if err := ws.updateWhisperXDependencies(whisperxPath); err != nil {
			return fmt.Errorf("failed to update WhisperX dependencies: %v", err)
		}

		if err := ws.uvSyncWhisperX(whisperxPath); err != nil {
			return fmt.Errorf("failed to sync WhisperX: %v", err)
		}

		fmt.Printf("DEBUG: WhisperX setup completed\n")
	} else {
		fmt.Printf("DEBUG: WhisperX already working, skipping setup\n")
	}

	// Setup NVIDIA environment if needed (used for both Parakeet and Canary)
	if !nvidiaEnvWorking {
		fmt.Printf("DEBUG: Setting up NVIDIA environment at: %s\n", nvidiaPath)

		// Remove existing NVIDIA directory if it exists
		if err := os.RemoveAll(nvidiaPath); err != nil {
			return fmt.Errorf("failed to remove existing NVIDIA environment: %v", err)
		}

		if err := ws.setupParakeetEnv(nvidiaPath); err != nil {
			return fmt.Errorf("failed to setup NVIDIA environment: %v", err)
		}

		fmt.Printf("DEBUG: NVIDIA environment setup completed\n")
	} else {
		fmt.Printf("DEBUG: NVIDIA environment already working, skipping setup\n")
	}

	// Download Parakeet model if needed
	if !parakeetModelExists {
		fmt.Printf("DEBUG: Downloading Parakeet model\n")
		if err := ws.downloadParakeetModel(nvidiaPath); err != nil {
			return fmt.Errorf("failed to download Parakeet model: %v", err)
		}
		fmt.Printf("DEBUG: Parakeet model download completed\n")
	} else {
		fmt.Printf("DEBUG: Parakeet model already exists, skipping download\n")
	}

	// Download Canary model if needed
	if !canaryModelExists {
		fmt.Printf("DEBUG: Downloading Canary model\n")
		if err := ws.downloadCanaryModel(nvidiaPath); err != nil {
			return fmt.Errorf("failed to download Canary model: %v", err)
		}
		fmt.Printf("DEBUG: Canary model download completed\n")
	} else {
		fmt.Printf("DEBUG: Canary model already exists, skipping download\n")
	}

	// Download Sortformer model if needed
	if !sortformerModelExists {
		fmt.Printf("DEBUG: Downloading Sortformer diarization model\n")
		if err := ws.downloadSortformerModel(nvidiaPath); err != nil {
			return fmt.Errorf("failed to download Sortformer model: %v", err)
		}
		fmt.Printf("DEBUG: Sortformer model download completed\n")
	} else {
		fmt.Printf("DEBUG: Sortformer model already exists, skipping download\n")
	}

	// Create nemo_diarize.py script if needed
	if !nemoScriptExists {
		fmt.Printf("DEBUG: Creating nemo_diarize.py script\n")
		if err := ws.createNemoDiarizeScript(nvidiaPath); err != nil {
			return fmt.Errorf("failed to create nemo_diarize.py script: %v", err)
		}
		fmt.Printf("DEBUG: nemo_diarize.py script created\n")
	} else {
		fmt.Printf("DEBUG: nemo_diarize.py script already exists, skipping creation\n")
	}

	fmt.Printf("DEBUG: Environment setup completed successfully\n")
	return nil
}

// cloneWhisperX clones the WhisperX repository
func (ws *WhisperXService) cloneWhisperX(envPath string) error {
	cmd := exec.Command("git", "clone", "https://github.com/m-bain/WhisperX.git")
	cmd.Dir = envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// updateWhisperXDependencies modifies WhisperX pyproject.toml to update ctranslate2 and add yt-dlp
func (ws *WhisperXService) updateWhisperXDependencies(whisperxPath string) error {
	pyprojectPath := filepath.Join(whisperxPath, "pyproject.toml")

	// Read the existing pyproject.toml
	data, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return fmt.Errorf("failed to read pyproject.toml: %v", err)
	}

	content := string(data)

	// Replace ctranslate2 dependency
	content = strings.ReplaceAll(content, "ctranslate2<4.5.0", "ctranslate2==4.6.0")

	// Add yt-dlp if not already present
	if !strings.Contains(content, "yt-dlp") {
		// Find the dependencies section and add yt-dlp
		content = strings.ReplaceAll(content,
			`"transformers>=4.48.0",`,
			`"transformers>=4.48.0",
    "yt-dlp",`)
	}

	// Write back the modified content
	if err := os.WriteFile(pyprojectPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %v", err)
	}

	return nil
}

// uvSyncWhisperX runs `uv sync --all-extras --dev` for WhisperX
func (ws *WhisperXService) uvSyncWhisperX(whisperxPath string) error {
	cmd := exec.Command("uv", "sync", "--all-extras", "--dev", "--native-tls")
	cmd.Dir = whisperxPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// uvSync runs `uv sync` for the given project path
func (ws *WhisperXService) uvSync(projectPath string) error {
	cmd := exec.Command("uv", "sync", "--native-tls", "--project", projectPath)
	cmd.Dir = projectPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// setupParakeetEnv sets up the Parakeet environment with NVIDIA ASR dependencies (without model download)
func (ws *WhisperXService) setupParakeetEnv(parakeetPath string) error {
	// Create the parakeet directory
	if err := os.MkdirAll(parakeetPath, 0755); err != nil {
		return fmt.Errorf("failed to create parakeet directory: %v", err)
	}

	// Create pyproject.toml for NVIDIA models (Parakeet & Canary) with NeMo from main branch for timestamps support
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
	pyprojectPath := filepath.Join(parakeetPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		return fmt.Errorf("failed to write parakeet pyproject.toml: %v", err)
	}

	// Create the transcription script
	transcribeScript := `#!/usr/bin/env python3
"""
Audio transcription script using NVIDIA models (Parakeet TDT 0.6B v3 or Canary 1B v2).
Supports multiple European languages with automatic language detection.
"""

import argparse
import json
import sys
from pathlib import Path
import nemo.collections.asr as nemo_asr


def transcribe_audio(
    audio_path: str, timestamps: bool = False, output_file: str = None, 
    model_type: str = "parakeet", source_lang: str = "en", target_lang: str = "en",
    context_left: int = 256, context_right: int = 256
):
    """
    Transcribe audio file using NVIDIA models.

    Args:
        audio_path: Path to audio file (.wav or .flac)
        timestamps: Whether to include timestamps in output
        output_file: Optional output file path for results
        model_type: Type of model to use ("parakeet" or "canary")
        source_lang: Source language for Canary model
        target_lang: Target language for Canary model
        context_left: Left attention context size for long-form audio (Parakeet only)
        context_right: Right attention context size for long-form audio (Parakeet only)
    """
    if model_type == "canary":
        model_path = "./canary-1b-v2.nemo"
        print(f"Loading NVIDIA Canary model from: {model_path}")
    else:
        model_path = "./parakeet-tdt-0.6b-v3.nemo"
        print(f"Loading NVIDIA Parakeet model from: {model_path}")
    
    asr_model = nemo_asr.models.ASRModel.restore_from(model_path)
    
    # Configure for long-form audio if context sizes are not default (Parakeet only)
    if model_type == "parakeet" and (context_left != 256 or context_right != 256):
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
        if model_type == "canary":
            # Canary model supports both transcription and translation
            output = asr_model.transcribe([audio_path], source_lang=source_lang, target_lang=target_lang, timestamps=True)
        else:
            # Parakeet model
            output = asr_model.transcribe([audio_path], timestamps=True)

        # Extract text and timestamps
        text = output[0].text
        word_timestamps = output[0].timestamp.get("word", [])
        segment_timestamps = output[0].timestamp.get("segment", [])

        print(f"\nTranscription: {text}")
        print("\nSegment timestamps:")
        for stamp in segment_timestamps:
            print(f"{stamp['start']:.2f}s - {stamp['end']:.2f}s : {stamp['segment']}")

        # Save detailed output if requested
        if output_file:
            result_data = {
                "transcription": text,
                "word_timestamps": word_timestamps,
                "segment_timestamps": segment_timestamps,
                "audio_file": audio_path,
                "source_language": source_lang,
                "target_language": target_lang
            }
            
            if output_file.endswith('.json'):
                with open(output_file, 'w', encoding='utf-8') as f:
                    json.dump(result_data, f, indent=2, ensure_ascii=False)
                print(f"\nResults saved to JSON: {output_file}")
            else:
                with open(output_file, "w", encoding="utf-8") as f:
                    f.write(f"Transcription: {text}\n\n")
                    f.write("Segment timestamps:\n")
                    for stamp in segment_timestamps:
                        f.write(
                            f"{stamp['start']:.2f}s - {stamp['end']:.2f}s : {stamp['segment']}\n"
                        )
                    f.write("\nWord timestamps:\n")
                    for stamp in word_timestamps:
                        f.write(
                            f"{stamp['start']:.2f}s - {stamp['end']:.2f}s : {stamp['word']}\n"
                        )
                print(f"\nResults saved to: {output_file}")

    else:
        if model_type == "canary":
            # Canary model supports both transcription and translation
            output = asr_model.transcribe([audio_path], source_lang=source_lang, target_lang=target_lang)
        else:
            # Parakeet model
            output = asr_model.transcribe([audio_path])
        
        text = output[0].text
        print(f"\nTranscription: {text}")

        if output_file:
            if output_file.endswith('.json'):
                result_data = {
                    "transcription": text,
                    "audio_file": audio_path,
                    "source_language": source_lang,
                    "target_language": target_lang
                }
                with open(output_file, 'w', encoding='utf-8') as f:
                    json.dump(result_data, f, indent=2, ensure_ascii=False)
                print(f"\nResults saved to JSON: {output_file}")
            else:
                with open(output_file, "w", encoding="utf-8") as f:
                    f.write(text)
                print(f"\nResults saved to: {output_file}")


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe audio using NVIDIA models (Parakeet or Canary)"
    )
    parser.add_argument("audio_file", help="Path to audio file (.wav or .flac format)")
    parser.add_argument(
        "--timestamps",
        action="store_true",
        help="Include word and segment level timestamps",
    )
    parser.add_argument(
        "--output", "-o", help="Output file path to save transcription results"
    )
    parser.add_argument(
        "--model", choices=["parakeet", "canary"], default="parakeet",
        help="Model type to use (default: parakeet)"
    )
    parser.add_argument(
        "--source-lang", default="en",
        help="Source language (for Canary model, default: en)"
    )
    parser.add_argument(
        "--target-lang", default="en", 
        help="Target language (for Canary model, default: en)"
    )
    parser.add_argument(
        "--context-left", type=int, default=256,
        help="Left attention context size for long-form audio - Parakeet only (default: 256)"
    )
    parser.add_argument(
        "--context-right", type=int, default=256, 
        help="Right attention context size for long-form audio - Parakeet only (default: 256)"
    )

    args = parser.parse_args()

    # Validate input file
    audio_path = Path(args.audio_file)
    if not audio_path.exists():
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

    if audio_path.suffix.lower() not in [".wav", ".flac"]:
        print(f"Warning: File extension '{audio_path.suffix}' may not be supported.")
        print("Recommended formats: .wav, .flac")

    try:
        transcribe_audio(
            audio_path=str(audio_path),
            timestamps=args.timestamps,
            output_file=args.output,
            model_type=args.model,
            source_lang=args.source_lang,
            target_lang=args.target_lang,
            context_left=args.context_left,
            context_right=args.context_right,
        )
    except Exception as e:
        print(f"Error during transcription: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
`
	transcriptPath := filepath.Join(parakeetPath, "transcribe.py")
	if err := os.WriteFile(transcriptPath, []byte(transcribeScript), 0755); err != nil {
		return fmt.Errorf("failed to write parakeet transcription script: %v", err)
	}

	// Create the diarization script
	diarizeScript := `#!/usr/bin/env python3
"""
Speaker diarization script using Pyannote.audio.
Processes audio files to identify and separate different speakers.
"""

import argparse
import sys
import os
from pathlib import Path
from pyannote.audio import Pipeline


def diarize_audio(
    audio_path: str, 
    output_file: str, 
    hf_token: str, 
    min_speakers: int = None, 
    max_speakers: int = None
):
    """
    Perform speaker diarization on audio file using Pyannote.

    Args:
        audio_path: Path to audio file
        output_file: Path to save RTTM output file
        hf_token: Hugging Face token for model access
        min_speakers: Minimum number of speakers (optional)
        max_speakers: Maximum number of speakers (optional)
    """
    print(f"Loading Pyannote speaker diarization pipeline...")
    
    try:
        # Initialize the diarization pipeline
        pipeline = Pipeline.from_pretrained(
            "pyannote/speaker-diarization-3.1",
            use_auth_token=hf_token
        )
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
        
        # Save the diarization output to RTTM format
        with open(output_file, "w") as rttm:
            diarization.write_rttm(rttm)
        
        # Print summary
        speakers = set()
        total_speech_time = 0.0
        
        with open(output_file, "r") as f:
            for line in f:
                if line.startswith("SPEAKER"):
                    parts = line.strip().split()
                    if len(parts) >= 8:
                        speaker = parts[7]
                        duration = float(parts[4])
                        speakers.add(speaker)
                        total_speech_time += duration
        
        print(f"\nDiarization Summary:")
        print(f"  Speakers detected: {len(speakers)}")
        print(f"  Speaker labels: {sorted(speakers)}")
        print(f"  Total speech time: {total_speech_time:.2f} seconds")
        print(f"  RTTM file saved: {output_file}")
        
    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Perform speaker diarization using Pyannote.audio"
    )
    parser.add_argument(
        "audio_file", 
        help="Path to audio file"
    )
    parser.add_argument(
        "--output", "-o", 
        required=True,
        help="Output RTTM file path"
    )
    parser.add_argument(
        "--hf-token", 
        required=True,
        help="Hugging Face access token"
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

    args = parser.parse_args()

    # Validate input file
    audio_path = Path(args.audio_file)
    if not audio_path.exists():
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
            audio_path=str(audio_path),
            output_file=args.output,
            hf_token=args.hf_token,
            min_speakers=args.min_speakers,
            max_speakers=args.max_speakers
        )
    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
`
	diarizePath := filepath.Join(parakeetPath, "diarize.py")
	if err := os.WriteFile(diarizePath, []byte(diarizeScript), 0755); err != nil {
		return fmt.Errorf("failed to write parakeet diarization script: %v", err)
	}

	// Run uv sync to install dependencies
	fmt.Printf("DEBUG: Installing Parakeet dependencies in: %s\n", parakeetPath)
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = parakeetPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed for parakeet: %v: %s", err, strings.TrimSpace(string(out)))
	}

	fmt.Printf("DEBUG: Parakeet environment setup completed successfully\n")
	return nil
}

// downloadParakeetModel downloads the Parakeet TDT 0.6B v3 model
func (ws *WhisperXService) downloadParakeetModel(parakeetPath string) error {
	modelURL := "https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3/resolve/main/parakeet-tdt-0.6b-v3.nemo?download=true"
	modelFileName := "parakeet-tdt-0.6b-v3.nemo"
	modelPath := filepath.Join(parakeetPath, modelFileName)

	// Ensure the parakeet directory exists before downloading
	if err := os.MkdirAll(parakeetPath, 0755); err != nil {
		return fmt.Errorf("failed to create parakeet directory for model download: %v", err)
	}

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		fmt.Printf("DEBUG: Parakeet model already exists at: %s (size: %d bytes)\n", modelPath, stat.Size())
		return nil
	}

	fmt.Printf("DEBUG: Downloading Parakeet model from: %s\n", modelURL)
	fmt.Printf("DEBUG: Saving to: %s\n", modelPath)

	// Use curl to download the model with timeout and progress indicator
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Create temporary file for safer download
	tempPath := modelPath + ".tmp"

	// Remove any existing temp file
	os.Remove(tempPath)

	cmd := exec.CommandContext(ctx, "curl",
		"-L",             // Follow redirects
		"--progress-bar", // Show progress bar
		"--create-dirs",  // Create directories if needed
		"-o", tempPath,   // Output to temp file
		modelURL)

	fmt.Printf("DEBUG: Running curl command: %v\n", cmd.Args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up temp file on error
		os.Remove(tempPath)
		return fmt.Errorf("failed to download Parakeet model: %v: %s", err, strings.TrimSpace(string(out)))
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, modelPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to move downloaded model to final location: %v", err)
	}

	// Verify the downloaded file exists and has reasonable size
	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("downloaded model file not found: %v", err)
	}
	if stat.Size() < 1024*1024 { // Less than 1MB suggests download failed
		return fmt.Errorf("downloaded model file appears incomplete (size: %d bytes)", stat.Size())
	}

	fmt.Printf("DEBUG: Successfully downloaded Parakeet model (size: %d bytes)\n", stat.Size())
	return nil
}

// downloadCanaryModel downloads the Canary 1B v2 model
func (ws *WhisperXService) downloadCanaryModel(nvidiaPath string) error {
	modelURL := "https://huggingface.co/nvidia/canary-1b-v2/resolve/main/canary-1b-v2.nemo?download=true"
	modelFileName := "canary-1b-v2.nemo"
	modelPath := filepath.Join(nvidiaPath, modelFileName)

	// Ensure the nvidia directory exists before downloading
	if err := os.MkdirAll(nvidiaPath, 0755); err != nil {
		return fmt.Errorf("failed to create nvidia directory for model download: %v", err)
	}

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		fmt.Printf("DEBUG: Canary model already exists at: %s (size: %d bytes)\n", modelPath, stat.Size())
		return nil
	}

	fmt.Printf("DEBUG: Downloading Canary model from: %s\n", modelURL)
	fmt.Printf("DEBUG: Saving to: %s\n", modelPath)

	// Use curl to download the model with timeout and progress indicator
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Create temporary file for safer download
	tempPath := modelPath + ".tmp"

	// Remove any existing temp file
	os.Remove(tempPath)

	cmd := exec.CommandContext(ctx, "curl",
		"-L",             // Follow redirects
		"--progress-bar", // Show progress bar
		"--create-dirs",  // Create directories if needed
		"-o", tempPath,   // Output to temp file
		modelURL)

	fmt.Printf("DEBUG: Running curl command: %v\n", cmd.Args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up temp file on error
		os.Remove(tempPath)
		return fmt.Errorf("failed to download Canary model: %v: %s", err, strings.TrimSpace(string(out)))
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, modelPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to move downloaded model to final location: %v", err)
	}

	// Verify the downloaded file exists and has reasonable size
	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("downloaded model file not found: %v", err)
	}
	if stat.Size() < 1024*1024 { // Less than 1MB suggests download failed
		return fmt.Errorf("downloaded model file appears incomplete (size: %d bytes)", stat.Size())
	}

	fmt.Printf("DEBUG: Successfully downloaded Canary model (size: %d bytes)\n", stat.Size())
	return nil
}

// downloadSortformerModel downloads the NVIDIA Sortformer diarization model
func (ws *WhisperXService) downloadSortformerModel(nvidiaPath string) error {
	modelURL := "https://huggingface.co/nvidia/diar_streaming_sortformer_4spk-v2/resolve/main/diar_streaming_sortformer_4spk-v2.nemo?download=true"
	modelFileName := "diar_streaming_sortformer_4spk-v2.nemo"
	modelPath := filepath.Join(nvidiaPath, modelFileName)

	// Ensure the nvidia directory exists before downloading
	if err := os.MkdirAll(nvidiaPath, 0755); err != nil {
		return fmt.Errorf("failed to create nvidia directory for model download: %v", err)
	}

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		fmt.Printf("DEBUG: Sortformer model already exists at: %s (size: %d bytes)\n", modelPath, stat.Size())
		return nil
	}

	fmt.Printf("DEBUG: Downloading Sortformer model from: %s\n", modelURL)
	fmt.Printf("DEBUG: Saving to: %s\n", modelPath)

	// Use curl to download the model with timeout and progress indicator
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Create temporary file for safer download
	tempPath := modelPath + ".tmp"

	// Remove any existing temp file
	os.Remove(tempPath)

	cmd := exec.CommandContext(ctx, "curl",
		"-L",             // Follow redirects
		"-#",             // Progress bar
		"--max-time", "1800", // 30 minutes timeout
		"-o", tempPath,   // Output file
		modelURL,
	)

	// Run the download command
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("curl download failed: %v: %s", err, strings.TrimSpace(string(out)))
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, modelPath); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to move downloaded file: %v", err)
	}

	// Verify the downloaded file
	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("downloaded model file not found: %v", err)
	}
	if stat.Size() < 1024*1024 { // Less than 1MB suggests download failed
		return fmt.Errorf("downloaded model file appears incomplete (size: %d bytes)", stat.Size())
	}

	fmt.Printf("DEBUG: Successfully downloaded Sortformer model (size: %d bytes)\n", stat.Size())
	return nil
}

// createNemoDiarizeScript creates the nemo_diarize.py script in the nvidia path
func (ws *WhisperXService) createNemoDiarizeScript(nvidiaPath string) error {
	scriptPath := filepath.Join(nvidiaPath, "nemo_diarize.py")

	scriptContent := `#!/usr/bin/env python3
"""
Speaker diarization script using NVIDIA's Sortformer model.
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
):
    """
    Perform speaker diarization using NVIDIA's Sortformer model.

    Args:
        audio_path: Path to audio file
        output_file: Path to save output file (JSON or RTTM format)
        batch_size: Batch size for processing
        device: Device to use (cuda/cpu, auto-detected if None)
        max_speakers: Maximum number of speakers (4 optimized for this model)
    """
    if device is None:
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
        print(
            f"Running diarization with batch_size={batch_size}, max_speakers={max_speakers}"
        )
        predicted_segments = diar_model.diarize(audio=audio_path, batch_size=batch_size)

        print(f"Diarization completed. Found segments: {len(predicted_segments)}")

        # Process and save results
        save_results(predicted_segments, output_file, audio_path)

    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


def save_results(segments, output_file: str, audio_path: str):
    """
    Save diarization results to output file.
    Supports both JSON and RTTM formats based on file extension.
    """
    output_path = Path(output_file)

    # Determine output format based on extension
    if output_path.suffix.lower() == ".rttm":
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
                }
            elif isinstance(segment, (list, tuple)) and len(segment) >= 3:
                # List/tuple format: [start, end, speaker]
                segment_data = {
                    "start": float(segment[0]),
                    "end": float(segment[1]),
                    "speaker": str(segment[2]),
                    "duration": float(segment[1] - segment[0]),
                }
            elif isinstance(segment, dict):
                # Dictionary format
                segment_data = {
                    "start": float(segment.get('start', 0)),
                    "end": float(segment.get('end', 0)),
                    "speaker": str(segment.get('speaker', segment.get('label', f'speaker_{i}'))),
                    "duration": float(segment.get('end', 0) - segment.get('start', 0)),
                }
            else:
                # Fallback: try to extract attributes dynamically
                segment_data = {
                    "start": float(getattr(segment, 'start', 0)),
                    "end": float(getattr(segment, 'end', 0)),
                    "speaker": str(getattr(segment, 'label', getattr(segment, 'speaker', f'speaker_{i}'))),
                    "duration": float(getattr(segment, 'end', 0) - getattr(segment, 'start', 0)),
                }

            results["segments"].append(segment_data)

        except Exception as e:
            print(f"Warning: Could not process segment {i}: {e}")
            print(f"Segment: {segment}")

    # Sort by start time
    if results["segments"]:
        results["segments"].sort(key=lambda x: x["start"])

    # Add summary statistics
    speakers = set(seg["speaker"] for seg in results["segments"])
    results["summary"] = {
        "total_segments": len(results["segments"]),
        "speakers": list(speakers),
        "speaker_count": len(speakers),
        "total_duration": max(seg["end"] for seg in results["segments"])
        if results["segments"]
        else 0,
    }

    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    print(f"Results saved to: {output_file}")
    print(f"Found {len(speakers)} speakers: {', '.join(speakers)}")


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
    print(f"Found {len(speakers)} speakers: {', '.join(speakers)}")


def main():
    parser = argparse.ArgumentParser(
        description="Speaker diarization using NVIDIA Sortformer model (local model only)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    # Basic diarization with JSON output
    python nemo_diarize.py samples/sample.wav output.json
    
    # Generate RTTM format output
    python nemo_diarize.py samples/sample.wav output.rttm
    
    # Specify device and batch size
    python nemo_diarize.py --device cuda --batch-size 2 samples/sample.wav output.json

Note: This script requires diar_streaming_sortformer_4spk-v2.nemo to be in the same directory.
        """,
    )

    parser.add_argument("audio_file", help="Path to input audio file (WAV, FLAC, etc.)")

    parser.add_argument(
        "output_file",
        help="Path to output file (.json for JSON format, .rttm for RTTM format)",
    )

    parser.add_argument(
        "--batch-size",
        type=int,
        default=1,
        help="Batch size for processing (default: 1)",
    )

    parser.add_argument(
        "--device",
        choices=["cuda", "cpu", "auto"],
        default="auto",
        help="Device to use for inference (default: auto-detect)",
    )

    parser.add_argument(
        "--max-speakers",
        type=int,
        default=4,
        help="Maximum number of speakers (default: 4, optimized for this model)",
    )

    args = parser.parse_args()

    # Validate inputs
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

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
    )


if __name__ == "__main__":
    main()
`

	// Write the script to file
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write nemo_diarize.py script: %v", err)
	}

	fmt.Printf("DEBUG: Successfully created nemo_diarize.py script at: %s\n", scriptPath)
	return nil
}

// InitEmbeddedPythonEnv initializes the Python env on app start (blocking).
// Assumes uv is installed and accessible in system PATH.
func (ws *WhisperXService) InitEmbeddedPythonEnv() error {
	if err := ws.ensurePythonEnv(); err != nil {
		return err
	}
	return nil
}

// buildWhisperXArgs builds the WhisperX command arguments
func (ws *WhisperXService) buildWhisperXArgs(job *models.TranscriptionJob, outputDir string) ([]string, error) {
	p := job.Parameters

	// Debug: log diarization status
	fmt.Printf("DEBUG: Job ID %s, Diarize parameter: %v, Job Diarization field: %v\n", job.ID, p.Diarize, job.Diarization)

	// Use WhisperX CLI for both regular transcription and diarization
	whisperxPath := filepath.Join(ws.getEnvPath(), "WhisperX")
	args := []string{
		"run", "--native-tls", "--project", whisperxPath, "python", "-m", "whisperx",
		job.AudioPath,
		"--output_dir", outputDir,
	}

	// Core parameters
	args = append(args, "--model", p.Model)
	if p.ModelCacheOnly {
		args = append(args, "--model_cache_only", "True")
	}
	if p.ModelDir != nil {
		args = append(args, "--model_dir", *p.ModelDir)
	}

	// Device and computation
	args = append(args, "--device", p.Device)
	args = append(args, "--device_index", strconv.Itoa(p.DeviceIndex))
	args = append(args, "--batch_size", strconv.Itoa(p.BatchSize))
	args = append(args, "--compute_type", p.ComputeType)
	if p.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(p.Threads))
	}

	// Output settings - hard-coded for consistency
	args = append(args, "--output_format", "all")
	args = append(args, "--verbose", "True")

	// Task and language
	args = append(args, "--task", p.Task)
	if p.Language != nil {
		args = append(args, "--language", *p.Language)
	}

	// Alignment settings
	if p.AlignModel != nil {
		args = append(args, "--align_model", *p.AlignModel)
	}
	args = append(args, "--interpolate_method", p.InterpolateMethod)
	if p.NoAlign {
		args = append(args, "--no_align")
	}
	if p.ReturnCharAlignments {
		args = append(args, "--return_char_alignments")
	}

	// VAD settings
	args = append(args, "--vad_method", p.VadMethod)
	args = append(args, "--vad_onset", fmt.Sprintf("%.3f", p.VadOnset))
	args = append(args, "--vad_offset", fmt.Sprintf("%.3f", p.VadOffset))
	args = append(args, "--chunk_size", strconv.Itoa(p.ChunkSize))

	// Diarization settings
	if p.Diarize && p.DiarizeModel != "nvidia_sortformer" {
		// Use WhisperX built-in diarization for pyannote
		args = append(args, "--diarize")
		if p.MinSpeakers != nil {
			args = append(args, "--min_speakers", strconv.Itoa(*p.MinSpeakers))
		}
		if p.MaxSpeakers != nil {
			args = append(args, "--max_speakers", strconv.Itoa(*p.MaxSpeakers))
		}
		// Map simple model name to full model path for WhisperX
		diarizeModel := p.DiarizeModel
		if diarizeModel == "pyannote" {
			diarizeModel = "pyannote/speaker-diarization-3.1"
		}
		args = append(args, "--diarize_model", diarizeModel)
		if p.SpeakerEmbeddings {
			args = append(args, "--speaker_embeddings")
		}
	}
	// Note: If nvidia_sortformer is selected, diarization will be done separately after transcription

	// Transcription quality settings
	args = append(args, "--temperature", fmt.Sprintf("%.2f", p.Temperature))
	args = append(args, "--best_of", strconv.Itoa(p.BestOf))
	args = append(args, "--beam_size", strconv.Itoa(p.BeamSize))
	args = append(args, "--patience", fmt.Sprintf("%.2f", p.Patience))
	args = append(args, "--length_penalty", fmt.Sprintf("%.2f", p.LengthPenalty))
	if p.SuppressTokens != nil {
		args = append(args, "--suppress_tokens", *p.SuppressTokens)
	}
	if p.SuppressNumerals {
		args = append(args, "--suppress_numerals")
	}
	if p.InitialPrompt != nil {
		args = append(args, "--initial_prompt", *p.InitialPrompt)
	}
	if p.ConditionOnPreviousText {
		args = append(args, "--condition_on_previous_text", "True")
	}
	if !p.Fp16 {
		args = append(args, "--fp16", "False")
	}
	args = append(args, "--temperature_increment_on_fallback", fmt.Sprintf("%.2f", p.TemperatureIncrementOnFallback))
	args = append(args, "--compression_ratio_threshold", fmt.Sprintf("%.2f", p.CompressionRatioThreshold))
	args = append(args, "--logprob_threshold", fmt.Sprintf("%.2f", p.LogprobThreshold))
	args = append(args, "--no_speech_threshold", fmt.Sprintf("%.2f", p.NoSpeechThreshold))

	// Output formatting - hard-coded for consistency
	// Hard-coded: no max line width/count restrictions
	args = append(args, "--highlight_words", "False")
	args = append(args, "--segment_resolution", "sentence")

	// Token and progress
	if p.HfToken != nil {
		args = append(args, "--hf_token", *p.HfToken)
	}
	// Hard-coded: disable print progress for cleaner output
	args = append(args, "--print_progress", "False")

	// Debug: log the command being executed
	fmt.Printf("DEBUG: WhisperX command: uv %v\n", args)

	return args, nil
}

// parseAndSaveResult parses WhisperX output and saves to database
func (ws *WhisperXService) parseAndSaveResult(jobID, resultPath string) error {
	var resultFile string

	// WhisperX creates files based on input filename, not result.json
	// Look for JSON files that match the expected WhisperX output pattern
	files, err := filepath.Glob(filepath.Join(filepath.Dir(resultPath), "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find result files: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no result files found")
	}

	// Filter out result.json (which is Parakeet/Canary format) and find WhisperX format
	var whisperxFile string
	for _, file := range files {
		if filepath.Base(file) != "result.json" {
			whisperxFile = file
			break
		}
	}

	if whisperxFile == "" {
		// Fall back to result.json if no other files found
		if _, err := os.Stat(resultPath); err == nil {
			whisperxFile = resultPath
		} else {
			return fmt.Errorf("no WhisperX result files found")
		}
	}

	resultFile = whisperxFile
	fmt.Printf("DEBUG: Using WhisperX result file: %s\n", resultFile)

	// Read the result file
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return fmt.Errorf("failed to read result file: %v", err)
	}

	// Parse the result
	var result TranscriptResult
	fmt.Printf("DEBUG: Raw JSON data: %s\n", string(data))
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse JSON result: %v", err)
	}
	fmt.Printf("DEBUG: Parsed result - Segments: %d, Words: %d, Language: '%s', Text: '%s'\n",
		len(result.Segments), len(result.Word), result.Language, result.Text)
	if len(result.Segments) > 0 {
		speaker := "nil"
		if result.Segments[0].Speaker != nil {
			speaker = *result.Segments[0].Speaker
		}
		fmt.Printf("DEBUG: First segment: start=%.2f, end=%.2f, text='%s', speaker='%s'\n",
			result.Segments[0].Start, result.Segments[0].End, result.Segments[0].Text, speaker)
	}

	// Ensure Text field is populated for WhisperX results
	if result.Text == "" && len(result.Segments) > 0 {
		var texts []string
		for _, segment := range result.Segments {
			texts = append(texts, segment.Text)
		}
		result.Text = strings.Join(texts, " ")
		fmt.Printf("DEBUG: Generated text from segments: '%s'\n", result.Text)
	}

	// Convert to JSON string for database storage
	transcriptJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal transcript: %v", err)
	}
	transcriptStr := string(transcriptJSON)

	// Clear any existing speaker mappings since we're retranscribing
	if err := database.DB.Where("transcription_job_id = ?", jobID).Delete(&models.SpeakerMapping{}).Error; err != nil {
		return fmt.Errorf("failed to clear old speaker mappings: %v", err)
	}

	// Update the job in the database
	if err := database.DB.Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Update("transcript", &transcriptStr).Error; err != nil {
		return fmt.Errorf("failed to update job transcript: %v", err)
	}

	return nil
}

// getEnvPath returns the hardcoded path for the WhisperX environment.
// Creates the environment in a local "whisperx-env" directory.
func (ws *WhisperXService) getEnvPath() string {
	return "whisperx-env"
}

// GetSupportedModels returns a list of supported WhisperX models
func (ws *WhisperXService) GetSupportedModels() []string {
	return []string{
		"tiny", "tiny.en",
		"base", "base.en",
		"small", "small.en",
		"medium", "medium.en",
		"large", "large-v1", "large-v2", "large-v3",
	}
}

// GetSupportedLanguages returns a list of supported languages
func (ws *WhisperXService) GetSupportedLanguages() []string {
	return []string{
		"en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr", "pl", "ca", "nl",
		"ar", "sv", "it", "id", "hi", "fi", "vi", "he", "uk", "el", "ms", "cs", "ro",
		"da", "hu", "ta", "no", "th", "ur", "hr", "bg", "lt", "la", "mi", "ml", "cy",
		"sk", "te", "fa", "lv", "bn", "sr", "az", "sl", "kn", "et", "mk", "br", "eu",
		"is", "hy", "ne", "mn", "bs", "kk", "sq", "sw", "gl", "mr", "pa", "si", "km",
		"sn", "yo", "so", "af", "oc", "ka", "be", "tg", "sd", "gu", "am", "yi", "lo",
		"uz", "fo", "ht", "ps", "tk", "nn", "mt", "sa", "lb", "my", "bo", "tl", "mg",
		"as", "tt", "haw", "ln", "ha", "ba", "jw", "su",
	}
}

// TranscribeAudioFile transcribes a single audio file directly without requiring a database job
// This is a cleaner approach for multi-track processing that avoids temporary database records
func (ws *WhisperXService) TranscribeAudioFile(ctx context.Context, audioPath string, params models.WhisperXParams) (*TranscriptResult, error) {
	fmt.Printf("DEBUG: TranscribeAudioFile starting for: %s\n", audioPath)

	// Ensure Python environment is set up
	if err := ws.ensurePythonEnv(); err != nil {
		return nil, fmt.Errorf("failed to setup Python environment: %v", err)
	}

	// Check if audio file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("audio file not found: %s", audioPath)
	}

	// Create temporary output directory for this transcription
	tempDir := filepath.Join("data", "temp", fmt.Sprintf("track_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temporary output directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory

	fmt.Printf("DEBUG: Using temporary directory: %s\n", tempDir)

	// Create a temporary job-like structure for building args
	tempJob := &models.TranscriptionJob{
		ID:         "temp", // Give it a temporary ID for logging
		AudioPath:  audioPath,
		Parameters: params,
	}

	// Build WhisperX command
	args, err := ws.buildWhisperXArgs(tempJob, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %v", err)
	}

	// Create command with context for proper cancellation support
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	// Configure process attributes for cross-platform kill behavior
	configureCmdSysProcAttr(cmd)

	fmt.Printf("DEBUG: Executing WhisperX command for track\n")

	// Execute WhisperX
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("transcription was cancelled")
	}
	if err != nil {
		fmt.Printf("DEBUG: WhisperX stderr/stdout: %s\n", string(output))
		return nil, fmt.Errorf("WhisperX execution failed: %v", err)
	}

	fmt.Printf("DEBUG: WhisperX completed successfully, parsing results from: %s\n", tempDir)

	// Parse the result from the temporary output
	resultPath := filepath.Join(tempDir, "result.json")
	result, err := ws.parseResultFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %v", err)
	}

	fmt.Printf("DEBUG: Successfully parsed track result with %d segments\n", len(result.Segments))
	return result, nil
}

// parseResultFile parses a WhisperX result JSON file and returns the transcript result
// This is extracted from parseAndSaveResult to avoid database operations
func (ws *WhisperXService) parseResultFile(expectedResultPath string) (*TranscriptResult, error) {
	// WhisperX creates files based on input filename, not result.json
	// Look for JSON files that match the expected WhisperX output pattern
	outputDir := filepath.Dir(expectedResultPath)
	files, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find result files: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no result files found in %s", outputDir)
	}

	// Filter out result.json (which is Parakeet/Canary format) and find WhisperX format
	var whisperxFile string
	for _, file := range files {
		if filepath.Base(file) != "result.json" {
			whisperxFile = file
			break
		}
	}

	if whisperxFile == "" {
		// Fall back to result.json if no other files found
		if _, err := os.Stat(expectedResultPath); err == nil {
			whisperxFile = expectedResultPath
		} else {
			return nil, fmt.Errorf("no WhisperX result files found in %s", outputDir)
		}
	}

	fmt.Printf("DEBUG: Using WhisperX result file: %s\n", whisperxFile)

	// Read result file
	resultData, err := os.ReadFile(whisperxFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %v", err)
	}

	// Parse JSON
	var result TranscriptResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %v", err)
	}

	return &result, nil
}

// runSortformerDiarization runs NVIDIA Sortformer diarization on the audio file
// and merges the diarization results with the existing WhisperX transcription
func (ws *WhisperXService) runSortformerDiarization(ctx context.Context, job *models.TranscriptionJob, outputDir string) error {
	fmt.Printf("DEBUG: Starting Sortformer diarization for job %s\n", job.ID)

	// Build diarization command
	parakeetDir := filepath.Join(ws.getEnvPath(), "parakeet")
	nemoDiarizeScript := filepath.Join(parakeetDir, "nemo_diarize.py")
	rttmOutputPath := filepath.Join(outputDir, "diarization.rttm")

	args := []string{
		"run", "--native-tls", "--project", parakeetDir, "python", nemoDiarizeScript,
	}

	// Add speaker constraints if specified
	if job.Parameters.MinSpeakers != nil {
		args = append(args, "--min-speakers", strconv.Itoa(*job.Parameters.MinSpeakers))
	}
	if job.Parameters.MaxSpeakers != nil {
		args = append(args, "--max-speakers", strconv.Itoa(*job.Parameters.MaxSpeakers))
	}

	// Add positional arguments: audio_file and output_file
	args = append(args, job.AudioPath, rttmOutputPath)

	fmt.Printf("DEBUG: Running diarization command: uv %v\n", args)

	// Execute diarization
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	configureCmdSysProcAttr(cmd)

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return fmt.Errorf("diarization was cancelled")
	}
	if err != nil {
		fmt.Printf("DEBUG: Diarization stderr/stdout: %s\n", string(output))
		return fmt.Errorf("diarization execution failed: %v", err)
	}

	fmt.Printf("DEBUG: Diarization completed successfully\n")

	// Now merge the diarization results with WhisperX transcription
	return ws.mergeDiarizationWithTranscription(outputDir, rttmOutputPath)
}

// mergeDiarizationWithTranscription merges RTTM diarization results with WhisperX transcription
func (ws *WhisperXService) mergeDiarizationWithTranscription(outputDir, rttmPath string) error {
	fmt.Printf("DEBUG: Merging diarization results with transcription\n")

	// Find the WhisperX result file
	files, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find result files: %v", err)
	}

	var whisperxFile string
	for _, file := range files {
		if filepath.Base(file) != "result.json" {
			whisperxFile = file
			break
		}
	}

	if whisperxFile == "" {
		return fmt.Errorf("no WhisperX result file found for merging")
	}

	// Read WhisperX transcription
	transcriptData, err := os.ReadFile(whisperxFile)
	if err != nil {
		return fmt.Errorf("failed to read transcription file: %v", err)
	}

	var transcript TranscriptResult
	if err := json.Unmarshal(transcriptData, &transcript); err != nil {
		return fmt.Errorf("failed to parse transcription JSON: %v", err)
	}

	// Read RTTM diarization results
	rttmData, err := os.ReadFile(rttmPath)
	if err != nil {
		return fmt.Errorf("failed to read RTTM file: %v", err)
	}

	// Parse RTTM format: SPEAKER <filename> 1 <start> <duration> <U> <U> <speaker_id> <U>
	type RTTMSegment struct {
		Start    float64
		End      float64
		Speaker  string
	}

	var rttmSegments []RTTMSegment
	lines := strings.Split(string(rttmData), "\n")
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

		rttmSegments = append(rttmSegments, RTTMSegment{
			Start:   start,
			End:     end,
			Speaker: speaker,
		})
	}

	fmt.Printf("DEBUG: Found %d RTTM segments\n", len(rttmSegments))
	
	// Debug: Show first few RTTM segments
	for i, rttm := range rttmSegments {
		if i >= 3 { // Show first 3 segments
			break
		}
		fmt.Printf("DEBUG: RTTM segment %d: start=%.2f, end=%.2f, speaker='%s'\n", i, rttm.Start, rttm.End, rttm.Speaker)
	}

	// Assign speakers to transcript segments based on timing overlap
	for i := range transcript.Segments {
		segment := &transcript.Segments[i]
		
		// Find the RTTM segment with maximum overlap
		maxOverlap := 0.0
		bestSpeaker := ""

		for _, rttm := range rttmSegments {
			// Calculate overlap between transcript segment and RTTM segment
			overlapStart := math.Max(segment.Start, rttm.Start)
			overlapEnd := math.Min(segment.End, rttm.End)
			overlap := math.Max(0, overlapEnd - overlapStart)

			if overlap > maxOverlap {
				maxOverlap = overlap
				bestSpeaker = rttm.Speaker
			}
		}

		if bestSpeaker != "" {
			segment.Speaker = &bestSpeaker
			fmt.Printf("DEBUG: Assigned speaker '%s' to segment %d (%.2f-%.2f)\n", bestSpeaker, i, segment.Start, segment.End)
		} else {
			// Assign a default speaker if no overlap found
			defaultSpeaker := "SPEAKER_00"
			segment.Speaker = &defaultSpeaker
			fmt.Printf("DEBUG: Assigned default speaker '%s' to segment %d (%.2f-%.2f) - no overlap found\n", defaultSpeaker, i, segment.Start, segment.End)
		}
	}

	// Also assign speakers to word-level segments if they exist
	for i := range transcript.Word {
		word := &transcript.Word[i]
		
		// Find the RTTM segment with maximum overlap
		maxOverlap := 0.0
		bestSpeaker := ""

		for _, rttm := range rttmSegments {
			// Calculate overlap between word and RTTM segment
			overlapStart := math.Max(word.Start, rttm.Start)
			overlapEnd := math.Min(word.End, rttm.End)
			overlap := math.Max(0, overlapEnd - overlapStart)

			if overlap > maxOverlap {
				maxOverlap = overlap
				bestSpeaker = rttm.Speaker
			}
		}

		if bestSpeaker != "" {
			word.Speaker = &bestSpeaker
		} else {
			// Assign a default speaker if no overlap found
			defaultSpeaker := "SPEAKER_00"
			word.Speaker = &defaultSpeaker
		}
	}

	fmt.Printf("DEBUG: Successfully merged diarization with transcription\n")

	// Write the merged result back to the file
	mergedData, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal merged transcript: %v", err)
	}

	if err := os.WriteFile(whisperxFile, mergedData, 0644); err != nil {
		return fmt.Errorf("failed to write merged transcript: %v", err)
	}

	fmt.Printf("DEBUG: Merged transcription saved to %s\n", whisperxFile)
	return nil
}
