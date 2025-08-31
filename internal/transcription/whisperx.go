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

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	assets "scriberr/internal/transcription/assets"
)

// WhisperXService handles WhisperX transcription
type WhisperXService struct {
    config *config.Config
}

// NewWhisperXService creates a new WhisperX service
func NewWhisperXService(cfg *config.Config) *WhisperXService {
	return &WhisperXService{
		config: cfg,
	}
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
	// Get the job from database
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("failed to get job: %v", err)
	}

	// Ensure Python environment is set up
	if err := ws.ensurePythonEnv(); err != nil {
		return fmt.Errorf("failed to setup Python environment: %v", err)
	}

	// Check if audio file exists
	if _, err := os.Stat(job.AudioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", job.AudioPath)
	}

	// Prepare output directory
	outputDir := filepath.Join("data", "transcripts", jobID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Build WhisperX command (handles both regular transcription and diarization)
	cmd, err := ws.buildWhisperXCommand(&job, outputDir)
	if err != nil {
		return fmt.Errorf("failed to build command: %v", err)
	}

	// Set context for cancellation support
	cmdWithCtx := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmdWithCtx.Env = cmd.Env

	// Execute WhisperX
	output, err := cmdWithCtx.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return fmt.Errorf("job was cancelled")
	}
	if err != nil {
		fmt.Printf("DEBUG: WhisperX stderr/stdout: %s\n", string(output))
		return fmt.Errorf("WhisperX execution failed: %v", err)
	}

	// Load and parse the result
	resultPath := filepath.Join(outputDir, "result.json")
	if err := ws.parseAndSaveResult(jobID, resultPath); err != nil {
		return fmt.Errorf("failed to parse result: %v", err)
	}

	return nil
}

// ensurePythonEnv ensures the Python environment is set up
func (ws *WhisperXService) ensurePythonEnv() error {
    envPath := ws.getEnvPath()
    // Ensure base directory exists
    if err := os.MkdirAll(envPath, 0755); err != nil {
        return fmt.Errorf("failed to create environment directory: %v", err)
    }

    // Always write embedded assets (pyproject) to keep them in sync
    _ = ws.writeEmbeddedFile("pyproject.toml", filepath.Join(envPath, "pyproject.toml"))

    // If we have a pyproject, prefer syncing via uv
    if _, err := os.Stat(filepath.Join(envPath, "pyproject.toml")); err == nil {
        if err := ws.uvSync(envPath); err == nil {
            return nil
        }
        // fall through to import check if sync failed
    }

    // Check if WhisperX import works; if not, try to install via fallback
    cmd := exec.Command(ws.getUVPath(), "run", "--native-tls", "--project", envPath, "python", "-c", "import whisperx")
    if err := cmd.Run(); err != nil {
        return ws.installWhisperX()
    }
    return nil
}

// createPythonEnv creates a new Python environment
func (ws *WhisperXService) createPythonEnv() error {
    envPath := ws.getEnvPath()
    
    // Create directory
    if err := os.MkdirAll(envPath, 0755); err != nil {
        return fmt.Errorf("failed to create environment directory: %v", err)
    }
    
    // Write embedded pyproject file
    if err := ws.writeEmbeddedFile("pyproject.toml", filepath.Join(envPath, "pyproject.toml")); err != nil {
        return err
    }

    // Sync dependencies using uv
    if err := ws.uvSync(envPath); err != nil {
        return fmt.Errorf("failed to sync uv project: %v", err)
    }
    return nil
}

// installWhisperX installs WhisperX and dependencies
func (ws *WhisperXService) installWhisperX() error {
    envPath := ws.getEnvPath()
    
    // Install WhisperX and diarization dependencies
    cmd := exec.Command(ws.getUVPath(), "add", "--native-tls", "--project", envPath, 
        "git+https://github.com/m-bain/whisperX.git", 
        "torch", "torchaudio", "numpy", "pandas", 
        "pyannote.audio", "faster-whisper")
    cmd.Dir = envPath
    
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to install WhisperX: %v", err)
    }
    
    return nil
}

// uvSync runs `uv sync` for the given project path
func (ws *WhisperXService) uvSync(projectPath string) error {
    cmd := exec.Command(ws.getUVPath(), "sync", "--native-tls", "--project", projectPath)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("uv sync failed: %v: %s", err, strings.TrimSpace(string(out)))
    }
    return nil
}

// writeEmbeddedFile writes an embedded asset to disk
func (ws *WhisperXService) writeEmbeddedFile(name, dest string) error {
    data, err := assets.FS.ReadFile(name)
    if err != nil {
        // asset missing in the binary; not fatal
        return nil
    }
    if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
        return err
    }
    return os.WriteFile(dest, data, 0644)
}

// InitEmbeddedPythonEnv initializes the Python env on app start (blocking).
// Assumes uv is installed and accessible via config.UVPath.
func (ws *WhisperXService) InitEmbeddedPythonEnv() error {
    if err := ws.ensurePythonEnv(); err != nil {
        return err
    }
    return nil
}

// buildWhisperXCommand builds the WhisperX command
func (ws *WhisperXService) buildWhisperXCommand(job *models.TranscriptionJob, outputDir string) (*exec.Cmd, error) {
	p := job.Parameters
	
	// Debug: log diarization status
	fmt.Printf("DEBUG: Job ID %s, Diarize parameter: %v, Job Diarization field: %v\n", job.ID, p.Diarize, job.Diarization)
	
	// Use WhisperX CLI for both regular transcription and diarization
	args := []string{
		"run", "--native-tls", "--project", ws.config.WhisperXEnv, "python", "-m", "whisperx",
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

    cmd := exec.Command(ws.getUVPath(), args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	
	// Debug: log the command being executed
	fmt.Printf("DEBUG: WhisperX command: %s %v\n", ws.config.UVPath, args)
	
	return cmd, nil
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

	// Update the job in the database
	if err := database.DB.Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Update("transcript", &transcriptStr).Error; err != nil {
		return fmt.Errorf("failed to update job transcript: %v", err)
	}

    return nil
}

// getEnvPath returns the path used for the WhisperX environment, defaulting
// to data/whisperx-env when not configured.
func (ws *WhisperXService) getEnvPath() string {
    // Always use a stable default under the app's data directory.
    return filepath.Join("data", "whisperx-env")
}

// getUVPath returns the uv binary path, defaulting to "uv".
func (ws *WhisperXService) getUVPath() string { return "uv" }

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
