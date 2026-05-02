# Sprint Tracker: Tag-Based Audio Organization Backend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/tag-based-audio-organization-backend-sprint-plan.md`.

Status: completed through Sprint 2. Sprint 3 REST API implementation is in progress.

## Sprint 1: Schema and Migration

Status: completed

Completed tasks:

- Added normalized persistence records for tags and tag assignments.
- Registered `audio_tags` and `audio_tag_assignments` in the target schema.
- Bumped the schema version to record the tag schema addition.
- Added active partial unique indexes for per-user normalized tag names and per-transcription assignments.
- Added read-path indexes for tag lookup, assignment listing, and tag-based transcription filtering.
- Added database coverage for fresh schema creation, uniqueness, soft-delete reuse, and hard-delete cascade.

Artifacts:

- `internal/models/tag.go`
- `internal/database/schema.go`
- `internal/database/steps.go`
- `internal/database/database_test.go`

Commit:

- `0e74689` (`feat: add audio tag schema`)

Verification:

- `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database`

## Sprint 2: Repository and Service

Status: completed

Completed tasks:

- Added `repository.TagRepository` with user-scoped tag and assignment persistence methods.
- Added `tags.Service` for create, list, get, update, delete, assignment, replacement, removal, and tag-filter lookup workflows.
- Added public ID parsing for `tag_...` and `tr_...`.
- Added transcription ownership checks through `JobRepository.FindTranscriptionByIDForUser`.
- Added validation for tag names, request limits, public IDs, and hex colors.
- Added duplicate-name conflict handling.
- Added small event publisher interface for API/SSE wiring.
- Added service coverage for tag CRUD, assignment replacement/removal, tag-filter lookup, invalid requests, duplicate names, and event emission.

Artifacts:

- `internal/tags/service.go`
- `internal/tags/service_test.go`
- `internal/repository/implementations.go`

Commit:

- `8e9bbec` (`feat: add audio tag service`)

Verification:

- `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/tags ./internal/repository`

## Sprint 3: REST API

Status: in progress

Completed tasks:

- Started wiring `tags.Service` into the API handler.
- Started registering `/api/v1/tags` and `/api/v1/transcriptions/{id}/tags` routes.
- Started handler implementation for tag CRUD and transcription tag assignment workflows.
- Started extending `GET /api/v1/transcriptions` with tag filters.

Remaining tasks:

- Finish handler compilation and imports.
- Add REST handler tests for tag CRUD, assignment replacement, add/remove, ownership, validation, and list filtering by one or more tags.
- Update route contract tests.
- Run focused API, service, repository, and database verification.
- Commit the completed REST slice atomically.

## Sprint 4: Events and UI Readiness

Status: pending

Remaining tasks:

- Verify `tag.created`, `tag.updated`, `tag.deleted`, and `transcription.tags.updated` event payloads through API/SSE tests.
- Document frontend integration expectations if needed.
