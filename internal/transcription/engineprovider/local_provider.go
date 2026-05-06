package engineprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	appconfig "scriberr/internal/config"
	"scriberr/internal/transcription/asrcontract"
	"scriberr/pkg/logger"

	speechengine "scriberr-engine/speech/engine"
	"scriberr-engine/speech/runtime"
)

type LocalConfig struct {
	CacheDir     string
	Provider     string
	Threads      int
	MaxLoaded    int
	AutoDownload bool
}

type speechEngine interface {
	Inspect(ctx context.Context) (*speechengine.ProviderInfo, error)
	Models(ctx context.Context) ([]speechengine.ModelCard, error)
	Status(ctx context.Context) (*speechengine.ProviderStatus, error)
	LoadedModels() []speechengine.LoadedModel
	Transcribe(ctx context.Context, req speechengine.TranscriptionRequest) (*speechengine.TranscriptionResult, error)
	Diarize(ctx context.Context, req speechengine.DiarizationRequest) (*speechengine.DiarizationResult, error)
	Close() error
}

type modelLoader interface {
	LoadModel(ctx context.Context, modelID string) error
	UnloadModel(modelID string) error
}

type LocalProvider struct {
	id       string
	cfg      LocalConfig
	engine   speechEngine
	provider runtime.Provider
}

func NewLocalProvider(cfg appconfig.EngineConfig) (*LocalProvider, error) {
	return NewLocalProviderFromConfig(LocalConfig{
		CacheDir:     cfg.CacheDir,
		Provider:     cfg.Provider,
		Threads:      cfg.Threads,
		MaxLoaded:    cfg.MaxLoaded,
		AutoDownload: cfg.AutoDownload,
	})
}

func NewLocalProviderFromConfig(cfg LocalConfig) (*LocalProvider, error) {
	provider, err := runtime.ParseProvider(cfg.Provider)
	if err != nil {
		return nil, sanitizeError(err)
	}
	engineCfg := speechengine.Config{
		CacheDir:        cfg.CacheDir,
		Provider:        provider,
		Threads:         cfg.Threads,
		MaxLoaded:       cfg.MaxLoaded,
		AutoDownload:    cfg.AutoDownload,
		AutoDownloadSet: true,
		Logger:          slog.Default(),
	}
	engine, err := speechengine.New(engineCfg)
	if err != nil {
		return nil, sanitizeError(err)
	}
	logger.Info("Engine provider initialized",
		"provider_id", DefaultProviderID,
		"requested_provider", cfg.Provider,
		"cache_dir", cfg.CacheDir,
		"threads", cfg.Threads,
		"max_loaded", cfg.MaxLoaded,
		"auto_download", cfg.AutoDownload,
	)
	return newLocalProviderWithEngine(DefaultProviderID, cfg, provider, engine), nil
}

func newLocalProviderWithEngine(id string, cfg LocalConfig, provider runtime.Provider, engine speechEngine) *LocalProvider {
	if strings.TrimSpace(id) == "" {
		id = DefaultProviderID
	}
	return &LocalProvider{
		id:       id,
		cfg:      cfg,
		engine:   engine,
		provider: provider,
	}
}

func (p *LocalProvider) ID() string {
	return p.id
}

func (p *LocalProvider) Inspect(ctx context.Context) (*asrcontract.ProviderInfo, error) {
	info, err := p.engine.Inspect(ctx)
	if err != nil {
		return nil, sanitizeError(err)
	}
	return providerInfoFromEngine(info, p.id), nil
}

func (p *LocalProvider) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	models, err := p.engine.Models(ctx)
	if err != nil {
		return nil, sanitizeError(err)
	}
	out := make([]asrcontract.ModelCard, 0, len(models))
	for _, model := range models {
		out = append(out, modelCardFromEngine(model, p.id, p.cfg, p.provider))
	}
	return out, nil
}

func (p *LocalProvider) Status(ctx context.Context) (*asrcontract.ProviderStatus, error) {
	status, err := p.engine.Status(ctx)
	if err != nil {
		return nil, sanitizeError(err)
	}
	return providerStatusFromEngine(status), nil
}

func (p *LocalProvider) LoadModel(ctx context.Context, req asrcontract.LoadModelRequest) error {
	if loader, ok := p.engine.(modelLoader); ok {
		return sanitizeError(loader.LoadModel(ctx, req.Model))
	}
	return asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local engine model loading is not available", false)
}

