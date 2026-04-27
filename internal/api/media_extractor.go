package api

import (
	"context"
	"os/exec"
)

type MediaExtractor interface {
	ExtractAudio(ctx context.Context, inputPath, outputPath string) error
}

type ffmpegMediaExtractor struct{}

func (ffmpegMediaExtractor) ExtractAudio(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)
	return cmd.Run()
}
