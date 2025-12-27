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

// SortformerAdapter implements the DiarizationAdapter interface for NVIDIA Sortformer
type SortformerAdapter struct {
	*BaseAdapter
	envPath string
}

// NewSortformerAdapter creates a new NVIDIA Sortformer diarization adapter
func NewSortformerAdapter(envPath string) *SortformerAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "sortformer",
		ModelFamily:        "nvidia_sortformer",
		DisplayName:        "NVIDIA Sortformer 4-Speaker v2",
		Description:        "NVIDIA's streaming Sortformer model optimized for 4-speaker diarization",
		Version:            "2.0.0",
		SupportedLanguages: []string{"*"}, // Language-agnostic
		SupportedFormats:   []string{"wav", "flac"},
		RequiresGPU:        false, // Optional GPU support
		MemoryRequirement:  3072,  // 3GB recommended
		Features: map[string]bool{
			"speaker_detection":    true,
			"streaming":            true,
			"optimized_4_speakers": true,
			"fast_processing":      true,
			"no_token_required":    true,
		},
		Metadata: map[string]string{
			"engine":       "nvidia_nemo",
			"framework":    "nemo_toolkit",
			"license":      "CC-BY-4.0",
			"optimization": "4_speakers",
			"sample_rate":  "16000",
			"format":       "16khz_mono_wav",
			"no_auth":      "true",
		},
	}

	schema := []interfaces.ParameterSchema{
		// Core settings
		{
			Name:        "max_speakers",
			Type:        "int",
			Required:    false,
			Default:     4,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{8}[0],
			Description: "Maximum number of speakers (optimized for 4)",
			Group:       "basic",
		},
		{
			Name:        "batch_size",
			Type:        "int",
			Required:    false,
			Default:     1,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{4}[0],
			Description: "Batch size for processing",
			Group:       "basic",
		},

		// Output settings
		{
			Name:        "output_format",
			Type:        "string",
			Required:    false,
			Default:     "rttm",
			Options:     []string{"rttm", OutputFormatJSON},
			Description: "Output format for diarization results",
			Group:       "advanced",
		},

		// Performance settings
		{
			Name:        "device",
			Type:        "string",
			Required:    false,
			Default:     "auto",
			Options:     []string{"cpu", "cuda", "auto"},
			Description: "Device to use for computation (cpu, cuda for NVIDIA GPUs, auto for automatic detection)",
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

		// Model-specific settings
		{
			Name:        "streaming_mode",
			Type:        "bool",
			Required:    false,
			Default:     false,
			Description: "Enable streaming processing mode",
			Group:       "advanced",
		},
		{
			Name:        "chunk_length_s",
			Type:        "float",
			Required:    false,
			Default:     30.0,
			Min:         &[]float64{5.0}[0],
			Max:         &[]float64{120.0}[0],
			Description: "Chunk length in seconds for streaming mode",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("sortformer", envPath, capabilities, schema)

	adapter := &SortformerAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}

	return adapter
}

// GetMaxSpeakers returns the maximum number of speakers Sortformer can handle
func (s *SortformerAdapter) GetMaxSpeakers() int {
	return 8 // Can handle more but optimized for 4
}

// GetMinSpeakers returns the minimum number of speakers Sortformer requires
func (s *SortformerAdapter) GetMinSpeakers() int {
	return 1
}

// PrepareEnvironment sets up the Sortformer environment (shared with NVIDIA models)
func (s *SortformerAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing NVIDIA Sortformer environment", "env_path", s.envPath)

	// Copy diarization script
	if err := s.copyDiarizationScript(); err != nil {
		return fmt.Errorf("failed to copy diarization script: %w", err)
	}

	// Check if environment is already ready (using cache to speed up repeated checks)
	if CheckEnvironmentReady(s.envPath, "from nemo.collections.asr.models import SortformerEncLabelModel") {
		modelPath := filepath.Join(s.envPath, "diar_streaming_sortformer_4spk-v2.nemo")
		if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
			scriptPath := filepath.Join(s.envPath, "sortformer_diarize.py")
			if _, err := os.Stat(scriptPath); err == nil {
				logger.Info("Sortformer environment already ready")
				s.initialized = true
				return nil
			}
		}
	}

	// Check if the shared environment exists (created by other NVIDIA adapters)
	pyprojectPath := filepath.Join(s.envPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err != nil {
		// Create environment if it doesn't exist
		if err := s.setupSortformerEnvironment(); err != nil {
			return fmt.Errorf("failed to setup Sortformer environment: %w", err)
		}
	}

	// Download model
	if err := s.downloadSortformerModel(); err != nil {
		return fmt.Errorf("failed to download Sortformer model: %w", err)
	}

	s.initialized = true
	logger.Info("Sortformer environment prepared successfully")
	return nil
}

// setupSortformerEnvironment creates the Python environment if it doesn't exist
func (s *SortformerAdapter) setupSortformerEnvironment() error {
	if err := os.MkdirAll(s.envPath, 0755); err != nil {
		return fmt.Errorf("failed to create sortformer directory: %w", err)
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

	pyprojectPath := filepath.Join(s.envPath, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run uv sync
	logger.Info("Installing Sortformer dependencies")
	cmd := exec.Command("uv", "sync", "--native-tls")
	cmd.Dir = s.envPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// downloadSortformerModel downloads the Sortformer model file
func (s *SortformerAdapter) downloadSortformerModel() error {
	modelFileName := "diar_streaming_sortformer_4spk-v2.nemo"
	modelPath := filepath.Join(s.envPath, modelFileName)

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		logger.Info("Sortformer model already exists", "path", modelPath, "size", stat.Size())
		return nil
	}

	logger.Info("Downloading Sortformer model", "path", modelPath)

	modelURL := "https://huggingface.co/nvidia/diar_streaming_sortformer_4spk-v2/resolve/main/diar_streaming_sortformer_4spk-v2.nemo?download=true"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := downloader.DownloadFile(ctx, modelURL, modelPath); err != nil {
		return fmt.Errorf("failed to download Sortformer model: %w", err)
	}

	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("downloaded model file not found: %w", err)
	}
	if stat.Size() < 1024*1024 {
		return fmt.Errorf("downloaded model file appears incomplete (size: %d bytes)", stat.Size())
	}

	logger.Info("Successfully downloaded Sortformer model", "size", stat.Size())
	return nil
}

// copyDiarizationScript creates the Python script for Sortformer diarization
func (s *SortformerAdapter) copyDiarizationScript() error {
	scriptContent, err := nvidiaScripts.ReadFile("py/nvidia/sortformer_diarize.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded sortformer_diarize.py: %w", err)
	}

	scriptPath := filepath.Join(s.envPath, "sortformer_diarize.py")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write diarization script: %w", err)
	}

	return nil
}

