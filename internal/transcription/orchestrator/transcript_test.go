package orchestrator

import (
	"encoding/json"
	"testing"

	"scriberr/internal/transcription/engineprovider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCanonicalTranscriptWithWordsAndDiarization(t *testing.T) {
	transcript, err := BuildCanonicalTranscript(
		&engineprovider.TranscriptionResult{
			Text:     "Hello world. Bye now.",
			Language: "en",
			ModelID:  "whisper-base",
			EngineID: "local",
			Words: []engineprovider.TranscriptWord{
				{Start: 0.0, End: 0.4, Word: "Hello"},
				{Start: 0.5, End: 0.9, Word: "world"},
				{Start: 2.0, End: 2.3, Word: "Bye"},
				{Start: 2.4, End: 2.8, Word: "now"},
			},
			Segments: []engineprovider.TranscriptSegment{
				{Start: 0.0, End: 1.0, Text: "Hello world."},
				{Start: 2.0, End: 3.0, Text: "Bye now."},
			},
		},
		&engineprovider.DiarizationResult{
			ModelID:  "diarization-default",
			EngineID: "local",
			Segments: []engineprovider.DiarizationSegment{
				{Start: 0.0, End: 1.2, Speaker: "speaker-a"},
				{Start: 1.8, End: 3.2, Speaker: "speaker-b"},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello world. Bye now.", transcript.Text)
	assert.Equal(t, "en", transcript.Language)
	require.Len(t, transcript.Segments, 2)
	assert.Equal(t, "seg_000001", transcript.Segments[0].ID)
	assert.Equal(t, "SPEAKER_00", transcript.Segments[0].Speaker)
	assert.Equal(t, "SPEAKER_01", transcript.Segments[1].Speaker)
	require.Len(t, transcript.Words, 4)
	assert.Equal(t, "SPEAKER_00", transcript.Words[0].Speaker)
	assert.Equal(t, "SPEAKER_01", transcript.Words[3].Speaker)
	assert.Equal(t, "local", transcript.Engine.Provider)
	assert.Equal(t, "whisper-base", transcript.Engine.TranscriptionModel)
	assert.Equal(t, "diarization-default", transcript.Engine.DiarizationModel)
}

func TestBuildCanonicalTranscriptWithoutWordsPreservesEmptyWords(t *testing.T) {
	transcript, err := BuildCanonicalTranscript(
		&engineprovider.TranscriptionResult{
			Text:     "No token timestamps.",
			Language: "en",
			ModelID:  "whisper-base",
			EngineID: "local",
			Segments: []engineprovider.TranscriptSegment{
				{Start: 0.0, End: 2.0, Text: "No token timestamps."},
			},
		},
		nil,
	)

	require.NoError(t, err)
	assert.Empty(t, transcript.Words)
	require.NotNil(t, transcript.Words)
	require.Len(t, transcript.Segments, 1)
	assert.Empty(t, transcript.Segments[0].Speaker)

	encoded, err := json.Marshal(transcript)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"words":[]`)
	assert.NotContains(t, string(encoded), `"speaker"`)
}

func TestBuildCanonicalTranscriptGeneratesFallbackSegmentFromWords(t *testing.T) {
	transcript, err := BuildCanonicalTranscript(
		&engineprovider.TranscriptionResult{
			Text:     "Fallback segment.",
			Language: "en",
			ModelID:  "whisper-base",
			EngineID: "local",
			Words: []engineprovider.TranscriptWord{
				{Start: 1.5, End: 1.8, Word: "Fallback"},
				{Start: 2.0, End: 2.4, Word: "segment"},
			},
		},
		nil,
	)

	require.NoError(t, err)
	require.Len(t, transcript.Segments, 1)
	assert.Equal(t, "seg_000001", transcript.Segments[0].ID)
	assert.Equal(t, 1.5, transcript.Segments[0].Start)
	assert.Equal(t, 2.4, transcript.Segments[0].End)
	assert.Equal(t, "Fallback segment.", transcript.Segments[0].Text)
}

func TestBuildCanonicalTranscriptRejectsEmptyEngineOutput(t *testing.T) {
	_, err := BuildCanonicalTranscript(&engineprovider.TranscriptionResult{
		ModelID:  "parakeet-v3",
		EngineID: "local",
		Segments: []engineprovider.TranscriptSegment{
			{Start: 0, End: 1, Text: "   "},
		},
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no usable text")
}

func TestParseStoredTranscriptFallbacks(t *testing.T) {
	plain, err := ParseStoredTranscript("legacy plain text")
	require.NoError(t, err)
	assert.Equal(t, "legacy plain text", plain.Text)
	assert.Empty(t, plain.Segments)
	require.NotNil(t, plain.Words)

	older, err := ParseStoredTranscript(`{"text":"old json","segments":[{"id":"seg_000001","start":0,"end":1,"text":"old json"}]}`)
	require.NoError(t, err)
	assert.Equal(t, "old json", older.Text)
	require.Len(t, older.Segments, 1)
	assert.Empty(t, older.Words)
	require.NotNil(t, older.Words)
}
