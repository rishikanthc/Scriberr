package chat

import (
	"fmt"
	"strings"

	"scriberr/internal/transcription/orchestrator"
)

// PlainTranscriptText converts a stored transcript artifact into chat-safe plaintext.
func PlainTranscriptText(value string) (string, error) {
	transcript, err := orchestrator.ParseStoredTranscript(value)
	if err != nil {
		return "", err
	}
	if len(transcript.Segments) == 0 {
		return strings.TrimSpace(transcript.Text), nil
	}

	speakerLabels := make(map[string]string)
	nextSpeaker := 1
	var lines []string
	var currentSpeaker string
	var currentText []string
	flush := func() {
		if len(currentText) == 0 {
			return
		}
		text := strings.Join(currentText, " ")
		if currentSpeaker == "" {
			lines = append(lines, text)
		} else {
			lines = append(lines, fmt.Sprintf("%s: %s", currentSpeaker, text))
		}
		currentSpeaker = ""
		currentText = nil
	}

	for _, segment := range transcript.Segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		speaker := strings.TrimSpace(segment.Speaker)
		if speaker != "" {
			if _, ok := speakerLabels[speaker]; !ok {
				speakerLabels[speaker] = fmt.Sprintf("Speaker %d", nextSpeaker)
				nextSpeaker++
			}
			speaker = speakerLabels[speaker]
		}
		if len(currentText) > 0 && speaker != currentSpeaker {
			flush()
		}
		currentSpeaker = speaker
		currentText = append(currentText, text)
	}
	flush()
	if len(lines) > 0 {
		return strings.TrimSpace(strings.Join(lines, "\n")), nil
	}
	return strings.TrimSpace(transcript.Text), nil
}
