package summarization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"scriberr/internal/llm"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/orchestrator"
	"scriberr/pkg/logger"

	"gorm.io/gorm"
)

const summaryPrompt = `You are an expert summarization system designed to extract the core essence of long-form transcripts.

Your task:
Given a transcript, produce a single, concise paragraph (4-6 sentences max) that captures:
- The central theme or purpose of the discussion
- The key ideas or arguments presented
- The overall direction or conclusion (if any)
- The tone or intent (e.g., exploratory, instructional, persuasive, reflective)

Guidelines:
- Focus on meaning, not chronology - do NOT summarize turn-by-turn.
- Prioritize signal over detail - omit examples, anecdotes, and filler unless they are essential to the main idea.
- Synthesize, don't compress - rewrite in your own words rather than extracting phrases.
- Maintain coherence - the paragraph should read like a well-written abstract, not bullet points stitched together.
- Avoid vague language like "they talk about various things" - be specific about themes.
- Do NOT include quotes, speaker labels, or meta commentary.

Output format:
- Exactly one paragraph
- No bullet points, no headings, no extra explanation

If the transcript is noisy or unstructured:
- Infer the most likely core theme and summarize that confidently.

Now summarize the following transcript:
`

type EventPublisher interface {
	PublishSummaryStatus(ctx context.Context, event StatusEvent)
}

type StatusEvent struct {
	Name            string
	SummaryID       string
	TranscriptionID string
	UserID          uint
	Status          string
	Truncated       bool
}

type Config struct {
	PollInterval time.Duration
	StopTimeout  time.Duration
}

type Service struct {
	summaries repository.SummaryRepository
	llmConfig repository.LLMConfigRepository
	jobs      repository.JobRepository
	events    EventPublisher
	cfg       Config

	mu      sync.Mutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	wake    chan struct{}
}

func NewService(summaries repository.SummaryRepository, llmConfig repository.LLMConfigRepository, jobs repository.JobRepository, cfg Config) *Service {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 30 * time.Second
	}
	return &Service{
		summaries: summaries,
		llmConfig: llmConfig,
		jobs:      jobs,
		cfg:       cfg,
		wake:      make(chan struct{}, 1),
	}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.summaries == nil || s.llmConfig == nil || s.jobs == nil {
		return fmt.Errorf("summary repositories are required")
	}
	if recovered, err := s.summaries.RecoverProcessingSummaries(ctx); err != nil {
		return err
	} else if recovered > 0 {
		logger.Info("Recovered processing summary jobs", "count", recovered)
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true
	s.wg.Add(1)
	go s.workerLoop()
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.started = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.cfg.StopTimeout):
		return context.DeadlineExceeded
	}
}

func (s *Service) EnqueueForTranscription(ctx context.Context, job *models.TranscriptionJob) error {
	if job == nil || job.Status != models.StatusCompleted || job.Transcript == nil || strings.TrimSpace(*job.Transcript) == "" {
		return nil
	}
	config, err := s.llmConfig.GetActiveByUser(ctx, job.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !llmReady(config) {
		return nil
	}
	summary, created, err := s.summaries.EnqueueAutomaticSummary(ctx, job.ID, job.UserID, strings.TrimSpace(*config.SmallModel), config.Provider)
	if err != nil {
		return err
	}
	if created {
		s.publish(ctx, "summary.pending", summary)
		s.notify()
	}
	return nil
}

func (s *Service) workerLoop() {
	defer s.wg.Done()
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
		case <-timer.C:
		}
		if err := s.claimAndProcess(); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Summary worker failed", "error", err)
		}
		timer.Reset(s.cfg.PollInterval)
	}
}

func (s *Service) claimAndProcess() error {
	summary, err := s.summaries.ClaimNextPendingSummary(s.ctx, time.Now())
	if err != nil {
		return err
	}
	s.publish(s.ctx, "summary.processing", summary)
	if err := s.processSummary(s.ctx, summary); err != nil {
		message := sanitizeError(err)
		if failErr := s.summaries.FailSummary(context.Background(), summary.ID, message, time.Now()); failErr != nil {
			return failErr
		}
		summary.Status = "failed"
		summary.ErrorMessage = &message
		s.publish(context.Background(), "summary.failed", summary)
		return nil
	}
	return nil
}

