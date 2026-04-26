package engineprovider

import (
	"context"
	"errors"
	"strings"
	"testing"

	speechengine "scriberr-engine/speech/engine"
	speechmodels "scriberr-engine/speech/models"
	"scriberr-engine/speech/runtime"
)

type fakeSpeechEngine struct {
	transcriptionReq speechengine.TranscriptionRequest
	diarizationReq   speechengine.DiarizationRequest
	transcriptionOut *speechengine.TranscriptionResult
	diarizationOut   *speechengine.DiarizationResult
	err              error
	installed        map[string]bool
	closed           bool
}

func (e *fakeSpeechEngine) Transcribe(ctx context.Context, req speechengine.TranscriptionRequest) (*speechengine.TranscriptionResult, error) {
	e.transcriptionReq = req
	if e.err != nil {
		return nil, e.err
	}
	return e.transcriptionOut, nil
}

func (e *fakeSpeechEngine) Diarize(ctx context.Context, req speechengine.DiarizationRequest) (*speechengine.DiarizationResult, error) {
	e.diarizationReq = req
	if e.err != nil {
		return nil, e.err
	}
	return e.diarizationOut, nil
}

func (e *fakeSpeechEngine) IsModelInstalled(modelID string) bool {
	return e.installed[modelID]
}

func (e *fakeSpeechEngine) Close() error {
	e.closed = true
	return nil
}

