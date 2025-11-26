# CLAUDE.md - Scriberr Development Guide

## Project Overview

Scriberr is a self-hosted, offline audio transcription application built with Go (backend) and React/TypeScript (frontend). It converts audio to text with features like speaker diarization, transcript summarization, and LLM-based chat.

## Quick Start

```bash
# Backend development
go run cmd/server/main.go

# Frontend development (separate terminal)
cd web/frontend
npm ci
npm run dev

# Full production build
./build.sh
./scriberr
```

## Architecture

### Directory Structure

```
cmd/server/main.go          # Application entry point
internal/
  api/                      # REST API handlers (Gin framework)
    handlers.go             # Main API handlers
    router.go               # Route definitions
  audio/                    # Audio processing (ffmpeg integration)
  auth/                     # JWT authentication
  config/                   # Environment configuration
  database/                 # SQLite with GORM
  llm/                      # Ollama/OpenAI integration
  models/                   # Data models
  queue/                    # Job queue with auto-scaling workers
  transcription/            # Transcription engine
    adapters/               # WhisperX, Parakeet, Canary adapters
    unified_service.go      # Main orchestrator
pkg/
  logger/                   # Structured logging
  middleware/               # HTTP middleware
web/frontend/               # React/TypeScript UI (Vite build)
```

### Key Technologies

- **Backend**: Go 1.24, Gin HTTP framework, GORM ORM, SQLite
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS, Radix UI
- **Transcription**: WhisperX (via UV Python environment), yt-dlp
- **Audio**: ffmpeg for format conversion
- **LLM**: Ollama (local) or OpenAI (cloud)

## Database

SQLite with WAL mode. Models auto-migrate on startup.

Key models:
- `TranscriptionJob` - Core job record with audio path, status, transcript
- `TranscriptionProfile` - Saved configuration presets
- `User`, `APIKey` - Authentication
- `ChatSession`, `ChatMessage` - LLM conversations

## Job Processing

The `queue.TaskQueue` manages transcription jobs:
- Auto-scaling workers (2-6 based on CPU)
- Jobs progress: `uploaded` → `pending` → `processing` → `completed|failed`
- Scanner polls for pending jobs every 10 seconds

## API Patterns

Handlers in `internal/api/handlers.go`:
- Use Gin context for request/response
- Access config via `h.config`
- Use `database.DB` for GORM operations
- Use `logger` package for structured logging

```go
func (h *Handler) ExampleHandler(c *gin.Context) {
    var req RequestType
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Process...
    c.JSON(http.StatusOK, response)
}
```

## Python Integration

WhisperX and yt-dlp run via UV package manager:
```go
cmd := exec.Command(h.config.UVPath, "run", "--native-tls",
    "--project", h.config.WhisperXEnv,
    "python", "-m", "module_name", args...)
```

## Environment Variables

```env
PORT=8080
HOST=localhost
DATABASE_PATH=data/scriberr.db
UPLOAD_DIR=data/uploads
WHISPERX_ENV=data/whisperx-env
UV_PATH=/path/to/uv          # Auto-detected if not set
JWT_SECRET=your-secret       # Auto-generated if not set
QUEUE_WORKERS=2              # Fixed worker count (disables auto-scale)
```

## Testing & Linting

```bash
go fmt ./...
go vet ./...
cd web/frontend && npm run lint
```

## Common Tasks

### Adding a new API endpoint

1. Define request/response structs in `handlers.go`
2. Create handler method on `Handler` struct
3. Register route in `router.go` with appropriate middleware
4. Add Swagger annotations for API docs

### Adding a new transcription adapter

1. Create adapter in `internal/transcription/adapters/`
2. Implement `TranscriptionAdapter` interface
3. Register in `internal/transcription/registry/`

### Modifying database schema

1. Update model in `internal/models/`
2. GORM auto-migrates on startup
3. For complex migrations, add to `internal/database/database.go`

## YouTube Download

Existing implementation in `DownloadFromYouTube` handler:
- Validates YouTube URL format
- Uses yt-dlp to extract audio as MP3
- Creates TranscriptionJob record
- File saved to `{uploadDir}/{jobID}.mp3`

## CSV Batch Processing

The CSV batch processor (`internal/csvbatch/`) enables bulk YouTube video transcription:
- Import CSV with YouTube URLs (one per row)
- Sequential processing per rowId
- Downloads video → converts to audio → deletes video → transcribes
- Output: `{rowId}-{videoFilename}.json`

See `/api/v1/csv-batch/*` endpoints for usage.
