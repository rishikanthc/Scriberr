package adapters

import (
	"context"
	"encoding/json"
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

const OutputFormatJSON = "json"

// PyAnnoteAdapter implements the DiarizationAdapter interface for PyAnnote via the diarization engine.
type PyAnnoteAdapter struct {
	*BaseAdapter
}

// NewPyAnnoteAdapter creates a new PyAnnote diarization adapter
func NewPyAnnoteAdapter(_ string) *PyAnnoteAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "pyannote",
		ModelFamily:        "pyannote",
		DisplayName:        "PyAnnote Speaker Diarization Community 1",
		Description:        "PyAnnote community model for speaker diarization",
		Version:            "3.x",
		SupportedLanguages: []string{"*"},
		SupportedFormats:   []string{"wav", "mp3", "flac", "m4a", "ogg"},
		RequiresGPU:        false,
		MemoryRequirement:  2048,
		Features: map[string]bool{
			"speaker_detection":   true,
			"speaker_constraints": true,
			"confidence_scores":   true,
			"rttm_output":         true,
			"flexible_speakers":   true,
		},
		Metadata: map[string]string{
			"engine":    "pyannote_audio",
			"framework": "pytorch",
			"license":   "MIT",
			"requires":  "huggingface_token",
			"model_hub": "huggingface",
		},
	}

	schema := []interfaces.ParameterSchema{
		{
			Name:        "hf_token",
			Type:        "string",
			Required:    false,
			Default:     nil,
			Description: "HuggingFace token for model access (optional if HF_TOKEN env var is set)",
			Group:       "basic",
		},
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     "pyannote/speaker-diarization-community-1",
			Options:     []string{"pyannote/speaker-diarization-community-1", "pyannote/speaker-diarization-3.1"},
			Description: "PyAnnote model to use",
			Group:       "basic",
		},
		{
			Name:        "min_speakers",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{20}[0],
			Description: "Minimum number of speakers",
			Group:       "basic",
		},
		{
			Name:        "max_speakers",
			Type:        "int",
			Required:    false,
			Default:     nil,
			Min:         &[]float64{1}[0],
			Max:         &[]float64{20}[0],
			Description: "Maximum number of speakers",
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
			Options:     []string{"auto", "cpu", "cuda"},
			Description: "Device to use for computation (auto, cpu, or cuda)",
			Group:       "advanced",
		},
		{
			Name:        "segmentation_onset",
			Type:        "float",
			Required:    false,
			Default:     0.5,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "Voice activity detection onset threshold",
			Group:       "advanced",
		},
		{
			Name:        "segmentation_offset",
			Type:        "float",
			Required:    false,
			Default:     0.363,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "Voice activity detection offset threshold",
			Group:       "advanced",
		},
		{
			Name:        "exclusive",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Force non-overlapping diarization (single speaker active)",
			Group:       "advanced",
		},
		{
			Name:        "auto_convert_audio",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Automatically convert audio to supported format",
			Group:       "advanced",
		},
	}

	baseAdapter := NewBaseAdapter("pyannote", "", capabilities, schema)
	return &PyAnnoteAdapter{BaseAdapter: baseAdapter}
}

func (p *PyAnnoteAdapter) GetMaxSpeakers() int {
	return 20
}

func (p *PyAnnoteAdapter) GetMinSpeakers() int {
	return 1
}

// PrepareEnvironment ensures the diarization engine is running.
func (p *PyAnnoteAdapter) PrepareEnvironment(ctx context.Context) error {
	if err := diarengine.Default().EnsureRunning(ctx); err != nil {
		return fmt.Errorf("failed to start diarization engine: %w", err)
	}
	p.initialized = true
	return nil
}

func (p *PyAnnoteAdapter) Diarize(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	startTime := time.Now()
	p.LogProcessingStart(input, procCtx)
	defer func() {
		p.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	if err := p.ValidateAudioInput(input); err != nil {
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}
	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	tempDir, err := p.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer p.CleanupTempDirectory(tempDir)

	audioInput := input
	if p.GetBoolParameter(params, "auto_convert_audio") {
		convertedInput, err := p.ConvertAudioFormat(ctx, input, "wav", 16000)
		if err != nil {
			logger.Warn("Audio conversion failed, using original", "error", err)
		} else {
			audioInput = convertedInput
		}
	}

	result, err := p.diarizeWithEngine(ctx, audioInput, params, procCtx)
	if err != nil {
		return nil, err
	}

	result.ProcessingTime = time.Since(startTime)
	result.ModelUsed = p.GetStringParameter(params, "model")
	if result.ModelUsed == "" {
		result.ModelUsed = "pyannote/speaker-diarization-community-1"
	}
	result.Metadata = p.CreateDefaultMetadata(params)

	logger.Info("PyAnnote diarization completed",
		"segments", len(result.Segments),
		"speakers", result.SpeakerCount,
		"processing_time", result.ProcessingTime)

	return result, nil
}

func (p *PyAnnoteAdapter) diarizeWithEngine(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
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
		ModelId:   "pyannote",
		ModelName: "pyannote/speaker-diarization-community-1",
	}
	if modelName := p.GetStringParameter(params, "model"); modelName != "" {
		spec.ModelName = modelName
	}
	if modelPath := strings.TrimSpace(os.Getenv("DIAR_ENGINE_PYANNOTE_MODEL_PATH")); modelPath != "" {
		spec.ModelPath = modelPath
	}

	if err := manager.LoadModel(ctx, spec); err != nil {
		return nil, fmt.Errorf("failed to load pyannote model: %w", err)
	}

	engineParams := buildDiarEngineParams(p.BaseAdapter, params)
	engineParams["model_family"] = "pyannote"

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		manager.StopJob(context.Background(), procCtx.JobID)
	}()

	status, err := manager.RunJob(jobCtx, procCtx.JobID, inputPath, outputDir, engineParams)
	if err != nil {
		return nil, fmt.Errorf("pyannote engine job failed: %w", err)
	}
	if status.State == pb.JobState_JOB_STATE_FAILED {
		return nil, fmt.Errorf("pyannote engine failed: %s", status.Message)
	}
	if status.State == pb.JobState_JOB_STATE_CANCELLED {
		return nil, fmt.Errorf("pyannote diarization was cancelled")
	}

	resultPath := status.Outputs["diarization"]
	if resultPath == "" {
		return nil, fmt.Errorf("pyannote engine missing diarization output")
	}

	return parseDiarizationJSON(resultPath)
}

