package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/preprocess"
	"scriberr/internal/transcription/worker"
	"scriberr/pkg/logger"
)

type EventPublisher interface {
	Publish(ctx context.Context, event ProgressEvent)
}

type JobLogger interface {
	Info(jobID string, message string, fields ...any)
	Error(jobID string, message string, fields ...any)
}

type ProgressEvent struct {
	Name     string           `json:"name"`
	JobID    string           `json:"job_id"`
	FileID   string           `json:"file_id"`
	UserID   uint             `json:"user_id"`
	Stage    string           `json:"stage"`
	Progress float64          `json:"progress"`
	Status   models.JobStatus `json:"status"`
}

type providerProgressSink struct {
	processor *Processor
	job       *models.TranscriptionJob
}

func (s providerProgressSink) Report(ctx context.Context, event asrcontract.ProviderProgress) {
	if s.processor == nil || s.job == nil {
		return
	}
	stage := string(event.Stage)
	if strings.TrimSpace(stage) == "" {
		stage = "processing"
	}
	progress := providerProgressValue(event)
	_ = s.processor.publishProgress(ctx, s.job, stage, progress, models.StatusProcessing)
}

type processorBoundaryReporter struct {
	processor *Processor
	job       *models.TranscriptionJob
}

func (r processorBoundaryReporter) ReportPlanBoundary(ctx context.Context, boundary PlanBoundary) error {
	if r.processor == nil || r.job == nil {
		return ctx.Err()
	}
	stage := "processing"
	switch boundary.Operation {
	case models.ASRStepTranscription:
		stage = "transcribing"
	case models.ASRStepDiarization:
		stage = "diarizing"
	case models.ASRStepSpeakerIdentification:
		stage = "identifying_speakers"
	}
	return r.processor.publishProgress(ctx, r.job, stage, boundary.Progress, models.StatusProcessing)
}

type Processor struct {
	Jobs      repository.JobRepository
	Providers engineprovider.Registry
	Events    EventPublisher
	Logs      JobLogger
	Artifacts TranscriptStore
	Audio     preprocess.Preprocessor
	OutputDir string
}

type resolvedASRStep struct {
	Kind        string
	ProviderID  string
	Provider    engineprovider.Provider
	Model       string
	ModelFamily string
	Options     map[string]any
}

