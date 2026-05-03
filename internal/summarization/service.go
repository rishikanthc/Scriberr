package summarization

import (
	"context"
	"encoding/json"
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

const titlePrompt = `Generate a concise title for this audio recording based on the current title and summary below.

Rules:
- Maximum 7 words.
- Return only the title.
- No quotes.
- No markdown.
- No trailing punctuation.
- Be specific and descriptive.
- Do not use generic titles like "Audio Recording", "Transcript Summary", or "Discussion".
- If the current title already faithfully describes the audio and is 7 words or fewer, repeat the current title exactly.
- If the current title is generic, misleading, too vague, or longer than 7 words, provide an improved title.
`

const titlePromptCurrentTitleSection = `Current title:
`

const titlePromptSummarySection = `

Summary:
`

const descriptionPrompt = `Write a description for this audio recording.

Rules:
- Output exactly 2 lines.
- Each line must be a concise sentence fragment or sentence.
- No bullets, numbering, quotes, markdown, heading, prefix, or extra commentary.
- Use only the summary and outline below.
- Do not mention "summary", "outline", or "transcript".
`

const descriptionPromptSummarySection = `Summary:
`

const descriptionPromptOutlineSection = `

Outline:
`

const markdownTypesetInstructions = `Format the response as clean Markdown suitable for read-only typeset rendering.

Use Markdown structure only when it improves scanning: concise headings, bullet lists, numbered lists, tables, emphasis, or checkboxes where appropriate.
Do not wrap the entire response in a code fence.
Do not include front matter, HTML, or editor instructions.
Return only the Markdown content.`

type EventPublisher interface {
	PublishSummaryStatus(ctx context.Context, event StatusEvent)
	PublishFileEvent(ctx context.Context, name string, payload map[string]any)
}

type StatusEvent struct {
	Name            string
	SummaryID       string
	TranscriptionID string
	WidgetRunID     string
	WidgetID        string
	UserID          uint
	Status          string
	Truncated       bool
}

type Config struct {
	StopTimeout time.Duration
}

type UserSettingsReader interface {
	FindByID(ctx context.Context, id interface{}) (*models.User, error)
}

var ErrNotFound = errors.New("summary not found")

type Service struct {
	summaries repository.SummaryRepository
	llmConfig repository.LLMConfigRepository
	jobs      repository.JobRepository
	users     UserSettingsReader
	events    EventPublisher
	cfg       Config
	clientFor func(*models.LLMConfig) (llm.Service, error)

	mu      sync.Mutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	wake    chan struct{}
}

func NewService(summaries repository.SummaryRepository, llmConfig repository.LLMConfigRepository, jobs repository.JobRepository, cfg Config) *Service {
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 30 * time.Second
	}
	return &Service{
		summaries: summaries,
		llmConfig: llmConfig,
		jobs:      jobs,
		cfg:       cfg,
		clientFor: clientForConfig,
		wake:      make(chan struct{}, 1),
	}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) SetUserSettingsReader(users UserSettingsReader) {
	s.users = users
}

