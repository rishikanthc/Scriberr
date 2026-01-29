package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/asrengine"
	"scriberr/internal/asrengine/pb"
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
		DisplayName:        "NVIDIA Parakeet TDT 0.6B",
		Description:        "NVIDIA Parakeet TDT 0.6B models (v2 English, v3 multilingual) with timestamps",
		Version:            "0.6.3",
		SupportedLanguages: []string{"en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"},
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
			"language":    "multilingual",
			"sample_rate": "16000",
			"format":      "16khz_mono_wav",
		},
	}

	schema := []interfaces.ParameterSchema{
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     "nemo-parakeet-tdt-0.6b-v3",
			Options:     []string{"nemo-parakeet-tdt-0.6b-v2", "nemo-parakeet-tdt-0.6b-v3"},
			Description: "Parakeet model variant",
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
	return []string{"nemo-parakeet-tdt-0.6b-v2", "nemo-parakeet-tdt-0.6b-v3"}
}

// PrepareEnvironment sets up the Parakeet environment
func (p *ParakeetAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Parakeet environment", "env_path", p.envPath)
	if err := asrengine.Default().EnsureRunning(ctx); err != nil {
		return fmt.Errorf("failed to start ASR engine: %w", err)
	}
	p.initialized = true
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
	// Ensure directory exists before writing script
	if err := os.MkdirAll(p.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

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
	result, err := p.transcribeWithEngine(ctx, audioInput, params, procCtx)
	if err != nil {
		return nil, err
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = p.GetStringParameter(params, "model")
	if result.ModelUsed == "" {
		result.ModelUsed = "nemo-parakeet-tdt-0.6b-v3"
	}
	result.Metadata = p.CreateDefaultMetadata(params)

	logger.Info("Parakeet transcription completed",
		"segments", len(result.Segments),
		"words", len(result.WordSegments),
		"processing_time", result.ProcessingTime)

	return result, nil
}

func (p *ParakeetAdapter) transcribeWithEngine(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	manager := asrengine.Default()
	if err := os.MkdirAll(procCtx.OutputDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare output directory: %w", err)
	}
	spec := pb.ModelSpec{
		ModelId:   "parakeet",
		ModelName: "nemo-parakeet-tdt-0.6b-v3",
	}
	if modelName := p.GetStringParameter(params, "model"); modelName != "" {
		if modelName == "nemo-parakeet-tdt-0.6b-v2" || modelName == "nemo-parakeet-tdt-0.6b-v3" {
			spec.ModelName = modelName
		}
	}
	if modelPath := strings.TrimSpace(os.Getenv("ASR_ENGINE_PARAKEET_MODEL_PATH")); modelPath != "" {
		spec.ModelPath = modelPath
	}

	if err := manager.LoadModel(ctx, spec); err != nil {
		return nil, fmt.Errorf("failed to load parakeet model: %w", err)
	}

	engineParams := buildEngineParams(params)
	engineParams["model_family"] = "nvidia_parakeet"

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		manager.StopJob(context.Background(), procCtx.JobID)
	}()

	status, err := manager.RunJob(jobCtx, procCtx.JobID, input.FilePath, procCtx.OutputDirectory, engineParams)
	if err != nil {
		return nil, fmt.Errorf("parakeet engine job failed: %w", err)
	}
	if status.State == pb.JobState_JOB_STATE_FAILED {
		return nil, fmt.Errorf("parakeet engine failed: %s", status.Message)
	}
	if status.State == pb.JobState_JOB_STATE_CANCELLED {
		return nil, fmt.Errorf("parakeet transcription was cancelled")
	}

	transcriptPath := status.Outputs["transcript"]
	if transcriptPath == "" {
		return nil, fmt.Errorf("parakeet engine missing transcript output")
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

	language := p.GetStringParameter(params, "language")
	if language == "" {
		language = "en"
	}

	return &interfaces.TranscriptResult{
		Text:         text,
		Language:     language,
		Segments:     segments,
		WordSegments: words,
		Confidence:   0.0,
	}, nil
}
