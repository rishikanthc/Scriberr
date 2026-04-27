# Sprint 1 API Foundation Notes

## Completed

- Removed the legacy `internal/api` handler modules for transcription, chat, notes, summaries, speaker mapping, OpenAI validation, CLI auth/install, and old log retrieval.
- Removed generated old API docs under `api-docs/`.
- Removed old API-heavy tests under `tests/` that asserted deleted routes and old response contracts.
- Added a new canonical API skeleton in `internal/api/router.go`.
- Added API foundation tests in `internal/api/router_test.go`.
- Added request ID middleware with `X-Request-ID` echoing and generated request IDs.
- Added structured error responses matching the new `{ "error": { ... } }` contract.
- Added health/readiness endpoints:
  - `GET /health`
  - `GET /api/v1/health`
  - `GET /api/v1/ready`
- Added JWT/API-key auth guard foundations for protected routes.
- Added placeholder canonical route groups for files, transcriptions, profiles, settings, events, models, and admin queue.
- Replaced the old slog-backed logger implementation with a Zap-backed structured logger while preserving the existing package-level logger call sites.
- Deleted old API auth middleware from `pkg/middleware/auth.go`; API auth now lives in the API layer for the new foundation.

## Placeholder Behavior

Most Sprint 1 routes return `501 Not Implemented` after authentication. This is intentional and matches the API spec allowance for the foundation sprint.

Command-style routes that Gin cannot register cleanly as static colon paths are handled by the API `NoRoute` fallback for now:

- `POST /api/v1/files:import-youtube`
- `POST /api/v1/transcriptions:submit`

Resource command routes are registered through parameterized handlers:

- `POST /api/v1/transcriptions/{id}:cancel`
- `POST /api/v1/transcriptions/{id}:retry`
- `POST /api/v1/profiles/{id}:set-default`

## Known Follow-Ups

- `go.uber.org/zap v1.27.0` was added to `go.mod`, but `go.sum` could not be updated in this environment because the Go toolchain is unavailable.
- The CLI client and embedded frontend bundle still reference old `/api/v1/transcription/*` routes. They are outside this API-only sprint scope. Decide before Sprint 3 whether to add temporary API aliases or update those clients in a separate approved change.
- `api-docs/` is now empty. Sprint 6 should create new canonical API documentation.

## Verification

- `git diff --check` passes.
- `go test ./internal/api ./pkg/logger ./pkg/middleware` could not run because `go` is not installed in this environment.
- `gofmt` could not run because `gofmt` is not installed in this environment.
