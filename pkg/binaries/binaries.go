package binaries

import "os"

func resolve(envKey, fallback string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return fallback
}

// UV returns the configured uv executable path.
func UV() string {
	return resolve("SCRIBERR_UV_BIN", "uv")
}

// FFmpeg returns the configured ffmpeg executable path.
func FFmpeg() string {
	return resolve("SCRIBERR_FFMPEG_BIN", "ffmpeg")
}

// FFprobe returns the configured ffprobe executable path.
func FFprobe() string {
	return resolve("SCRIBERR_FFPROBE_BIN", "ffprobe")
}

// YtDLP returns the configured yt-dlp executable path.
func YtDLP() string {
	return resolve("SCRIBERR_YTDLP_BIN", "yt-dlp")
}
