package engineprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	appconfig "scriberr/internal/config"
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

func (p *LocalProvider) Capabilities(ctx context.Context) ([]ModelCapability, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]ModelCapability, 0, len(p.specs))
	for _, spec := range p.specs {
		capability := ModelCapability{
			ID:           string(spec.ID),
			Name:         spec.DisplayName,
			Provider:     p.id,
			Installed:    p.engine.IsModelInstalled(string(spec.ID)),
			Default:      isDefaultModel(spec.ID),
			Capabilities: capabilitiesForFamily(spec.Family),
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
