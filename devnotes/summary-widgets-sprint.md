# Summary Widgets Sprint

## Goal

Add configurable summary widgets that automatically extract custom transcript-derived information after automatic summarization completes. Widgets are defined in Settings, selected per transcript by the LLM when conditional, executed as durable backend work, and displayed as additional sections below the Overview in the audio detail Summary tab.

This sprint intentionally plans the feature only. Implementation should follow in small coherent commits.

## Product Rules

- Summary widgets run only for new summary workflows going forward.
- Widget execution waits for the automatic summary to complete.
- Missing or incomplete LLM provider settings disable widget execution without failing transcription or summarization.
- Use the configured small LLM for widget selection and widget generation unless a later sprint adds per-widget model selection.
- Always-enabled widgets run for every completed summary.
- Conditional widgets are first passed through an LLM relevance selection step using the completed summary as context.
- Conditional selection returns exact widget names only. The backend maps returned names to persisted widget IDs and ignores unknown names.
- Each selected widget runs sequentially to avoid provider overload and to keep state transitions simple.
- Widget context is configurable per widget: completed summary or plain transcript text.
- Transcript context is assembled the same way as summarization input: segment text only, no timestamps, no speaker labels.
- If context exceeds the small model context window, truncate to fit and persist truncation metadata on the widget run.
- Widget results are persisted. Provider model lists and runtime capability details are not persisted.
- SSE events remain lightweight invalidation/status notifications. The Summary tab fetches persisted widget results as source of truth.
- Markdown rendering is display-only. If `render_markdown` is enabled, render with the `textforge` npm package in read-only mode with editing disabled.
- If `render_markdown` is enabled, append markdown typeset formatting instructions to the widget prompt before attaching the summary/transcript context.
- Do not install `textforge` from `references/textforge`; use the npm package. The local reference is documentation only.

## Data Model Draft

Add durable widget definitions:

```txt
summary_widgets
- id
- user_id
- name
- description nullable
- always_enabled boolean
- when_to_use text nullable
- context_source enum: "summary" | "transcript"
- prompt text
- render_markdown boolean
- display_title text
- enabled boolean
- created_at
- updated_at
- deleted_at
```

Add durable widget runs:

```txt
summary_widget_runs
- id
- summary_id
- transcription_id
- widget_id
- user_id
- widget_name
- display_title
- context_source
- render_markdown
- model_name
- provider
- status: pending | processing | completed | failed
- output text
- error_message nullable
- context_truncated boolean
- context_window int
- input_characters int
- started_at nullable
- completed_at nullable
- failed_at nullable
- created_at
- updated_at
```

Repository methods should express lifecycle operations rather than exposing generic updates:

- List/create/update/delete widget definitions for a user.
- List enabled widgets for a user.
- Enqueue widget runs for a completed summary.
- Claim next pending widget run.
- Complete/fail widget run.
- Recover processing widget runs on startup.
- List widget runs by summary or transcription for display.

## Prompt Design Draft

### Conditional Widget Selection

Input:

- Completed summary as context.
- Names and `when_to_use` text for enabled widgets where `always_enabled = false`.

Output contract:

- Return a JSON object with one field: `widget_names`.
- `widget_names` is an array of exact strings copied from the provided widget list.
- Return an empty array when none apply.
- Do not include explanations, markdown, or extra keys.

Selection prompt should emphasize:

- Choose only widgets clearly relevant to this recording.
- Prefer not running a widget when relevance is uncertain.
- Use exact names from the list.
- Never invent widgets.

### Widget Generation

Input:

- Widget prompt.
- Optional markdown typeset formatting instructions when `render_markdown = true`.
- Selected context, either summary or transcript.

Output contract:

- Return only the widget output.
- Do not mention the prompt, internal rules, model, or context source.
- If the requested information is not present, produce a concise useful empty-state response instead of fabricating details.
- When `render_markdown = true`, the backend appends markdown typeset formatting instructions after the saved widget prompt and before the context block. When false, prefer plain text.

Prompt assembly order:

```txt
<saved widget prompt>

<markdown typeset formatting instructions, only when render_markdown is true>

Context:
<summary or transcript context>
```

## Sprint Run 1: Backend Widget Configuration API

Scope:

- Add `SummaryWidget` model and migration/indexes.
- Add repository operations for user-owned widget definitions.
- Add authenticated API routes under settings, for example:
  - `GET /api/v1/settings/summary-widgets`
  - `POST /api/v1/settings/summary-widgets`
  - `PATCH /api/v1/settings/summary-widgets/:id`
  - `DELETE /api/v1/settings/summary-widgets/:id`
- Validate:
  - `name` is required and bounded.
  - `display_title` is required and bounded.
  - `prompt` is required and bounded.
  - `when_to_use` is required when `always_enabled = false`.
  - `context_source` is `summary` or `transcript`.
  - Duplicate widget names for the same user are rejected or normalized by a clear rule.
- Keep handlers thin: auth, request parsing, one service/repository call, response mapping.
- Add route contract coverage and validation tests.

Notes:

- Preserve soft delete for definitions so historical runs can keep their widget metadata.
- Store run display metadata on `summary_widget_runs` so old results still render if a widget is later renamed.

## Sprint Run 2: Backend Widget Execution Workflow

Scope:

- Add widget run model and repository lifecycle methods.
- Extend summarization service or add a sibling widget service that starts after `summary.completed`.
- On summary completion:
  - Load enabled widgets.
  - Split always-enabled and conditional widgets.
  - Run conditional selection using the completed summary.
  - Enqueue one run per selected widget.
  - Wake an event-driven worker.
