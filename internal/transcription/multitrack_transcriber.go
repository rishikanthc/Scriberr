package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/pkg/logger"

	"gorm.io/gorm"
)

// Note: TrackCursor removed - using simpler segment-based approach

// MultiTrackTranscriber handles transcription of multi-track audio jobs
type MultiTrackTranscriber struct {
	whisperX *WhisperXService
	db       *gorm.DB
}

// NewMultiTrackTranscriber creates a new multi-track transcriber
func NewMultiTrackTranscriber(whisperX *WhisperXService) *MultiTrackTranscriber {
	return &MultiTrackTranscriber{
		whisperX: whisperX,
		db:       database.DB,
	}
}

// TrackTranscript represents a transcript for a single track with metadata
type TrackTranscript struct {
	FileName string              `json:"file_name"`
	Speaker  string              `json:"speaker"`
	Offset   float64             `json:"offset"`
	Result   *TranscriptResult   `json:"result"`
}

// ProcessMultiTrackTranscription processes a multi-track transcription job
func (mt *MultiTrackTranscriber) ProcessMultiTrackTranscription(ctx context.Context, jobID string) error {
	// Load the job and track files
	var job models.TranscriptionJob
	if err := mt.db.Preload("MultiTrackFiles").Where("id = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("failed to load job: %w", err)
	}

	if !job.IsMultiTrack {
		return fmt.Errorf("job %s is not a multi-track job", jobID)
	}

	if len(job.MultiTrackFiles) == 0 {
		return fmt.Errorf("no track files found for multi-track job %s", jobID)
	}

	logger.Info("Starting multi-track transcription", 
		"job_id", jobID, 
		"tracks_count", len(job.MultiTrackFiles))

	// Process each track individually
	trackTranscripts := make([]TrackTranscript, 0, len(job.MultiTrackFiles))
	individualTranscripts := make(map[string]string)

	for i, trackFile := range job.MultiTrackFiles {
		logger.Info("Processing track", 
			"job_id", jobID,
			"track_index", i+1,
			"track_name", trackFile.FileName,
			"offset", trackFile.Offset)

		// Create a temporary job for this individual track
		trackResult, err := mt.transcribeIndividualTrack(ctx, &job, &trackFile)
		if err != nil {
			return fmt.Errorf("failed to transcribe track %s: %w", trackFile.FileName, err)
		}

		// Store individual transcript
		trackTranscriptJSON, err := json.Marshal(trackResult)
		if err != nil {
			return fmt.Errorf("failed to serialize track transcript: %w", err)
		}
		individualTranscripts[trackFile.FileName] = string(trackTranscriptJSON)
		
		// Log individual transcript details for debugging
		mt.logIndividualTranscript(trackFile.FileName, trackResult, trackFile.Offset)

		// Create track transcript with metadata
		trackTranscript := TrackTranscript{
			FileName: trackFile.FileName,
			Speaker:  getBaseFileName(trackFile.FileName), // Use filename as speaker name
			Offset:   trackFile.Offset,
			Result:   trackResult,
		}

		trackTranscripts = append(trackTranscripts, trackTranscript)
	}

	// Merge all track transcripts
	logger.Info("Merging track transcripts", "job_id", jobID, "tracks_count", len(trackTranscripts))
	
	mergedTranscript, err := mt.mergeTrackTranscripts(trackTranscripts)
	if err != nil {
		return fmt.Errorf("failed to merge track transcripts: %w", err)
	}

	// Serialize merged transcript to JSON
	mergedTranscriptJSON, err := json.Marshal(mergedTranscript)
	if err != nil {
		return fmt.Errorf("failed to serialize merged transcript: %w", err)
	}
	mergedTranscriptStr := string(mergedTranscriptJSON)

	// Serialize individual transcripts to JSON
	individualTranscriptsJSON, err := json.Marshal(individualTranscripts)
	if err != nil {
		return fmt.Errorf("failed to serialize individual transcripts: %w", err)
	}
	individualTranscriptsStr := string(individualTranscriptsJSON)

	// Save results to database
	updates := map[string]interface{}{
		"transcript":              &mergedTranscriptStr,
		"individual_transcripts": &individualTranscriptsStr,
		"status":                 models.StatusCompleted,
	}

	if err := mt.db.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to save transcription results: %w", err)
	}

	logger.Info("Multi-track transcription completed successfully", 
		"job_id", jobID,
		"merged_segments", len(mergedTranscript.Segments))

	return nil
}

