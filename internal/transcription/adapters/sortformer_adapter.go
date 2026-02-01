package adapters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/asrengine/pb"
	"scriberr/internal/diarengine"
	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

// SortformerAdapter implements the DiarizationAdapter interface for NVIDIA Sortformer via the diarization engine.
type SortformerAdapter struct {
	*BaseAdapter
}

// NewSortformerAdapter creates a new NVIDIA Sortformer diarization adapter
func NewSortformerAdapter(_ string) *SortformerAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "sortformer",
		ModelFamily:        "nvidia_sortformer",
		DisplayName:        "NVIDIA Sortformer Diarization",
		Description:        "NVIDIA's Sortformer model optimized for 4-speaker diarization",
		Version:            "diar_streaming_sortformer_4spk-v2",
		SupportedLanguages: []string{"*"},
		SupportedFormats:   []string{"wav", "mp3", "flac", "m4a", "ogg"},
		RequiresGPU:        true,
		MemoryRequirement:  4096,
		Features: map[string]bool{
			"speaker_detection":   true,
			"speaker_constraints": true,
			"confidence_scores":   true,
			"rttm_output":         true,
			"fast_inference":      true,
		},
		Metadata: map[string]string{
			"engine":    "nemo_sortformer",
			"framework": "pytorch",
			"license":   "NVIDIA",
			"model_hub": "huggingface",
		},
	}

	schema := []interfaces.ParameterSchema{
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
		{
			Name:        "output_format",
			Type:        "string",
			Required:    false,
			Default:     "rttm",
			Options:     []string{"rttm", OutputFormatJSON},
			Description: "Output format for diarization results",
			Group:       "advanced",
		},
		{
			Name:        "device",
			Type:        "string",
			Required:    false,
			Default:     "auto",
			Options:     []string{"cpu", "cuda", "auto"},
			Description: "Device to use for computation (cpu, cuda for NVIDIA GPUs, auto for automatic detection)",
			Group:       "advanced",
		},
		{
			Name:        "streaming_mode",
			Type:        "bool",
			Required:    false,
			Default:     false,
			Description: "Enable streaming mode for low latency",
			Group:       "advanced",
		},
		{
			Name:        "chunk_length_s",
			Type:        "float",
			Required:    false,
			Default:     30.0,
			Min:         &[]float64{5.0}[0],
			Max:         &[]float64{120.0}[0],
			Description: "Chunk length in seconds for streaming",
			Group:       "advanced",
		},
		{
			Name:        "chunk_len",
			Type:        "int",
			Required:    false,
			Default:     340,
			Min:         &[]float64{40}[0],
			Max:         &[]float64{1024}[0],
			Description: "Chunk length in frames (streaming)",
			Group:       "advanced",
		},
		{
			Name:        "chunk_right_context",
			Type:        "int",
			Required:    false,
			Default:     40,
			Min:         &[]float64{20}[0],
			Max:         &[]float64{200}[0],
			Description: "Right context frames (streaming)",
			Group:       "advanced",
		},
		{
			Name:        "fifo_len",
			Type:        "int",
			Required:    false,
			Default:     40,
			Min:         &[]float64{10}[0],
			Max:         &[]float64{200}[0],
			Description: "FIFO length for streaming cache",
			Group:       "advanced",
		},
		{
			Name:        "spkcache_update_period",
			Type:        "int",
			Required:    false,
			Default:     300,
			Min:         &[]float64{50}[0],
			Max:         &[]float64{1000}[0],
			Description: "Speaker cache update period",
			Group:       "advanced",
		},
		{
			Name:        "auto_convert_audio",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Automatically convert audio to 16kHz mono WAV",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("sortformer", "", capabilities, schema)
	return &SortformerAdapter{BaseAdapter: baseAdapter}
}

func (s *SortformerAdapter) GetMaxSpeakers() int {
	return 8
}

func (s *SortformerAdapter) GetMinSpeakers() int {
	return 1
}

func (s *SortformerAdapter) PrepareEnvironment(ctx context.Context) error {
	if err := diarengine.Default().EnsureRunning(ctx); err != nil {
		return fmt.Errorf("failed to start diarization engine: %w", err)
	}
	s.initialized = true
	return nil
}

func (s *SortformerAdapter) Diarize(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	startTime := time.Now()
	s.LogProcessingStart(input, procCtx)
	defer func() {
		s.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	if err := s.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}
	if err := s.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	tempDir, err := s.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer s.CleanupTempDirectory(tempDir)

	audioInput := input
	if s.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := s.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	result, err := s.diarizeWithEngine(ctx, audioInput, params, procCtx)
	if err != nil {
		return nil, err
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = "nvidia/diar_streaming_sortformer_4spk-v2"
	result.Metadata = s.CreateDefaultMetadata(params)

	logger.Info("Sortformer diarization completed",
		"segments", len(result.Segments),
		"speakers", result.SpeakerCount,
		"processing_time", result.ProcessingTime)

	return result, nil
}

func (s *SortformerAdapter) diarizeWithEngine(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	manager := diarengine.Default()
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
		ModelId:   "sortformer",
		ModelName: "nvidia/diar_streaming_sortformer_4spk-v2",
	}
	if modelPath := strings.TrimSpace(os.Getenv("DIAR_ENGINE_SORTFORMER_MODEL_PATH")); modelPath != "" {
		spec.ModelPath = modelPath
	}

	if err := manager.LoadModel(ctx, spec); err != nil {
		return nil, fmt.Errorf("failed to load sortformer model: %w", err)
	}
	defer func() {
		_ = manager.UnloadModel(context.Background(), spec.ModelId)
	}()

	engineParams := buildDiarEngineParams(s.BaseAdapter, params)
	engineParams["model_family"] = "nvidia_sortformer"

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		manager.StopJob(context.Background(), procCtx.JobID)
	}()

	status, err := manager.RunJob(jobCtx, procCtx.JobID, inputPath, outputDir, engineParams)
	if err != nil {
		return nil, fmt.Errorf("sortformer engine job failed: %w", err)
	}
	if status.State == pb.JobState_JOB_STATE_FAILED {
		return nil, fmt.Errorf("sortformer engine failed: %s", status.Message)
	}
	if status.State == pb.JobState_JOB_STATE_CANCELLED {
		return nil, fmt.Errorf("sortformer diarization was cancelled")
	}

	resultPath := status.Outputs["diarization"]
	if resultPath == "" {
		return nil, fmt.Errorf("sortformer engine missing diarization output")
	}

	return parseDiarizationJSON(resultPath)
}
