package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/downloader"
	"scriberr/pkg/logger"
)

// ParakeetAdapter implements the TranscriptionAdapter interface for NVIDIA Parakeet
type ParakeetAdapter struct {
	*BaseAdapter
	envPath string
}

// NewParakeetAdapter creates a new Parakeet adapter
func NewParakeetAdapter(envPath string) *ParakeetAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "parakeet",
		ModelFamily:        "nvidia_parakeet",
		DisplayName:        "NVIDIA Parakeet TDT 0.6B v3",
		Description:        "NVIDIA's Parakeet model for English transcription with timestamps",
		Version:            "0.6.3",
		SupportedLanguages: []string{"en"}, // English only
		SupportedFormats:   []string{"wav", "flac"},
		RequiresGPU:        false, // Can run on CPU but GPU recommended
		MemoryRequirement:  4096,  // 4GB recommended
		Features: map[string]bool{
			"timestamps":        true,
			"word_level":        true,
			"long_form":         true,
			"attention_context": true,
			"high_quality":      true,
		},
		Metadata: map[string]string{
			"engine":      "nvidia_nemo",
			"framework":   "nemo_toolkit",
			"license":     "CC-BY-4.0",
			"language":    "english_only",
			"sample_rate": "16000",
			"format":      "16khz_mono_wav",
		},
	}

	schema := []interfaces.ParameterSchema{
		// Core settings
		{
			Name:        "timestamps",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Include word and segment level timestamps",
			Group:       "basic",
		},
		{
			Name:        "output_format",
			Type:        "string",
			Required:    false,
			Default:     "json",
			Options:     []string{"json", "text"},
			Description: "Output format for results",
			Group:       "basic",
		},

		// Long-form audio settings (Parakeet specific)
		{
			Name:        "context_left",
			Type:        "int",
			Required:    false,
			Default:     256,
			Min:         &[]float64{0}[0],
			Max:         &[]float64{1024}[0],
			Description: "Left attention context size for long-form audio",
			Group:       "advanced",
		},
		{
			Name:        "context_right",
			Type:        "int",
			Required:    false,
			Default:     256,
			Min:         &[]float64{0}[0],
			Max:         &[]float64{1024}[0],
			Description: "Right attention context size for long-form audio",
			Group:       "advanced",
		},

		// Audio preprocessing
		{
			Name:        "auto_convert_audio",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Automatically convert audio to 16kHz mono WAV",
			Group:       "advanced",
		},

		// Note: include_confidence removed as it's not supported by Parakeet script
	}

	baseAdapter := NewBaseAdapter("parakeet", envPath, capabilities, schema)

	adapter := &ParakeetAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetSupportedModels returns the specific Parakeet model available
func (p *ParakeetAdapter) GetSupportedModels() []string {
	return []string{"parakeet-tdt-0.6b-v3"}
}

// PrepareEnvironment sets up the Parakeet environment
func (p *ParakeetAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Parakeet environment", "env_path", p.envPath)

	// Copy transcription scripts (standard and buffered)
	if err := p.copyTranscriptionScript(); err != nil {
		return fmt.Errorf("failed to copy transcription script: %w", err)
	}

	if err := p.copyBufferedScript(); err != nil {
		return fmt.Errorf("failed to create buffered script: %w", err)
	}

	// Check if environment is already ready (using cache to speed up repeated checks)
	if CheckEnvironmentReady(p.envPath, "import nemo.collections.asr") {
		modelPath := filepath.Join(p.envPath, "parakeet-tdt-0.6b-v3.nemo")
		scriptPath := filepath.Join(p.envPath, "parakeet_transcribe.py")
		bufferedScriptPath := filepath.Join(p.envPath, "parakeet_transcribe_buffered.py")

		// Check model, standard script, and buffered script all exist
		if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
			_, scriptErr := os.Stat(scriptPath)
			_, bufferedErr := os.Stat(bufferedScriptPath)

			if scriptErr == nil && bufferedErr == nil {
				logger.Info("Parakeet environment already ready")
				p.initialized = true
				return nil
			}
			logger.Info("Parakeet model exists but scripts missing, recreating scripts")
		} else {
			logger.Info("Parakeet model file missing or incomplete, redownloading")
		}
	} else {
		logger.Info("Parakeet environment not ready, setting up")
	}

	// Setup environment
	if err := p.setupParakeetEnvironment(); err != nil {
		return fmt.Errorf("failed to setup Parakeet environment: %w", err)
	}

	// Download model
	if err := p.downloadParakeetModel(); err != nil {
		return fmt.Errorf("failed to download Parakeet model: %w", err)
	}

	p.initialized = true
	logger.Info("Parakeet environment prepared successfully")
	return nil
}