func TestLocalProviderTranscribeMapsRequestAndWords(t *testing.T) {
	fake := &fakeSpeechEngine{
		transcriptionOut: &speechengine.TranscriptionResult{
			Text:     "hello world",
			Language: "en",
			Words: []speechengine.TranscriptWord{
				{Text: "hello", StartSec: 0.1, EndSec: 0.4},
				{Text: "world", StartSec: 0.5, EndSec: 0.9},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 4}, runtime.ProviderCPU, fake, nil)

	result, err := provider.Transcribe(context.Background(), TranscriptionRequest{
		JobID:     "job-1",
		UserID:    7,
		AudioPath: "/tmp/audio.wav",
		ModelID:   "whisper-tiny",
		Language:  "en",
		Task:      "translate",
		Threads:   2,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}

	if fake.transcriptionReq.ModelID != "whisper-tiny" {
		t.Fatalf("ModelID = %q", fake.transcriptionReq.ModelID)
	}
	if fake.transcriptionReq.Language != "en" {
		t.Fatalf("Language = %q", fake.transcriptionReq.Language)
	}
	if fake.transcriptionReq.Task != "translate" {
		t.Fatalf("Task = %q", fake.transcriptionReq.Task)
	}
	if fake.transcriptionReq.NumThreads != 2 {
		t.Fatalf("NumThreads = %d", fake.transcriptionReq.NumThreads)
	}
	if fake.transcriptionReq.Provider != runtime.ProviderCPU {
		t.Fatalf("Provider = %q", fake.transcriptionReq.Provider)
	}
	if fake.transcriptionReq.EnableTokenTimestamps == nil || !*fake.transcriptionReq.EnableTokenTimestamps {
		t.Fatalf("EnableTokenTimestamps was not forced on")
	}
	if result.Text != "hello world" || result.Language != "en" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(result.Words) != 2 || result.Words[0].Word != "hello" || result.Words[1].End != 0.9 {
		t.Fatalf("unexpected words: %#v", result.Words)
	}
	if result.ModelID != "whisper-tiny" || result.EngineID != "local" {
		t.Fatalf("unexpected model/engine ids: %#v", result)
	}
}

func TestLocalProviderTranscribeDefaultsAndEmptyWords(t *testing.T) {
	fake := &fakeSpeechEngine{
		transcriptionOut: &speechengine.TranscriptionResult{Text: "text"},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 4}, runtime.ProviderCPU, fake, nil)

	result, err := provider.Transcribe(context.Background(), TranscriptionRequest{})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if fake.transcriptionReq.ModelID != DefaultTranscriptionModel {
		t.Fatalf("ModelID = %q, want %q", fake.transcriptionReq.ModelID, DefaultTranscriptionModel)
	}
	if fake.transcriptionReq.Task != "transcribe" {
		t.Fatalf("Task = %q, want transcribe", fake.transcriptionReq.Task)
	}
	if fake.transcriptionReq.NumThreads != 4 {
		t.Fatalf("NumThreads = %d, want 4", fake.transcriptionReq.NumThreads)
	}
	if result.Words == nil {
		t.Fatalf("Words is nil, want empty array")
	}
}

func TestLocalProviderDiarizeMapsRequestAndSpeakers(t *testing.T) {
	fake := &fakeSpeechEngine{
		diarizationOut: &speechengine.DiarizationResult{
			Segments: []speechengine.DiarizationSegment{
				{Speaker: 0, Start: 0, End: 1.5},
				{Speaker: 12, Start: 1.6, End: 3.2},
			},
		},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{Threads: 3}, runtime.ProviderCPU, fake, nil)

	result, err := provider.Diarize(context.Background(), DiarizationRequest{
		AudioPath:   "/tmp/audio.wav",
		ModelID:     "diarization-default",
		NumSpeakers: 2,
	})
	if err != nil {
		t.Fatalf("Diarize returned error: %v", err)
	}
	if fake.diarizationReq.ModelID != "diarization-default" {
		t.Fatalf("ModelID = %q", fake.diarizationReq.ModelID)
	}
	if fake.diarizationReq.NumClusters != 2 {
		t.Fatalf("NumClusters = %d", fake.diarizationReq.NumClusters)
	}
	if fake.diarizationReq.NumThreads != 3 {
		t.Fatalf("NumThreads = %d", fake.diarizationReq.NumThreads)
	}
	if len(result.Segments) != 2 || result.Segments[0].Speaker != "SPEAKER_00" || result.Segments[1].Speaker != "SPEAKER_12" {
		t.Fatalf("unexpected segments: %#v", result.Segments)
	}
}

func TestLocalProviderCapabilitiesUseModelRegistryAndInstallState(t *testing.T) {
	fake := &fakeSpeechEngine{installed: map[string]bool{"whisper-base": true}}
	specs := []speechmodels.ModelSpec{
		{ID: "whisper-base", DisplayName: "Whisper Base", Family: speechmodels.FamilyWhisper},
		{ID: "diarization-default", DisplayName: "Diarization", Family: speechmodels.FamilyDiarize},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake, specs)

	capabilities, err := provider.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if len(capabilities) != 2 {
		t.Fatalf("capabilities length = %d", len(capabilities))
	}
	if !capabilities[0].Installed || !capabilities[0].Default {
		t.Fatalf("whisper-base capability missing installed/default: %#v", capabilities[0])
	}
	if strings.Join(capabilities[0].Capabilities, ",") != "transcription,word_timestamps" {
		t.Fatalf("whisper capabilities = %#v", capabilities[0].Capabilities)
	}
	if capabilities[1].Installed || !capabilities[1].Default {
		t.Fatalf("diarization capability installed/default mismatch: %#v", capabilities[1])
	}
	if strings.Join(capabilities[1].Capabilities, ",") != "diarization" {
		t.Fatalf("diarization capabilities = %#v", capabilities[1].Capabilities)
	}
}

func TestLocalProviderSanitizesErrors(t *testing.T) {
	fake := &fakeSpeechEngine{
		err: errors.New("load /Users/zade/Code/asr/Scriberr/data/uploads/audio.wav failed token=secret"),
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake, nil)

	_, err := provider.Transcribe(context.Background(), TranscriptionRequest{})
	if err == nil {
		t.Fatalf("Transcribe returned nil error")
	}
	msg := err.Error()
	if strings.Contains(msg, "/Users/") || strings.Contains(msg, "secret") {
		t.Fatalf("error was not sanitized: %q", msg)
	}
	if !strings.Contains(msg, "[redacted-path]") || !strings.Contains(msg, "token=[redacted]") {
		t.Fatalf("error missing sanitized markers: %q", msg)
	}
}

func TestLocalProviderCloseClosesEngine(t *testing.T) {
	fake := &fakeSpeechEngine{}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake, nil)

	if err := provider.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !fake.closed {
		t.Fatalf("fake engine was not closed")
	}
}
