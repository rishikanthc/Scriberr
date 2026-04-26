package orchestrator

import (
	"encoding/json"
	"fmt"
	"sort"

	"scriberr/internal/transcription/engineprovider"
)

type CanonicalTranscript struct {
	Text     string             `json:"text"`
	Language string             `json:"language,omitempty"`
	Segments []CanonicalSegment `json:"segments"`
	Words    []CanonicalWord    `json:"words"`
	Engine   TranscriptEngine   `json:"engine,omitempty"`
}

type CanonicalSegment struct {
	ID      string  `json:"id"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker,omitempty"`
	Text    string  `json:"text"`
}

type CanonicalWord struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Word    string  `json:"word"`
	Speaker string  `json:"speaker,omitempty"`
}

type TranscriptEngine struct {
	Provider           string `json:"provider,omitempty"`
	TranscriptionModel string `json:"transcription_model,omitempty"`
	DiarizationModel   string `json:"diarization_model,omitempty"`
}

func BuildCanonicalTranscript(transcription *engineprovider.TranscriptionResult, diarization *engineprovider.DiarizationResult) (*CanonicalTranscript, error) {
	if transcription == nil {
		return nil, fmt.Errorf("transcription result is required")
	}
	words := make([]CanonicalWord, 0, len(transcription.Words))
	for _, word := range transcription.Words {
		words = append(words, CanonicalWord{
			Start: word.Start,
			End:   word.End,
			Word:  word.Word,
		})
	}

	segments := make([]CanonicalSegment, 0, len(transcription.Segments))
	for i, segment := range transcription.Segments {
		id := segment.ID
		if id == "" {
			id = segmentID(i)
		}
		segments = append(segments, CanonicalSegment{
			ID:    id,
			Start: segment.Start,
			End:   segment.End,
			Text:  segment.Text,
		})
	}
	if len(segments) == 0 && len(words) > 0 {
		segments = []CanonicalSegment{{
			ID:    segmentID(0),
			Start: words[0].Start,
			End:   words[len(words)-1].End,
			Text:  transcription.Text,
		}}
	}

	if diarization != nil && len(diarization.Segments) > 0 {
		assignSpeakers(words, segments, diarization.Segments)
	}

	return &CanonicalTranscript{
		Text:     transcription.Text,
		Language: transcription.Language,
		Segments: segments,
		Words:    words,
		Engine: TranscriptEngine{
			Provider:           transcription.EngineID,
			TranscriptionModel: transcription.ModelID,
			DiarizationModel:   diarizationModelID(diarization),
		},
	}, nil
}

func ParseStoredTranscript(value string) (*CanonicalTranscript, error) {
	var transcript CanonicalTranscript
	if err := json.Unmarshal([]byte(value), &transcript); err != nil {
		return &CanonicalTranscript{
			Text:     value,
			Segments: []CanonicalSegment{},
			Words:    []CanonicalWord{},
		}, nil
	}
	if transcript.Segments == nil {
		transcript.Segments = []CanonicalSegment{}
	}
	if transcript.Words == nil {
		transcript.Words = []CanonicalWord{}
	}
	return &transcript, nil
}

func assignSpeakers(words []CanonicalWord, segments []CanonicalSegment, diarization []engineprovider.DiarizationSegment) {
	labels := stableSpeakerLabels(diarization)
	for i := range words {
		if speaker := bestOverlapSpeaker(words[i].Start, words[i].End, diarization, labels); speaker != "" {
			words[i].Speaker = speaker
		}
	}
	for i := range segments {
		if speaker := bestOverlapSpeaker(segments[i].Start, segments[i].End, diarization, labels); speaker != "" {
			segments[i].Speaker = speaker
		}
	}
}

func stableSpeakerLabels(segments []engineprovider.DiarizationSegment) map[string]string {
	firstSeen := make(map[string]int, len(segments))
	for i, segment := range segments {
		if segment.Speaker == "" {
			continue
		}
		if _, ok := firstSeen[segment.Speaker]; !ok {
			firstSeen[segment.Speaker] = i
		}
	}
	speakers := make([]string, 0, len(firstSeen))
	for speaker := range firstSeen {
		speakers = append(speakers, speaker)
	}
	sort.SliceStable(speakers, func(i, j int) bool {
		return firstSeen[speakers[i]] < firstSeen[speakers[j]]
	})
	labels := make(map[string]string, len(speakers))
	for i, speaker := range speakers {
		labels[speaker] = fmt.Sprintf("SPEAKER_%02d", i)
	}
	return labels
}

func bestOverlapSpeaker(start, end float64, segments []engineprovider.DiarizationSegment, labels map[string]string) string {
	var bestSpeaker string
	var bestOverlap float64
	for _, segment := range segments {
		if segment.Speaker == "" {
			continue
		}
		overlap := intervalOverlap(start, end, segment.Start, segment.End)
		if overlap > bestOverlap {
			bestOverlap = overlap
			bestSpeaker = segment.Speaker
		}
	}
	if bestSpeaker == "" {
		return ""
	}
	return labels[bestSpeaker]
}

func intervalOverlap(aStart, aEnd, bStart, bEnd float64) float64 {
	start := max(aStart, bStart)
	end := min(aEnd, bEnd)
	if end <= start {
		return 0
	}
	return end - start
}

func segmentID(index int) string {
	return fmt.Sprintf("seg_%06d", index+1)
}

func diarizationModelID(diarization *engineprovider.DiarizationResult) string {
	if diarization == nil {
		return ""
	}
	return diarization.ModelID
}
