package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
)

// ParakeetService handles NVIDIA Parakeet transcription
type ParakeetService struct {
}

// NewParakeetService creates a new Parakeet service
func NewParakeetService(cfg *config.Config) *ParakeetService {
	return &ParakeetService{}
}

// ParakeetResult represents the Parakeet output format
type ParakeetResult struct {
	Transcription      string             `json:"transcription"`
	WordTimestamps     []ParakeetWord     `json:"word_timestamps"`
	SegmentTimestamps  []ParakeetSegment  `json:"segment_timestamps"`
	AudioFile          string             `json:"audio_file"`
}

// ParakeetWord represents word-level timestamps from Parakeet
type ParakeetWord struct {
	Word        string  `json:"word"`
	StartOffset int     `json:"start_offset"`
	EndOffset   int     `json:"end_offset"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
}

// ParakeetSegment represents segment-level timestamps from Parakeet
type ParakeetSegment struct {
	Segment     string  `json:"segment"`
	StartOffset int     `json:"start_offset"`
	EndOffset   int     `json:"end_offset"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
}

// ProcessJob implements the JobProcessor interface
func (ps *ParakeetService) ProcessJob(ctx context.Context, jobID string) error {
	// Call the enhanced version with a no-op register function
	return ps.ProcessJobWithProcess(ctx, jobID, func(*exec.Cmd) {})
}

