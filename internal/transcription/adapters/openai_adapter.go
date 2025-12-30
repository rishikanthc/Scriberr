package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

// OpenAIAdapter implements the TranscriptionAdapter interface for OpenAI API
type OpenAIAdapter struct {
	*BaseAdapter
	apiKey string
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(apiKey string) *OpenAIAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:     "openai_whisper",
		ModelFamily: "openai",
		DisplayName: "OpenAI Whisper API",
		Description: "Cloud-based transcription using OpenAI's Whisper model",
		Version:     "v1",
		SupportedLanguages: []string{
			"af", "ar", "hy", "az", "be", "bs", "bg", "ca", "zh", "hr", "cs", "da", "nl", "en", "et", "fi", "fr", "gl", "de", "el", "he", "hi", "hu", "is", "id", "it", "ja", "kn", "kk", "ko", "lv", "lt", "mk", "ms", "mr", "mi", "ne", "no", "fa", "pl", "pt", "ro", "ru", "sr", "sk", "sl", "es", "sw", "sv", "tl", "ta", "th", "tr", "uk", "ur", "vi", "cy",
		},
		SupportedFormats:  []string{"flac", "mp3", "mp4", "mpeg", "mpga", "m4a", "ogg", "wav", "webm"},
		RequiresGPU:       false,
		MemoryRequirement: 0, // Cloud-based
		Features: map[string]bool{
			"timestamps":         true,  // Verbose JSON response includes segments
			"word_level":         false, // Not supported by standard API yet (unless using verbose_json with timestamp_granularities which is beta)
			"diarization":        false, // Not supported by OpenAI API
			"translation":        true,
			"language_detection": true,
			"vad":                true, // Implicit
		},
		Metadata: map[string]string{
			"provider": "openai",
			"api_url":  "https://api.openai.com/v1/audio/transcriptions",
		},
	}

	schema := []interfaces.ParameterSchema{
		{
			Name:        "api_key",
			Type:        "string",
			Required:    false, // Can be provided in config
			Description: "OpenAI API Key (overrides system default)",
			Group:       "authentication",
		},
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     "whisper-1",
			Options:     []string{"whisper-1"},
			Description: "ID of the model to use",
			Group:       "basic",
		},
		{
			Name:        "language",
			Type:        "string",
			Required:    false,
			Description: "Language of the input audio (ISO-639-1)",
			Group:       "basic",
		},
		{
			Name:        "prompt",
			Type:        "string",
			Required:    false,
			Description: "Optional text to guide the model's style or continue a previous audio segment",
			Group:       "advanced",
		},
		{
			Name:        "temperature",
			Type:        "float",
			Required:    false,
			Default:     0.0,
			Min:         &[]float64{0.0}[0],
			Max:         &[]float64{1.0}[0],
			Description: "Sampling temperature",
			Group:       "quality",
		},
	}

	baseAdapter := NewBaseAdapter("openai_whisper", "", capabilities, schema)

	return &OpenAIAdapter{
		BaseAdapter: baseAdapter,
		apiKey:      apiKey,
	}
}

// GetSupportedModels returns the list of OpenAI models supported
func (a *OpenAIAdapter) GetSupportedModels() []string {
	return []string{"whisper-1"}
}

// PrepareEnvironment is a no-op for cloud adapters
func (a *OpenAIAdapter) PrepareEnvironment(ctx context.Context) error {
	a.initialized = true
	return nil
}

