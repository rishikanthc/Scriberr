package transcription

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/google/uuid"
)

// QuickTranscriptionJob represents a temporary transcription job
type QuickTranscriptionJob struct {
	ID           string                `json:"id"`
	Status       models.JobStatus      `json:"status"`
	AudioPath    string                `json:"audio_path"`
	Transcript   *string               `json:"transcript,omitempty"`
	Parameters   models.WhisperXParams `json:"parameters"`
	CreatedAt    time.Time             `json:"created_at"`
	ExpiresAt    time.Time             `json:"expires_at"`
	ErrorMessage *string               `json:"error_message,omitempty"`
}

// QuickTranscriptionService handles temporary transcriptions without database persistence
type QuickTranscriptionService struct {
	config           *config.Config
	unifiedProcessor *UnifiedJobProcessor
	jobRepo          repository.JobRepository
	jobs             map[string]*QuickTranscriptionJob
	jobsMutex        sync.RWMutex
	tempDir          string
	cleanupTicker    *time.Ticker
	stopCleanup      chan bool
}

// NewQuickTranscriptionService creates a new quick transcription service
func NewQuickTranscriptionService(cfg *config.Config, unifiedProcessor *UnifiedJobProcessor, jobRepo repository.JobRepository) (*QuickTranscriptionService, error) {
	// Create temporary directory for quick transcriptions
	tempDir := filepath.Join(cfg.UploadDir, "quick_transcriptions")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	service := &QuickTranscriptionService{
		config:           cfg,
		unifiedProcessor: unifiedProcessor,
		jobRepo:          jobRepo,
		jobs:             make(map[string]*QuickTranscriptionJob),
		tempDir:          tempDir,
		stopCleanup:      make(chan bool),
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

	// Create temporary transcription job for WhisperX processing
	tempJob := models.TranscriptionJob{
		ID:         jobID,
		AudioPath:  job.AudioPath,
		Parameters: job.Parameters,
		Status:     models.StatusProcessing,
	}

	// Create a temporary database entry for unified processing
	ctx := context.Background()

	// Save temporary job to database for processing
	if err := qs.jobRepo.Create(ctx, &tempJob); err != nil {
		qs.jobsMutex.Lock()
		if job, exists := qs.jobs[jobID]; exists {
			job.Status = models.StatusFailed
			errMsg := fmt.Sprintf("failed to create temp database entry: %v", err)
			job.ErrorMessage = &errMsg
		}
		qs.jobsMutex.Unlock()
		return
	}

	// Process with unified service
	err := qs.unifiedProcessor.ProcessJob(ctx, jobID)

	// Load the processed result back
	if processedJob, loadErr := qs.jobRepo.FindByID(ctx, jobID); loadErr == nil {
		// Copy result back to quick job if successful
		if err == nil {
			if processedJob.Transcript != nil {
				// Save transcript to temp file for loadTranscriptFromTemp
				transcriptPath := filepath.Join(qs.tempDir, jobID+"_transcript.json")
				_ = os.WriteFile(transcriptPath, []byte(*processedJob.Transcript), 0644)
			}
		}
	}

	// Clean up temporary database entry
	_ = qs.jobRepo.Delete(ctx, jobID)

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

// Close stops the cleanup routine
func (qs *QuickTranscriptionService) Close() {
	if qs.cleanupTicker != nil {
		close(qs.stopCleanup)
	}
}