func (p *Processor) Process(ctx context.Context, job *models.TranscriptionJob) (worker.ProcessResult, error) {
	if err := ctx.Err(); err != nil {
		return canceledResult(), err
	}
	if job == nil {
		return failedResult("transcription job is required"), fmt.Errorf("transcription job is required")
	}
	if p.Jobs == nil {
		return failedResult("job repository is required"), fmt.Errorf("job repository is required")
	}
	if p.Providers == nil {
		return failedResult("engine provider registry is required"), fmt.Errorf("engine provider registry is required")
	}

	pipeline, err := p.resolvePipeline(ctx, job)
	if err != nil {
		return failedResult(sanitizeErrorMessage(err)), err
	}
	transcriptionStep, ok := firstStepByKind(pipeline, models.ASRStepTranscription)
	if !ok {
		err := fmt.Errorf("ASR pipeline requires a transcription step")
		return failedResult(sanitizeErrorMessage(err)), err
	}
	diarizationStep, diarizationEnabled := firstStepByKind(pipeline, models.ASRStepDiarization)
	transcriptionModel := transcriptionStep.Model
	providerID := transcriptionStep.ProviderID
	plan, err := p.buildPlan(ctx, job, pipeline)
	if err != nil {
		return failedResult(sanitizeErrorMessage(err)), err
	}

	startedAt := time.Now()
	execution := &models.TranscriptionJobExecution{
		TranscriptionJobID: job.ID,
		UserID:             job.UserID,
		Status:             models.StatusProcessing,
		Provider:           providerID,
		ModelName:          transcriptionModel,
		ModelFamily:        defaultString(transcriptionStep.ModelFamily, "transcription"),
		StartedAt:          startedAt,
		ActualParameters:   job.Parameters,
		ConfigJSON:         executionConfigJSON(pipeline, plan),
	}
	if err := p.Jobs.CreateExecution(ctx, execution); err != nil {
		return failedResult(sanitizeErrorMessage(err)), err
	}
	withExecution := func(result worker.ProcessResult, err error) (worker.ProcessResult, error) {
		result.ExecutionID = execution.ID
		return result, err
	}

	if err := p.publishProgress(ctx, job, "preparing", 0.05, models.StatusProcessing); err != nil {
		return withExecution(canceledResult(), err)
	}
	if err := validateAudioPath(job.AudioPath); err != nil {
		message := sanitizeErrorMessage(err)
		return withExecution(failedResult(message), err)
	}
	audio, err := p.audioPreprocessor().Prepare(ctx, preprocess.Request{
		JobID:          job.ID,
		SourcePath:     job.AudioPath,
		SourceFileHash: sourceHashForJob(job),
	})
	if err != nil {
		message := sanitizeErrorMessage(err)
		return withExecution(failedResult(message), err)
	}
	boundaryReporter := processorBoundaryReporter{processor: p, job: job}
	if err := plan.ReportBoundary(ctx, models.ASRStepTranscription, boundaryReporter); err != nil {
		return withExecution(canceledResult(), err)
	}
	progressSink := providerProgressSink{processor: p, job: job}
	if err := prepareStepProvider(ctx, transcriptionStep); err != nil {
		return withExecution(p.errorResult(ctx, err))
	}
	transcription, err := transcriptionStep.Provider.Transcribe(ctx, engineprovider.TranscriptionRequest{
		JobID:      job.ID,
		UserID:     job.UserID,
		AudioPath:  audio.ProviderPath,
		Progress:   progressSink,
		ModelID:    transcriptionModel,
		Parameters: providerParametersForStep(transcriptionStep),
	})
	if err != nil {
		return withExecution(p.errorResult(ctx, err))
	}
	if transcription == nil {
		return withExecution(failedResult("transcription provider returned no result"), fmt.Errorf("transcription provider returned no result"))
	}
	if transcription.ModelID == "" {
		transcription.ModelID = transcriptionModel
	}
	if transcription.EngineID == "" {
		transcription.EngineID = transcriptionStep.ProviderID
	}

	var diarization *engineprovider.DiarizationResult
	if diarizationEnabled {
		if err := plan.ReportBoundary(ctx, models.ASRStepDiarization, boundaryReporter); err != nil {
			return withExecution(canceledResult(), err)
		}
		if err := prepareStepProvider(ctx, diarizationStep); err != nil {
			return withExecution(p.errorResult(ctx, err))
		}
		diarization, err = diarizationStep.Provider.Diarize(ctx, engineprovider.DiarizationRequest{
			JobID:      job.ID,
			UserID:     job.UserID,
			AudioPath:  audio.ProviderPath,
			Progress:   progressSink,
			ModelID:    diarizationStep.Model,
			Parameters: providerParametersForStep(diarizationStep),
		})
		if err != nil {
			return withExecution(p.errorResult(ctx, err))
		}
		if diarization != nil {
			if diarization.ModelID == "" {
				diarization.ModelID = diarizationStep.Model
			}
			if diarization.EngineID == "" {
				diarization.EngineID = diarizationStep.ProviderID
			}
		}
	}
	for _, speakerStep := range stepsByKind(pipeline, models.ASRStepSpeakerIdentification) {
		if err := plan.ReportBoundary(ctx, models.ASRStepSpeakerIdentification, boundaryReporter); err != nil {
			return withExecution(canceledResult(), err)
		}
		if err := prepareStepProvider(ctx, speakerStep); err != nil {
			return withExecution(p.errorResult(ctx, err))
		}
		_, err := speakerStep.Provider.IdentifySpeakers(ctx, asrcontract.SpeakerIDRequest{
			RequestID: job.ID,
			Audio: asrcontract.AudioInput{
				Path:       audio.ProviderPath,
				SampleRate: 16000,
				Channels:   1,
				Format:     "wav",
			},
			Model: speakerStep.Model,
		})
		if err != nil {
			return withExecution(p.errorResult(ctx, err))
		}
	}

	if err := p.publishProgress(ctx, job, "merging", 0.85, models.StatusProcessing); err != nil {
		return withExecution(canceledResult(), err)
	}
	canonical, err := BuildCanonicalTranscript(transcription, diarization)
	if err != nil {
		message := sanitizeErrorMessage(err)
		return withExecution(failedResult(message), err)
	}
	transcriptJSON, err := json.Marshal(canonical)
	if err != nil {
		message := sanitizeErrorMessage(err)
		return withExecution(failedResult(message), err)
	}

	if err := p.publishProgress(ctx, job, "saving", 0.95, models.StatusProcessing); err != nil {
		return withExecution(canceledResult(), err)
	}
	outputPath, err := p.transcriptStore().SaveTranscriptJSON(ctx, job.ID, transcriptJSON)
	if err != nil {
		message := sanitizeErrorMessage(err)
		return withExecution(failedResult(message), err)
	}
	p.publishFinal(ctx, job, "completed", 1.0, models.StatusCompleted)
	logger.Info("Transcription job processed", "job_id", job.ID, "provider", providerID, "model", transcriptionModel)
	return worker.ProcessResult{
		ExecutionID:    execution.ID,
		Status:         models.StatusCompleted,
		TranscriptJSON: string(transcriptJSON),
		OutputJSONPath: &outputPath,
		CompletedAt:    time.Now(),
	}, nil
}