func (p *LocalProvider) UnloadModel(ctx context.Context, req asrcontract.UnloadModelRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if loader, ok := p.engine.(modelLoader); ok {
		return sanitizeError(loader.UnloadModel(req.Model))
	}
	return asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local engine model unloading is not available", false)
}

func (p *LocalProvider) LoadedModels(ctx context.Context) ([]asrcontract.LoadedModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	models := p.engine.LoadedModels()
	out := make([]asrcontract.LoadedModel, 0, len(models))
	for _, model := range models {
		out = append(out, loadedModelFromEngine(model))
	}
	return out, nil
}

func (p *LocalProvider) Capabilities(ctx context.Context) ([]ModelCapability, error) {
	models, err := p.Models(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ModelCapability, 0, len(models))
	for _, model := range models {
		capability := ModelCapability{
			ID:           model.ID,
			Name:         model.DisplayName,
			Provider:     p.id,
			Installed:    model.Installed,
			Default:      model.Default,
			Capabilities: capabilityNames(model.Capabilities),
		}
		out = append(out, capability)
	}
	return out, nil
}

func (p *LocalProvider) Prepare(ctx context.Context) error {
	return ctx.Err()
}

func (p *LocalProvider) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error) {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = DefaultTranscriptionModel
	}
	engineReq := speechengine.TranscriptionRequest{
		RequestID:  req.JobID,
		ModelID:    modelID,
		AudioPath:  req.AudioPath,
		Parameters: copyParameters(req.Parameters),
		Progress:   localProgressSink{downstream: req.Progress},
	}
	out, err := p.engine.Transcribe(ctx, engineReq)
	if err != nil {
		return nil, sanitizeError(err)
	}
	if out == nil {
		return nil, sanitizeErrorf("local engine returned no transcription result")
	}
	words := make([]TranscriptWord, 0, len(out.Words))
	for _, word := range out.Words {
		words = append(words, TranscriptWord{
			Start: word.StartSec,
			End:   word.EndSec,
			Word:  word.Text,
		})
	}
	segments := make([]TranscriptSegment, 0, len(out.Segments))
	for idx, segment := range out.Segments {
		segments = append(segments, TranscriptSegment{
			ID:    fmt.Sprintf("seg_%04d", idx),
			Start: segment.StartSec,
			End:   segment.EndSec,
			Text:  segment.Text,
		})
	}
	return &TranscriptionResult{
		Text:     out.Text,
		Language: out.Language,
		Words:    words,
		Segments: segments,
		ModelID:  modelID,
		EngineID: p.id,
		Metadata: localTranscriptionMetadata(modelID, out),
	}, nil
}

func localTranscriptionMetadata(modelID string, out *speechengine.TranscriptionResult) map[string]any {
	metadata := map[string]any{
		"model": modelID,
	}
	if out == nil {
		return metadata
	}
	if out.Metrics.AudioDurationSec > 0 {
		metadata["audio_duration_s"] = out.Metrics.AudioDurationSec
	}
	if out.Metrics.DecodeDuration > 0 {
		metadata["decode_time_ms"] = out.Metrics.DecodeDuration.Milliseconds()
	}
	if out.Metrics.ChunkCount > 0 {
		metadata["chunk_count"] = out.Metrics.ChunkCount
	}
	if out.Metrics.BatchSize > 0 {
		metadata["batch_size"] = out.Metrics.BatchSize
	}
	if out.Metrics.HypothesisWords > 0 {
		metadata["hypothesis_words"] = out.Metrics.HypothesisWords
	}
	if rtf := out.Metrics.RealTimeFactor(); rtf > 0 {
		metadata["rtf"] = rtf
	}
	if strings.TrimSpace(out.Plan.ChunkingMode) != "" {
		metadata["chunking_mode"] = out.Plan.ChunkingMode
	}
	if strings.TrimSpace(out.Plan.Task) != "" {
		metadata["task"] = out.Plan.Task
	}
	metadata["plan"] = out.Plan
	return metadata
}

