package chat

import (
	"context"
	"fmt"
	"time"

	"scriberr/internal/models"
)

type ContextStore interface {
	FindSessionForUser(ctx context.Context, userID uint, sessionID string) (*models.ChatSession, error)
	FindCompletedTranscriptionForUser(ctx context.Context, userID uint, transcriptionID string) (*models.TranscriptionJob, error)
	UpsertContextSource(ctx context.Context, userID uint, sessionID string, source *models.ChatContextSource) (*models.ChatContextSource, error)
	SetContextSourceEnabled(ctx context.Context, userID uint, sessionID string, sourceID string, enabled bool) error
	ListContextSources(ctx context.Context, userID uint, sessionID string, enabledOnly bool) ([]models.ChatContextSource, error)
}

type ContextBuilder struct {
	store     ContextStore
	estimator TokenEstimator
}

func NewContextBuilder(store ContextStore, estimator TokenEstimator) *ContextBuilder {
	if estimator == nil {
		estimator = ApproxTokenEstimator{}
	}
	return &ContextBuilder{store: store, estimator: estimator}
}

type SourceMutation struct {
	Source *models.ChatContextSource
}

func (b *ContextBuilder) AddTranscriptSource(ctx context.Context, userID uint, sessionID string, transcriptionID string, kind models.ChatContextSourceKind) (*SourceMutation, error) {
	if kind == "" {
		kind = models.ChatContextSourceKindTranscript
	}
	job, err := b.store.FindCompletedTranscriptionForUser(ctx, userID, transcriptionID)
	if err != nil {
		return nil, err
	}
	plain, err := transcriptPlaintextFromJob(job)
	if err != nil {
		return nil, err
	}
	sourceVersion := job.UpdatedAt.UTC().Format(time.RFC3339Nano)
	source, err := b.store.UpsertContextSource(ctx, userID, sessionID, &models.ChatContextSource{
		TranscriptionID:   transcriptionID,
		Kind:              kind,
		Enabled:           true,
		PlainTextSnapshot: &plain,
		SourceVersion:     &sourceVersion,
		CompactionStatus:  models.ChatContextCompactionStatusNone,
		MetadataJSON:      "{}",
	})
	if err != nil {
		return nil, err
	}
	return &SourceMutation{Source: source}, nil
}

func (b *ContextBuilder) AddParentSource(ctx context.Context, userID uint, sessionID string) (*SourceMutation, error) {
	session, err := b.store.FindSessionForUser(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	return b.AddTranscriptSource(ctx, userID, sessionID, session.ParentTranscriptionID, models.ChatContextSourceKindParentTranscript)
}

func (b *ContextBuilder) SetSourceEnabled(ctx context.Context, userID uint, sessionID string, sourceID string, enabled bool) error {
	return b.store.SetContextSourceEnabled(ctx, userID, sessionID, sourceID, enabled)
}

type BuildOptions struct {
	Budget ContextBudget
}

type BuiltContext struct {
	Sources         []BuiltContextSource
	Content         string
	TokensEstimated int
	Truncated       bool
}

type BuiltContextSource struct {
	Source          models.ChatContextSource
	Content         string
	TokensEstimated int
	Truncated       bool
}

func (b *ContextBuilder) Build(ctx context.Context, userID uint, sessionID string, opts BuildOptions) (*BuiltContext, error) {
	sources, err := b.store.ListContextSources(ctx, userID, sessionID, true)
	if err != nil {
		return nil, err
	}
	remaining := opts.Budget.AvailableTranscriptTokens()
	if opts.Budget.ContextWindow <= 0 {
		remaining = 0
	}
	result := &BuiltContext{Sources: make([]BuiltContextSource, 0, len(sources))}
	for _, source := range sources {
		content, err := b.sourceContent(ctx, userID, &source)
		if err != nil {
			return nil, err
		}
		maxTokens := remaining
		if opts.Budget.ContextWindow <= 0 {
			maxTokens = b.estimator.EstimateTokens(content)
		}
		fitted, estimated, truncated := FitTextToTokenBudget(content, maxTokens, b.estimator)
		if opts.Budget.ContextWindow > 0 {
			remaining -= estimated
			if remaining < 0 {
				remaining = 0
			}
		}
		result.Sources = append(result.Sources, BuiltContextSource{
			Source:          source,
			Content:         fitted,
			TokensEstimated: estimated,
			Truncated:       truncated,
		})
		if result.Content != "" && fitted != "" {
			result.Content += "\n\n"
		}
		result.Content += fitted
		result.TokensEstimated += estimated
		result.Truncated = result.Truncated || truncated
	}
	return result, nil
}

func (b *ContextBuilder) sourceContent(ctx context.Context, userID uint, source *models.ChatContextSource) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.CompactedSnapshot != nil && *source.CompactedSnapshot != "" {
		return *source.CompactedSnapshot, nil
	}
	if source.PlainTextSnapshot != nil && *source.PlainTextSnapshot != "" {
		return *source.PlainTextSnapshot, nil
	}
	job, err := b.store.FindCompletedTranscriptionForUser(ctx, userID, source.TranscriptionID)
	if err != nil {
		return "", err
	}
	return transcriptPlaintextFromJob(job)
}

func transcriptPlaintextFromJob(job *models.TranscriptionJob) (string, error) {
	if job == nil || job.Transcript == nil || *job.Transcript == "" {
		return "", fmt.Errorf("completed transcript has no transcript text")
	}
	return PlainTranscriptText(*job.Transcript)
}