func (p *Processor) buildPlan(ctx context.Context, job *models.TranscriptionJob, steps []resolvedASRStep) (ExecutionPlan, error) {
	models := []asrcontract.ModelCard{}
	if p != nil && p.Providers != nil {
		cards, err := p.Providers.Models(ctx)
		if err == nil {
			models = cards
		}
	}
	return buildExecutionPlan(ctx, planRequest{
		Params: job.Parameters,
		Steps:  steps,
		Models: models,
		Limits: defaultPlanLimits(),
	})
}

func (p *Processor) resolvePipeline(ctx context.Context, job *models.TranscriptionJob) ([]resolvedASRStep, error) {
	steps := pipelineStepsForJob(job)
	out := make([]resolvedASRStep, 0, len(steps))
	for _, step := range steps {
		kind := strings.TrimSpace(step.Kind)
		model := strings.TrimSpace(step.Model)
		requires := []string{}
		switch kind {
		case models.ASRStepTranscription:
			model = defaultString(model, engineprovider.DefaultTranscriptionModel)
			requires = []string{string(asrcontract.CapabilityTranscription)}
		case models.ASRStepDiarization:
			model = defaultString(model, engineprovider.DefaultDiarizationModel)
			requires = []string{string(asrcontract.CapabilityDiarization)}
		case models.ASRStepSpeakerIdentification:
			requires = []string{string(asrcontract.CapabilitySpeakerIdentification)}
		default:
			return nil, fmt.Errorf("unsupported ASR pipeline step %q", kind)
		}
		provider, capability, err := p.Providers.Select(ctx, engineprovider.SelectionRequest{
			ProviderID: strings.TrimSpace(step.Provider),
			ModelID:    model,
			Requires:   requires,
		})
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, fmt.Errorf("selected engine provider is not available")
		}
		if model == "" && capability != nil {
			model = capability.ID
		}
		out = append(out, resolvedASRStep{
			Kind:        kind,
			ProviderID:  provider.ID(),
			Provider:    provider,
			Model:       model,
			ModelFamily: strings.TrimSpace(step.ModelFamily),
			Options:     copyStepOptions(step.Options),
		})
	}
	return out, nil
}

func providerParametersForStep(step resolvedASRStep) map[string]any {
	return copyStepOptions(step.Options)
}

func copyStepOptions(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func pipelineStepsForJob(job *models.TranscriptionJob) []models.ASRStep {
	if job == nil {
		return nil
	}
	if len(job.Parameters.Pipeline) > 0 {
		return job.Parameters.Pipeline
	}
	return []models.ASRStep{{
		Kind:        models.ASRStepTranscription,
		Provider:    providerFromJob(job),
		Model:       engineprovider.DefaultTranscriptionModel,
		ModelFamily: "whisper",
	}}
}

func providerFromJob(job *models.TranscriptionJob) string {
	if job == nil {
		return ""
	}
	if job.EngineID != nil && strings.TrimSpace(*job.EngineID) != "" {
		return strings.TrimSpace(*job.EngineID)
	}
	return strings.TrimSpace(job.Parameters.Provider)
}

func firstStepByKind(steps []resolvedASRStep, kind string) (resolvedASRStep, bool) {
	for _, step := range steps {
		if step.Kind == kind {
			return step, true
		}
	}
	return resolvedASRStep{}, false
}

func stepsByKind(steps []resolvedASRStep, kind string) []resolvedASRStep {
	out := []resolvedASRStep{}
	for _, step := range steps {
		if step.Kind == kind {
			out = append(out, step)
		}
	}
	return out
}

func prepareStepProvider(ctx context.Context, step resolvedASRStep) error {
	if step.Provider == nil {
		return fmt.Errorf("selected engine provider is not available")
	}
	return step.Provider.Prepare(ctx)
}

func (p *Processor) publishProgress(ctx context.Context, job *models.TranscriptionJob, stage string, progress float64, status models.JobStatus) error {
	if err := ctx.Err(); err != nil {
		p.publishFinal(context.Background(), job, "stopped", progress, models.StatusStopped)
		return err
	}
	if err := p.Jobs.UpdateProgress(ctx, job.ID, progress, stage); err != nil {
		return err
	}
	p.publishFinal(ctx, job, stage, progress, status)
	return nil
}

func (p *Processor) publishFinal(ctx context.Context, job *models.TranscriptionJob, stage string, progress float64, status models.JobStatus) {
	if p.Events == nil {
		return
	}
	name := "transcription.progress"
	switch stage {
	case "completed":
		name = "transcription.completed"
	case "failed":
		name = "transcription.failed"
	case "stopped", "canceled":
		name = "transcription.stopped"
	case "queued":
		name = "transcription.queued"
	}
	p.Events.Publish(ctx, ProgressEvent{
		Name:     name,
		JobID:    job.ID,
		FileID:   fileIDForJob(job),
		UserID:   job.UserID,
		Stage:    stage,
		Progress: progress,
		Status:   status,
	})
}

func fileIDForJob(job *models.TranscriptionJob) string {
	if job == nil {
		return ""
	}
	if job.SourceFileHash != nil && *job.SourceFileHash != "" {
		return "file_" + *job.SourceFileHash
	}
	return "file_" + job.ID
}

func sourceHashForJob(job *models.TranscriptionJob) string {
	if job == nil || job.SourceFileHash == nil {
		return ""
	}
	return *job.SourceFileHash
}

func (p *Processor) errorResult(ctx context.Context, err error) (worker.ProcessResult, error) {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return canceledResult(), context.Canceled
	}
	message := sanitizeErrorMessage(err)
	return failedResult(message), err
}

