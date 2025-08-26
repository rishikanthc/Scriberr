package api

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "gorm.io/gorm"

    "scriberr/internal/database"
    "scriberr/internal/models"
)

// NoteCreateRequest is the payload for creating a note
type NoteCreateRequest struct {
    StartWordIndex int     `json:"start_word_index" binding:"required,min=0"`
    EndWordIndex   int     `json:"end_word_index" binding:"required,min=0"`
    StartTime      float64 `json:"start_time" binding:"required"`
    EndTime        float64 `json:"end_time" binding:"required"`
    Quote          string  `json:"quote" binding:"required,min=1"`
    Content        string  `json:"content" binding:"required,min=1"`
}

// NoteUpdateRequest updates content of a note
type NoteUpdateRequest struct {
    Content string `json:"content" binding:"required,min=1"`
}

// ListNotes returns all notes for a transcription
func (h *Handler) ListNotes(c *gin.Context) {
    transcriptionID := c.Param("id")
    if transcriptionID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID is required"})
        return
    }

    // Ensure transcription exists
    var job models.TranscriptionJob
    if err := database.DB.Where("id = ?", transcriptionID).First(&job).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Transcription not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transcription"})
        return
    }

    var notes []models.Note
    if err := database.DB.Where("transcription_id = ?", transcriptionID).
        Order("start_time ASC, created_at ASC").Find(&notes).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notes"})
        return
    }

    c.JSON(http.StatusOK, notes)
}

// CreateNote stores a new note for a transcription
func (h *Handler) CreateNote(c *gin.Context) {
    transcriptionID := c.Param("id")
    if transcriptionID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID is required"})
        return
    }

    var req NoteCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.EndWordIndex < req.StartWordIndex {
        c.JSON(http.StatusBadRequest, gin.H{"error": "end_word_index must be >= start_word_index"})
        return
    }
    if req.EndTime < req.StartTime {
        c.JSON(http.StatusBadRequest, gin.H{"error": "end_time must be >= start_time"})
        return
    }

    // Ensure transcription exists
    var job models.TranscriptionJob
    if err := database.DB.Where("id = ?", transcriptionID).First(&job).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Transcription not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transcription"})
        return
    }

    n := models.Note{
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

    if err := database.DB.Create(&n).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create note"})
        return
    }

    c.JSON(http.StatusCreated, n)
}

// GetNote returns a note by ID
func (h *Handler) GetNote(c *gin.Context) {
    noteID := c.Param("note_id")
    var n models.Note
    if err := database.DB.Where("id = ?", noteID).First(&n).Error; err != nil {
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
func (h *Handler) UpdateNote(c *gin.Context) {
    noteID := c.Param("note_id")
    var req NoteUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var n models.Note
    if err := database.DB.Where("id = ?", noteID).First(&n).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch note"})
        return
    }

    n.Content = req.Content
    n.UpdatedAt = time.Now()

    if err := database.DB.Save(&n).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update note"})
        return
    }

    c.JSON(http.StatusOK, n)
}

// DeleteNote removes a note by ID
func (h *Handler) DeleteNote(c *gin.Context) {
    noteID := c.Param("note_id")
    if err := database.DB.Delete(&models.Note{}, "id = ?", noteID).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete note"})
        return
    }
    c.Status(http.StatusNoContent)
}

