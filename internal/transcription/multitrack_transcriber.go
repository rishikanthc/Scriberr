package transcription

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// Note: TrackCursor removed - using simpler segment-based approach

// MultiTrackTranscriber handles transcription of multi-track audio jobs
type MultiTrackTranscriber struct {
	unifiedProcessor *UnifiedJobProcessor
	db               *gorm.DB
	// Track active temporary jobs for termination support
	activeTrackJobs map[string][]string // main job ID -> list of track job IDs
	trackJobsMutex  sync.RWMutex
}

// NewMultiTrackTranscriber creates a new multi-track transcriber
func NewMultiTrackTranscriber(unifiedProcessor *UnifiedJobProcessor) *MultiTrackTranscriber {
	return &MultiTrackTranscriber{
		unifiedProcessor: unifiedProcessor,
		db:               database.DB,
		activeTrackJobs:  make(map[string][]string),
	}
}

// TrackTranscript represents a transcript for a single track with metadata
type TrackTranscript struct {
	FileName string                       `json:"file_name"`
	Speaker  string                       `json:"speaker"`
	Offset   float64                      `json:"offset"`
	Result   *interfaces.TranscriptResult `json:"result"`
}

// ProcessMultiTrackTranscription processes a multi-track transcription job
func (mt *MultiTrackTranscriber) ProcessMultiTrackTranscription(ctx context.Context, jobID string) error {
	overallStartTime := time.Now()

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

	// Initialize tracking for this multi-track job
	mt.trackJobsMutex.Lock()
	mt.activeTrackJobs[jobID] = make([]string, 0, len(job.MultiTrackFiles))
	mt.trackJobsMutex.Unlock()

	// Clear any existing individual transcripts to ensure clean progress tracking from 0/N
	if err := mt.db.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Update("individual_transcripts", nil).Error; err != nil {
		logger.Warn("Failed to clear individual transcripts at start", "job_id", jobID, "error", err)
	}

	// Ensure cleanup of tracking on exit
	defer func() {
		mt.trackJobsMutex.Lock()
		delete(mt.activeTrackJobs, jobID)
		mt.trackJobsMutex.Unlock()
	}()

	// Process each track individually and track timing
	trackTranscripts := make([]TrackTranscript, 0, len(job.MultiTrackFiles))
	individualTranscripts := make(map[string]string)
	trackTimings := make([]models.MultiTrackTiming, 0, len(job.MultiTrackFiles))

	for i, trackFile := range job.MultiTrackFiles {
		trackStartTime := time.Now()

		logger.Info("Processing track",
			"job_id", jobID,
			"track_index", i+1,
			"track_name", trackFile.FileName,
			"offset", trackFile.Offset)

		// Create a temporary job for this individual track
		trackResult, err := mt.transcribeIndividualTrack(ctx, &job, &trackFile)
		trackEndTime := time.Now()
		trackDuration := trackEndTime.Sub(trackStartTime).Milliseconds()

		if err != nil {
			return fmt.Errorf("failed to transcribe track %s: %w", trackFile.FileName, err)
		}

		// Store timing data for this track
		trackTiming := models.MultiTrackTiming{
			TrackName: trackFile.FileName,
			StartTime: trackStartTime,
			EndTime:   trackEndTime,
			Duration:  trackDuration,
		}
		trackTimings = append(trackTimings, trackTiming)

		logger.Info("Completed track transcription",
			"job_id", jobID,
			"track_name", trackFile.FileName,
			"duration_ms", trackDuration)

		// Store individual transcript
		trackTranscriptJSON, err := json.Marshal(trackResult)
		if err != nil {
			return fmt.Errorf("failed to serialize track transcript: %w", err)
		}
		individualTranscripts[trackFile.FileName] = string(trackTranscriptJSON)

		// Save current progress to database (so API can show real-time progress)
		individualTranscriptsJSON, err := json.Marshal(individualTranscripts)
		if err != nil {
			logger.Warn("Failed to serialize individual transcripts for progress update", "error", err)
		} else {
			individualTranscriptsStr := string(individualTranscriptsJSON)
			if err := mt.db.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Update("individual_transcripts", &individualTranscriptsStr).Error; err != nil {
				logger.Warn("Failed to update individual transcripts progress", "job_id", jobID, "error", err)
			}
		}

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

	// Merge all track transcripts with timing
	mergeStartTime := time.Now()
	logger.Info("Merging track transcripts", "job_id", jobID, "tracks_count", len(trackTranscripts))

	mergedTranscript, err := mt.mergeTrackTranscripts(trackTranscripts)
	mergeEndTime := time.Now()
	mergeDuration := mergeEndTime.Sub(mergeStartTime).Milliseconds()

	if err != nil {
		return fmt.Errorf("failed to merge track transcripts: %w", err)
	}

	logger.Info("Completed transcript merge",
		"job_id", jobID,
		"merge_duration_ms", mergeDuration)

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

	// Create speaker mappings for multi-track transcription (so speakers can be renamed in UI)
	if err := mt.createSpeakerMappings(jobID, trackTranscripts); err != nil {
		logger.Warn("Failed to create speaker mappings", "job_id", jobID, "error", err)
		// Don't fail the entire job for speaker mapping issues, just log the warning
	}

	// Save results to database
	updates := map[string]interface{}{
		"transcript":             &mergedTranscriptStr,
		"individual_transcripts": &individualTranscriptsStr,
		"status":                 models.StatusCompleted,
	}

	if err := mt.db.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to save transcription results: %w", err)
	}

	// Create execution record with timing data for multi-track job
	overallEndTime := time.Now()
	overallDuration := overallEndTime.Sub(overallStartTime).Milliseconds()

	if err := mt.createMultiTrackExecutionRecord(jobID, overallStartTime, overallEndTime, overallDuration,
		trackTimings, mergeStartTime, mergeEndTime, mergeDuration, job.Parameters); err != nil {
		logger.Warn("Failed to create execution record", "job_id", jobID, "error", err)
		// Don't fail the job for execution record issues, just log the warning
	}

	logger.Info("Multi-track transcription completed successfully",
		"job_id", jobID,
		"merged_segments", len(mergedTranscript.Segments),
		"total_duration_ms", overallDuration)

	return nil
}

