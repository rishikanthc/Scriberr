package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/csvbatch"
	"scriberr/internal/database"
	"scriberr/internal/models"
)

const (
	version     = "1.0.0"
	sessionFile = ".csvbatch_session.json"
)

// Session represents a resumable processing session
type Session struct {
	BatchID     string    `json:"batch_id"`
	CSVFile     string    `json:"csv_file"`
	OutputDir   string    `json:"output_dir"`
	ProfileID   string    `json:"profile_id,omitempty"`
	Model       string    `json:"model"`
	Device      string    `json:"device"`
	Language    string    `json:"language,omitempty"`
	Diarize     bool      `json:"diarize"`
	HFToken     string    `json:"hf_token,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	LastUpdated time.Time `json:"last_updated"`
}

// Logger handles logging to file and console
type Logger struct {
	file    *os.File
	verbose bool
}

func newLogger(logFile string, verbose bool) (*Logger, error) {
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f, verbose: verbose}, nil
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)

	// Always write to log file
	if l.file != nil {
		l.file.WriteString(logLine)
	}

	// Print to console based on verbosity
	if l.verbose || level == "ERROR" || level == "SUCCESS" || level == "INFO" {
		switch level {
		case "ERROR":
			fmt.Printf("\033[31m%s\033[0m", logLine) // Red
		case "SUCCESS":
			fmt.Printf("\033[32m%s\033[0m", logLine) // Green
		case "WARN":
			fmt.Printf("\033[33m%s\033[0m", logLine) // Yellow
		case "DEBUG":
			if l.verbose {
				fmt.Printf("\033[90m%s\033[0m", logLine) // Gray
			}
		default:
			fmt.Print(logLine)
		}
	}
}

func (l *Logger) Info(format string, args ...interface{})    { l.log("INFO", format, args...) }
func (l *Logger) Error(format string, args ...interface{})   { l.log("ERROR", format, args...) }
func (l *Logger) Success(format string, args ...interface{}) { l.log("SUCCESS", format, args...) }
func (l *Logger) Warn(format string, args ...interface{})    { l.log("WARN", format, args...) }
func (l *Logger) Debug(format string, args ...interface{})   { l.log("DEBUG", format, args...) }

// ProgressBar displays live progress
type ProgressBar struct {
	total     int
	current   int
	width     int
	startTime time.Time
}

func newProgressBar(total int) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     50,
		startTime: time.Now(),
	}
}

func (p *ProgressBar) Update(current int, status string) {
	p.current = current
	percent := float64(current) / float64(p.total) * 100
	filled := int(float64(p.width) * float64(current) / float64(p.total))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	elapsed := time.Since(p.startTime)
	var eta string
	if current > 0 {
		remaining := time.Duration(float64(elapsed) / float64(current) * float64(p.total-current))
		eta = fmt.Sprintf("ETA: %s", formatDuration(remaining))
	} else {
		eta = "ETA: calculating..."
	}

	// Clear line and print progress
	fmt.Printf("\r\033[K[%s] %.1f%% (%d/%d) %s | %s",
		bar, percent, current, p.total, status, eta)
}

func (p *ProgressBar) Complete() {
	elapsed := time.Since(p.startTime)
	fmt.Printf("\r\033[K[%s] 100%% (%d/%d) Complete! | Total time: %s\n",
		strings.Repeat("█", p.width), p.total, p.total, formatDuration(elapsed))
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func main() {
	// Define command-line flags
	csvFile := flag.String("csv", "", "Path to CSV file with YouTube URLs")
	outputDir := flag.String("output", "", "Output directory for JSON files")
	profileID := flag.String("profile", "", "Transcription profile ID to use")
	model := flag.String("model", "small", "Whisper model (tiny, base, small, medium, large)")
	device := flag.String("device", "cpu", "Device to use (cpu, cuda, mps)")
	language := flag.String("language", "", "Language code (auto-detect if empty)")
	diarize := flag.Bool("diarize", false, "Enable speaker diarization")
	hfToken := flag.String("hf-token", "", "Hugging Face token for diarization")
	resume := flag.String("resume", "", "Resume a previous session by batch ID")
	listSessions := flag.Bool("list-sessions", false, "List resumable sessions")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	logFile := flag.String("log", "csvbatch.log", "Log file path")
	showHelp := flag.Bool("help", false, "Show help")
	showVersion := flag.Bool("version", false, "Show version")
	init := flag.Bool("init", false, "Run first-time setup")

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("Scriberr CSV Batch Processor v%s\n", version)
		os.Exit(0)
	}

	// Show help
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Initialize logger
	logger, err := newLogger(*logFile, *verbose)
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Scriberr CSV Batch Processor v%s starting...", version)

	// First-time init
	if *init {
		runFirstTimeSetup(logger)
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		logger.Error("Failed to initialize database: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create processor
	processor := csvbatch.NewProcessor(cfg, nil)

	// List sessions
	if *listSessions {
		listResumableSessions(processor, logger)
		os.Exit(0)
	}

	// Resume mode
	if *resume != "" {
		resumeSession(processor, *resume, logger, *verbose)
		os.Exit(0)
	}

	// Interactive mode - prompt for missing required inputs
	reader := bufio.NewReader(os.Stdin)

	// Get CSV file
	if *csvFile == "" {
		*csvFile = promptInput(reader, "Enter path to CSV file with YouTube URLs", "")
		if *csvFile == "" {
			logger.Error("CSV file is required")
			os.Exit(1)
		}
	}

	// Validate CSV file exists
	if _, err := os.Stat(*csvFile); os.IsNotExist(err) {
		logger.Error("CSV file not found: %s", *csvFile)
		os.Exit(1)
	}

	// Get output directory
	if *outputDir == "" {
		defaultOutput := filepath.Join(".", "csv_output")
		*outputDir = promptInput(reader, "Enter output directory", defaultOutput)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		logger.Error("Failed to create output directory: %v", err)
		os.Exit(1)
	}

	// Interactive model selection
	if *model == "small" {
		fmt.Println("\nAvailable Whisper models:")
		fmt.Println("  1. tiny    - Fastest, least accurate")
		fmt.Println("  2. base    - Fast, moderate accuracy")
		fmt.Println("  3. small   - Balanced speed/accuracy (default)")
		fmt.Println("  4. medium  - Slower, high accuracy")
		fmt.Println("  5. large   - Slowest, highest accuracy")
		modelChoice := promptInput(reader, "Select model (1-5 or name)", "3")
		switch modelChoice {
		case "1", "tiny":
			*model = "tiny"
		case "2", "base":
			*model = "base"
		case "3", "small":
			*model = "small"
		case "4", "medium":
			*model = "medium"
		case "5", "large":
			*model = "large"
		}
	}

	// Interactive device selection
	fmt.Println("\nAvailable devices:")
	fmt.Println("  1. cpu   - CPU processing (default)")
	fmt.Println("  2. cuda  - NVIDIA GPU")
	fmt.Println("  3. mps   - Apple Silicon GPU")
	deviceChoice := promptInput(reader, "Select device (1-3 or name)", "1")
	switch deviceChoice {
	case "1", "cpu":
		*device = "cpu"
	case "2", "cuda":
		*device = "cuda"
	case "3", "mps":
		*device = "mps"
	}

	// Diarization prompt
	if !*diarize {
		diarizeChoice := promptInput(reader, "Enable speaker diarization? (y/N)", "n")
		*diarize = strings.ToLower(diarizeChoice) == "y" || strings.ToLower(diarizeChoice) == "yes"
	}

	if *diarize && *hfToken == "" {
		*hfToken = promptInput(reader, "Enter Hugging Face token for diarization", "")
		if *hfToken == "" {
			logger.Warn("Diarization requires a Hugging Face token - disabling")
			*diarize = false
		}
	}

	// Language prompt
	if *language == "" {
		*language = promptInput(reader, "Enter language code (or press Enter for auto-detect)", "")
	}

	// Build parameters
	params := models.WhisperXParams{
		Model:   *model,
		Device:  *device,
		Diarize: *diarize,
	}
	if *language != "" {
		params.Language = language
	}
	if *hfToken != "" {
		params.HfToken = hfToken
	}

	// Display configuration
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Configuration Summary")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  CSV File:    %s\n", *csvFile)
	fmt.Printf("  Output Dir:  %s\n", *outputDir)
	fmt.Printf("  Model:       %s\n", *model)
	fmt.Printf("  Device:      %s\n", *device)
	fmt.Printf("  Language:    %s\n", ifEmpty(*language, "auto-detect"))
	fmt.Printf("  Diarization: %v\n", *diarize)
	fmt.Println(strings.Repeat("=", 60))

	// Confirm
	confirm := promptInput(reader, "\nProceed with processing? (Y/n)", "y")
	if strings.ToLower(confirm) == "n" || strings.ToLower(confirm) == "no" {
		logger.Info("Processing cancelled by user")
		os.Exit(0)
	}

	// Create batch
	logger.Info("Creating batch from CSV file: %s", *csvFile)

	var profilePtr *string
	if *profileID != "" {
		profilePtr = profileID
	}

	batch, err := processor.CreateBatch(filepath.Base(*csvFile), *csvFile, &params, profilePtr)
	if err != nil {
		logger.Error("Failed to create batch: %v", err)
		os.Exit(1)
	}

	logger.Success("Batch created: %s (%d URLs)", batch.ID, batch.TotalRows)

	// Save session for resume capability
	session := Session{
		BatchID:     batch.ID,
		CSVFile:     *csvFile,
		OutputDir:   *outputDir,
		ProfileID:   *profileID,
		Model:       *model,
		Device:      *device,
		Language:    *language,
		Diarize:     *diarize,
		HFToken:     *hfToken,
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
	}
	saveSession(session, logger)

	// Start processing with progress monitoring
	runBatchWithProgress(processor, batch.ID, batch.TotalRows, logger, *verbose)
}

func promptInput(reader *bufio.Reader, prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}
	return input
}

func ifEmpty(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func printHelp() {
	fmt.Println(`
Scriberr CSV Batch Processor - Bulk YouTube Video Transcription

USAGE:
    scriberr-csvbatch [OPTIONS] [--csv <file>]

OPTIONS:
    --csv <file>          Path to CSV file with YouTube URLs
    --output <dir>        Output directory for JSON files (default: ./csv_output)
    --profile <id>        Transcription profile ID to use
    --model <name>        Whisper model: tiny, base, small, medium, large (default: small)
    --device <device>     Device: cpu, cuda, mps (default: cpu)
    --language <code>     Language code (auto-detect if not specified)
    --diarize             Enable speaker diarization
    --hf-token <token>    Hugging Face token for diarization
    --resume <batch-id>   Resume a previous session by batch ID
    --list-sessions       List all resumable sessions
    --verbose             Enable verbose output
    --log <file>          Log file path (default: csvbatch.log)
    --init                Run first-time setup
    --help                Show this help message
    --version             Show version information

EXAMPLES:
    # Interactive mode (prompts for inputs)
    scriberr-csvbatch

    # Specify CSV file only (prompts for other options)
    scriberr-csvbatch --csv videos.csv

    # Full command-line specification
    scriberr-csvbatch --csv videos.csv --output ./transcripts --model medium --device cuda

    # Resume a previous session
    scriberr-csvbatch --resume abc123-def456

    # List resumable sessions
    scriberr-csvbatch --list-sessions

CSV FORMAT:
    The CSV file should contain YouTube URLs, one per row:

    url
    https://www.youtube.com/watch?v=VIDEO_ID
    https://youtu.be/SHORT_ID

OUTPUT:
    Each video produces a JSON file: {rowId}-{videoTitle}.json
    containing the transcript and metadata.
`)
}

func runFirstTimeSetup(logger *Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  Scriberr CSV Batch Processor - First Time Setup")
	fmt.Println(strings.Repeat("=", 60))

	reader := bufio.NewReader(os.Stdin)

	// Check for required dependencies
	fmt.Println("\nChecking dependencies...")

	// Create default directories
	dirs := []string{"data", "data/uploads", "data/csv-batch"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Error("Failed to create directory %s: %v", dir, err)
		} else {
			logger.Success("Created directory: %s", dir)
		}
	}

	// Database initialization
	fmt.Println("\nInitializing database...")
	cfg := config.Load()
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		logger.Error("Failed to initialize database: %v", err)
	} else {
		logger.Success("Database initialized: %s", cfg.DatabasePath)
		database.Close()
	}

	// Prompt for Hugging Face token (optional)
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Speaker Diarization Setup (Optional)")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("To use speaker diarization, you need a Hugging Face token.")
	fmt.Println("Get one at: https://huggingface.co/settings/tokens")
	hfToken := promptInput(reader, "\nEnter Hugging Face token (or press Enter to skip)", "")
	if hfToken != "" {
		// Save to .env file
		envFile, err := os.OpenFile(".env", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			envFile.WriteString(fmt.Sprintf("\nHF_TOKEN=%s\n", hfToken))
			envFile.Close()
			logger.Success("Hugging Face token saved to .env")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  Setup Complete!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nYou can now run: scriberr-csvbatch --csv your_file.csv")
}

func saveSession(session Session, logger *Logger) {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		logger.Warn("Failed to save session: %v", err)
		return
	}

	sessionPath := filepath.Join(".", sessionFile)
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		logger.Warn("Failed to write session file: %v", err)
		return
	}

	logger.Debug("Session saved: %s", session.BatchID)
}

func loadSession() (*Session, error) {
	sessionPath := filepath.Join(".", sessionFile)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func listResumableSessions(processor *csvbatch.Processor, logger *Logger) {
	batches, err := processor.ListBatches()
	if err != nil {
		logger.Error("Failed to list batches: %v", err)
		return
	}

	if len(batches) == 0 {
		fmt.Println("No resumable sessions found.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  Resumable Sessions")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-36s %-12s %6s %6s %6s %s\n",
		"Batch ID", "Status", "Total", "Done", "Failed", "Created")
	fmt.Println(strings.Repeat("-", 80))

	for _, batch := range batches {
		completed := batch.SuccessRows + batch.FailedRows
		fmt.Printf("%-36s %-12s %6d %6d %6d %s\n",
			batch.ID,
			batch.Status,
			batch.TotalRows,
			completed,
			batch.FailedRows,
			batch.CreatedAt.Format("2006-01-02 15:04"))
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\nTo resume a session: scriberr-csvbatch --resume <batch-id>")
}

func resumeSession(processor *csvbatch.Processor, batchID string, logger *Logger, verbose bool) {
	logger.Info("Attempting to resume batch: %s", batchID)

	batch, err := processor.GetBatchStatus(batchID)
	if err != nil {
		logger.Error("Batch not found: %s", batchID)
		return
	}

	// Check if batch can be resumed
	if batch.Status == models.CSVBatchStatusCompleted {
		logger.Info("Batch already completed")
		return
	}

	if batch.Status == models.CSVBatchStatusProcessing {
		logger.Warn("Batch is currently processing")
		return
	}

	// Get pending row count
	rows, err := processor.GetBatchRows(batchID)
	if err != nil {
		logger.Error("Failed to get batch rows: %v", err)
		return
	}

	pendingCount := 0
	for _, row := range rows {
		if row.Status == models.CSVRowStatusPending {
			pendingCount++
		}
	}

	logger.Info("Found %d pending rows out of %d total", pendingCount, batch.TotalRows)

	if pendingCount == 0 {
		logger.Info("No pending rows to process")
		return
	}

	// Start processing
	runBatchWithProgress(processor, batchID, batch.TotalRows, logger, verbose)
}

func runBatchWithProgress(processor *csvbatch.Processor, batchID string, totalRows int, logger *Logger, verbose bool) {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start batch processing
	if err := processor.StartBatch(batchID); err != nil {
		logger.Error("Failed to start batch: %v", err)
		return
	}

	logger.Info("Batch processing started")
	fmt.Println()

	// Create progress bar
	progress := newProgressBar(totalRows)

	// Monitor progress
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	done := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				batch, err := processor.GetBatchStatus(batchID)
				if err != nil {
					continue
				}

				completed := batch.SuccessRows + batch.FailedRows
				var status string
				if batch.Status == models.CSVBatchStatusProcessing {
					status = fmt.Sprintf("Row %d", batch.CurrentRow)
				} else {
					status = string(batch.Status)
				}

				progress.Update(completed, status)

				// Check if complete
				if batch.Status == models.CSVBatchStatusCompleted ||
					batch.Status == models.CSVBatchStatusFailed ||
					batch.Status == models.CSVBatchStatusCancelled {
					done <- true
					return
				}

			case <-sigChan:
				fmt.Println("\n\nReceived interrupt signal. Stopping batch...")
				processor.StopBatch(batchID)
				logger.Warn("Batch processing interrupted by user")
				logger.Info("To resume: scriberr-csvbatch --resume %s", batchID)
				done <- true
				return
			}
		}
	}()

	// Wait for completion
	<-done

	// Get final status
	batch, err := processor.GetBatchStatus(batchID)
	if err != nil {
		logger.Error("Failed to get final status: %v", err)
		return
	}

	progress.Complete()
	fmt.Println()

	// Print summary
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  Processing Summary")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  Status:     %s\n", batch.Status)
	fmt.Printf("  Total:      %d\n", batch.TotalRows)
	fmt.Printf("  Successful: %d\n", batch.SuccessRows)
	fmt.Printf("  Failed:     %d\n", batch.FailedRows)
	fmt.Printf("  Output Dir: %s\n", batch.OutputDir)
	fmt.Println(strings.Repeat("=", 60))

	if batch.Status == models.CSVBatchStatusCompleted {
		logger.Success("Batch processing completed successfully!")
	} else if batch.Status == models.CSVBatchStatusFailed {
		logger.Error("Batch processing failed: %s", ifEmpty(ptrToString(batch.ErrorMessage), "Unknown error"))
	} else if batch.Status == models.CSVBatchStatusCancelled {
		logger.Warn("Batch processing was cancelled")
		logger.Info("To resume: scriberr-csvbatch --resume %s", batchID)
	}

	// List failed rows if any
	if batch.FailedRows > 0 && verbose {
		fmt.Println("\nFailed rows:")
		rows, _ := processor.GetBatchRows(batchID)
		for _, row := range rows {
			if row.Status == models.CSVRowStatusFailed {
				fmt.Printf("  Row %d: %s\n", row.RowID, ifEmpty(ptrToString(row.ErrorMessage), "Unknown error"))
			}
		}
	}

	// Remove session file if completed
	if batch.Status == models.CSVBatchStatusCompleted {
		os.Remove(sessionFile)
	}
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
