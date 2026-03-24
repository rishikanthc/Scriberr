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
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

type WhisperAPIAdapter struct {
	*BaseAdapter
	apiURL string // global default from server config
	apiKey string // global default from server config
}

// NewWhisperAPIAdapter creates a new Whisper API adapter; globalURL/globalKey are server
// defaults overridden by per-job params when provided.
func NewWhisperAPIAdapter(globalURL, globalKey string) *WhisperAPIAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:     "whisper_api",
		ModelFamily: "whisper_api",
		DisplayName: "External Whisper API",
		Description: "External Whisper API compatible with OpenAI's /v1/audio/transcriptions format",
		Version:     "1.0",
		SupportedLanguages: []string{
			"en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr", "pl", "ca", "nl",
			"ar", "sv", "it", "id", "hi", "fi", "vi", "he", "uk", "el", "ms", "cs", "ro",
			"da", "hu", "ta", "no", "th", "ur", "hr", "bg", "lt", "la", "mi", "ml", "cy",
			"sk", "te", "fa", "lv", "bn", "sr", "az", "sl", "kn", "et", "mk", "br", "eu",
			"is", "hy", "ne", "mn", "bs", "kk", "sq", "sw", "gl", "mr", "pa", "si", "km",
			"sn", "yo", "so", "af", "oc", "ka", "be", "tg", "sd", "gu", "am", "yi", "lo",
			"uz", "fo", "ht", "ps", "tk", "nn", "mt", "sa", "lb", "my", "bo", "tl", "mg",
			"as", "tt", "haw", "ln", "ha", "ba", "jw", "su", "auto",
		},
		SupportedFormats:  []string{"flac", "mp3", "mp4", "mpeg", "mpga", "m4a", "ogg", "wav", "webm"},
		RequiresGPU:       false,
		MemoryRequirement: 0,
		Features: map[string]bool{
			"timestamps":         true,
			"word_level":         true,
			"diarization":        true,
			"translation":        true,
			"language_detection": true,
			"vad":                true,
		},
		Metadata: map[string]string{
			"provider": "external_whisper_api",
		},
	}

	schema := []interfaces.ParameterSchema{
		{
			Name:        "api_url",
			Type:        "string",
			Required:    true,
			Description: "External Whisper API URL (e.g. http://localhost:8000/v1/audio/transcriptions)",
			Group:       "authentication",
		},
		{
			Name:        "api_key",
			Type:        "string",
			Required:    false,
			Description: "API Key if required by the external API",
			Group:       "authentication",
		},
		{
			Name:        "model",
			Type:        "string",
			Required:    false,
			Default:     "whisper-1",
			Options:     []string{"whisper-1", "large-v3", "large-v2", "large-v1", "medium", "small", "base", "tiny"},
			Description: "Model name/ID to use",
			Group:       "basic",
		},
		{
			Name:        "language",
			Type:        "string",
			Required:    false,
			Description: "Language of the input audio (ISO-639-1)",
			Group:       "basic",
		},
	}

	baseAdapter := NewBaseAdapter("whisper_api", "", capabilities, schema)

	return &WhisperAPIAdapter{
		BaseAdapter: baseAdapter,
		apiURL:      globalURL,
		apiKey:      globalKey,
	}
}

func (a *WhisperAPIAdapter) GetSupportedModels() []string {
	return []string{"whisper-1"}
}

func (a *WhisperAPIAdapter) PrepareEnvironment(ctx context.Context) error {
	a.initialized = true
	return nil
}

func (a *WhisperAPIAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	audioDuration := input.Duration
	if audioDuration == 0 {
		return 30 * time.Second
	}
	return time.Duration(float64(audioDuration) * 0.15)
}

// Transcribe processes audio using the external Whisper API
//
//nolint:gocyclo // API interaction involves many steps
func (a *WhisperAPIAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
	startTime := time.Now()
	a.LogProcessingStart(input, procCtx)
	defer func() {
		a.LogProcessingEnd(procCtx, time.Since(startTime), nil)
	}()

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

	writeLog("Starting external Whisper API transcription for job %s", procCtx.JobID)
	writeLog("Input file: %s", input.FilePath)

	if err := a.ValidateAudioInput(input); err != nil {
		writeLog("Error: Invalid audio input: %v", err)
		return nil, fmt.Errorf("invalid audio input: %w", err)
	}

	// Apply fallback: global config → per-job param → error
	apiUrl := a.apiURL
	if jobURL := a.GetStringParameter(params, "api_url"); jobURL != "" {
		apiUrl = jobURL
	}
	if apiUrl == "" {
		writeLog("Error: api_url is required but not provided (set WHISPER_API_URL or provide api_url in job params)")
		return nil, fmt.Errorf("api_url is required but not provided")
	}

	apiKey := a.apiKey
	if jobKey := a.GetStringParameter(params, "api_key"); jobKey != "" {
		apiKey = jobKey
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

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

	model := a.GetStringParameter(params, "model")
	if model == "" {
		model = "whisper-1"
	}
	writeLog("Model: %s", model)
	_ = writer.WriteField("model", model)

	// Request timestamps
	_ = writer.WriteField("response_format", "verbose_json")
	_ = writer.WriteField("timestamp_granularities[]", "word")
	_ = writer.WriteField("timestamp_granularities[]", "segment")

	if lang := a.GetStringParameter(params, "language"); lang != "" {
		writeLog("Language: %s", lang)
		_ = writer.WriteField("language", lang)
	}

	if err := writer.Close(); err != nil {
		writeLog("Error: Failed to close multipart writer: %v", err)
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	writeLog("Sending request to %s...", apiUrl)
	req, err := http.NewRequestWithContext(ctx, "POST", apiUrl, body)
	if err != nil {
		writeLog("Error: Failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{
		Timeout: 30 * time.Minute, // Generous timeout for large files
	}
	resp, err := client.Do(req)
	if err != nil {
		writeLog("Error: Request failed: %v", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		writeLog("Error: API error (status %d): %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	writeLog("Response received. Parsing...")

	var apiResponse struct {
		Language string  `json:"language"`
		Duration float64 `json:"duration"`
		Text     string  `json:"text"`
		Segments []struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
			Text  string  `json:"text"`
		} `json:"segments"`
		Words []struct {
			Word  string  `json:"word"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"words"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		writeLog("Error: Failed to decode response: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	writeLog("Transcription completed successfully.")

	result := &interfaces.TranscriptResult{
		Language:       apiResponse.Language,
		Text:           apiResponse.Text,
		Segments:       make([]interfaces.TranscriptSegment, len(apiResponse.Segments)),
		WordSegments:   make([]interfaces.TranscriptWord, len(apiResponse.Words)),
		ProcessingTime: time.Since(startTime),
		ModelUsed:      model,
		Metadata:       a.CreateDefaultMetadata(params),
	}

	if len(apiResponse.Segments) > 0 {
		for i, seg := range apiResponse.Segments {
			result.Segments[i] = interfaces.TranscriptSegment{
				Start: seg.Start,
				End:   seg.End,
				Text:  seg.Text,
			}
		}
	} else if apiResponse.Text != "" {
		// Fallback if segments aren't present
		result.Segments = []interfaces.TranscriptSegment{
			{
				Start: 0,
				End:   apiResponse.Duration,
				Text:  apiResponse.Text,
			},
		}
	}

	for i, word := range apiResponse.Words {
		result.WordSegments[i] = interfaces.TranscriptWord{
			Word:  word.Word,
			Start: word.Start,
			End:   word.End,
		}
	}

	return result, nil
}