// transcribeIndividualTrack transcribes a single track file using the direct transcription method
func (mt *MultiTrackTranscriber) transcribeIndividualTrack(ctx context.Context, job *models.TranscriptionJob, trackFile *models.MultiTrackFile) (*interfaces.TranscriptResult, error) {
	// Create a proper copy of parameters for this track (disable diarization, enable word timestamps)
	trackParams := job.Parameters

	// Ensure essential fields are properly set for individual track processing
	trackParams.Diarize = false             // Never diarize individual tracks
	trackParams.IsMultiTrackEnabled = false // Individual tracks are not multi-track jobs
	trackParams.ReturnCharAlignments = true // Enable word-level timestamps for better merging

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

	// Create a temporary database job for unified processing
	logger.Info("Transcribing individual track using unified service",
		"track_name", trackFile.FileName,
		"model_family", trackParams.ModelFamily,
		"file_path", trackFile.FilePath)

	// Create temporary job ID for this track with unique suffix to avoid conflicts
	uniqueBytes := make([]byte, 4)
	_, _ = rand.Read(uniqueBytes)
	uniqueID := hex.EncodeToString(uniqueBytes)
	trackJobID := fmt.Sprintf("track_%s_%s_%s", job.ID, trackFile.FileName, uniqueID)

	// Add this track job to the active list for termination support
	mt.trackJobsMutex.Lock()
	if trackJobs, exists := mt.activeTrackJobs[job.ID]; exists {
		mt.activeTrackJobs[job.ID] = append(trackJobs, trackJobID)
	}
	mt.trackJobsMutex.Unlock()

	// Create temporary job for unified processing
	// Use StatusProcessing to prevent the main queue scanner from picking it up
	tempJob := models.TranscriptionJob{
		ID:         trackJobID,
		AudioPath:  trackFile.FilePath,
		Parameters: trackParams,
		Status:     models.StatusProcessing, // Prevent queue scanner from picking this up
	}

	// Save temporary job to database for processing
	if err := mt.db.Create(&tempJob).Error; err != nil {
		return nil, fmt.Errorf("failed to create temp database entry for track: %w", err)
	}

	// Process with unified service - check for cancellation first
	select {
	case <-ctx.Done():
		mt.cleanupTempJob(trackJobID)
		return nil, fmt.Errorf("track transcription was cancelled")
	default:
	}

	err := mt.unifiedProcessor.ProcessJob(ctx, trackJobID)
	if err != nil {
		// Clean up temp job and associated records
		mt.cleanupTempJob(trackJobID)
		return nil, fmt.Errorf("failed to transcribe track file %s: %w", trackFile.FilePath, err)
	}

	// Load the processed result
	var processedJob models.TranscriptionJob
	if err := mt.db.Where("id = ?", trackJobID).First(&processedJob).Error; err != nil {
		mt.cleanupTempJob(trackJobID)
		return nil, fmt.Errorf("failed to load processed track result: %w", err)
	}

	// Parse the transcript result
	var result *interfaces.TranscriptResult
	if processedJob.Transcript != nil {
		result = &interfaces.TranscriptResult{}
		if err := json.Unmarshal([]byte(*processedJob.Transcript), result); err != nil {
			mt.cleanupTempJob(trackJobID)
			return nil, fmt.Errorf("failed to parse track transcript: %w", err)
		}
	} else {
		mt.cleanupTempJob(trackJobID)
		return nil, fmt.Errorf("no transcript found for track")
	}

	// Clean up temporary database entry and associated records
	mt.cleanupTempJob(trackJobID)

	logger.Info("Successfully transcribed track",
		"track_name", trackFile.FileName,
		"model_family", trackParams.ModelFamily,
		"word_count", len(result.WordSegments),
		"segment_count", len(result.Segments))

	return result, nil
}

