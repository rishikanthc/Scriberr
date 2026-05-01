package mediaimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidYouTubeURL(t *testing.T) {
	for _, rawURL := range []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://music.youtube.com/watch?v=dQw4w9WgXcQ",
	} {
		require.True(t, ValidYouTubeURL(rawURL), rawURL)
	}

	for _, rawURL := range []string{
		"file:///etc/passwd",
		"https://youtube.evil.test/watch?v=dQw4w9WgXcQ",
		"https://example.com/watch?v=dQw4w9WgXcQ",
	} {
		require.False(t, ValidYouTubeURL(rawURL), rawURL)
	}
}

func TestParseYTDLPProgress(t *testing.T) {
	progress, ok := parseYTDLPProgress("[download]  42.7% of 12.00MiB at 1.00MiB/s ETA 00:07")
	require.True(t, ok)
	require.InDelta(t, 42.7, progress, 0.001)

	progress, ok = parseYTDLPProgress("[download] 100.0% of 12.00MiB")
	require.True(t, ok)
	require.Equal(t, 100.0, progress)

	_, ok = parseYTDLPProgress("[ExtractAudio] Destination: output.mp3")
	require.False(t, ok)
}
