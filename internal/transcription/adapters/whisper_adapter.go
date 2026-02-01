package adapters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/asrengine"
	"scriberr/internal/asrengine/pb"
	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

const defaultWhisperOnnxModel = "onnx-community/whisper-small"

var whisperOnnxModelOptions = []string{
	"whisper-ort",
	"whisper-base-ort",
	"whisper-base",
	"onnx-community/whisper-tiny",
	"onnx-community/whisper-base",
	"onnx-community/whisper-small",
	"onnx-community/whisper-medium",
	"onnx-community/whisper-large-v2",
	"onnx-community/whisper-large-v3",
	"onnx-community/whisper-large-v3-turbo",
	"tiny",
	"tiny.en",
	"base",
	"base.en",
	"small",
	"small.en",
	"medium",
	"medium.en",
	"large",
	"large-v1",
	"large-v2",
	"large-v3",
}

var whisperModelAliases = map[string]string{
	"tiny":      "onnx-community/whisper-tiny",
	"tiny.en":   "onnx-community/whisper-tiny",
	"base":      "onnx-community/whisper-base",
	"base.en":   "onnx-community/whisper-base",
	"small":     "onnx-community/whisper-small",
	"small.en":  "onnx-community/whisper-small",
	"medium":    "onnx-community/whisper-medium",
	"medium.en": "onnx-community/whisper-medium",
	"large":     "onnx-community/whisper-large-v3",
	"large-v1":  "onnx-community/whisper-large-v2",
	"large-v2":  "onnx-community/whisper-large-v2",
	"large-v3":  "onnx-community/whisper-large-v3",
}

// WhisperAdapter implements the TranscriptionAdapter interface using the ASR engine.
type WhisperAdapter struct {
	*BaseAdapter
}

// NewWhisperAdapter creates a new Whisper adapter backed by the ASR engine.
func NewWhisperAdapter(_ string) *WhisperAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:     "whisper",
		ModelFamily: "whisper",
		DisplayName: "Whisper (ONNX)",
		Description: "OpenAI Whisper models via ASR engine (onnx-asr under the hood)",
		Version:     "onnx-asr",
		SupportedLanguages: []string{
			"auto", "en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr", "pl", "ca", "nl",
			"ar", "sv", "it", "id", "hi", "fi", "vi", "he", "uk", "el", "ms", "cs", "ro",
			"da", "hu", "ta", "no", "th", "ur", "hr", "bg", "lt", "la", "mi", "ml", "cy",
			"sk", "te", "fa", "lv", "bn", "sr", "az", "sl", "kn", "et", "mk", "br", "eu",
			"is", "hy", "ne", "mn", "bs", "kk", "sq", "sw", "gl", "mr", "pa", "si", "km",
			"sn", "yo", "so", "af", "oc", "ka", "be", "tg", "sd", "gu", "am", "yi", "lo",
			"uz", "fo", "ht", "ps", "tk", "nn", "mt", "sa", "lb", "my", "bo", "tl", "mg",
			"as", "tt", "haw", "ln", "ha", "ba", "jw", "su",
		},
		SupportedFormats:  []string{"wav", "flac", "mp3", "m4a", "ogg"},
		RequiresGPU:       false,
		MemoryRequirement: 2048,
		Features: map[string]bool{
			"timestamps":         true,
			"word_level":         true,
			"language_detection": true,
			"vad":                true,
		},
		Metadata: map[string]string{
			"engine":    "onnx-asr",
			"framework": "onnxruntime",
			"license":   "MIT",
		},
	}

	schema := []interfaces.ParameterSchema{
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     defaultWhisperOnnxModel,
			Options:     whisperOnnxModelOptions,
			Description: "Whisper ONNX model name (onnx-community/whisper-*)",
			Group:       "basic",
		},
		{
			Name:        "language",
			Type:        "string",
			Required:    false,
			Default:     "auto",
			Description: "Spoken language (auto for detection)",
			Group:       "basic",
		},
		{
			Name:        "timestamps",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Include segment and word-level timestamps",
			Group:       "basic",
		},
		{
			Name:        "auto_convert_audio",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Convert audio to 16kHz mono WAV before recognition",
			Group:       "advanced",
		},
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

	baseAdapter := NewBaseAdapter("whisper", "", capabilities, schema)
	return &WhisperAdapter{BaseAdapter: baseAdapter}
}

