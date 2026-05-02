# Sprint Tracker: Automatic Audio Description Generation

This tracker belongs to `devnotes/v2.0.0/sprint-plans/automatic-audio-description-generation-sprint.md`.

Status: planned.

## Sprint 1: Backend Persistence and Contract

Status: pending

Progress:

- [ ] Add generated description metadata columns to the transcription/recording persistence model.
- [ ] Add repository method to persist description on both transcription and parent recording rows.
- [ ] Extend file list/get responses with `description`.
- [ ] Extend file SSE payload handling to include `description`.
- [ ] Add backend contract tests.

Verification:

- [ ] Database migration tests.
- [ ] Repository tests.
- [ ] API tests.

## Sprint 2: Description Generator Workflow

Status: pending

Progress:

- [ ] Resolve completed summary plus outline context for a transcription.
- [ ] Add small-model description prompt and provider call.
- [ ] Validate exactly two non-empty lines after normalization.
- [ ] Persist description metadata and publish `file.updated`.
- [ ] Add bounded recovery for completed summary-plus-outline results missing descriptions.

Verification:

- [ ] Missing provider/model no-op test.
- [ ] Missing outline skip test.
- [ ] Summary-plus-outline provider-call test.
- [ ] Transcript-not-used test.
- [ ] Invalid output rejection tests.
- [ ] Valid output persistence and SSE test.

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