func (s *Service) LatestForTranscription(ctx context.Context, userID uint, transcriptionID string) (*models.Summary, error) {
	summary, err := s.summaries.GetLatestSummary(ctx, transcriptionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if summary.UserID != userID {
		return nil, ErrNotFound
	}
	return summary, nil
}

func (s *Service) ListWidgetRunsForTranscription(ctx context.Context, userID uint, transcriptionID string) ([]models.SummaryWidgetRun, error) {
	return s.summaries.ListSummaryWidgetRunsByTranscription(ctx, transcriptionID, userID)
}

func (s *Service) ListWidgets(ctx context.Context, userID uint) ([]models.SummaryWidget, error) {
	return s.summaries.ListSummaryWidgetsByUser(ctx, userID)
}

func (s *Service) FindWidget(ctx context.Context, userID uint, widgetID string) (*models.SummaryWidget, error) {
	widget, err := s.summaries.FindSummaryWidgetByIDForUser(ctx, widgetID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return widget, err
}

func (s *Service) CreateWidget(ctx context.Context, widget *models.SummaryWidget) error {
	return s.summaries.CreateSummaryWidget(ctx, widget)
}

func (s *Service) UpdateWidget(ctx context.Context, widget *models.SummaryWidget) error {
	return s.summaries.UpdateSummaryWidget(ctx, widget)
}

func (s *Service) DeleteWidget(ctx context.Context, userID uint, widgetID string) error {
	if _, err := s.FindWidget(ctx, userID, widgetID); err != nil {
		return err
	}
	return s.summaries.DeleteSummaryWidget(ctx, widgetID, userID)
}

func (s *Service) WidgetNameExists(ctx context.Context, userID uint, exceptID string, name string) (bool, error) {
	widgets, err := s.summaries.ListSummaryWidgetsByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, widget := range widgets {
		if widget.ID != exceptID && strings.EqualFold(strings.TrimSpace(widget.Name), strings.TrimSpace(name)) {
			return true, nil
		}
	}
	return false, nil
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
	if recovered, err := s.summaries.RecoverProcessingSummaryWidgetRuns(ctx); err != nil {
		return err
	} else if recovered > 0 {
		logger.Info("Recovered processing summary widget jobs", "count", recovered)
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true
	s.wg.Add(1)
	go s.workerLoop()
	s.wg.Add(1)
	go s.generateMissingRecordingTitles()
	s.notify()
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
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
		}
		s.drainPending()
	}
}

func (s *Service) drainPending() {
	for {
		progressed := false
		if err := s.claimAndProcessSummary(); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Try widget runs below before deciding the worker is idle.
			} else {
				logger.Error("Summary worker failed", "error", err)
				return
			}
		} else {
			progressed = true
		}
		if err := s.claimAndProcessWidgetRun(); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if !progressed {
					return
				}
				continue
			}
			logger.Error("Summary widget worker failed", "error", err)
			return
		} else {
			progressed = true
		}
		if !progressed {
			return
		}
	}
}

func (s *Service) generateMissingRecordingTitles() {
	defer s.wg.Done()
	summaries, err := s.summaries.ListCompletedSummariesForTitleGeneration(s.ctx, 25)
	if err != nil {
		logger.Error("Failed to list summaries for title generation", "error", err)
		return
	}
	for i := range summaries {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		if err := s.generateTitleForSummary(s.ctx, &summaries[i], summaries[i].Content); err != nil {
			logger.Error("Automatic title generation recovery failed", "summary_id", summaries[i].ID, "transcription_id", summaries[i].TranscriptionID, "error", err)
		}
	}
}

func (s *Service) claimAndProcessSummary() error {
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
	if err := s.enqueueWidgetsForSummary(context.Background(), summary); err != nil {
		logger.Error("Summary widget enqueue failed", "summary_id", summary.ID, "error", err)
	}
	return nil
}