// GetSupportedModels returns the Whisper ONNX model names we expose.
func (w *WhisperAdapter) GetSupportedModels() []string {
	return whisperOnnxModelOptions
}

// PrepareEnvironment ensures the ASR engine is running.
func (w *WhisperAdapter) PrepareEnvironment(ctx context.Context) error {
	if err := asrengine.Default().EnsureRunning(ctx); err != nil {
		return fmt.Errorf("failed to start ASR engine: %w", err)
	}
	w.initialized = true
	return nil
}

// Transcribe processes audio using Whisper ONNX via the ASR engine.
func (w *WhisperAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	w.LogProcessingStart(input, procCtx)
	defer func() {
		w.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	if err := w.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}
	if err := w.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	tempDir, err := w.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer w.CleanupTempDirectory(tempDir)

	audioInput := input
	if w.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := w.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	result, err := w.transcribeWithEngine(ctx, audioInput, params, procCtx)
	if err != nil {
		return nil, err
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = w.GetStringParameter(params, "model")
	if result.ModelUsed == "" {
		result.ModelUsed = defaultWhisperOnnxModel
	}
	result.Metadata = w.CreateDefaultMetadata(params)

	logger.Info("Whisper transcription completed",
		"segments", len(result.Segments),
		"words", len(result.WordSegments),
		"processing_time", result.ProcessingTime)

	return result, nil
}

func (w *WhisperAdapter) transcribeWithEngine(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
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

	modelName := normalizeWhisperModelName(w.GetStringParameter(params, "model"))
	spec := pb.ModelSpec{
		ModelId:   "whisper",
		ModelName: modelName,
	}
	if modelPath := strings.TrimSpace(os.Getenv("ASR_ENGINE_WHISPER_MODEL_PATH")); modelPath != "" {
		spec.ModelPath = modelPath
	}

	if err := manager.LoadModel(ctx, spec); err != nil {
		return nil, fmt.Errorf("failed to load whisper model: %w", err)
	}
	defer func() {
		_ = manager.UnloadModel(context.Background(), spec.ModelId)
	}()

	engineParams := buildEngineParams(params)
	engineParams["model_family"] = "whisper"

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		manager.StopJob(context.Background(), procCtx.JobID)
	}()

	status, err := manager.RunJob(jobCtx, procCtx.JobID, inputPath, outputDir, engineParams)
	if err != nil {
		return nil, fmt.Errorf("whisper engine job failed: %w", err)
	}
	if status.State == pb.JobState_JOB_STATE_FAILED {
		return nil, fmt.Errorf("whisper engine failed: %s", status.Message)
	}
	if status.State == pb.JobState_JOB_STATE_CANCELLED {
		return nil, fmt.Errorf("whisper transcription was cancelled")
	}

	transcriptPath := status.Outputs["transcript"]
	if transcriptPath == "" {
		return nil, fmt.Errorf("whisper engine missing transcript output")
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

	language := w.GetStringParameter(params, "language")
	if language == "" {
		language = "auto"
	}

	return &interfaces.TranscriptResult{
		Text:         text,
		Language:     language,
		Segments:     segments,
		WordSegments: words,
		Confidence:   0.0,
	}, nil
}

func normalizeWhisperModelName(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return defaultWhisperOnnxModel
	}
	if mapped, ok := whisperModelAliases[strings.ToLower(clean)]; ok {
		return mapped
	}
	return clean
}
