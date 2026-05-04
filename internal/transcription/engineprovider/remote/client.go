package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
)

const (
	defaultTimeout         = 30 * time.Second
	defaultPollInterval    = 2 * time.Second
	defaultMaxResponseSize = 4 << 20
)

type Config struct {
	ID               string
	BaseURL          string
	HTTPClient       *http.Client
	Timeout          time.Duration
	PollInterval     time.Duration
	MaxResponseBytes int64
}

type Client struct {
	id               string
	baseURL          *url.URL
	httpClient       *http.Client
	timeout          time.Duration
	pollInterval     time.Duration
	maxResponseBytes int64
}

func NewClient(cfg Config) (*Client, error) {
	id := strings.TrimSpace(cfg.ID)
	if id == "" {
		return nil, fmt.Errorf("remote provider id is required")
	}
	rawURL := strings.TrimSpace(cfg.BaseURL)
	if rawURL == "" {
		return nil, fmt.Errorf("remote provider base url is required")
	}
	baseURL, err := url.Parse(rawURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("remote provider base url is invalid")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	maxResponseBytes := cfg.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultMaxResponseSize
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{
		id:               id,
		baseURL:          baseURL,
		httpClient:       httpClient,
		timeout:          timeout,
		pollInterval:     pollInterval,
		maxResponseBytes: maxResponseBytes,
	}, nil
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Inspect(ctx context.Context) (*asrcontract.ProviderInfo, error) {
	var out asrcontract.ProviderInfo
	if err := c.doJSON(ctx, http.MethodGet, "/v1/provider", nil, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Provider.ID) == "" {
		out.Provider.ID = c.id
	}
	return &out, nil
}

func (c *Client) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	var out []asrcontract.ModelCard
	if err := c.doJSON(ctx, http.MethodGet, "/v1/models", nil, &out); err != nil {
		return nil, err
	}
	for i := range out {
		if strings.TrimSpace(out[i].Provider) == "" {
			out[i].Provider = c.id
		}
	}
	return out, nil
}

func (c *Client) Status(ctx context.Context) (*asrcontract.ProviderStatus, error) {
	var out asrcontract.ProviderStatus
	if err := c.doJSON(ctx, http.MethodGet, "/v1/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) LoadModel(ctx context.Context, req asrcontract.LoadModelRequest) error {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "model is required", false)
	}
	return c.doJSON(ctx, http.MethodPost, "/v1/models/"+url.PathEscape(model)+":load", req, nil)
}

func (c *Client) UnloadModel(ctx context.Context, req asrcontract.UnloadModelRequest) error {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "model is required", false)
	}
	return c.doJSON(ctx, http.MethodPost, "/v1/models/"+url.PathEscape(model)+":unload", req, nil)
}

func (c *Client) LoadedModels(ctx context.Context) ([]asrcontract.LoadedModel, error) {
	var out []asrcontract.LoadedModel
	if err := c.doJSON(ctx, http.MethodGet, "/v1/models/loaded", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Capabilities(ctx context.Context) ([]engineprovider.ModelCapability, error) {
	models, err := c.Models(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]engineprovider.ModelCapability, 0, len(models))
	for _, model := range models {
		out = append(out, engineprovider.ModelCapability{
			ID:           model.ID,
			Name:         model.DisplayName,
			Provider:     coalesceString(model.Provider, c.id),
			Installed:    model.Installed,
			Default:      model.Default,
			Capabilities: capabilityStrings(model.Capabilities),
		})
	}
	return out, nil
}

func (c *Client) Prepare(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodGet, "/v1/health", nil, nil)
}

