package chat

import (
	"context"
	"fmt"
	"strings"

	"scriberr/internal/models"
)

const (
	defaultCompactionThresholdRatio = 0.80
	defaultRecentMessageWindow      = 10
	defaultMaxMessagesForCompaction = 1000
)

type CompactionStore interface {
	ContextStore
	UpdateContextSourceCompaction(ctx context.Context, userID uint, sessionID string, sourceID string, status models.ChatContextCompactionStatus, compactedSnapshot *string) error
	ListMessages(ctx context.Context, userID uint, sessionID string, offset, limit int) ([]models.ChatMessage, int64, error)
	SaveContextSummary(ctx context.Context, summary *models.ChatContextSummary) error
	ListContextSummaries(ctx context.Context, userID uint, sessionID string, summaryType models.ChatContextSummaryType) ([]models.ChatContextSummary, error)
}

type TextCompactor interface {
	Compact(ctx context.Context, req CompactionRequest) (string, error)
}

type CompactionRequest struct {
	Type         models.ChatContextSummaryType
	Input        string
	TargetTokens int
	Provider     string
	Model        string
}

type ExtractiveCompactor struct {
	Estimator TokenEstimator
}

func (c ExtractiveCompactor) Compact(ctx context.Context, req CompactionRequest) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	estimator := c.Estimator
	if estimator == nil {
		estimator = ApproxTokenEstimator{}
	}
	prefix := "Compacted context:\n"
	target := req.TargetTokens - estimator.EstimateTokens(prefix)
	if target < 1 {
		target = 1
	}
	fitted, _, _ := FitTextToTokenBudget(strings.TrimSpace(req.Input), target, estimator)
	if fitted == "" && strings.TrimSpace(req.Input) != "" {
		fitted = firstRunes(strings.TrimSpace(req.Input), DefaultCharsPerToken)
	}
	return prefix + fitted, nil
}

type CompactionConfig struct {
	ThresholdRatio      float64
	RecentMessageWindow int
	MaxMessages         int
}

func (c CompactionConfig) withDefaults() CompactionConfig {
	if c.ThresholdRatio <= 0 || c.ThresholdRatio > 1 {
		c.ThresholdRatio = defaultCompactionThresholdRatio
	}
	if c.RecentMessageWindow <= 0 {
		c.RecentMessageWindow = defaultRecentMessageWindow
	}
	if c.MaxMessages <= 0 {
		c.MaxMessages = defaultMaxMessagesForCompaction
	}
	return c
}

type Compactor struct {
	store         CompactionStore
	estimator     TokenEstimator
	textCompactor TextCompactor
	config        CompactionConfig
}

func NewCompactor(store CompactionStore, estimator TokenEstimator, textCompactor TextCompactor, config CompactionConfig) *Compactor {
	if estimator == nil {
		estimator = ApproxTokenEstimator{}
	}
	if textCompactor == nil {
		textCompactor = ExtractiveCompactor{Estimator: estimator}
	}
	return &Compactor{
		store:         store,
		estimator:     estimator,
		textCompactor: textCompactor,
		config:        config.withDefaults(),
	}
}

type TranscriptCompactionResult struct {
	SourceID          string
	SummaryID         string
	InputTokens       int
	OutputTokens      int
	Compacted         bool
	CompactionStatus  models.ChatContextCompactionStatus
	SourceTranscripts string
}

func (c *Compactor) CompactOversizedTranscripts(ctx context.Context, userID uint, sessionID string, budget ContextBudget, provider string, model string) ([]TranscriptCompactionResult, error) {
	if budget.ContextWindow <= 0 {
		return nil, fmt.Errorf("context window is required for transcript compaction")
	}
	targetTokens := budget.AvailableTranscriptTokens()
	if targetTokens < 1 {
		targetTokens = 1
	}
	sources, err := c.store.ListContextSources(ctx, userID, sessionID, true)
	if err != nil {
		return nil, err
	}
	results := make([]TranscriptCompactionResult, 0, len(sources))
	for _, source := range sources {
		content, err := c.sourcePlainText(ctx, userID, source)
		if err != nil {
			return nil, err
		}
		inputTokens := c.estimator.EstimateTokens(content)
		result := TranscriptCompactionResult{
			SourceID:          source.ID,
			InputTokens:       inputTokens,
			CompactionStatus:  source.CompactionStatus,
			SourceTranscripts: source.TranscriptionID,
		}
		if inputTokens <= targetTokens {
			results = append(results, result)
			continue
		}
		if err := c.store.UpdateContextSourceCompaction(ctx, userID, sessionID, source.ID, models.ChatContextCompactionStatusCompacting, nil); err != nil {
			return nil, err
		}
		compacted, err := c.textCompactor.Compact(ctx, CompactionRequest{
			Type:         models.ChatContextSummaryTypeTranscript,
			Input:        content,
			TargetTokens: targetTokens,
			Provider:     provider,
			Model:        model,
		})
		if err != nil {
			_ = c.store.UpdateContextSourceCompaction(ctx, userID, sessionID, source.ID, models.ChatContextCompactionStatusFailed, nil)
			return nil, err
		}
		outputTokens := c.estimator.EstimateTokens(compacted)
		summary := &models.ChatContextSummary{
			UserID:                userID,
			ChatSessionID:         sessionID,
			SummaryType:           models.ChatContextSummaryTypeTranscript,
			SourceTranscriptionID: &source.TranscriptionID,
			Content:               compacted,
			Model:                 model,
			Provider:              provider,
			InputTokensEstimated:  inputTokens,
			OutputTokensEstimated: outputTokens,
		}
		if err := c.store.SaveContextSummary(ctx, summary); err != nil {
			_ = c.store.UpdateContextSourceCompaction(ctx, userID, sessionID, source.ID, models.ChatContextCompactionStatusFailed, nil)
			return nil, err
		}
		if err := c.store.UpdateContextSourceCompaction(ctx, userID, sessionID, source.ID, models.ChatContextCompactionStatusCompacted, &compacted); err != nil {
			return nil, err
		}
		result.SummaryID = summary.ID
		result.OutputTokens = outputTokens
		result.Compacted = true
		result.CompactionStatus = models.ChatContextCompactionStatusCompacted
		results = append(results, result)
	}
	return results, nil
}

