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
	Transcription     string          `json:"transcription"`
	WordTimestamps    []CanaryWord    `json:"word_timestamps"`
	SegmentTimestamps []CanarySegment `json:"segment_timestamps"`
	AudioFile         string          `json:"audio_file"`
	SourceLanguage    string          `json:"source_language"`
	TargetLanguage    string          `json:"target_language"`
	Diarized          bool            `json:"diarized"`
}

// CanaryWord represents word-level timestamps from Canary
type CanaryWord struct {
	Word        string  `json:"word"`
	StartOffset int     `json:"start_offset"`
	EndOffset   int     `json:"end_offset"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Speaker     string  `json:"speaker,omitempty"`
}

// CanarySegment represents segment-level timestamps from Canary
type CanarySegment struct {
	Segment     string  `json:"segment"`
	StartOffset int     `json:"start_offset"`
	EndOffset   int     `json:"end_offset"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Speaker     string  `json:"speaker,omitempty"`
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
	// In Docker environments, we may encounter permission issues, so we'll try preprocessing
	// but fall back to using the original file if preprocessing fails
	preprocessedAudioPath, err := cs.preprocessAudioForCanary(job.AudioPath, outputDir)
	if err != nil {
		fmt.Printf("WARNING: Audio preprocessing failed: %v\n", err)
		fmt.Printf("WARNING: Attempting to use original audio file directly\n")

		// Check if original file is in a format that Canary might accept
		ext := strings.ToLower(filepath.Ext(job.AudioPath))
		if ext == ".wav" || ext == ".flac" {
			fmt.Printf("DEBUG: Original file is %s format, trying direct processing\n", ext)
			preprocessedAudioPath = job.AudioPath // Use original file
		} else {
			// For non-WAV/FLAC files, we really need preprocessing
			errMsg := fmt.Sprintf("failed to preprocess audio and original format (%s) may not be supported: %v", ext, err)
			updateExecutionStatus(models.StatusFailed, errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	// Ensure cleanup of temporary file on function exit (only if we created a temp file)
	defer func() {
		if preprocessedAudioPath != "" && preprocessedAudioPath != job.AudioPath {
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

	// Check available memory before running Canary (which requires ~8-12GB)
	if err := cs.checkMemoryAvailability(); err != nil {
		fmt.Printf("WARNING: %v\n", err)
	}

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

		// Check for OOM kill (exit status 137 = SIGKILL, usually due to out of memory)
		if strings.Contains(err.Error(), "exit status 137") {
			errMsg = fmt.Sprintf("Canary model was killed due to insufficient memory (OOM). Exit status 137 indicates the container needs more RAM. Current recommendation: 32GB. Error: %v - Output: %s", err, string(output))
		}

		updateExecutionStatus(models.StatusFailed, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Run diarization if enabled
	fmt.Printf("DEBUG: Checking diarization conditions - Diarize: %v, DiarizeModel: %s\n", job.Parameters.Diarize, job.Parameters.DiarizeModel)
	if job.Parameters.HfToken != nil {
		fmt.Printf("DEBUG: HfToken value: %s\n", *job.Parameters.HfToken)
	}

	if job.Parameters.Diarize {
		// Check if we have the required tokens/credentials for the chosen model
		canRunDiarization := false
		
		if job.Parameters.DiarizeModel == "nvidia_sortformer" {
			// NVIDIA Sortformer doesn't need HF token, can always run
			canRunDiarization = true
			fmt.Printf("DEBUG: NVIDIA Sortformer selected, no token required\n")
		} else {
			// Pyannote (default) requires HF token
			if job.Parameters.HfToken != nil && *job.Parameters.HfToken != "" {
				canRunDiarization = true
				fmt.Printf("DEBUG: Pyannote selected, HF token available\n")
			} else {
				fmt.Printf("DEBUG: Pyannote selected but no HF token available\n")
			}
		}
		
		if canRunDiarization {
			fmt.Printf("DEBUG: Running diarization for Canary job %s with model: %s\n", jobID, job.Parameters.DiarizeModel)
			if err := cs.runDiarization(job.AudioPath, resultPath, job.Parameters); err != nil {
				fmt.Printf("WARNING: Diarization failed: %v. Continuing with transcript without speaker information.\n", err)
				// Don't fail the job, just continue without diarization
			}
		} else {
			fmt.Printf("DEBUG: Diarization enabled but credentials not available for model %s\n", job.Parameters.DiarizeModel)
		}
	} else {
		fmt.Printf("DEBUG: Diarization disabled - skipping diarization\n")
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

// TranscribeAudioFile transcribes an audio file directly without database operations
// This is used for multi-track transcription where each track is processed individually
func (cs *CanaryService) TranscribeAudioFile(ctx context.Context, audioPath string, params models.WhisperXParams) (*TranscriptResult, error) {
	// Ensure Python environment is set up
	ws := &WhisperXService{}
	if err := ws.ensurePythonEnv(); err != nil {
		return nil, fmt.Errorf("failed to setup Python environment: %v", err)
	}

	// Check if audio file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("audio file not found: %s", audioPath)
	}

	// Create temporary directory for this transcription
	tempDir := filepath.Join("data", "temp", fmt.Sprintf("track_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory

	// Preprocess audio for Canary
	preprocessedAudioPath, err := cs.preprocessAudioForCanary(audioPath, tempDir)
	if err != nil {
		// Fall back to original audio if preprocessing fails
		fmt.Printf("WARNING: Audio preprocessing failed: %v\n", err)
		ext := strings.ToLower(filepath.Ext(audioPath))
		if ext == ".wav" || ext == ".flac" {
			preprocessedAudioPath = audioPath
		} else {
			return nil, fmt.Errorf("failed to preprocess audio and original format (%s) may not be supported: %v", ext, err)
		}
	}

	// Ensure cleanup of temporary file (only if we created a temp file)
	defer func() {
		if preprocessedAudioPath != "" && preprocessedAudioPath != audioPath {
			os.Remove(preprocessedAudioPath)
		}
	}()

	// Build Canary command
	resultPath := filepath.Join(tempDir, "result.json")
	args, err := cs.buildCanaryArgs(preprocessedAudioPath, resultPath, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %v", err)
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	nvidiaPath := filepath.Join("whisperx-env", "parakeet")
	cmd.Dir = nvidiaPath

	// Configure process attributes
	configureCmdSysProcAttr(cmd)

	// Check memory availability for Canary
	if err := cs.checkMemoryAvailability(); err != nil {
		fmt.Printf("WARNING: %v\n", err)
	}

	// Execute Canary
	fmt.Printf("DEBUG: Running Canary for track: uv %v\n", args)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("transcription was cancelled")
	}
	if err != nil {
		errMsg := fmt.Sprintf("Canary execution failed: %v - Output: %s", err, string(output))
		// Check for OOM kill specifically for Canary
		if strings.Contains(err.Error(), "exit status 137") {
			errMsg = fmt.Sprintf("Canary model was killed due to insufficient memory (OOM). Exit status 137 indicates the container needs more RAM. Current recommendation: 32GB. Error: %v - Output: %s", err, string(output))
		}
		return nil, fmt.Errorf(errMsg)
	}

	// Parse the result file
	return cs.parseResultFile(resultPath)
}

// parseResultFile parses the Canary result file and returns TranscriptResult
func (cs *CanaryService) parseResultFile(resultPath string) (*TranscriptResult, error) {
	// Read the result file
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %v", err)
	}

	// Parse the Canary result
	var canaryResult CanaryResult
	if err := json.Unmarshal(data, &canaryResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %v", err)
	}

	// Convert to standard TranscriptResult format
	return cs.convertToWhisperXFormat(&canaryResult), nil
}

// buildCanaryArgs builds the Canary command arguments
func (cs *CanaryService) buildCanaryArgs(audioPath, outputFile string, params *models.WhisperXParams) ([]string, error) {
	// Enhanced debugging for Docker path resolution
	workingDir, _ := os.Getwd()
	fmt.Printf("DEBUG: Current working directory: %s\n", workingDir)
	fmt.Printf("DEBUG: Input audio path: %s\n", audioPath)
	fmt.Printf("DEBUG: Input output file: %s\n", outputFile)

	// Check if input audio file exists before processing
	if stat, err := os.Stat(audioPath); err != nil {
		fmt.Printf("DEBUG: Audio file stat failed: %v\n", err)
		return nil, fmt.Errorf("audio file not accessible: %s - %v", audioPath, err)
	} else {
		fmt.Printf("DEBUG: Audio file exists, size: %d bytes\n", stat.Size())
	}

	// Convert audio path to absolute path since we'll be running from canary directory
	absAudioPath, err := filepath.Abs(audioPath)
	if err != nil {
		fmt.Printf("DEBUG: Failed to convert audio path to absolute: %v\n", err)
		return nil, fmt.Errorf("failed to get absolute audio path: %v", err)
	}

	// Convert output file to absolute path
	absOutputFile, err := filepath.Abs(outputFile)
	if err != nil {
		fmt.Printf("DEBUG: Failed to convert output path to absolute: %v\n", err)
		return nil, fmt.Errorf("failed to get absolute output path: %v", err)
	}

	// Verify absolute paths are accessible
	fmt.Printf("DEBUG: Absolute audio path: %s\n", absAudioPath)
	fmt.Printf("DEBUG: Absolute output path: %s\n", absOutputFile)

	// Check that the absolute audio path is still accessible
	if _, err := os.Stat(absAudioPath); err != nil {
		fmt.Printf("DEBUG: Absolute audio path not accessible: %v\n", err)
		return nil, fmt.Errorf("absolute audio path not accessible: %s - %v", absAudioPath, err)
	}

	// Build command to run the transcription script with Canary model
	// Since cmd.Dir is set to nvidia directory, we use "." as project path
	args := []string{
		"run", "--native-tls", "--project", ".", "python", "transcribe.py",
		absAudioPath,
		"--model", "canary", // Specify to use Canary model
		"--timestamps", // Always include timestamps for consistency
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

	// Note: Diarization will be handled separately after transcription

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

	// Create speaker mappings if diarization was performed
	if canaryResult.Diarized {
		speakerLabels := make(map[string]bool)
		// Collect unique speaker labels from segments
		for _, segment := range canaryResult.SegmentTimestamps {
			if segment.Speaker != "" {
				speakerLabels[segment.Speaker] = true
			}
		}

		// Create speaker mapping records
		for speaker := range speakerLabels {
			speakerMapping := &models.SpeakerMapping{
				TranscriptionJobID: jobID,
				OriginalSpeaker:    speaker,
				CustomName:         speaker, // Default to original label
			}
			if err := database.DB.Create(speakerMapping).Error; err != nil {
				return fmt.Errorf("failed to create speaker mapping: %v", err)
			}
		}
		fmt.Printf("DEBUG: Created speaker mappings for %d speakers\n", len(speakerLabels))
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
				Start:   seg.Start,
				End:     seg.End,
				Text:    strings.TrimSpace(seg.Segment),
				Speaker: stringPtr(seg.Speaker), // Speaker information from diarization
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
			Start:   word.Start,
			End:     word.End,
			Word:    word.Word,
			Score:   1.0,                     // Canary doesn't provide confidence scores
			Speaker: stringPtr(word.Speaker), // Speaker information from diarization
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

// runDiarization performs speaker diarization on the audio file and merges with transcript
func (cs *CanaryService) runDiarization(audioPath string, transcriptPath string, params models.WhisperXParams) error {
	// Create temporary RTTM file path
	rttmPath := transcriptPath + ".rttm"

	// Build diarization command based on selected model
	nvidiaPath := filepath.Join("whisperx-env", "parakeet")
	absAudioPath, err := filepath.Abs(audioPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute audio path: %v", err)
	}

	// Convert RTTM path to absolute path since we're running from parakeet directory
	absRttmPath, err := filepath.Abs(rttmPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute RTTM path: %v", err)
	}

	var args []string
	var scriptName string

	// Choose the appropriate diarization script based on the model
	if params.DiarizeModel == "nvidia_sortformer" {
		fmt.Printf("DEBUG: Using NVIDIA Sortformer diarization for Canary job\n")
		scriptName = "nemo_diarize.py"
		args = []string{
			"run", "--native-tls", "--project", ".", "python", scriptName,
		}
		
		// Add speaker constraints for NVIDIA Sortformer (no HF token needed)
		if params.MinSpeakers != nil {
			args = append(args, "--min-speakers", fmt.Sprintf("%d", *params.MinSpeakers))
		}
		if params.MaxSpeakers != nil {
			args = append(args, "--max-speakers", fmt.Sprintf("%d", *params.MaxSpeakers))
		}
		
		// Add positional arguments: audio_file and output_file
		args = append(args, absAudioPath, absRttmPath)
	} else {
		// Default to Pyannote diarization
		fmt.Printf("DEBUG: Using Pyannote diarization for Canary job\n")
		scriptName = "diarize.py"
		args = []string{
			"run", "--native-tls", "--project", ".", "python", scriptName,
			absAudioPath,
			"--output", absRttmPath,
		}

		// Pyannote requires HuggingFace token
		if params.HfToken != nil && *params.HfToken != "" {
			args = append(args, "--hf-token", *params.HfToken)
		} else {
			return fmt.Errorf("HuggingFace token is required for Pyannote diarization")
		}

		if params.MinSpeakers != nil {
			args = append(args, "--min-speakers", fmt.Sprintf("%d", *params.MinSpeakers))
		}
		if params.MaxSpeakers != nil {
			args = append(args, "--max-speakers", fmt.Sprintf("%d", *params.MaxSpeakers))
		}
	}

	fmt.Printf("DEBUG: Running diarization command with %s: uv %v\n", scriptName, args)

	// Execute diarization
	cmd := exec.Command("uv", args...)
	cmd.Dir = nvidiaPath
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("DEBUG: Diarization failed: %v\nOutput: %s\n", err, string(output))
		return fmt.Errorf("diarization failed: %v", err)
	}

	fmt.Printf("DEBUG: Diarization completed successfully\n")

	// Now merge the diarization results with the transcript
	return cs.mergeDiarizationWithTranscript(transcriptPath, rttmPath)
}

// mergeDiarizationWithTranscript reads the transcript and RTTM files, merges them, and updates the transcript
func (cs *CanaryService) mergeDiarizationWithTranscript(transcriptPath, rttmPath string) error {
	// Read the original transcript
	transcriptData, err := os.ReadFile(transcriptPath)
	if err != nil {
		return fmt.Errorf("failed to read transcript file: %v", err)
	}

	var result CanaryResult
	if err := json.Unmarshal(transcriptData, &result); err != nil {
		return fmt.Errorf("failed to unmarshal transcript: %v", err)
	}

	// Parse RTTM file
	speakers, err := cs.parseRTTMFile(rttmPath)
	if err != nil {
		return fmt.Errorf("failed to parse RTTM file: %v", err)
	}

	// Assign speakers to segments and words
	cs.assignSpeakersToSegments(&result.SegmentTimestamps, speakers)
	cs.assignSpeakersToWords(&result.WordTimestamps, speakers)

	// Mark as diarized
	result.Diarized = true

	// Save the updated transcript
	updatedData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal updated transcript: %v", err)
	}

	if err := os.WriteFile(transcriptPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to save updated transcript: %v", err)
	}

	// Clean up RTTM file
	os.Remove(rttmPath)

	return nil
}

// parseRTTMFile parses the RTTM file and returns speaker segments (shared with Parakeet)
func (cs *CanaryService) parseRTTMFile(rttmPath string) ([]CanarySpeakerSegment, error) {
	data, err := os.ReadFile(rttmPath)
	if err != nil {
		return nil, err
	}

	var speakers []CanarySpeakerSegment
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "SPEAKER") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}

		start, err1 := strconv.ParseFloat(parts[3], 64)
		duration, err2 := strconv.ParseFloat(parts[4], 64)
		speaker := parts[7]

		if err1 != nil || err2 != nil {
			continue
		}

		speakers = append(speakers, CanarySpeakerSegment{
			Start:   start,
			End:     start + duration,
			Speaker: speaker,
		})
	}

	return speakers, nil
}

// CanarySpeakerSegment represents a speaker segment from RTTM for Canary
type CanarySpeakerSegment struct {
	Start   float64
	End     float64
	Speaker string
}

// assignSpeakersToSegments assigns speaker labels to transcript segments
func (cs *CanaryService) assignSpeakersToSegments(segments *[]CanarySegment, speakers []CanarySpeakerSegment) {
	for i := range *segments {
		segment := &(*segments)[i]
		bestSpeaker := cs.findBestSpeaker(segment.Start, segment.End, speakers)
		segment.Speaker = bestSpeaker
	}
}

// assignSpeakersToWords assigns speaker labels to words
func (cs *CanaryService) assignSpeakersToWords(words *[]CanaryWord, speakers []CanarySpeakerSegment) {
	for i := range *words {
		word := &(*words)[i]
		bestSpeaker := cs.findBestSpeaker(word.Start, word.End, speakers)
		word.Speaker = bestSpeaker
	}
}

// findBestSpeaker finds the speaker with the maximum overlap for the given time range
func (cs *CanaryService) findBestSpeaker(start, end float64, speakers []CanarySpeakerSegment) string {
	maxOverlap := 0.0
	bestSpeaker := "SPEAKER_00" // Default speaker

	for _, speaker := range speakers {
		overlapStart := math.Max(start, speaker.Start)
		overlapEnd := math.Min(end, speaker.End)

		if overlapStart < overlapEnd {
			overlap := overlapEnd - overlapStart
			if overlap > maxOverlap {
				maxOverlap = overlap
				bestSpeaker = speaker.Speaker
			}
		}
	}

	return bestSpeaker
}

// preprocessAudioForCanary converts audio to Canary-compatible format using ffmpeg
func (cs *CanaryService) preprocessAudioForCanary(inputPath, outputDir string) (string, error) {
	// Enhanced debugging for Docker preprocessing
	fmt.Printf("DEBUG: Starting Canary audio preprocessing\n")
	workingDir, _ := os.Getwd()
	fmt.Printf("DEBUG: Current working directory: %s\n", workingDir)
	fmt.Printf("DEBUG: Input path: %s\n", inputPath)
	fmt.Printf("DEBUG: Output directory: %s\n", outputDir)

	// Verify input file exists and is accessible
	if stat, err := os.Stat(inputPath); err != nil {
		fmt.Printf("DEBUG: Input file not accessible: %v\n", err)
		return "", fmt.Errorf("input audio file not accessible: %s - %v", inputPath, err)
	} else {
		fmt.Printf("DEBUG: Input file exists, size: %d bytes, mode: %s\n", stat.Size(), stat.Mode())
	}

	// Verify output directory exists and is writable
	if stat, err := os.Stat(outputDir); err != nil {
		fmt.Printf("DEBUG: Output directory not accessible: %v\n", err)
		return "", fmt.Errorf("output directory not accessible: %s - %v", outputDir, err)
	} else {
		fmt.Printf("DEBUG: Output directory exists, mode: %s\n", stat.Mode())
	}

	// Generate a unique temporary filename in the output directory
	baseName := filepath.Base(inputPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	tempFileName := fmt.Sprintf("%s_canary_temp.wav", nameWithoutExt)
	tempPath := filepath.Join(outputDir, tempFileName)

	fmt.Printf("DEBUG: Temp file path: %s\n", tempPath)

	// Test if we can create a file in the output directory
	testFile := filepath.Join(outputDir, "test_write.tmp")
	if file, err := os.Create(testFile); err != nil {
		fmt.Printf("DEBUG: Cannot create test file in output directory: %v\n", err)
		return "", fmt.Errorf("output directory not writable: %s - %v", outputDir, err)
	} else {
		file.Close()
		os.Remove(testFile)
		fmt.Printf("DEBUG: Output directory is writable\n")
	}

	// Check ffmpeg availability
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Printf("DEBUG: ffmpeg not found in PATH: %v\n", err)
		return "", fmt.Errorf("ffmpeg not available: %v", err)
	} else {
		fmt.Printf("DEBUG: ffmpeg found in PATH\n")
	}

	// Build ffmpeg command: ffmpeg -i input.mp3 -ar 16000 -ac 1 -c:a pcm_s16le output.wav
	cmd := exec.Command("ffmpeg",
		"-i", inputPath, // Input file
		"-ar", "16000", // Sample rate 16kHz
		"-ac", "1", // Mono (1 channel)
		"-c:a", "pcm_s16le", // PCM 16-bit little-endian codec
		"-y",     // Overwrite output file if it exists
		tempPath, // Output file
	)

	fmt.Printf("DEBUG: ffmpeg command: %v\n", cmd.Args)

	// Capture ffmpeg output for debugging
	output, err := cmd.CombinedOutput()
	fmt.Printf("DEBUG: ffmpeg exit code: %v\n", err)
	fmt.Printf("DEBUG: ffmpeg output:\n%s\n", string(output))

	if err != nil {
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

// checkMemoryAvailability checks if sufficient memory is available for Canary model
func (cs *CanaryService) checkMemoryAvailability() error {
	// Try to read memory info from /proc/meminfo (Linux containers)
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		// Not on Linux or can't read meminfo, skip check
		fmt.Printf("DEBUG: Cannot read memory info: %v\n", err)
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var memTotal, memAvailable int64

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				memTotal = val * 1024 // Convert from KB to bytes
			}
		case "MemAvailable:":
			if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				memAvailable = val * 1024 // Convert from KB to bytes
			}
		}
	}

	// Convert to GB for logging
	totalGB := float64(memTotal) / (1024 * 1024 * 1024)
	availableGB := float64(memAvailable) / (1024 * 1024 * 1024)

	fmt.Printf("DEBUG: System memory - Total: %.1fGB, Available: %.1fGB\n", totalGB, availableGB)

	// Canary model requires approximately 8-12GB of RAM
	const requiredGB = 8.0

	if availableGB < requiredGB {
		return fmt.Errorf("insufficient memory for Canary model: %.1fGB available, %vGB recommended (model size: 6.3GB + overhead)", availableGB, requiredGB)
	}

	if availableGB < requiredGB*1.5 {
		fmt.Printf("WARNING: Low available memory (%.1fGB) for Canary model. Consider increasing Docker memory allocation.\n", availableGB)
	}

	return nil
}