// Diarize processes audio using Sortformer
func (s *SortformerAdapter) Diarize(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	startTime := time.Now()
	s.LogProcessingStart(input, procCtx)
	defer func() {
		s.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Validate input
	if err := s.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Validate parameters
	if err := s.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Create temporary directory
	tempDir, err := s.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer s.CleanupTempDirectory(tempDir)

	// Convert audio if needed (Sortformer requires 16kHz mono WAV)
	audioInput := input
	if s.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := s.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	// Build command arguments
	args, err := s.buildSortformerArgs(audioInput, params, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Execute Sortformer
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

	logger.Info("Executing Sortformer command", "args", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("diarization was cancelled")
		}

		// Read tail of log file for context
		logPath := filepath.Join(procCtx.OutputDirectory, "transcription.log")
		logTail, readErr := s.ReadLogTail(logPath, 2048)
		if readErr != nil {
			logger.Warn("Failed to read log tail", "error", readErr)
		}

		logger.Error("Sortformer execution failed", "error", err)
		return nil, fmt.Errorf("Sortformer execution failed: %w\nLogs:\n%s", err, logTail)
	}

	// Parse result
	result, err := s.parseResult(tempDir, audioInput, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "diar_streaming_sortformer_4spk-v2"
	result.Metadata = s.CreateDefaultMetadata(params)

	logger.Info("Sortformer diarization completed",
		"segments", len(result.Segments),
		"speakers", result.SpeakerCount,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// buildSortformerArgs builds the command arguments for Sortformer
func (s *SortformerAdapter) buildSortformerArgs(input interfaces.AudioInput, params map[string]interface{}, tempDir string) ([]string, error) {
	outputFormat := s.GetStringParameter(params, "output_format")
	var outputFile string
	if outputFormat == OutputFormatJSON {
		outputFile = filepath.Join(tempDir, "result.json")
	} else {
		outputFile = filepath.Join(tempDir, "result.rttm")
	}

	scriptPath := filepath.Join(s.envPath, "sortformer_diarize.py")
	args := []string{
		"run", "--native-tls", "--project", s.envPath, "python", scriptPath,
		input.FilePath,
		outputFile,
	}

	// Add batch size
	if batchSize := s.GetIntParameter(params, "batch_size"); batchSize > 0 {
		args = append(args, "--batch-size", strconv.Itoa(batchSize))
	}

	// Add device
	if device := s.GetStringParameter(params, "device"); device != "" {
		args = append(args, "--device", device)
	}

	// Add max speakers
	if maxSpeakers := s.GetIntParameter(params, "max_speakers"); maxSpeakers > 0 {
		args = append(args, "--max-speakers", strconv.Itoa(maxSpeakers))
	}

	// Add output format
	args = append(args, "--output-format", outputFormat)

	// Add streaming mode
	if s.GetBoolParameter(params, "streaming_mode") {
		args = append(args, "--streaming")
		if chunkLength := s.GetFloatParameter(params, "chunk_length_s"); chunkLength > 0 {
			args = append(args, "--chunk-length-s", fmt.Sprintf("%.1f", chunkLength))
		}
	}

	return args, nil
}

// parseResult parses the Sortformer output
func (s *SortformerAdapter) parseResult(tempDir string, input interfaces.AudioInput, params map[string]interface{}) (*interfaces.DiarizationResult, error) {
	outputFormat := s.GetStringParameter(params, "output_format")

	if outputFormat == OutputFormatJSON {
		return s.parseJSONResult(tempDir)
	}
	return s.parseRTTMResult(tempDir, input)
}

// parseJSONResult parses JSON format output
func (s *SortformerAdapter) parseJSONResult(tempDir string) (*interfaces.DiarizationResult, error) {
	resultFile := filepath.Join(tempDir, "result.json")

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var sortformerResult struct {
		AudioFile string `json:"audio_file"`
		Model     string `json:"model"`
		Segments  []struct {
			Start      float64 `json:"start"`
			End        float64 `json:"end"`
			Speaker    string  `json:"speaker"`
			Confidence float64 `json:"confidence"`
			Duration   float64 `json:"duration"`
		} `json:"segments"`
		Speakers      []string `json:"speakers"`
		SpeakerCount  int      `json:"speaker_count"`
		TotalSegments int      `json:"total_segments"`
		TotalDuration float64  `json:"total_duration"`
	}

	if err := json.Unmarshal(data, &sortformerResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	// Convert to standard format
	result := &interfaces.DiarizationResult{
		Segments:     make([]interfaces.DiarizationSegment, len(sortformerResult.Segments)),
		SpeakerCount: sortformerResult.SpeakerCount,
		Speakers:     sortformerResult.Speakers,
	}

	for i, seg := range sortformerResult.Segments {
		result.Segments[i] = interfaces.DiarizationSegment{
			Start:      seg.Start,
			End:        seg.End,
			Speaker:    seg.Speaker,
			Confidence: seg.Confidence,
		}
	}

	return result, nil
}

// parseRTTMResult parses RTTM format output
func (s *SortformerAdapter) parseRTTMResult(tempDir string, input interfaces.AudioInput) (*interfaces.DiarizationResult, error) {
	resultFile := filepath.Join(tempDir, "result.rttm")

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var segments []interfaces.DiarizationSegment
	speakers := make(map[string]bool)

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "SPEAKER") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}

		start, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			continue
		}

		duration, err := strconv.ParseFloat(parts[4], 64)
		if err != nil {
			continue
		}

		end := start + duration
		speaker := parts[7]
		speakers[speaker] = true

		segments = append(segments, interfaces.DiarizationSegment{
			Start:      start,
			End:        end,
			Speaker:    speaker,
			Confidence: 1.0, // RTTM doesn't include confidence scores
		})
	}

	// Convert speakers map to slice
	speakerList := make([]string, 0, len(speakers))
	for speaker := range speakers {
		speakerList = append(speakerList, speaker)
	}

	result := &interfaces.DiarizationResult{
		Segments:     segments,
		SpeakerCount: len(speakers),
		Speakers:     speakerList,
	}

	return result, nil
}

// GetEstimatedProcessingTime provides Sortformer-specific time estimation
func (s *SortformerAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Sortformer is typically very fast, often faster than real-time
	baseTime := s.BaseAdapter.GetEstimatedProcessingTime(input)

	// Sortformer typically processes at about 5-10% of audio duration
	return time.Duration(float64(baseTime) * 0.3)
}
