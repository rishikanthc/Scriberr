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
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

// WhisperXAdapter implements the TranscriptionAdapter interface for WhisperX
type WhisperXAdapter struct {
	*BaseAdapter
	envPath string
}

// NewWhisperXAdapter creates a new WhisperX adapter
func NewWhisperXAdapter() *WhisperXAdapter {
	envPath := "whisperx-env"
	
	capabilities := interfaces.ModelCapabilities{
		ModelID:     "whisperx",
		ModelFamily: "whisper",
		DisplayName: "WhisperX",
		Description: "OpenAI Whisper with speaker diarization and word-level timestamps",
		Version:     "3.0.0",
		SupportedLanguages: []string{
			"en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr", "pl", "ca", "nl",
			"ar", "sv", "it", "id", "hi", "fi", "vi", "he", "uk", "el", "ms", "cs", "ro",
			"da", "hu", "ta", "no", "th", "ur", "hr", "bg", "lt", "la", "mi", "ml", "cy",
			"sk", "te", "fa", "lv", "bn", "sr", "az", "sl", "kn", "et", "mk", "br", "eu",
			"is", "hy", "ne", "mn", "bs", "kk", "sq", "sw", "gl", "mr", "pa", "si", "km",
			"sn", "yo", "so", "af", "oc", "ka", "be", "tg", "sd", "gu", "am", "yi", "lo",
			"uz", "fo", "ht", "ps", "tk", "nn", "mt", "sa", "lb", "my", "bo", "tl", "mg",
			"as", "tt", "haw", "ln", "ha", "ba", "jw", "su", "auto",
		},
		SupportedFormats: []string{"wav", "mp3", "flac", "m4a", "ogg", "wma"},
		RequiresGPU:      false, // Optional GPU support
		MemoryRequirement: 2048, // 2GB base requirement
		Features: map[string]bool{
			"timestamps":     true,
			"word_level":     true,
			"diarization":    true,
			"translation":    true,
			"language_detection": true,
			"vad":            true,
		},
		Metadata: map[string]string{
			"engine":      "openai_whisper",
			"framework":   "transformers",
			"license":     "MIT",
			"python_env":  "whisperx",
		},
	}

	schema := []interfaces.ParameterSchema{
		// Model selection
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     "small",
			Options:     []string{"tiny", "tiny.en", "base", "base.en", "small", "small.en", "medium", "medium.en", "large", "large-v1", "large-v2", "large-v3"},
			Description: "Whisper model size to use",
			Group:       "basic",
		},
		
		// Device and computation
		{
			Name:        "device",
			Type:        "string",
			Required:    false,
			Default:     "cpu",
			Options:     []string{"cpu", "cuda"},
			Description: "Device to use for computation",
			Group:       "basic",
		},
		{
			Name:        "device_index",
			Type:        "int",
			Required:    false,
			Default:     0,
			Min:         &[]float64{0}[0],
			Max:         &[]float64{7}[0],
			Description: "GPU device index to use",
			Group:       "advanced",
		},
		{
			Name:        "batch_size",
			Type:        "int",
			Required:    false,
			Default:     8,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{64}[0],
			Description: "Batch size for processing",
			Group:       "advanced",
		},
		{
			Name:        "compute_type",
			Type:        "string",
			Required:    false,
			Default:     "float32",
			Options:     []string{"float16", "float32", "int8"},
			Description: "Computation precision",
			Group:       "advanced",
		},
		{
			Name:        "threads",
			Type:        "int",
			Required:    false,
			Default:     0,
			Min:         &[]float64{0}[0],
			Max:         &[]float64{32}[0],
			Description: "Number of CPU threads (0 = auto)",
			Group:       "advanced",
		},

		// Language and task
		{
			Name:        "language",
			Type:        "string",
			Required:    false,
			Default:     nil,
			Description: "Language code (auto-detect if not specified)",
			Group:       "basic",
		},
		{
			Name:        "task",
			Type:        "string",
			Required:    false,
			Default:     "transcribe",
			Options:     []string{"transcribe", "translate"},
			Description: "Task to perform",
			Group:       "basic",
		},

		// Diarization
		{
			Name:        "diarize",
			Type:        "bool",
			Required:    false,
			Default:     false,
			Description: "Enable speaker diarization",
			Group:       "basic",
		},
		{
			Name:        "diarize_model",
			Type:        "string",
			Required:    false,
			Default:     "pyannote/speaker-diarization-3.1",
			Options:     []string{"pyannote/speaker-diarization-3.1", "pyannote"},
			Description: "Diarization model to use",
			Group:       "advanced",
		},
		{
			Name:        "min_speakers",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{20}[0],
			Description: "Minimum number of speakers",
			Group:       "advanced",
		},
		{
			Name:        "max_speakers",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{20}[0],
			Description: "Maximum number of speakers",
			Group:       "advanced",
		},
		{
			Name:        "hf_token",
			Type:        "string",
			Required:    false,
			Default:     nil,
			Description: "HuggingFace token for diarization models",
			Group:       "advanced",
		},

		// Quality settings
		{
			Name:        "temperature",
			Type:        "float",
			Required:    false,
			Default:     0.0,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "Sampling temperature",
			Group:       "quality",
		},
		{
			Name:        "best_of",
			Type:        "int",
			Required:    false,
			Default:     5,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{10}[0],
			Description: "Number of candidates to consider",
			Group:       "quality",
		},
		{
			Name:        "beam_size",
			Type:        "int",
			Required:    false,
			Default:     5,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{10}[0],
			Description: "Beam search size",
			Group:       "quality",
		},
		{
			Name:        "patience",
			Type:        "float",
			Required:    false,
			Default:     1.0,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{2.0}[0],
			Description: "Beam search patience",
			Group:       "quality",
		},

		// VAD settings
		{
			Name:        "vad_method",
			Type:        "string",
			Required:    false,
			Default:     "pyannote",
			Options:     []string{"pyannote", "silero"},
			Description: "Voice activity detection method",
			Group:       "advanced",
		},
		{
			Name:        "vad_onset",
			Type:        "float",
			Required:    false,
			Default:     0.5,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "VAD onset threshold",
			Group:       "advanced",
		},
		{
			Name:        "vad_offset",
			Type:        "float",
			Required:    false,
			Default:     0.363,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "VAD offset threshold",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("whisperx", filepath.Join(envPath, "WhisperX"), capabilities, schema)
	
	adapter := &WhisperXAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetSupportedModels returns the list of Whisper models supported
func (w *WhisperXAdapter) GetSupportedModels() []string {
	return []string{
		"tiny", "tiny.en",
		"base", "base.en", 
		"small", "small.en",
		"medium", "medium.en",
		"large", "large-v1", "large-v2", "large-v3",
	}
}

// PrepareEnvironment sets up the WhisperX environment
func (w *WhisperXAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing WhisperX environment", "env_path", w.envPath)

	whisperxPath := filepath.Join(w.envPath, "WhisperX")
	
	// Check if WhisperX is already set up and working
	cmd := exec.Command("uv", "run", "--native-tls", "--project", whisperxPath, "python", "-c", "import whisperx")
	if cmd.Run() == nil {
		logger.Info("WhisperX environment already ready")
		w.initialized = true
		return nil
	}

	// Ensure base directory exists
	if err := os.MkdirAll(w.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}

	// Clone WhisperX
	if err := w.cloneWhisperX(); err != nil {
		return fmt.Errorf("failed to clone WhisperX: %w", err)
	}

	// Update dependencies
	if err := w.updateWhisperXDependencies(whisperxPath); err != nil {
		return fmt.Errorf("failed to update WhisperX dependencies: %w", err)
	}

	// Install dependencies
	if err := w.uvSyncWhisperX(whisperxPath); err != nil {
		return fmt.Errorf("failed to sync WhisperX: %w", err)
	}

	w.initialized = true
	logger.Info("WhisperX environment prepared successfully")
	return nil
}

// cloneWhisperX clones the WhisperX repository
func (w *WhisperXAdapter) cloneWhisperX() error {
	cmd := exec.Command("git", "clone", "https://github.com/m-bain/WhisperX.git")
	cmd.Dir = w.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// updateWhisperXDependencies modifies WhisperX pyproject.toml
func (w *WhisperXAdapter) updateWhisperXDependencies(whisperxPath string) error {
	pyprojectPath := filepath.Join(whisperxPath, "pyproject.toml")

	data, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return fmt.Errorf("failed to read pyproject.toml: %w", err)
	}

	content := string(data)
	content = strings.ReplaceAll(content, "ctranslate2<4.5.0", "ctranslate2==4.6.0")

	if !strings.Contains(content, "yt-dlp") {
		content = strings.ReplaceAll(content,
			`"transformers>=4.48.0",`,
			`"transformers>=4.48.0",
    "yt-dlp",`)
	}

	if err := os.WriteFile(pyprojectPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	return nil
}

// uvSyncWhisperX runs uv sync for WhisperX
func (w *WhisperXAdapter) uvSyncWhisperX(whisperxPath string) error {
	cmd := exec.Command("uv", "sync", "--all-extras", "--dev", "--native-tls")
	cmd.Dir = whisperxPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Transcribe processes audio using WhisperX
func (w *WhisperXAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	w.LogProcessingStart(input, procCtx)
	defer func() {
		w.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := w.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := w.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := w.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer w.CleanupTempDirectory(tempDir)

	// Build WhisperX command
	args, err := w.buildWhisperXArgs(input, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute WhisperX
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	logger.Info("Executing WhisperX command", "args", strings.Join(args, " "))
	
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("transcription was cancelled")
	}
	if err != nil {
		logger.Error("WhisperX execution failed", "output", string(output), "error", err)
		return nil, fmt.Errorf("WhisperX execution failed: %w", err)
	}

	// Parse result
	result, err := w.parseResult(tempDir, input, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = w.GetStringParameter(params, "model")
	result.Metadata = w.CreateDefaultMetadata(params)

	logger.Info("WhisperX transcription completed", 
		"segments", len(result.Segments),
		"words", len(result.Words),
		"processing_time", result.ProcessingTime)

	return result, nil
}

// buildWhisperXArgs builds the command arguments for WhisperX
func (w *WhisperXAdapter) buildWhisperXArgs(input interfaces.AudioInput, params map[string]interface{}, outputDir string) ([]string, error) {
	whisperxPath := filepath.Join(w.envPath, "WhisperX")
	
	args := []string{
		"run", "--native-tls", "--project", whisperxPath, "python", "-m", "whisperx",
		input.FilePath,
		"--output_dir", outputDir,
	}

	// Core parameters
	args = append(args, "--model", w.GetStringParameter(params, "model"))
	args = append(args, "--device", w.GetStringParameter(params, "device"))
	args = append(args, "--device_index", strconv.Itoa(w.GetIntParameter(params, "device_index")))
	args = append(args, "--batch_size", strconv.Itoa(w.GetIntParameter(params, "batch_size")))
	args = append(args, "--compute_type", w.GetStringParameter(params, "compute_type"))
	
	if threads := w.GetIntParameter(params, "threads"); threads > 0 {
		args = append(args, "--threads", strconv.Itoa(threads))
	}

	// Output settings
	args = append(args, "--output_format", "all")
	args = append(args, "--verbose", "True")

	// Task and language
	args = append(args, "--task", w.GetStringParameter(params, "task"))
	if language := w.GetStringParameter(params, "language"); language != "" {
		args = append(args, "--language", language)
	}

	// VAD settings
	args = append(args, "--vad_method", w.GetStringParameter(params, "vad_method"))
	args = append(args, "--vad_onset", fmt.Sprintf("%.3f", w.GetFloatParameter(params, "vad_onset")))
	args = append(args, "--vad_offset", fmt.Sprintf("%.3f", w.GetFloatParameter(params, "vad_offset")))

	// Diarization
	if w.GetBoolParameter(params, "diarize") {
		args = append(args, "--diarize")
		
		diarizeModel := w.GetStringParameter(params, "diarize_model")
		if diarizeModel == "pyannote" {
			diarizeModel = "pyannote/speaker-diarization-3.1"
		}
		args = append(args, "--diarize_model", diarizeModel)
		
		if minSpeakers := w.GetIntParameter(params, "min_speakers"); minSpeakers > 0 {
			args = append(args, "--min_speakers", strconv.Itoa(minSpeakers))
		}
		if maxSpeakers := w.GetIntParameter(params, "max_speakers"); maxSpeakers > 0 {
			args = append(args, "--max_speakers", strconv.Itoa(maxSpeakers))
		}
	}

	// Quality settings
	args = append(args, "--temperature", fmt.Sprintf("%.2f", w.GetFloatParameter(params, "temperature")))
	args = append(args, "--best_of", strconv.Itoa(w.GetIntParameter(params, "best_of")))
	args = append(args, "--beam_size", strconv.Itoa(w.GetIntParameter(params, "beam_size")))
	args = append(args, "--patience", fmt.Sprintf("%.2f", w.GetFloatParameter(params, "patience")))

	// HuggingFace token
	if hfToken := w.GetStringParameter(params, "hf_token"); hfToken != "" {
		args = append(args, "--hf_token", hfToken)
	}

	// Disable print progress for cleaner output
	args = append(args, "--print_progress", "False")

	return args, nil
}

// parseResult parses the WhisperX output files
func (w *WhisperXAdapter) parseResult(outputDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.TranscriptResult, error) {
	// Find JSON result files
	files, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find result files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no result files found in %s", outputDir)
	}

	// Use the first JSON file found
	resultFile := files[0]
	
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	// Parse WhisperX JSON format
	var whisperxResult struct {
		Segments []struct {
			Start   float64 `json:"start"`
			End     float64 `json:"end"`
			Text    string  `json:"text"`
			Speaker *string `json:"speaker,omitempty"`
		} `json:"segments"`
		Word []struct {
			Start   float64 `json:"start"`
			End     float64 `json:"end"`
			Word    string  `json:"word"`
			Score   float64 `json:"score"`
			Speaker *string `json:"speaker,omitempty"`
		} `json:"word_segments,omitempty"`
		Language string `json:"language"`
		Text     string `json:"text,omitempty"`
	}

	if err := json.Unmarshal(data, &whisperxResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Convert to standard format
	result := &interfaces.TranscriptResult{
		Language:   whisperxResult.Language,
		Segments:   make([]interfaces.TranscriptSegment, len(whisperxResult.Segments)),
		Words:      make([]interfaces.TranscriptWord, len(whisperxResult.Word)),
		Confidence: 0.0, // WhisperX doesn't provide overall confidence
	}

	// Convert segments
	var textParts []string
	for i, seg := range whisperxResult.Segments {
		result.Segments[i] = interfaces.TranscriptSegment{
			Start:   seg.Start,
			End:     seg.End,
			Text:    seg.Text,
			Speaker: seg.Speaker,
		}
		textParts = append(textParts, seg.Text)
	}

	// Convert words
	for i, word := range whisperxResult.Word {
		result.Words[i] = interfaces.TranscriptWord{
			Start:   word.Start,
			End:     word.End,
			Word:    word.Word,
			Score:   word.Score,
			Speaker: word.Speaker,
		}
	}

	// Set full text
	if whisperxResult.Text != "" {
		result.Text = whisperxResult.Text
	} else {
		result.Text = strings.Join(textParts, " ")
	}

	return result, nil
}

// GetEstimatedProcessingTime provides WhisperX-specific time estimation
func (w *WhisperXAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// WhisperX processing time varies by model size
	baseTime := w.BaseAdapter.GetEstimatedProcessingTime(input)
	
	// Adjust based on model size (if we can determine it)
	// This would need model size information from parameters
	// For now, use base estimation
	return baseTime
}

// init registers the WhisperX adapter
func init() {
	registry.RegisterTranscriptionAdapter("whisperx", NewWhisperXAdapter())
}