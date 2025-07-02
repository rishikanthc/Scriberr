package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"scriberr-backend/internal/summary_tasks"
)

// SummarizeRequest defines the structure for the summarization request.
type SummarizeRequest struct {
	AudioID    string `json:"audio_id"`
	TemplateID string `json:"template_id"`
	Model      string `json:"model"`
}

// SummarizeAudio handles the request to start a new summarization job.
func SummarizeAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AudioID == "" || req.TemplateID == "" {
		writeJSONError(w, "Missing audio_id or template_id", http.StatusBadRequest)
		return
	}

	// Set default model if not provided
	if req.Model == "" {
		req.Model = "gpt-3.5-turbo"
	}

	job, err := summary_tasks.NewJob(req.AudioID, req.TemplateID, req.Model)
	if err != nil {
		log.Printf("Error creating new summary job for audio ID %s: %v", req.AudioID, err)
		// It's good to return the specific error from NewJob, as it might contain useful info
		// like "job already in progress".
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"job_id": job.ID})
}

// GetSummarizeStatus handles requests to check the status of a summarization job.
func GetSummarizeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.PathValue("jobid")
	if jobID == "" {
		writeJSONError(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	job, err := summary_tasks.GetJobStatus(jobID)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		log.Printf("Error encoding summary job status to JSON for JobID %s: %v", jobID, err)
	}
}

// GetSummarizeStatusByAudioID handles requests to check the status of the latest summary job for a given audio ID.
func GetSummarizeStatusByAudioID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	audioID := r.PathValue("id")
	if audioID == "" {
		writeJSONError(w, "Missing audio ID", http.StatusBadRequest)
		return
	}

	job, err := summary_tasks.GetJobStatusByAudioID(audioID)
	if err != nil {
		// This will correctly return a 404 if no job is found.
		writeJSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		log.Printf("Error encoding summary job status to JSON for AudioID %s: %v", audioID, err)
	}
}
