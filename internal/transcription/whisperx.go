package transcription

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

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
)

// WhisperXService handles WhisperX transcription
type WhisperXService struct {
}

// NewWhisperXService creates a new WhisperX service
func NewWhisperXService(cfg *config.Config) *WhisperXService {
	return &WhisperXService{}
}

// TranscriptResult represents the WhisperX output format
type TranscriptResult struct {
	Segments []Segment `json:"segments"`
	Word     []Word    `json:"word_segments,omitempty"`
	Language string    `json:"language"`
}

// Segment represents a transcript segment
type Segment struct {
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Text     string  `json:"text"`
	Speaker  *string `json:"speaker,omitempty"`
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
	// Get the job from database to check model family
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("failed to get job: %v", err)
	}

	// Route to appropriate service based on model family
	if job.Parameters.ModelFamily == "nvidia" {
		fmt.Printf("DEBUG: Routing job %s to Parakeet service\n", jobID)
		parakeetService := NewParakeetService(nil)
		return parakeetService.ProcessJobWithProcess(ctx, jobID, registerProcess)
	}

	// Default to WhisperX processing
	fmt.Printf("DEBUG: Processing job %s with WhisperX\n", jobID)
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
		Status:            models.StatusProcessing,
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

// ensurePythonEnv ensures the Python environment is set up by cloning WhisperX from git and setting up Parakeet
func (ws *WhisperXService) ensurePythonEnv() error {
    envPath := ws.getEnvPath()
    whisperxPath := filepath.Join(envPath, "WhisperX")
    parakeetPath := filepath.Join(envPath, "parakeet")
    
    // Get absolute paths for debugging
    absEnvPath, _ := filepath.Abs(envPath)
    absWhisperxPath, _ := filepath.Abs(whisperxPath)
    absParakeetPath, _ := filepath.Abs(parakeetPath)
    workingDir, _ := os.Getwd()
    
    fmt.Printf("DEBUG: Current working directory: %s\n", workingDir)
    fmt.Printf("DEBUG: Relative WhisperX path: %s\n", whisperxPath)
    fmt.Printf("DEBUG: Absolute WhisperX path: %s\n", absWhisperxPath)
    fmt.Printf("DEBUG: Absolute Parakeet path: %s\n", absParakeetPath)
    fmt.Printf("DEBUG: Absolute env path: %s\n", absEnvPath)
    
    // Check WhisperX and Parakeet environments independently
    whisperxCmd := exec.Command("uv", "run", "--native-tls", "--project", whisperxPath, "python", "-c", "import whisperx")
    parakeetCmd := exec.Command("uv", "run", "--native-tls", "--project", parakeetPath, "python", "-c", "import nemo.collections.asr")
    
    whisperxWorking := whisperxCmd.Run() == nil
    parakeetEnvWorking := parakeetCmd.Run() == nil
    
    // Check if Parakeet model exists
    modelPath := filepath.Join(parakeetPath, "parakeet-tdt-0.6b-v3.nemo")
    parakeetModelExists := false
    if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
        parakeetModelExists = true
    }

    fmt.Printf("DEBUG: Environment status - WhisperX: %v, Parakeet Env: %v, Parakeet Model: %v\n", 
        whisperxWorking, parakeetEnvWorking, parakeetModelExists)

    // If everything is working, we're done
    if whisperxWorking && parakeetEnvWorking && parakeetModelExists {
        fmt.Printf("DEBUG: WhisperX and Parakeet fully set up and working\n")
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

    // Setup Parakeet environment if needed
    if !parakeetEnvWorking {
        fmt.Printf("DEBUG: Setting up Parakeet environment at: %s\n", parakeetPath)
        
        // Remove existing Parakeet directory if it exists
        if err := os.RemoveAll(parakeetPath); err != nil {
            return fmt.Errorf("failed to remove existing Parakeet environment: %v", err)
        }

        if err := ws.setupParakeetEnv(parakeetPath); err != nil {
            return fmt.Errorf("failed to setup Parakeet environment: %v", err)
        }
        
        fmt.Printf("DEBUG: Parakeet environment setup completed\n")
    } else {
        fmt.Printf("DEBUG: Parakeet environment already working, skipping setup\n")
    }

    // Download Parakeet model if needed (independent of environment setup)
    if !parakeetModelExists {
        fmt.Printf("DEBUG: Downloading Parakeet model\n")
        if err := ws.downloadParakeetModel(parakeetPath); err != nil {
            return fmt.Errorf("failed to download Parakeet model: %v", err)
        }
        fmt.Printf("DEBUG: Parakeet model download completed\n")
    } else {
        fmt.Printf("DEBUG: Parakeet model already exists, skipping download\n")
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

    // Create pyproject.toml for Parakeet dependencies
    pyprojectContent := `[project]
name = "parakeet-transcription"
version = "0.1.0"
description = "Audio transcription using NVIDIA Parakeet models"
requires-python = ">=3.11"
dependencies = [
    "nemo_toolkit[asr]",
    "torch",
    "torchaudio",
    "librosa",
    "soundfile",
    "ml_dtypes>=0.3.1,<0.5.0",
    "onnx>=1.15.0,<1.18.0",
]
`
    pyprojectPath := filepath.Join(parakeetPath, "pyproject.toml")
    if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
        return fmt.Errorf("failed to write parakeet pyproject.toml: %v", err)
    }

    // Create the transcription script
    transcribeScript := `#!/usr/bin/env python3
"""
Audio transcription script using NVIDIA Parakeet TDT 0.6B v3 model.
Supports 25 European languages with automatic language detection.
"""

import argparse
import json
import sys
from pathlib import Path
import nemo.collections.asr as nemo_asr


def transcribe_audio(
    audio_path: str, timestamps: bool = False, output_file: str = None, model_path: str = None
):
    """
    Transcribe audio file using NVIDIA Parakeet model.

    Args:
        audio_path: Path to audio file (.wav or .flac)
        timestamps: Whether to include timestamps in output
        output_file: Optional output file path for results
        model_path: Path to the Parakeet model file
    """
    if not model_path:
        model_path = "./parakeet-tdt-0.6b-v3.nemo"
    
    print(f"Loading NVIDIA Parakeet model from: {model_path}")
    asr_model = nemo_asr.models.ASRModel.restore_from(model_path)

    print(f"Transcribing: {audio_path}")

    if timestamps:
        output = asr_model.transcribe([audio_path], timestamps=True)

        # Extract text and timestamps
        text = output[0].text
        word_timestamps = output[0].timestamp["word"]
        segment_timestamps = output[0].timestamp["segment"]

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
                "audio_file": audio_path
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
        output = asr_model.transcribe([audio_path])
        text = output[0].text
        print(f"\nTranscription: {text}")

        if output_file:
            if output_file.endswith('.json'):
                result_data = {
                    "transcription": text,
                    "audio_file": audio_path
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
        description="Transcribe audio using NVIDIA Parakeet TDT 0.6B v3 model"
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
        "--model-path", help="Path to the Parakeet model file"
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
            model_path=args.model_path,
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
        "-L",                    // Follow redirects
        "--progress-bar",        // Show progress bar
        "--create-dirs",         // Create directories if needed
        "-o", tempPath,          // Output to temp file
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
	if p.Diarize {
		args = append(args, "--diarize")
		if p.MinSpeakers != nil {
			args = append(args, "--min_speakers", strconv.Itoa(*p.MinSpeakers))
		}
		if p.MaxSpeakers != nil {
			args = append(args, "--max_speakers", strconv.Itoa(*p.MaxSpeakers))
		}
		args = append(args, "--diarize_model", p.DiarizeModel)
		if p.SpeakerEmbeddings {
			args = append(args, "--speaker_embeddings")
		}
	}

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

	// Diarization settings
	if p.Diarize {
		args = append(args, "--diarize")
		if p.MinSpeakers != nil {
			args = append(args, "--min_speakers", strconv.Itoa(*p.MinSpeakers))
		}
		if p.MaxSpeakers != nil {
			args = append(args, "--max_speakers", strconv.Itoa(*p.MaxSpeakers))
		}
		args = append(args, "--diarize_model", p.DiarizeModel)
		if p.SpeakerEmbeddings {
			args = append(args, "--speaker_embeddings")
		}
	}

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
	
	// Check if result.json exists (from diarization script)
	if _, err := os.Stat(resultPath); err == nil {
		resultFile = resultPath
	} else {
		// Find the actual result file (WhisperX creates files based on input filename)
		files, err := filepath.Glob(filepath.Join(filepath.Dir(resultPath), "*.json"))
		if err != nil {
			return fmt.Errorf("failed to find result files: %v", err)
		}
		
		if len(files) == 0 {
			return fmt.Errorf("no result files found")
		}
		
		// Use the first JSON file found
		resultFile = files[0]
	}
	
	// Read the result file
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return fmt.Errorf("failed to read result file: %v", err)
	}

	// Parse the result
	var result TranscriptResult
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse JSON result: %v", err)
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
