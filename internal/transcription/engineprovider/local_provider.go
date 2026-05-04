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
	speechmodels "scriberr-engine/speech/models"
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
	Transcribe(ctx context.Context, req speechengine.TranscriptionRequest) (*speechengine.TranscriptionResult, error)
	Diarize(ctx context.Context, req speechengine.DiarizationRequest) (*speechengine.DiarizationResult, error)
	IsModelInstalled(modelID string) bool
	Close() error
}

type modelLoader interface {
	LoadModel(ctx context.Context, modelID string) error
	UnloadModel(modelID string) error
	ListLoadedModels() []speechmodels.ModelID
}

type LocalProvider struct {
	id       string
	cfg      LocalConfig
	engine   speechEngine
	specs    []speechmodels.ModelSpec
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
	return newLocalProviderWithEngine(DefaultProviderID, cfg, provider, engine, speechmodels.DefaultModelSpecs()), nil
}

func newLocalProviderWithEngine(id string, cfg LocalConfig, provider runtime.Provider, engine speechEngine, specs []speechmodels.ModelSpec) *LocalProvider {
	if strings.TrimSpace(id) == "" {
		id = DefaultProviderID
	}
	return &LocalProvider{
		id:       id,
		cfg:      cfg,
		engine:   engine,
		specs:    specs,
		provider: provider,
	}
}

func (p *LocalProvider) ID() string {
	return p.id
}

func (p *LocalProvider) Inspect(ctx context.Context) (*asrcontract.ProviderInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &asrcontract.ProviderInfo{
		ContractVersion: asrcontract.ContractVersionV1,
		Provider: asrcontract.ProviderIdentity{
			ID:     p.id,
			Name:   "Sherpa ONNX",
			Vendor: "scriberr",
		},
		Runtime: asrcontract.RuntimeInfo{
			DeviceBackends:       []string{"cpu", "cuda"},
			ActiveBackend:        p.provider.String(),
			SupportsConcurrent:   false,
			MaxConcurrentJobs:    1,
			ProviderCapabilities: []asrcontract.Capability{asrcontract.CapabilityTranscription, asrcontract.CapabilityDiarization},
		},
		AudioInput: asrcontract.AudioInputSpec{
			RequiredSampleRate: 16000,
			RequiredChannels:   1,
			Formats:            []string{"wav"},
			PathMode:           asrcontract.PathModeMountedFile,
		},
	}, nil
}

func (p *LocalProvider) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	loaded := make(map[string]struct{})
	if loader, ok := p.engine.(modelLoader); ok {
		for _, id := range loader.ListLoadedModels() {
			loaded[string(id)] = struct{}{}
		}
	}
	out := make([]asrcontract.ModelCard, 0, len(p.specs))
	for _, spec := range p.specs {
		_, isLoaded := loaded[string(spec.ID)]
		out = append(out, asrcontract.ModelCard{
			ID:           string(spec.ID),
			DisplayName:  spec.DisplayName,
			Provider:     p.id,
			Family:       modelFamilyName(spec.Family),
			Version:      spec.ModelType,
			Installed:    p.engine.IsModelInstalled(string(spec.ID)),
			Loaded:       isLoaded,
			Default:      isDefaultModel(spec.ID),
			Tasks:        tasksForFamily(spec.Family),
			Capabilities: capabilitiesForModelCard(spec.Family),
			ResourceRequirements: asrcontract.ResourceRequirements{
				Backends: []string{p.provider.String()},
			},
		})
	}
	return out, nil
}

func (p *LocalProvider) Status(ctx context.Context) (*asrcontract.ProviderStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	loaded, err := p.LoadedModels(ctx)
	if err != nil {
		return nil, err
	}
	return &asrcontract.ProviderStatus{
		State:        asrcontract.ProviderStateIdle,
		LoadedModels: loaded,
		Capacity: asrcontract.ProviderCapacity{
			MaxConcurrentJobs: 1,
			AvailableSlots:    1,
		},
	}, nil
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
	loader, ok := p.engine.(modelLoader)
	if !ok {
		return []asrcontract.LoadedModel{}, nil
	}
	ids := loader.ListLoadedModels()
	out := make([]asrcontract.LoadedModel, 0, len(ids))
	for _, id := range ids {
		out = append(out, asrcontract.LoadedModel{ID: string(id)})
	}
	return out, nil
}

func (p *LocalProvider) Capabilities(ctx context.Context) ([]ModelCapability, error) {
	models, err := p.Models(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ModelCapability, 0, len(p.specs))
	for _, model := range models {
		capability := ModelCapability{
			ID:           model.ID,
			Name:         model.DisplayName,
			Provider:     p.id,
			Installed:    model.Installed,
			Default:      model.Default,
			Capabilities: legacyCapabilities(model.Capabilities),
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
		ModelID:                 modelID,
		AudioPath:               req.AudioPath,
		Language:                req.Language,
		Task:                    task,
		TailPaddings:            req.TailPaddings,
		EnableTokenTimestamps:   &enableTokenTimestamps,
		EnableSegmentTimestamps: &enableSegmentTimestamps,
		CanarySourceLanguage:    req.CanarySourceLanguage,
		CanaryTargetLanguage:    req.CanaryTargetLanguage,
		CanaryUsePunctuation:    req.CanaryUsePunctuation,
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
		ModelID:        modelID,
		AudioPath:      req.AudioPath,
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

func capabilitiesForFamily(family speechmodels.Family) []string {
	switch family {
	case speechmodels.FamilyDiarize:
		return []string{"diarization"}
	case speechmodels.FamilyWhisper, speechmodels.FamilyNemo, speechmodels.FamilyCanary:
		return []string{"transcription", "word_timestamps"}
	default:
		return []string{}
	}
}

func capabilitiesForModelCard(family speechmodels.Family) asrcontract.Capabilities {
	switch family {
	case speechmodels.FamilyDiarize:
		return asrcontract.Capabilities{Diarization: true}
	case speechmodels.FamilyWhisper, speechmodels.FamilyNemo, speechmodels.FamilyCanary:
		return asrcontract.Capabilities{
			Transcription:     true,
			WordTimestamps:    true,
			SegmentTimestamps: true,
		}
	default:
		return asrcontract.Capabilities{}
	}
}

func legacyCapabilities(capabilities asrcontract.Capabilities) []string {
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

func tasksForFamily(family speechmodels.Family) []asrcontract.Task {
	switch family {
	case speechmodels.FamilyWhisper, speechmodels.FamilyNemo, speechmodels.FamilyCanary:
		return []asrcontract.Task{asrcontract.TaskTranscribe}
	default:
		return nil
	}
}

func modelFamilyName(family speechmodels.Family) string {
	switch family {
	case speechmodels.FamilyWhisper:
		return "whisper"
	case speechmodels.FamilyNemo:
		return "nemo_transducer"
	case speechmodels.FamilyCanary:
		return "canary"
	default:
		return string(family)
	}
}

func isDefaultModel(id speechmodels.ModelID) bool {
	return string(id) == DefaultTranscriptionModel || string(id) == DefaultDiarizationModel
}

func coalesceInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
