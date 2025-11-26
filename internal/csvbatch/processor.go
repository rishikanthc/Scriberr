package csvbatch

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/pkg/logger"

	"github.com/google/uuid"
)

// Processor handles CSV batch processing for YouTube URLs
type Processor struct {
	config         *config.Config
	transcribeFunc func(ctx context.Context, audioPath string, params models.WhisperXParams) (string, error)
	mu             sync.Mutex
	activeBatches  map[string]context.CancelFunc
}

// NewProcessor creates a new CSV batch processor
func NewProcessor(cfg *config.Config, transcribeFn func(ctx context.Context, audioPath string, params models.WhisperXParams) (string, error)) *Processor {
	return &Processor{
		config:         cfg,
		transcribeFunc: transcribeFn,
		activeBatches:  make(map[string]context.CancelFunc),
	}
}

// ParseCSV parses a CSV file and extracts YouTube URLs
func (p *Processor) ParseCSV(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable fields
	reader.TrimLeadingSpace = true

	var urls []string
	rowNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV at row %d: %w", rowNum+1, err)
		}

		rowNum++

		// Skip header row if it contains "url" (case-insensitive)
		if rowNum == 1 {
			hasHeader := false
			for _, field := range record {
				if strings.EqualFold(strings.TrimSpace(field), "url") {
					hasHeader = true
					break
				}
			}
			if hasHeader {
				continue
			}
		}

		// Find YouTube URL in the row
		for _, field := range record {
			field = strings.TrimSpace(field)
			if isYouTubeURL(field) {
				urls = append(urls, field)
				break
			}
		}
	}

	return urls, nil
}

// isYouTubeURL checks if a string is a valid YouTube URL
func isYouTubeURL(url string) bool {
	patterns := []string{
		`^https?://(www\.)?youtube\.com/watch\?v=[\w-]+`,
		`^https?://(www\.)?youtu\.be/[\w-]+`,
		`^https?://(www\.)?youtube\.com/shorts/[\w-]+`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, url)
		if matched {
			return true
		}
	}
	return false
}

// CreateBatch creates a new batch job from a CSV file
func (p *Processor) CreateBatch(name, csvFilePath string, params *models.WhisperXParams, profileID *string) (*models.CSVBatch, error) {
	// Parse CSV to get URLs
	urls, err := p.ParseCSV(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no valid YouTube URLs found in CSV")
	}

	// Create output directory
	batchID := uuid.New().String()
	outputDir := filepath.Join(p.config.UploadDir, "csv-batch", batchID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create batch record
	batch := &models.CSVBatch{
		ID:          batchID,
		Name:        name,
		Status:      models.CSVBatchStatusPending,
		TotalRows:   len(urls),
		CurrentRow:  0,
		SuccessRows: 0,
		FailedRows:  0,
		OutputDir:   outputDir,
		CSVFilePath: csvFilePath,
		ProfileID:   profileID,
	}

	// Apply parameters if provided
	if params != nil {
		batch.Parameters = *params
	}

	// Save batch to database
	if err := database.DB.Create(batch).Error; err != nil {
		return nil, fmt.Errorf("failed to save batch: %w", err)
	}

	// Create row records
	for i, url := range urls {
		row := &models.CSVBatchRow{
			BatchID: batchID,
			RowID:   i + 1, // 1-indexed
			URL:     url,
			Status:  models.CSVRowStatusPending,
		}
		if err := database.DB.Create(row).Error; err != nil {
			return nil, fmt.Errorf("failed to save batch row %d: %w", i+1, err)
		}
	}

	logger.Info("CSV batch created", "batch_id", batchID, "total_rows", len(urls))
	return batch, nil
}

// StartBatch starts processing a batch
func (p *Processor) StartBatch(batchID string) error {
	// Check if batch exists and is in pending state
	var batch models.CSVBatch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		return fmt.Errorf("batch not found: %w", err)
	}

	if batch.Status != models.CSVBatchStatusPending && batch.Status != models.CSVBatchStatusFailed {
		return fmt.Errorf("batch is not in a startable state: %s", batch.Status)
	}

	// Create context for batch processing
	ctx, cancel := context.WithCancel(context.Background())

	p.mu.Lock()
	p.activeBatches[batchID] = cancel
	p.mu.Unlock()

	// Start processing in background
	go p.processBatch(ctx, batchID)

	return nil
}

// StopBatch stops a running batch
func (p *Processor) StopBatch(batchID string) error {
	p.mu.Lock()
	cancel, exists := p.activeBatches[batchID]
	p.mu.Unlock()

	if !exists {
		return fmt.Errorf("batch is not running")
	}

	cancel()
	return nil
}