// transcribeIndividualTrack transcribes a single track file using the direct transcription method
func (mt *MultiTrackTranscriber) transcribeIndividualTrack(ctx context.Context, job *models.TranscriptionJob, trackFile *models.MultiTrackFile) (*TranscriptResult, error) {
	// Create a proper copy of parameters for this track (disable diarization, enable word timestamps)
	trackParams := job.Parameters
	
	// Ensure essential fields are properly set for individual track processing
	trackParams.Diarize = false                    // Never diarize individual tracks
	trackParams.IsMultiTrackEnabled = false       // Individual tracks are not multi-track jobs
	trackParams.ReturnCharAlignments = true       // Enable word-level timestamps for better merging
	
	// Ensure we have sensible defaults for core parameters to avoid command issues
	if trackParams.Model == "" {
		trackParams.Model = "small"
	}
	if trackParams.Device == "" {
		trackParams.Device = "cpu"
	}
	if trackParams.ComputeType == "" {
		trackParams.ComputeType = "float32"
	}
	if trackParams.Task == "" {
		trackParams.Task = "transcribe"
	}
	if trackParams.OutputFormat == "" {
		trackParams.OutputFormat = "all"
	}
	if trackParams.InterpolateMethod == "" {
		trackParams.InterpolateMethod = "nearest"
	}
	if trackParams.VadMethod == "" {
		trackParams.VadMethod = "silero"
	}
	if trackParams.DiarizeModel == "" {
		trackParams.DiarizeModel = "pyannote/speaker-diarization-3.1"
	}
	
	// Use the new direct transcription method - no temporary database jobs needed!
	result, err := mt.whisperX.TranscribeAudioFile(ctx, trackFile.FilePath, trackParams)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe track file %s: %w", trackFile.FilePath, err)
	}

	return result, nil
}

// mergeTrackTranscripts merges multiple track transcripts using sort-and-group algorithm
func (mt *MultiTrackTranscriber) mergeTrackTranscripts(trackTranscripts []TrackTranscript) (*TranscriptResult, error) {
	if len(trackTranscripts) == 0 {
		return nil, fmt.Errorf("no track transcripts to merge")
	}

	logger.Info("Starting sort-and-group transcript merging", "track_count", len(trackTranscripts))

	// Phase 1: Collect ALL words from all tracks with offset adjustment
	var allWords []Word
	
	for _, trackTranscript := range trackTranscripts {
		if trackTranscript.Result == nil {
			continue
		}
		
		speaker := trackTranscript.Speaker
		offset := trackTranscript.Offset
		
		logger.Info("Collecting words from track", 
			"speaker", speaker,
			"offset", offset,
			"word_count", len(trackTranscript.Result.Word))
		
		// Collect words with offset adjustment and speaker assignment
		for _, word := range trackTranscript.Result.Word {
			adjustedWord := Word{
				Start:   word.Start + offset,
				End:     word.End + offset,
				Word:    word.Word,
				Score:   word.Score,
				Speaker: &speaker,
			}
			allWords = append(allWords, adjustedWord)
		}
	}

	if len(allWords) == 0 {
		return nil, fmt.Errorf("no words found in any track transcript")
	}

	logger.Info("Collected all words", "total_words", len(allWords))

	// Phase 2: Sort ALL words chronologically by start time
	sort.Slice(allWords, func(i, j int) bool {
		return allWords[i].Start < allWords[j].Start
	})

	logger.Info("Sorted all words chronologically")
	
	// Log the chronologically sorted words for debugging
	logger.Info("=== CHRONOLOGICALLY SORTED WORDS ===")
	for i, word := range allWords {
		speakerName := "unknown"
		if word.Speaker != nil {
			speakerName = *word.Speaker
		}
		logger.Info("Sorted Word", 
			"index", i+1,
			"start", word.Start,
			"end", word.End,
			"word", word.Word,
			"speaker", speakerName,
			"score", word.Score)
	}
	logger.Info("=== END SORTED WORDS ===", "total_words", len(allWords))

	// Phase 3: Group consecutive words from same speaker into turns
	speakerTurns := mt.createSpeakerTurns(allWords)

	// Determine language from first available result
	language := "unknown"
	for _, trackTranscript := range trackTranscripts {
		if trackTranscript.Result != nil && trackTranscript.Result.Language != "" {
			language = trackTranscript.Result.Language
			break
		}
	}

	// Generate merged text from speaker turns
	var mergedText strings.Builder
	for i, turn := range speakerTurns {
		if i > 0 {
			mergedText.WriteString(" ")
		}
		// Include speaker label in merged text
		if turn.Speaker != nil && *turn.Speaker != "" {
			mergedText.WriteString(fmt.Sprintf("[%s]: %s", *turn.Speaker, strings.TrimSpace(turn.Text)))
		} else {
			mergedText.WriteString(strings.TrimSpace(turn.Text))
		}
	}

	mergedResult := &TranscriptResult{
		Segments: speakerTurns,
		Word:     allWords,
		Language: language,
		Text:     mergedText.String(),
	}

	logger.Info("Sort-and-group merging completed successfully", 
		"input_words", len(allWords),
		"output_turns", len(speakerTurns),
		"text_length", len(mergedResult.Text))

	return mergedResult, nil
}

