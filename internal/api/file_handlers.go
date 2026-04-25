package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) uploadFile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	job, mimeType, kind, ok := h.storeUploadedFile(c, userID)
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, fileResponse(job, mimeType, kind))
}
func (h *Handler) importYouTube(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req importYouTubeRequest
	if !bindJSON(c, &req) {
		return
	}
	rawURL := strings.TrimSpace(req.URL)
	if !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(rawURL, "http://") {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "url is invalid", stringPtr("url"))
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "YouTube import"
	}
	job := models.TranscriptionJob{
		ID:             randomHex(16),
		UserID:         userID,
		Title:          &title,
		Status:         models.StatusProcessing,
		AudioPath:      "",
		SourceFileName: "youtube:" + rawURL,
	}
	if err := database.DB.Create(&job).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create youtube import", nil)
		return
	}
	c.JSON(http.StatusAccepted, fileResponse(&job, "", "youtube"))
}
func (h *Handler) storeUploadedFile(c *gin.Context, userID uint) (*models.TranscriptionJob, string, string, bool) {
	header, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "file is required", stringPtr("file"))
		return nil, "", "", false
	}
	source, err := header.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "file could not be read", stringPtr("file"))
		return nil, "", "", false
	}
	defer source.Close()

	mimeType := mediaType(header.Header.Get("Content-Type"), header.Filename)
	kind := fileKind(mimeType)
	if kind == "" {
		writeError(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "unsupported media type", stringPtr("file"))
		return nil, "", "", false
	}

	uploadDir := h.config.UploadDir
	if uploadDir == "" {
		uploadDir = filepath.Join(os.TempDir(), "scriberr-uploads")
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not prepare file storage", nil)
		return nil, "", "", false
	}

	jobID := randomHex(16)
	filename := safeFilename(header.Filename)
	if filename == "" {
		filename = jobID
	}
	storedName := jobID + filepath.Ext(filename)
	storagePath := filepath.Join(uploadDir, storedName)
	destination, err := os.OpenFile(storagePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not store file", nil)
		return nil, "", "", false
	}
	if _, err := io.Copy(destination, source); err != nil {
		_ = destination.Close()
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not store file", nil)
		return nil, "", "", false
	}
	if err := destination.Close(); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not store file", nil)
		return nil, "", "", false
	}

	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		title = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	job := models.TranscriptionJob{
		ID:             jobID,
		UserID:         userID,
		Title:          &title,
		Status:         models.StatusUploaded,
		AudioPath:      storagePath,
		SourceFileName: filename,
	}
	if err := database.DB.Create(&job).Error; err != nil {
		_ = os.Remove(storagePath)
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create file record", nil)
		return nil, "", "", false
	}
	return &job, mimeType, kind, true
}
func (h *Handler) listFiles(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var jobs []models.TranscriptionJob
	if err := database.DB.Where("user_id = ? AND source_file_hash IS NULL", userID).Order("created_at DESC").Find(&jobs).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list files", nil)
		return
	}
	items := make([]gin.H, 0, len(jobs))
	for i := range jobs {
		mimeType := mediaType("", jobs[i].SourceFileName)
		items = append(items, fileResponse(&jobs[i], mimeType, fileKind(mimeType)))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}
func (h *Handler) getFile(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	mimeType := mediaType("", job.SourceFileName)
	c.JSON(http.StatusOK, fileResponse(job, mimeType, fileKind(mimeType)))
}
func (h *Handler) updateFile(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	var req updateFileRequest
	if !bindJSON(c, &req) {
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "title is required", stringPtr("title"))
		return
	}
	if err := database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("title", title).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update file", nil)
		return
	}
	job.Title = &title
	mimeType := mediaType("", job.SourceFileName)
	c.JSON(http.StatusOK, fileResponse(job, mimeType, fileKind(mimeType)))
}
func (h *Handler) deleteFile(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	if err := database.DB.Delete(&models.TranscriptionJob{}, "id = ?", job.ID).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete file", nil)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *Handler) streamFileAudio(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	file, err := os.Open(job.AudioPath)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file audio not found", nil)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file audio not found", nil)
		return
	}
	mimeType := mediaType("", job.SourceFileName)
	c.Header("Content-Type", mimeType)
	c.Header("Accept-Ranges", "bytes")
	http.ServeContent(c.Writer, c.Request, job.SourceFileName, stat.ModTime(), file)
}
func (h *Handler) fileByPublicID(c *gin.Context) (*models.TranscriptionJob, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return nil, false
	}
	id := strings.TrimPrefix(c.Param("id"), "file_")
	if id == "" || id == c.Param("id") {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return nil, false
	}
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ? AND user_id = ? AND source_file_hash IS NULL", id, userID).First(&job).Error; err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return nil, false
	}
	return &job, true
}
