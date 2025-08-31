package assets

import "embed"

// Embed the Python project file used to bootstrap the WhisperX environment.
// This file is written to the configured WHISPERX_ENV directory at runtime.

//go:embed pyproject.toml
var FS embed.FS