func copyParameters(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (p *LocalProvider) Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error) {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = DefaultDiarizationModel
	}
	engineReq := speechengine.DiarizationRequest{
		RequestID:  req.JobID,
		ModelID:    modelID,
		AudioPath:  req.AudioPath,
		Parameters: copyParameters(req.Parameters),
		Progress:   localProgressSink{downstream: req.Progress},
	}
	out, err := p.engine.Diarize(ctx, engineReq)
	if err != nil {
		return nil, sanitizeError(err)
	}
	if out == nil {
		return nil, sanitizeErrorf("local engine returned no diarization result")
	}
	segments := make([]DiarizationSegment, 0, len(out.Segments))
	for _, segment := range out.Segments {
		segments = append(segments, DiarizationSegment{
			Start:   segment.Start,
			End:     segment.End,
			Speaker: fmt.Sprintf("SPEAKER_%02d", segment.Speaker),
		})
	}
	return &DiarizationResult{
		Segments: segments,
		ModelID:  modelID,
		EngineID: p.id,
	}, nil
}

func (p *LocalProvider) IdentifySpeakers(ctx context.Context, req asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local provider does not support speaker identification", false)
}

func (p *LocalProvider) Close() error {
	if p.engine == nil {
		return nil
	}
	if err := p.engine.Close(); err != nil {
		return sanitizeError(err)
	}
	return nil
}

func capabilityNames(capabilities asrcontract.Capabilities) []string {
	out := []string{}
	if capabilities.Transcription {
		out = append(out, "transcription")
	}
	if capabilities.Diarization {
		out = append(out, "diarization")
	}
	if capabilities.WordTimestamps {
		out = append(out, "word_timestamps")
	}
	return out
}

func coalesceInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

type localProgressSink struct {
	downstream ProgressSink
}

func (s localProgressSink) Report(ctx context.Context, progress speechengine.Progress) {
	if s.downstream == nil {
		return
	}
	s.downstream.Report(ctx, asrcontract.ProviderProgress{
		Stage:     asrcontract.Stage(progress.Stage),
		Progress:  progress.Progress,
		Message:   progress.Message,
		Operation: asrcontract.Operation(progress.Operation),
		Model:     progress.Model,
		Timestamp: progress.Timestamp,
	})
}

func providerInfoFromEngine(info *speechengine.ProviderInfo, providerID string) *asrcontract.ProviderInfo {
	if info == nil {
		return nil
	}
	return &asrcontract.ProviderInfo{
		ContractVersion: asrcontract.ContractVersionV1,
		Provider: asrcontract.ProviderIdentity{
			ID:      providerID,
			Name:    info.Provider.Name,
			Version: info.Provider.Version,
			Vendor:  info.Provider.Vendor,
		},
		Runtime: asrcontract.RuntimeInfo{
			DeviceBackends:       append([]string(nil), info.Runtime.DeviceBackends...),
			ActiveBackend:        info.Runtime.ActiveBackend,
			SupportsConcurrent:   info.Runtime.SupportsConcurrent,
			MaxConcurrentJobs:    info.Runtime.MaxConcurrentJobs,
			ProviderCapabilities: providerCapabilitiesFromEngine(info.Runtime.ProviderCapabilities),
		},
		AudioInput: asrcontract.AudioInputSpec{
			RequiredSampleRate: info.AudioInput.RequiredSampleRate,
			RequiredChannels:   info.AudioInput.RequiredChannels,
			Formats:            append([]string(nil), info.AudioInput.Formats...),
			PathMode:           asrcontract.PathMode(info.AudioInput.PathMode),
		},
	}
}

func modelCardFromEngine(model speechengine.ModelCard, providerID string, cfg LocalConfig, provider runtime.Provider) asrcontract.ModelCard {
	capabilities := capabilitiesFromEngine(model.Capabilities)
	return asrcontract.ModelCard{
		ID:                   model.ID,
		DisplayName:          model.DisplayName,
		Provider:             providerID,
		Family:               model.Family,
		Version:              model.Version,
		Installed:            model.Installed,
		Loaded:               model.Loaded,
		Default:              model.Default,
		Tasks:                tasksFromEngine(model.Tasks),
		Languages:            append([]string(nil), model.Languages...),
		LanguageSupport:      languageSupportForModel(model),
		Capabilities:         capabilities,
		ResourceRequirements: resourceRequirementsFromEngine(model.ResourceRequirements),
		Chunking:             chunkingCapabilitiesForModel(model, capabilities),
		ParameterSchema:      parameterSchemaForModel(model, capabilities, cfg, provider),
		RecommendedDefaults:  recommendedDefaultsForModel(model, capabilities, cfg, provider),
		Extensions:           modelDescriptorExtensions(model, capabilities),
	}
}

