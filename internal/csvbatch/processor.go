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
	"gorm.io/gorm"
)

// Processor handles sequential processing of YouTube URLs from CSV files
type Processor struct {
	config        *config.Config
	mu            sync.Mutex
	activeBatches map[string]context.CancelFunc
}

// New creates a new CSV batch processor
func New(cfg *config.Config) *Processor {
	return &Processor{
		config:        cfg,
		activeBatches: make(map[string]context.CancelFunc),
	}
}

// CreateBatch parses a CSV file and creates a new batch job
func (p *Processor) CreateBatch(name, csvPath string, params *models.WhisperXParams) (*models.CSVBatch, error) {
	urls, err := parseCSV(csvPath)
	if err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no valid YouTube URLs found in CSV")
	}

	batchID := uuid.New().String()
	outputDir := filepath.Join(p.config.UploadDir, "csv-batch", batchID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	batch := &models.CSVBatch{
		ID:        batchID,
		Name:      name,
		Status:    models.BatchPending,
		OutputDir: outputDir,
		TotalRows: len(urls),
	}
	if params != nil {
		batch.Parameters = *params
	}

	// Use transaction to ensure atomicity - either all rows are created or none
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(batch).Error; err != nil {
			return fmt.Errorf("failed to save batch: %w", err)
		}

		for i, url := range urls {
			row := &models.CSVBatchRow{
				BatchID: batchID,
				RowNum:  i + 1,
				URL:     url,
				Status:  models.RowPending,
			}
			if err := tx.Create(row).Error; err != nil {
				return fmt.Errorf("failed to save row %d: %w", i+1, err)
			}
		}
		return nil
	})
	if err != nil {
		os.RemoveAll(outputDir) // Cleanup output dir on failure
		return nil, err
	}

	logger.Info("Batch created", "id", batchID, "rows", len(urls))
	return batch, nil
}

// Start begins processing a batch (can resume from pending/failed/cancelled)
func (p *Processor) Start(batchID string) error {
	var batch models.CSVBatch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		return fmt.Errorf("batch not found: %w", err)
	}

	if batch.Status == models.BatchProcessing {
		return fmt.Errorf("batch is already processing")
	}
	if batch.Status == models.BatchCompleted {
		return fmt.Errorf("batch is already completed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.activeBatches[batchID] = cancel
	p.mu.Unlock()

	go p.process(ctx, batchID)
	return nil
}

// Stop cancels a running batch
func (p *Processor) Stop(batchID string) error {
	p.mu.Lock()
	cancel, exists := p.activeBatches[batchID]
	p.mu.Unlock()

	if !exists {
		return fmt.Errorf("batch is not running")
	}
	cancel()
	return nil
}

// GetStatus returns the current batch status with rows
func (p *Processor) GetStatus(batchID string) (*models.CSVBatch, error) {
	var batch models.CSVBatch
	if err := database.DB.Preload("Rows").First(&batch, "id = ?", batchID).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

// GetRows returns all rows for a batch
func (p *Processor) GetRows(batchID string) ([]models.CSVBatchRow, error) {
	var rows []models.CSVBatchRow
	err := database.DB.Where("batch_id = ?", batchID).Order("row_num ASC").Find(&rows).Error
	return rows, err
}

// List returns all batches
func (p *Processor) List() ([]models.CSVBatch, error) {
	var batches []models.CSVBatch
	err := database.DB.Order("created_at DESC").Find(&batches).Error
	return batches, err
}

// Delete removes a batch and its files
func (p *Processor) Delete(batchID string) error {
	var batch models.CSVBatch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		return err
	}

	p.Stop(batchID)

	if batch.OutputDir != "" {
		os.RemoveAll(batch.OutputDir)
	}
	database.DB.Where("batch_id = ?", batchID).Delete(&models.CSVBatchRow{})
	return database.DB.Delete(&batch).Error
}

// process handles the sequential processing of all pending rows
func (p *Processor) process(ctx context.Context, batchID string) {
	defer func() {
		p.mu.Lock()
		delete(p.activeBatches, batchID)
		p.mu.Unlock()
	}()

	now := time.Now()
	if err := database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
		"status":     models.BatchProcessing,
		"started_at": now,
	}).Error; err != nil {
		logger.Error("Failed to update batch status", "id", batchID, "error", err)
		p.failBatch(batchID, "failed to update status: "+err.Error())
		return
	}

	logger.Info("Processing batch", "id", batchID)

	var rows []models.CSVBatchRow
	if err := database.DB.Where("batch_id = ? AND status = ?", batchID, models.RowPending).
		Order("row_num ASC").Find(&rows).Error; err != nil {
		p.failBatch(batchID, "failed to fetch rows: "+err.Error())
		return
	}

	var batch models.CSVBatch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		p.failBatch(batchID, "failed to fetch batch: "+err.Error())
		return
	}

	for _, row := range rows {
		select {
		case <-ctx.Done():
			logger.Info("Batch cancelled", "id", batchID)
			p.updateBatchStatus(batchID, models.BatchCancelled)
			return
		default:
		}

		if err := database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Update("current_row", row.RowNum).Error; err != nil {
			logger.Error("Failed to update current_row", "batch", batchID, "error", err)
		}

		if p.processRow(ctx, &batch, &row) {
			if err := database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).
				UpdateColumn("success_rows", database.DB.Raw("success_rows + 1")).Error; err != nil {
				logger.Error("Failed to update success_rows", "batch", batchID, "error", err)
			}
		} else {
			if err := database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).
				UpdateColumn("failed_rows", database.DB.Raw("failed_rows + 1")).Error; err != nil {
				logger.Error("Failed to update failed_rows", "batch", batchID, "error", err)
			}
		}
	}

	p.completeBatch(batchID)
}

