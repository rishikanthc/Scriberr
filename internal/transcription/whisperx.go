package transcription

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
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
func (ws *WhisperXService) ProcessJob(jobID string) error {
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

	// Build WhisperX command
	cmd, err := ws.buildWhisperXCommand(&job, outputDir)
	if err != nil {
		return fmt.Errorf("failed to build command: %v", err)
	}

	// Execute WhisperX
	if err := cmd.Run(); err != nil {
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
	envPath := ws.config.WhisperXEnv
	
	// Check if environment exists
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return ws.createPythonEnv()
	}
	
	// Check if WhisperX is installed
	cmd := exec.Command(ws.config.UVPath, "run", "--project", envPath, "python", "-c", "import whisperx")
	if err := cmd.Run(); err != nil {
		return ws.installWhisperX()
	}
	
	return nil
}

// createPythonEnv creates a new Python environment
func (ws *WhisperXService) createPythonEnv() error {
	envPath := ws.config.WhisperXEnv
	
	// Create directory
	if err := os.MkdirAll(envPath, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %v", err)
	}
	
	// Initialize uv project
	cmd := exec.Command(ws.config.UVPath, "init", "--python", "3.9", envPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize uv project: %v", err)
	}
	
	return ws.installWhisperX()
}

// installWhisperX installs WhisperX and dependencies
func (ws *WhisperXService) installWhisperX() error {
	envPath := ws.config.WhisperXEnv
	
	// Install WhisperX
	cmd := exec.Command(ws.config.UVPath, "add", "--project", envPath, 
		"git+https://github.com/m-bain/whisperX.git", 
		"torch", "torchaudio", "numpy", "pandas")
	cmd.Dir = envPath
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install WhisperX: %v", err)
	}
	
	return nil
}

// buildWhisperXCommand builds the WhisperX command
func (ws *WhisperXService) buildWhisperXCommand(job *models.TranscriptionJob, outputDir string) (*exec.Cmd, error) {
	args := []string{
		"run", "--project", ws.config.WhisperXEnv, "python", "-m", "whisperx",
		job.AudioPath,
		"--output_dir", outputDir,
		"--output_format", "json",
		"--model", job.Parameters.Model,
		"--batch_size", strconv.Itoa(job.Parameters.BatchSize),
		"--compute_type", job.Parameters.ComputeType,
		"--device", job.Parameters.Device,
	}

	if job.Parameters.Language != nil {
		args = append(args, "--language", *job.Parameters.Language)
	}

	if job.Parameters.VadFilter {
		args = append(args, "--vad_filter")
		args = append(args, "--vad_onset", fmt.Sprintf("%.3f", job.Parameters.VadOnset))
		args = append(args, "--vad_offset", fmt.Sprintf("%.3f", job.Parameters.VadOffset))
	}

	if job.Diarization {
		args = append(args, "--diarize")
		if job.Parameters.MinSpeakers != nil {
			args = append(args, "--min_speakers", strconv.Itoa(*job.Parameters.MinSpeakers))
		}
		if job.Parameters.MaxSpeakers != nil {
			args = append(args, "--max_speakers", strconv.Itoa(*job.Parameters.MaxSpeakers))
		}
	}

	cmd := exec.Command(ws.config.UVPath, args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	
	return cmd, nil
}

// parseAndSaveResult parses WhisperX output and saves to database
func (ws *WhisperXService) parseAndSaveResult(jobID, resultPath string) error {
	// Find the actual result file (WhisperX creates files based on input filename)
	files, err := filepath.Glob(filepath.Join(filepath.Dir(resultPath), "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find result files: %v", err)
	}
	
	if len(files) == 0 {
		return fmt.Errorf("no result files found")
	}
	
	// Use the first JSON file found
	resultFile := files[0]
	
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