// cleanupTempJob properly deletes a temporary job and all associated records
func (mt *MultiTrackTranscriber) cleanupTempJob(jobID string) {
	// Delete execution records first (foreign key constraint)
	if err := mt.db.Where("transcription_job_id = ?", jobID).Delete(&models.TranscriptionJobExecution{}).Error; err != nil {
		logger.Warn("Failed to delete temp job execution records", "job_id", jobID, "error", err)
	}

	// Delete speaker mappings if any
	if err := mt.db.Where("transcription_job_id = ?", jobID).Delete(&models.SpeakerMapping{}).Error; err != nil {
		logger.Warn("Failed to delete temp job speaker mappings", "job_id", jobID, "error", err)
	}

	// Delete the job itself
	if err := mt.db.Delete(&models.TranscriptionJob{}, "id = ?", jobID).Error; err != nil {
		logger.Warn("Failed to delete temp job", "job_id", jobID, "error", err)
	}

	logger.Info("Cleaned up temporary job", "job_id", jobID)
}

// TerminateMultiTrackJob terminates a multi-track job and all its active track jobs
func (mt *MultiTrackTranscriber) TerminateMultiTrackJob(jobID string) error {
	mt.trackJobsMutex.RLock()
	trackJobs, exists := mt.activeTrackJobs[jobID]
	if !exists {
		mt.trackJobsMutex.RUnlock()
		return fmt.Errorf("multi-track job %s not found or not active", jobID)
	}

	// Make a copy of the track job IDs to avoid holding the lock during cleanup
	trackJobsCopy := make([]string, len(trackJobs))
	copy(trackJobsCopy, trackJobs)
	mt.trackJobsMutex.RUnlock()

	logger.Info("Terminating multi-track job", "job_id", jobID, "track_count", len(trackJobsCopy))

	// Clean up all temporary track jobs
	for _, trackJobID := range trackJobsCopy {
		logger.Info("Cleaning up track job", "main_job_id", jobID, "track_job_id", trackJobID)
		mt.cleanupTempJob(trackJobID)
	}

	// Remove from active tracking
	mt.trackJobsMutex.Lock()
	delete(mt.activeTrackJobs, jobID)
	mt.trackJobsMutex.Unlock()

	// Update main job status to failed
	if err := mt.db.Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":        models.StatusFailed,
			"error_message": "Job was terminated by user",
		}).Error; err != nil {
		logger.Warn("Failed to update main job status after termination", "job_id", jobID, "error", err)
	}

	logger.Info("Multi-track job terminated successfully", "job_id", jobID)
	return nil
}