func (c *Client) Transcribe(ctx context.Context, req engineprovider.TranscriptionRequest) (*engineprovider.TranscriptionResult, error) {
	remoteReq := asrcontract.TranscriptionRequest{
		RequestID: req.JobID,
		Audio:     mountedWAV(req.AudioPath),
		Model:     coalesceString(req.ModelID, engineprovider.DefaultTranscriptionModel),
		Task:      asrcontract.Task(coalesceString(req.Task, string(asrcontract.TaskTranscribe))),
		Language:  req.Language,
		Features: asrcontract.Capabilities{
			Transcription:     true,
			WordTimestamps:    boolValue(req.EnableTokenTimestamps, true),
			SegmentTimestamps: boolValue(req.EnableSegmentTimestamps, true),
			TokenTimestamps:   boolValue(req.EnableTokenTimestamps, false),
		},
		Options: map[string]any{
			"threads":                   req.Threads,
			"tail_paddings":             req.TailPaddings,
			"decoding_method":           req.DecodingMethod,
			"chunking":                  req.Chunking,
			"chunk_duration_sec":        req.ChunkDurationSec,
			"enable_token_timestamps":   req.EnableTokenTimestamps,
			"enable_segment_timestamps": req.EnableSegmentTimestamps,
		},
	}
	var result asrcontract.TranscriptionResult
	if err := c.runJob(ctx, jobCreateRequest{Operation: asrcontract.OperationTranscription, Transcription: &remoteReq}, req.Progress, &result); err != nil {
		return nil, err
	}
	return transcriptionResult(result, c.id), nil
}

func (c *Client) Diarize(ctx context.Context, req engineprovider.DiarizationRequest) (*engineprovider.DiarizationResult, error) {
	remoteReq := asrcontract.DiarizationRequest{
		RequestID: req.JobID,
		Audio:     mountedWAV(req.AudioPath),
		Model:     coalesceString(req.ModelID, engineprovider.DefaultDiarizationModel),
		Options: map[string]any{
			"num_speakers":     req.NumSpeakers,
			"threshold":        req.Threshold,
			"min_duration_on":  req.MinDurationOn,
			"min_duration_off": req.MinDurationOff,
		},
	}
	var result asrcontract.DiarizationResult
	if err := c.runJob(ctx, jobCreateRequest{Operation: asrcontract.OperationDiarization, Diarization: &remoteReq}, req.Progress, &result); err != nil {
		return nil, err
	}
	return diarizationResult(result, c.id), nil
}

func (c *Client) IdentifySpeakers(ctx context.Context, req asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error) {
	var result asrcontract.SpeakerIDResult
	if err := c.runJob(ctx, jobCreateRequest{Operation: asrcontract.OperationSpeakerIdentification, SpeakerIdentification: &req}, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Close() error {
	return nil
}

type jobCreateRequest struct {
	Operation             asrcontract.Operation             `json:"operation"`
	Transcription         *asrcontract.TranscriptionRequest `json:"transcription,omitempty"`
	Diarization           *asrcontract.DiarizationRequest   `json:"diarization,omitempty"`
	SpeakerIdentification *asrcontract.SpeakerIDRequest     `json:"speaker_identification,omitempty"`
}

type jobCreateResponse struct {
	JobID string `json:"job_id"`
}

type jobStatusResponse struct {
	JobID                 string                           `json:"job_id"`
	Status                string                           `json:"status"`
	Progress              *asrcontract.ProviderProgress    `json:"progress,omitempty"`
	Error                 *asrcontract.ProviderError       `json:"error,omitempty"`
	Transcription         *asrcontract.TranscriptionResult `json:"transcription,omitempty"`
	Diarization           *asrcontract.DiarizationResult   `json:"diarization,omitempty"`
	SpeakerIdentification *asrcontract.SpeakerIDResult     `json:"speaker_identification,omitempty"`
}

type eventsResponse struct {
	Events []asrcontract.ProviderProgress `json:"events"`
}

func (c *Client) runJob(ctx context.Context, req jobCreateRequest, progress engineprovider.ProgressSink, result any) error {
	var created jobCreateResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/jobs", req, &created); err != nil {
		return err
	}
	jobID := strings.TrimSpace(created.JobID)
	if jobID == "" {
		return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "remote provider returned no job id", false)
	}
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	seenEvents := 0
	for {
		status, err := c.pollJob(ctx, jobID)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.cancelRemoteJob(jobID)
			}
			return err
		}
		if status.Progress != nil && progress != nil {
			progress.Report(ctx, *status.Progress)
		}
		events, eventErr := c.jobEvents(ctx, jobID)
		if eventErr != nil {
			if errors.Is(eventErr, context.Canceled) || errors.Is(eventErr, context.DeadlineExceeded) {
				c.cancelRemoteJob(jobID)
			}
			return eventErr
		}
		for _, event := range events[seenEvents:] {
			if progress != nil {
				progress.Report(ctx, event)
			}
		}
		if len(events) > seenEvents {
			seenEvents = len(events)
		}
		if terminalJobStatus(status.Status) {
			if status.Error != nil {
				return sanitizeProviderError(status.Error)
			}
			return copyJobResult(status, result)
		}
		select {
		case <-ctx.Done():
			c.cancelRemoteJob(jobID)
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *Client) pollJob(ctx context.Context, jobID string) (*jobStatusResponse, error) {
	var status jobStatusResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/jobs/"+url.PathEscape(jobID), nil, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *Client) jobEvents(ctx context.Context, jobID string) ([]asrcontract.ProviderProgress, error) {
	var out eventsResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/jobs/"+url.PathEscape(jobID)+"/events", nil, &out); err != nil {
		var providerErr *asrcontract.ProviderError
		if errors.As(err, &providerErr) && providerErr.Code == asrcontract.CodeUnsupportedOperation {
			return nil, nil
		}
		return nil, err
	}
	return out.Events, nil
}

