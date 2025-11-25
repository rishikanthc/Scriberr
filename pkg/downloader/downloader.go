package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DownloadFile downloads a file from a URL to a destination path with progress tracking
func DownloadFile(ctx context.Context, url, dest string) error {
	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file
	tempDest := dest + ".tmp"
	out, err := os.Create(tempDest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create progress tracker
	size := resp.ContentLength
	tracker := &progressTracker{
		Total:    size,
		Filename: filepath.Base(dest),
		LastLog:  time.Now(),
	}

	// Copy with progress
	_, err = io.Copy(out, io.TeeReader(resp.Body, tracker))
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Close file before renaming
	out.Close()

	// Rename temp file to final destination
	if err := os.Rename(tempDest, dest); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	// Print final newline
	fmt.Println()

	return nil
}

type progressTracker struct {
	Total       int64
	Current     int64
	Filename    string
	LastLog     time.Time
	LastPercent int
}

func (pt *progressTracker) Write(p []byte) (int, error) {
	n := len(p)
	pt.Current += int64(n)
	pt.printProgress()
	return n, nil
}

func (pt *progressTracker) printProgress() {
	// Calculate percentage
	percent := int(float64(pt.Current) / float64(pt.Total) * 100)

	// Update only if percentage changed significantly or enough time passed
	if percent != pt.LastPercent && (percent%5 == 0 || time.Since(pt.LastLog) > 1*time.Second) {
		pt.LastPercent = percent
		pt.LastLog = time.Now()

		// Clear line and print progress
		// \r moves cursor to start of line
		// \033[K clears the line
		fmt.Printf("\r\033[KDownloading %s: %d%% (%s / %s)",
			pt.Filename,
			percent,
			formatBytes(pt.Current),
			formatBytes(pt.Total))
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