func (p *Processor) transcriptStore() TranscriptStore {
	if p.Artifacts != nil {
		return p.Artifacts
	}
	return NewLocalTranscriptStore(p.OutputDir)
}

func (p *Processor) audioPreprocessor() preprocess.Preprocessor {
	if p.Audio != nil {
		return p.Audio
	}
	return preprocess.PassthroughPreprocessor{}
}

func executionConfigJSON(steps []resolvedASRStep, plan ExecutionPlan) string {
	type executionStep struct {
		Operation string `json:"operation"`
		Provider  string `json:"provider"`
		Model     string `json:"model"`
	}
	executionSteps := make([]executionStep, 0, len(steps))
	for _, step := range steps {
		executionSteps = append(executionSteps, executionStep{
			Operation: step.Kind,
			Provider:  step.ProviderID,
			Model:     step.Model,
		})
	}
	transcriptionStep, _ := firstStepByKind(steps, models.ASRStepTranscription)
	diarizationStep, hasDiarization := firstStepByKind(steps, models.ASRStepDiarization)
	payload := struct {
		Provider           string          `json:"provider"`
		TranscriptionModel string          `json:"transcription_model"`
		DiarizationModel   string          `json:"diarization_model,omitempty"`
		Steps              []executionStep `json:"steps"`
		Plan               any             `json:"plan,omitempty"`
	}{
		Provider:           transcriptionStep.ProviderID,
		TranscriptionModel: transcriptionStep.Model,
		Steps:              executionSteps,
		Plan:               plan.Summary(),
	}
	if hasDiarization {
		payload.DiarizationModel = diarizationStep.Model
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func planStepForOperation(plan ExecutionPlan, operation string) PlannedStep {
	for _, step := range plan.Steps {
		if step.Operation == operation {
			return step
		}
	}
	return PlannedStep{}
}

func providerProgressValue(event asrcontract.ProviderProgress) float64 {
	if event.Progress != nil {
		switch {
		case *event.Progress < 0:
			return 0
		case *event.Progress > 1:
			return 1
		default:
			return *event.Progress
		}
	}
	switch event.Stage {
	case asrcontract.StagePreprocessing:
		return 0.10
	case asrcontract.StageLoadingModel:
		return 0.15
	case asrcontract.StageTranscribing:
		return 0.35
	case asrcontract.StageDiarizing:
		return 0.70
	case asrcontract.StagePostprocessing:
		return 0.82
	case asrcontract.StageCompleted:
		return 0.90
	default:
		return 0.20
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func validateAudioPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("source audio path is required")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return nil
}

func failedResult(message string) worker.ProcessResult {
	return worker.ProcessResult{
		Status:       models.StatusFailed,
		ErrorMessage: message,
		FailedAt:     time.Now(),
	}
}

func canceledResult() worker.ProcessResult {
	return worker.ProcessResult{
		Status:   models.StatusStopped,
		FailedAt: time.Now(),
	}
}

var absolutePathPattern = regexp.MustCompile(`(?:[A-Za-z]:\\|/)[^\s:;,'")]+`)

func sanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := absolutePathPattern.ReplaceAllString(err.Error(), "[redacted-path]")
	parts := strings.Fields(msg)
	for i, part := range parts {
		lower := strings.ToLower(part)
		if strings.Contains(lower, "token") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") {
			if strings.Contains(part, "=") {
				key := strings.SplitN(part, "=", 2)[0]
				parts[i] = key + "=[redacted]"
			}
		}
	}
	return strings.Join(parts, " ")
}