type SessionCompactionResult struct {
	SummaryID              string
	SourceMessageThroughID string
	InputTokens            int
	OutputTokens           int
	Compacted              bool
	RecentMessageCount     int
}

func (c *Compactor) CompactSessionHistory(ctx context.Context, userID uint, sessionID string, contextWindow int, provider string, model string) (*SessionCompactionResult, error) {
	if contextWindow <= 0 {
		return nil, fmt.Errorf("context window is required for session compaction")
	}
	messages, _, err := c.store.ListMessages(ctx, userID, sessionID, 0, c.config.MaxMessages)
	if err != nil {
		return nil, err
	}
	result := &SessionCompactionResult{RecentMessageCount: min(len(messages), c.config.RecentMessageWindow)}
	if len(messages) <= c.config.RecentMessageWindow {
		return result, nil
	}
	totalTokens := c.estimator.EstimateTokens(formatMessagesForCompaction(messages))
	thresholdTokens := int(float64(contextWindow) * c.config.ThresholdRatio)
	if thresholdTokens < 1 {
		thresholdTokens = 1
	}
	if totalTokens < thresholdTokens {
		result.InputTokens = totalTokens
		return result, nil
	}

	boundaryIndex := len(messages) - c.config.RecentMessageWindow - 1
	if boundaryIndex < 0 {
		return result, nil
	}
	olderMessages := messages[:boundaryIndex+1]
	input := formatMessagesForCompaction(olderMessages)
	inputTokens := c.estimator.EstimateTokens(input)
	targetTokens := max(1, thresholdTokens/2)
	compacted, err := c.textCompactor.Compact(ctx, CompactionRequest{
		Type:         models.ChatContextSummaryTypeSession,
		Input:        input,
		TargetTokens: targetTokens,
		Provider:     provider,
		Model:        model,
	})
	if err != nil {
		return nil, err
	}
	outputTokens := c.estimator.EstimateTokens(compacted)
	sourceMessageThroughID := olderMessages[len(olderMessages)-1].ID
	summary := &models.ChatContextSummary{
		UserID:                 userID,
		ChatSessionID:          sessionID,
		SummaryType:            models.ChatContextSummaryTypeSession,
		SourceMessageThroughID: &sourceMessageThroughID,
		Content:                compacted,
		Model:                  model,
		Provider:               provider,
		InputTokensEstimated:   inputTokens,
		OutputTokensEstimated:  outputTokens,
	}
	if err := c.store.SaveContextSummary(ctx, summary); err != nil {
		return nil, err
	}
	result.SummaryID = summary.ID
	result.SourceMessageThroughID = sourceMessageThroughID
	result.InputTokens = inputTokens
	result.OutputTokens = outputTokens
	result.Compacted = true
	return result, nil
}

func (c *Compactor) sourcePlainText(ctx context.Context, userID uint, source models.ChatContextSource) (string, error) {
	if source.PlainTextSnapshot != nil && *source.PlainTextSnapshot != "" {
		return *source.PlainTextSnapshot, nil
	}
	job, err := c.store.FindCompletedTranscriptionForUser(ctx, userID, source.TranscriptionID)
	if err != nil {
		return "", err
	}
	return transcriptPlaintextFromJob(job)
}

func formatMessagesForCompaction(messages []models.ChatMessage) string {
	var builder strings.Builder
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(roleLabel(message.Role))
		builder.WriteString(": ")
		builder.WriteString(content)
	}
	return builder.String()
}

func firstRunes(value string, count int) string {
	runes := []rune(value)
	if count > len(runes) {
		count = len(runes)
	}
	return string(runes[:count])
}

func roleLabel(role models.ChatMessageRole) string {
	switch role {
	case models.ChatMessageRoleUser:
		return "User"
	case models.ChatMessageRoleAssistant:
		return "Assistant"
	case models.ChatMessageRoleSystem:
		return "System"
	case models.ChatMessageRoleTool:
		return "Tool"
	default:
		return "Message"
	}
}
