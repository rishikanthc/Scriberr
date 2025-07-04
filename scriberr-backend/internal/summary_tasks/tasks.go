package summary_tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"scriberr-backend/internal/database"
	"scriberr-backend/internal/models"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// JobStatus represents the status of a summarization job.
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

// Job represents a summarization job.
type Job struct {
	ID         string    `json:"id"`
	AudioID    string    `json:"audio_id"`
	TemplateID string    `json:"template_id"`
	Model      string    `json:"model"`
	Status     JobStatus `json:"status"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}



var (
	jobQueue     = make(chan Job, 100) // Buffered channel for jobs
	jobStore     = make(map[string]*Job)
	storeMutex   = &sync.RWMutex{}
	once         sync.Once
	openaiClient *openai.Client
	ollamaClient *OllamaClient
)

// Init starts the job queue worker for summarization.
// This should be called once when the application starts.
func Init() {
	once.Do(func() {
		log.Println("Initializing summary job queue worker...")

		// Initialize OpenAI client
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Println("WARNING: OPENAI_API_KEY environment variable not set. OpenAI summarization will not work.")
		} else {
			openaiClient = openai.NewClient(apiKey)
			log.Println("OpenAI client initialized")
		}

		// Initialize Ollama client
		ollamaClient = NewOllamaClient()
		if ollamaClient.IsAvailable(context.Background()) {
			log.Println("Ollama client initialized and available")
		} else {
			log.Println("WARNING: Ollama is not available. Ollama summarization will not work.")
		}

		go worker()
	})
}

// NewJob creates a new summary job and adds it to the queue.
func NewJob(audioID, templateID, model string) (*Job, error) {
	storeMutex.Lock()
	defer storeMutex.Unlock()

	// Optional: Check if a job for this audioID already exists and is not failed.
	for _, existingJob := range jobStore {
		if existingJob.AudioID == audioID && existingJob.TemplateID == templateID && (existingJob.Status == StatusPending || existingJob.Status == StatusProcessing) {
			return nil, fmt.Errorf("a summary job for this audio and template is already in progress (ID: %s)", existingJob.ID)
		}
	}

	job := &Job{
		ID:         uuid.NewString(),
		AudioID:    audioID,
		TemplateID: templateID,
		Model:      model,
		Status:     StatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	jobStore[job.ID] = job
	jobQueue <- *job // Send a copy to the queue

	log.Printf("New summary job created and queued. JobID: %s, AudioID: %s, TemplateID: %s", job.ID, job.AudioID, job.TemplateID)
	return job, nil
}

// GetJobStatus retrieves the status of a specific job.
func GetJobStatus(jobID string) (*Job, error) {
	storeMutex.RLock()
	defer storeMutex.RUnlock()

	job, exists := jobStore[jobID]
	if !exists {
		return nil, fmt.Errorf("summary job with ID %s not found", jobID)
	}

	return job, nil
}

// GetJobStatusByAudioID retrieves the status of the latest job for a given audio ID.
func GetJobStatusByAudioID(audioID string) (*Job, error) {
	storeMutex.RLock()
	defer storeMutex.RUnlock()

	var latestJob *Job
	for _, job := range jobStore {
		if job.AudioID == audioID {
			if latestJob == nil || job.CreatedAt.After(latestJob.CreatedAt) {
				latestJob = job
			}
		}
	}

	if latestJob == nil {
		return nil, fmt.Errorf("no summary job found for audio ID %s", audioID)
	}

	return latestJob, nil
}

// GetActiveJobs retrieves all summary jobs that are currently pending or processing.
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
				log.Printf("Could not retrieve title for audio_id %s for active summary job %s: %v", job.AudioID, job.ID, err)
				title = "Unknown Title" // Provide a default title
			}

			activeJobs = append(activeJobs, models.ActiveJob{
				ID:         job.ID,
				AudioID:    job.AudioID,
				AudioTitle: title,
				Status:     string(job.Status),
				Type:       "summarization",
				CreatedAt:  job.CreatedAt,
			})
		}
	}

	return activeJobs, nil
}

// worker is a long-running goroutine that processes summary jobs from the queue.
func worker() {
	for job := range jobQueue {
		log.Printf("Processing summary job %s for audio %s", job.ID, job.AudioID)
		updateJobStatus(job.ID, StatusProcessing, "")

		// Determine which service to use based on model name
		useOllama := strings.HasPrefix(job.Model, "ollama:")
		useOpenAI := !useOllama

		// Check if the required service is available
		if useOpenAI {
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				log.Printf("Error for job %s: OPENAI_API_KEY is not set.", job.ID)
				updateJobStatus(job.ID, StatusFailed, "OpenAI API key is not configured on the server.")
				continue
			}
		} else if useOllama {
			if !ollamaClient.IsAvailable(context.Background()) {
				log.Printf("Error for job %s: Ollama is not available.", job.ID)
				updateJobStatus(job.ID, StatusFailed, "Ollama is not available on the server.")
				continue
			}
		}

		db := database.GetDB()

		// 1. Fetch Audio Record and check for a transcript
		var transcript sql.NullString
		err := db.QueryRow("SELECT transcript FROM audio_records WHERE id = ?", job.AudioID).Scan(&transcript)
		if err != nil {
			log.Printf("Error fetching audio record for job %s: %v", job.ID, err)
			updateJobStatus(job.ID, StatusFailed, "Failed to fetch audio record from database.")
			continue
		}

		if !transcript.Valid || transcript.String == "" || transcript.String == "{}" || transcript.String == "[]" {
			log.Printf("Job %s failed: Transcript for audio %s is not available.", job.ID, job.AudioID)
			updateJobStatus(job.ID, StatusFailed, "Transcription not found or is empty. Please transcribe the audio first.")
			continue
		}

		// Parse the transcript JSON to plain text
		var transcriptData models.JSONTranscript
		if err := json.Unmarshal([]byte(transcript.String), &transcriptData); err != nil {
			log.Printf("Job %s failed: could not parse transcript JSON: %v", job.ID, err)
			updateJobStatus(job.ID, StatusFailed, "Failed to parse transcript data.")
			continue
		}

		var fullTranscriptText string
		for _, segment := range transcriptData.Segments {
			fullTranscriptText += segment.Text + " "
		}

		// 2. Fetch Summary Template
		var template models.SummaryTemplate
		err = db.QueryRow("SELECT id, title, prompt, created_at FROM summary_templates WHERE id = ?", job.TemplateID).Scan(&template.ID, &template.Title, &template.Prompt, &template.CreatedAt)
		if err != nil {
			log.Printf("Error fetching summary template for job %s: %v", job.ID, err)
			updateJobStatus(job.ID, StatusFailed, "Failed to fetch summary template from database.")
			continue
		}

		// 3. Call AI API (OpenAI or Ollama)
		finalPrompt := template.Prompt + "\n\n---\n\n" + fullTranscriptText + "\n\n---\n\nProvide a summary in markdown format. Return only the summary without code blocks or additional text."

		var summaryText string

		if useOpenAI {
			// Use OpenAI
			resp, err := openaiClient.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model: job.Model,
					Messages: []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleUser,
							Content: finalPrompt,
						},
					},
				},
			)

			if err != nil {
				log.Printf("Error from OpenAI API for job %s: %v", job.ID, err)
				updateJobStatus(job.ID, StatusFailed, "Failed to get summary from OpenAI.")
				continue
			}

			if len(resp.Choices) == 0 {
				log.Printf("OpenAI API returned no choices for job %s", job.ID)
				updateJobStatus(job.ID, StatusFailed, "OpenAI returned an empty response.")
				continue
			}

			summaryText = resp.Choices[0].Message.Content
		} else {
			// Use Ollama
			ollamaModel := strings.TrimPrefix(job.Model, "ollama:")
			options := &OllamaOptions{
				Temperature: 0.7,
				TopP:        0.9,
			}

			summaryText, err = ollamaClient.GenerateText(context.Background(), ollamaModel, finalPrompt, options)
			if err != nil {
				log.Printf("Error from Ollama API for job %s: %v", job.ID, err)
				updateJobStatus(job.ID, StatusFailed, "Failed to get summary from Ollama.")
				continue
			}
		}

		// 4. Update the database with the summary
		query := `UPDATE audio_records SET summary = ? WHERE id = ?`
		_, err = db.Exec(query, summaryText, job.AudioID)
		if err != nil {
			log.Printf("Error updating database with summary for job %s: %v", job.ID, err)
			updateJobStatus(job.ID, StatusFailed, "Failed to save summary to database.")
			continue
		}

		updateJobStatus(job.ID, StatusCompleted, "")
		log.Printf("Successfully completed summary job %s.", job.ID)
	}
}

// updateJobStatus safely updates the status of a job in the store.
func updateJobStatus(jobID string, status JobStatus, errorMsg string) {
	storeMutex.Lock()
	defer storeMutex.Unlock()

	if job, exists := jobStore[jobID]; exists {
		job.Status = status
		job.Error = errorMsg
	}
}

// GetAvailableModels returns all available models for summarization
func GetAvailableModels() (map[string][]string, error) {
	models := make(map[string][]string)
	
	// Add OpenAI models
	openaiModels := []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo", "gpt-4o", "gpt-4o-mini"}
	models["openai"] = openaiModels
	
	// Add Ollama models if available
	if ollamaClient != nil && ollamaClient.IsAvailable(context.Background()) {
		ollamaModels, err := ollamaClient.GetModels(context.Background())
		if err != nil {
			log.Printf("Failed to get Ollama models: %v", err)
		} else {
			var modelNames []string
			for _, model := range ollamaModels {
				modelNames = append(modelNames, "ollama:"+model.Name)
			}
			models["ollama"] = modelNames
		}
	}
	
	return models, nil
}
