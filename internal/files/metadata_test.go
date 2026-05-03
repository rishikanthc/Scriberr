package files

import (
	"os"
	"path/filepath"
	"testing"

	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func TestFileMetadataUsesStorageBoundaryAndSafeDefaults(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "clip.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("audio-bytes"), 0600))
	durationMs := int64(2500)
	job := &models.TranscriptionJob{
		AudioPath:        audioPath,
		SourceFileName:   "clip.wav",
		SourceDurationMs: &durationMs,
		SourceFileHash:   stringPtrForTest("source"),
	}
	service := NewService(nil, Config{})

	metadata := service.Metadata(job)

	require.Equal(t, "audio/wav", metadata.MimeType)
	require.Equal(t, "audio", metadata.Kind)
	require.Equal(t, int64(len("audio-bytes")), metadata.SizeBytes)
	require.Equal(t, &durationMs, metadata.DurationMs)
}

func TestFileMetadataDoesNotFailWhenPhysicalFileIsMissing(t *testing.T) {
	job := &models.TranscriptionJob{
		AudioPath:      filepath.Join(t.TempDir(), "missing.mp3"),
		SourceFileName: "missing.mp3",
	}
	service := NewService(nil, Config{})

	metadata := service.Metadata(job)

	require.Equal(t, "audio/mpeg", metadata.MimeType)
	require.Equal(t, "audio", metadata.Kind)
	require.Zero(t, metadata.SizeBytes)
}

func TestFileMetadataHandlesYouTubeSourceNames(t *testing.T) {
	service := NewService(nil, Config{})

	metadata := service.Metadata(&models.TranscriptionJob{SourceFileName: "youtube:https://youtu.be/example"})

	require.Equal(t, "youtube", metadata.Kind)
}

func stringPtrForTest(value string) *string {
	return &value
}
