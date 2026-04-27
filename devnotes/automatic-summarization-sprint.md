# Automatic Summarization Sprint

## Goal

Generate a concise transcript summary automatically for new transcription completions when, and only when, the user has a working LLM provider plus both large and small model selections configured. Summaries are generated with the configured small model and displayed in the audio detail Summary tab.

Existing completed transcriptions without summaries are intentionally out of scope. This feature only applies to new transcription workflows going forward.

## Product Rules

- Automatic summarization is available only when the default LLM provider is configured, reachable, and has both `large_model` and `small_model` set.
- Trigger summarization after a new transcript becomes available for an audio recording.
- Use the small model for summary generation.
- Assemble transcript input as plain text by joining transcript segment text only. Do not include timestamps, speakers, or labels.
- Resolve the small model context size before generation.
- If the prompt plus transcript exceed the available context budget, truncate transcript text to fit and surface that truncation in the UI.
- Persist the generated summary, not volatile model lists or fetched provider metadata.
- Summary output is exactly one paragraph generated from the provided prompt.

## Sprint Run 1: Backend Workflow

Scope:
- Add a summary generation service that loads the active LLM provider, validates readiness, builds a provider client, resolves context size, assembles the prompt, truncates transcript text when needed, and stores a `summaries` row.
- Trigger the service after a transcription is newly completed by the transcription worker.
- Publish a lightweight summary event so active clients can refresh the Summary tab.
- Add authenticated summary read endpoint for the audio detail page.

Implementation notes:
- Keep the HTTP handler thin: parse public transcription IDs, authorize ownership, call repository/service helpers, return a typed response.
- Keep summarization out of request handlers and attach it to the durable transcription completion path.
- Use bounded prompt construction and provider calls with context.
- Treat missing/incomplete LLM settings as a no-op, not a transcription failure.
- Store failures as summary rows with `status = failed` only when summarization starts and provider generation fails.

Tests:
- Provider not configured or models incomplete does not generate a summary.
- Completed transcription with configured provider generates a completed summary.
- Long transcript input is truncated and persisted with truncation metadata in the response.
- Summary read endpoint returns latest summary and 404/empty state appropriately.

## Sprint Run 2: Frontend Summary Surface

Scope:
- Add feature-local transcription summary API and TanStack Query hook.
- Invalidate summary query from summary SSE events.
- Render the Summary tab with an Overview heading and generated paragraph, matching the provided screenshot while ignoring the template control.
- Show explicit pending/failed/unavailable states without adding decorative cards.
- Show a toast warning when the latest summary indicates transcript truncation.

Implementation notes:
- Keep API types explicit and use backend snake_case fields.
- Fetch summary by transcription ID, not file ID.
- Avoid direct fetches in leaf components.
- Keep the Summary tab quiet and content-first: tab bar, Overview row, paragraph.

Tests:
- `npm run build`
- Manual browser check of the audio detail Summary tab after a new transcription completion.

## Sprint Run 3: Verification and Commit

Scope:
- Run focused backend tests for summarization and LLM provider touched paths.
- Run frontend build.
- Run `git diff --check`.
- Commit in small coherent units.

Risks:
- Provider context windows are approximate for OpenAI-compatible endpoints that do not expose metadata. Use the existing service fallback behavior.
- Automatic summarization runs after transcription completion; the transcript remains successful even if summarization fails.
- Existing legacy summary/template routes are not revived unless needed by the new detail page.
