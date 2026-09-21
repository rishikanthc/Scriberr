package autosummary

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	transcriptioninterfaces "scriberr/internal/transcription/interfaces"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newAutoSummaryTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:auto-summary-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.SummarySetting{}, &models.SummaryTemplate{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return &Service{summaryRepo: repository.NewSummaryRepository(db)}, db
}

func TestProcessDoesNothingWithoutSettings(t *testing.T) {
	service, _ := newAutoSummaryTestService(t)

	if err := service.Process(context.Background(), "job-123"); err != nil {
		t.Fatalf("Process returned an error: %v", err)
	}
}

func TestProcessDoesNothingWhenDisabled(t *testing.T) {
	service, db := newAutoSummaryTestService(t)
	if err := db.Create(&models.SummarySetting{AutoSummarize: false}).Error; err != nil {
		t.Fatalf("create summary settings: %v", err)
	}

	if err := service.Process(context.Background(), "job-123"); err != nil {
		t.Fatalf("Process returned an error: %v", err)
	}
}

func TestBuildSummaryContentUsesPlainTranscriptText(t *testing.T) {
	transcriptJSON := marshalTranscript(t, transcriptioninterfaces.TranscriptResult{
		Text: "Hello from the transcript.",
	})

	content, err := buildSummaryContent(transcriptJSON, "Summarize this.", false, nil)
	if err != nil {
		t.Fatalf("buildSummaryContent returned an error: %v", err)
	}

	want := "Transcript:\nHello from the transcript.\n\nInstructions:\nSummarize this."
	if content != want {
		t.Fatalf("buildSummaryContent() = %q, want %q", content, want)
	}
}

func TestBuildSummaryContentUsesSpeakerLabelsAndMappings(t *testing.T) {
	speakerOne := "SPEAKER_00"
	speakerTwo := "SPEAKER_01"
	transcriptJSON := marshalTranscript(t, transcriptioninterfaces.TranscriptResult{
		Text: "Hello. Hi.",
		Segments: []transcriptioninterfaces.TranscriptSegment{
			{Text: " Hello. ", Speaker: &speakerOne},
			{Text: "Hi.", Speaker: &speakerTwo},
		},
	})
	mappings := []models.SpeakerMapping{
		{OriginalSpeaker: speakerOne, CustomName: "Jordan"},
	}

	content, err := buildSummaryContent(transcriptJSON, "Summarize this.", true, mappings)
	if err != nil {
		t.Fatalf("buildSummaryContent returned an error: %v", err)
	}

	want := "Transcript (with speaker labels - each line is prefixed with [SPEAKER_NAME]):\n" +
		"[Jordan] Hello.\n[SPEAKER_01] Hi.\n\nInstructions:\nSummarize this."
	if content != want {
		t.Fatalf("buildSummaryContent() = %q, want %q", content, want)
	}
}

func TestBuildSummaryContentRejectsInvalidTranscriptJSON(t *testing.T) {
	if _, err := buildSummaryContent("not JSON", "Summarize this.", false, nil); err == nil {
		t.Fatal("buildSummaryContent returned no error for invalid transcript JSON")
	}
}

func marshalTranscript(t *testing.T, transcript transcriptioninterfaces.TranscriptResult) string {
	t.Helper()

	data, err := json.Marshal(transcript)
	if err != nil {
		t.Fatalf("marshal transcript: %v", err)
	}
	return string(data)
}