func (s *Service) claimAndProcessWidgetRun() error {
	run, err := s.summaries.ClaimNextPendingSummaryWidgetRun(s.ctx, time.Now())
	if err != nil {
		return err
	}
	s.publishWidgetRun(s.ctx, "summary_widget.processing", run)
	if err := s.processWidgetRun(s.ctx, run); err != nil {
		message := sanitizeError(err)
		if failErr := s.summaries.FailSummaryWidgetRun(context.Background(), run.ID, message, time.Now()); failErr != nil {
			return failErr
		}
		run.Status = "failed"
		run.ErrorMessage = &message
		s.publishWidgetRun(context.Background(), "summary_widget.failed", run)
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
	client, err := s.clientFor(config)
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
	if truncated {
		summary.TranscriptTruncated = true
		summary.ContextWindow = contextWindow
		summary.InputCharacters = len(transcriptText)
		s.publish(ctx, "summary.truncated", summary)
	}
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
	if err := s.generateTitleForSummary(context.Background(), summary, content); err != nil {
		logger.Error("Automatic title generation failed", "summary_id", summary.ID, "transcription_id", summary.TranscriptionID, "error", err)
	}
	if err := s.generateDescriptionForSummary(context.Background(), summary); err != nil {
		logger.Error("Automatic description generation failed", "summary_id", summary.ID, "transcription_id", summary.TranscriptionID, "error", err)
	}
	return nil
}

func (s *Service) generateTitleForSummary(ctx context.Context, summary *models.Summary, summaryContent string) error {
	if summary == nil || strings.TrimSpace(summaryContent) == "" {
		return nil
	}
	if enabled, err := s.autoRenameEnabled(ctx, summary.UserID); err != nil || !enabled {
		return err
	}
	job, err := s.jobs.FindByID(ctx, summary.TranscriptionID)
	if err != nil {
		return err
	}
	recording := job
	if job.SourceFileHash != nil && strings.TrimSpace(*job.SourceFileHash) != "" {
		parent, err := s.jobs.FindByID(ctx, strings.TrimSpace(*job.SourceFileHash))
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if parent != nil {
			recording = parent
		}
	}
	if recording.LLMTitleGenerated {
		return nil
	}
	config, err := s.llmConfig.GetActiveByUser(ctx, summary.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !titleLLMReady(config) {
		return nil
	}
	client, err := s.clientFor(config)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(*config.SmallModel)
	currentTitle := strings.TrimSpace(valueOrEmpty(recording.Title))
	prompt := titlePrompt + titlePromptCurrentTitleSection + currentTitle + titlePromptSummarySection + strings.TrimSpace(summaryContent)
	response, err := client.ChatCompletion(ctx, model, []llm.ChatMessage{{Role: "user", Content: prompt}}, 0)
	if err != nil {
		return err
	}
	content := ""
	if response != nil && len(response.Choices) > 0 {
		content = response.Choices[0].Message.Content
	}
	title := sanitizeGeneratedTitle(content)
	if title == "" {
		return nil
	}
	generatedAt := time.Now()
	if err := s.jobs.UpdateLLMGeneratedTitle(ctx, summary.TranscriptionID, recording.ID, title, generatedAt); err != nil {
		return err
	}
	if s.events != nil && !sameGeneratedTitle(currentTitle, title) {
		s.events.PublishFileEvent(ctx, "file.updated", map[string]any{
			"id":     "file_" + recording.ID,
			"title":  title,
			"status": string(recording.Status),
		})
	}
	return nil
}

func (s *Service) autoRenameEnabled(ctx context.Context, userID uint) (bool, error) {
	if s.users == nil {
		return true, nil
	}
	user, err := s.users.FindByID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return user != nil && user.AutoRenameEnabled, nil
}

func (s *Service) enqueueWidgetsForSummary(ctx context.Context, summary *models.Summary) error {
	if summary == nil || summary.Status != "completed" {
		return nil
	}
	config, err := s.llmConfig.GetActiveByUser(ctx, summary.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !llmReady(config) {
		return nil
	}
	widgets, err := s.summaries.ListEnabledSummaryWidgets(ctx, summary.UserID)
	if err != nil || len(widgets) == 0 {
		return err
	}
	always := make([]models.SummaryWidget, 0, len(widgets))
	conditional := make([]models.SummaryWidget, 0, len(widgets))
	for _, widget := range widgets {
		if widget.AlwaysEnabled {
			always = append(always, widget)
		} else if strings.TrimSpace(valueOrEmpty(widget.WhenToUse)) != "" {
			conditional = append(conditional, widget)
		}
	}
	selected := append([]models.SummaryWidget{}, always...)
	if len(conditional) > 0 {
		client, err := s.clientFor(config)
		if err != nil {
			return err
		}
		model := strings.TrimSpace(*config.SmallModel)
		matched, err := selectRelevantWidgets(ctx, client, model, summary.Content, conditional)
		if err != nil {
			logger.Error("Summary widget selector failed", "summary_id", summary.ID, "error", err)
		}
		selected = append(selected, matched...)
	}
	if len(selected) == 0 {
		return nil
	}
	model := strings.TrimSpace(*config.SmallModel)
	runs, err := s.summaries.EnqueueSummaryWidgetRuns(ctx, summary, selected, model, config.Provider)
	if err != nil {
		return err
	}
	for i := range runs {
		if runs[i].Status == "pending" {
			s.publishWidgetRun(ctx, "summary_widget.pending", &runs[i])
		}
	}
	s.notify()
	return nil
}

func (s *Service) processWidgetRun(ctx context.Context, run *models.SummaryWidgetRun) error {
	if run == nil {
		return nil
	}
	summary, err := s.summaries.GetSummaryByID(ctx, run.SummaryID)
	if err != nil {
		return err
	}
	job, err := s.jobs.FindByID(ctx, run.TranscriptionID)
	if err != nil {
		return err
	}
	config, err := s.llmConfig.GetActiveByUser(ctx, run.UserID)
	if err != nil {
		return err
	}
	if !llmReady(config) {
		return fmt.Errorf("LLM provider is not fully configured")
	}
	client, err := s.clientFor(config)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(*config.SmallModel)
	contextWindow, err := client.GetContextWindow(ctx, model)
	if err != nil || contextWindow <= 0 {
		contextWindow = 4096
	}
	contextText := strings.TrimSpace(summary.Content)
	if run.ContextSource == "transcript" {
		transcriptJSON := ""
		if job.Transcript != nil {
			transcriptJSON = *job.Transcript
		}
		contextText, err = plainTranscriptText(transcriptJSON)
		if err != nil {
			return err
		}
	}
	prompt := widgetPrompt(run.Widget.Prompt, run.RenderMarkdown)
	input, truncated := fitTextToContext(contextText, prompt, contextWindow)
	if truncated {
		run.ContextTruncated = true
		run.ContextWindow = contextWindow
		run.InputCharacters = len(contextText)
		s.publishWidgetRun(ctx, "summary_widget.truncated", run)
	}
	messages := []llm.ChatMessage{{Role: "user", Content: prompt + "\n\nContext:\n" + input}}
	response, err := client.ChatCompletion(ctx, model, messages, 0)
	if err != nil {
		return err
	}
	output := ""
	if response != nil && len(response.Choices) > 0 {
		output = strings.TrimSpace(response.Choices[0].Message.Content)
	}
	if output == "" {
		return fmt.Errorf("summary widget provider returned empty content")
	}
	if err := s.summaries.CompleteSummaryWidgetRun(context.Background(), run.ID, output, truncated, contextWindow, len(contextText), time.Now()); err != nil {
		return err
	}
	run.Output = output
	run.Status = "completed"
	run.ContextTruncated = truncated
	run.ContextWindow = contextWindow
	run.InputCharacters = len(contextText)
	s.publishWidgetRun(context.Background(), "summary_widget.completed", run)
	if err := s.generateDescriptionForSummary(context.Background(), summary); err != nil {
		logger.Error("Automatic description generation failed", "summary_id", summary.ID, "transcription_id", summary.TranscriptionID, "widget_run_id", run.ID, "error", err)
	}
	return nil
}

func (s *Service) generateDescriptionForSummary(ctx context.Context, summary *models.Summary) error {
	if summary == nil || strings.TrimSpace(summary.Content) == "" {
		return nil
	}
	outline, err := s.summaries.GetCompletedOutlineRun(ctx, summary.ID, summary.TranscriptionID, summary.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	config, err := s.llmConfig.GetActiveByUser(ctx, summary.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !titleLLMReady(config) {
		return nil
	}
	job, err := s.jobs.FindByID(ctx, summary.TranscriptionID)
	if err != nil {
		return err
	}
	recording := job
	if job.SourceFileHash != nil && strings.TrimSpace(*job.SourceFileHash) != "" {
		parent, err := s.jobs.FindByID(ctx, strings.TrimSpace(*job.SourceFileHash))
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if parent != nil {
			recording = parent
		}
	}
	if recording.LLMDescriptionSourceSummaryID != nil && *recording.LLMDescriptionSourceSummaryID == summary.ID {
		return nil
	}
	client, err := s.clientFor(config)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(*config.SmallModel)
	prompt := descriptionPrompt + descriptionPromptSummarySection + strings.TrimSpace(summary.Content) + descriptionPromptOutlineSection + strings.TrimSpace(outline.Output)
	response, err := client.ChatCompletion(ctx, model, []llm.ChatMessage{{Role: "user", Content: prompt}}, 0)
	if err != nil {
		return err
	}
	content := ""
	if response != nil && len(response.Choices) > 0 {
		content = response.Choices[0].Message.Content
	}
	description := sanitizeGeneratedDescription(content)
	if description == "" {
		return nil
	}
	generatedAt := time.Now()
	if err := s.jobs.UpdateLLMGeneratedDescription(ctx, summary.TranscriptionID, recording.ID, summary.ID, description, generatedAt); err != nil {
		return err
	}
	if s.events != nil {
		s.events.PublishFileEvent(ctx, "file.updated", map[string]any{
			"id":          "file_" + recording.ID,
			"description": description,
			"status":      string(recording.Status),
		})
	}
	return nil
}

func llmReady(config *models.LLMConfig) bool {
	return config != nil &&
		strings.TrimSpace(config.Provider) != "" &&
		strings.TrimSpace(llmBaseURL(config)) != "" &&
		config.LargeModel != nil && strings.TrimSpace(*config.LargeModel) != "" &&
		config.SmallModel != nil && strings.TrimSpace(*config.SmallModel) != ""
}

func titleLLMReady(config *models.LLMConfig) bool {
	return config != nil &&
		strings.TrimSpace(config.Provider) != "" &&
		strings.TrimSpace(llmBaseURL(config)) != "" &&
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

func sanitizeGeneratedTitle(value string) string {
	title := strings.TrimSpace(value)
	title = strings.TrimPrefix(title, "```")
	title = strings.TrimSuffix(title, "```")
	title = strings.TrimSpace(title)
	title = strings.Trim(title, "\"'“”‘`*_")
	title = strings.TrimSpace(title)
	title = strings.TrimRight(title, " .,:;!?")
	title = strings.Join(strings.Fields(title), " ")
	if title == "" || isGenericGeneratedTitle(title) {
		return ""
	}
	words := strings.Fields(title)
	if len(words) > 7 {
		title = strings.Join(words[:7], " ")
		title = strings.TrimRight(title, " .,:;!?")
	}
	if title == "" || isGenericGeneratedTitle(title) {
		return ""
	}
	return title
}

func isGenericGeneratedTitle(title string) bool {
	normalized := strings.ToLower(strings.TrimSpace(title))
	normalized = strings.TrimRight(normalized, " .,:;!?")
	switch normalized {
	case "audio recording", "recording", "transcript summary", "summary", "discussion", "conversation", "audio transcript", "untitled recording":
		return true
	default:
		return false
	}
}

func sameGeneratedTitle(currentTitle string, generatedTitle string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(currentTitle), " "), strings.Join(strings.Fields(generatedTitle), " "))
}

func sanitizeGeneratedDescription(value string) string {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	rawLines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if isDescriptionListLine(line) {
			return ""
		}
		line = strings.Trim(line, "\"'“”‘`*_")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.Join(strings.Fields(line), " ")
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, truncateDescriptionLine(line))
	}
	if len(lines) != 2 {
		return ""
	}
	if isGenericGeneratedDescription(lines[0]) || isGenericGeneratedDescription(lines[1]) {
		return ""
	}
	return strings.Join(lines, "\n")
}

func isDescriptionListLine(line string) bool {
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return true
	}
	if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' && line[2] == ' ' {
		return true
	}
	return false
}

