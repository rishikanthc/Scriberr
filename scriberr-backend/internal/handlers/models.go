package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"scriberr-backend/internal/summary_tasks"
)

// GetAvailableModels handles requests to get available models for summarization.
func GetAvailableModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := summary_tasks.GetAvailableModels()
	if err != nil {
		log.Printf("Error getting available models: %v", err)
		writeJSONError(w, "Failed to get available models", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(models); err != nil {
		log.Printf("Error encoding models to JSON: %v", err)
	}
} 