# Project Overview

Scriberr is a self-hosted offline transcription app for converting audio into text. It uses WhisperX with open-source Whisper models for accurate transcription. Built with React (frontend) and Go (backend), packaged as a single binary. The app allows recording or uploading audio, transcribing it, and interacting with transcripts via summarization or chat using preferred LLM providers.

## Folder Structure

- `cmd/server/`: Main Go server entry point (main.go)
- `internal/`: Internal packages including api, auth, config, database, dropzone, llm, models, queue, transcription, web
- `pkg/`: Public packages like middleware
- `web/frontend/`: React frontend application
- `tests/`: All Go test files (_test.go)
- `docs/`: Documentation and API docs
- `assets/`: Static assets like logos
- `screenshots/`: Screenshots for documentation
- `api-docs/`: API documentation files
- Root files: build scripts, Docker files, go.mod, README.md, etc.

## Libraries and Frameworks

- Backend: Go 1.24.1 with Gin web framework, GORM ORM, SQLite database, JWT for auth
- Frontend: React (JavaScript/TypeScript)
- Transcription: WhisperX (Python-based with open-source Whisper models)
- Testing: testify for Go unit tests
- Other: fsnotify for file watching, godotenv for env vars, swag for API docs

## Coding Standards

- Go: Use `go fmt ./...` for formatting and `go vet ./...` for static analysis
- Frontend: Run `npm run lint` in `web/frontend/` directory
- Naming: Follow Go conventions (camelCase for unexported, PascalCase for exported)
- Tests: All test files must be in `tests/` folder with `_test.go` suffix
- Error handling: Use proper error returns in Go, avoid panics in production code
- Imports: Group standard library, third-party, and internal imports separately

## Build and Run Instructions

- Development backend: `go run cmd/server/main.go`
- Development frontend: `cd web/frontend && npm ci && npm run dev`
- Full build: `./build.sh` (embeds frontend in Go binary)
- Docker: Use `docker-compose.yml` or specific GPU variants (cuda/rocm)
- Environment: Copy `.env.example` to `.env` for configuration

## Testing

- Run all tests: `go test ./tests/...`
- Test files are located in `tests/` directory
- Use testify for assertions and mocking
- Ensure tests cover API handlers, services, and utilities

## Deployment

- Single binary deployment (backend + embedded frontend)
- Supports Docker with volume mounts for data persistence
- GPU acceleration available for Nvidia (CUDA) and AMD (ROCm)
- Default port 8080, configurable via environment

## Key Features to Understand

- Transcription with word-level timing
- Speaker diarization using pyannote models
- Transcript reader with audio follow-along
- LLM integration for chat and summarization
- REST API with JWT and API key authentication
- YouTube video transcription support
- Multiple output formats (JSON, SRT, TXT)
