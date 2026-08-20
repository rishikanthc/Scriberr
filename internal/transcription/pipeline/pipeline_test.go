package pipeline

import (
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