func buildDiarEngineParams(adapter *BaseAdapter, params map[string]interface{}) map[string]string {
	engineParams := make(map[string]string)

	if val := adapter.GetStringParameter(params, "hf_token"); val != "" {
		engineParams["hf_token"] = val
	}
	if val := adapter.GetStringParameter(params, "model"); val != "" {
		engineParams["model"] = val
	}
	if val := adapter.GetStringParameter(params, "output_format"); val != "" {
		engineParams["output_format"] = val
	}
	if val := adapter.GetStringParameter(params, "device"); val != "" {
		engineParams["device"] = val
	}
	if val := adapter.GetIntParameter(params, "min_speakers"); val > 0 {
		engineParams["min_speakers"] = fmt.Sprintf("%d", val)
	}
	if val := adapter.GetIntParameter(params, "max_speakers"); val > 0 {
		engineParams["max_speakers"] = fmt.Sprintf("%d", val)
	}
	if val := adapter.GetFloatParameter(params, "segmentation_onset"); val > 0 {
		engineParams["segmentation_onset"] = fmt.Sprintf("%.3f", val)
	}
	if val := adapter.GetFloatParameter(params, "segmentation_offset"); val > 0 {
		engineParams["segmentation_offset"] = fmt.Sprintf("%.3f", val)
	}
	if val := adapter.GetIntParameter(params, "batch_size"); val > 0 {
		engineParams["batch_size"] = fmt.Sprintf("%d", val)
	}
	if val := adapter.GetBoolParameter(params, "streaming_mode"); val {
		engineParams["streaming_mode"] = "true"
	}
	if val := adapter.GetFloatParameter(params, "chunk_length_s"); val > 0 {
		engineParams["chunk_length_s"] = fmt.Sprintf("%.2f", val)
	}
	if val := adapter.GetIntParameter(params, "chunk_len"); val > 0 {
		engineParams["chunk_len"] = fmt.Sprintf("%d", val)
	}
	if val := adapter.GetIntParameter(params, "chunk_right_context"); val > 0 {
		engineParams["chunk_right_context"] = fmt.Sprintf("%d", val)
	}
	if val := adapter.GetIntParameter(params, "fifo_len"); val > 0 {
		engineParams["fifo_len"] = fmt.Sprintf("%d", val)
	}
	if val := adapter.GetIntParameter(params, "spkcache_update_period"); val > 0 {
		engineParams["spkcache_update_period"] = fmt.Sprintf("%d", val)
	}
	if val := adapter.GetBoolParameter(params, "exclusive"); val {
		engineParams["exclusive"] = "true"
	}

	return engineParams
}

func parseDiarizationJSON(path string) (*interfaces.DiarizationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read diarization output: %w", err)
	}

	var resultPayload struct {
		AudioFile string `json:"audio_file"`
		Model     string `json:"model"`
		ModelID   string `json:"model_id"`
		Segments  []struct {
			Start      float64 `json:"start"`
			End        float64 `json:"end"`
			Speaker    string  `json:"speaker"`
			Confidence float64 `json:"confidence"`
			Duration   float64 `json:"duration"`
		} `json:"segments"`
		Speakers     []string `json:"speakers"`
		SpeakerCount int      `json:"speaker_count"`
	}

	if err := json.Unmarshal(data, &resultPayload); err != nil {
		return nil, fmt.Errorf("failed to parse diarization JSON: %w", err)
	}

	result := &interfaces.DiarizationResult{
		Segments:     make([]interfaces.DiarizationSegment, len(resultPayload.Segments)),
		SpeakerCount: resultPayload.SpeakerCount,
		Speakers:     resultPayload.Speakers,
		ModelUsed:    resultPayload.Model,
	}

	for i, seg := range resultPayload.Segments {
		result.Segments[i] = interfaces.DiarizationSegment{
			Start:      seg.Start,
			End:        seg.End,
			Speaker:    seg.Speaker,
			Confidence: seg.Confidence,
		}
	}

	return result, nil
}
