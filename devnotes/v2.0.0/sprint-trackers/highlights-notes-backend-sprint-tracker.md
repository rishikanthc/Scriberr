# Sprint Tracker: Highlights and Notes Backend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/highlights-notes-backend-sprint-plan.md`.

Status: completed through Sprint 5. Highlights and notes backend plan is complete.

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

Status: completed

Completed tasks:

- Add request/response types separate from GORM models.
- Register routes under `/api/v1/transcriptions/{id}/annotations`.
- Implement list/create/get/update/delete handlers.
- Add bounded pagination, `kind`, and `updated_after` filters.
- Add route contract tests, error-envelope tests, ownership tests, and public response shape tests.
- Add OpenAPI documentation for the new routes and request schemas.

Artifacts:

- `internal/api/annotation_handlers.go`
- `internal/api/annotation_handlers_test.go`
- `internal/api/router.go`
- `internal/api/middleware.go`
- `internal/api/route_contract_test.go`
- `internal/annotations/service.go`
- `internal/repository/implementations.go`
- `internal/repository/annotation_repository_test.go`
- `docs/api/openapi.json`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestAnnotation|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestAPIDocsContainOnlyCanonicalRoutes|TestTranscriptionCreateListGetPatchCancelDelete|TestTranscriptionValidationTranscriptRetryAndAudioAlias'`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/annotations ./internal/repository ./internal/database`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/annotations ./internal/repository ./internal/database ./internal/api`

## Sprint 4: Events and Transcript Integrity

Status: completed

Completed tasks:

- Publish small `annotation.created`, `annotation.updated`, and `annotation.deleted` events through the existing SSE broker.
- Include public annotation/transcription IDs, annotation kind, and anchor status in annotation events.
- Add transcript anchor hash verification for annotations with `anchor.text_hash`.
- Mark annotations as `stale` when the stored hash no longer matches the current transcript anchor text.
- Refresh annotation anchor statuses after transcription completion through the worker completion observer path.
- Keep annotations on transcript replacement/recompletion; stale status is the explicit integrity signal.
- Add tests for SSE payload shape, active/stale hash behavior, and completion-time status refresh.

Artifacts:

- `internal/annotations/service.go`
- `internal/annotations/service_test.go`
- `internal/models/annotation.go`
- `internal/repository/implementations.go`
- `internal/repository/annotation_repository_test.go`
- `internal/api/router.go`
- `internal/api/annotation_handlers.go`
- `internal/api/annotation_handlers_test.go`
- `internal/api/events_test.go`
- `internal/transcription/worker/service.go`
- `cmd/server/main.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/annotations ./internal/repository ./internal/transcription/worker`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestSSEReceivesAnnotationEvents|TestAnnotation|TestCanonicalRouteRegistration|TestAPIDocsContainOnlyCanonicalRoutes'`
- `GOCACHE=/tmp/scriberr-go-cache go test ./cmd/server`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/annotations ./internal/repository ./internal/transcription/worker ./internal/database ./internal/api ./cmd/server`

## Sprint 5: Future Multi-User Hardening

Status: completed

Completed tasks:

- Added a `TranscriptionAccessPolicy` seam to the annotation service.
- Kept the default policy as current transcript ownership via `JobRepository.FindTranscriptionByIDForUser`.
- Added service tests for future shared-transcript access where two users may access the same transcript but only see their own annotations.
- Added repository tests for two users with different transcriptions and annotations.
- Added forged update/status/delete regression coverage to prove repository writes remain scoped by `user_id` and `transcription_id`.
- Added API tests for two authenticated users with different transcriptions and annotations.
- Reviewed ownership fields and kept the existing schema: annotation `user_id` remains independent of transcription ownership for future sharing support.
- Did not add audit fields in this sprint because no product requirement currently needs `created_by_user_id` or `updated_by_user_id`.

Artifacts:

- `internal/annotations/service.go`
- `internal/annotations/service_test.go`
- `internal/repository/annotation_repository_test.go`
- `internal/api/annotation_handlers_test.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/annotations ./internal/repository`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestAnnotations|TestAnnotation'`
