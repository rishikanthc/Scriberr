package adapters

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/downloader"
	"scriberr/pkg/logger"
)

//go:embed py/nvidia/*
var nvidiaScripts embed.FS

// CanaryAdapter implements the TranscriptionAdapter interface for NVIDIA Canary
type CanaryAdapter struct {
	*BaseAdapter
	envPath string
}

// NewCanaryAdapter creates a new Canary adapter
func NewCanaryAdapter(envPath string) *CanaryAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:     "canary",
		ModelFamily: "nvidia_canary",
		DisplayName: "NVIDIA Canary 1B v2",
		Description: "NVIDIA's multilingual Canary model with translation capabilities",
		Version:     "1.2.0",
		SupportedLanguages: []string{
			"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh",
			// Canary supports many more languages
		},
		SupportedFormats:  []string{"wav", "flac"},
		RequiresGPU:       false, // Can run on CPU but GPU strongly recommended
		MemoryRequirement: 8192,  // 8GB+ recommended for Canary
		Features: map[string]bool{
			"timestamps":     true,
			"word_level":     true,
			"multilingual":   true,
			"translation":    true,
			"high_quality":   true,
			"code_switching": true,
		},
		Metadata: map[string]string{
			"engine":         "nvidia_nemo",
			"framework":      "nemo_toolkit",
			"license":        "CC-BY-4.0",
			"multilingual":   "true",
			"sample_rate":    "16000",
			"format":         "16khz_mono_wav",
			"memory_warning": "requires_8gb_plus",
		},
	}

	schema := []interfaces.ParameterSchema{
		// Language settings
		{
			Name:        "source_lang",
			Type:        "string",
			Required:    false,
			Default:     "en",
			Options:     []string{"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"},
			Description: "Source language of the audio",
			Group:       "basic",
		},
		{
			Name:        "target_lang",
			Type:        "string",
			Required:    false,
			Default:     "en",
			Options:     []string{"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"},
			Description: "Target language for transcription/translation",
			Group:       "basic",
		},
		{
			Name:        "task",
			Type:        "string",
			Required:    false,
			Default:     "transcribe",
			Options:     []string{"transcribe", "translate"},
			Description: "Task to perform: transcribe (same language) or translate",
			Group:       "basic",
		},

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

		// Audio preprocessing
		{
			Name:        "auto_convert_audio",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Automatically convert audio to 16kHz mono WAV",
			Group:       "advanced",
		},

		// Performance settings
		{
			Name:        "batch_size",
			Type:        "int",
			Required:    false,
			Default:     1,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{8}[0],
			Description: "Batch size for processing (higher uses more memory)",
			Group:       "advanced",
		},

		// Output settings
		{
			Name:        "include_confidence",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Include confidence scores in output",
			Group:       "advanced",
		},
		{
			Name:        "preserve_formatting",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Preserve punctuation and capitalization",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("canary", envPath, capabilities, schema)

	adapter := &CanaryAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetSupportedModels returns the specific Canary model available
func (c *CanaryAdapter) GetSupportedModels() []string {
	return []string{"canary-1b-v2"}
}

// PrepareEnvironment sets up the Canary environment (shared with Parakeet)
func (c *CanaryAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Canary environment", "env_path", c.envPath)

	// Copy transcription script
	if err := c.copyTranscriptionScript(); err != nil {
		return fmt.Errorf("failed to copy transcription script: %w", err)
	}

	// Check if environment is already ready (using cache to speed up repeated checks)
	if CheckEnvironmentReady(c.envPath, "import nemo.collections.asr") {
		modelPath := filepath.Join(c.envPath, "canary-1b-v2.nemo")
		if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
			logger.Info("Canary environment already ready")
			c.initialized = true
			return nil
		}
	}

	// Setup environment (reuse Parakeet setup since they share the same environment)
	if err := c.setupCanaryEnvironment(); err != nil {
		return fmt.Errorf("failed to setup Canary environment: %w", err)
	}

	// Download model
	if err := c.downloadCanaryModel(); err != nil {
		return fmt.Errorf("failed to download Canary model: %w", err)
	}

	c.initialized = true
	logger.Info("Canary environment prepared successfully")
	return nil
}

// setupCanaryEnvironment creates the Python environment (shared with Parakeet)
func (c *CanaryAdapter) setupCanaryEnvironment() error {
	if err := os.MkdirAll(c.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create canary directory: %w", err)
	}

	// Check if pyproject.toml already exists from Parakeet setup
	pyprojectPath := filepath.Join(c.envPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		logger.Info("Environment already configured by Parakeet")
		return nil
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

	pyprojectPath = filepath.Join(c.envPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing Canary dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = c.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// downloadCanaryModel downloads the Canary model file
func (c *CanaryAdapter) downloadCanaryModel() error {
	modelFileName := "canary-1b-v2.nemo"
	modelPath := filepath.Join(c.envPath, modelFileName)

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		logger.Info("Canary model already exists", "path", modelPath, "size", stat.Size())
		return nil
	}

	logger.Info("Downloading Canary model", "path", modelPath)

	modelURL := "https://huggingface.co/nvidia/canary-1b-v2/resolve/main/canary-1b-v2.nemo?download=true"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := downloader.DownloadFile(ctx, modelURL, modelPath); err != nil {
		return fmt.Errorf("failed to download Canary model: %w", err)
	}

	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("downloaded model file not found: %w", err)
	}
	if stat.Size() < 1024*1024 {
		return fmt.Errorf("downloaded model file appears incomplete (size: %d bytes)", stat.Size())
	}

	logger.Info("Successfully downloaded Canary model", "size", stat.Size())
	return nil
}

// copyTranscriptionScript creates the Python script for Canary transcription
func (c *CanaryAdapter) copyTranscriptionScript() error {
	scriptContent, err := nvidiaScripts.ReadFile("py/nvidia/canary_transcribe.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded canary_transcribe.py: %w", err)
	}

	scriptPath := filepath.Join(c.envPath, "canary_transcribe.py")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write transcription script: %w", err)
	}

	return nil
}

