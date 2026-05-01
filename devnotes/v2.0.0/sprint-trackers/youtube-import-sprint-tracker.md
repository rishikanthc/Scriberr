# Sprint Tracker: YouTube Import

This tracker belongs to `devnotes/v2.0.0/sprint-plans/youtube-import-sprint-plan.md`.

Status: planned.

## Sprint 1: Backend Contract and Route Wiring

Status: pending

Planned tasks:

- [ ] Audit existing YouTube scaffolding and remove dead or unreachable paths.
- [ ] Register `POST /api/v1/files:import-youtube`.
- [ ] Keep request and response shapes typed at the HTTP boundary.
- [ ] Validate YouTube hosts and return structured validation errors.
- [ ] Update route contract tests and OpenAPI docs.

Verification:

- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run TestRouteContract`
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run TestOpenAPIContract`

## Sprint 2: Durable Import Service and Repository Boundary

Status: pending

Planned tasks:

- [ ] Add a media import service/use-case for YouTube imports.
- [ ] Add repository methods for create processing import, progress update, ready, and failed states.
- [ ] Publish `file.processing`, `file.ready`, and `file.failed` from state transitions.
- [ ] Prevent local paths and raw command details from leaking through API responses/events.
- [ ] Add repository/service tests for success, failure, and invalid transitions.

Verification:

- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/repository ./internal/api`

## Sprint 3: yt-dlp Provider and Progress Parsing

Status: pending

Planned tasks:

- [ ] Keep `yt-dlp` execution behind a narrow importer interface.
- [ ] Download audio only with deterministic storage paths.
- [ ] Parse progress from `yt-dlp` output when available.
- [ ] Sanitize command errors before persistence or API exposure.
- [ ] Clean up partial files on failure.
- [ ] Add fake importer tests for progress, completion, cancellation, and failure.

Verification:

- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...`

## Sprint 4: Frontend API and Import Workflow Hook

Status: pending

Planned tasks:

- [ ] Add typed frontend `importYouTube` API helper.
- [ ] Extend `useFileImport` to support URL imports.
- [ ] Represent YouTube imports in the existing progress shelf.
- [ ] Update file event handling for processing/progress/ready/failed.
- [ ] Invalidate file and audio-file queries at the smallest useful scope.

Verification:

- [ ] `npm run type-check` from `web/frontend`.

## Sprint 5: Homepage Import Dropdown and Dialog

Status: pending

Planned tasks:

- [ ] Replace direct Import click with an import dropdown menu.
- [ ] Add Upload files and Import from YouTube menu items.
- [ ] Add one-input YouTube URL dialog.
- [ ] Submit URL through the import hook and close the dialog after accepted creation.
- [ ] Preserve hidden file input behavior for Upload files.
- [ ] Remove dead legacy frontend YouTube components/hooks.

Verification:

- [ ] `npm run type-check` from `web/frontend`.
- [ ] Browser verification on homepage import flow.

## Sprint 6: UX Polish and End-to-End Verification

Status: pending

Planned tasks:

- [ ] Verify progress shelf copy for upload, YouTube processing, ready, and failed states.
- [ ] Verify recording list updates from SSE without refresh.
- [ ] Verify imported YouTube audio opens on the audio detail page.
- [ ] Verify imported YouTube audio can be submitted for transcription.
- [ ] Verify desktop and mobile layout.
- [ ] Update tracker with completed artifacts and residual risks.

Verification:

- [ ] `npm run type-check` from `web/frontend`.
- [ ] `npm run build` from `web/frontend`.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...`
