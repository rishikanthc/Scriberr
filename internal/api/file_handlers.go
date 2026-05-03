package api

import (
	"errors"
	"net/http"
	"strings"

	filesdomain "scriberr/internal/files"
	"scriberr/internal/mediaimport"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) uploadFile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	result, ok := h.storeUploadedFile(c, userID)
	if !ok {
		return
	}
	response := fileResponse(result.Job, result.MimeType, result.Kind)
	if result.Job.Status == models.StatusProcessing {
		c.JSON(http.StatusAccepted, response)
		return
	}
	c.JSON(http.StatusCreated, response)
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
	if !mediaimport.ValidYouTubeURL(rawURL) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "url is invalid", stringPtr("url"))
		return
	}
	if h.mediaImport == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "media import service is not configured", nil)
		return
	}
	job, err := h.mediaImport.ImportYouTube(c.Request.Context(), mediaimport.ImportYouTubeCommand{
		UserID: userID,
		URL:    rawURL,
		Title:  strings.TrimSpace(req.Title),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create youtube import", nil)
		return
	}
	c.JSON(http.StatusAccepted, fileResponse(job, "", "youtube"))
}

func (h *Handler) storeUploadedFile(c *gin.Context, userID uint) (*filesdomain.UploadResult, bool) {
	limit := uploadSizeLimit(h)
	if c.Request.ContentLength > limit {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "upload is too large", stringPtr("file"))
		return nil, false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)

	header, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "upload is too large", stringPtr("file"))
			return nil, false
		}
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "file is required", stringPtr("file"))
		return nil, false
	}
	source, err := header.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "file could not be read", stringPtr("file"))
		return nil, false
	}
	defer source.Close()

	if h.files == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "file service is not configured", nil)
		return nil, false
	}
	result, err := h.files.Upload(c.Request.Context(), filesdomain.UploadCommand{
		UserID:      userID,
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Title:       c.PostForm("title"),
		Body:        source,
	})
	if errors.Is(err, filesdomain.ErrUnsupportedMediaType) {
		writeError(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "unsupported media type", stringPtr("file"))
		return nil, false
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not store file", nil)
		return nil, false
	}
	return result, true
}

func (h *Handler) listFiles(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	opts, ok := parseListQuery(c, allowedResourceSorts())
	if !ok {
		return
	}
	kind := strings.TrimSpace(c.Query("kind"))
	if kind != "" && kind != "audio" && kind != "video" && kind != "youtube" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "kind is invalid", stringPtr("kind"))
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && !validFileStatus(status) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "status is invalid", stringPtr("status"))
		return
	}
	if h.files == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "file service is not configured", nil)
		return
	}
	jobs, err := h.files.List(c.Request.Context(), userID, fileListOptions(kind, status, opts))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list files", nil)
		return
	}
	jobs, nextCursor := trimListPage(jobs, opts)
	items := make([]FileResponse, 0, len(jobs))
	for i := range jobs {
		mimeType := mediaType("", jobs[i].SourceFileName)
		items = append(items, fileResponse(&jobs[i], mimeType, fileKind(mimeType)))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nextCursor})
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
	userID, id, ok := h.fileRequestIdentity(c)
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
	job, err := h.files.UpdateTitle(c.Request.Context(), userID, id, title)
	if err != nil {
		if errors.Is(err, filesdomain.ErrNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update file", nil)
		return
	}
	mimeType := mediaType("", job.SourceFileName)
	c.JSON(http.StatusOK, fileResponse(job, mimeType, fileKind(mimeType)))
}

func (h *Handler) deleteFile(c *gin.Context) {
	userID, id, ok := h.fileRequestIdentity(c)
	if !ok {
		return
	}
	if err := h.files.Delete(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, filesdomain.ErrNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete file", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) streamFileAudio(c *gin.Context) {
	userID, id, ok := h.fileRequestIdentity(c)
	if !ok {
		return
	}
	file, job, err := h.files.OpenAudio(c.Request.Context(), userID, id)
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
	userID, id, ok := h.fileRequestIdentity(c)
	if !ok {
		return nil, false
	}
	job, err := h.files.Get(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return nil, false
	}
	return job, true
}

func (h *Handler) fileRequestIdentity(c *gin.Context) (uint, string, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return 0, "", false
	}
	id := strings.TrimPrefix(c.Param("id"), "file_")
	if id == "" || id == c.Param("id") {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return 0, "", false
	}
	if h.files == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "file service is not configured", nil)
		return 0, "", false
	}
	return userID, id, true
}

func fileListOptions(kind, status string, opts *listQuery) filesdomain.ListOptions {
	var cursor *filesdomain.ListCursor
	if opts.Cursor != nil {
		cursor = &filesdomain.ListCursor{Value: opts.Cursor.Value, ID: opts.Cursor.ID}
	}
	return filesdomain.ListOptions{
		Kind:         kind,
		Status:       status,
		Query:        opts.Query,
		UpdatedAfter: opts.UpdatedAfter,
		Limit:        opts.Limit,
		SortColumn:   opts.SortColumn,
		SortDesc:     opts.SortDesc,
		Cursor:       cursor,
	}
}
