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
func NewJob(audioID string, model string, diarize bool, minSpeakers int, maxSpeakers int) (*Job, error) {
	storeMutex.Lock()
	defer storeMutex.Unlock()

	if model == "" {
		model = "small" // Default model
	}

	// Set default values if not provided
	if minSpeakers == 0 {
		minSpeakers = 1
	}
	if maxSpeakers == 0 {
		maxSpeakers = 2
	}

	job := &Job{
		ID:          uuid.NewString(),
		AudioID:     audioID,
		Model:       model,
		Diarize:     diarize,
		MinSpeakers: minSpeakers,
		MaxSpeakers: maxSpeakers,
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
		Cmd:         nil,
	}

	jobStore[job.ID] = job
	jobQueue <- *job // Send a copy of the job to the queue

	log.Printf("New job created and queued. JobID: %s, AudioID: %s, Model: %s, Diarize: %t, MinSpeakers: %d, MaxSpeakers: %d", job.ID, job.AudioID, job.Model, job.Diarize, job.MinSpeakers, job.MaxSpeakers)
	return job, nil
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
		args := []string{"run", "whisperx", audioPath, "--model", model, "--compute_type", "int8", "--output_format", "json", "--output_dir", transcriptOutputDir}
		
		// Add diarization flag if enabled
		if job.Diarize {
			args = append(args, "--diarize")
			// Add speaker count parameters
			args = append(args, "--min_speakers", fmt.Sprintf("%d", job.MinSpeakers))
			args = append(args, "--max_speakers", fmt.Sprintf("%d", job.MaxSpeakers))
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

// processJSONAndUpdateDB reads the .json output from whisperx, parses it, and updates the database.
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

	// Store the complete JSON transcript as-is
	db := database.GetDB()
	query := `UPDATE audio_records SET transcript = ?, speaker_map = ?, summary = ? WHERE id = ?`

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare database update statement: %w", err)
	}
	defer stmt.Close()

	// Set speaker_map and summary to an empty JSON object.
	_, err = stmt.Exec(string(jsonData), "{}", "{}", audioID)
	if err != nil {
		return fmt.Errorf("failed to update database with transcription results: %w", err)
	}

	log.Printf("Successfully updated database for audio record %s", audioID)
	return nil
}