func languageSupportForModel(model speechengine.ModelCard) *asrcontract.LanguageSupport {
	if len(model.Languages) == 0 {
		return nil
	}
	mode := "fixed"
	if len(model.Languages) > 1 {
		mode = "user_configurable"
	}
	return &asrcontract.LanguageSupport{Languages: append([]string(nil), model.Languages...), Mode: mode}
}

func chunkingCapabilitiesForModel(model speechengine.ModelCard, capabilities asrcontract.Capabilities) *asrcontract.ChunkingCapabilities {
	if !capabilities.Transcription {
		return nil
	}
	if isParakeetFamily(model.Family) {
		return &asrcontract.ChunkingCapabilities{
			SupportsEngineChunking:   true,
			SupportsProviderChunking: false,
			PreferredMode:            "fixed",
			RecommendedChunkSeconds:  float64Ptr(30),
			MaxChunkSeconds:          float64Ptr(120),
			SupportsBatching:         true,
			RecommendedBatchSize:     intPtr(1),
			MaxBatchSize:             intPtr(1),
		}
	}
	return &asrcontract.ChunkingCapabilities{
		SupportsEngineChunking:   true,
		SupportsProviderChunking: false,
		PreferredMode:            "vad",
		RecommendedChunkSeconds:  float64Ptr(30),
		MaxChunkSeconds:          float64Ptr(120),
		SupportsBatching:         false,
		RecommendedBatchSize:     intPtr(1),
		MaxBatchSize:             intPtr(1),
	}
}

