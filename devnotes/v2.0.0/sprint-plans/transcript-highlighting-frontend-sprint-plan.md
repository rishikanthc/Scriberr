# Transcript Highlighting Frontend Sprint Plan

This plan implements transcript highlighting end to end by integrating the v1 annotation backend with the React transcript UI.

Related backend plan:

- `devnotes/v2.0.0/sprint-plans/highlights-notes-backend-sprint-plan.md`

Frontend rules:

- `devnotes/v2.0.0/rules/react-architecture-rules.md`
- `devnotes/v2.0.0/rules/design-system-philosophy.md`

## Current Assessment

The backend now exposes canonical annotation routes:

```http
GET    /api/v1/transcriptions/{id}/annotations
POST   /api/v1/transcriptions/{id}/annotations
GET    /api/v1/transcriptions/{id}/annotations/{annotation_id}
PATCH  /api/v1/transcriptions/{id}/annotations/{annotation_id}
DELETE /api/v1/transcriptions/{id}/annotations/{annotation_id}
```

The frontend transcript view currently lives mostly in:

- `web/frontend/src/features/transcription/components/AudioDetailView.tsx`
- `web/frontend/src/features/transcription/api/transcriptionsApi.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptions.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptionDetailEvents.ts`
- `web/frontend/src/styles/design-system.css`

Useful existing transcript structure:

- Each transcript segment renders metadata separately from transcript body text.
- Transcript body paragraphs already use `.scr-transcript-text` and `data-transcript-text`.
- Speaker names use `.scr-transcript-speaker`.
- Timestamps use `.scr-transcript-time`.

That separation should be preserved and strengthened so browser selection and highlight anchoring only operate on transcript text, never speaker labels or timestamps.

Legacy or partial code exists but should not drive the new implementation:

- `web/frontend/src/features/transcription/hooks/useTranscriptSelection.ts`
- `web/frontend/src/features/transcription/hooks/useSelectionMenu.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptionNotes.ts`
- `web/frontend/src/types/note.ts`

Those files still reference older note routes or selection assumptions. The new work should use canonical annotation routes and typed feature-owned API/hooks.

## Product Goal

Users can select transcript text with the cursor, click a compact floating menu, and persist a highlight. Persisted highlights render automatically whenever the transcript is opened again.

Initial scope:

- Support highlight creation only.
- Show a disabled or non-functional note/comment icon for visual affordance only if needed by design, but do not wire note creation yet.
- Highlight selections must exclude speaker labels and timestamps.
- Persisted highlights must survive refresh and transcript reopen.
- Highlight rendering must be scoped to the selected transcription and authenticated user.

Out of scope for this sprint set:

- Note editor/comment creation.
- Highlight color picker.
- Editing or deleting highlights in the UI.
- Shared multi-user annotation UI.
- Full transcript virtualization unless performance testing proves it is needed immediately.

## Target Frontend Model

Add a feature-owned annotation API boundary:

```txt
features/transcription/api/annotationsApi.ts
features/transcription/hooks/useTranscriptAnnotations.ts
```

Use backend field names at the API boundary:

```ts
type TranscriptAnnotation = {
  id: string;
  transcription_id: string;
  kind: "highlight" | "note";
  content: string | null;
  color: string | null;
  quote: string;
  anchor: {
    start_ms: number;
    end_ms: number;
    start_word?: number;
    end_word?: number;
    start_char?: number;
    end_char?: number;
    text_hash?: string;
  };
  status: "active" | "stale";
  created_at: string;
  updated_at: string;
};
```

The hook layer should expose:

- `transcriptAnnotationsQueryKey(transcriptionId)`
- `useTranscriptAnnotations(transcriptionId, enabled)`
- `useCreateTranscriptHighlight(transcriptionId)`
- event invalidation for `annotation.created`, `annotation.updated`, and `annotation.deleted`

Mutation success should invalidate only the annotation query for the active transcription.

## Target Selection Model

