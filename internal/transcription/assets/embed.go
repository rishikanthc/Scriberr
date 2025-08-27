package assets

import "embed"

// Embed the Python helper files used to bootstrap the WhisperX environment.
// These files are written to the configured WHISPERX_ENV directory at runtime.

//go:embed pyproject.toml diarize_transcript.py
var FS embed.FS

