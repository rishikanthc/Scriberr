package tasks

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"scriberr-backend/internal/database"
	"scriberr-backend/internal/models"

	"github.com/google/uuid"
)

// JobStatus represents the status of a transcription job.
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusTerminated JobStatus = "terminated"
)

// Job represents a transcription job.
type Job struct {
	ID          string    `json:"id"`
	AudioID     string    `json:"audio_id"`
	Model       string    `json:"model,omitempty"`
	Diarize     bool      `json:"diarize,omitempty"`
	MinSpeakers int       `json:"min_speakers,omitempty"`
	MaxSpeakers int       `json:"max_speakers,omitempty"`
	Status      JobStatus `json:"status"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Cmd         *exec.Cmd `json:"-"` // The running command, not serialized
	
	// Additional transcription parameters
	BatchSize                    int     `json:"batch_size,omitempty"`
	ComputeType                  string  `json:"compute_type,omitempty"`
	VadOnset                     float64 `json:"vad_onset,omitempty"`
	VadOffset                    float64 `json:"vad_offset,omitempty"`
	ConditionOnPreviousText      bool    `json:"condition_on_previous_text,omitempty"`
	CompressionRatioThreshold    float64 `json:"compression_ratio_threshold,omitempty"`
	LogprobThreshold             float64 `json:"logprob_threshold,omitempty"`
	NoSpeechThreshold            float64 `json:"no_speech_threshold,omitempty"`
	Temperature                  float64 `json:"temperature,omitempty"`
	BestOf                       int     `json:"best_of,omitempty"`
	BeamSize                     int     `json:"beam_size,omitempty"`
	Patience                     float64 `json:"patience,omitempty"`
	LengthPenalty                float64 `json:"length_penalty,omitempty"`
	SuppressNumerals             bool    `json:"suppress_numerals,omitempty"`
	InitialPrompt                string  `json:"initial_prompt,omitempty"`
	TemperatureIncrementOnFallback float64 `json:"temperature_increment_on_fallback,omitempty"`
}

// TranscriptSegment represents one entry in a parsed transcript file.
type TranscriptSegment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

var (
	jobQueue   = make(chan Job, 100) // Buffered channel for jobs
	jobStore   = make(map[string]*Job)
	storeMutex = &sync.RWMutex{}
	once       sync.Once
)

const (
	convertedDir = "./storage/converted"
	outputDir    = "./storage/transcripts"
)

// Init starts the job queue worker.
// This should be called once when the application starts.
func Init() {
	once.Do(func() {
		log.Println("Initializing job queue worker...")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("Error creating transcripts directory: %v", err)
		}
		go worker()
	})
}

// NewJob creates a new transcription job and adds it to the queue.
func NewJob(audioID, model string, diarize bool, minSpeakers, maxSpeakers int, params ...map[string]interface{}) (*Job, error) {
	// Clear old transcript data when starting a new transcription
	if err := clearOldTranscriptData(audioID); err != nil {
		log.Printf("Warning: Failed to clear old transcript data for audio %s: %v", audioID, err)
	}

	job := &Job{
		ID:          uuid.New().String(),
		AudioID:     audioID,
		Model:       model,
		Diarize:     diarize,
		MinSpeakers: minSpeakers,
		MaxSpeakers: maxSpeakers,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		
		// Set default values for additional parameters
		BatchSize:                    16,
		ComputeType:                  "int8",
		VadOnset:                     0.5,
		VadOffset:                    0.5,
		ConditionOnPreviousText:      true,
		CompressionRatioThreshold:    2.4,
		LogprobThreshold:             -1.0,
		NoSpeechThreshold:            0.6,
		Temperature:                  0.0,
		BestOf:                       5,
		BeamSize:                     5,
		Patience:                     1.0,
		LengthPenalty:                1.0,
		SuppressNumerals:             false,
		InitialPrompt:                "",
		TemperatureIncrementOnFallback: 0.2,
	}

	// Apply custom parameters if provided
	if len(params) > 0 {
		paramMap := params[0]
		if v, ok := paramMap["batch_size"].(int); ok && v > 0 {
			job.BatchSize = v
		}
		if v, ok := paramMap["compute_type"].(string); ok && v != "" {
			job.ComputeType = v
		}
		if v, ok := paramMap["vad_onset"].(float64); ok && v > 0 {
			job.VadOnset = v
		}
		if v, ok := paramMap["vad_offset"].(float64); ok && v > 0 {
			job.VadOffset = v
		}
		if v, ok := paramMap["condition_on_previous_text"].(bool); ok {
			job.ConditionOnPreviousText = v
		}
		if v, ok := paramMap["compression_ratio_threshold"].(float64); ok && v > 0 {
			job.CompressionRatioThreshold = v
		}
		if v, ok := paramMap["logprob_threshold"].(float64); ok {
			job.LogprobThreshold = v
		}
		if v, ok := paramMap["no_speech_threshold"].(float64); ok && v > 0 {
			job.NoSpeechThreshold = v
		}
		if v, ok := paramMap["temperature"].(float64); ok {
			job.Temperature = v
		}
		if v, ok := paramMap["best_of"].(int); ok && v > 0 {
			job.BestOf = v
		}
		if v, ok := paramMap["beam_size"].(int); ok && v > 0 {
			job.BeamSize = v
		}
		if v, ok := paramMap["patience"].(float64); ok && v > 0 {
			job.Patience = v
		}
		if v, ok := paramMap["length_penalty"].(float64); ok && v > 0 {
			job.LengthPenalty = v
		}
		if v, ok := paramMap["suppress_numerals"].(bool); ok {
			job.SuppressNumerals = v
		}
		if v, ok := paramMap["initial_prompt"].(string); ok {
			job.InitialPrompt = v
		}
		if v, ok := paramMap["temperature_increment_on_fallback"].(float64); ok && v > 0 {
			job.TemperatureIncrementOnFallback = v
		}
	}

	storeMutex.Lock()
	jobStore[job.ID] = job
	storeMutex.Unlock()

	select {
	case jobQueue <- *job:
		log.Printf("New job created and queued. JobID: %s, AudioID: %s, Model: %s, Diarize: %t, MinSpeakers: %d, MaxSpeakers: %d", job.ID, job.AudioID, job.Model, job.Diarize, job.MinSpeakers, job.MaxSpeakers)
		return job, nil
	default:
		// Queue is full, remove the job from store
		storeMutex.Lock()
		delete(jobStore, job.ID)
		storeMutex.Unlock()
		return nil, fmt.Errorf("job queue is full")
	}
}

// clearOldTranscriptData removes old transcript, speaker map, and summary data when retranscribing
func clearOldTranscriptData(audioID string) error {
	db := database.GetDB()
	query := `UPDATE audio_records SET transcript = NULL, speaker_map = NULL, summary = NULL WHERE id = ?`
	
	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare database cleanup statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(audioID)
	if err != nil {
		return fmt.Errorf("failed to clear old transcript data: %w", err)
	}

	log.Printf("Cleared old transcript data for audio record %s", audioID)
	return nil
}

// GetJobStatus retrieves the status of a specific job.
func GetJobStatus(jobID string) (*Job, error) {
	storeMutex.RLock()
	defer storeMutex.RUnlock()

	job, exists := jobStore[jobID]
	if !exists {
		return nil, fmt.Errorf("job with ID %s not found", jobID)
	}

	return job, nil
}

// GetActiveJobs retrieves all jobs that are currently pending or processing.
func GetActiveJobs() ([]models.ActiveJob, error) {
	storeMutex.RLock()
	defer storeMutex.RUnlock()

	var activeJobs []models.ActiveJob
	db := database.GetDB()

	for _, job := range jobStore {
		if job.Status == StatusPending || job.Status == StatusProcessing {
			var title string
			// Query the database for the title of the audio file.
			err := db.QueryRow("SELECT title FROM audio_records WHERE id = ?", job.AudioID).Scan(&title)
			if err != nil {
				// Log the error but continue; maybe the record was deleted but job not cleaned up
				log.Printf("Could not retrieve title for audio_id %s for active job %s: %v", job.AudioID, job.ID, err)
				title = "Unknown Title" // Provide a default title
			}

			activeJobs = append(activeJobs, models.ActiveJob{
				ID:         job.ID,
				AudioID:    job.AudioID,
				AudioTitle: title,
				Status:     string(job.Status),
				Type:       "transcription",
				CreatedAt:  job.CreatedAt,
			})
		}
	}

	return activeJobs, nil
}

// TerminateJob stops a pending or running job.
func TerminateJob(jobID string) error {
	storeMutex.Lock()
	defer storeMutex.Unlock()

	job, exists := jobStore[jobID]
	if !exists {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	switch job.Status {
	case StatusPending:
		job.Status = StatusTerminated
		log.Printf("Job %s was pending and has been terminated before processing.", jobID)
	case StatusProcessing:
		if job.Cmd != nil && job.Cmd.Process != nil {
			log.Printf("Attempting to terminate running process for job %s (PID: %d)", jobID, job.Cmd.Process.Pid)
			if err := job.Cmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill process for job %s: %v", jobID, err)
				// Even if killing fails, we mark the job as terminated to prevent further processing.
			} else {
				log.Printf("Successfully killed process for job %s", jobID)
			}
		}
		job.Status = StatusTerminated
		job.Error = "Job was terminated by user."
	case StatusCompleted, StatusFailed, StatusTerminated:
		log.Printf("Job %s is already in a final state (%s) and cannot be terminated.", jobID, job.Status)
		return fmt.Errorf("job %s is already finished", jobID)
	default:
		return fmt.Errorf("job %s is in an unknown state and cannot be terminated", jobID)
	}

	return nil
}

// worker is a long-running goroutine that processes jobs from the queue.
func worker() {
	for job := range jobQueue {
		// It's possible the job was terminated while in the queue.
		// Let's get the latest status from the central store.
		currentJobState, err := GetJobStatus(job.ID)
		if err != nil {
			log.Printf("Job %s from queue not found in store, skipping.", job.ID)
			continue
		}
		if currentJobState.Status == StatusTerminated {
			log.Printf("Job %s was terminated while pending, skipping processing.", job.ID)
			continue
		}

		log.Printf("Processing job %s for audio %s", job.ID, job.AudioID)
		updateJobStatus(job.ID, StatusProcessing, "")

		audioPath := filepath.Join(convertedDir, job.AudioID+".wav")
		transcriptOutputDir := filepath.Join(outputDir, job.ID)

		if err := os.MkdirAll(transcriptOutputDir, 0755); err != nil {
			log.Printf("Error creating transcript output directory for job %s: %v", job.ID, err)
			updateJobStatus(job.ID, StatusFailed, "Failed to create output directory")
			continue
		}

		model := job.Model
		if model == "" {
			model = "small" // Fallback default
		}

		// Build command arguments
		var args []string
		
		// Use dedicated diarization script for better robustness
		if job.Diarize {
			// Use the dedicated diarization script
			outputFile := filepath.Join(transcriptOutputDir, job.AudioID+".json")
			args = []string{"run", "diarize.py", audioPath, "--model", model, "--min-speakers", fmt.Sprintf("%d", job.MinSpeakers), "--max-speakers", fmt.Sprintf("%d", job.MaxSpeakers), "--output", outputFile}
			
			// Add additional parameters to diarization script
			args = append(args, "--batch-size", fmt.Sprintf("%d", job.BatchSize))
			args = append(args, "--compute-type", job.ComputeType)
			
			// Add VAD parameters
			args = append(args, "--vad-onset", fmt.Sprintf("%.1f", job.VadOnset))
			args = append(args, "--vad-offset", fmt.Sprintf("%.1f", job.VadOffset))
			
			// Add transcription quality parameters
			conditionText := "False"
			if job.ConditionOnPreviousText {
				conditionText = "True"
			}
			args = append(args, "--condition-on-previous-text", conditionText)
			args = append(args, "--compression-ratio-threshold", fmt.Sprintf("%.1f", job.CompressionRatioThreshold))
			args = append(args, "--logprob-threshold", fmt.Sprintf("%.1f", job.LogprobThreshold))
			args = append(args, "--no-speech-threshold", fmt.Sprintf("%.1f", job.NoSpeechThreshold))
			
			// Add additional parameters
			args = append(args, "--temperature", fmt.Sprintf("%.1f", job.Temperature))
			args = append(args, "--best-of", fmt.Sprintf("%d", job.BestOf))
			args = append(args, "--beam-size", fmt.Sprintf("%d", job.BeamSize))
			args = append(args, "--patience", fmt.Sprintf("%.1f", job.Patience))
			args = append(args, "--length-penalty", fmt.Sprintf("%.1f", job.LengthPenalty))
			
			if job.SuppressNumerals {
				args = append(args, "--suppress-numerals")
			}
			
			if job.InitialPrompt != "" {
				args = append(args, "--initial-prompt", job.InitialPrompt)
			}
			
			args = append(args, "--temperature-increment-on-fallback", fmt.Sprintf("%.1f", job.TemperatureIncrementOnFallback))
			
			// Add HF token if available
			if hfToken := os.Getenv("HF_TOKEN"); hfToken != "" {
				args = append(args, "--hf-token", hfToken)
			}
		} else {
			// Use regular whisperx for non-diarized transcription with basic parameters only
			args = []string{"run", "whisperx", audioPath, "--model", model, "--compute_type", job.ComputeType, "--output_format", "json", "--output_dir", transcriptOutputDir}
			
			// Add batch size
			args = append(args, "--batch_size", fmt.Sprintf("%d", job.BatchSize))
			
			// Add basic transcription parameters that are definitely supported
			args = append(args, "--temperature", fmt.Sprintf("%.1f", job.Temperature))
			args = append(args, "--best_of", fmt.Sprintf("%d", job.BestOf))
			args = append(args, "--beam_size", fmt.Sprintf("%d", job.BeamSize))
			args = append(args, "--patience", fmt.Sprintf("%.1f", job.Patience))
			args = append(args, "--length_penalty", fmt.Sprintf("%.1f", job.LengthPenalty))
			
			if job.SuppressNumerals {
				args = append(args, "--suppress_numerals")
			}
			
			if job.InitialPrompt != "" {
				args = append(args, "--initial_prompt", job.InitialPrompt)
			}
		}

		// Command: uv run whisperx <audio_path> --model <model_size> --compute_type int8 --output_format json --output_dir <output_dir> [--diarize]
		cmd := exec.Command("uv", args...)

		// Store the command on the job so it can be terminated if requested.
		storeMutex.Lock()
		if jobStore[job.ID] != nil {
			jobStore[job.ID].Cmd = cmd
		}
		storeMutex.Unlock()

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Re-check status to see if it was terminated.
			// The error might be "signal: killed" if we terminated it.
			jobAfterRun, _ := GetJobStatus(job.ID)
			if jobAfterRun.Status == StatusTerminated {
				log.Printf("Job %s execution was stopped because it was terminated.", job.ID)
				// The status is already set, so we just continue.
			} else {
				errorMsg := fmt.Sprintf("transcription failed for job %s: %v. Output: %s", job.ID, err, string(output))
				log.Println(errorMsg)
				updateJobStatus(job.ID, StatusFailed, "Transcription command failed.")
			}
			continue
		}

		log.Printf("Transcription successful for job %s.", job.ID)

		if err := processJSONAndUpdateDB(job.ID, job.AudioID, transcriptOutputDir); err != nil {
			log.Printf("Error processing transcription results for job %s: %v", job.ID, err)
			updateJobStatus(job.ID, StatusFailed, "Failed to process transcription results.")
			continue
		}

		updateJobStatus(job.ID, StatusCompleted, "")
		log.Printf("Successfully completed job %s.", job.ID)
	}
}

// updateJobStatus updates the status and error message of a job in the store.
func updateJobStatus(jobID string, status JobStatus, errorMsg string) {
	storeMutex.Lock()
	defer storeMutex.Unlock()

	if job, exists := jobStore[jobID]; exists {
		job.Status = status
		job.Error = errorMsg
		// If the job is entering a final state, clear the command reference.
		if status == StatusCompleted || status == StatusFailed || status == StatusTerminated {
			job.Cmd = nil
		}
	}
}

// processJSONAndUpdateDB reads the .json output from whisperx or diarization script, parses it, and updates the database.
func processJSONAndUpdateDB(jobID, audioID, outputDir string) error {
	jsonFilePath := filepath.Join(outputDir, audioID+".json")

	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to read json file '%s': %w", jsonFilePath, err)
	}

	// Parse the JSON transcript directly
	var transcript models.JSONTranscript
	if err := json.Unmarshal(jsonData, &transcript); err != nil {
		return fmt.Errorf("failed to parse json file for job %s: %w", jobID, err)
	}

	// The dedicated diarization script already includes post-processing, so we don't need to do it again
	// Just store the result directly
	processedJSONData := jsonData

	// Store the JSON transcript
	db := database.GetDB()
	query := `UPDATE audio_records SET transcript = ?, speaker_map = ?, summary = ? WHERE id = ?`

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare database update statement: %w", err)
	}
	defer stmt.Close()

	// Set speaker_map and summary to an empty JSON object.
	_, err = stmt.Exec(string(processedJSONData), "{}", "{}", audioID)
	if err != nil {
		return fmt.Errorf("failed to update database with transcription results: %w", err)
	}

	log.Printf("Successfully updated database for audio record %s", audioID)
	return nil
}