// setupParakeetEnvironment creates the Python environment for Parakeet
func (p *ParakeetAdapter) setupParakeetEnvironment() error {
	if err := os.MkdirAll(p.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create parakeet directory: %w", err)
	}

	// Read pyproject.toml
	pyprojectContent, err := nvidiaScripts.ReadFile("py/nvidia/pyproject.toml")
	if err != nil {
		return fmt.Errorf("failed to read embedded pyproject.toml: %w", err)
	}

	// Replace the hardcoded PyTorch URL with the dynamic one based on environment
	// The static file contains the default cu126 URL
	contentStr := strings.Replace(
		string(pyprojectContent),
		"https://download.pytorch.org/whl/cu126",
		GetPyTorchWheelURL(),
		1,
	)

	pyprojectPath := filepath.Join(p.envPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing Parakeet dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = p.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// downloadParakeetModel downloads the Parakeet model file
func (p *ParakeetAdapter) downloadParakeetModel() error {
	modelFileName := "parakeet-tdt-0.6b-v3.nemo"
	modelPath := filepath.Join(p.envPath, modelFileName)

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		logger.Info("Parakeet model already exists", "path", modelPath, "size", stat.Size())
		return nil
	}

	logger.Info("Downloading Parakeet model", "path", modelPath)

	modelURL := "https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3/resolve/main/parakeet-tdt-0.6b-v3.nemo?download=true"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := downloader.DownloadFile(ctx, modelURL, modelPath); err != nil {
		return fmt.Errorf("failed to download Parakeet model: %w", err)
	}

	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("downloaded model file not found: %w", err)
	}
	if stat.Size() < 1024*1024 {
		return fmt.Errorf("downloaded model file appears incomplete (size: %d bytes)", stat.Size())
	}

	logger.Info("Successfully downloaded Parakeet model", "size", stat.Size())
	return nil
}

// copyTranscriptionScript creates the Python script for Parakeet transcription
func (p *ParakeetAdapter) copyTranscriptionScript() error {
	scriptContent, err := nvidiaScripts.ReadFile("py/nvidia/parakeet_transcribe.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded transcribe.py: %w", err)
	}

	scriptPath := filepath.Join(p.envPath, "parakeet_transcribe.py")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write transcription script: %w", err)
	}

	return nil
}

