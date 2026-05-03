package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"scriberr/internal/models"
)

func TestFileResponseDTOUsesPublicShapeAndOmitsPaths(t *testing.T) {
	title := "Board call"
	description := "Weekly sync"
	durationMs := int64(90500)
	job := &models.TranscriptionJob{
		ID:               "file-123",
		UserID:           42,
		Title:            &title,
		Status:           models.StatusUploaded,
		AudioPath:        "/private/tmp/scriberr/audio/file-123.wav",
		SourceFileName:   "call.wav",
		SourceDurationMs: &durationMs,
		LLMDescription:   &description,
		CreatedAt:        time.Unix(100, 0).UTC(),
		UpdatedAt:        time.Unix(200, 0).UTC(),
	}

	response := fileResponse(job, "audio/wav", "audio")
	if response.ID != "file_file-123" {
		t.Fatalf("unexpected file id: %q", response.ID)
	}
	if response.Status != "ready" {
		t.Fatalf("uploaded files should be exposed as ready, got %q", response.Status)
	}
	if response.DurationSeconds != 90.5 {
		t.Fatalf("unexpected duration seconds: %#v", response.DurationSeconds)
	}
	assertJSONDoesNotContain(t, response, "AudioPath", "audio_path", "source_file_path", "/private/tmp", "user_id", "deleted_at")
}

func TestTranscriptionResponseDTOSanitizesErrorAndUsesPublicIDs(t *testing.T) {
	title := "Transcript"
	sourceFileID := "source-file"
	language := "en"
	message := "failed reading /Users/zade/secret/audio.wav with hf_token=abc123"
	job := &models.TranscriptionJob{
		ID:             "tr-internal",
		Title:          &title,
		Status:         models.StatusFailed,
		AudioPath:      "/Users/zade/secret/audio.wav",
		SourceFileHash: &sourceFileID,
		Language:       &language,
		ErrorMessage:   &message,
		Progress:       0.4,
		ProgressStage:  "failed",
	}

	response := transcriptionResponse(job)
	if response.ID != "tr_tr-internal" {
		t.Fatalf("unexpected transcription id: %q", response.ID)
	}
	if response.FileID != "file_source-file" {
		t.Fatalf("unexpected file id: %q", response.FileID)
	}
	if response.Language != "en" {
		t.Fatalf("unexpected language: %#v", response.Language)
	}
	assertJSONDoesNotContain(t, response, "/Users/zade", "audio.wav", "hf_token=abc123", "AudioPath", "audio_path", "user_id", "deleted_at")
}

func TestProfileRecordingAndSummaryResponsesUseDTOs(t *testing.T) {
	description := "Default profile"
	profile := profileResponse(&models.TranscriptionProfile{
		ID:          "profile-id",
		Name:        "Default",
		Description: &description,
		IsDefault:   true,
		Parameters:  models.WhisperXParams{Model: "whisper-base", Diarize: true},
	})
	if profile.ID != "profile_profile-id" {
		t.Fatalf("unexpected profile id: %q", profile.ID)
	}
	assertJSONDoesNotContain(t, profile, "user_id", "deleted_at")

	fileID := "file-id"
	transcriptionID := "tr-id"
	durationMs := int64(1234)
	recording := recordingResponse(&models.RecordingSession{
		ID:              "rec-id",
		Status:          models.RecordingStatusReady,
		SourceKind:      models.RecordingSourceKindMicrophone,
		MimeType:        "audio/webm",
		FileID:          &fileID,
		TranscriptionID: &transcriptionID,
		DurationMs:      &durationMs,
	})
	if recording.ID != "rec_rec-id" || recording.FileID != "file_file-id" || recording.TranscriptionID != "tr_tr-id" {
		t.Fatalf("unexpected recording ids: %#v", recording)
	}
	assertJSONDoesNotContain(t, recording, "ClaimedBy", "claim_expires_at", "TemporaryArtifactsCleanedAt", "user_id", "deleted_at")

	errMessage := "provider failed at /tmp/private/model.bin"
	summary := summaryResponse(&models.Summary{
		ID:              "summary-id",
		TranscriptionID: "tr-id",
		Content:         "summary",
		Status:          "failed",
		ErrorMessage:    &errMessage,
	})
	if summary.TranscriptionID != "tr_tr-id" {
		t.Fatalf("unexpected summary transcription id: %q", summary.TranscriptionID)
	}
	assertJSONDoesNotContain(t, summary, "/tmp/private", "model.bin", "user_id")
}

func assertJSONDoesNotContain(t *testing.T, value any, forbidden ...string) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	body := string(data)
	for _, needle := range forbidden {
		if strings.Contains(body, needle) {
			t.Fatalf("response leaked %q in JSON: %s", needle, body)
		}
	}
}
