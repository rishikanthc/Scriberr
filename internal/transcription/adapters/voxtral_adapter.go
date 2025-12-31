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
	"scriberr/pkg/logger"
)

//go:embed py/voxtral/*
var voxtralScripts embed.FS

// VoxtralAdapter implements the TranscriptionAdapter interface for Mistral Voxtral-mini
type VoxtralAdapter struct {
	*BaseAdapter
	envPath string
}

// NewVoxtralAdapter creates a new Voxtral adapter
func NewVoxtralAdapter(envPath string) *VoxtralAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:     "voxtral",
		ModelFamily: "mistral_voxtral",
		DisplayName: "Mistral Voxtral-mini",
		Description: "Mistral's multilingual audio transcription model",
		Version:     "1.0.0",
		SupportedLanguages: []string{
			"en", "es", "fr", "de", "it", "pt", "nl", "pl", "ru", "zh", "ja", "ko",
			// Voxtral supports many languages
		},
		SupportedFormats:  []string{"wav", "mp3", "flac", "m4a", "ogg"},
		RequiresGPU:       false, // Can run on CPU but GPU recommended
		MemoryRequirement: 4096,  // 4GB recommended
		Features: map[string]bool{
			"timestamps":         false, // Voxtral doesn't provide word-level timestamps
			"word_level":         false,
			"multilingual":       true,
			"high_quality":       true,
			"fast_inference":     true,
			"transformers_based": true,
		},
		Metadata: map[string]string{
			"engine":             "mistral_ai",
			"framework":          "transformers",
			"license":            "Apache-2.0",
			"model_id":           "mistralai/Voxtral-mini",
			"no_word_timestamps": "true", // Important metadata for frontend
		},
	}

	schema := []interfaces.ParameterSchema{
		// Language selection
		{
			Name:        "language",
			Type:        "string",
			Required:    false,
			Default:     "en",
			Options:     []string{"en", "es", "fr", "de", "it", "pt", "nl", "pl", "ru", "zh", "ja", "ko"},
			Description: "Language of the audio",
			Group:       "basic",
		},

		// Generation settings
		{
			Name:        "max_new_tokens",
			Type:        "int",
			Required:    false,
			Default:     8192,
			Min:         &[]float64{1024}[0],
			Max:         &[]float64{16384}[0],
			Description: "Maximum number of tokens to generate (Voxtral has 32k context window)",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("voxtral", envPath, capabilities, schema)

	adapter := &VoxtralAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetSupportedModels returns the available Voxtral models
func (v *VoxtralAdapter) GetSupportedModels() []string {
	return []string{"mistralai/Voxtral-mini"}
}

// PrepareEnvironment sets up the Voxtral environment
func (v *VoxtralAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing Voxtral environment", "env_path", v.envPath)

	// Copy transcription script
	if err := v.copyTranscriptionScript(); err != nil {
		return fmt.Errorf("failed to copy transcription script: %w", err)
	}

	// Check if environment is already ready (check both transformers AND mistral-common)
	if CheckEnvironmentReady(v.envPath, "from transformers import VoxtralForConditionalGeneration") &&
		CheckEnvironmentReady(v.envPath, "import mistral_common") {
		logger.Info("Voxtral environment already ready")
		v.initialized = true
		return nil
	}

	// Setup environment
	if err := v.setupVoxtralEnvironment(); err != nil {
		return fmt.Errorf("failed to setup Voxtral environment: %w", err)
	}

	v.initialized = true
	logger.Info("Voxtral environment prepared successfully")
	return nil
}

// setupVoxtralEnvironment creates the Python environment for Voxtral
func (v *VoxtralAdapter) setupVoxtralEnvironment() error {
	if err := os.MkdirAll(v.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create voxtral directory: %w", err)
	}

	// Read pyproject.toml
	pyprojectContent, err := voxtralScripts.ReadFile("py/voxtral/pyproject.toml")
	if err != nil {
		return fmt.Errorf("failed to read embedded pyproject.toml: %w", err)
	}

	// Replace the hardcoded PyTorch URL with the dynamic one based on environment
	contentStr := strings.Replace(
		string(pyprojectContent),
		"https://download.pytorch.org/whl/cu126",
		GetPyTorchWheelURL(),
		1,
	)

	pyprojectPath := filepath.Join(v.envPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing Voxtral dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = v.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// copyTranscriptionScript creates the Python scripts for Voxtral transcription
func (v *VoxtralAdapter) copyTranscriptionScript() error {
	// Ensure directory exists before writing script
	if err := os.MkdirAll(v.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Copy regular transcription script
	scriptContent, err := voxtralScripts.ReadFile("py/voxtral/voxtral_transcribe.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded voxtral_transcribe.py: %w", err)
	}

	scriptPath := filepath.Join(v.envPath, "voxtral_transcribe.py")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write transcription script: %w", err)
	}

	// Copy buffered transcription script for long audio files
	bufferedContent, err := voxtralScripts.ReadFile("py/voxtral/voxtral_transcribe_buffered.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded voxtral_transcribe_buffered.py: %w", err)
	}

	bufferedPath := filepath.Join(v.envPath, "voxtral_transcribe_buffered.py")
	if err := os.WriteFile(bufferedPath, bufferedContent, 0755); err != nil {
		return fmt.Errorf("failed to write buffered transcription script: %w", err)
	}

	return nil
}

// Transcribe processes audio using Voxtral
func (v *VoxtralAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	v.LogProcessingStart(input, procCtx)
	defer func() {
		v.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := v.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := v.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := v.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer v.CleanupTempDirectory(tempDir)

	// Build command arguments
	args, err := v.buildVoxtralArgs(input, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute Voxtral
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	// Setup log file
	logFile, err := os.OpenFile(filepath.Join(procCtx.OutputDirectory, "transcription.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Warn("Failed to create log file", "error", err)
	} else {
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	logger.Info("Executing Voxtral command", "args", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("transcription was cancelled")
		}

		// Read tail of log file for context
		logPath := filepath.Join(procCtx.OutputDirectory, "transcription.log")
		logTail, readErr := v.ReadLogTail(logPath, 2048)
		if readErr != nil {
			logger.Warn("Failed to read log tail", "error", readErr)
		}

		logger.Error("Voxtral execution failed", "error", err)
		return nil, fmt.Errorf("Voxtral execution failed: %w\nLogs:\n%s", err, logTail)
	}

	// Parse result
	result, err := v.parseResult(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "mistralai/Voxtral-mini"

	logger.Info("Voxtral transcription completed",
		"text_length", len(result.Text),
		"processing_time", result.ProcessingTime)

	return result, nil
}

// buildVoxtralArgs builds the command arguments for Voxtral
func (v *VoxtralAdapter) buildVoxtralArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFile := filepath.Join(tempDir, "result.json")

	// Determine if we should use buffered mode based on audio duration
	// Voxtral handles 30-40 minutes natively, use buffered mode for longer files
	useBuffered := input.Duration > 30*60 // More than 30 minutes

	var scriptPath string
	if useBuffered {
		scriptPath = filepath.Join(v.envPath, "voxtral_transcribe_buffered.py")
		logger.Info("Using buffered Voxtral for long audio", "duration", input.Duration)
	} else {
		scriptPath = filepath.Join(v.envPath, "voxtral_transcribe.py")
	}

	args := []string{
		"run", "--native-tls", "--project", v.envPath, "python", scriptPath,
		input.FilePath,
		outputFile,
	}

	// Add language
	if language := v.GetStringParameter(params, "language"); language != "" {
		args = append(args, "--language", language)
	}

	// Device auto-detection (like Parakeet/Canary) - no device parameter needed
	// Python script will auto-detect and use GPU if available

	// Add max tokens
	if maxTokens := v.GetIntParameter(params, "max_new_tokens"); maxTokens > 0 {
		args = append(args, "--max-new-tokens", fmt.Sprintf("%d", maxTokens))
	}

	// Add chunk length for buffered mode (default: 25 minutes = 1500 seconds)
	if useBuffered {
		args = append(args, "--chunk-len", "1500")
	}

	return args, nil
}

// parseResult parses the Voxtral output
func (v *VoxtralAdapter) parseResult(tempDir string) (*interfaces.TranscriptResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var voxtralResult struct {
		Text              string `json:"text"`
		Language          string `json:"language"`
		Model             string `json:"model"`
		HasWordTimestamps bool   `json:"has_word_timestamps"`
		Segments          []struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
			Text  string  `json:"text"`
		} `json:"segments"`
	}

	if err := json.Unmarshal(data, &voxtralResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Convert to standard format
	// Note: Voxtral doesn't provide word-level timestamps, so we create segments without words
	result := &interfaces.TranscriptResult{
		Text:     voxtralResult.Text,
		Language: voxtralResult.Language,
		Segments: make([]interfaces.TranscriptSegment, len(voxtralResult.Segments)),
	}

	for i, seg := range voxtralResult.Segments {
		result.Segments[i] = interfaces.TranscriptSegment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Text,
		}
	}

	return result, nil
}

// GetEstimatedProcessingTime provides Voxtral-specific time estimation
func (v *VoxtralAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Voxtral is relatively fast
	baseTime := v.BaseAdapter.GetEstimatedProcessingTime(input)

	// Voxtral typically processes at about 10-20% of audio duration
	return time.Duration(float64(baseTime) * 0.15)
}
