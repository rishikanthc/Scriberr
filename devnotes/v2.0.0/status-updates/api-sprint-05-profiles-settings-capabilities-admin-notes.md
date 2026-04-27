# Sprint 5 Notes

## Scope

Implemented the API foundation endpoints for profiles, settings, model capabilities, and queue stats.

## Decisions

- Profiles are backed by `models.TranscriptionProfile` and exposed with `profile_` public IDs.
- Profile options map to the existing `WhisperXParams` model fields.
- Settings are backed by the existing single-user settings JSON on `models.User`.
- `local_only` is always true in the API response for this first pass.
- Model capabilities return a conservative local Whisper placeholder.
- Queue stats are derived from transcription job status counts and exclude uploaded file rows.
- Global events remain a clear `501` placeholder until an API-safe broadcaster integration is introduced.

## Verification

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestProfile|TestSettings|TestCapabilities'`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware`
- `git diff --check`
