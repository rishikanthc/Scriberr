package api

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type YouTubeImporter interface {
	Import(ctx context.Context, job youtubeImportJob) (youtubeImportResult, error)
}

type youtubeImportJob struct {
	URL        string
	OutputPath string
	Title      string
}

type youtubeImportResult struct {
	Filename   string
	MimeType   string
	DurationMs *int64
}

type ytDLPImporter struct{}

func (ytDLPImporter) Import(ctx context.Context, job youtubeImportJob) (youtubeImportResult, error) {
	if err := os.MkdirAll(filepath.Dir(job.OutputPath), 0755); err != nil {
		return youtubeImportResult{}, fmt.Errorf("prepare output directory: %w", err)
	}

	outputTemplate := strings.TrimSuffix(job.OutputPath, filepath.Ext(job.OutputPath)) + ".%(ext)s"
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--output", outputTemplate,
		"--",
		job.URL,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return youtubeImportResult{}, fmt.Errorf("yt-dlp failed: %s", sanitizeCommandOutput(string(output)))
	}

	matches, err := filepath.Glob(strings.TrimSuffix(job.OutputPath, filepath.Ext(job.OutputPath)) + ".*")
	if err != nil || len(matches) == 0 {
		return youtubeImportResult{}, fmt.Errorf("downloaded file was not created")
	}
	downloadedPath := matches[0]
	if downloadedPath != job.OutputPath {
		if err := os.Rename(downloadedPath, job.OutputPath); err != nil {
			return youtubeImportResult{}, fmt.Errorf("finalize downloaded file: %w", err)
		}
	}
	return youtubeImportResult{
		Filename: safeFilename(job.Title) + ".mp3",
		MimeType: "audio/mpeg",
	}, nil
}

func sanitizeCommandOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "download failed"
	}
	lines := strings.Split(output, "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return "download failed"
	}
	if len(last) > 160 {
		last = last[:160]
	}
	return last
}
