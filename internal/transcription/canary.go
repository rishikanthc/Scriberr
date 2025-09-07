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
	Transcription      string             `json:"transcription"`
	WordTimestamps     []CanaryWord       `json:"word_timestamps"`
	SegmentTimestamps  []CanarySegment    `json:"segment_timestamps"`
	AudioFile          string             `json:"audio_file"`
	SourceLanguage     string             `json:"source_language"`
	TargetLanguage     string             `json:"target_language"`
	Diarized          bool               `json:"diarized"`
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

	// Run diarization if enabled
	fmt.Printf("DEBUG: Checking diarization conditions - Diarize: %v, HfToken != nil: %v\n", job.Parameters.Diarize, job.Parameters.HfToken != nil)
	if job.Parameters.HfToken != nil {
		fmt.Printf("DEBUG: HfToken value: %s\n", *job.Parameters.HfToken)
	}
	
	if job.Parameters.Diarize && job.Parameters.HfToken != nil && *job.Parameters.HfToken != "" {
		fmt.Printf("DEBUG: Running diarization for Canary job %s\n", jobID)
		if err := cs.runDiarization(job.AudioPath, resultPath, job.Parameters); err != nil {
			fmt.Printf("WARNING: Diarization failed: %v. Continuing with transcript without speaker information.\n", err)
			// Don't fail the job, just continue without diarization
		}
	} else {
		fmt.Printf("DEBUG: Diarization conditions not met - skipping diarization\n")
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
				Start: seg.Start,
				End:   seg.End,
				Text:  strings.TrimSpace(seg.Segment),
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
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
			Score: 1.0, // Canary doesn't provide confidence scores
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
	
	// Build diarization command
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
	
	args := []string{
		"run", "--native-tls", "--project", ".", "python", "diarize.py",
		absAudioPath,
		"--output", absRttmPath,
	}
	
	if params.HfToken != nil && *params.HfToken != "" {
		args = append(args, "--hf-token", *params.HfToken)
	} else {
		return fmt.Errorf("HuggingFace token is required for diarization")
	}
	
	if params.MinSpeakers != nil {
		args = append(args, "--min-speakers", fmt.Sprintf("%d", *params.MinSpeakers))
	}
	if params.MaxSpeakers != nil {
		args = append(args, "--max-speakers", fmt.Sprintf("%d", *params.MaxSpeakers))
	}
	
	fmt.Printf("DEBUG: Running diarization command: uv %v\n", args)
	
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