Selection must be derived from DOM ranges that intersect transcript text nodes only:

- Accept selection only when every meaningful selected text node is inside `[data-transcript-text]`.
- Ignore or reject selection crossing `.scr-transcript-speaker`, `.scr-transcript-time`, player UI, tabs, or summaries.
- Compute quote from transcript text nodes only, not from `window.getSelection().toString()` when that includes metadata.
- Compute anchor fields from segment-local metadata:
  - `start_ms`
  - `end_ms`
  - `start_word` / `end_word` when word offsets exist
  - `start_char` / `end_char`
  - `text_hash`

The transcript rendering should provide stable selection metadata:

```txt
data-transcript-segment-index
data-transcript-text
data-start-char
data-end-char
```

If word-level offsets are unavailable, support character/time anchoring where possible. If no reliable time anchor can be computed, do not show the highlight action.

## Target UI

After a valid selection:

- Show a compact floating menu directly above the selected text.
- Center the menu horizontally over the selected range.
- Flip below the selection if there is not enough space above.
- Use icon buttons from Lucide.
- Highlight button is enabled.
- Note/comment button is present only as disabled or deferred UI if it matches the desired interaction, with tooltip/accessible label.
- Clear the menu when selection collapses, user scrolls, route changes, or highlight creation succeeds.

The menu should feel contextual and quiet:

- Low visual weight.
- No explanatory in-app text.
- Buttons are real buttons with accessible names/tooltips.
- Responsive behavior must work on desktop and mobile, but desktop cursor selection is the primary first pass.

## Target Highlight Rendering

Persisted highlights should render as inline marks over transcript text, not as overlay rectangles detached from text flow.

Preferred rendering approach:

- Split segment text into ranges based on active highlight anchors.
- Render matching ranges as `<mark>` inside `.scr-transcript-text`.
- Use design tokens for a subtle translucent background.
- Preserve karaoke/current-word highlighting without fighting persistent highlights.

Rules:

- Never highlight speaker/timestamp nodes.
- Do not expose local paths or internal DB fields.
- Do not reparse large transcript payloads on every render.
- Memoize segment-to-highlight mapping by `transcript` and `annotations`.
- Handle stale annotations visibly but subtly, or omit stale highlights from inline rendering until a later UX is designed.

## Sprint 1: API Boundary and Query Hooks

Goal: add typed frontend access to backend annotations without touching transcript UI behavior yet.

Tasks:

- Add `features/transcription/api/annotationsApi.ts`.
- Add types for annotation responses, list response, create request, and anchor.
- Add `listTranscriptAnnotations` and `createTranscriptAnnotation`.
- Add `features/transcription/hooks/useTranscriptAnnotations.ts`.
- Add query keys and create-highlight mutation.
- Invalidate annotation queries from `useTranscriptionDetailEvents` when annotation SSE events arrive.
- Remove or clearly quarantine old direct-fetch note hooks from the new path.

Acceptance criteria:

- Frontend code uses canonical `/api/v1/transcriptions/{id}/annotations` routes.
- No new component fetches annotation URLs inline.
- API types use backend snake_case fields at the boundary.
- Annotation SSE events invalidate only the active transcription annotation query.

## Sprint 2: Clean Transcript Selection and Anchor Builder

Goal: make browser selection operate only on transcript text, never metadata.

Tasks:

- Extract transcript rendering concerns from `AudioDetailView.tsx` into focused feature components if needed:
  - `TranscriptPanel`
  - `TranscriptSegment`
  - `TranscriptSelectionMenu`
- Add a selection hook such as `useTranscriptTextSelection`.
- Ensure speaker and time elements are not selectable by adding CSS and selection validation.
- Build a DOM range parser that only accepts `[data-transcript-text]` text nodes.
- Compute clean quote and anchor metadata.
- Add `sha256:` text hash generation in the frontend for selected quote text.
- Reject invalid selections that include metadata, empty text, or unreliable anchor timing.

Acceptance criteria:

- Dragging over transcript text produces a clean quote without speaker labels or timestamps.
- Dragging from speaker/timestamp into text does not create a valid highlight action.
- Selection coordinates are available for menu positioning.
- Anchor payload is ready for backend create requests.

## Sprint 3: Floating Selection Menu

Goal: show the contextual menu above selected transcript text.

Tasks:

- Add `TranscriptSelectionMenu` using design-system primitives and Lucide icons.
- Position menu fixed relative to the viewport using selected range rect.
- Center above selection, clamp inside viewport, and flip below if needed.
- Add highlight and note/comment icon buttons.
- Wire highlight button to the create-highlight mutation.
- Keep note/comment visually disabled or deferred for this sprint set.
- Clear browser selection and menu after successful highlight creation.
- Add loading/disabled state for highlight mutation.

Acceptance criteria:

- Menu appears automatically after valid text selection.
- Menu is centered directly above selected text when space allows.
- Highlight button creates an annotation.
- Note/comment control does not imply working note creation.
- Icon buttons have accessible labels/tooltips.

## Sprint 4: Persisted Highlight Rendering

Goal: render saved highlights whenever a transcript is opened.

Tasks:

- Fetch annotations in `TranscriptPanel` when the latest transcription is completed.
- Render active highlight annotations inline in transcript text.
- Map annotation anchors to segment-local character ranges.
- Handle overlapping highlights deterministically.
- Exclude stale annotations from inline highlight rendering or render them with a distinct subtle stale style if needed.
- Add empty/loading/error behavior that does not block transcript reading.
- Invalidate and refresh after highlight creation and annotation SSE events.

Acceptance criteria:

- Created highlights appear immediately after mutation success.
- Refreshing and reopening the transcript shows persisted highlights.
- Highlight rendering never includes timestamps or speaker labels.
- Existing transcript reading and karaoke highlighting remain usable.

## Sprint 5: UX, Accessibility, and Regression Coverage

Goal: harden the workflow around user-visible risks.

Tasks:

- Add component/hook tests for:
  - clean quote extraction,
  - metadata-skipping selection,
  - anchor payload generation,
  - highlight range rendering,
  - annotation query invalidation.
- Add browser-level verification with desktop and mobile viewport screenshots.
- Verify long selections, multi-line selections, cross-segment selections, and no-word-offset transcripts.
- Check keyboard and assistive behavior for menu buttons.
- Tune CSS so native selection, persistent highlight marks, and karaoke highlight are visually distinct but quiet.
- Remove unused legacy note-selection paths if they are no longer referenced.

Acceptance criteria:

- The selection menu is accessible and keyboard reachable.
- Text never overflows or overlaps the floating menu incoherently.
- Highlight UI is responsive on mobile and desktop.
- All tests pass.
- Final screenshots show clean text-only highlighting and persisted highlights.

## Implementation Notes

- Prefer feature-owned components under `web/frontend/src/features/transcription/components`.
- Prefer feature-owned hooks under `web/frontend/src/features/transcription/hooks`.
- Keep data fetching in API functions and TanStack Query hooks.
- Keep selection state local to the transcript workflow.
- Avoid global state for selected text.
- Keep transcript hot paths memoized around transcript and annotation inputs.
- Use design tokens in `web/frontend/src/styles/design-system.css`.
- Use Lucide icons for the selection menu controls.

## Risks and Decisions

- Cross-segment selections may need a range model that creates one annotation spanning multiple display segments. If backend anchors are sufficient, use one annotation with global word/char indexes; otherwise split into multiple highlights in a later sprint.
- Backend `anchor_start_char` / `anchor_end_char` are transcript-level. The frontend must normalize display segment offsets back to transcript-level indexes.
- If transcripts lack word data, timing must come from segment bounds. If selection is within a segment but no word offsets exist, anchor to segment start/end and char range.
- Native browser selection styling and persisted `<mark>` styling must remain visually distinct.
- The note/comment icon is intentionally deferred until note creation UX is designed.
