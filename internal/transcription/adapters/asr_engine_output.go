package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"scriberr/internal/transcription/interfaces"
)

type engineSegmentRecord struct {
	Start *float64 `json:"start"`
	End   *float64 `json:"end"`
	Text  string   `json:"text"`
}

type engineWordRecord struct {
	Start *float64 `json:"start"`
	End   *float64 `json:"end"`
	Word  string   `json:"word"`
}

func readEngineTranscript(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readEngineSegments(path string) ([]interfaces.TranscriptSegment, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var segments []interfaces.TranscriptSegment
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec engineSegmentRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("invalid segment record: %w", err)
		}
		start := 0.0
		end := 0.0
		if rec.Start != nil {
			start = *rec.Start
		}
		if rec.End != nil {
			end = *rec.End
		}
		segments = append(segments, interfaces.TranscriptSegment{
			Start: start,
			End:   end,
			Text:  rec.Text,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return segments, nil
}

func readEngineWords(path string) ([]interfaces.TranscriptWord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var words []interfaces.TranscriptWord
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec engineWordRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("invalid word record: %w", err)
		}
		start := 0.0
		end := 0.0
		if rec.Start != nil {
			start = *rec.Start
		}
		if rec.End != nil {
			end = *rec.End
		}
		words = append(words, interfaces.TranscriptWord{
			Start: start,
			End:   end,
			Word:  rec.Word,
			Score: 1.0,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return words, nil
}
