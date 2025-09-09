package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TrackInfo represents information needed for merging a track
type TrackInfo struct {
	FilePath string
	Offset   float64 // in seconds
	Gain     float64
	Pan      float64
	Mute     bool
}

// MergeProgress represents the progress of an audio merge operation
type MergeProgress struct {
	Stage       string  // "starting", "processing", "completed", "failed"
	Progress    float64 // 0-100 percentage
	ErrorMsg    string
	OutputPath  string
}

// AudioMerger handles merging multiple audio tracks with timing offsets
type AudioMerger struct {
	ffmpegPath string
}

// NewAudioMerger creates a new audio merger instance
func NewAudioMerger() *AudioMerger {
	return &AudioMerger{
		ffmpegPath: "ffmpeg", // Assumes ffmpeg is in PATH
	}
}

// NewAudioMergerWithPath creates an audio merger with custom ffmpeg path
func NewAudioMergerWithPath(ffmpegPath string) *AudioMerger {
	return &AudioMerger{
		ffmpegPath: ffmpegPath,
	}
}

// MergeTracksWithOffsets merges audio tracks using their offset information
func (m *AudioMerger) MergeTracksWithOffsets(ctx context.Context, tracks []TrackInfo, outputPath string, progressCallback func(MergeProgress)) error {
	if len(tracks) == 0 {
		return fmt.Errorf("no tracks provided for merging")
	}

	// Report starting
	if progressCallback != nil {
		progressCallback(MergeProgress{Stage: "starting", Progress: 0})
	}

	// Validate all input files exist
	for i, track := range tracks {
		if _, err := os.Stat(track.FilePath); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", track.FilePath)
		}
		// Skip muted tracks
		if track.Mute {
			continue
		}
		if progressCallback != nil {
			progressCallback(MergeProgress{
				Stage:    "validating",
				Progress: float64(i+1) / float64(len(tracks)) * 20, // 0-20% for validation
			})
		}
	}

	// Filter out muted tracks
	activeTracks := make([]TrackInfo, 0, len(tracks))
	for _, track := range tracks {
		if !track.Mute {
			activeTracks = append(activeTracks, track)
		}
	}

	if len(activeTracks) == 0 {
		return fmt.Errorf("no active (non-muted) tracks to merge")
	}

	// Build ffmpeg command
	cmd := m.buildFFmpegCommand(activeTracks, outputPath)

	if progressCallback != nil {
		progressCallback(MergeProgress{Stage: "processing", Progress: 25})
	}

	// Execute ffmpeg command
	if err := m.executeFFmpegCommand(ctx, cmd, progressCallback); err != nil {
		if progressCallback != nil {
			progressCallback(MergeProgress{Stage: "failed", Progress: 0, ErrorMsg: err.Error()})
		}
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		if progressCallback != nil {
			progressCallback(MergeProgress{Stage: "failed", Progress: 0, ErrorMsg: "output file was not created"})
		}
		return fmt.Errorf("output file was not created: %s", outputPath)
	}

	if progressCallback != nil {
		progressCallback(MergeProgress{Stage: "completed", Progress: 100, OutputPath: outputPath})
	}

	return nil
}

// buildFFmpegCommand constructs the ffmpeg command for merging tracks
func (m *AudioMerger) buildFFmpegCommand(tracks []TrackInfo, outputPath string) *exec.Cmd {
	args := []string{
		"-y", // Overwrite output file if it exists
	}

	// Add input files
	for _, track := range tracks {
		args = append(args, "-i", track.FilePath)
	}

	// Build filter complex
	var filterParts []string
	var mixInputs []string

	for i, track := range tracks {
		// Create adelay filter for each track
		delayFilter := fmt.Sprintf("[%d:a]adelay=%.3fs:all=1", i, track.Offset)
		
		// Apply gain if not default (1.0)
		if track.Gain != 1.0 && track.Gain != 0.0 {
			delayFilter += fmt.Sprintf(",volume=%.3f", track.Gain)
		}
		
		// Apply pan if not center (0.0)
		if track.Pan != 0.0 {
			// Convert pan value (-1.0 to 1.0) to ffmpeg pan format
			panValue := (track.Pan + 1.0) / 2.0 // Convert to 0-1 range
			delayFilter += fmt.Sprintf(",pan=stereo|c0<%g*c0+%g*c1|c1<%g*c0+%g*c1",
				1-panValue, panValue, panValue, 1-panValue)
		}
		
		trackLabel := fmt.Sprintf("[a%d]", i)
		filterParts = append(filterParts, delayFilter+trackLabel)
		mixInputs = append(mixInputs, trackLabel)
	}

	// Add amix filter to combine all tracks
	amixFilter := fmt.Sprintf("%samix=inputs=%d:duration=longest:normalize=0[aout]",
		strings.Join(mixInputs, ""), len(tracks))
	filterParts = append(filterParts, amixFilter)

	// Combine all filter parts
	filterComplex := strings.Join(filterParts, ";")

	args = append(args,
		"-filter_complex", filterComplex,
		"-map", "[aout]",
		"-c:a", "libmp3lame", // Use MP3 for output (smaller file size)
		"-b:a", "192k",       // 192 kbps bitrate
		outputPath,
	)

	return exec.Command(m.ffmpegPath, args...)
}

// executeFFmpegCommand runs the ffmpeg command with progress tracking
func (m *AudioMerger) executeFFmpegCommand(ctx context.Context, cmd *exec.Cmd, progressCallback func(MergeProgress)) error {
	// Create pipes for stderr to capture ffmpeg output
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Monitor stderr for progress (ffmpeg outputs progress to stderr)
	go func() {
		buf := make([]byte, 1024)
		progressReported := 25.0 // Start from 25% (after validation and setup)
		
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				output := string(buf[:n])
				
				// Simple progress estimation based on ffmpeg output
				// In a production system, you'd parse the actual progress info
				if strings.Contains(output, "time=") && progressCallback != nil {
					progressReported += 2.0 // Increment progress
					if progressReported > 95.0 {
						progressReported = 95.0 // Cap at 95% until completion
					}
					progressCallback(MergeProgress{
						Stage:    "processing", 
						Progress: progressReported,
					})
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for the command to complete or context to be cancelled
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill ffmpeg process: %w", err)
		}
		return fmt.Errorf("merge operation cancelled")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg process failed: %w", err)
		}
		return nil
	}
}

// ValidateFFmpeg checks if ffmpeg is available and working
func (m *AudioMerger) ValidateFFmpeg() error {
	cmd := exec.Command(m.ffmpegPath, "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg not found or not working: %w", err)
	}
	return nil
}