// processRow handles a single URL: download -> convert -> transcribe -> save JSON
func (p *Processor) processRow(ctx context.Context, batch *models.CSVBatch, row *models.CSVBatchRow) bool {
	start := time.Now()
	if err := database.DB.Model(row).Updates(map[string]interface{}{
		"status":     models.RowProcessing,
		"started_at": start,
	}).Error; err != nil {
		logger.Error("Failed to update row status to processing", "row", row.RowNum, "error", err)
	}

	logger.Info("Processing row", "batch", batch.ID, "row", row.RowNum)

	// Get video title
	title := p.getVideoTitle(ctx, row.URL)
	if title == "" {
		title = fmt.Sprintf("video_%d", row.RowNum)
	}
	filename := sanitizeFilename(title)
	if err := database.DB.Model(row).Updates(map[string]interface{}{
		"title":    title,
		"filename": filename,
	}).Error; err != nil {
		logger.Error("Failed to update row title/filename", "row", row.RowNum, "error", err)
	}

	// Download and extract audio
	audioPath, err := p.downloadAudio(ctx, row.URL, batch.ID, row.RowNum)
	if err != nil {
		return p.failRow(row, "download failed: "+err.Error())
	}
	if err := database.DB.Model(row).Update("audio_path", audioPath).Error; err != nil {
		logger.Error("Failed to update row audio_path", "row", row.RowNum, "error", err)
	}

	// Transcribe
	transcript, err := p.transcribe(ctx, audioPath, batch.Parameters)
	if err != nil {
		return p.failRow(row, "transcription failed: "+err.Error())
	}

	// Save JSON output
	outputPath := filepath.Join(batch.OutputDir, fmt.Sprintf("%d-%s.json", row.RowNum, filename))
	output := map[string]interface{}{
		"row_num":      row.RowNum,
		"url":          row.URL,
		"title":        title,
		"transcript":   json.RawMessage(transcript),
		"processed_at": time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return p.failRow(row, "failed to marshal JSON: "+err.Error())
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return p.failRow(row, "failed to write output: "+err.Error())
	}

	// Mark complete
	now := time.Now()
	if err := database.DB.Model(row).Updates(map[string]interface{}{
		"status":       models.RowCompleted,
		"output_path":  outputPath,
		"completed_at": now,
	}).Error; err != nil {
		logger.Error("Failed to update row completion", "row", row.RowNum, "error", err)
	}

	logger.Info("Row completed", "batch", batch.ID, "row", row.RowNum, "duration", time.Since(start))
	return true
}