- Worker behavior:
  - Recover processing runs once on startup.
  - Wake only on enqueue/startup, not on interval polling.
  - Drain pending runs per wake.
  - Claim and process one widget run at a time.
  - Generate widget output using configured context and widget prompt.
  - Append markdown typeset formatting instructions after the widget prompt and before the context when `render_markdown = true`.
  - Persist completed or failed state.
  - Publish `summary_widget.pending`, `summary_widget.processing`, `summary_widget.completed`, `summary_widget.failed`, and optional `summary_widget.truncated` SSE events.
- Add `GET /api/v1/transcriptions/:id/summary/widgets` or include runs in the summary response after deciding the cleaner frontend query boundary.

Tests:

- Missing LLM settings does not enqueue widget runs.
- Summary completion with no widgets is a no-op.
- Always-enabled widget creates a run.
- Conditional selector maps exact returned names to widget IDs and ignores unknown names.
- Widget runs execute sequentially and persist output.
- Transcript context uses segment text only.
- Context truncation is persisted.
- Failed widget generation does not fail summary or transcription.
- Startup recovery moves processing widget runs back to pending.
- Empty queue lookup is quiet and does not log noisy record-not-found errors.

## Sprint Run 3: Settings Summarization Tab

Scope:

- Add a new Settings tab: `Summarization`.
- Add a `Widgets` section with a compact header and `New widget` button, following the ASR profile pattern.
- Add feature-local frontend files:
  - `web/frontend/src/features/settings/api/summaryWidgetsApi.ts`
  - `web/frontend/src/features/settings/hooks/useSummaryWidgets.ts`
  - `web/frontend/src/features/settings/components/SummaryWidgetsPanel.tsx`
  - `web/frontend/src/features/settings/components/SummaryWidgetDialog.tsx`
- Dialog fields:
  - Name.
  - Optional description.
  - Always enabled switch.
  - When-to-use textarea, visible and required only when always enabled is off.
  - Context selector: transcript or summary.
  - Prompt textarea.
  - Render as markdown switch.
  - Display title input.
- List rows should show name, enabled/conditional state, context source, markdown state, and display title.
- Support edit/delete with confirmation for delete.
- Use TanStack Query for all server state and mutations.
- Keep UI compact, quiet, and aligned with existing settings panel styles.

Tests:

- `npm run build`.
- Manual settings check for create/edit/delete states and validation errors.

## Sprint Run 4: Summary Tab Widget Results

Scope:

- Add transcription feature API/hook for widget run results, unless widget runs are included in the summary endpoint.
- Invalidate the widget results query from `summary_widget.*` SSE events.
- Keep the detail page subscribed while a latest transcription exists, matching the current summary SSE behavior.
- Render completed widget runs below Overview as content sections:
  - Section icon can be chosen from a small fixed mapping or default to a list/text icon.
  - Heading uses persisted `display_title`.
  - Failed runs show a subdued inline failure state.
  - Pending/processing runs show a compact inline status row.
- Keep Summary tab correct after refresh by fetching persisted runs.
- Avoid hot-path rerenders in `AudioDetailView`; extract `SummaryPanel`, overview, and widget result components if needed.

Tests:

- `npm run build`.
- Manual check that widget sections appear after summary completion and refresh correctly.

## Sprint Run 5: Textforge Markdown Rendering

Scope:

- Install `textforge` from npm in `web/frontend`.
- Create a small read-only wrapper component in the transcription feature or shared UI if reusable:
  - Import `TextforgeEditor` from `textforge`.
  - Import `textforge/textforge.css`.
  - Mount into a ref.
  - Use `contentType: "markdown"`.
  - Set `editable: false`.
  - Set `locked: true`.
  - Disable unnecessary heavy features if supported and visually appropriate for multiple instances.
  - Destroy editor on unmount.
  - Update content when the widget output changes.
- Use plain text rendering for widgets where `render_markdown = false`.
- Scope CSS so Textforge output fits the quiet Summary tab layout and does not introduce toolbar/editor affordances.

Tests:

- `npm run build`.
- Browser check with a markdown widget containing headings, bullets, and checkboxes if Textforge supports the syntax.
- Browser check with multiple markdown widgets mounted at once.

## Sprint Run 6: End-to-End Verification and Commit

Scope:

- Run focused backend tests:
  - summary widget API tests
  - summarization/widget service tests
  - route contract tests
  - migration/schema tests if touched
- Run frontend build.
- Run `git diff --check`.
- Manually test:
  - Create always-enabled widget.
  - Create conditional widget.
  - Generate a new transcription.
  - Summary appears first.
  - Widget selection and runs follow.
  - Widget outputs persist after refresh.
  - Failed widget does not affect summary/transcription.
- Commit in small coherent units:
  - backend widget config API
  - backend widget workflow
  - settings UI
  - summary tab rendering/Textforge integration
  - polish/tests

## Open Decisions

- Whether widget runs should be returned from `GET /transcriptions/:id/summary` or a separate `/summary/widgets` endpoint. Prefer separate if it keeps query invalidation and loading states clearer.
- Whether conditional selection should be persisted as a separate selection record. Prefer not to add it initially; persist the resulting widget runs and failed run states.
- Whether duplicate widget names should be forbidden globally per user or allowed with selection ambiguity rules. Prefer forbidding duplicate active names per user because selector output is name-based.
- Whether transcript-context widgets should wait for summary. Current product flow says yes, because conditional selection depends on summary and the UI should show Overview first.
- Whether truncation warnings should toast. Prefer no toast initially unless requested; persist and optionally show a small inline note on affected widget runs.