// getBaseFileName extracts the filename without extension to use as speaker name
func getBaseFileName(filename string) string {
	base := filepath.Base(filename)
	// Remove file extension
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	// Clean up the name (replace underscores/hyphens with spaces, capitalize)
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.Title(base)
	return base
}

// GetIndividualTranscripts returns the individual track transcripts for a job
func (mt *MultiTrackTranscriber) GetIndividualTranscripts(jobID string) (map[string]string, error) {
	var job models.TranscriptionJob
	if err := mt.db.Select("individual_transcripts").Where("id = ?", jobID).First(&job).Error; err != nil {
		return nil, fmt.Errorf("failed to load job: %w", err)
	}

	if job.IndividualTranscripts == nil {
		return make(map[string]string), nil
	}

	var transcripts map[string]string
	if err := json.Unmarshal([]byte(*job.IndividualTranscripts), &transcripts); err != nil {
		return nil, fmt.Errorf("failed to parse individual transcripts: %w", err)
	}

	return transcripts, nil
}

// createSpeakerTurns groups consecutive words from the same speaker into turns
func (mt *MultiTrackTranscriber) createSpeakerTurns(sortedWords []Word) []Segment {
	if len(sortedWords) == 0 {
		return []Segment{}
	}
	
	var turns []Segment
	var currentTurnWords []Word
	var currentSpeaker *string
	
	logger.Info("Creating speaker turns from sorted words", "total_words", len(sortedWords))
	
	for i, word := range sortedWords {
		// Check if speaker changed (or first word)
		speakerChanged := currentSpeaker == nil || word.Speaker == nil || *word.Speaker != *currentSpeaker
		
		if speakerChanged {
			// Finalize current turn if it has words
			if len(currentTurnWords) > 0 {
				turn := mt.createTurnFromWords(currentTurnWords, currentSpeaker)
				turns = append(turns, turn)
				
				logger.Debug("Finalized speaker turn", 
					"speaker", *currentSpeaker,
					"word_count", len(currentTurnWords),
					"start", turn.Start,
					"end", turn.End,
					"text", turn.Text)
			}
			
			// Start new turn
			currentTurnWords = []Word{word}
			currentSpeaker = word.Speaker
			
			if word.Speaker != nil {
				logger.Debug("Started new speaker turn", 
					"speaker", *word.Speaker,
					"word_index", i,
					"start_time", word.Start,
					"word", word.Word)
			}
		} else {
			// Same speaker - continue current turn
			currentTurnWords = append(currentTurnWords, word)
		}
	}
	
	// Don't forget the last turn
	if len(currentTurnWords) > 0 {
		turn := mt.createTurnFromWords(currentTurnWords, currentSpeaker)
		turns = append(turns, turn)
		
		if currentSpeaker != nil {
			logger.Debug("Finalized final speaker turn", 
				"speaker", *currentSpeaker,
				"word_count", len(currentTurnWords),
				"text", turn.Text)
		}
	}
	
	logger.Info("Created speaker turns", 
		"input_words", len(sortedWords),
		"output_turns", len(turns))
	
	return turns
}