// processBatch processes all rows in a batch sequentially
func (p *Processor) processBatch(ctx context.Context, batchID string) {
	defer func() {
		p.mu.Lock()
		delete(p.activeBatches, batchID)
		p.mu.Unlock()
	}()

	// Update batch status
	now := time.Now()
	database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
		"status":     models.CSVBatchStatusProcessing,
		"started_at": now,
	})

	logger.Info("Starting CSV batch processing", "batch_id", batchID)

	// Get all pending rows
	var rows []models.CSVBatchRow
	if err := database.DB.Where("batch_id = ? AND status = ?", batchID, models.CSVRowStatusPending).Order("row_id ASC").Find(&rows).Error; err != nil {
		logger.Error("Failed to fetch batch rows", "batch_id", batchID, "error", err)
		p.updateBatchStatus(batchID, models.CSVBatchStatusFailed, "Failed to fetch batch rows: "+err.Error())
		return
	}

	// Process each row sequentially
	for _, row := range rows {
		// Check for cancellation
		select {
		case <-ctx.Done():
			logger.Info("Batch processing cancelled", "batch_id", batchID)
			p.updateBatchStatus(batchID, models.CSVBatchStatusCancelled, "Processing cancelled by user")
			return
		default:
		}

		// Update current row
		database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Update("current_row", row.RowID)

		// Process the row
		success := p.processRow(ctx, batchID, &row)

		// Update batch counters
		if success {
			database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).UpdateColumn("success_rows", database.DB.Raw("success_rows + 1"))
		} else {
			database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).UpdateColumn("failed_rows", database.DB.Raw("failed_rows + 1"))
		}
	}

	// Mark batch as completed
	completedAt := time.Now()
	database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
		"status":       models.CSVBatchStatusCompleted,
		"completed_at": completedAt,
	})

	logger.Info("CSV batch processing completed", "batch_id", batchID)
}

// processRow processes a single row from the batch
func (p *Processor) processRow(ctx context.Context, batchID string, row *models.CSVBatchRow) bool {
	rowStart := time.Now()

	// Update row status
	database.DB.Model(row).Updates(map[string]interface{}{
		"status":     models.CSVRowStatusProcessing,
		"started_at": rowStart,
	})

	logger.Info("Processing CSV row", "batch_id", batchID, "row_id", row.RowID, "url", row.URL)

	// Step 1: Get video title
	title, err := p.getVideoTitle(ctx, row.URL)
	if err != nil {
		logger.Warn("Failed to get video title, using fallback", "row_id", row.RowID, "error", err)
		title = fmt.Sprintf("Video_%d", row.RowID)
	}

	// Sanitize filename
	videoFilename := sanitizeFilename(title)
	database.DB.Model(row).Updates(map[string]interface{}{
		"video_title":    title,
		"video_filename": videoFilename,
	})

	// Step 2: Download video and extract audio
	audioPath, err := p.downloadAndConvert(ctx, row.URL, batchID, row.RowID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to download/convert: %v", err)
		p.markRowFailed(row, errMsg)
		return false
	}
	database.DB.Model(row).Update("audio_file_path", audioPath)

	// Step 3: Transcribe audio
	var params models.WhisperXParams

	// Get batch parameters
	var batch models.CSVBatch
	database.DB.First(&batch, "id = ?", batchID)
	params = batch.Parameters

	transcript, err := p.transcribeAudio(ctx, audioPath, params)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to transcribe: %v", err)
		p.markRowFailed(row, errMsg)
		return false
	}

	// Step 4: Save JSON output
	outputPath := filepath.Join(batch.OutputDir, fmt.Sprintf("%d-%s.json", row.RowID, videoFilename))

	outputData := map[string]interface{}{
		"row_id":         row.RowID,
		"url":            row.URL,
		"video_title":    title,
		"video_filename": videoFilename,
		"transcript":     transcript,
		"processed_at":   time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal JSON: %v", err)
		p.markRowFailed(row, errMsg)
		return false
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		errMsg := fmt.Sprintf("Failed to write JSON file: %v", err)
		p.markRowFailed(row, errMsg)
		return false
	}

	// Step 5: Mark row as completed
	completedAt := time.Now()
	database.DB.Model(row).Updates(map[string]interface{}{
		"status":           models.CSVRowStatusCompleted,
		"output_file_path": outputPath,
		"completed_at":     completedAt,
	})

	logger.Info("CSV row processed successfully",
		"batch_id", batchID,
		"row_id", row.RowID,
		"output", outputPath,
		"duration", time.Since(rowStart))

	return true
}