// Transcribe processes audio using Canary
func (c *CanaryAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	c.LogProcessingStart(input, procCtx)
	defer func() {
		c.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := c.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := c.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := c.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer c.CleanupTempDirectory(tempDir)

	// Convert audio if needed (Canary requires 16kHz mono WAV)
	audioInput := input
	if c.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := c.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	// Build command arguments
	args, err := c.buildCanaryArgs(audioInput, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute Canary
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(),
		"PYTHONUNBUFFERED=1",
		"PYTORCH_CUDA_ALLOC_CONF=expandable_segments:True")

	// Setup log file
	logFile, err := os.OpenFile(filepath.Join(procCtx.OutputDirectory, "transcription.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Warn("Failed to create log file", "error", err)
	} else {
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	logger.Info("Executing Canary command", "args", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("transcription was cancelled")
		}

		// Read tail of log file for context
		logPath := filepath.Join(procCtx.OutputDirectory, "transcription.log")
		logTail, readErr := c.ReadLogTail(logPath, 2048)
		if readErr != nil {
			logger.Warn("Failed to read log tail", "error", readErr)
		}

		logger.Error("Canary execution failed", "error", err)
		return nil, fmt.Errorf("Canary execution failed: %w\nLogs:\n%s", err, logTail)
	}

	// Parse result
	result, err := c.parseResult(tempDir, audioInput, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "canary-1b-v2"
	result.Metadata = c.CreateDefaultMetadata(params)

	logger.Info("Canary transcription completed",
		"segments", len(result.Segments),
		"words", len(result.WordSegments),
		"processing_time", result.ProcessingTime,
		"task", c.GetStringParameter(params, "task"))

	return result, nil
}

// buildCanaryArgs builds the command arguments for Canary
func (c *CanaryAdapter) buildCanaryArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFile := filepath.Join(tempDir, "result.json")

	scriptPath := filepath.Join(c.envPath, "canary_transcribe.py")
	args := []string{
		"run", "--native-tls", "--project", c.envPath, "python", scriptPath,
		input.FilePath,
		"--output", outputFile,
	}

	// Add language settings
	args = append(args, "--source-lang", c.GetStringParameter(params, "source_lang"))
	args = append(args, "--target-lang", c.GetStringParameter(params, "target_lang"))
	args = append(args, "--task", c.GetStringParameter(params, "task"))

	// Add timestamps flag
	if c.GetBoolParameter(params, "timestamps") {
		args = append(args, "--timestamps")
	} else {
		args = append(args, "--no-timestamps")
	}

	// Add confidence flag
	if c.GetBoolParameter(params, "include_confidence") {
		args = append(args, "--include-confidence")
	} else {
		args = append(args, "--no-confidence")
	}

	// Add formatting flag
	if c.GetBoolParameter(params, "preserve_formatting") {
		args = append(args, "--preserve-formatting")
	}

	return args, nil
}

// parseResult parses the Canary output
func (c *CanaryAdapter) parseResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.TranscriptResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var canaryResult struct {
		Transcription  string `json:"transcription"`
		SourceLanguage string `json:"source_language"`
		TargetLanguage string `json:"target_language"`
		Task           string `json:"task"`
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

	if err := json.Unmarshal(data, &canaryResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Determine the language for the result
	resultLanguage := canaryResult.TargetLanguage
	if canaryResult.Task == "transcribe" {
		resultLanguage = canaryResult.SourceLanguage
	}

	// Convert to standard format
	result := &interfaces.TranscriptResult{
		Text:         canaryResult.Transcription,
		Language:     resultLanguage,
		Segments:     make([]interfaces.TranscriptSegment, len(canaryResult.SegmentTimestamps)),
		WordSegments: make([]interfaces.TranscriptWord, len(canaryResult.WordTimestamps)),
		Confidence:   0.0, // Default confidence
	}

	// Convert segments
	for i, seg := range canaryResult.SegmentTimestamps {
		result.Segments[i] = interfaces.TranscriptSegment{
			Start:    seg.Start,
			End:      seg.End,
			Text:     seg.Segment,
			Language: &resultLanguage,
		}
	}

	// Convert words
	for i, word := range canaryResult.WordTimestamps {
		result.WordSegments[i] = interfaces.TranscriptWord{
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
			Score: 1.0, // Canary doesn't provide word-level scores
		}
	}

	return result, nil
}

// GetEstimatedProcessingTime provides Canary-specific time estimation
func (c *CanaryAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Canary is typically slower than Parakeet due to its multilingual capabilities
	baseTime := c.BaseAdapter.GetEstimatedProcessingTime(input)

	// Canary typically processes at about 40-50% of audio duration
	return time.Duration(float64(baseTime) * 2.0)
}
