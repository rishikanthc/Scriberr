package engineprovider

import "context"

const (
	DefaultProviderID         = "local"
	DefaultTranscriptionModel = "whisper-base"
	DefaultDiarizationModel   = "diarization-default"
)

type Provider interface {
	ID() string
	Capabilities(ctx context.Context) ([]ModelCapability, error)
	Prepare(ctx context.Context) error
	Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error)
	Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error)
	Close() error
}

type Registry interface {
	DefaultProvider() Provider
	Provider(id string) (Provider, bool)
	Capabilities(ctx context.Context) ([]ModelCapability, error)
}

type ModelCapability struct {
	ID           string
	Name         string
	Provider     string
	Installed    bool
	Default      bool
	Capabilities []string
}

type TranscriptionRequest struct {
	JobID     string
	UserID    uint
	AudioPath string
	ModelID   string
	Language  string
	Task      string
	Threads   int
}

type DiarizationRequest struct {
	JobID       string
	UserID      uint
	AudioPath   string
	ModelID     string
	NumSpeakers int
	MinSpeakers *int
	MaxSpeakers *int
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
}

type DiarizationResult struct {
	Segments []DiarizationSegment
	ModelID  string
	EngineID string
}
