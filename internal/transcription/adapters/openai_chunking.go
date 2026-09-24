package adapters

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

// openAIModelDurationCaps lists the maximum audio duration the OpenAI API
// accepts per model. The newer speech models reject anything longer with an
// HTTP 400 ("audio duration X seconds is longer than 1400 seconds which is the
// maximum for this model"), which rules out meeting-length recordings unless the
// audio is split client side.
//
// The API does not expose these limits, so they are hardcoded and may need
// updating if OpenAI changes them. Models absent from this map are treated as
// uncapped (whisper-1 only has the 25MB request-size limit) and keep taking the
// single-shot path.
var openAIModelDurationCaps = map[string]time.Duration{
	"gpt-4o-transcribe":      1400 * time.Second,
	"gpt-4o-mini-transcribe": 1400 * time.Second,
}

const (
	// openAIChunkSafetyMargin keeps each chunk comfortably below the cap, since
	// cutting on frame boundaries can make a chunk marginally longer than asked.
	openAIChunkSafetyMargin = 10 * time.Second

	// openAIChunkOverlap is repeated at the start of every chunk after the first
	// so a word spoken across a cut point is not lost. The duplicated text is
	// removed again while reassembling.
	openAIChunkOverlap = 3 * time.Second

	// openAIOverlapWordWindow bounds how many words the overlap de-duplication
	// compares; openAIChunkOverlap of speech is well under this.
	openAIOverlapWordWindow = 24
)

// openAIDurationCap returns the maximum audio duration for a model, or 0 when
// the model has no known duration limit.
func openAIDurationCap(model string) time.Duration {
	return openAIModelDurationCaps[model]
}

// audioChunk is a half-open [Start, End) slice of the original audio.
type audioChunk struct {
	Index int
	Start time.Duration
	End   time.Duration
}

// Duration returns the length of the chunk.
func (c audioChunk) Duration() time.Duration {
	return c.End - c.Start
}

// planAudioChunks splits total into chunks of at most maxLength, each overlapping the
// previous one by overlap. Audio that already fits within maxLength yields a single
// chunk covering the whole file, so callers never chunk needlessly.
func planAudioChunks(total, maxLength, overlap time.Duration) []audioChunk {
	if total <= 0 {
		return nil
	}
	if maxLength <= 0 || total <= maxLength {
		return []audioChunk{{Index: 0, Start: 0, End: total}}
	}
	if overlap < 0 || overlap >= maxLength {
		overlap = 0
	}

	step := maxLength - overlap
	var chunks []audioChunk
	for start := time.Duration(0); start < total; start += step {
		end := start + maxLength
		if end > total {
			end = total
		}
		chunks = append(chunks, audioChunk{Index: len(chunks), Start: start, End: end})
		if end == total {
			break
		}
	}

	return chunks
}

// chunkCutPoint returns, in seconds, the middle of the region two consecutive
// chunks both cover. Content before it sits more centrally in prev, content
// after it more centrally in next, which is where the merge switches chunks.
func chunkCutPoint(prev, next audioChunk) float64 {
	return next.Start.Seconds() + (prev.End.Seconds()-next.Start.Seconds())/2
}

// mergeChunkTranscripts stitches per-chunk results back into one transcript.
// Timestamps are shifted to be relative to the original audio, and the region
// two chunks share is taken from whichever chunk covers it more centrally. Models
// that answer without segment timestamps (the gpt-4o family returns plain JSON)
// produce one segment per chunk, for which the overlap is removed by dropping
// the repeated leading words instead.
func mergeChunkTranscripts(chunks []audioChunk, results []*interfaces.TranscriptResult) *interfaces.TranscriptResult {
	merged := &interfaces.TranscriptResult{}
	var texts []string

	for i, chunk := range chunks {
		if i >= len(results) || results[i] == nil {
			continue
		}
		result := results[i]

		if merged.Language == "" {
			merged.Language = result.Language
		}

		offset := chunk.Start.Seconds()
		lower := math.Inf(-1)
		if i > 0 {
			lower = chunkCutPoint(chunks[i-1], chunk)
		}
		upper := math.Inf(1)
		if i+1 < len(chunks) {
			upper = chunkCutPoint(chunk, chunks[i+1])
		}

		firstOfChunk := true
		for _, segment := range result.Segments {
			segment.Start += offset
			segment.End += offset
			if mid := (segment.Start + segment.End) / 2; mid < lower || mid >= upper {
				continue
			}

			if i > 0 && firstOfChunk && len(merged.Segments) > 0 {
				previous := merged.Segments[len(merged.Segments)-1]
				segment.Text = trimRepeatedPrefix(previous.Text, segment.Text, openAIOverlapWordWindow)
				if strings.TrimSpace(segment.Text) == "" {
					continue
				}
				// Keep the timeline monotonic across a boundary the chunks share.
				if segment.Start < previous.End {
					segment.Start = previous.End
				}
				if segment.End < segment.Start {
					segment.End = segment.Start
				}
			}
			firstOfChunk = false

			merged.Segments = append(merged.Segments, segment)
			if text := strings.TrimSpace(segment.Text); text != "" {
				texts = append(texts, text)
			}
		}

		for _, word := range result.WordSegments {
			word.Start += offset
			word.End += offset
			if mid := (word.Start + word.End) / 2; mid < lower || mid >= upper {
				continue
			}
			merged.WordSegments = append(merged.WordSegments, word)
		}
	}

	merged.Text = strings.Join(texts, " ")
	return merged
}

