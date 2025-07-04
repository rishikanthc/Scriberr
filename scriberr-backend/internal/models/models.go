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

// ChatSession represents a chat session for a specific audio transcript
type ChatSession struct {
	ID        string    `json:"id"`
	AudioID   string    `json:"audio_id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessage represents a single message in a chat session
type ChatMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatRequest represents a request to send a message in a chat session
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	Model     string `json:"model,omitempty"` // Optional, will use session model if not provided
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
	Model     string `json:"model"`
}

// CreateChatSessionRequest represents a request to create a new chat session
type CreateChatSessionRequest struct {
	AudioID string `json:"audio_id"`
	Title   string `json:"title"`
	Model   string `json:"model"`
}
