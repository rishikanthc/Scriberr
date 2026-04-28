# Sprint Tracker: Highlights and Notes Backend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/highlights-notes-backend-sprint-plan.md`.

Status: completed through Sprint 2. Sprint 3 is next.

## Sprint 1: Schema and Migration

Status: completed

Completed tasks:

- Added `models.TranscriptAnnotation` as the new persistence record for notes and highlights.
- Registered `transcript_annotations` in the target schema.
- Added indexes for per-user transcript listing, per-user kind listing, and time-range lookup.
- Removed the legacy notes backend path instead of migrating old note rows into the new model.
- Added database coverage for fresh schema creation, FK behavior, soft delete, and hard-delete cascade.

Artifacts:

- `internal/models/annotation.go`
- `internal/database/schema.go`
- `internal/database/database_test.go`
- `internal/database/legacy.go`
- `internal/database/migrate.go`
- `internal/database/steps.go`
- `internal/repository/implementations.go`
- `devnotes/v2.0.0/sprint-plans/highlights-notes-backend-sprint-plan.md`

Commit:

- `4e13bb9` (`Add transcript annotation schema`)

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/database`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/database ./internal/repository ./internal/api`

## Sprint 2: Repository and Service

Status: completed

Completed tasks:

- Added `repository.AnnotationRepository` with scoped domain persistence methods.
- Added `annotations.Service` for create, list, get, update, and delete workflows.
- Added public ID parsing for `tr_...` and `ann_...`.
- Added transcript ownership checks through `JobRepository.FindTranscriptionByIDForUser`.
- Added validation for annotation kind, note content, quote, timestamp anchors, word anchors, and character anchors.
- Added a small event publisher interface for later SSE wiring without coupling the service to `internal/api`.
- Added repository and service tests for ownership, invalid IDs, validation, update, soft delete, and event emission.

Artifacts:

- `internal/annotations/service.go`
- `internal/annotations/service_test.go`
- `internal/repository/implementations.go`
- `internal/repository/annotation_repository_test.go`

Commit:

- `16788e0` (`Add annotation repository service`)

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/annotations ./internal/repository`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/annotations ./internal/database ./internal/repository ./internal/api`

## Sprint 3: REST API

Status: next

Planned tasks:

- Add request/response types separate from GORM models.
- Register routes under `/api/v1/transcriptions/{id}/annotations`.
- Implement list/create/get/update/delete handlers.
- Add pagination using the existing list-query helpers where appropriate.
- Add route contract tests and error-envelope tests.
- Add OpenAPI documentation for the new routes.

## Sprint 4: Events and Transcript Integrity

Status: pending

## Sprint 5: Future Multi-User Hardening

Status: pending
