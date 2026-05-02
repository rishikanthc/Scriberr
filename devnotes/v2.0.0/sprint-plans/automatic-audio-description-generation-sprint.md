# Automatic Audio Description Generation Sprint

## Goal

After an audio transcription has both the automatic summary and outline available, generate a concise two-line description with the configured small LLM model. Persist the description on the audio recording row and show it inside each Home page recording card. The Home page must update reactively through SSE as soon as the description is saved.

## Product Rules

- Run after transcription-triggered summarization produces the required context.
- Use the active LLM provider's configured small model.
- If no LLM provider or small model is configured, do nothing and do not fail transcription, summary, outline, or title generation.
- Use completed summary and outline context only. Do not send the full transcript to the description prompt.
- The generated description must be exactly 2 non-empty lines.
- Enforce the two-line shape in both the prompt and backend validation/post-processing.
- Persist the description on the visible recording/file row, not only on the transcription job row.
- This generation should happen every time a new transcription workflow reaches the required summary-plus-outline state.
- Publish a small `file.updated` SSE event with the file id and description after persistence.
- A full page refresh must reconstruct the description from persisted metadata.

## Data Model

Add recording description metadata to `transcriptions`:

- `llm_description` nullable text
- `llm_description_generated_at` nullable timestamp
- optional `llm_description_source_summary_id` text, if useful for skipping stale duplicates or tracing which summary produced the current description

Rules:

- Store the final two-line description in `llm_description`.
- Save the same description on the transcription row and parent recording row when the summary belongs to a transcription job.
- Existing rows can migrate with null description metadata.
- Manual title edits do not affect description metadata.
- Regeneration is allowed for a new transcription workflow because the description should track the latest summary and outline.

## Context Source

Primary context:

- Automatic summary content from `summaries.content`.
- Outline content from the completed summary widget run whose widget identity maps to the outline workflow.

Implementation guidance:

- Prefer a focused repository helper that resolves the latest completed summary plus completed outline run for a transcription id.
- Keep the service logic resilient if outline is missing or still pending: skip description generation until both pieces are available.
- Do not parse or load transcript JSON for description generation.

## Prompt Contract

The prompt should be strict and easy to validate:

```txt
Write a description for this audio recording.

Rules:
- Output exactly 2 lines.
- Each line must be a concise sentence fragment or sentence.
- No bullets, numbering, quotes, markdown, heading, prefix, or extra commentary.
- Use only the summary and outline below.
- Do not mention "summary", "outline", or "transcript".

Summary:
<summary content>

Outline:
<outline content>
```

Backend validation:

- Normalize line endings.
- Trim whitespace and surrounding quotes.
- Remove empty lines.
- Accept only exactly 2 non-empty lines after normalization.
- Bound each line length to a sane card-friendly maximum.
- Reject generic outputs such as "Audio recording" or "This audio discusses the topic".

## Sprint Run 1: Backend Persistence and Contract

Scope:

- Add DB migration/model fields for generated description metadata.
- Add repository method to persist generated description on both the transcription and parent recording rows.
- Extend file API response shape with `description`.
- Extend file SSE event payload typing to include `description`.
- Add route/API contract coverage for the new response field.

Tests:

- Migration opens older DBs with null description metadata.
- File list and get responses include `description` as an empty string when absent.
- Persisting description updates the parent recording row used by the Home page.
- `file.updated` can carry description without leaking summary or outline content.

Commit:

- Commit backend schema/API contract changes atomically.

## Sprint Run 2: Description Generator Workflow

Scope:

- Add a small LLM description generation step in the summarization workflow after summary and outline are both completed.
- Reuse active LLM config lookup and small-model client construction.
- Build prompt from summary plus outline only.
- Add sanitizer/validator for exactly 2 lines.
- Persist the description and publish `file.updated` after the DB update.
- If provider config is missing, outline is unavailable, provider output is invalid, or generation fails, log and skip without failing upstream work.
- Add recovery pass for completed summary-plus-outline results missing a generated description, bounded to avoid startup spikes.

Tests:

- Missing provider or small model is a no-op.
- Missing outline skips generation.
- Completed summary plus outline triggers one provider call.
- Provider receives summary and outline, not transcript text.
- Invalid one-line, three-line, empty, markdown, or generic output is rejected.
- Valid two-line output is persisted and emits `file.updated`.
- Regeneration updates metadata when a later summary/outline pair is available.

Commit:

- Commit backend generation workflow and tests atomically.

## Sprint Run 3: Home Page Card UI and SSE Reactivity

Scope:

- Extend `ScriberrFile` frontend API type with `description`.
- Update `useFileEvents` cache patching so `file.updated.description` updates file list and single-file caches immediately.
- Display the description inside Home page recording cards with compact, quiet typography.
- Preserve scan speed: two lines maximum, muted text, stable card height behavior, no decorative container.
- Add empty-state behavior by simply omitting the description when absent.
- Ensure long generated lines clamp/wrap cleanly on desktop and mobile.

Tests:

- Frontend type-check passes.
- Browser check: Home page shows no placeholder for recordings without descriptions.
- Browser check: when a `file.updated` event includes description, the matching card updates without page refresh.
- Browser check: card actions, import menu, ASR profile picker, and progress states remain visually stable.

Commit:

- Commit frontend API/UI/SSE changes atomically.

## Sprint Run 4: End-to-End Verification and Cleanup

Scope:

- Run focused backend package tests for database, repository, API, and summarization.
- Run frontend type-check and build.
- Run `git diff --check`.
- Verify a real transcription flow: summary completes, outline completes, description appears on Home without refresh.
- Update the sprint tracker with completed artifacts, commands, and residual risks.

Verification:

- Backend tests for description workflow.
- Frontend `npm run type-check`.
- Frontend `npm run build`.
- Browser verification through the in-app browser.

Risks:

- The exact source of "outline" depends on the configured summary widget identity. Sprint 1 should confirm the existing outline widget/run contract before wiring the generator.
- LLM providers may return prose with extra blank lines or markdown. Backend validation should reject questionable output instead of saving bad card text.
- Description generation should stay out of the transcript hot path and must not parse large transcript JSON.
