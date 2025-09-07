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

// CanaryService handles NVIDIA Canary transcription
type CanaryService struct {
}

// NewCanaryService creates a new Canary service
func NewCanaryService(cfg *config.Config) *CanaryService {
	return &CanaryService{}
}

// CanaryResult represents the Canary output format
type CanaryResult struct {
	Transcription      string             `json:"transcription"`
	WordTimestamps     []CanaryWord       `json:"word_timestamps"`
	SegmentTimestamps  []CanarySegment    `json:"segment_timestamps"`
	AudioFile          string             `json:"audio_file"`
	SourceLanguage     string             `json:"source_language"`
	TargetLanguage     string             `json:"target_language"`
}

// CanaryWord represents word-level timestamps from Canary
type CanaryWord struct {
	Word        string  `json:"word"`
	StartOffset int     `json:"start_offset"`
	EndOffset   int     `json:"end_offset"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
}

// CanarySegment represents segment-level timestamps from Canary
type CanarySegment struct {
	Segment     string  `json:"segment"`
	StartOffset int     `json:"start_offset"`
	EndOffset   int     `json:"end_offset"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
}

// ProcessJob implements the JobProcessor interface
func (cs *CanaryService) ProcessJob(ctx context.Context, jobID string) error {
	// Call the enhanced version with a no-op register function
	return cs.ProcessJobWithProcess(ctx, jobID, func(*exec.Cmd) {})
}

// ProcessJobWithProcess implements the enhanced JobProcessor interface
func (cs *CanaryService) ProcessJobWithProcess(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error {
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

	// Preprocess audio for NVIDIA Canary (convert to 16kHz mono WAV)
	preprocessedAudioPath, err := cs.preprocessAudioForCanary(job.AudioPath, outputDir)
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

	// Build Canary command using the preprocessed audio
	resultPath := filepath.Join(outputDir, "result.json")
	args, err := cs.buildCanaryArgs(preprocessedAudioPath, resultPath, &job.Parameters)
	if err != nil {
		errMsg := fmt.Sprintf("failed to build command: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Create command with context for proper cancellation support
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	// Set working directory to the nvidia project directory (shared with parakeet)
	nvidiaPath := filepath.Join("whisperx-env", "parakeet")
	cmd.Dir = nvidiaPath
	
	// Configure process attributes for cross-platform kill behavior
	configureCmdSysProcAttr(cmd)
	
	// Register the process for immediate termination capability
	registerProcess(cmd)

	// Execute Canary with enhanced debugging
	fmt.Printf("DEBUG: Running Canary command: uv %v\n", args)
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
	
	// Check if nvidia environment exists
	if stat, err := os.Stat(nvidiaPath); err != nil {
		fmt.Printf("DEBUG: NVIDIA env error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: NVIDIA env exists, is dir: %v\n", stat.IsDir())
	}
	
	// Check if transcription script exists
	scriptPath := filepath.Join(nvidiaPath, "transcribe.py")
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
		errMsg := fmt.Sprintf("Canary execution failed: %v - Output: %s", err, string(output))
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Load and parse the result
	if err := cs.parseAndSaveResult(jobID, resultPath); err != nil {
		errMsg := fmt.Sprintf("failed to parse result: %v", err)
		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Success! Update execution status
	updateExecutionStatus(models.StatusCompleted, "")

	return nil
}

// buildCanaryArgs builds the Canary command arguments
func (cs *CanaryService) buildCanaryArgs(audioPath, outputFile string, params *models.WhisperXParams) ([]string, error) {
	// Convert audio path to absolute path since we'll be running from canary directory
	absAudioPath, err := filepath.Abs(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute audio path: %v", err)
	}
	
	// Convert output file to absolute path
	absOutputFile, err := filepath.Abs(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute output path: %v", err)
	}
	
	// Build command to run the transcription script with Canary model
	// Since cmd.Dir is set to nvidia directory, we use "." as project path
	args := []string{
		"run", "--native-tls", "--project", ".", "python", "transcribe.py",
		absAudioPath,
		"--model", "canary",  // Specify to use Canary model
		"--timestamps",       // Always include timestamps for consistency
		"--output", absOutputFile,
	}
	
	// Add source language parameter
	sourceLang := "en" // Default to English
	if params.Language != nil && *params.Language != "" {
		sourceLang = *params.Language
	}
	args = append(args, "--source-lang", sourceLang)
	
	// For now, we only support transcription (not translation)
	// Target language same as source language for transcription
	args = append(args, "--target-lang", sourceLang)
	
	fmt.Printf("DEBUG: Canary command: uv %v\n", args)
	fmt.Printf("DEBUG: Audio path (abs): %s\n", absAudioPath)
	fmt.Printf("DEBUG: Output path (abs): %s\n", absOutputFile)
	
	return args, nil
}

// parseAndSaveResult parses Canary output and saves to database
func (cs *CanaryService) parseAndSaveResult(jobID, resultPath string) error {
	// Read the result file
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return fmt.Errorf("failed to read result file: %v", err)
	}

	// Parse the Canary result
	var canaryResult CanaryResult
	if err := json.Unmarshal(data, &canaryResult); err != nil {
		return fmt.Errorf("failed to parse JSON result: %v", err)
	}

	// Convert Canary format to WhisperX format for database storage
	transcriptResult := cs.convertToWhisperXFormat(&canaryResult)

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

// convertToWhisperXFormat converts Canary output to WhisperX format
func (cs *CanaryService) convertToWhisperXFormat(canaryResult *CanaryResult) *TranscriptResult {
	var segments []Segment
	
	// Use Canary's segment timestamps directly as they come from the model
	if len(canaryResult.SegmentTimestamps) > 0 {
		segments = make([]Segment, len(canaryResult.SegmentTimestamps))
		for i, seg := range canaryResult.SegmentTimestamps {
			segments[i] = Segment{
				Start: seg.Start,
				End:   seg.End,
				Text:  strings.TrimSpace(seg.Segment),
			}
		}
	} else {
		// Fallback: create a single segment with the full transcription
		if canaryResult.Transcription != "" {
			segments = []Segment{
				{
					Start: 0.0,
					End:   0.0, // We don't know the duration, frontend can handle this
					Text:  strings.TrimSpace(canaryResult.Transcription),
				},
			}
		}
	}

	// Convert word-level timestamps
	words := make([]Word, len(canaryResult.WordTimestamps))
	for i, word := range canaryResult.WordTimestamps {
		words[i] = Word{
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
			Score: 1.0, // Canary doesn't provide confidence scores
			// No speaker information
		}
	}

	// Determine language from result or use detected source language
	language := canaryResult.SourceLanguage
	if language == "" {
		language = "en" // Default fallback
	}

	// Generate full text from segments
	fullText := ""
	if len(segments) > 0 {
		var texts []string
		for _, segment := range segments {
			texts = append(texts, segment.Text)
		}
		fullText = strings.Join(texts, " ")
	}

	return &TranscriptResult{
		Segments: segments,
		Word:     words,
		Language: language,
		Text:     fullText,
	}
}


// preprocessAudioForCanary converts audio to Canary-compatible format using ffmpeg
func (cs *CanaryService) preprocessAudioForCanary(inputPath, outputDir string) (string, error) {
	// Generate a unique temporary filename in the output directory
	baseName := filepath.Base(inputPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	tempFileName := fmt.Sprintf("%s_canary_temp.wav", nameWithoutExt)
	tempPath := filepath.Join(outputDir, tempFileName)
	
	fmt.Printf("DEBUG: Preprocessing audio for Canary\n")
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