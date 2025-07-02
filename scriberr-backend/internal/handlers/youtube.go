package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"scriberr-backend/internal/database"
	"scriberr-backend/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// YouTubeDownloadRequest defines the structure for the YouTube download request.
type YouTubeDownloadRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// DownloadYouTubeAudio handles the request to download and process a YouTube video.
func DownloadYouTubeAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req YouTubeDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		writeJSONError(w, "Missing YouTube URL", http.StatusBadRequest)
		return
	}

	// Validate YouTube URL
	if !isValidYouTubeURL(req.URL) {
		writeJSONError(w, "Invalid YouTube URL", http.StatusBadRequest)
		return
	}

	// Generate unique ID for this audio record
	recordID := uuid.NewString()

	// Set default title if not provided
	if req.Title == "" {
		req.Title = "YouTube Video"
	}

	// Ensure storage directories exist
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

	// Define file paths
	// yt-dlp will add its own extension, so we use a template
	originalFilePathTemplate := filepath.Join(uploadsDir, recordID+".%(ext)s")
	convertedFileName := recordID + ".wav"
	convertedFilePath := filepath.Join(convertedDir, convertedFileName)

	// Step 1: Download audio from YouTube using yt-dlp
	log.Printf("Starting YouTube download for URL: %s", req.URL)
	
	// yt-dlp command to download best audio quality
	// -x: extract audio only
	// --audio-format best: get best available audio format
	// -o: output template with extension
	cmd := exec.Command("uv", "run", "yt-dlp", 
		"-x",                    // extract audio only
		"--audio-format", "best", // get best available audio format
		"-o", originalFilePathTemplate,   // output path template
		req.URL)                 // YouTube URL

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("yt-dlp download failed for URL %s. Error: %v. Output: %s", req.URL, err, string(output))
		writeJSONError(w, fmt.Sprintf("Failed to download YouTube video: %s", string(output)), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully downloaded YouTube audio")

	// Find the actual downloaded file (yt-dlp adds its own extension)
	originalFilePath, err := findDownloadedFile(uploadsDir, recordID)
	if err != nil {
		log.Printf("Error finding downloaded file for %s: %v", recordID, err)
		writeJSONError(w, "Failed to locate downloaded file", http.StatusInternalServerError)
		return
	}
	log.Printf("Found downloaded file: %s", originalFilePath)

	// Step 2: Insert metadata into database
	db := database.GetDB()
	audioRecord := models.Audio{
		ID:         recordID,
		Title:      req.Title,
		CreatedAt:  time.Now().UTC(),
		Transcript: "{}", // Default empty JSON object
		SpeakerMap: "{}", // Default empty JSON object
		Summary:    "{}", // Default empty JSON object
	}

	query := `INSERT INTO audio_records (id, title, transcript, speaker_map, summary, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing database statement: %v", err)
		// Cleanup downloaded file if DB preparation fails
		os.Remove(originalFilePath)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	if _, err := stmt.Exec(audioRecord.ID, audioRecord.Title, audioRecord.Transcript, audioRecord.SpeakerMap, audioRecord.Summary, audioRecord.CreatedAt); err != nil {
		log.Printf("Error executing database insert: %v", err)
		// Cleanup downloaded file if DB insert fails
		os.Remove(originalFilePath)
		writeJSONError(w, "Failed to create record in database", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully inserted record %s into database", recordID)

	// Step 3: Convert audio to WAV using ffmpeg
	log.Printf("Converting downloaded audio to WAV format")
	// Command: ffmpeg -i "${inputPath}" -ar 16000 -ac 1 -c:a pcm_s16le "${outputPath}"
	convertCmd := exec.Command("ffmpeg", "-i", originalFilePath, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", convertedFilePath)

	convertOutput, err := convertCmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg conversion failed for %s. Error: %v. Output: %s", recordID, err, string(convertOutput))
		// If conversion fails, clean up the database record and downloaded file
		deleteQuery := "DELETE FROM audio_records WHERE id = ?"
		if _, delErr := db.Exec(deleteQuery, recordID); delErr != nil {
			log.Printf("CRITICAL: Failed to delete database record %s after ffmpeg failure: %v", recordID, delErr)
		}
		os.Remove(originalFilePath)
		writeJSONError(w, fmt.Sprintf("Failed to convert audio file: %s", string(convertOutput)), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully converted file to %s", convertedFilePath)

	// Step 4: Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": recordID})
}

// isValidYouTubeURL checks if the provided URL is a valid YouTube URL.
func isValidYouTubeURL(url string) bool {
	url = strings.ToLower(url)
	
	// Check for exact YouTube domain patterns
	youtubeDomains := []string{
		"youtube.com/watch",
		"youtu.be/",
		"youtube.com/embed/",
		"youtube.com/v/",
		"youtube.com/shorts/",
	}

	for _, pattern := range youtubeDomains {
		if strings.Contains(url, pattern) {
			// Ensure the pattern is preceded by a valid domain
			if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
				// Additional check to ensure it's not a subdomain or fake domain
				if strings.HasPrefix(url, "http://youtube.com") || 
				   strings.HasPrefix(url, "https://youtube.com") ||
				   strings.HasPrefix(url, "http://www.youtube.com") ||
				   strings.HasPrefix(url, "https://www.youtube.com") ||
				   strings.HasPrefix(url, "http://youtu.be") ||
				   strings.HasPrefix(url, "https://youtu.be") {
					return true
				}
			}
		}
	}
	return false
}

// findDownloadedFile finds the actual downloaded file by looking for files that start with the recordID
func findDownloadedFile(uploadsDir, recordID string) (string, error) {
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), recordID+".") {
			return filepath.Join(uploadsDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no downloaded file found for record ID %s", recordID)
} 