// GetActiveTrackJobs returns the list of active track jobs for a multi-track job
func (mt *MultiTrackTranscriber) GetActiveTrackJobs(jobID string) []string {
	mt.trackJobsMutex.RLock()
	defer mt.trackJobsMutex.RUnlock()

	if trackJobs, exists := mt.activeTrackJobs[jobID]; exists {
		result := make([]string, len(trackJobs))
		copy(result, trackJobs)
		return result
	}

	return nil
}

// mergeTrackTranscripts merges multiple track transcripts using sort-and-group algorithm
func (mt *MultiTrackTranscriber) mergeTrackTranscripts(trackTranscripts []TrackTranscript) (*interfaces.TranscriptResult, error) {
	if len(trackTranscripts) == 0 {
		return nil, fmt.Errorf("no track transcripts to merge")
	}

	logger.Info("Starting sort-and-group transcript merging", "track_count", len(trackTranscripts))

	// Phase 1: Collect ALL words from all tracks with offset adjustment
	var allWords []interfaces.Word

	for _, trackTranscript := range trackTranscripts {
		if trackTranscript.Result == nil {
			continue
		}

		speaker := trackTranscript.Speaker
		offset := trackTranscript.Offset

		logger.Info("Collecting words from track",
			"speaker", speaker,
			"offset", offset,
			"word_count", len(trackTranscript.Result.WordSegments))

		// Collect words with offset adjustment and speaker assignment
		for _, word := range trackTranscript.Result.WordSegments {
			adjustedWord := interfaces.Word{
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
		// Include only text without speaker labels (speaker info preserved in segments)
		mergedText.WriteString(strings.TrimSpace(turn.Text))
	}

	mergedResult := &interfaces.TranscriptResult{
		Segments:     speakerTurns,
		WordSegments: allWords,
		Language:     language,
		Text:         mergedText.String(),
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
	base = cases.Title(language.English).String(base)
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
func (mt *MultiTrackTranscriber) createSpeakerTurns(sortedWords []interfaces.Word) []interfaces.Segment {
	if len(sortedWords) == 0 {
		return []interfaces.Segment{}
	}

	var turns []interfaces.Segment
	var currentTurnWords []interfaces.Word
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
			currentTurnWords = []interfaces.Word{word}
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
func (mt *MultiTrackTranscriber) createTurnFromWords(words []interfaces.Word, speaker *string) interfaces.Segment {
	if len(words) == 0 {
		return interfaces.Segment{}
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

	turn := interfaces.Segment{
		Start:   start,
		End:     end,
		Text:    textBuilder.String(),
		Speaker: speaker,
	}

	return turn
}

// logIndividualTranscript provides detailed logging of individual track transcripts
func (mt *MultiTrackTranscriber) logIndividualTranscript(fileName string, result *interfaces.TranscriptResult, offset float64) {
	speaker := getBaseFileName(fileName)

	logger.Info("=== INDIVIDUAL TRANSCRIPT DETAILS ===",
		"file", fileName,
		"speaker", speaker,
		"offset", offset,
		"language", result.Language,
		"total_segments", len(result.Segments),
		"total_words", len(result.WordSegments))

	// Log segment-level data
	logger.Info("--- SEGMENTS (Original Timestamps) ---", "file", fileName)
	for i, segment := range result.Segments {
		logger.Info("Segment",
			"file", fileName,
			"index", i+1,
			"start", segment.Start,
			"end", segment.End,
			"duration", segment.End-segment.Start,
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
			"duration", segment.End-segment.Start,
			"text", segment.Text)
	}

	// Log word-level data (original timestamps)
	logger.Info("--- WORDS (Original Timestamps) ---", "file", fileName)
	for i, word := range result.WordSegments {
		logger.Debug("Word",
			"file", fileName,
			"index", i+1,
			"word", word.Word,
			"start", word.Start,
			"end", word.End,
			"duration", word.End-word.Start,
			"score", word.Score)
	}

	// Log word-level data with offset applied
	logger.Info("--- WORDS (With Offset Applied) ---", "file", fileName, "offset", offset)
	for i, word := range result.WordSegments {
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
			"duration", word.End-word.Start,
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

// createSpeakerMappings creates speaker mappings for multi-track transcription
// This allows users to rename speakers in the UI
func (mt *MultiTrackTranscriber) createSpeakerMappings(jobID string, trackTranscripts []TrackTranscript) error {
	// Clear any existing speaker mappings for this job
	if err := mt.db.Where("transcription_job_id = ?", jobID).Delete(&models.SpeakerMapping{}).Error; err != nil {
		return fmt.Errorf("failed to clear existing speaker mappings: %w", err)
	}

	// Create speaker mappings for each track
	for _, trackTranscript := range trackTranscripts {
		speakerMapping := &models.SpeakerMapping{
			TranscriptionJobID: jobID,
			OriginalSpeaker:    trackTranscript.Speaker,
			CustomName:         trackTranscript.Speaker, // Default to using the filename as the speaker name
		}

		if err := mt.db.Create(speakerMapping).Error; err != nil {
			return fmt.Errorf("failed to create speaker mapping for %s: %w", trackTranscript.Speaker, err)
		}

		logger.Info("Created speaker mapping",
			"job_id", jobID,
			"original_speaker", trackTranscript.Speaker,
			"custom_name", trackTranscript.Speaker)
	}

	logger.Info("Successfully created speaker mappings for multi-track job",
		"job_id", jobID,
		"speaker_count", len(trackTranscripts))

	return nil
}

// createMultiTrackExecutionRecord creates execution record with multi-track timing data
func (mt *MultiTrackTranscriber) createMultiTrackExecutionRecord(
	jobID string,
	startTime, endTime time.Time,
	totalDuration int64,
	trackTimings []models.MultiTrackTiming,
	mergeStartTime, mergeEndTime time.Time,
	mergeDuration int64,
	parameters models.WhisperXParams) error {

	// Serialize track timings to JSON
	trackTimingsJSON, err := json.Marshal(trackTimings)
	if err != nil {
		return fmt.Errorf("failed to serialize track timings: %w", err)
	}
	trackTimingsStr := string(trackTimingsJSON)

	// Create execution record
	execution := &models.TranscriptionJobExecution{
		TranscriptionJobID: jobID,
		StartedAt:          startTime,
		CompletedAt:        &endTime,
		ProcessingDuration: &totalDuration,

		// Multi-track specific data
		MultiTrackTimings: &trackTimingsStr,
		MergeStartTime:    &mergeStartTime,
		MergeEndTime:      &mergeEndTime,
		MergeDuration:     &mergeDuration,

		ActualParameters: parameters,
		Status:           models.StatusCompleted,
	}

	if err := mt.db.Create(execution).Error; err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	logger.Info("Created multi-track execution record",
		"job_id", jobID,
		"total_duration_ms", totalDuration,
		"merge_duration_ms", mergeDuration,
		"tracks_count", len(trackTimings))

	return nil
}
