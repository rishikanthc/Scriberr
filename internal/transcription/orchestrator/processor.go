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
	"scriberr/internal/transcription/engineprovider"
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

type Processor struct {
	Jobs      repository.JobRepository
	Providers engineprovider.Registry
	Events    EventPublisher
	Logs      JobLogger
	Artifacts TranscriptStore
	OutputDir string
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

	provider, providerID, err := p.resolveProvider(job)
	if err != nil {
		return failedResult(err.Error()), err
	}
	transcriptionModel := defaultString(job.Parameters.Model, engineprovider.DefaultTranscriptionModel)
	decodingMethod := supportedDecodingMethod(job.Parameters.ModelFamily, job.Parameters.DecodingMethod)
	diarizationEnabled := job.Diarization || job.Parameters.Diarize
	diarizationModel := ""
	if diarizationEnabled {
		diarizationModel = defaultString(job.Parameters.DiarizeModel, engineprovider.DefaultDiarizationModel)
	}

	startedAt := time.Now()
	execution := &models.TranscriptionJobExecution{
		TranscriptionJobID: job.ID,
		UserID:             job.UserID,
		Status:             models.StatusProcessing,
		Provider:           providerID,
		ModelName:          transcriptionModel,
		ModelFamily:        defaultString(job.Parameters.ModelFamily, "transcription"),
		StartedAt:          startedAt,
		ActualParameters:   job.Parameters,
		ConfigJSON:         executionConfigJSON(providerID, transcriptionModel, diarizationModel),
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
	if err := provider.Prepare(ctx); err != nil {
		return withExecution(p.errorResult(ctx, err))
	}

	if err := p.publishProgress(ctx, job, "transcribing", 0.20, models.StatusProcessing); err != nil {
		return withExecution(canceledResult(), err)
	}
	transcription, err := provider.Transcribe(ctx, engineprovider.TranscriptionRequest{
		JobID:                   job.ID,
		UserID:                  job.UserID,
		AudioPath:               job.AudioPath,
		ModelID:                 transcriptionModel,
		Language:                languageFromJob(job),
		Task:                    job.Parameters.Task,
		Threads:                 job.Parameters.Threads,
		TailPaddings:            job.Parameters.TailPaddings,
		EnableTokenTimestamps:   job.Parameters.EnableTokenTimestamps,
		EnableSegmentTimestamps: job.Parameters.EnableSegmentTimestamps,
		CanarySourceLanguage:    job.Parameters.CanarySourceLanguage,
		CanaryTargetLanguage:    job.Parameters.CanaryTargetLanguage,
		CanaryUsePunctuation:    job.Parameters.CanaryUsePunctuation,
		DecodingMethod:          decodingMethod,
		Chunking:                job.Parameters.ChunkingStrategy,
		ChunkDurationSec:        float64(job.Parameters.ChunkSize),
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
		transcription.EngineID = providerID
	}

	var diarization *engineprovider.DiarizationResult
	if diarizationEnabled {
		if err := p.publishProgress(ctx, job, "diarizing", 0.70, models.StatusProcessing); err != nil {
			return withExecution(canceledResult(), err)
		}
		diarization, err = provider.Diarize(ctx, engineprovider.DiarizationRequest{
			JobID:          job.ID,
			UserID:         job.UserID,
			AudioPath:      job.AudioPath,
			ModelID:        diarizationModel,
			NumSpeakers:    job.Parameters.NumSpeakers,
			Threshold:      job.Parameters.DiarizationThreshold,
			MinDurationOn:  job.Parameters.MinDurationOn,
			MinDurationOff: job.Parameters.MinDurationOff,
		})
		if err != nil {
			return withExecution(p.errorResult(ctx, err))
		}
		if diarization != nil {
			if diarization.ModelID == "" {
				diarization.ModelID = diarizationModel
			}
			if diarization.EngineID == "" {
				diarization.EngineID = providerID
			}
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

func (p *Processor) resolveProvider(job *models.TranscriptionJob) (engineprovider.Provider, string, error) {
	req := engineprovider.SelectionRequest{}
	if job.EngineID != nil && strings.TrimSpace(*job.EngineID) != "" {
		req.ProviderID = strings.TrimSpace(*job.EngineID)
	}
	provider, _, err := p.Providers.Select(context.Background(), req)
	if err != nil {
		return nil, "", err
	}
	if provider == nil {
		return nil, "", fmt.Errorf("selected engine provider is not available")
	}
	return provider, provider.ID(), nil
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

func executionConfigJSON(providerID, transcriptionModel, diarizationModel string) string {
	payload := map[string]string{
		"provider":            providerID,
		"transcription_model": transcriptionModel,
	}
	if diarizationModel != "" {
		payload["diarization_model"] = diarizationModel
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func languageFromJob(job *models.TranscriptionJob) string {
	if job.Language != nil {
		return *job.Language
	}
	if job.Parameters.Language != nil {
		return *job.Parameters.Language
	}
	return ""
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func supportedDecodingMethod(modelFamily, decodingMethod string) string {
	method := strings.TrimSpace(decodingMethod)
	if strings.TrimSpace(modelFamily) == "whisper" {
		return "greedy_search"
	}
	return method
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