// trimRepeatedPrefix removes the leading words of next that repeat the trailing
// words of prev, which is what an overlapping chunk boundary produces. Only the
// longest match within maxWords is considered; unrelated text is left alone.
func trimRepeatedPrefix(prev, next string, maxWords int) string {
	prevWords := strings.Fields(prev)
	nextWords := strings.Fields(next)
	if len(prevWords) == 0 || len(nextWords) == 0 {
		return next
	}

	limit := maxWords
	if len(prevWords) < limit {
		limit = len(prevWords)
	}
	if len(nextWords) < limit {
		limit = len(nextWords)
	}

	for n := limit; n > 0; n-- {
		if wordsEqual(prevWords[len(prevWords)-n:], nextWords[:n]) {
			return strings.Join(nextWords[n:], " ")
		}
	}

	return next
}

// wordsEqual compares two word runs ignoring case and punctuation, which differ
// between two transcriptions of the same speech.
func wordsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		left, right := normalizeWord(a[i]), normalizeWord(b[i])
		if left == "" || left != right {
			return false
		}
	}
	return true
}

func normalizeWord(word string) string {
	return strings.ToLower(strings.TrimFunc(word, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}))
}

// transcribeInChunks splits the audio into chunks that fit the model's duration
// cap, transcribes each one on its own, and reassembles the results.
func (a *OpenAIAdapter) transcribeInChunks(
	ctx context.Context,
	input interfaces.AudioInput,
	model, apiKey string,
	params map[string]interface{},
	procCtx interfaces.ProcessingContext,
	durationCap time.Duration,
	writeLog func(format string, args ...interface{}),
) (*interfaces.TranscriptResult, error) {
	chunkLength := durationCap - openAIChunkSafetyMargin
	if chunkLength <= 0 {
		chunkLength = durationCap
	}

	chunks := planAudioChunks(input.Duration, chunkLength, openAIChunkOverlap)
	writeLog("Audio is %.0fs, longer than the %.0fs limit of %s: splitting into %d chunks",
		input.Duration.Seconds(), durationCap.Seconds(), model, len(chunks))
	logger.Info("Chunking audio for duration-capped OpenAI model",
		"model", model,
		"duration", input.Duration,
		"max_duration", durationCap,
		"chunks", len(chunks))

	chunkPaths, err := splitAudioIntoChunks(ctx, input.FilePath, chunkDirectory(procCtx), procCtx.JobID, chunks)
	defer func() {
		for _, path := range chunkPaths {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				logger.Warn("Failed to clean up audio chunk", "file", path, "error", err)
			}
		}
	}()
	if err != nil {
		writeLog("Error: Failed to split audio: %v", err)
		return nil, err
	}

	results := make([]*interfaces.TranscriptResult, len(chunks))
	for i, chunk := range chunks {
		writeLog("Transcribing chunk %d/%d (%.0fs - %.0fs)", i+1, len(chunks), chunk.Start.Seconds(), chunk.End.Seconds())
		result, err := a.transcribeFile(ctx, chunkPaths[i], model, apiKey, chunk.Duration(), params, writeLog)
		if err != nil {
			return nil, fmt.Errorf("chunk %d/%d failed: %w", i+1, len(chunks), err)
		}
		results[i] = result
	}

	merged := mergeChunkTranscripts(chunks, results)
	writeLog("Merged %d chunks into %d segments", len(chunks), len(merged.Segments))

	return merged, nil
}

// chunkDirectory picks where chunk files are written.
func chunkDirectory(procCtx interfaces.ProcessingContext) string {
	if procCtx.TempDirectory != "" {
		return procCtx.TempDirectory
	}
	return os.TempDir()
}

// splitAudioIntoChunks writes each planned chunk to its own file and returns the
// paths. Any file already written is removed when a later chunk fails.
func splitAudioIntoChunks(ctx context.Context, sourcePath, tempDir, jobID string, chunks []audioChunk) ([]string, error) {
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunk directory: %w", err)
	}

	extension := filepath.Ext(sourcePath)
	if extension == "" {
		extension = ".mp3"
	}

	paths := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("scriberr_%s_chunk_%03d%s", jobID, chunk.Index, extension))
		if err := extractAudioChunk(ctx, sourcePath, outputPath, chunk); err != nil {
			return paths, err
		}
		paths = append(paths, outputPath)
	}

	return paths, nil
}

// extractAudioChunk cuts one chunk out of the source file with FFmpeg.
func extractAudioChunk(ctx context.Context, sourcePath, outputPath string, chunk audioChunk) error {
	// Stream copying is exact for audio-only streams and avoids re-encoding what
	// can be hours of audio; containers that cannot be cut that way fall back to
	// a re-encode.
	args := []string{
		"-nostdin",
		"-ss", formatFFmpegSeconds(chunk.Start),
		"-i", sourcePath,
		"-t", formatFFmpegSeconds(chunk.Duration()),
		"-vn",
		"-map", "0:a:0",
		"-c:a", "copy",
		"-y",
		outputPath,
	}

	output, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput()
	if err == nil {
		return nil
	}
	logger.Warn("FFmpeg stream copy failed, re-encoding chunk",
		"chunk", chunk.Index, "output", string(output), "error", err)

	args = []string{
		"-nostdin",
		"-ss", formatFFmpegSeconds(chunk.Start),
		"-i", sourcePath,
		"-t", formatFFmpegSeconds(chunk.Duration()),
		"-vn",
		"-map", "0:a:0",
		"-y",
		outputPath,
	}

	if output, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to extract audio chunk %d: %w (%s)", chunk.Index, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func formatFFmpegSeconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}