func (s *Service) processSummary(ctx context.Context, summary *models.Summary) error {
	job, err := s.jobs.FindByID(ctx, summary.TranscriptionID)
	if err != nil {
		return err
	}
	config, err := s.llmConfig.GetActiveByUser(ctx, summary.UserID)
	if err != nil {
		return err
	}
	if !llmReady(config) {
		return fmt.Errorf("LLM provider is not fully configured")
	}
	client, err := clientForConfig(config)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(*config.SmallModel)
	contextWindow, err := client.GetContextWindow(ctx, model)
	if err != nil || contextWindow <= 0 {
		contextWindow = 4096
	}
	transcriptJSON := ""
	if job.Transcript != nil {
		transcriptJSON = *job.Transcript
	}
	transcriptText, err := plainTranscriptText(transcriptJSON)
	if err != nil {
		return err
	}
	input, truncated := fitTranscriptToContext(transcriptText, contextWindow)
	messages := []llm.ChatMessage{{Role: "user", Content: summaryPrompt + "\n\n" + input}}
	response, err := client.ChatCompletion(ctx, model, messages, 0)
	if err != nil {
		return err
	}
	content := ""
	if response != nil && len(response.Choices) > 0 {
		content = strings.TrimSpace(response.Choices[0].Message.Content)
	}
	if content == "" {
		return fmt.Errorf("summary provider returned empty content")
	}
	if err := s.summaries.CompleteSummary(context.Background(), summary.ID, content, truncated, contextWindow, len(transcriptText), time.Now()); err != nil {
		return err
	}
	summary.Content = content
	summary.Status = "completed"
	summary.TranscriptTruncated = truncated
	summary.ContextWindow = contextWindow
	summary.InputCharacters = len(transcriptText)
	s.publish(context.Background(), "summary.completed", summary)
	return nil
}

func llmReady(config *models.LLMConfig) bool {
	return config != nil &&
		strings.TrimSpace(config.Provider) != "" &&
		strings.TrimSpace(llmBaseURL(config)) != "" &&
		config.LargeModel != nil && strings.TrimSpace(*config.LargeModel) != "" &&
		config.SmallModel != nil && strings.TrimSpace(*config.SmallModel) != ""
}

func clientForConfig(config *models.LLMConfig) (llm.Service, error) {
	baseURL := llmBaseURL(config)
	switch config.Provider {
	case "ollama":
		return llm.NewOllamaService(baseURL), nil
	case "openai_compatible", "openai":
		apiKey := ""
		if config.APIKey != nil {
			apiKey = strings.TrimSpace(*config.APIKey)
		}
		return llm.NewOpenAIService(apiKey, &baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider")
	}
}

func llmBaseURL(config *models.LLMConfig) string {
	if config.BaseURL != nil && strings.TrimSpace(*config.BaseURL) != "" {
		return strings.TrimSpace(*config.BaseURL)
	}
	if config.OpenAIBaseURL != nil {
		return strings.TrimSpace(*config.OpenAIBaseURL)
	}
	return ""
}

func plainTranscriptText(value string) (string, error) {
	transcript, err := orchestrator.ParseStoredTranscript(value)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(transcript.Segments))
	for _, segment := range transcript.Segments {
		if text := strings.TrimSpace(segment.Text); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n"), nil
	}
	return strings.TrimSpace(transcript.Text), nil
}

func fitTranscriptToContext(transcript string, contextWindow int) (string, bool) {
	const charsPerToken = 4
	const completionReserveTokens = 512
	promptTokens := (len(summaryPrompt) / charsPerToken) + 1
	budgetTokens := contextWindow - promptTokens - completionReserveTokens
	if budgetTokens < 256 {
		budgetTokens = 256
	}
	maxChars := budgetTokens * charsPerToken
	if len(transcript) <= maxChars {
		return transcript, false
	}
	return strings.TrimSpace(transcript[:maxChars]), true
}

func (s *Service) publish(ctx context.Context, name string, summary *models.Summary) {
	if s.events == nil || summary == nil {
		return
	}
	s.events.PublishSummaryStatus(ctx, StatusEvent{
		Name:            name,
		SummaryID:       summary.ID,
		TranscriptionID: summary.TranscriptionID,
		UserID:          summary.UserID,
		Status:          summary.Status,
		Truncated:       summary.TranscriptTruncated,
	})
}

func (s *Service) notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 300 {
		return message[:300] + "..."
	}
	return message
}