// getVideoTitle retrieves the video title using yt-dlp
func (p *Processor) getVideoTitle(ctx context.Context, url string) (string, error) {
	cmd := exec.CommandContext(ctx, p.config.UVPath, "run", "--native-tls",
		"--project", p.config.WhisperXEnv,
		"python", "-m", "yt_dlp", "--get-title", url)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// downloadAndConvert downloads video and converts to audio
func (p *Processor) downloadAndConvert(ctx context.Context, url, batchID string, rowID int) (string, error) {
	// Create temp directory for this row
	tempDir := filepath.Join(p.config.UploadDir, "csv-batch", batchID, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate unique filename
	audioFilename := fmt.Sprintf("row_%d_%s.mp3", rowID, uuid.New().String()[:8])
	outputPath := filepath.Join(tempDir, audioFilename)

	// Download and extract audio using yt-dlp
	cmd := exec.CommandContext(ctx, p.config.UVPath, "run", "--native-tls",
		"--project", p.config.WhisperXEnv,
		"python", "-m", "yt_dlp",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--output", outputPath,
		"--no-playlist",
		url)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %v, stderr: %s", err, stderr.String())
	}

	// Find the actual output file (yt-dlp might change extension)
	pattern := filepath.Join(tempDir, fmt.Sprintf("row_%d_*", rowID))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("downloaded audio file not found")
	}

	return matches[0], nil
}

// transcribeAudio transcribes an audio file
func (p *Processor) transcribeAudio(ctx context.Context, audioPath string, params models.WhisperXParams) (string, error) {
	if p.transcribeFunc != nil {
		return p.transcribeFunc(ctx, audioPath, params)
	}

	// Fallback: Use direct whisperx call
	return p.directTranscribe(ctx, audioPath, params)
}

// directTranscribe performs transcription directly using whisperx
func (p *Processor) directTranscribe(ctx context.Context, audioPath string, params models.WhisperXParams) (string, error) {
	// Build whisperx command
	args := []string{
		"run", "--native-tls",
		"--project", p.config.WhisperXEnv,
		"whisperx",
		audioPath,
		"--model", params.Model,
		"--device", params.Device,
		"--output_format", "json",
		"--output_dir", filepath.Dir(audioPath),
	}

	if params.Language != nil && *params.Language != "" {
		args = append(args, "--language", *params.Language)
	}

	if params.Diarize && params.HfToken != nil && *params.HfToken != "" {
		args = append(args, "--diarize", "--hf_token", *params.HfToken)
	}

	cmd := exec.CommandContext(ctx, p.config.UVPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisperx failed: %v, stderr: %s", err, stderr.String())
	}

	// Read the output JSON
	baseName := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	jsonPath := filepath.Join(filepath.Dir(audioPath), baseName+".json")

	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription output: %w", err)
	}

	return string(jsonData), nil
}

// markRowFailed marks a row as failed with an error message
func (p *Processor) markRowFailed(row *models.CSVBatchRow, errMsg string) {
	completedAt := time.Now()
	database.DB.Model(row).Updates(map[string]interface{}{
		"status":        models.CSVRowStatusFailed,
		"error_message": errMsg,
		"completed_at":  completedAt,
	})
	logger.Error("CSV row processing failed", "row_id", row.RowID, "error", errMsg)
}

// updateBatchStatus updates batch status with optional error message
func (p *Processor) updateBatchStatus(batchID string, status models.CSVBatchStatus, errMsg string) {
	updates := map[string]interface{}{
		"status": status,
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	if status == models.CSVBatchStatusCompleted || status == models.CSVBatchStatusFailed || status == models.CSVBatchStatusCancelled {
		updates["completed_at"] = time.Now()
	}
	database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(updates)
}

// GetBatchStatus retrieves the current status of a batch
func (p *Processor) GetBatchStatus(batchID string) (*models.CSVBatch, error) {
	var batch models.CSVBatch
	if err := database.DB.Preload("Rows").First(&batch, "id = ?", batchID).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

// GetBatchRows retrieves all rows for a batch
func (p *Processor) GetBatchRows(batchID string) ([]models.CSVBatchRow, error) {
	var rows []models.CSVBatchRow
	if err := database.DB.Where("batch_id = ?", batchID).Order("row_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListBatches returns all batches
func (p *Processor) ListBatches() ([]models.CSVBatch, error) {
	var batches []models.CSVBatch
	if err := database.DB.Order("created_at DESC").Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// DeleteBatch deletes a batch and its associated files
func (p *Processor) DeleteBatch(batchID string) error {
	var batch models.CSVBatch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		return err
	}

	// Stop if running
	p.StopBatch(batchID)

	// Delete output directory
	if batch.OutputDir != "" {
		os.RemoveAll(batch.OutputDir)
	}

	// Delete CSV file
	if batch.CSVFilePath != "" {
		os.Remove(batch.CSVFilePath)
	}

	// Delete rows
	if err := database.DB.Where("batch_id = ?", batchID).Delete(&models.CSVBatchRow{}).Error; err != nil {
		return err
	}

	// Delete batch
	return database.DB.Delete(&batch).Error
}

// sanitizeFilename removes or replaces invalid characters in filenames
func sanitizeFilename(name string) string {
	// Replace invalid characters
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	name = invalidChars.ReplaceAllString(name, "_")

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	// Trim spaces and dots
	name = strings.Trim(name, " .")

	if name == "" {
		name = "untitled"
	}

	return name
}
