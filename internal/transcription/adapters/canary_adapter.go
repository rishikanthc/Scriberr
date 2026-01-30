package adapters

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/asrengine"
	"scriberr/internal/asrengine/pb"
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
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     "nemo-canary-1b-v2",
			Options:     []string{"nemo-canary-1b-v2"},
			Description: "Canary model variant",
			Group:       "basic",
		},
		// Language settings
		{
			Name:        "language",
			Type:        "string",
			Required:    false,
			Default:     "en",
			Options:     []string{"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"},
			Description: "Source language of the audio",
			Group:       "basic",
		},
		{
			Name:        "target_language",
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
		{
			Name:        "pnc",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Output punctuation and capitalization",
			Group:       "advanced",
		},
		// VAD tuning (onnx-asr)
		{
			Name:        "vad_preset",
			Type:        "string",
			Required:    false,
			Default:     "balanced",
			Options:     []string{"conservative", "balanced", "aggressive"},
			Description: "VAD preset for speech segmentation",
			Group:       "advanced",
		},
		{
			Name:        "vad_speech_pad_ms",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Description: "VAD speech pad (ms)",
			Group:       "advanced",
		},
		{
			Name:        "vad_min_silence_ms",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Description: "VAD min silence duration (ms)",
			Group:       "advanced",
		},
		{
			Name:        "vad_min_speech_ms",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Description: "VAD min speech duration (ms)",
			Group:       "advanced",
		},
		{
			Name:        "vad_max_speech_s",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Description: "VAD max speech duration (s)",
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
	return []string{"nemo-canary-1b-v2"}
}

// PrepareEnvironment sets up the Canary environment (shared with Parakeet)
func (c *CanaryAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Canary environment", "env_path", c.envPath)
	if err := asrengine.Default().EnsureRunning(ctx); err != nil {
		return fmt.Errorf("failed to start ASR engine: %w", err)
	}
	c.initialized = true
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
	// Ensure directory exists before writing script
	if err := os.MkdirAll(c.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

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

	result, err := c.transcribeWithEngine(ctx, audioInput, params, procCtx)
	if err != nil {
		return nil, err
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = c.GetStringParameter(params, "model")
	if result.ModelUsed == "" {
		result.ModelUsed = "nemo-canary-1b-v2"
	}
	result.Metadata = c.CreateDefaultMetadata(params)

	logger.Info("Canary transcription completed",
		"segments", len(result.Segments),
		"words", len(result.WordSegments),
		"processing_time", result.ProcessingTime,
		"task", c.GetStringParameter(params, "task"))

	return result, nil
}

func (c *CanaryAdapter) transcribeWithEngine(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	manager := asrengine.Default()
	outputDir := procCtx.OutputDirectory
	if absOutput, err := filepath.Abs(outputDir); err == nil {
		outputDir = absOutput
	}
	inputPath := input.FilePath
	if absInput, err := filepath.Abs(inputPath); err == nil {
		inputPath = absInput
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare output directory: %w", err)
	}
	spec := pb.ModelSpec{
		ModelId:   "canary",
		ModelName: "nemo-canary-1b-v2",
	}
	if modelName := c.GetStringParameter(params, "model"); modelName != "" {
		if modelName == "nemo-canary-1b-v2" {
			spec.ModelName = modelName
		}
	}
	if modelPath := strings.TrimSpace(os.Getenv("ASR_ENGINE_CANARY_MODEL_PATH")); modelPath != "" {
		spec.ModelPath = modelPath
	}

	if err := manager.LoadModel(ctx, spec); err != nil {
		return nil, fmt.Errorf("failed to load canary model: %w", err)
	}

	engineParams := buildEngineParams(params)
	engineParams["model_family"] = "nvidia_canary"

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		manager.StopJob(context.Background(), procCtx.JobID)
	}()

	status, err := manager.RunJob(jobCtx, procCtx.JobID, inputPath, outputDir, engineParams)
	if err != nil {
		return nil, fmt.Errorf("canary engine job failed: %w", err)
	}
	if status.State == pb.JobState_JOB_STATE_FAILED {
		return nil, fmt.Errorf("canary engine failed: %s", status.Message)
	}
	if status.State == pb.JobState_JOB_STATE_CANCELLED {
		return nil, fmt.Errorf("canary transcription was cancelled")
	}

	transcriptPath := status.Outputs["transcript"]
	if transcriptPath == "" {
		return nil, fmt.Errorf("canary engine missing transcript output")
	}
	text, err := readEngineTranscript(transcriptPath)
	if err != nil {
		return nil, err
	}

	var segments []interfaces.TranscriptSegment
	if path := status.Outputs["segments"]; path != "" {
		segments, err = readEngineSegments(path)
		if err != nil {
			return nil, err
		}
	}

	var words []interfaces.TranscriptWord
	if path := status.Outputs["words"]; path != "" {
		words, err = readEngineWords(path)
		if err != nil {
			return nil, err
		}
	}

	resultLanguage := c.GetStringParameter(params, "language")
	if c.GetStringParameter(params, "task") == "translate" {
		if target := c.GetStringParameter(params, "target_language"); target != "" {
			resultLanguage = target
		}
	}
	if resultLanguage == "" {
		resultLanguage = "en"
	}

	for i := range segments {
		segments[i].Language = &resultLanguage
	}

	return &interfaces.TranscriptResult{
		Text:         text,
		Language:     resultLanguage,
		Segments:     segments,
		WordSegments: words,
		Confidence:   0.0,
	}, nil
}