// ProcessJobWithProcess implements the enhanced JobProcessor interface
func (ps *ParakeetService) ProcessJobWithProcess(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error {
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
		ActualParameters:   job.Parameters,
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

	// Ensure Python environment is set up (reuse WhisperX service logic)
	ws := &WhisperXService{}
	if err := ws.ensurePythonEnv(); err != nil {
		errMsg := fmt.Sprintf("failed to setup Python environment: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Check if original audio file exists
	if _, err := os.Stat(job.AudioPath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("audio file not found: %s", job.AudioPath)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Prepare output directory first
	outputDir := filepath.Join("data", "transcripts", jobID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		errMsg := fmt.Sprintf("failed to create output directory: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Preprocess audio for NVIDIA Parakeet (convert to 16kHz mono WAV)
	preprocessedAudioPath, err := ps.preprocessAudioForParakeet(job.AudioPath, outputDir)
	if err != nil {
		errMsg := fmt.Sprintf("failed to preprocess audio: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}
	
	// Ensure cleanup of temporary file on function exit
	defer func() {
		if preprocessedAudioPath != "" {
			if err := os.Remove(preprocessedAudioPath); err != nil {
				fmt.Printf("DEBUG: Warning - failed to cleanup temporary audio file %s: %v\n", preprocessedAudioPath, err)
			} else {
				fmt.Printf("DEBUG: Cleaned up temporary audio file: %s\n", preprocessedAudioPath)
			}
		}
	}()

	// Build Parakeet command using the preprocessed audio
	resultPath := filepath.Join(outputDir, "result.json")
	args, err := ps.buildParakeetArgs(preprocessedAudioPath, resultPath)
	if err != nil {
		errMsg := fmt.Sprintf("failed to build command: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Create command with context for proper cancellation support
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	// Set working directory to the parakeet project directory
	parakeetPath := filepath.Join("whisperx-env", "parakeet")
	cmd.Dir = parakeetPath
	
	// Configure process attributes for cross-platform kill behavior
	configureCmdSysProcAttr(cmd)
	
	// Register the process for immediate termination capability
	registerProcess(cmd)

	// Execute Parakeet with enhanced debugging
	fmt.Printf("DEBUG: Running Parakeet command: uv %v\n", args)
	fmt.Printf("DEBUG: Working directory: %s\n", cmd.Dir)
	fmt.Printf("DEBUG: Audio file path: %s\n", job.AudioPath)
	fmt.Printf("DEBUG: Output path: %s\n", resultPath)
	fmt.Printf("DEBUG: Job parameters: %+v\n", job.Parameters)
	
	// Check if audio file is accessible
	if stat, err := os.Stat(job.AudioPath); err != nil {
		fmt.Printf("DEBUG: Audio file stat error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Audio file size: %d bytes\n", stat.Size())
	}
	
	// Check if parakeet environment exists
	parakeetPath = filepath.Join("whisperx-env", "parakeet")
	if stat, err := os.Stat(parakeetPath); err != nil {
		fmt.Printf("DEBUG: Parakeet env error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Parakeet env exists, is dir: %v\n", stat.IsDir())
	}
	
	// Check if transcription script exists
	scriptPath := filepath.Join(parakeetPath, "transcribe.py")
	if stat, err := os.Stat(scriptPath); err != nil {
		fmt.Printf("DEBUG: Transcription script error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Transcription script size: %d bytes\n", stat.Size())
	}
	
	output, err := cmd.CombinedOutput()
	fmt.Printf("DEBUG: Command exit code: %v\n", err)
	fmt.Printf("DEBUG: Command output:\n%s\n", string(output))
	
	if ctx.Err() == context.Canceled {
		errMsg := "job was cancelled"
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}
	if err != nil {
		errMsg := fmt.Sprintf("Parakeet execution failed: %v - Output: %s", err, string(output))
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Load and parse the result
	if err := ps.parseAndSaveResult(jobID, resultPath); err != nil {
		errMsg := fmt.Sprintf("failed to parse result: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Success! Update execution status
	updateExecutionStatus(models.StatusCompleted, "")

	return nil
}

// buildParakeetArgs builds the Parakeet command arguments
func (ps *ParakeetService) buildParakeetArgs(audioPath, outputFile string) ([]string, error) {
	// Convert audio path to absolute path since we'll be running from parakeet directory
	absAudioPath, err := filepath.Abs(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute audio path: %v", err)
	}
	
	// Convert output file to absolute path
	absOutputFile, err := filepath.Abs(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute output path: %v", err)
	}
	
	// Build command to run the Parakeet transcription script
	// Since cmd.Dir is set to parakeet directory, we use "." as project path
	args := []string{
		"run", "--native-tls", "--project", ".", "python", "transcribe.py",
		absAudioPath,
		"--timestamps",  // Always include timestamps for consistency
		"--output", absOutputFile,
	}

	// Add model path if we want to specify a custom model location
	// The script will use the default downloaded model if not specified
	
	fmt.Printf("DEBUG: Fixed Parakeet command: uv %v\n", args)
	fmt.Printf("DEBUG: Audio path (abs): %s\n", absAudioPath)
	fmt.Printf("DEBUG: Output path (abs): %s\n", absOutputFile)
	
	return args, nil
}

// preprocessAudioForParakeet converts audio to Parakeet-compatible format using ffmpeg
func (ps *ParakeetService) preprocessAudioForParakeet(inputPath, outputDir string) (string, error) {
	// Generate a unique temporary filename in the output directory
	baseName := filepath.Base(inputPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	tempFileName := fmt.Sprintf("%s_parakeet_temp.wav", nameWithoutExt)
	tempPath := filepath.Join(outputDir, tempFileName)
	
	fmt.Printf("DEBUG: Preprocessing audio for Parakeet\n")
	fmt.Printf("DEBUG: Input: %s\n", inputPath)
	fmt.Printf("DEBUG: Output: %s\n", tempPath)
	
	// Build ffmpeg command: ffmpeg -i input.mp3 -ar 16000 -ac 1 -c:a pcm_s16le output.wav
	cmd := exec.Command("ffmpeg", 
		"-i", inputPath,           // Input file
		"-ar", "16000",            // Sample rate 16kHz
		"-ac", "1",                // Mono (1 channel)
		"-c:a", "pcm_s16le",       // PCM 16-bit little-endian codec
		"-y",                      // Overwrite output file if it exists
		tempPath,                  // Output file
	)
	
	// Capture ffmpeg output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("DEBUG: ffmpeg failed: %v\n", err)
		fmt.Printf("DEBUG: ffmpeg output:\n%s\n", string(output))
		return "", fmt.Errorf("ffmpeg preprocessing failed: %v - %s", err, string(output))
	}
	
	// Verify the output file was created
	if stat, err := os.Stat(tempPath); err != nil {
		return "", fmt.Errorf("preprocessed audio file not created: %v", err)
	} else {
		fmt.Printf("DEBUG: Preprocessed audio created successfully (size: %d bytes)\n", stat.Size())
	}
	
	return tempPath, nil
}

// parseAndSaveResult parses Parakeet output and saves to database
func (ps *ParakeetService) parseAndSaveResult(jobID, resultPath string) error {
	// Read the result file
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return fmt.Errorf("failed to read result file: %v", err)
	}

	// Parse the Parakeet result
	var parakeetResult ParakeetResult
	if err := json.Unmarshal(data, &parakeetResult); err != nil {
		return fmt.Errorf("failed to parse JSON result: %v", err)
	}

	// Convert Parakeet format to WhisperX format for database storage
	transcriptResult := ps.convertToWhisperXFormat(&parakeetResult)

	// Convert to JSON string for database storage
	transcriptJSON, err := json.Marshal(transcriptResult)
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

// convertToWhisperXFormat converts Parakeet output to WhisperX format
func (ps *ParakeetService) convertToWhisperXFormat(parakeetResult *ParakeetResult) *TranscriptResult {
	// Convert segments
	segments := make([]Segment, len(parakeetResult.SegmentTimestamps))
	for i, seg := range parakeetResult.SegmentTimestamps {
		segments[i] = Segment{
			Start: seg.Start,
			End:   seg.End,
			Text:  strings.TrimSpace(seg.Segment),
			// No speaker information from Parakeet (diarization not supported)
		}
	}

	// Convert word-level timestamps
	words := make([]Word, len(parakeetResult.WordTimestamps))
	for i, word := range parakeetResult.WordTimestamps {
		words[i] = Word{
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
			Score: 1.0, // Parakeet doesn't provide confidence scores
			// No speaker information
		}
	}

	return &TranscriptResult{
		Segments: segments,
		Word:     words,
		Language: "en", // Parakeet only supports English
	}
}