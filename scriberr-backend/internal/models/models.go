package models

import "time"

// Audio represents the metadata for an audio record in the database.
type Audio struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Transcript string    `json:"transcript,omitempty"`  // Stored as a JSON string
	SpeakerMap string    `json:"speaker_map,omitempty"` // Stored as a JSON string
	Summary    string    `json:"summary,omitempty"`     // Stored as a JSON string
	CreatedAt  time.Time `json:"created_at"`
}

// SummaryTemplate represents a template for generating summaries.
type SummaryTemplate struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Prompt    string    `json:"prompt"`
	CreatedAt time.Time `json:"created_at"`
}

// Job represents a transcription job.
type Job struct {
	ID        string    `json:"id"`
	AudioID   string    `json:"audio_id"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ActiveJob represents a job in progress, including the title of the audio.
type ActiveJob struct {
	ID         string    `json:"id"`
	AudioID    string    `json:"audio_id"`
	AudioTitle string    `json:"audio_title"`
	Status     string    `json:"status"`
	Type       string    `json:"type"` // "transcription" or "summarization"
	CreatedAt  time.Time `json:"created_at"`
}

// TranscriptSegment represents one entry in a parsed transcript file.
type TranscriptSegment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Word represents a single word with timing and confidence score
type Word struct {
	Word    string  `json:"word"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Score   float64 `json:"score"`
	Speaker string  `json:"speaker,omitempty"`
}

// JSONTranscriptSegment represents a segment in the new JSON format
type JSONTranscriptSegment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Words   []Word  `json:"words"`
	Speaker string  `json:"speaker,omitempty"`
}

// JSONTranscript represents the complete transcript in the new JSON format
type JSONTranscript struct {
	Segments     []JSONTranscriptSegment `json:"segments"`
	WordSegments []Word                  `json:"word_segments"`
	Language     string                  `json:"language"`
}
