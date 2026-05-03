# Backend Architecture Review Remediation: Sprint 14 File Metadata Boundary

Status: complete

Commit: `434099c backend: keep file metadata behind service boundary`

## Scope

Sprint 14 removed direct filesystem metadata access from API file response mapping. API DTOs now receive file metadata from `internal/files`, while audio streaming continues to open storage only through service methods that first enforce ownership.

## Changes

- Added `files.Metadata` and `files.MetadataFromJob`.
- Moved MIME type, kind, duration, and size lookup behind the files service boundary.
- Removed `os.Stat`, `os`, and `path/filepath` imports from `internal/api/response_models.go`.
- Updated file list/get/update/import/upload responses to pass metadata into DTO mapping.
- Updated file and transcription audio streaming MIME detection to use `files.MediaType`.
- Added an architecture guard preventing response model filesystem/path metadata access.
- Added tests for missing physical files returning safe metadata without leaking storage paths.

## Verification

- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/files ./internal/api -run 'TestFile|TestResponse|TestProduction|TestBackendDependencyDirection'`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/recording ./internal/mediaimport ./internal/transcription/orchestrator`
- `git diff --check`

## Follow-Up

The second backend architecture review remediation queue is complete. The unrelated broad API-suite issue noted in Sprint 13 remains: `TestRecordingChunkRequestCancellationDoesNotPersist` returns 401 on a pre-canceled request.
