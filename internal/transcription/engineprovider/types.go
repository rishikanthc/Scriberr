package engineprovider

import (
	"context"

	"scriberr/internal/transcription/asrcontract"
)

const (
	DefaultProviderID         = "local"
	DefaultTranscriptionModel = "whisper-base"
	DefaultDiarizationModel   = "diarization-default"
)

type Provider interface {
	ID() string
	Inspect(ctx context.Context) (*asrcontract.ProviderInfo, error)
	Models(ctx context.Context) ([]asrcontract.ModelCard, error)
	Status(ctx context.Context) (*asrcontract.ProviderStatus, error)
	LoadModel(ctx context.Context, req asrcontract.LoadModelRequest) error
	UnloadModel(ctx context.Context, req asrcontract.UnloadModelRequest) error
	LoadedModels(ctx context.Context) ([]asrcontract.LoadedModel, error)
	Capabilities(ctx context.Context) ([]ModelCapability, error)
	Prepare(ctx context.Context) error
	Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error)
	Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error)
	IdentifySpeakers(ctx context.Context, req asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error)
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
	Capabilities(ctx context.Context) ([]ModelCapability, error)
	Select(ctx context.Context, req SelectionRequest) (Provider, *ModelCapability, error)
}

type ModelCapability struct {
	ID           string
	Name         string
	Provider     string
	Installed    bool
	Default      bool
	Capabilities []string
}

type SelectionRequest struct {
	ProviderID string
	ModelID    string
	Requires   []string
}

type TranscriptionRequest struct {
	JobID      string
	UserID     uint
	AudioPath  string
	Progress   ProgressSink
	ModelID    string
	Parameters map[string]any
}

type DiarizationRequest struct {
	JobID      string
	UserID     uint
	AudioPath  string
	Progress   ProgressSink
	ModelID    string
	Parameters map[string]any
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
