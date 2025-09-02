package transcription

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/models"

	"github.com/google/uuid"
)

// QuickTranscriptionJob represents a temporary transcription job
type QuickTranscriptionJob struct {
	ID          string                  `json:"id"`
	Status      models.JobStatus        `json:"status"`
	AudioPath   string                  `json:"audio_path"`
	Transcript  *string                 `json:"transcript,omitempty"`
	Parameters  models.WhisperXParams   `json:"parameters"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	ErrorMessage *string               `json:"error_message,omitempty"`
}

// QuickTranscriptionService handles temporary transcriptions without database persistence
type QuickTranscriptionService struct {
	config       *config.Config
	whisperX     *WhisperXService
	jobs         map[string]*QuickTranscriptionJob
	jobsMutex    sync.RWMutex
	tempDir      string
	cleanupTicker *time.Ticker
	stopCleanup  chan bool
}

// NewQuickTranscriptionService creates a new quick transcription service
func NewQuickTranscriptionService(cfg *config.Config, whisperX *WhisperXService) (*QuickTranscriptionService, error) {
	// Create temporary directory for quick transcriptions
	tempDir := filepath.Join(cfg.UploadDir, "quick_transcriptions")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	service := &QuickTranscriptionService{
		config:       cfg,
		whisperX:     whisperX,
		jobs:         make(map[string]*QuickTranscriptionJob),
		tempDir:      tempDir,
		stopCleanup:  make(chan bool),
	}

	// Start cleanup routine (run every hour)
	service.startCleanupRoutine()

	return service, nil
}

// SubmitQuickJob creates and processes a temporary transcription job
func (qs *QuickTranscriptionService) SubmitQuickJob(audioData io.Reader, filename string, params models.WhisperXParams) (*QuickTranscriptionJob, error) {
	// Generate unique job ID
	jobID := uuid.New().String()
	
	// Create temporary file for audio
	ext := filepath.Ext(filename)
	audioFilename := fmt.Sprintf("%s%s", jobID, ext)
	audioPath := filepath.Join(qs.tempDir, audioFilename)
	
	// Save audio file
	audioFile, err := os.Create(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio file: %v", err)
	}
	defer audioFile.Close()
	
	if _, err := io.Copy(audioFile, audioData); err != nil {
		os.Remove(audioPath)
		return nil, fmt.Errorf("failed to save audio file: %v", err)
	}

	// Create quick transcription job
	now := time.Now()
	job := &QuickTranscriptionJob{
		ID:         jobID,
		Status:     models.StatusPending,
		AudioPath:  audioPath,
		Parameters: params,
		CreatedAt:  now,
		ExpiresAt:  now.Add(6 * time.Hour),
	}

	// Store in memory
	qs.jobsMutex.Lock()
	qs.jobs[jobID] = job
	qs.jobsMutex.Unlock()

	// Start processing in background
	go qs.processQuickJob(jobID)

	return job, nil
}

// GetQuickJob retrieves a quick transcription job by ID
func (qs *QuickTranscriptionService) GetQuickJob(jobID string) (*QuickTranscriptionJob, error) {
	qs.jobsMutex.RLock()
	defer qs.jobsMutex.RUnlock()
	
	job, exists := qs.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}
	
	// Check if expired
	if time.Now().After(job.ExpiresAt) {
		return nil, fmt.Errorf("job expired")
	}
	
	return job, nil
}

// processQuickJob processes a quick transcription job
func (qs *QuickTranscriptionService) processQuickJob(jobID string) {
	// Update job status to processing
	qs.jobsMutex.Lock()
	job, exists := qs.jobs[jobID]
	if !exists {
		qs.jobsMutex.Unlock()
		return
	}
	job.Status = models.StatusProcessing
	qs.jobsMutex.Unlock()

    // Ensure Python environment and embedded assets are ready
    if err := qs.whisperX.ensurePythonEnv(); err != nil {
        qs.jobsMutex.Lock()
        if job, exists := qs.jobs[jobID]; exists {
            job.Status = models.StatusFailed
            msg := fmt.Sprintf("env setup failed: %v", err)
            job.ErrorMessage = &msg
        }
        qs.jobsMutex.Unlock()
        return
    }

    // Create temporary transcription job for WhisperX processing
    tempJob := models.TranscriptionJob{
        ID:         jobID,
        AudioPath:  job.AudioPath,
        Parameters: job.Parameters,
        Status:     models.StatusProcessing,
	}

	// Process with WhisperX
	ctx := context.Background()
	err := qs.processWithWhisperX(ctx, &tempJob)
	
	// Update job with results
	qs.jobsMutex.Lock()
	defer qs.jobsMutex.Unlock()
	
	if job, exists := qs.jobs[jobID]; exists {
		if err != nil {
			job.Status = models.StatusFailed
			errMsg := err.Error()
			job.ErrorMessage = &errMsg
		} else {
			job.Status = models.StatusCompleted
			// Load transcript from temporary file
			if transcript, loadErr := qs.loadTranscriptFromTemp(jobID); loadErr == nil {
				job.Transcript = &transcript
			}
		}
	}
}

// processWithWhisperX processes the job using WhisperX service
func (qs *QuickTranscriptionService) processWithWhisperX(ctx context.Context, job *models.TranscriptionJob) error {
	// Check if audio file exists
	if _, err := os.Stat(job.AudioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", job.AudioPath)
	}

	// Prepare output directory in temp location
	outputDir := filepath.Join(qs.tempDir, job.ID+"_output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Build WhisperX command using the existing service
	args, err := qs.buildWhisperXArgs(job, outputDir)
	if err != nil {
		return fmt.Errorf("failed to build command: %v", err)
	}

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	
	// Set process group ID so we can kill the entire process tree if needed
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Execute WhisperX
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return fmt.Errorf("job was cancelled")
	}
	if err != nil {
		fmt.Printf("DEBUG: WhisperX stderr/stdout: %s\n", string(output))
		return fmt.Errorf("WhisperX execution failed: %v", err)
	}

	// Load and save the result to temporary location
	resultPath := filepath.Join(outputDir, "result.json")
	return qs.saveTranscriptToTemp(job.ID, resultPath, outputDir)
}

// saveTranscriptToTemp saves transcript to temporary file
func (qs *QuickTranscriptionService) saveTranscriptToTemp(jobID, resultPath, outputDir string) error {
	var resultFile string
	
	// Check if result.json exists (from diarization script)
	if _, err := os.Stat(resultPath); err == nil {
		resultFile = resultPath
	} else {
		// Find the actual result file
		files, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
		if err != nil {
			return fmt.Errorf("failed to find result files: %v", err)
		}
		
		if len(files) == 0 {
			return fmt.Errorf("no result files found")
		}
		
		resultFile = files[0]
	}
	
	// Read the result file
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return fmt.Errorf("failed to read result file: %v", err)
	}

	// Save to a well-known location for this job
	transcriptPath := filepath.Join(qs.tempDir, jobID+"_transcript.json")
	return os.WriteFile(transcriptPath, data, 0644)
}

// loadTranscriptFromTemp loads transcript from temporary file
func (qs *QuickTranscriptionService) loadTranscriptFromTemp(jobID string) (string, error) {
	transcriptPath := filepath.Join(qs.tempDir, jobID+"_transcript.json")
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// startCleanupRoutine starts the background cleanup routine
func (qs *QuickTranscriptionService) startCleanupRoutine() {
	qs.cleanupTicker = time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-qs.cleanupTicker.C:
				qs.cleanupExpiredJobs()
			case <-qs.stopCleanup:
				qs.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupExpiredJobs removes expired jobs and their files
func (qs *QuickTranscriptionService) cleanupExpiredJobs() {
	qs.jobsMutex.Lock()
	defer qs.jobsMutex.Unlock()
	
	now := time.Now()
	for jobID, job := range qs.jobs {
		if now.After(job.ExpiresAt) {
			// Remove files
			os.Remove(job.AudioPath)
			os.Remove(filepath.Join(qs.tempDir, jobID+"_transcript.json"))
			os.RemoveAll(filepath.Join(qs.tempDir, jobID+"_output"))
			
			// Remove from memory
			delete(qs.jobs, jobID)
			
			fmt.Printf("DEBUG: Cleaned up expired quick transcription job: %s\n", jobID)
		}
	}
}

// buildWhisperXArgs builds the WhisperX command arguments for quick transcription
func (qs *QuickTranscriptionService) buildWhisperXArgs(job *models.TranscriptionJob, outputDir string) ([]string, error) {
	p := job.Parameters
	
	// Debug: log diarization status
	fmt.Printf("DEBUG: Quick Job ID %s, Diarize parameter: %v\n", job.ID, p.Diarize)
	
	// Use standard WhisperX command for quick transcriptions
    args := []string{
        "run", "--native-tls", "--project", "whisperx-env", "python", "-m", "whisperx",
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

	// Output settings
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

	// Output formatting
	args = append(args, "--highlight_words", "False")
	args = append(args, "--segment_resolution", "sentence")

	// Token and progress
	if p.HfToken != nil {
		args = append(args, "--hf_token", *p.HfToken)
	}
	args = append(args, "--print_progress", "False")

	// Debug: log the command being executed
	fmt.Printf("DEBUG: Quick WhisperX command: uv %v\n", args)
	
	return args, nil
}

// Close stops the cleanup routine
func (qs *QuickTranscriptionService) Close() {
	if qs.cleanupTicker != nil {
		close(qs.stopCleanup)
	}
}
