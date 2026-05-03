# Backend Service Boundary Sprint 06 Automation Notes

Sprint 6 is complete. This checkpoint adds the post-file automation boundary for newly ready files.

## Completed

- Added `internal/automation.Service` as the file-ready automation entry point.
- Routed `files.Service.FileReady` through the automation observer before publishing `file.ready`.
- Implemented auto-transcription for newly ready audio when `auto_transcription_enabled` is true and a default profile exists.
- Added runtime no-op behavior for missing default profile, missing file/user records, duplicate transcription requests, and missing small LLM readiness.
- Published `transcription.created` only after the durable transcription row is created.
- Added fake-based automation service tests and an API integration test for auto-transcription on upload.

## Auto-Rename Trigger

Auto-rename is triggered at `summary-completed`, not at file-ready. The title-generation boundary already has summary context and can produce better names than raw file metadata. Sprint 6 gates that existing title generation with `auto_rename_enabled`; if the setting is disabled, summary completion does not rename the file/transcription.

At file-ready, automation only validates small-LLM readiness for runtime no-op behavior. It does not call an LLM before transcript or summary context exists.

## Verification

- `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api ./internal/recording ./internal/mediaimport`
- `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/summarization ./internal/automation ./internal/repository ./cmd/server`
- `git diff --check`
