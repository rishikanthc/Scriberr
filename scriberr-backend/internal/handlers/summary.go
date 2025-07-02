package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"scriberr-backend/internal/database"
	"scriberr-backend/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateOrUpdateSummaryTemplate handles creating or updating a summary template.
// This function performs an "upsert" operation.
// If an ID is provided in the body, it updates the record.
// If the ID is null or not provided, it creates a new record.
func CreateOrUpdateSummaryTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use a struct with a pointer for the ID to handle null values from JSON.
	type TemplatePayload struct {
		ID     *string `json:"id"`
		Title  string  `json:"title"`
		Prompt string  `json:"prompt"`
	}

	var payload TemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Prompt) == "" {
		writeJSONError(w, "Title and prompt cannot be empty", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// If ID is nil or empty, it's a new template.
	if payload.ID == nil || *payload.ID == "" {
		newTemplate := models.SummaryTemplate{
			ID:        uuid.NewString(),
			Title:     payload.Title,
			Prompt:    payload.Prompt,
			CreatedAt: time.Now().UTC(),
		}
		query := `INSERT INTO summary_templates (id, title, prompt, created_at) VALUES (?, ?, ?, ?)`
		_, err := db.Exec(query, newTemplate.ID, newTemplate.Title, newTemplate.Prompt, newTemplate.CreatedAt)
		if err != nil {
			log.Printf("Error creating summary template: %v", err)
			writeJSONError(w, "Failed to create summary template", http.StatusInternalServerError)
			return
		}
		log.Printf("New summary template created with ID: %s", newTemplate.ID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newTemplate)
		return
	}

	// If ID is provided, it's an update.
	templateID := *payload.ID
	query := `UPDATE summary_templates SET title = ?, prompt = ? WHERE id = ?`
	res, err := db.Exec(query, payload.Title, payload.Prompt, templateID)
	if err != nil {
		log.Printf("Error updating summary template %s: %v", templateID, err)
		writeJSONError(w, "Failed to update summary template", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for update on template %s: %v", templateID, err)
		writeJSONError(w, "Failed to verify update", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		writeJSONError(w, "Summary template not found, no update performed", http.StatusNotFound)
		return
	}

	// Fetch the updated record to return it with the correct created_at timestamp
	var updatedTemplate models.SummaryTemplate
	err = db.QueryRow("SELECT id, title, prompt, created_at FROM summary_templates WHERE id = ?", templateID).Scan(&updatedTemplate.ID, &updatedTemplate.Title, &updatedTemplate.Prompt, &updatedTemplate.CreatedAt)
	if err != nil {
		log.Printf("Error fetching updated template %s: %v", templateID, err)
		writeJSONError(w, "Failed to fetch template after update", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated summary template %s", templateID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedTemplate)
}

// GetAllSummaryTemplates retrieves all summary templates from the database.
func GetAllSummaryTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := database.GetDB()
	query := `SELECT id, title, prompt, created_at FROM summary_templates ORDER BY created_at DESC`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying summary templates: %v", err)
		writeJSONError(w, "Failed to retrieve summary templates", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var templates []models.SummaryTemplate
	for rows.Next() {
		var t models.SummaryTemplate
		if err := rows.Scan(&t.ID, &t.Title, &t.Prompt, &t.CreatedAt); err != nil {
			log.Printf("Error scanning summary template row: %v", err)
			// In a real app, you might want to decide if you should return a partial list or an error.
			// For now, we'll continue and log the error.
			continue
		}
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating summary template rows: %v", err)
		writeJSONError(w, "Error processing results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if len(templates) == 0 {
		w.Write([]byte("[]")) // Return empty JSON array instead of null
		return
	}
	json.NewEncoder(w).Encode(templates)
}

// GetSummaryTemplate retrieves a single summary template by ID.
func GetSummaryTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	templateID := r.PathValue("id")
	if templateID == "" {
		writeJSONError(w, "Template ID is required", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	var t models.SummaryTemplate
	query := `SELECT id, title, prompt, created_at FROM summary_templates WHERE id = ?`
	err := db.QueryRow(query, templateID).Scan(&t.ID, &t.Title, &t.Prompt, &t.CreatedAt)
	if err != nil {
		log.Printf("Could not find summary template with ID %s: %v", templateID, err)
		writeJSONError(w, "Summary template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

// DeleteSummaryTemplate handles deleting a summary template by ID.
func DeleteSummaryTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	templateID := r.PathValue("id")
	if templateID == "" {
		writeJSONError(w, "Template ID is required", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	query := `DELETE FROM summary_templates WHERE id = ?`
	res, err := db.Exec(query, templateID)
	if err != nil {
		log.Printf("Error deleting summary template with ID %s: %v", templateID, err)
		writeJSONError(w, "Failed to delete summary template", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for delete on template %s: %v", templateID, err)
		writeJSONError(w, "Failed to verify delete", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		writeJSONError(w, "Summary template not found, no delete performed", http.StatusNotFound)
		return
	}

	log.Printf("Successfully deleted summary template %s", templateID)
	w.WriteHeader(http.StatusNoContent)
}