func truncateDescriptionLine(line string) string {
	const maxRunes = 180
	runes := []rune(line)
	if len(runes) <= maxRunes {
		return strings.TrimSpace(line)
	}
	return strings.TrimRight(strings.TrimSpace(string(runes[:maxRunes])), " .,;:-")
}

func isGenericGeneratedDescription(line string) bool {
	normalized := strings.ToLower(strings.TrimSpace(line))
	normalized = strings.TrimRight(normalized, " .,:;!?")
	switch normalized {
	case "audio recording", "this audio discusses the topic", "this audio is about a topic", "this recording discusses a topic", "a summary of the audio":
		return true
	default:
		return false
	}
}

func fitTranscriptToContext(transcript string, contextWindow int) (string, bool) {
	return fitTextToContext(transcript, summaryPrompt, contextWindow)
}

func fitTextToContext(text string, prompt string, contextWindow int) (string, bool) {
	const charsPerToken = 4
	const completionReserveTokens = 512
	promptTokens := (len(prompt) / charsPerToken) + 1
	budgetTokens := contextWindow - promptTokens - completionReserveTokens
	if budgetTokens < 256 {
		budgetTokens = 256
	}
	maxChars := budgetTokens * charsPerToken
	if len(text) <= maxChars {
		return text, false
	}
	return strings.TrimSpace(text[:maxChars]), true
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

func (s *Service) publishWidgetRun(ctx context.Context, name string, run *models.SummaryWidgetRun) {
	if s.events == nil || run == nil {
		return
	}
	s.events.PublishSummaryStatus(ctx, StatusEvent{
		Name:            name,
		SummaryID:       run.SummaryID,
		TranscriptionID: run.TranscriptionID,
		WidgetRunID:     run.ID,
		WidgetID:        run.WidgetID,
		UserID:          run.UserID,
		Status:          run.Status,
		Truncated:       run.ContextTruncated,
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

func widgetPrompt(savedPrompt string, renderMarkdown bool) string {
	prompt := strings.TrimSpace(savedPrompt)
	if renderMarkdown {
		return prompt + "\n\n" + markdownTypesetInstructions
	}
	return prompt
}

func selectRelevantWidgets(ctx context.Context, client llm.Service, model string, summary string, widgets []models.SummaryWidget) ([]models.SummaryWidget, error) {
	if len(widgets) == 0 {
		return nil, nil
	}
	prompt := buildWidgetSelectionPrompt(summary, widgets)
	response, err := client.ChatCompletion(ctx, model, []llm.ChatMessage{{Role: "user", Content: prompt}}, 0)
	if err != nil {
		return nil, err
	}
	content := ""
	if response != nil && len(response.Choices) > 0 {
		content = strings.TrimSpace(response.Choices[0].Message.Content)
	}
	if content == "" {
		return nil, fmt.Errorf("widget selector returned empty content")
	}
	var parsed struct {
		WidgetNames []string `json:"widget_names"`
	}
	if err := json.Unmarshal([]byte(selectorJSONPayload(content)), &parsed); err != nil {
		return nil, fmt.Errorf("parse widget selector response: %w", err)
	}
	byName := make(map[string]models.SummaryWidget, len(widgets))
	for _, widget := range widgets {
		byName[widget.Name] = widget
	}
	selected := make([]models.SummaryWidget, 0, len(parsed.WidgetNames))
	seen := make(map[string]struct{}, len(parsed.WidgetNames))
	for _, name := range parsed.WidgetNames {
		name = strings.TrimSpace(name)
		if _, ok := seen[name]; ok {
			continue
		}
		widget, ok := byName[name]
		if !ok {
			continue
		}
		selected = append(selected, widget)
		seen[name] = struct{}{}
	}
	return selected, nil
}

func selectorJSONPayload(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
			end := len(lines) - 1
			for end > 0 && strings.TrimSpace(lines[end]) == "" {
				end--
			}
			if strings.HasPrefix(strings.TrimSpace(lines[end]), "```") {
				return strings.TrimSpace(strings.Join(lines[1:end], "\n"))
			}
		}
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(trimmed[start : end+1])
	}
	return trimmed
}

func buildWidgetSelectionPrompt(summary string, widgets []models.SummaryWidget) string {
	var builder strings.Builder
	builder.WriteString("You are a strict JSON classification service. Decide which summary widgets are relevant for an audio transcript summary.\n\n")
	builder.WriteString("Return raw JSON only. The first character of your response must be `{` and the last character must be `}`. Do not use markdown fences. Do not include explanations, comments, prose, or extra keys.\n\n")
	builder.WriteString("Schema:\n")
	builder.WriteString(`{"widget_names":["Exact Widget Name"]}`)
	builder.WriteString("\n\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- `widget_names` must be an array of exact strings copied from the allowed widget names.\n")
	builder.WriteString("- Return `{\"widget_names\":[]}` when none clearly apply.\n")
	builder.WriteString("- Choose only widgets that are clearly relevant.\n")
	builder.WriteString("- Never invent, rename, abbreviate, or translate widget names.\n")
	builder.WriteString("- Never include a widget name that is not in the allowed list.\n\n")
	builder.WriteString("Allowed widget names:\n")
	for _, widget := range widgets {
		builder.WriteString("- ")
		builder.WriteString(widget.Name)
		builder.WriteString("\n")
	}
	builder.WriteString("\nWidget selection criteria:\n")
	builder.WriteString("Available widgets:\n")
	for _, widget := range widgets {
		builder.WriteString("- Name: ")
		builder.WriteString(widget.Name)
		builder.WriteString("\n  When to use: ")
		builder.WriteString(strings.TrimSpace(valueOrEmpty(widget.WhenToUse)))
		builder.WriteString("\n")
	}
	builder.WriteString("\nExample response:\n")
	builder.WriteString(`{"widget_names":[]}`)
	builder.WriteString("\n\n")
	builder.WriteString("\nSummary context:\n")
	builder.WriteString(strings.TrimSpace(summary))
	return builder.String()
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