func (c *Client) cancelRemoteJob(jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	_ = c.doJSON(ctx, http.MethodDelete, "/v1/jobs/"+url.PathEscape(jobID), nil, nil)
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, in any, out any) error {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	var body io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "remote request is invalid", false)
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(endpoint), body)
	if err != nil {
		return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "remote request is invalid", false)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return asrcontract.NewProviderError(asrcontract.CodeProviderUnhealthy, "remote provider request failed", true)
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, c.maxResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return asrcontract.NewProviderError(asrcontract.CodeProviderUnhealthy, "remote provider response could not be read", true)
	}
	if int64(len(data)) > c.maxResponseBytes {
		return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "remote provider response is too large", false)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if providerErr := decodeProviderError(data, resp.StatusCode); providerErr != nil {
			return providerErr
		}
		return asrcontract.NewProviderError(errorCodeForStatus(resp.StatusCode), "remote provider returned an error", resp.StatusCode >= 500)
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "remote provider response is invalid", false)
	}
	return nil
}

func (c *Client) endpoint(endpoint string) string {
	next := *c.baseURL
	basePath := strings.TrimRight(next.Path, "/")
	next.Path = path.Join(basePath, endpoint)
	if strings.HasSuffix(endpoint, ":load") || strings.HasSuffix(endpoint, ":unload") {
		next.Path = strings.Replace(next.Path, "%3A", ":", 1)
	}
	return next.String()
}

func decodeProviderError(data []byte, status int) *asrcontract.ProviderError {
	var wrapper struct {
		Error *asrcontract.ProviderError `json:"error"`
	}
	if json.Unmarshal(data, &wrapper) == nil && wrapper.Error != nil {
		return sanitizeProviderError(wrapper.Error)
	}
	var direct asrcontract.ProviderError
	if json.Unmarshal(data, &direct) == nil && direct.Code != "" {
		return sanitizeProviderError(&direct)
	}
	return asrcontract.NewProviderError(errorCodeForStatus(status), "remote provider returned an error", status >= 500)
}

func sanitizeProviderError(err *asrcontract.ProviderError) *asrcontract.ProviderError {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(err.Message)
	if message == "" {
		message = string(err.Code)
	}
	out := asrcontract.NewProviderError(err.Code, message, err.Retryable)
	if len(err.Details) > 0 {
		out.Details = boundedDetails(err.Details)
	}
	return out
}

