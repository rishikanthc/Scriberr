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
		out = append(out, modelCardFromEngine(model, p.id))
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
	task := strings.TrimSpace(req.Task)
	if task == "" {
		task = "transcribe"
	}
	enableTokenTimestamps := true
	enableSegmentTimestamps := true
	engineReq := speechengine.TranscriptionRequest{
		RequestID:               req.JobID,
		ModelID:                 modelID,
		AudioPath:               req.AudioPath,
		Progress:                localProgressSink{downstream: req.Progress},
		Language:                req.Language,
		Task:                    task,
		TailPaddings:            req.TailPaddings,
		EnableTokenTimestamps:   &enableTokenTimestamps,
		EnableSegmentTimestamps: &enableSegmentTimestamps,
		DecodingMethod:          req.DecodingMethod,
		Chunking:                req.Chunking,
		ChunkDurationSec:        req.ChunkDurationSec,
		NumThreads:              coalesceInt(req.Threads, p.cfg.Threads),
		Provider:                p.provider,
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
	}, nil
}

func (p *LocalProvider) Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error) {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = DefaultDiarizationModel
	}
	engineReq := speechengine.DiarizationRequest{
		RequestID:      req.JobID,
		ModelID:        modelID,
		AudioPath:      req.AudioPath,
		Progress:       localProgressSink{downstream: req.Progress},
		NumClusters:    req.NumSpeakers,
		Threshold:      req.Threshold,
		MinDurationOn:  req.MinDurationOn,
		MinDurationOff: req.MinDurationOff,
		NumThreads:     p.cfg.Threads,
		Provider:       p.provider,
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

func modelCardFromEngine(model speechengine.ModelCard, providerID string) asrcontract.ModelCard {
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
		Chunking:             chunkingCapabilitiesForModel(capabilities),
		ParameterSchema:      parameterSchemaForModel(model, capabilities),
		RecommendedDefaults:  recommendedDefaultsForModel(model, capabilities),
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

func chunkingCapabilitiesForModel(capabilities asrcontract.Capabilities) *asrcontract.ChunkingCapabilities {
	if !capabilities.Transcription {
		return nil
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

func parameterSchemaForModel(model speechengine.ModelCard, capabilities asrcontract.Capabilities) asrcontract.ParameterSchema {
	var schema asrcontract.ParameterSchema
	if capabilities.Transcription {
		schema = append(schema,
			asrcontract.ParameterDescriptor{
				Key:            asrcontract.CommonParameterRuntimeNumThreads,
				Label:          "Threads",
				Type:           asrcontract.ParameterTypeInteger,
				Default:        float64(0),
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
				Default: "vad",
				Options: []asrcontract.ParameterOption{
					{Value: "fixed", Label: "Fixed"},
					{Value: "vad", Label: "VAD"},
				},
				Scope: asrcontract.ParameterScopeChunking,
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
		schema = append(schema, asrcontract.ParameterDescriptor{
			Key:      "sherpa.whisper.tail_paddings",
			Label:    "Tail paddings",
			Type:     asrcontract.ParameterTypeInteger,
			Default:  float64(-1),
			Min:      float64Ptr(-1),
			Max:      float64Ptr(16),
			Step:     float64Ptr(1),
			Scope:    asrcontract.ParameterScopeDecoding,
			Advanced: true,
		})
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

func recommendedDefaultsForModel(model speechengine.ModelCard, capabilities asrcontract.Capabilities) map[string]any {
	defaults := map[string]any{}
	if capabilities.Transcription {
		defaults[asrcontract.CommonParameterRuntimeNumThreads] = 0
		defaults[asrcontract.CommonParameterDecodingMethod] = "greedy_search"
		defaults[asrcontract.CommonParameterChunkingMode] = "vad"
		defaults[asrcontract.CommonParameterOutputWordTimestamps] = true
	}
	if capabilities.Transcription && strings.Contains(strings.ToLower(model.Family), "whisper") {
		defaults["sherpa.whisper.tail_paddings"] = -1
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
