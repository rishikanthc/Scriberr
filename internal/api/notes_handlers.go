package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"scriberr/internal/models"
)

// NoteCreateRequest is the payload for creating a note
type NoteCreateRequest struct {
	// Use gte=0 so 0 is valid (first word/time); avoid 'required' which fails for zero values
	StartWordIndex int     `json:"start_word_index" binding:"gte=0"`
	EndWordIndex   int     `json:"end_word_index" binding:"gte=0"`
	StartTime      float64 `json:"start_time" binding:"gte=0"`
	EndTime        float64 `json:"end_time" binding:"gte=0"`
	Quote          string  `json:"quote" binding:"required,min=1"`
	Content        string  `json:"content" binding:"required,min=1"`
}

// NoteUpdateRequest updates content of a note
type NoteUpdateRequest struct {
	Content string `json:"content" binding:"required,min=1"`
}

// ListNotes returns all notes for a transcription
// @Summary List notes for a transcription
// @Description Get all notes attached to a transcription, ordered by time and creation
// @Tags notes
// @Produce json
// @Param id path string true "Transcription ID"
// @Success 200 {array} models.Note
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /api/v1/transcription/{id}/notes [get]
func (h *Handler) ListNotes(c *gin.Context) {
	transcriptionID := c.Param("id")
	if transcriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID is required"})
		return
	}

	// Ensure transcription exists
	_, err := h.jobRepo.FindByID(c.Request.Context(), transcriptionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transcription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transcription"})
		return
	}

	notes, err := h.noteRepo.ListByJob(c.Request.Context(), transcriptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notes"})
		return
	}

	c.JSON(http.StatusOK, notes)
}

// CreateNote stores a new note for a transcription
// @Summary Create a note for a transcription
// @Description Create a new note attached to the specified transcription
// @Tags notes
// @Accept json
// @Produce json
// @Param id path string true "Transcription ID"
// @Param request body NoteCreateRequest true "Note create payload"
// @Success 201 {object} models.Note
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /api/v1/transcription/{id}/notes [post]
func (h *Handler) CreateNote(c *gin.Context) {
	transcriptionID := c.Param("id")
	if transcriptionID == "" {
		log.Printf("notes.CreateNote: missing transcription ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID is required"})
		return
	}

	var req NoteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("notes.CreateNote: invalid payload for transcription %s: %v", transcriptionID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "details": err.Error()})
		return
	}

	if req.EndWordIndex < req.StartWordIndex {
		log.Printf("notes.CreateNote: invalid indices (start=%d end=%d) for transcription %s", req.StartWordIndex, req.EndWordIndex, transcriptionID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_word_index must be >= start_word_index", "start_word_index": req.StartWordIndex, "end_word_index": req.EndWordIndex})
		return
	}
	if req.EndTime < req.StartTime {
		log.Printf("notes.CreateNote: invalid times (start=%.3f end=%.3f) for transcription %s", req.StartTime, req.EndTime, transcriptionID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_time must be >= start_time", "start_time": req.StartTime, "end_time": req.EndTime})
		return
	}

	// Ensure transcription exists
	_, err := h.jobRepo.FindByID(c.Request.Context(), transcriptionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("notes.CreateNote: transcription %s not found", transcriptionID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Transcription not found"})
			return
		}
		log.Printf("notes.CreateNote: failed to fetch transcription %s: %v", transcriptionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transcription"})
		return
	}

	n := &models.Note{
		ID:              uuid.New().String(),
		TranscriptionID: transcriptionID,
		StartWordIndex:  req.StartWordIndex,
		EndWordIndex:    req.EndWordIndex,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		Quote:           req.Quote,
		Content:         req.Content,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := h.noteRepo.Create(c.Request.Context(), n); err != nil {
		log.Printf("notes.CreateNote: DB error creating note for transcription %s (start=%d end=%d startTime=%.3f endTime=%.3f): %v", transcriptionID, n.StartWordIndex, n.EndWordIndex, n.StartTime, n.EndTime, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create note"})
		return
	}

	log.Printf("notes.CreateNote: created note %s for transcription %s (start=%d end=%d startTime=%.3f endTime=%.3f quoteLen=%d)", n.ID, transcriptionID, n.StartWordIndex, n.EndWordIndex, n.StartTime, n.EndTime, len(n.Quote))
	// Tests expect 200 on creation
	c.JSON(http.StatusOK, n)
}

// GetNote returns a note by ID
// @Summary Get a note
// @Description Get a note by its ID
// @Tags notes
// @Produce json
// @Param note_id path string true "Note ID"
// @Success 200 {object} models.Note
// @Failure 404 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /api/v1/notes/{note_id} [get]
func (h *Handler) GetNote(c *gin.Context) {
	noteID := c.Param("note_id")
	n, err := h.noteRepo.FindByID(c.Request.Context(), noteID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch note"})
		return
	}
	c.JSON(http.StatusOK, n)
}

// UpdateNote updates the content of an existing note
// @Summary Update a note
// @Description Update the content of a note
// @Tags notes
// @Accept json
// @Produce json
// @Param note_id path string true "Note ID"
// @Param request body NoteUpdateRequest true "Note update payload"
// @Success 200 {object} models.Note
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /api/v1/notes/{note_id} [put]
func (h *Handler) UpdateNote(c *gin.Context) {
	noteID := c.Param("note_id")
	var req NoteUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n, err := h.noteRepo.FindByID(c.Request.Context(), noteID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch note"})
		return
	}

	n.Content = req.Content
	n.UpdatedAt = time.Now()

	if err := h.noteRepo.Update(c.Request.Context(), n); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update note"})
		return
	}

	c.JSON(http.StatusOK, n)
}

// DeleteNote removes a note by ID
// @Summary Delete a note
// @Description Delete a note by its ID
// @Tags notes
// @Produce json
// @Param note_id path string true "Note ID"
// @Success 204 {string} string "No Content"
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /api/v1/notes/{note_id} [delete]
func (h *Handler) DeleteNote(c *gin.Context) {
	noteID := c.Param("note_id")
	if err := h.noteRepo.Delete(c.Request.Context(), noteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete note"})
		return
	}
	// Tests expect 200 on deletion
	c.JSON(http.StatusOK, gin.H{"message": "Note deleted"})
}