func boundedDetails(details map[string]any) map[string]any {
	out := make(map[string]any, len(details))
	for key, value := range details {
		if len(out) >= 8 {
			break
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.Contains(strings.ToLower(key), "path") || strings.Contains(strings.ToLower(key), "url") || strings.Contains(strings.ToLower(key), "token") {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func errorCodeForStatus(status int) asrcontract.ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return asrcontract.CodeInvalidRequest
	case http.StatusNotFound:
		return asrcontract.CodeUnsupportedModel
	case http.StatusConflict, http.StatusTooManyRequests:
		return asrcontract.CodeProviderBusy
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return asrcontract.CodeTimeout
	case http.StatusServiceUnavailable:
		return asrcontract.CodeProviderUnhealthy
	default:
		if status >= 500 {
			return asrcontract.CodeInferenceFailed
		}
		return asrcontract.CodeInvalidRequest
	}
}

func terminalJobStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

func copyJobResult(status *jobStatusResponse, out any) error {
	switch target := out.(type) {
	case *asrcontract.TranscriptionResult:
		if status.Transcription == nil {
			return asrcontract.NewProviderError(asrcontract.CodeInferenceFailed, "remote provider returned no transcription result", false)
		}
		*target = *status.Transcription
	case *asrcontract.DiarizationResult:
		if status.Diarization == nil {
			return asrcontract.NewProviderError(asrcontract.CodeInferenceFailed, "remote provider returned no diarization result", false)
		}
		*target = *status.Diarization
	case *asrcontract.SpeakerIDResult:
		if status.SpeakerIdentification == nil {
			return asrcontract.NewProviderError(asrcontract.CodeInferenceFailed, "remote provider returned no speaker identification result", false)
		}
		*target = *status.SpeakerIdentification
	default:
		return asrcontract.NewProviderError(asrcontract.CodeInvalidRequest, "remote result target is invalid", false)
	}
	return nil
}

func mountedWAV(audioPath string) asrcontract.AudioInput {
	return asrcontract.AudioInput{
		Path:       audioPath,
		SampleRate: 16000,
		Channels:   1,
		Format:     "wav",
	}
}

func transcriptionResult(in asrcontract.TranscriptionResult, providerID string) *engineprovider.TranscriptionResult {
	words := make([]engineprovider.TranscriptWord, 0, len(in.Words))
	for _, word := range in.Words {
		words = append(words, engineprovider.TranscriptWord{
			Start:   word.Start,
			End:     word.End,
			Word:    word.Word,
			Speaker: word.Speaker,
		})
	}
	segments := make([]engineprovider.TranscriptSegment, 0, len(in.Segments))
	for _, segment := range in.Segments {
		segments = append(segments, engineprovider.TranscriptSegment{
			ID:      segment.ID,
			Start:   segment.Start,
			End:     segment.End,
			Speaker: segment.Speaker,
			Text:    segment.Text,
		})
	}
	return &engineprovider.TranscriptionResult{
		Text:     in.Text,
		Language: in.Language,
		Words:    words,
		Segments: segments,
		ModelID:  in.Model,
		EngineID: providerID,
	}
}

func diarizationResult(in asrcontract.DiarizationResult, providerID string) *engineprovider.DiarizationResult {
	segments := make([]engineprovider.DiarizationSegment, 0, len(in.Segments))
	for _, segment := range in.Segments {
		segments = append(segments, engineprovider.DiarizationSegment{
			Start:   segment.Start,
			End:     segment.End,
			Speaker: segment.Speaker,
		})
	}
	return &engineprovider.DiarizationResult{
		Segments: segments,
		ModelID:  in.Model,
		EngineID: providerID,
	}
}

func capabilityStrings(capabilities asrcontract.Capabilities) []string {
	out := []string{}
	if capabilities.Transcription {
		out = append(out, string(asrcontract.CapabilityTranscription))
	}
	if capabilities.Diarization {
		out = append(out, string(asrcontract.CapabilityDiarization))
	}
	if capabilities.SpeakerIdentification {
		out = append(out, string(asrcontract.CapabilitySpeakerIdentification))
	}
	if capabilities.WordTimestamps {
		out = append(out, string(asrcontract.CapabilityWordTimestamps))
	}
	if capabilities.SegmentTimestamps {
		out = append(out, string(asrcontract.CapabilitySegmentTimestamps))
	}
	return out
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ engineprovider.Provider = (*Client)(nil)
