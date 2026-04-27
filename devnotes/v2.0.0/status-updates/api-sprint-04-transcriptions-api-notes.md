# Sprint 4 Transcriptions API Notes

## Completed

- Added transcription API tests before implementation in `internal/api/transcriptions_test.go`.
- Implemented authenticated transcription create/list/get/update/delete routes.
- Implemented cancel and retry command routes.
- Implemented transcript read route with placeholder segment/word arrays.
- Implemented transcription audio alias using the same range streaming behavior as file audio.
- Added validation for missing files, invalid language options, invalid status filters, and invalid IDs.

## Implementation Notes

- The current database has no separate file table, so transcriptions are represented by existing `transcriptions` rows with `source_file_hash` pointing at the source file row ID.
- Public transcription IDs are `tr_{transcription_id}`.
- Actual transcription execution remains deferred; create/retry produce queued records only.
- Logs, execution metadata, and SSE event backends remain explicit `501` placeholders.

## Verification

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.
