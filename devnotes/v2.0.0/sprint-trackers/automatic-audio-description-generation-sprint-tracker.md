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
- [x] Keep description generation tied to new transcription-triggered summary/outline completion.
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

Status: complete

Progress:

- [x] Add `description` to frontend file API types.
- [x] Patch TanStack Query caches from `file.updated.description`.
- [x] Render description in Home page recording cards with compact two-line styling.
- [x] Omit description area when absent.
- [x] Confirm card actions and progress states remain stable by keeping the description in the title/meta column only.

Verification:

- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.
- [x] Browser check for absent description: Home page rendered 10 cards and 0 description blocks without placeholder gaps.
- [ ] Browser check for reactive description update.
- [x] Browser desktop visual pass for absent descriptions.
- [ ] Browser narrow viewport visual pass.

## Sprint 4: End-to-End Verification and Cleanup

Status: complete

Progress:

- [x] Run backend focused tests.
- [x] Run frontend type-check and build.
- [x] Run `git diff --check`.
- [x] Verify Home page absent-description behavior through the in-app browser.
- [x] Update tracker with findings, completed commands, and residual risks.

Verification:

- [x] `env GOCACHE=/tmp/go-build-cache go test ./internal/database ./internal/repository ./internal/summarization`
- [x] `env GOCACHE=/tmp/go-build-cache go test ./internal/api`
- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.
- [x] `git diff --check`
- [x] Browser desktop check on `http://127.0.0.1:5174/`: 10 cards, 0 description blocks, no empty placeholder area.
- [ ] Live SSE description update after a new generated description.

Residual Risks:

- Live SSE description update was not reproduced in the current browser session because no new transcription-triggered description event was generated after the running backend loaded this code. The code path is covered by the existing `file.updated.description` cache patching and backend event emission, but a hands-on check after restarting the backend and running a new transcription is still useful.
- Narrow mobile viewport was not separately resized in the in-app browser during this final pass.
- LLM output quality varies; invalid or generic descriptions are rejected rather than persisted.