func parameterSchemaForModel(model speechengine.ModelCard, capabilities asrcontract.Capabilities, cfg LocalConfig, provider runtime.Provider) asrcontract.ParameterSchema {
	var schema asrcontract.ParameterSchema
	if capabilities.Transcription {
		schema = append(schema,
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.offline.sample_rate",
				Label:          "Sample rate",
				Type:           asrcontract.ParameterTypeInteger,
				Default:        float64(16000),
				Min:            float64Ptr(8000),
				Max:            float64Ptr(48000),
				Step:           float64Ptr(1000),
				Scope:          asrcontract.ParameterScopeModel,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.offline.feature_dim",
				Label:          "Feature dim",
				Type:           asrcontract.ParameterTypeInteger,
				Default:        float64(80),
				Min:            float64Ptr(40),
				Max:            float64Ptr(128),
				Step:           float64Ptr(1),
				Scope:          asrcontract.ParameterScopeModel,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.runtime.provider",
				Label:          "Provider",
				Type:           asrcontract.ParameterTypeEnum,
				Default:        string(provider),
				Options:        sherpaProviderOptions(),
				Scope:          asrcontract.ParameterScopeRuntime,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            asrcontract.CommonParameterRuntimeNumThreads,
				Label:          "Threads",
				Type:           asrcontract.ParameterTypeInteger,
				Default:        float64(defaultThreadsForModel(model, cfg)),
				Min:            float64Ptr(0),
				Max:            float64Ptr(64),
				Step:           float64Ptr(1),
				Scope:          asrcontract.ParameterScopeRuntime,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:     asrcontract.CommonParameterDecodingMethod,
				Label:   "Decoding method",
				Type:    asrcontract.ParameterTypeEnum,
				Default: "greedy_search",
				Options: []asrcontract.ParameterOption{
					{Value: "greedy_search", Label: "Greedy search"},
					{Value: "modified_beam_search", Label: "Modified beam search"},
				},
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:     asrcontract.CommonParameterChunkingMode,
				Label:   "Chunking mode",
				Type:    asrcontract.ParameterTypeEnum,
				Default: defaultChunkingModeForModel(model),
				Options: []asrcontract.ParameterOption{
					{Value: "fixed", Label: "Fixed"},
					{Value: "vad", Label: "VAD"},
					{Value: "provider", Label: "Provider"},
					{Value: "none", Label: "None"},
				},
				Scope: asrcontract.ParameterScopeChunking,
			},
			asrcontract.ParameterDescriptor{
				Key:     asrcontract.CommonParameterChunkingChunkSeconds,
				Label:   "Chunk seconds",
				Type:    asrcontract.ParameterTypeNumber,
				Default: defaultChunkSecondsForModel(model),
				Min:     float64Ptr(1),
				Max:     float64Ptr(120),
				Step:    float64Ptr(1),
				Scope:   asrcontract.ParameterScopeChunking,
			},
			asrcontract.ParameterDescriptor{
				Key:      asrcontract.CommonParameterBatchingBatchSize,
				Label:    "Batch size",
				Type:     asrcontract.ParameterTypeInteger,
				Default:  float64(defaultBatchSizeForModel(model)),
				Min:      float64Ptr(1),
				Max:      float64Ptr(float64(defaultBatchSizeForModel(model))),
				Step:     float64Ptr(1),
				Scope:    asrcontract.ParameterScopeRuntime,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.runtime.debug",
				Label:          "Debug",
				Type:           asrcontract.ParameterTypeBoolean,
				Default:        false,
				Scope:          asrcontract.ParameterScopeRuntime,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.decoding.max_active_paths",
				Label:    "Max active paths",
				Type:     asrcontract.ParameterTypeInteger,
				Default:  float64(4),
				Min:      float64Ptr(1),
				Max:      float64Ptr(32),
				Step:     float64Ptr(1),
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.decoding.hotwords_file",
				Label:    "Hotwords file",
				Type:     asrcontract.ParameterTypePathRef,
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.decoding.hotwords_score",
				Label:    "Hotwords score",
				Type:     asrcontract.ParameterTypeNumber,
				Default:  float64(1.5),
				Min:      float64Ptr(0),
				Max:      float64Ptr(20),
				Step:     float64Ptr(0.1),
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.decoding.blank_penalty",
				Label:    "Blank penalty",
				Type:     asrcontract.ParameterTypeNumber,
				Default:  float64(0),
				Min:      float64Ptr(0),
				Max:      float64Ptr(10),
				Step:     float64Ptr(0.1),
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.decoding.rule_fsts",
				Label:    "Rule FSTs",
				Type:     asrcontract.ParameterTypePathRef,
				Scope:    asrcontract.ParameterScopePostprocess,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.decoding.rule_fars",
				Label:    "Rule FARs",
				Type:     asrcontract.ParameterTypePathRef,
				Scope:    asrcontract.ParameterScopePostprocess,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.lm.model",
				Label:    "LM model",
				Type:     asrcontract.ParameterTypePathRef,
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.lm.scale",
				Label:    "LM scale",
				Type:     asrcontract.ParameterTypeNumber,
				Default:  float64(0),
				Min:      float64Ptr(0),
				Max:      float64Ptr(5),
				Step:     float64Ptr(0.1),
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      asrcontract.CommonParameterOutputWordTimestamps,
				Label:    "Word timestamps",
				Type:     asrcontract.ParameterTypeBoolean,
				Default:  true,
				Scope:    asrcontract.ParameterScopeOutput,
				Advanced: true,
			},
		)
	}
	if capabilities.Transcription && strings.Contains(strings.ToLower(model.Family), "whisper") {
		schema = append(schema,
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.whisper.language",
				Label:    "Language",
				Type:     asrcontract.ParameterTypeString,
				Default:  "auto",
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: false,
			},
			asrcontract.ParameterDescriptor{
				Key:     "sherpa.whisper.task",
				Label:   "Task",
				Type:    asrcontract.ParameterTypeEnum,
				Default: "transcribe",
				Options: []asrcontract.ParameterOption{
					{Value: "transcribe", Label: "Transcribe"},
					{Value: "translate", Label: "Translate"},
				},
				Scope: asrcontract.ParameterScopeDecoding,
			},
			asrcontract.ParameterDescriptor{
				Key:      "sherpa.whisper.tail_paddings",
				Label:    "Tail paddings",
				Type:     asrcontract.ParameterTypeInteger,
				Default:  float64(-1),
				Min:      float64Ptr(-1),
				Max:      float64Ptr(16),
				Step:     float64Ptr(1),
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      asrcontract.CommonParameterOutputTokenTimestamps,
				Label:    "Token timestamps",
				Type:     asrcontract.ParameterTypeBoolean,
				Default:  false,
				Scope:    asrcontract.ParameterScopeOutput,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:      asrcontract.CommonParameterOutputTimestamps,
				Label:    "Segment timestamps",
				Type:     asrcontract.ParameterTypeBoolean,
				Default:  true,
				Scope:    asrcontract.ParameterScopeOutput,
				Advanced: true,
			},
		)
	}
	if capabilities.Transcription && isParakeetFamily(model.Family) {
		schema = append(schema,
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.nemo_transducer.encoder",
				Label:          "Encoder artifact",
				Type:           asrcontract.ParameterTypePathRef,
				Scope:          asrcontract.ParameterScopeModel,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.nemo_transducer.decoder",
				Label:          "Decoder artifact",
				Type:           asrcontract.ParameterTypePathRef,
				Scope:          asrcontract.ParameterScopeModel,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.nemo_transducer.joiner",
				Label:          "Joiner artifact",
				Type:           asrcontract.ParameterTypePathRef,
				Scope:          asrcontract.ParameterScopeModel,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.tokens",
				Label:          "Tokens artifact",
				Type:           asrcontract.ParameterTypePathRef,
				Scope:          asrcontract.ParameterScopeModel,
				Advanced:       true,
				RequiresReload: true,
			},
			asrcontract.ParameterDescriptor{
				Key:            "sherpa.model_type",
				Label:          "Model type",
				Type:           asrcontract.ParameterTypeEnum,
				Default:        "nemo_transducer",
				Options:        []asrcontract.ParameterOption{{Value: "nemo_transducer", Label: "NeMo transducer"}},
				Scope:          asrcontract.ParameterScopeModel,
				Advanced:       true,
				RequiresReload: true,
			},
		)
	}
	if capabilities.Diarization {
		schema = append(schema,
			asrcontract.ParameterDescriptor{
				Key:      "diarization.num_speakers",
				Label:    "Speakers",
				Type:     asrcontract.ParameterTypeInteger,
				Default:  float64(0),
				Min:      float64Ptr(0),
				Max:      float64Ptr(64),
				Step:     float64Ptr(1),
				Scope:    asrcontract.ParameterScopeDecoding,
				Advanced: true,
			},
			asrcontract.ParameterDescriptor{
				Key:     asrcontract.CommonParameterVADThreshold,
				Label:   "Threshold",
				Type:    asrcontract.ParameterTypeNumber,
				Default: float64(0.5),
				Min:     float64Ptr(0),
				Max:     float64Ptr(1),
				Step:    float64Ptr(0.01),
				Scope:   asrcontract.ParameterScopeVAD,
			},
		)
	}
	return schema
}

