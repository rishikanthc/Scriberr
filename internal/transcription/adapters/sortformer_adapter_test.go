package adapters

import (
	"testing"

	"scriberr/internal/transcription/interfaces"
)

func TestSortformerEnforceSpeakerLimitRemapsExtraSpeakers(t *testing.T) {
	adapter := NewSortformerAdapter("/tmp/sortformer")
	result := &interfaces.DiarizationResult{
		Segments: []interfaces.DiarizationSegment{
			{Start: 0, End: 10, Speaker: "speaker_0", Confidence: 1.0},
			{Start: 12, End: 20, Speaker: "speaker_1", Confidence: 1.0},
			{Start: 21, End: 22, Speaker: "speaker_2", Confidence: 1.0},
		},
		SpeakerCount: 3,
		Speakers:     []string{"speaker_0", "speaker_1", "speaker_2"},
	}

	capped := adapter.enforceSpeakerLimit(result, map[string]interface{}{
		"max_speakers": 2,
	})

	if capped.SpeakerCount != 2 {
		t.Fatalf("expected 2 speakers, got %d", capped.SpeakerCount)
	}
	if capped.Segments[2].Speaker != "speaker_1" {
		t.Fatalf("expected speaker_2 segment to map to nearest retained speaker_1, got %s", capped.Segments[2].Speaker)
	}
	for _, speaker := range capped.Speakers {
		if speaker == "speaker_2" {
			t.Fatal("speaker_2 should have been removed from the speaker summary")
		}
	}
}
