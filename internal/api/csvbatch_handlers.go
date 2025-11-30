package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"scriberr/internal/csvbatch"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CSVBatchHandler struct {
	processor *csvbatch.Processor
	handler   *Handler
}

func NewCSVBatchHandler(h *Handler, processor *csvbatch.Processor) *CSVBatchHandler {
	return &CSVBatchHandler{
		processor: processor,
		handler:   h,
	}
}

type CSVBatchUploadRequest struct {
	Name      string  `form:"name"`
	ProfileID *string `form:"profile_id"`
}

type CSVBatchStartRequest struct {
	ProfileID *string `json:"profile_id,omitempty"`
}

// @Summary Upload CSV file for batch processing
// @Description Upload a CSV file containing YouTube URLs for batch transcription
// @Tags csv-batch
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV file with YouTube URLs"
// @Param name formData string false "Batch name"
// @Param profile_id formData string false "Transcription profile ID to use"
// @Success 200 {object} models.CSVBatch
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/csv-batch/upload [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) UploadCSV(c *gin.Context) {
	// Limit file size to 10MB to prevent DoS
	const maxFileSize = 10 << 20 // 10MB
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	if header.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file too large (max 10MB)"})
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext != ".csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be a CSV file"})
		return
	}

	uploadDir := filepath.Join(h.handler.config.UploadDir, "csv-batch", "uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	filename := fmt.Sprintf("%s.csv", uuid.New().String())
	filePath := filepath.Join(uploadDir, filename)

	outFile, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	name := c.PostForm("name")
	if name == "" {
		name = header.Filename
	}

	var profileID *string
	if pid := c.PostForm("profile_id"); pid != "" {
		profileID = &pid
	}

	var params *models.WhisperXParams
	if profileID != nil {
		var profile models.TranscriptionProfile
		if err := database.DB.First(&profile, "id = ?", *profileID).Error; err == nil {
			params = &profile.Parameters
		}
	}

	batch, err := h.processor.CreateBatch(name, filePath, params)
	if err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("CSV batch uploaded", "batch_id", batch.ID, "total_rows", batch.TotalRows)
	c.JSON(http.StatusOK, batch)
}

// @Summary Start batch processing
// @Description Start processing a CSV batch job
// @Tags csv-batch
// @Accept json
// @Produce json
// @Param id path string true "Batch ID"
// @Param request body CSVBatchStartRequest false "Start request options"
// @Success 200 {object} models.CSVBatch
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id}/start [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) StartBatch(c *gin.Context) {
	batchID := c.Param("id")

	var req CSVBatchStartRequest
	// Ignore bind errors since body is optional, but log malformed JSON
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if req.ProfileID != nil {
		var profile models.TranscriptionProfile
		if err := database.DB.First(&profile, "id = ?", *req.ProfileID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
			"profile_id": req.ProfileID,
			"parameters": profile.Parameters,
		})
	}

	if err := h.processor.Start(batchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	batch, err := h.processor.GetStatus(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}
	c.JSON(http.StatusOK, batch)
}

// @Summary Stop batch processing
// @Description Stop a running batch job
// @Tags csv-batch
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {object} models.CSVBatch
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id}/stop [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) StopBatch(c *gin.Context) {
	batchID := c.Param("id")

	if err := h.processor.Stop(batchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	batch, err := h.processor.GetStatus(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}
	c.JSON(http.StatusOK, batch)
}

// @Summary Get batch status
// @Description Get the current status of a batch job
// @Tags csv-batch
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {object} models.CSVBatch
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id}/status [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) GetBatchStatus(c *gin.Context) {
	batchID := c.Param("id")

	batch, err := h.processor.GetStatus(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}

	c.JSON(http.StatusOK, batch)
}

// @Summary Get batch rows
// @Description Get all rows for a batch job
// @Tags csv-batch
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {array} models.CSVBatchRow
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id}/rows [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) GetBatchRows(c *gin.Context) {
	batchID := c.Param("id")

	rows, err := h.processor.GetRows(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}

	c.JSON(http.StatusOK, rows)
}

// @Summary List all batches
// @Description Get a list of all CSV batch jobs
// @Tags csv-batch
// @Produce json
// @Success 200 {array} models.CSVBatch
// @Failure 500 {object} map[string]string
// @Router /api/v1/csv-batch [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) ListBatches(c *gin.Context) {
	batches, err := h.processor.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, batches)
}

// @Summary Delete a batch
// @Description Delete a batch job and all associated files
// @Tags csv-batch
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id} [delete]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) DeleteBatch(c *gin.Context) {
	batchID := c.Param("id")

	if err := h.processor.Delete(batchID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Batch deleted successfully"})
}

// @Summary Download batch output
// @Description Download a specific output JSON file from a batch
// @Tags csv-batch
// @Produce application/json
// @Param id path string true "Batch ID"
// @Param row_id path int true "Row ID"
// @Success 200 {file} file "JSON output file"
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id}/output/{row_id} [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) DownloadOutput(c *gin.Context) {
	batchID := c.Param("id")
	rowIDStr := c.Param("row_id")

	// Validate row_id is a valid integer
	rowID, err := strconv.Atoi(rowIDStr)
	if err != nil || rowID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid row ID"})
		return
	}

	batch, err := h.processor.GetStatus(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}

	var row models.CSVBatchRow
	if err := database.DB.Where("batch_id = ? AND row_num = ?", batchID, rowID).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Row not found"})
		return
	}

	if row.OutputPath == nil || *row.OutputPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Output file not available"})
		return
	}

	// Path traversal protection: ensure output path is within batch directory
	outputPath := filepath.Clean(*row.OutputPath)
	batchDir := filepath.Clean(batch.OutputDir)
	if !strings.HasPrefix(outputPath, batchDir) {
		logger.Error("Path traversal attempt", "output_path", outputPath, "batch_dir", batchDir)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Output file not found"})
		return
	}

	filename := filepath.Base(outputPath)
	// Properly escape filename to prevent header injection
	safeFilename := strings.ReplaceAll(filename, "\"", "\\\"")
	safeFilename = strings.ReplaceAll(safeFilename, "\n", "")
	safeFilename = strings.ReplaceAll(safeFilename, "\r", "")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", safeFilename))
	c.Header("Content-Type", "application/json")
	c.File(outputPath)
}

// @Summary List batch output files
// @Description List all output JSON files from a batch
// @Tags csv-batch
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {object} map[string]interface{} "List of output files"
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id}/outputs [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) ListOutputs(c *gin.Context) {
	batchID := c.Param("id")

	batch, err := h.processor.GetStatus(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}

	if _, err := os.Stat(batch.OutputDir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Output directory not found"})
		return
	}

	files, err := os.ReadDir(batch.OutputDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read output directory"})
		return
	}

	var outputFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			outputFiles = append(outputFiles, f.Name())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"batch_id":    batchID,
		"output_dir":  batch.OutputDir,
		"files":       outputFiles,
		"total_files": len(outputFiles),
	})
}