// Transcribe processes audio using OpenAI API
//
//nolint:gocyclo // API interaction involves many steps
func (a *OpenAIAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	a.LogProcessingStart(input, procCtx)
	defer func() {
		a.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

	// Helper to write to job log file
	writeLog := func(format string, args ...interface{}) {
		logPath := filepath.Join(procCtx.OutputDirectory, "transcription.log")
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.Error("Failed to open log file", "path", logPath, "error", err)
			return
		}
		defer f.Close()

		msg := fmt.Sprintf(format, args...)
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(f, "[%s] %s\n", timestamp, msg)
	}

	writeLog("Starting OpenAI transcription for job %s", procCtx.JobID)
	writeLog("Input file: %s", input.FilePath)

	// Validate input
	if err := a.ValidateAudioInput(input); err != nil {
		writeLog("Error: Invalid audio input: %v", err)
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Get API Key
	apiKey := a.apiKey
	if key, ok := params["api_key"].(string); ok && key != "" {
		apiKey = key
	}

	if apiKey == "" {
		writeLog("Error: OpenAI API key is required but not provided")
		return nil, fmt.Errorf("OpenAI API key is required but not provided")
	}

	// Prepare request body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	file, err := os.Open(input.FilePath)
	if err != nil {
		writeLog("Error: Failed to open audio file: %v", err)
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(input.FilePath))
	if err != nil {
		writeLog("Error: Failed to create form file: %v", err)
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		writeLog("Error: Failed to copy file content: %v", err)
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add parameters
	model := a.GetStringParameter(params, "model")
	if model == "" {
		model = "whisper-1"
	}
	writeLog("Model: %s", model)
	_ = writer.WriteField("model", model)

	if strings.HasPrefix(model, "gpt-4o") {
		if strings.Contains(model, "diarize") {
			_ = writer.WriteField("response_format", "diarized_json")
		} else {
			_ = writer.WriteField("response_format", "json")
		}
		// gpt-4o models don't support timestamp_granularities with these formats
	} else {
		_ = writer.WriteField("response_format", "verbose_json")
		// timestamp_granularities is only supported for whisper-1
		if model == "whisper-1" {
			_ = writer.WriteField("timestamp_granularities[]", "word")    // Request word timestamps
			_ = writer.WriteField("timestamp_granularities[]", "segment") // Request segment timestamps
		}
	}

	if lang := a.GetStringParameter(params, "language"); lang != "" {
		writeLog("Language: %s", lang)
		_ = writer.WriteField("language", lang)
	}

	if prompt := a.GetStringParameter(params, "prompt"); prompt != "" {
		writeLog("Prompt provided")
		_ = writer.WriteField("prompt", prompt)
	}

	temp := a.GetFloatParameter(params, "temperature")
	writeLog("Temperature: %.2f", temp)
	_ = writer.WriteField("temperature", fmt.Sprintf("%.2f", temp))

	if err := writer.Close(); err != nil {
		writeLog("Error: Failed to close multipart writer: %v", err)
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	writeLog("Sending request to OpenAI API...")
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		writeLog("Error: Failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute request
	client := &http.Client{
		Timeout: 10 * time.Minute, // Generous timeout for large files
	}
	resp, err := client.Do(req)
	if err != nil {
		writeLog("Error: Request failed: %v", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		writeLog("Error: OpenAI API error (status %d): %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	writeLog("Response received. Parsing...")

	// Parse response
	var openAIResponse struct {
		Task     string  `json:"task"`
		Language string  `json:"language"`
		Duration float64 `json:"duration"`
		Text     string  `json:"text"`
		Segments []struct {
			ID               int     `json:"id"`
			Seek             int     `json:"seek"`
			Start            float64 `json:"start"`
			End              float64 `json:"end"`
			Text             string  `json:"text"`
			Tokens           []int   `json:"tokens"`
			Temperature      float64 `json:"temperature"`
			AvgLogprob       float64 `json:"avg_logprob"`
			CompressionRatio float64 `json:"compression_ratio"`
			NoSpeechProb     float64 `json:"no_speech_prob"`
		} `json:"segments"`
		Words []struct {
			Word  string  `json:"word"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"words"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResponse); err != nil {
		writeLog("Error: Failed to decode response: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	writeLog("Transcription completed successfully. Duration: %.2fs, Words: %d", openAIResponse.Duration, len(openAIResponse.Words))

	// Convert to TranscriptResult
	result := &interfaces.TranscriptResult{
		Language:       openAIResponse.Language,
		Text:           openAIResponse.Text,
		Segments:       make([]interfaces.TranscriptSegment, len(openAIResponse.Segments)),
		WordSegments:   make([]interfaces.TranscriptWord, len(openAIResponse.Words)),
		ProcessingTime: time.Since(startTime),
		ModelUsed:      model,
		Metadata:       a.CreateDefaultMetadata(params),
	}

	if len(openAIResponse.Segments) > 0 {
		for i, seg := range openAIResponse.Segments {
			result.Segments[i] = interfaces.TranscriptSegment{
				Start: seg.Start,
				End:   seg.End,
				Text:  seg.Text,
			}
		}
	} else if openAIResponse.Text != "" {
		// If no segments returned (e.g. standard json format), create one segment with the whole text
		result.Segments = []interfaces.TranscriptSegment{
			{
				Start: 0,
				End:   openAIResponse.Duration,
				Text:  openAIResponse.Text,
			},
		}
	}

	for i, word := range openAIResponse.Words {
		result.WordSegments[i] = interfaces.TranscriptWord{
			Word:  word.Word,
			Start: word.Start,
			End:   word.End,
		}
	}

	return result, nil
}

// GetEstimatedProcessingTime provides OpenAI-specific time estimation
func (a *OpenAIAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Cloud transcription is generally faster, approx 10-20% of audio duration
	audioDuration := input.Duration
	if audioDuration == 0 {
		return 30 * time.Second // Fallback
	}
	return time.Duration(float64(audioDuration) * 0.15)
}
