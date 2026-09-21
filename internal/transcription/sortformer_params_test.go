package transcription

import (
	"testing"

	"scriberr/internal/models"
)

func TestConvertToSortformerParamsPreservesSpeakerConstraints(t *testing.T) {
	minSpeakers := 2
	maxSpeakers := 2
	service := &UnifiedTranscriptionService{}

	params := service.convertToSortformerParams(models.WhisperXParams{
		MinSpeakers: &minSpeakers,
		MaxSpeakers: &maxSpeakers,
	})

	if params["min_speakers"] != minSpeakers {
		t.Fatalf("expected min_speakers=%d, got %v", minSpeakers, params["min_speakers"])
	}
	if params["max_speakers"] != maxSpeakers {
		t.Fatalf("expected max_speakers=%d, got %v", maxSpeakers, params["max_speakers"])
	}
}
