package engineprovider

import (
	"context"

	"scriberr/internal/transcription/asrcontract"
)

const (
	DefaultProviderID = "local"
)

type Provider interface {
	ID() string
	Inspect(ctx context.Context) (*asrcontract.ProviderInfo, error)
	Models(ctx context.Context) ([]asrcontract.ModelCard, error)
	Status(ctx context.Context) (*asrcontract.ProviderStatus, error)
	LoadModel(ctx context.Context, req asrcontract.LoadModelRequest) error
	UnloadModel(ctx context.Context, req asrcontract.UnloadModelRequest) error
	LoadedModels(ctx context.Context) ([]asrcontract.LoadedModel, error)
	ExecuteTask(ctx context.Context, req TaskRequest) (*TaskResult, error)
	Close() error
}

type ProgressSink interface {
	Report(ctx context.Context, event asrcontract.ProviderProgress)
}

type Registry interface {
	DefaultProvider() Provider
	Provider(id string) (Provider, bool)
	Providers() []Provider
	Models(ctx context.Context) ([]asrcontract.ModelCard, error)
	SelectModel(ctx context.Context, providerID string, modelID string, required ...asrcontract.Capability) (asrcontract.ModelCard, error)
	Select(ctx context.Context, req SelectionRequest) (Provider, asrcontract.ModelCard, error)
}

type SelectionRequest struct {
	ProviderID string
	ModelID    string
	Requires   []asrcontract.Capability
}

type TaskRequest struct {
	JobID      string
	UserID     uint
	Operation  asrcontract.Operation
	AudioPath  string
	Progress   ProgressSink
	ModelID    string
	Parameters map[string]any
}

type TaskResult struct {
	Operation asrcontract.Operation
	ModelID   string
	EngineID  string
	Result    any
	Metadata  map[string]any
}

type TranscriptWord struct {
	Start   float64
	End     float64
	Word    string
	Speaker string
}

type TranscriptSegment struct {
	ID      string
	Start   float64
	End     float64
	Speaker string
	Text    string
}

type DiarizationSegment struct {
	Start   float64
	End     float64
	Speaker string
}

type TranscriptionResult struct {
	Text     string
	Language string
	Words    []TranscriptWord
	Segments []TranscriptSegment
	ModelID  string
	EngineID string
	Metadata map[string]any
}

type DiarizationResult struct {
	Segments []DiarizationSegment
	ModelID  string
	EngineID string
}
