package pipeline

import (
	"context"
	"testing"

	"scriberr/internal/transcription/interfaces"
)

func TestAudioFormatPreprocessorAppliesTo(t *testing.T) {
	preprocessor := &AudioFormatPreprocessor{}

	tests := []struct {
		name         string
		capabilities interfaces.ModelCapabilities
		want         bool
	}{
		{
			name:         "local model normalizes by default",
			capabilities: interfaces.ModelCapabilities{ModelID: "whisperx"},
			want:         true,
		},
		{
			name:         "model opting out is skipped",
			capabilities: interfaces.ModelCapabilities{ModelID: "openai_whisper", SkipAudioNormalization: true},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := preprocessor.AppliesTo(tt.capabilities); got != tt.want {
				t.Errorf("AppliesTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

// countingPreprocessor stands in for AudioFormatPreprocessor so the test does
// not depend on ffmpeg being installed.
type countingPreprocessor struct {
	calls int
}

func (c *countingPreprocessor) AppliesTo(capabilities interfaces.ModelCapabilities) bool {
	return !capabilities.SkipAudioNormalization
}

func (c *countingPreprocessor) GetRequiredFormats() []string { return []string{"wav"} }

func (c *countingPreprocessor) Process(ctx context.Context, input interfaces.AudioInput) (interfaces.AudioInput, error) {
	c.calls++

	converted := input
	converted.Format = "wav"
	converted.SampleRate = 16000
	converted.Channels = 1
	converted.FilePath = input.FilePath + "_converted.wav"
	converted.TempFilePath = converted.FilePath

	return converted, nil
}

func newTestPipeline(preprocessor interfaces.Preprocessor) *ProcessingPipeline {
	pipeline := &ProcessingPipeline{}
	pipeline.RegisterPreprocessor(preprocessor)
	return pipeline
}

// A cloud transcription adapter and a local diarization adapter can consume the
// same job, so each must get audio prepared for its own requirements rather than
// one shared file that satisfies neither.
func TestProcessAudioIsIndependentPerAdapter(t *testing.T) {
	preprocessor := &countingPreprocessor{}
	pipeline := newTestPipeline(preprocessor)

	original := interfaces.AudioInput{
		FilePath:   "/tmp/meeting.mp3",
		Format:     "mp3",
		SampleRate: 44100,
		Channels:   2,
	}

	cloudTranscription := interfaces.ModelCapabilities{ModelID: "openai_whisper", SkipAudioNormalization: true}
	localDiarization := interfaces.ModelCapabilities{ModelID: "sortformer"}

	transcriptionInput, err := pipeline.ProcessAudio(context.Background(), original, cloudTranscription)
	if err != nil {
		t.Fatalf("ProcessAudio() for transcription returned error: %v", err)
	}

	diarizationInput, err := pipeline.ProcessAudio(context.Background(), original, localDiarization)
	if err != nil {
		t.Fatalf("ProcessAudio() for diarization returned error: %v", err)
	}

	if transcriptionInput.FilePath != original.FilePath {
		t.Errorf("transcription input = %q, want the untouched original %q", transcriptionInput.FilePath, original.FilePath)
	}
	if transcriptionInput.TempFilePath != "" {
		t.Errorf("transcription input created a temp file %q, want none", transcriptionInput.TempFilePath)
	}
	if diarizationInput.SampleRate != 16000 || diarizationInput.Channels != 1 {
		t.Errorf("diarization input = %d Hz/%d channels, want 16000 Hz/1 channel",
			diarizationInput.SampleRate, diarizationInput.Channels)
	}
	if preprocessor.calls != 1 {
		t.Errorf("preprocessor ran %d times, want 1 (only for the diarization model)", preprocessor.calls)
	}
}

func TestPreprocessorSignature(t *testing.T) {
	pipeline := newTestPipeline(&countingPreprocessor{})

	normalizing := interfaces.ModelCapabilities{ModelID: "whisperx"}
	alsoNormalizing := interfaces.ModelCapabilities{ModelID: "sortformer"}
	skipping := interfaces.ModelCapabilities{ModelID: "openai_whisper", SkipAudioNormalization: true}

	// Two consumers wanting the same output share a signature, so the caller can
	// reuse one conversion instead of running it twice.
	if pipeline.PreprocessorSignature(normalizing) != pipeline.PreprocessorSignature(alsoNormalizing) {
		t.Error("models with identical preprocessing needs got different signatures")
	}

	// Diverging needs must not collapse onto one shared file.
	if pipeline.PreprocessorSignature(normalizing) == pipeline.PreprocessorSignature(skipping) {
		t.Error("normalizing and skipping models got the same signature")
	}

	if got := pipeline.PreprocessorSignature(skipping); got != "" {
		t.Errorf("signature for a model skipping all preprocessing = %q, want empty", got)
	}
}