func recommendedDefaultsForModel(model speechengine.ModelCard, capabilities asrcontract.Capabilities, cfg LocalConfig, provider runtime.Provider) map[string]any {
	defaults := map[string]any{}
	if capabilities.Transcription {
		defaults["sherpa.offline.sample_rate"] = 16000
		defaults["sherpa.offline.feature_dim"] = 80
		defaults["sherpa.runtime.provider"] = string(provider)
		defaults[asrcontract.CommonParameterRuntimeNumThreads] = defaultThreadsForModel(model, cfg)
		defaults[asrcontract.CommonParameterDecodingMethod] = "greedy_search"
		defaults[asrcontract.CommonParameterChunkingMode] = defaultChunkingModeForModel(model)
		defaults[asrcontract.CommonParameterChunkingChunkSeconds] = defaultChunkSecondsForModel(model)
		defaults[asrcontract.CommonParameterBatchingBatchSize] = defaultBatchSizeForModel(model)
		defaults[asrcontract.CommonParameterOutputWordTimestamps] = true
	}
	if capabilities.Transcription && strings.Contains(strings.ToLower(model.Family), "whisper") {
		defaults["sherpa.whisper.language"] = "auto"
		defaults["sherpa.whisper.task"] = "transcribe"
		defaults["sherpa.whisper.tail_paddings"] = -1
		defaults[asrcontract.CommonParameterOutputTokenTimestamps] = false
		defaults[asrcontract.CommonParameterOutputTimestamps] = true
	}
	if capabilities.Transcription && isParakeetFamily(model.Family) {
		defaults["sherpa.model_type"] = "nemo_transducer"
	}
	if capabilities.Diarization {
		defaults["diarization.num_speakers"] = 0
		defaults[asrcontract.CommonParameterVADThreshold] = 0.5
	}
	if len(defaults) == 0 {
		return nil
	}
	return defaults
}

