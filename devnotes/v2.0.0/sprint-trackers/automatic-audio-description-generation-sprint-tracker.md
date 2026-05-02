# Sprint Tracker: Automatic Audio Description Generation

This tracker belongs to `devnotes/v2.0.0/sprint-plans/automatic-audio-description-generation-sprint.md`.

Status: in progress.

## Sprint 1: Backend Persistence and Contract

Status: complete

Progress:

- [x] Add generated description metadata columns to the transcription/recording persistence model.
- [x] Add repository method to persist description on both transcription and parent recording rows.
- [x] Extend file list/get responses with `description`.
- [x] Extend file SSE payload handling to include `description`.
- [x] Add backend contract tests.
- [x] Keep absent descriptions graceful as an empty string in file responses.
- [x] Confirm Sprint 2 generator must no-op when no active LLM provider or small model is configured.

Verification:

- [x] `env GOCACHE=/tmp/go-build-cache go test ./internal/database`
- [x] `env GOCACHE=/tmp/go-build-cache go test ./internal/repository`
- [x] `env GOCACHE=/tmp/go-build-cache go test ./internal/api`
- [x] `npm run type-check` from `web/frontend`

## Sprint 2: Description Generator Workflow

Status: complete

Progress:

- [x] Resolve completed summary plus outline context for a transcription.
- [x] Add small-model description prompt and provider call.
- [x] Validate exactly two non-empty lines after normalization.
- [x] Persist description metadata and publish `file.updated`.
- [x] Add bounded recovery for completed summary-plus-outline results missing descriptions.
- [x] Gracefully no-op when no active LLM provider or small model is configured.
- [x] Use summary and outline context only; transcript text is not loaded or sent.

Verification:

- [x] Missing provider/model no-op test.
- [x] Missing outline skip test.
- [x] Summary-plus-outline provider-call test.
- [x] Transcript-not-used test.
- [x] Invalid output rejection tests.
- [x] Valid output persistence and SSE test.
- [x] `env GOCACHE=/tmp/go-build-cache go test ./internal/summarization`
- [x] `env GOCACHE=/tmp/go-build-cache go test ./internal/repository`

## Sprint 3: Home Page Card UI and SSE Reactivity

Status: pending

Progress:

- [ ] Add `description` to frontend file API types.
- [ ] Patch TanStack Query caches from `file.updated.description`.
- [ ] Render description in Home page recording cards with compact two-line styling.
- [ ] Omit description area when absent.
- [ ] Confirm card actions and progress states remain stable.

Verification:

- [ ] Frontend type-check.
- [ ] Browser check for absent description.
- [ ] Browser check for reactive description update.
- [ ] Browser visual pass across desktop and narrow viewport.

## Sprint 4: End-to-End Verification and Cleanup

Status: pending

Progress:

- [ ] Run backend focused tests.
- [ ] Run frontend type-check and build.
- [ ] Run `git diff --check`.
- [ ] Verify real transcription flow through the in-app browser.
- [ ] Update tracker with findings, completed commands, and residual risks.

Verification:

- [ ] Backend tests pass.
- [ ] Frontend checks pass.
- [ ] Home page updates without refresh after description persistence.

Residual Risks:

- Outline source identity must be confirmed before implementation because outline currently appears to be produced through summary widget runs.
- LLM output quality varies; invalid or generic descriptions should be rejected rather than persisted.
