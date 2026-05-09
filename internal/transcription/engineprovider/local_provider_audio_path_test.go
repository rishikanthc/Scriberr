package engineprovider

import (
	"context"
	"testing"

	speechengine "scriberr-engine/speech/engine"
	"scriberr-engine/speech/runtime"
)

func TestLocalProviderUsesLocalAudioPathForInProcessEngine(t *testing.T) {
	fake := &fakeSpeechEngine{
		transcriptionOut: &speechengine.TranscriptionResult{Text: "hello"},
		diarizationOut:   &speechengine.DiarizationResult{},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake)

	if _, err := transcribeForTest(context.Background(), provider, TaskRequest{
		JobID:          "job-local-path",
		AudioPath:      "/provider-input/audio/audio.wav",
		LocalAudioPath: "/tmp/scriberr-normalized/audio.wav",
		ModelID:        "parakeet-v3",
	}); err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if fake.transcriptionReq.AudioPath != "/tmp/scriberr-normalized/audio.wav" {
		t.Fatalf("transcription AudioPath = %q", fake.transcriptionReq.AudioPath)
	}

	if _, err := diarizeForTest(context.Background(), provider, TaskRequest{
		JobID:          "job-local-path",
		AudioPath:      "/provider-input/audio/audio.wav",
		LocalAudioPath: "/tmp/scriberr-normalized/audio.wav",
		ModelID:        "diarization-default",
	}); err != nil {
		t.Fatalf("Diarize returned error: %v", err)
	}
	if fake.diarizationReq.AudioPath != "/tmp/scriberr-normalized/audio.wav" {
		t.Fatalf("diarization AudioPath = %q", fake.diarizationReq.AudioPath)
	}
}

func TestLocalProviderFallsBackToProviderAudioPath(t *testing.T) {
	fake := &fakeSpeechEngine{
		transcriptionOut: &speechengine.TranscriptionResult{Text: "hello"},
	}
	provider := newLocalProviderWithEngine("local", LocalConfig{}, runtime.ProviderCPU, fake)

	if _, err := transcribeForTest(context.Background(), provider, TaskRequest{
		JobID:     "job-provider-path",
		AudioPath: "/provider-input/audio/audio.wav",
		ModelID:   "parakeet-v3",
	}); err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if fake.transcriptionReq.AudioPath != "/provider-input/audio/audio.wav" {
		t.Fatalf("transcription AudioPath = %q", fake.transcriptionReq.AudioPath)
	}
}