func (p *Processor) getVideoTitle(ctx context.Context, url string) string {
	cmd := exec.CommandContext(ctx, p.config.UVPath, "run", "--native-tls",
		"--project", p.config.WhisperXEnv,
		"python", "-m", "yt_dlp", "--get-title", url)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (p *Processor) downloadAudio(ctx context.Context, url, batchID string, rowNum int) (string, error) {
	tempDir := filepath.Join(p.config.UploadDir, "csv-batch", batchID, "audio")
	os.MkdirAll(tempDir, 0755)

	outputPath := filepath.Join(tempDir, fmt.Sprintf("row_%d.mp3", rowNum))

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
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}

	// Find actual file (yt-dlp may change extension)
	pattern := filepath.Join(tempDir, fmt.Sprintf("row_%d.*", rowNum))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob error: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("audio file not found after download")
	}
	return matches[0], nil
}

func (p *Processor) transcribe(ctx context.Context, audioPath string, params models.WhisperXParams) (string, error) {
	model := params.Model
	if model == "" {
		model = "small"
	}
	device := params.Device
	if device == "" {
		device = "cpu"
	}

	args := []string{
		"run", "--native-tls",
		"--project", p.config.WhisperXEnv,
		"whisperx", audioPath,
		"--model", model,
		"--device", device,
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
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}

	baseName := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	jsonPath := filepath.Join(filepath.Dir(audioPath), baseName+".json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read transcript: %w", err)
	}
	return string(data), nil
}

func (p *Processor) failRow(row *models.CSVBatchRow, msg string) bool {
	now := time.Now()
	if err := database.DB.Model(row).Updates(map[string]interface{}{
		"status":        models.RowFailed,
		"error_message": msg,
		"completed_at":  now,
	}).Error; err != nil {
		logger.Error("Failed to update row failure status", "row", row.RowNum, "db_error", err)
	}
	logger.Error("Row failed", "row", row.RowNum, "error", msg)
	return false
}

func (p *Processor) failBatch(batchID, msg string) {
	now := time.Now()
	if err := database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
		"status":        models.BatchFailed,
		"error_message": msg,
		"completed_at":  now,
	}).Error; err != nil {
		logger.Error("Failed to update batch failure status", "batch", batchID, "db_error", err)
	}
	logger.Error("Batch failed", "id", batchID, "error", msg)
}

func (p *Processor) completeBatch(batchID string) {
	now := time.Now()
	if err := database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
		"status":       models.BatchCompleted,
		"completed_at": now,
	}).Error; err != nil {
		logger.Error("Failed to update batch completion", "batch", batchID, "error", err)
	}
	logger.Info("Batch completed", "id", batchID)
}

func (p *Processor) updateBatchStatus(batchID string, status models.BatchStatus) {
	if err := database.DB.Model(&models.CSVBatch{}).Where("id = ?", batchID).Update("status", status).Error; err != nil {
		logger.Error("Failed to update batch status", "batch", batchID, "status", status, "error", err)
	}
}

// parseCSV extracts YouTube URLs from a CSV file
func parseCSV(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var urls []string
	isFirstRow := true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV parse error: %w", err)
		}

		// Skip header row if it looks like a header
		if isFirstRow {
			isFirstRow = false
			isHeader := false
			for _, field := range record {
				if strings.EqualFold(strings.TrimSpace(field), "url") {
					isHeader = true
					break
				}
			}
			if isHeader {
				continue
			}
		}

		// Find YouTube URL in row
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

func isYouTubeURL(url string) bool {
	patterns := []string{
		`^https?://(www\.)?youtube\.com/watch\?v=[\w-]+`,
		`^https?://(www\.)?youtu\.be/[\w-]+`,
		`^https?://(www\.)?youtube\.com/shorts/[\w-]+`,
	}
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p, url); matched {
			return true
		}
	}
	return false
}

func sanitizeFilename(name string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	name = re.ReplaceAllString(name, "_")
	if len(name) > 80 {
		name = name[:80]
	}
	name = strings.Trim(name, " .")
	if name == "" {
		return "untitled"
	}
	return name
}