func sherpaProviderOptions() []asrcontract.ParameterOption {
	return []asrcontract.ParameterOption{
		{Value: string(runtime.ProviderAuto), Label: "Auto"},
		{Value: string(runtime.ProviderCPU), Label: "CPU"},
		{Value: string(runtime.ProviderCUDA), Label: "CUDA"},
	}
}

func defaultThreadsForModel(model speechengine.ModelCard, cfg LocalConfig) int {
	if isParakeetFamily(model.Family) {
		return 4
	}
	if cfg.Threads > 0 {
		return cfg.Threads
	}
	return 0
}

func defaultChunkingModeForModel(model speechengine.ModelCard) string {
	if isParakeetFamily(model.Family) {
		return "fixed"
	}
	return "vad"
}

func defaultChunkSecondsForModel(model speechengine.ModelCard) float64 {
	if isParakeetFamily(model.Family) {
		return 30
	}
	return 30
}

func defaultBatchSizeForModel(model speechengine.ModelCard) int {
	return 1
}

func modelDescriptorExtensions(model speechengine.ModelCard, capabilities asrcontract.Capabilities) map[string]any {
	if !capabilities.Transcription || !isParakeetFamily(model.Family) {
		return nil
	}
	return map[string]any{
		"artifact_requirements": []string{"encoder", "decoder", "joiner", "tokens"},
		"model_type":            "nemo_transducer",
		"default_profile":       "measured_cpu_fixed_30s_threads_4_batch_1",
	}
}

func isParakeetFamily(family string) bool {
	normalized := strings.ToLower(strings.TrimSpace(family))
	return normalized == "nemo_transducer" || strings.Contains(normalized, "parakeet")
}

func providerStatusFromEngine(status *speechengine.ProviderStatus) *asrcontract.ProviderStatus {
	if status == nil {
		return nil
	}
	loaded := make([]asrcontract.LoadedModel, 0, len(status.LoadedModels))
	for _, model := range status.LoadedModels {
		loaded = append(loaded, loadedModelFromEngine(model))
	}
	return &asrcontract.ProviderStatus{
		State:        asrcontract.ProviderState(status.State),
		ActiveJob:    activeJobFromEngine(status.ActiveJob),
		LoadedModels: loaded,
		Capacity: asrcontract.ProviderCapacity{
			MaxConcurrentJobs: status.Capacity.MaxConcurrentJobs,
			AvailableSlots:    status.Capacity.AvailableSlots,
		},
	}
}

func activeJobFromEngine(job *speechengine.ActiveJob) *asrcontract.ActiveJob {
	if job == nil {
		return nil
	}
	return &asrcontract.ActiveJob{
		ID:        job.ID,
		Operation: asrcontract.Operation(job.Operation),
		Model:     job.Model,
		Stage:     asrcontract.Stage(job.Stage),
		Progress:  job.Progress,
	}
}

func loadedModelFromEngine(model speechengine.LoadedModel) asrcontract.LoadedModel {
	return asrcontract.LoadedModel{
		ID:       model.ID,
		LoadedAt: model.LoadedAt,
	}
}

func providerCapabilitiesFromEngine(capabilities []speechengine.Capability) []asrcontract.Capability {
	out := make([]asrcontract.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, asrcontract.Capability(capability))
	}
	return out
}

func tasksFromEngine(tasks []speechengine.Task) []asrcontract.Task {
	out := make([]asrcontract.Task, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, asrcontract.Task(task))
	}
	return out
}

func capabilitiesFromEngine(capabilities speechengine.Capabilities) asrcontract.Capabilities {
	return asrcontract.Capabilities{
		Transcription:     capabilities.Transcription,
		Diarization:       capabilities.Diarization,
		WordTimestamps:    capabilities.WordTimestamps,
		SegmentTimestamps: capabilities.SegmentTimestamps,
		TokenTimestamps:   capabilities.TokenTimestamps,
		LanguageDetection: capabilities.LanguageDetection,
		SpeakerEmbeddings: capabilities.SpeakerEmbeddings,
	}
}

func resourceRequirementsFromEngine(requirements speechengine.ResourceRequirements) asrcontract.ResourceRequirements {
	return asrcontract.ResourceRequirements{
		Backends: append([]string(nil), requirements.Backends...),
	}
}

func float64Ptr(value float64) *float64 { return &value }

func intPtr(value int) *int { return &value }
