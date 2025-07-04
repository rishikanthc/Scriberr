package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"scriberr-backend/internal/database"
	"scriberr-backend/internal/models"
	"scriberr-backend/internal/summary_tasks"
	"scriberr-backend/internal/tasks"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	uploadsDir    = "./storage/uploads"
	convertedDir  = "./storage/converted"
	maxUploadSize = 2 << 30 // 2 GB
)

// writeJSONError sends a JSON formatted error message.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// TranscribeAudio handles the request to start a new transcription job.
func TranscribeAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AudioID     string `json:"audio_id"`
		ModelSize   string `json:"model_size"`
		Diarize     bool   `json:"diarize"`
		MinSpeakers int    `json:"min_speakers"`
		MaxSpeakers int    `json:"max_speakers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AudioID == "" {
		writeJSONError(w, "Missing audio_id", http.StatusBadRequest)
		return
	}

	// Optional: Check if the audio record and file actually exist before creating a job
	// For now, we'll trust the client and let the worker handle file-not-found errors.

	// Set default values if not provided
	if req.MinSpeakers == 0 {
		req.MinSpeakers = 1
	}
	if req.MaxSpeakers == 0 {
		req.MaxSpeakers = 2
	}

	job, err := tasks.NewJob(req.AudioID, req.ModelSize, req.Diarize, req.MinSpeakers, req.MaxSpeakers)
	if err != nil {
		log.Printf("Error creating new job for audio ID %s: %v", req.AudioID, err)
		writeJSONError(w, "Failed to create transcription job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"job_id": job.ID})
}

// GetTranscriptionStatus handles requests to check the status of a transcription job.
func GetTranscriptionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.PathValue("jobid")
	if jobID == "" {
		writeJSONError(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	job, err := tasks.GetJobStatus(jobID)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		log.Printf("Error encoding job status to JSON for JobID %s: %v", jobID, err)
	}
}

// GetActiveJobs retrieves all currently active (pending or processing) jobs.
func GetActiveJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	transcriptionJobs, err := tasks.GetActiveJobs()
	if err != nil {
		log.Printf("Error retrieving active transcription jobs: %v", err)
		writeJSONError(w, "Failed to retrieve active jobs", http.StatusInternalServerError)
		return
	}

	summaryJobs, err := summary_tasks.GetActiveJobs()
	if err != nil {
		log.Printf("Error retrieving active summary jobs: %v", err)
		writeJSONError(w, "Failed to retrieve active jobs", http.StatusInternalServerError)
		return
	}

	allActiveJobs := append(transcriptionJobs, summaryJobs...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(allActiveJobs); err != nil {
		log.Printf("Error encoding active jobs to JSON: %v", err)
	}
}

// TerminateJob handles a request to stop a transcription job.
func TerminateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.PathValue("jobid")
	if jobID == "" {
		writeJSONError(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	if err := tasks.TerminateJob(jobID); err != nil {
		log.Printf("Error terminating job %s: %v", jobID, err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			writeJSONError(w, errMsg, http.StatusNotFound)
		} else if strings.Contains(errMsg, "already finished") {
			writeJSONError(w, errMsg, http.StatusBadRequest)
		} else {
			writeJSONError(w, "Failed to terminate job", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Job %s termination request processed successfully.", jobID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Job termination initiated."})
}

// GetAudioRecord retrieves a single audio record from the database.
func GetAudioRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, "Missing record ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	query := "SELECT id, title, transcript, speaker_map, summary, created_at FROM audio_records WHERE id = ?"
	row := db.QueryRow(query, id)

	var record models.Audio
	var transcript, speakerMap, summary sql.NullString

	err := row.Scan(&record.ID, &record.Title, &transcript, &speakerMap, &summary, &record.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, "Record not found", http.StatusNotFound)
		} else {
			log.Printf("Error scanning audio record row for ID %s: %v", id, err)
			writeJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Safely assign nullable string values, defaulting to an empty JSON object string.
	if transcript.Valid {
		record.Transcript = transcript.String
	} else {
		record.Transcript = "{}"
	}
	if speakerMap.Valid {
		record.SpeakerMap = speakerMap.String
	} else {
		record.SpeakerMap = "{}"
	}
	if summary.Valid {
		record.Summary = summary.String
	} else {
		record.Summary = ""
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(record); err != nil {
		log.Printf("Error encoding audio record to JSON for ID %s: %v", id, err)
	}
}

// GetAudioFile serves the converted .wav audio file for playback.
func GetAudioFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, "Missing record ID", http.StatusBadRequest)
		return
	}

	// Serve the converted .wav file which is in a standard format for browsers.
	filePath := filepath.Join(convertedDir, id+".wav")

	// Verify the file exists before attempting to serve it.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Audio file not found at expected path: %s. Error: %v", filePath, err)
		writeJSONError(w, "Audio file not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}

// UpdateAudioTitle handles updating the title of an audio record.
func UpdateAudioTitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, "Missing record ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeJSONError(w, "Title cannot be empty", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	query := "UPDATE audio_records SET title = ? WHERE id = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing update statement for record %s: %v", id, err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(req.Title, id)
	if err != nil {
		log.Printf("Error updating record %s: %v", id, err)
		writeJSONError(w, "Failed to update record", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for update on record %s: %v", id, err)
		writeJSONError(w, "Failed to verify update", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		writeJSONError(w, "Record not found", http.StatusNotFound)
		return
	}

	log.Printf("Successfully updated title for record %s", id)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Title updated successfully"})
}

// CreateAudio handles the creation of a new audio record.
// It accepts a multipart/form-data request with an "audio" file and a "title" field.
func CreateAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ensure the storage directories exist.
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Printf("Error creating uploads directory: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := os.MkdirAll(convertedDir, 0755); err != nil {
		log.Printf("Error creating converted directory: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSONError(w, "The uploaded file is too big. Please choose a file that is less than 2GB in size.", http.StatusBadRequest)
		return
	}

	// Get the file from the form data.
	file, handler, err := r.FormFile("audio")
	if err != nil {
		writeJSONError(w, "Invalid file upload. Make sure the form field name is 'audio'.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get the title from the form data.
	title := r.FormValue("title")
	if title == "" {
		title = "Untitled"
	}

	// --- Step 1: Generate ID and Paths ---
	recordID := uuid.NewString()
	originalFileExt := filepath.Ext(handler.Filename)
	originalFileName := recordID + originalFileExt
	originalFilePath := filepath.Join(uploadsDir, originalFileName)

	convertedFileName := recordID + ".wav"
	convertedFilePath := filepath.Join(convertedDir, convertedFileName)

	// --- Step 2: Store the Original File ---
	dst, err := os.Create(originalFilePath)
	if err != nil {
		log.Printf("Error creating destination file: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("Error copying uploaded file: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully uploaded and saved original file to %s", originalFilePath)

	// --- Step 3: Update Metadata in Database ---
	db := database.GetDB()
	audioRecord := models.Audio{
		ID:         recordID,
		Title:      title,
		CreatedAt:  time.Now().UTC(),
		Transcript: "{}", // Default empty JSON object
		SpeakerMap: "{}", // Default empty JSON object
		Summary:    "{}", // Default empty JSON object
	}

	query := `INSERT INTO audio_records (id, title, transcript, speaker_map, summary, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing database statement: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	if _, err := stmt.Exec(audioRecord.ID, audioRecord.Title, audioRecord.Transcript, audioRecord.SpeakerMap, audioRecord.Summary, audioRecord.CreatedAt); err != nil {
		log.Printf("Error executing database insert: %v", err)
		// Cleanup the uploaded file if DB insert fails
		os.Remove(originalFilePath)
		writeJSONError(w, "Failed to create record in database", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully inserted record %s into database", recordID)

	// --- Step 4: Convert Audio to WAV using ffmpeg ---
	// Command: ffmpeg -i "${inputPath}" -ar 16000 -ac 1 -c:a pcm_s16le "${outputPath}"
	cmd := exec.Command("ffmpeg", "-i", originalFilePath, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", convertedFilePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg conversion failed for %s. Error: %v. Output: %s", recordID, err, string(output))
		// If conversion fails, we should ideally roll back the DB transaction
		// or at least delete the entry to avoid inconsistent state.
		// For now, we'll delete the DB record and the uploaded file.
		deleteQuery := "DELETE FROM audio_records WHERE id = ?"
		if _, delErr := db.Exec(deleteQuery, recordID); delErr != nil {
			log.Printf("CRITICAL: Failed to delete database record %s after ffmpeg failure: %v", recordID, delErr)
		}
		os.Remove(originalFilePath)

		writeJSONError(w, fmt.Sprintf("Failed to convert audio file: %s", string(output)), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully converted file to %s", convertedFilePath)

	// --- Step 5: Return Success Response ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": recordID})
}

// GetAllAudioRecords retrieves all audio records from the database.
func GetAllAudioRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := database.GetDB()
	rows, err := db.Query("SELECT id, title, created_at, transcript, summary FROM audio_records ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Error querying database for audio records: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var records []models.Audio
	for rows.Next() {
		var record models.Audio
		var transcript, summary sql.NullString
		// Note: We scan the transcript and summary fields.
		if err := rows.Scan(&record.ID, &record.Title, &record.CreatedAt, &transcript, &summary); err != nil {
			log.Printf("Error scanning audio record row: %v", err)
			writeJSONError(w, "Failed to read record from database", http.StatusInternalServerError)
			return
		}

		if transcript.Valid {
			record.Transcript = transcript.String
		} else {
			record.Transcript = "{}"
		}
		if summary.Valid {
			record.Summary = summary.String
		} else {
			record.Summary = "{}"
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error after iterating over audio record rows: %v", err)
		writeJSONError(w, "Error processing database results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(records); err != nil {
		log.Printf("Error encoding audio records to JSON: %v", err)
		// It's too late to send a different status code here, but we log the error.
	}
}

// DeleteAudio handles the deletion of an audio record and its associated files.
func DeleteAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, "Missing record ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// It's good practice to verify the record exists before trying to delete files.
	// This also helps prevent trying to delete things based on a bogus ID.
	var title string
	err := db.QueryRow("SELECT title FROM audio_records WHERE id = ?", id).Scan(&title)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, "Record not found", http.StatusNotFound)
		} else {
			log.Printf("Error checking for record %s: %v", id, err)
			writeJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Delete record from the database first. If this fails, we don't want to orphan files.
	query := "DELETE FROM audio_records WHERE id = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing delete statement for record %s: %v", id, err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	if _, err := stmt.Exec(id); err != nil {
		log.Printf("Error deleting record %s from database: %v", id, err)
		writeJSONError(w, "Failed to delete record from database", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted record %s from database", id)

	// --- File Deletion ---
	// Since we don't store the original extension, we need to search for the file.
	foundFiles, err := filepath.Glob(filepath.Join(uploadsDir, id+".*"))
	if err != nil {
		log.Printf("Error searching for original file for record %s: %v", id, err)
		// The DB record is already deleted, so we just log this error and continue.
	}
	for _, f := range foundFiles {
		if err := os.Remove(f); err != nil {
			log.Printf("Failed to delete original file %s for record %s: %v", f, id, err)
		} else {
			log.Printf("Successfully deleted original file %s", f)
		}
	}

	// Delete the converted file.
	convertedFilePath := filepath.Join(convertedDir, id+".wav")
	if err := os.Remove(convertedFilePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to delete converted file %s for record %s: %v", convertedFilePath, id, err)
	} else if err == nil {
		log.Printf("Successfully deleted converted file %s", convertedFilePath)
	}

	w.WriteHeader(http.StatusNoContent)
}

// DownloadTranscript handles requests to download a transcript in different formats.
func DownloadTranscript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, "Missing record ID", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "txt" // default format
	}

	// Validate format
	validFormats := map[string]bool{"txt": true, "json": true, "srt": true}
	if !validFormats[format] {
		writeJSONError(w, "Invalid format. Supported formats: txt, json, srt", http.StatusBadRequest)
		return
	}

	// Get the audio record from database
	db := database.GetDB()
	query := "SELECT id, title, transcript FROM audio_records WHERE id = ?"
	row := db.QueryRow(query, id)

	var record models.Audio
	var transcript sql.NullString

	err := row.Scan(&record.ID, &record.Title, &transcript)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, "Record not found", http.StatusNotFound)
		} else {
			log.Printf("Error scanning audio record row for ID %s: %v", id, err)
			writeJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Check if transcript exists
	if !transcript.Valid || transcript.String == "{}" || transcript.String == "" {
		writeJSONError(w, "No transcript available for this recording", http.StatusNotFound)
		return
	}

	// Parse the transcript JSON
	var transcriptData models.JSONTranscript
	if err := json.Unmarshal([]byte(transcript.String), &transcriptData); err != nil {
		log.Printf("Error parsing transcript JSON for ID %s: %v", id, err)
		writeJSONError(w, "Invalid transcript data", http.StatusInternalServerError)
		return
	}

	// Generate filename
	filename := fmt.Sprintf("%s_transcript.%s", record.Title, format)
	// Sanitize filename
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")

	// Set appropriate headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	var content string
	var contentType string

	switch format {
	case "txt":
		contentType = "text/plain"
		content = generateTxtTranscript(transcriptData)
	case "json":
		contentType = "application/json"
		content = transcript.String // Use the original JSON
	case "srt":
		contentType = "text/plain"
		content = generateSrtTranscript(transcriptData)
	default:
		writeJSONError(w, "Unsupported format", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

// generateTxtTranscript creates a plain text version of the transcript without timestamps
func generateTxtTranscript(transcript models.JSONTranscript) string {
	var lines []string
	for _, segment := range transcript.Segments {
		lines = append(lines, segment.Text)
	}
	return strings.Join(lines, " ")
}

// generateSrtTranscript creates an SRT format transcript with timestamps
func generateSrtTranscript(transcript models.JSONTranscript) string {
	var lines []string
	for i, segment := range transcript.Segments {
		// SRT format: sequence number, timestamp, text, blank line
		lines = append(lines, fmt.Sprintf("%d", i+1))
		lines = append(lines, fmt.Sprintf("%s --> %s", 
			formatSrtTime(segment.Start), 
			formatSrtTime(segment.End)))
		lines = append(lines, segment.Text)
		lines = append(lines, "") // blank line
	}
	return strings.Join(lines, "\n")
}

// formatSrtTime converts seconds to SRT timestamp format (HH:MM:SS,mmm)
func formatSrtTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millisecs := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millisecs)
}
