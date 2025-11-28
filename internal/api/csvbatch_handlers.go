package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

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
	c.ShouldBindJSON(&req)

	if req.ProfileID != nil {
		var profile models.TranscriptionProfile
		if err := database.DB.First(&profile, "id = ?", *req.ProfileID).Error; err == nil {
			database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
				"profile_id": req.ProfileID,
				"parameters": profile.Parameters,
			})
		}
	}

	if err := h.processor.Start(batchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	batch, _ := h.processor.GetStatus(batchID)
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

	batch, _ := h.processor.GetStatus(batchID)
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
	rowID := c.Param("row_id")

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

	if _, err := os.Stat(*row.OutputPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Output file not found"})
		return
	}

	filename := filepath.Base(*row.OutputPath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")
	c.File(*row.OutputPath)
}

// @Summary Download all batch outputs as ZIP
// @Description Download all output JSON files from a batch as a ZIP archive
// @Tags csv-batch
// @Produce application/zip
// @Param id path string true "Batch ID"
// @Success 200 {file} file "ZIP archive of all outputs"
// @Failure 404 {object} map[string]string
// @Router /api/v1/csv-batch/{id}/download-all [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *CSVBatchHandler) DownloadAllOutputs(c *gin.Context) {
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
