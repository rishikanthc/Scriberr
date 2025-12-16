package processing

import (
	"context"
	"fmt"
	"path/filepath"

	"scriberr/internal/audio"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/pkg/logger"

	"gorm.io/gorm"
)

// MultiTrackProcessor handles processing of multi-track audio jobs
type MultiTrackProcessor struct {
	aupParser   *audio.AupParser
	audioMerger *audio.AudioMerger
	db          *gorm.DB
	jobRepo     repository.JobRepository
}

// NewMultiTrackProcessor creates a new multi-track processor
func NewMultiTrackProcessor(db *gorm.DB, jobRepo repository.JobRepository) *MultiTrackProcessor {
	return &MultiTrackProcessor{
		aupParser:   audio.NewAupParser(),
		audioMerger: audio.NewAudioMerger(),
		db:          db,
		jobRepo:     jobRepo,
	}
}

// ProcessMultiTrackJob processes a multi-track job by parsing the .aup file and merging audio
func (p *MultiTrackProcessor) ProcessMultiTrackJob(ctx context.Context, jobID string) error {
	// Get the job from database
	var job models.TranscriptionJob
	if err := p.db.Preload("MultiTrackFiles").Where("id = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}

	// Verify it's a multi-track job
	if !job.IsMultiTrack || job.AupFilePath == nil {
		return fmt.Errorf("job %s is not a multi-track job", jobID)
	}

	logger.Info("Starting multi-track processing", "job_id", jobID)

	// Update status to processing
	if err := p.updateMergeStatus(jobID, "processing", nil); err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	// Parse the .aup file to get track information
	aupTracks, err := p.aupParser.ParseAupFile(*job.AupFilePath)
	if err != nil {
		errMsg := err.Error()
		_ = p.updateMergeStatus(jobID, "failed", &errMsg)
		return fmt.Errorf("failed to parse AUP file: %w", err)
	}

	logger.Info("Parsed AUP file", "job_id", jobID, "tracks_count", len(aupTracks))

	// Update MultiTrackFile records with offset information
	if err := p.updateTrackOffsets(jobID, aupTracks); err != nil {
		errMsg := err.Error()
		_ = p.updateMergeStatus(jobID, "failed", &errMsg)
		return fmt.Errorf("failed to update track offsets: %w", err)
	}

	// Get updated track files from database
	var trackFiles []models.MultiTrackFile
	if err := p.db.Where("transcription_job_id = ?", jobID).Order("track_index").Find(&trackFiles).Error; err != nil {
		errMsg := err.Error()
		_ = p.updateMergeStatus(jobID, "failed", &errMsg)
		return fmt.Errorf("failed to get track files: %w", err)
	}

	// Convert to TrackInfo for merger
	trackInfos := make([]audio.TrackInfo, len(trackFiles))
	for i, tf := range trackFiles {
		trackInfos[i] = audio.TrackInfo{
			FilePath: tf.FilePath,
			Offset:   tf.Offset,
			Gain:     tf.Gain,
			Pan:      tf.Pan,
			Mute:     tf.Mute,
		}
	}

	// Define output path
	outputPath := filepath.Join(*job.MultiTrackFolder, "merged.mp3")

	// Merge the audio tracks
	progressCallback := func(progress audio.MergeProgress) {
		logger.Info("Merge progress", "job_id", jobID, "stage", progress.Stage, "progress", progress.Progress)
		// In a production system, you might want to store intermediate progress
		// or emit progress events via websockets/SSE
	}

	if err := p.audioMerger.MergeTracksWithOffsets(ctx, trackInfos, outputPath, progressCallback); err != nil {
		errMsg := err.Error()
		_ = p.updateMergeStatus(jobID, "failed", &errMsg)
		return fmt.Errorf("failed to merge audio tracks: %w", err)
	}

	// Update job with merged audio path
	updates := map[string]interface{}{
		"merged_audio_path": outputPath,
		"merge_status":      "completed",
		"merge_error":       nil,
		"audio_path":        outputPath, // Update main audio path to point to merged file
	}

	if err := p.db.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		errMsg := err.Error()
		_ = p.updateMergeStatus(jobID, "failed", &errMsg)
		return fmt.Errorf("failed to update job with merged path: %w", err)
	}

	logger.Info("Successfully completed multi-track processing", "job_id", jobID, "output_path", outputPath)
	return nil
}

// updateMergeStatus updates the merge status of a job
func (p *MultiTrackProcessor) updateMergeStatus(jobID, status string, errorMsg *string) error {
	updates := map[string]interface{}{
		"merge_status": status,
	}

	if errorMsg != nil {
		updates["merge_error"] = *errorMsg
	} else {
		updates["merge_error"] = nil
	}

	return p.db.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(updates).Error
}

// updateTrackOffsets updates the MultiTrackFile records with information from .aup file
func (p *MultiTrackProcessor) updateTrackOffsets(jobID string, aupTracks []audio.AupTrack) error {
	// Get existing track files
	var trackFiles []models.MultiTrackFile
	if err := p.db.Where("transcription_job_id = ?", jobID).Find(&trackFiles).Error; err != nil {
		return fmt.Errorf("failed to get existing track files: %w", err)
	}

	// Create a map of filename to aup track for quick lookup
	aupTrackMap := make(map[string]audio.AupTrack)
	for _, track := range aupTracks {
		// Use base filename for matching
		baseFilename := filepath.Base(track.Filename)
		aupTrackMap[baseFilename] = track
	}

	// Update each track file with offset information
	for _, trackFile := range trackFiles {
		// Try to find matching aup track
		originalFilename := trackFile.FileName + filepath.Ext(trackFile.FilePath)
		if aupTrack, exists := aupTrackMap[originalFilename]; exists {
			updates := map[string]interface{}{
				"offset": aupTrack.Offset,
				"gain":   aupTrack.Gain,
				"pan":    aupTrack.Pan,
				"mute":   aupTrack.Mute == 1, // Convert int to bool
			}

			if err := p.db.Model(&models.MultiTrackFile{}).Where("id = ?", trackFile.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update track file %d: %w", trackFile.ID, err)
			}

			logger.Info("Updated track with AUP info",
				"track_id", trackFile.ID,
				"filename", originalFilename,
				"offset", aupTrack.Offset,
				"gain", aupTrack.Gain,
				"pan", aupTrack.Pan,
				"mute", aupTrack.Mute == 1)
		} else {
			logger.Warn("No matching AUP track found for file", "filename", originalFilename, "track_id", trackFile.ID)
			// Set default values for tracks not found in AUP
			updates := map[string]interface{}{
				"offset": 0.0,
				"gain":   1.0,
				"pan":    0.0,
				"mute":   false,
			}
			if err := p.db.Model(&models.MultiTrackFile{}).Where("id = ?", trackFile.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to set default values for track file %d: %w", trackFile.ID, err)
			}
		}
	}

	return nil
}

// GetMergeStatus returns the current merge status of a job
func (p *MultiTrackProcessor) GetMergeStatus(jobID string) (string, *string, error) {
	var job models.TranscriptionJob
	if err := p.db.Select("merge_status", "merge_error").Where("id = ?", jobID).First(&job).Error; err != nil {
		return "", nil, fmt.Errorf("failed to get job: %w", err)
	}
	return job.MergeStatus, job.MergeError, nil
}
