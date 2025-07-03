# WhisperLive Server

This folder contains the real-time transcription server for Scriberr, powered by [WhisperLive](https://github.com/collabora/WhisperLive).

## Quick Start

1. Install dependencies:
   ```sh
   ./install_dependencies.sh
   ```

2. Start the server:
   ```sh
   python3 whisperlive_server.py
   ```

The server will listen for WebSocket connections from the Go backend for live transcription.

## Platform Support

- **macOS ARM64**: Uses minimal dependencies to avoid compatibility issues
- **Other platforms**: Full WhisperLive support with audio recording capabilities

## Files
- `whisperlive_server.py`: Main entrypoint for the WebSocket transcription server
- `requirements.txt`: Full Python dependencies for WhisperLive
- `requirements_minimal.txt`: Minimal dependencies for platforms with compatibility issues
- `install_dependencies.sh`: Platform-specific installation script 