// Transcribe processes audio using Parakeet
func (p *ParakeetAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	p.LogProcessingStart(input, procCtx)
	defer func() {
		p.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := p.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := p.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer p.CleanupTempDirectory(tempDir)

	// Convert audio if needed (Parakeet requires 16kHz mono WAV)
	audioInput := input
	if p.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := p.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	// Detect audio duration and choose processing path
	audioDuration := input.Duration
	if audioDuration == 0 {
		// Duration not provided, try to detect it
		durationSecs, err := p.detectAudioDuration(audioInput.FilePath)
		if err != nil {
			logger.Warn("Failed to detect audio duration, using standard transcription", "error", err)
			audioDuration = 0
		} else {
			audioDuration = time.Duration(durationSecs * float64(time.Second))
		}
	}

	// Get chunk threshold from environment (default: 300 seconds = 5 minutes)
	chunkThreshold := 300
	if thresholdStr := os.Getenv("PARAKEET_CHUNK_THRESHOLD_SECS"); thresholdStr != "" {
		if parsed, err := strconv.Atoi(thresholdStr); err == nil && parsed > 0 {
			chunkThreshold = parsed
		}
	}

	// Choose processing path based on audio duration
	chunkThresholdDuration := time.Duration(chunkThreshold) * time.Second
	var result *interfaces.TranscriptResult
	if audioDuration > chunkThresholdDuration {
		logger.Info("Using buffered inference for long audio",
			"duration_secs", audioDuration.Seconds(),
			"threshold_secs", chunkThreshold)
		result, err = p.transcribeBuffered(ctx, audioInput, params, tempDir, procCtx.OutputDirectory)
	} else {
		logger.Info("Using standard transcription for short audio",
			"duration_secs", audioDuration.Seconds(),
			"threshold_secs", chunkThreshold)
		result, err = p.transcribeStandard(ctx, audioInput, params, tempDir, procCtx.OutputDirectory)
	}

	if err != nil {
		return nil, err
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "parakeet-tdt-0.6b-v3"
	result.Metadata = p.CreateDefaultMetadata(params)

	logger.Info("Parakeet transcription completed",
		"segments", len(result.Segments),
		"words", len(result.WordSegments),
		"processing_time", result.ProcessingTime)

	return result, nil
}

// detectAudioDuration uses ffprobe to detect audio duration
func (p *ParakeetAdapter) detectAudioDuration(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

// transcribeStandard uses the standard Parakeet transcription (original method)
func (p *ParakeetAdapter) transcribeStandard(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, tempDir, outputDir string) (*interfaces.TranscriptResult, error) {
	// Build command arguments
	args, err := p.buildParakeetArgs(input, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute Parakeet
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	// Setup log file
	logFile, err := os.OpenFile(filepath.Join(outputDir, "transcription.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Warn("Failed to create log file", "error", err)
	} else {
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	logger.Info("Executing Parakeet command", "args", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("transcription was cancelled")
		}

		// Read tail of log file for context
		logPath := filepath.Join(outputDir, "transcription.log")
		logTail, readErr := p.ReadLogTail(logPath, 2048)
		if readErr != nil {
			logger.Warn("Failed to read log tail", "error", readErr)
		}

		logger.Error("Parakeet execution failed", "error", err)
		return nil, fmt.Errorf("Parakeet execution failed: %w\nLogs:\n%s", err, logTail)
	}

	// Parse result
	result, err := p.parseResult(tempDir, input, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return result, nil
}

// transcribeBuffered uses NeMo's buffered inference for long audio
func (p *ParakeetAdapter) transcribeBuffered(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, tempDir, outputDir string) (*interfaces.TranscriptResult, error) {
	// Build command arguments for buffered inference
	args, err := p.buildBufferedArgs(input, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build buffered command: %w", err)
	}

	// Execute buffered inference
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	// Setup log file
	logFile, err := os.OpenFile(filepath.Join(outputDir, "transcription.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Warn("Failed to create log file", "error", err)
	} else {
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	logger.Info("Executing Parakeet buffered inference", "args", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("transcription was cancelled")
		}

		// Read tail of log file for context
		logPath := filepath.Join(outputDir, "transcription.log")
		logTail, readErr := p.ReadLogTail(logPath, 2048)
		if readErr != nil {
			logger.Warn("Failed to read log tail", "error", readErr)
		}

		logger.Error("Parakeet buffered execution failed", "error", err)
		return nil, fmt.Errorf("Parakeet buffered execution failed: %w\nLogs:\n%s", err, logTail)
	}

	// Parse buffered result
	result, err := p.parseBufferedResult(tempDir, input, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse buffered result: %w", err)
	}

	return result, nil
}

// buildParakeetArgs builds the command arguments for Parakeet
func (p *ParakeetAdapter) buildParakeetArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFile := filepath.Join(tempDir, "result.json")

	scriptPath := filepath.Join(p.envPath, "parakeet_transcribe.py")
	args := []string{
		"run", "--native-tls", "--project", p.envPath, "python", scriptPath,
		input.FilePath,
		"--output", outputFile,
	}

	// Add timestamps flag (Parakeet script supports --timestamps)
	if p.GetBoolParameter(params, "timestamps") {
		args = append(args, "--timestamps")
	}

	// Add context settings
	args = append(args, "--context-left", strconv.Itoa(p.GetIntParameter(params, "context_left")))
	args = append(args, "--context-right", strconv.Itoa(p.GetIntParameter(params, "context_right")))

	// Note: --include-confidence is not supported by Parakeet script, removed

	return args, nil
}

// parseResult parses the Parakeet output
func (p *ParakeetAdapter) parseResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.TranscriptResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var parakeetResult struct {
		Transcription  string `json:"transcription"`
		Language       string `json:"language"`
		WordTimestamps []struct {
			Word        string  `json:"word"`
			StartOffset int     `json:"start_offset"`
			EndOffset   int     `json:"end_offset"`
			Start       float64 `json:"start"`
			End         float64 `json:"end"`
		} `json:"word_timestamps"`
		SegmentTimestamps []struct {
			Segment     string  `json:"segment"`
			StartOffset int     `json:"start_offset"`
			EndOffset   int     `json:"end_offset"`
			Start       float64 `json:"start"`
			End         float64 `json:"end"`
		} `json:"segment_timestamps"`
		Confidence interface{} `json:"confidence,omitempty"`
	}

	if err := json.Unmarshal(data, &parakeetResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Convert to standard format
	result := &interfaces.TranscriptResult{
		Text:         parakeetResult.Transcription,
		Language:     parakeetResult.Language,
		Segments:     make([]interfaces.TranscriptSegment, len(parakeetResult.SegmentTimestamps)),
		WordSegments: make([]interfaces.TranscriptWord, len(parakeetResult.WordTimestamps)),
		Confidence:   0.0, // Default confidence
	}

	// Convert segments
	for i, seg := range parakeetResult.SegmentTimestamps {
		result.Segments[i] = interfaces.TranscriptSegment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Segment,
		}
	}

	// Convert words
	for i, word := range parakeetResult.WordTimestamps {
		result.WordSegments[i] = interfaces.TranscriptWord{
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
			Score: 1.0, // Parakeet doesn't provide word-level scores
		}
	}

	return result, nil
}

// copyBufferedScript creates the Python script for NeMo buffered inference
func (p *ParakeetAdapter) copyBufferedScript() error {
	scriptContent, err := nvidiaScripts.ReadFile("py/nvidia/parakeet_transcribe_buffered.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded transcribe_buffered.py: %w", err)
	}

	scriptPath := filepath.Join(p.envPath, "parakeet_transcribe_buffered.py")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write buffered script: %w", err)
	}

	logger.Info("Created buffered transcription script", "path", scriptPath)
	return nil
}

// buildBufferedArgs builds the command arguments for buffered inference
func (p *ParakeetAdapter) buildBufferedArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFile := filepath.Join(tempDir, "result.json")

	// Get chunk threshold from environment (default: 300 seconds = 5 minutes)
	chunkDuration := "300"
	if thresholdStr := os.Getenv("PARAKEET_CHUNK_THRESHOLD_SECS"); thresholdStr != "" {
		chunkDuration = thresholdStr
	}

	scriptPath := filepath.Join(p.envPath, "parakeet_transcribe_buffered.py")
	args := []string{
		"run", "--native-tls", "--project", p.envPath, "python", scriptPath,
		input.FilePath,
		"--output", outputFile,
		"--chunk-len", chunkDuration,
	}

	return args, nil
}

// parseBufferedResult parses the buffered inference output
func (p *ParakeetAdapter) parseBufferedResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.TranscriptResult, error) {
	// Buffered inference uses the same output format as standard transcription
	return p.parseResult(tempDir, input, params)
}

// GetEstimatedProcessingTime provides Parakeet-specific time estimation
func (p *ParakeetAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Parakeet is generally faster than WhisperX but slower than real-time
	baseTime := p.BaseAdapter.GetEstimatedProcessingTime(input)

	// Parakeet typically processes at about 30% of audio duration
	return time.Duration(float64(baseTime) * 1.5)
}