// createTurnFromWords creates a single segment/turn from a group of consecutive words
func (mt *MultiTrackTranscriber) createTurnFromWords(words []Word, speaker *string) Segment {
	if len(words) == 0 {
		return Segment{}
	}
	
	// Calculate turn timing from first and last word
	start := words[0].Start
	end := words[len(words)-1].End
	
	// Join all words into turn text
	var textBuilder strings.Builder
	for i, word := range words {
		if i > 0 {
			textBuilder.WriteString(" ")
		}
		textBuilder.WriteString(strings.TrimSpace(word.Word))
	}
	
	turn := Segment{
		Start:   start,
		End:     end,
		Text:    textBuilder.String(),
		Speaker: speaker,
	}
	
	return turn
}

// logIndividualTranscript provides detailed logging of individual track transcripts
func (mt *MultiTrackTranscriber) logIndividualTranscript(fileName string, result *TranscriptResult, offset float64) {
	speaker := getBaseFileName(fileName)
	
	logger.Info("=== INDIVIDUAL TRANSCRIPT DETAILS ===", 
		"file", fileName,
		"speaker", speaker,
		"offset", offset,
		"language", result.Language,
		"total_segments", len(result.Segments),
		"total_words", len(result.Word))
	
	// Log segment-level data
	logger.Info("--- SEGMENTS (Original Timestamps) ---", "file", fileName)
	for i, segment := range result.Segments {
		logger.Info("Segment", 
			"file", fileName,
			"index", i+1,
			"start", segment.Start,
			"end", segment.End,
			"duration", segment.End - segment.Start,
			"text", segment.Text)
	}
	
	// Log segment-level data with offset applied
	logger.Info("--- SEGMENTS (With Offset Applied) ---", "file", fileName, "offset", offset)
	for i, segment := range result.Segments {
		adjustedStart := segment.Start + offset
		adjustedEnd := segment.End + offset
		logger.Info("Adjusted Segment", 
			"file", fileName,
			"index", i+1,
			"original_start", segment.Start,
			"adjusted_start", adjustedStart,
			"original_end", segment.End,
			"adjusted_end", adjustedEnd,
			"duration", segment.End - segment.Start,
			"text", segment.Text)
	}
	
	// Log word-level data (original timestamps)
	logger.Info("--- WORDS (Original Timestamps) ---", "file", fileName)
	for i, word := range result.Word {
		logger.Debug("Word", 
			"file", fileName,
			"index", i+1,
			"word", word.Word,
			"start", word.Start,
			"end", word.End,
			"duration", word.End - word.Start,
			"score", word.Score)
	}
	
	// Log word-level data with offset applied
	logger.Info("--- WORDS (With Offset Applied) ---", "file", fileName, "offset", offset)
	for i, word := range result.Word {
		adjustedStart := word.Start + offset
		adjustedEnd := word.End + offset
		logger.Info("Adjusted Word", 
			"file", fileName,
			"index", i+1,
			"word", word.Word,
			"original_start", word.Start,
			"adjusted_start", adjustedStart,
			"original_end", word.End,
			"adjusted_end", adjustedEnd,
			"duration", word.End - word.Start,
			"score", word.Score,
			"speaker", speaker)
	}
	
	// Log full text for this track
	logger.Info("--- FULL TEXT ---", 
		"file", fileName,
		"speaker", speaker,
		"text", result.Text)
	
	logger.Info("=== END INDIVIDUAL TRANSCRIPT ===", "file", fileName)
}