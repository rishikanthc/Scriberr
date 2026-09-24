package adapters

import (
	"testing"
	"time"

	"scriberr/internal/transcription/interfaces"
)

func TestOpenAIDurationCap(t *testing.T) {
	tests := []struct {
		model string
		want  time.Duration
	}{
		{model: "gpt-4o-transcribe", want: 1400 * time.Second},
		{model: "gpt-4o-mini-transcribe", want: 1400 * time.Second},
		// whisper-1 only has the request-size limit, so it must stay uncapped.
		{model: "whisper-1", want: 0},
		{model: "some-future-model", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := openAIDurationCap(tt.model); got != tt.want {
				t.Errorf("openAIDurationCap(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

// Audio that fits the cap must not be split, so uncapped models and short files
// keep the single request they always used.
func TestPlanAudioChunksUnderCap(t *testing.T) {
	chunks := planAudioChunks(20*time.Minute, 1390*time.Second, openAIChunkOverlap)

	if len(chunks) != 1 {
		t.Fatalf("planAudioChunks() produced %d chunks, want 1", len(chunks))
	}
	if chunks[0].Start != 0 || chunks[0].End != 20*time.Minute {
		t.Errorf("chunk = [%v, %v], want the whole file [0s, 20m0s]", chunks[0].Start, chunks[0].End)
	}
}

func TestPlanAudioChunksCoversFileWithOverlap(t *testing.T) {
	total := 3600 * time.Second
	maxLength := 1390 * time.Second

	chunks := planAudioChunks(total, maxLength, openAIChunkOverlap)

	if len(chunks) < 3 {
		t.Fatalf("planAudioChunks() produced %d chunks, want at least 3 for a one-hour file", len(chunks))
	}
	if chunks[0].Start != 0 {
		t.Errorf("first chunk starts at %v, want 0", chunks[0].Start)
	}
	if last := chunks[len(chunks)-1]; last.End != total {
		t.Errorf("last chunk ends at %v, want the end of the file %v", last.End, total)
	}

	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("chunk %d carries index %d", i, chunk.Index)
		}
		if chunk.Duration() > maxLength {
			t.Errorf("chunk %d is %v long, over the %v limit", i, chunk.Duration(), maxLength)
		}
		if i == 0 {
			continue
		}
		// Consecutive chunks must overlap, never leave a gap, otherwise speech at
		// the cut point is lost.
		previous := chunks[i-1]
		if gap := chunk.Start - previous.End; gap > 0 {
			t.Errorf("chunk %d starts %v after the previous one ends", i, gap)
		}
		if overlap := previous.End - chunk.Start; overlap != openAIChunkOverlap && chunk.End != total {
			t.Errorf("chunk %d overlaps the previous one by %v, want %v", i, overlap, openAIChunkOverlap)
		}
	}
}

func TestPlanAudioChunksIgnoresUnusableOverlap(t *testing.T) {
	// An overlap at least as long as a chunk would never advance.
	chunks := planAudioChunks(100*time.Second, 30*time.Second, 30*time.Second)

	if len(chunks) != 4 {
		t.Fatalf("planAudioChunks() produced %d chunks, want 4 non-overlapping chunks", len(chunks))
	}
	if chunks[1].Start != 30*time.Second {
		t.Errorf("second chunk starts at %v, want 30s", chunks[1].Start)
	}
}

// Every chunk is transcribed on its own, so its timestamps start at zero and
// have to be shifted onto the timeline of the original recording.
func TestMergeChunkTranscriptsShiftsTimestamps(t *testing.T) {
	chunks := []audioChunk{
		{Index: 0, Start: 0, End: 100 * time.Second},
		{Index: 1, Start: 97 * time.Second, End: 197 * time.Second},
	}
	results := []*interfaces.TranscriptResult{
		{
			Language: "en",
			Segments: []interfaces.TranscriptSegment{
				{Start: 0, End: 10, Text: "first"},
				{Start: 10, End: 60, Text: "second"},
			},
			WordSegments: []interfaces.TranscriptWord{
				{Start: 0, End: 1, Word: "first"},
			},
		},
		{
			Language: "en",
			Segments: []interfaces.TranscriptSegment{
				{Start: 20, End: 40, Text: "third"},
			},
			WordSegments: []interfaces.TranscriptWord{
				{Start: 20, End: 21, Word: "third"},
			},
		},
	}

	merged := mergeChunkTranscripts(chunks, results)

	want := []interfaces.TranscriptSegment{
		{Start: 0, End: 10, Text: "first"},
		{Start: 10, End: 60, Text: "second"},
		{Start: 117, End: 137, Text: "third"},
	}
	if len(merged.Segments) != len(want) {
		t.Fatalf("merged into %d segments, want %d", len(merged.Segments), len(want))
	}
	for i, segment := range merged.Segments {
		if segment != want[i] {
			t.Errorf("segment %d = %+v, want %+v", i, segment, want[i])
		}
	}

	if merged.Text != "first second third" {
		t.Errorf("merged text = %q, want %q", merged.Text, "first second third")
	}
	if merged.Language != "en" {
		t.Errorf("merged language = %q, want %q", merged.Language, "en")
	}
	if len(merged.WordSegments) != 2 || merged.WordSegments[1].Start != 117 {
		t.Errorf("word segments = %+v, want the second one shifted to 117s", merged.WordSegments)
	}
}

// The seconds two chunks share are transcribed twice. Segments that fall on the
// far side of the shared region are taken from the chunk covering them more
// centrally, so no text appears twice.
func TestMergeChunkTranscriptsDropsOverlapDuplicates(t *testing.T) {
	chunks := []audioChunk{
		{Index: 0, Start: 0, End: 100 * time.Second},
		{Index: 1, Start: 96 * time.Second, End: 196 * time.Second},
	}
	results := []*interfaces.TranscriptResult{
		{
			Segments: []interfaces.TranscriptSegment{
				{Start: 80, End: 90, Text: "before the cut"},
				{Start: 96, End: 100, Text: "shared words"},
			},
		},
		{
			Segments: []interfaces.TranscriptSegment{
				// Same speech as the tail of the previous chunk, seen 96s later.
				{Start: 0, End: 4, Text: "shared words"},
				{Start: 4, End: 20, Text: "after the cut"},
			},
		},
	}

	merged := mergeChunkTranscripts(chunks, results)

	if merged.Text != "before the cut shared words after the cut" {
		t.Errorf("merged text = %q, want the overlap kept exactly once", merged.Text)
	}
	if len(merged.Segments) != 3 {
		t.Fatalf("merged into %d segments, want 3", len(merged.Segments))
	}
	for i := 1; i < len(merged.Segments); i++ {
		if merged.Segments[i].Start < merged.Segments[i-1].Start {
			t.Errorf("segment %d starts before its predecessor: %+v", i, merged.Segments)
		}
	}
}

// Models that answer without timestamps (the gpt-4o family returns plain JSON)
// give one segment per chunk, so the overlap can only be removed from the text.
func TestMergeChunkTranscriptsDeduplicatesUntimedChunkText(t *testing.T) {
	chunks := []audioChunk{
		{Index: 0, Start: 0, End: 100 * time.Second},
		{Index: 1, Start: 97 * time.Second, End: 197 * time.Second},
	}
	results := []*interfaces.TranscriptResult{
		{
			Text:     "we start the meeting and then agree on the budget",
			Segments: []interfaces.TranscriptSegment{{Start: 0, End: 100, Text: "we start the meeting and then agree on the budget"}},
		},
		{
			Text:     "Agree on the budget, before moving on.",
			Segments: []interfaces.TranscriptSegment{{Start: 0, End: 100, Text: "Agree on the budget, before moving on."}},
		},
	}

	merged := mergeChunkTranscripts(chunks, results)

	want := "we start the meeting and then agree on the budget before moving on."
	if merged.Text != want {
		t.Errorf("merged text = %q, want %q", merged.Text, want)
	}
	if len(merged.Segments) != 2 {
		t.Fatalf("merged into %d segments, want 2", len(merged.Segments))
	}
	if merged.Segments[1].Start < merged.Segments[0].End {
		t.Errorf("second segment starts at %v, before the first ends at %v",
			merged.Segments[1].Start, merged.Segments[0].End)
	}
}

func TestMergeChunkTranscriptsSingleChunkIsUnchanged(t *testing.T) {
	chunks := []audioChunk{{Index: 0, Start: 0, End: 60 * time.Second}}
	results := []*interfaces.TranscriptResult{
		{
			Language: "nl",
			Segments: []interfaces.TranscriptSegment{{Start: 1, End: 2, Text: "hallo"}},
		},
	}

	merged := mergeChunkTranscripts(chunks, results)

	if len(merged.Segments) != 1 || merged.Segments[0].Start != 1 || merged.Segments[0].End != 2 {
		t.Errorf("segments = %+v, want the original timings untouched", merged.Segments)
	}
	if merged.Text != "hallo" {
		t.Errorf("merged text = %q, want %q", merged.Text, "hallo")
	}
}

func TestTrimRepeatedPrefix(t *testing.T) {
	tests := []struct {
		name string
		prev string
		next string
		want string
	}{
		{
			name: "repeated tail is dropped",
			prev: "and then we agree on the budget",
			next: "on the budget before moving on",
			want: "before moving on",
		},
		{
			name: "punctuation and case are ignored",
			prev: "we agree on the budget",
			next: "Budget, before moving on",
			want: "before moving on",
		},
		{
			name: "unrelated text is left alone",
			prev: "we agree on the budget",
			next: "a completely different sentence",
			want: "a completely different sentence",
		},
		{
			name: "entirely repeated text becomes empty",
			prev: "we agree on the budget",
			next: "on the budget",
			want: "",
		},
		{
			name: "empty input is returned as is",
			prev: "",
			next: "on the budget",
			want: "on the budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimRepeatedPrefix(tt.prev, tt.next, openAIOverlapWordWindow); got != tt.want {
				t.Errorf("trimRepeatedPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}
