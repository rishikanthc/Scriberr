# Transcript Note Taking Frontend Sprint Plan

This plan adds the user-facing note-taking workflow on top of the completed annotation backend and transcript highlighting frontend.

Related plans:

- `devnotes/v2.0.0/sprint-plans/highlights-notes-backend-sprint-plan.md`
- `devnotes/v2.0.0/sprint-plans/transcript-highlighting-frontend-sprint-plan.md`

Architecture rules:

- `devnotes/v2.0.0/rules/backend-architecture-rules.md`
- `devnotes/v2.0.0/rules/react-architecture-rules.md`
- `devnotes/v2.0.0/rules/design-system-philosophy.md`

## Current Assessment

The backend annotation model already supports notes as first-class annotations:

```http
GET    /api/v1/transcriptions/{id}/annotations
POST   /api/v1/transcriptions/{id}/annotations
GET    /api/v1/transcriptions/{id}/annotations/{annotation_id}
PATCH  /api/v1/transcriptions/{id}/annotations/{annotation_id}
DELETE /api/v1/transcriptions/{id}/annotations/{annotation_id}
```

The frontend already has the foundation needed for note taking:

- `features/transcription/api/annotationsApi.ts` contains typed annotation contracts and create/delete helpers.
- `features/transcription/hooks/useTranscriptAnnotations.ts` owns annotation query state and highlight mutations.
- `TranscriptSelectionMenu` appears after valid transcript text selection.
- The note/comment icon exists but is intentionally disabled.
- Persisted highlights render inline from active `highlight` annotations.
- `StreamingAudioPlayer` owns the audio element and has an existing seek path through its range input.

Important gaps:

- No frontend mutation currently creates `kind: "note"` annotations.
- The selection menu does not open a note composer.
- No sidebar exists for listing notes.
- The current annotation contract represents one note per annotation; supporting multiple notes on the same highlighted text needs an explicit thread/reply model instead of duplicating anchors ad hoc.
- The audio seek behavior is owned inside `StreamingAudioPlayer`, so note timestamp clicks need a clean parent-owned seek command instead of reaching into the player from the sidebar.
- `AudioDetailView.tsx` is already dense; this work should extract focused components before adding sidebar complexity.

## Product Goal

Users can select transcript text, click the note action, write a short note in a floating composer anchored near the selection, and see saved notes in a collapsible right sidebar. Clicking a note timestamp seeks the audio player to the note's anchor time.

Non-negotiable product behavior:

- Notes attach to selected transcript text using the same clean transcript-text-only anchor model as highlights.
- The floating note composer should feel like a compact inline dialogue, not a page-level modal.
- The notes sidebar must not show user identity, author names, avatars, or user metadata.
- Timestamp information is allowed and should be clickable.
- Clicking a note timestamp seeks the audio player to `anchor.start_ms`.
- Notes survive refresh because persisted annotations remain the source of truth.

Initial scope:

- Create note annotations.
- List note annotations in a collapsible right sidebar.
- Add reply/thread support so a single highlighted quote and timestamp can contain multiple note entries.
- Seek audio from note timestamps.
- Show selected quote, note content, timestamp, and lightweight status.
- Support deleting notes if the existing annotation delete route is already available in the frontend path.

Out of scope for this sprint set:

- Multi-user comments, reactions, mentions, and presence.
- Displaying user information next to notes.
- Rich-text note content.
- Note color picker.
- Server-side search across notes.
- Sharing or collaborative permission UI.

## Target Frontend Model

Keep note taking inside the transcription feature boundary:

```txt
features/transcription/api/annotationsApi.ts
features/transcription/hooks/useTranscriptAnnotations.ts
features/transcription/components/TranscriptSelectionMenu.tsx
features/transcription/components/TranscriptNoteComposer.tsx
features/transcription/components/TranscriptNotesSidebar.tsx
features/transcription/components/TranscriptNoteItem.tsx
features/transcription/components/AudioDetailView.tsx
```

Add note-specific typed helpers without creating a separate notes API path:

```ts
type CreateTranscriptNoteRequest = Omit<CreateTranscriptAnnotationRequest, "kind"> & {
  kind?: "note";
  content: string;
};
```

Hook additions:

- `useCreateTranscriptNote(transcriptionId)`
- `useUpdateTranscriptNote(transcriptionId)` if edit support is included.
- `useDeleteTranscriptNote(transcriptionId)` may reuse the same delete API as highlights.
- A small selector/helper for active notes:
  - `kind === "note"`
  - sorted by `anchor.start_ms`, then `created_at`
  - stale notes are retained but visually quiet.

Server state rules:

- TanStack Query remains the owner of persisted annotations.
- Mutations invalidate only `transcriptAnnotationsQueryKey(transcriptionId)`.
- Annotation SSE events continue to invalidate the same query.
- Components do not assemble annotation URLs inline.

## Target Note Thread Model

The first note created from selected transcript text is the thread root. Replies added from the right sidebar belong to that root and reuse the same quote, anchor, and timestamp.

Backend model options should be evaluated in Sprint 5, but the preferred contract is a small child entity rather than duplicating full annotations:

```txt
annotation_thread/root annotation
  id
  transcription_id
  kind = "note"
  quote
  anchor
  created_at
  updated_at

annotation_note_entries
  id
  annotation_id
  content
  created_at
  updated_at
  deleted_at/null
```

This keeps the anchor as the shared context and lets each note entry remain independently created, edited, or deleted later.

Backend architecture rules:

- Keep HTTP handlers thin: parse transcription and annotation IDs, validate request bodies, call one service method, and map typed responses.
- Put thread/reply decisions in the annotation service, not in handlers.
- Put persistence operations in the annotation repository, including listing roots with entries in stable order.
- Keep API request/response types separate from GORM models.
- Add migrations and repository/service tests in the same sprint as the API contract.
- Publish small annotation events for created/updated/deleted replies; queries remain the source of truth after refresh.

Frontend model:

- `TranscriptNoteAnnotation` should expose a stable `entries` array once the backend contract supports it.
- Root note annotations do not carry note text; all note text lives in entries.
- `TranscriptNotesSidebar` groups by root annotation and renders entries in chronological order.
- The count chip shows the number of non-deleted entries for that root.
- The reply row becomes a compact input; submitting creates a new entry under the root, not a duplicate annotation.
- No user names, avatars, initials, author labels, or profile metadata are rendered for root notes or replies.

## Target Selection and Composer Model

The existing `useTranscriptTextSelection` hook remains the source of valid selected text, quote, anchor, and viewport rect.

Composer behavior:

- Clicking the note action opens a floating composer positioned from the current selection rect.
- The composer receives a frozen copy of the current selection payload so typing does not depend on live browser selection.
- The composer uses a textarea or compact text input with a clear accessible label.
- Submit creates `kind: "note"` with `content`, `quote`, and `anchor`.
- Escape cancels and returns focus to the transcript action surface when possible.
- Cmd/Ctrl+Enter submits when content is non-empty.
- Empty or whitespace-only notes cannot be submitted.
- Successful creation clears the native selection, closes the composer, and opens the notes sidebar if collapsed.

Positioning rules:

- Center the composer over the selected range where possible.
- Clamp inside the viewport.
- Flip below the selection when there is not enough room above.
- Reposition on resize and scroll, or dismiss on scroll if the current positioning infrastructure cannot remain stable without jank.
- Do not mount the composer unless it is active.

## Target Sidebar Model

Add a collapsible right sidebar that can host notes now and other detail panels later.

Desktop behavior:

- Sidebar docks to the right side of the audio detail view.
- It is collapsible, with an icon button and accessible label.
- It should not shrink the transcript below readable width; use min/max constraints.
- The list scrolls independently from the transcript when needed.
- Opening and closing should be smooth, but content should not jump.

Mobile behavior:

- Prefer a slide-over drawer or bottom sheet controlled by the same state.
- Keep transcript reading and audio controls reachable.
- Avoid fixed widths that overflow small screens.

Sidebar content:

- Header: "Notes" and a count.
- Empty state: quiet and short.
- Note item:
  - selected quote
  - clickable timestamp from `anchor.start_ms`
  - note content
  - optional stale status text when needed
  - delete/edit actions only if included in that sprint
- No user names, user avatars, profile labels, or author metadata.

Timestamp behavior:

- Timestamp buttons call a parent-provided seek function.
- Seek target is `annotation.anchor.start_ms / 1000`.
- The button should be keyboard reachable and have an accessible label such as `Seek to 0:14`.

## Target Audio Seek Architecture

Do not let the notes sidebar reach into `StreamingAudioPlayer` internals.

Recommended approach:

- Lift a `seekToSeconds` command boundary to the audio detail view.
- Pass a stable `onSeekRequest(seconds)` callback to `TranscriptNotesSidebar`.
- Pass an imperative seek request or controlled seek token into `StreamingAudioPlayer`.
- `StreamingAudioPlayer` remains the only component that mutates `audio.currentTime`.
- Seeking publishes the updated playback snapshot through `playbackSync` so karaoke highlighting updates immediately.

Acceptance rules:

- Clicking a note timestamp updates the visible player time.
- Karaoke/current-word highlighting receives the new current time.
- Seeking works whether audio is currently playing or paused.
- Seeking clamps to `[0, duration]` when duration is known.

## Target Design Behavior

Follow the screenshots as interaction references, but adapt them to Scriberr's quieter design system.

Design rules:

- Use existing shadcn-style primitives and Lucide icons.
- Use neutral surfaces, restrained borders, and the existing accent color only for action/focus.
- Do not show explanatory feature text inside the product surface.
- Keep the composer compact and visually tied to the selected transcript text.
- Keep notes scannable; the quote should be prominent enough to locate context, and the note content should remain easy to read.
- Do not create nested cards. Individual note rows may be lightly framed; the sidebar itself should be a docked panel.
- Motion should be short and subtle, around 100-200ms.

## Sprint 1: Note API Helpers and Hook Mutations

Goal: expose note creation through the existing annotation API boundary.

Tasks:

- Add `CreateTranscriptNoteRequest` and note helper types in `annotationsApi.ts`.
- Add `updateTranscriptAnnotation` if edit support is included in the first implementation pass.
- Add `useCreateTranscriptNote`.
- Add `useDeleteTranscriptNote` or generalize deletion naming so notes and highlights do not duplicate mutation logic.
- Add a notes selector utility that returns active notes sorted by anchor time.
- Keep all routes under `/api/v1/transcriptions/{id}/annotations`.
- Add no UI behavior in this sprint beyond type-safe plumbing.

Acceptance criteria:

- Notes are created with `kind: "note"` through the canonical annotation route.
- Components do not fetch or mutate annotation URLs inline.
- Successful note mutations invalidate only the active transcription annotation query.
- Type checking catches missing `content` for note creation.

## Sprint 2: Floating Note Composer

Goal: turn the disabled note button into a working composer anchored to selected transcript text.

Tasks:

- Update `TranscriptSelectionMenu` to accept `onCreateNote` or `onOpenNoteComposer`.
- Add `TranscriptNoteComposer`.
- Freeze selection payload when opening the composer.
- Position the composer from the selection rect with viewport clamping and flip behavior.
- Add textarea/input, submit button, cancel behavior, keyboard shortcuts, and loading state.
- Create note annotations through `useCreateTranscriptNote`.
- Clear selection and close composer after successful save.
- Open the sidebar after successful save when it is collapsed.

Acceptance criteria:

- Selecting transcript text and clicking note opens a floating composer.
- The composer saves a note tied to the selected quote and anchor.
- Empty notes cannot be submitted.
- Escape cancels without creating an annotation.
- Note creation does not trigger playback-time re-renders across the transcript.

## Sprint 3: Collapsible Notes Sidebar

Goal: display saved note annotations in a right sidebar without user metadata.

Tasks:

- Add `TranscriptNotesSidebar`.
- Add `TranscriptNoteItem`.
- Add parent layout state for sidebar open/collapsed.
- Fetch notes from the existing annotations query data.
- Render note quote, content, clickable timestamp, stale status if needed, and optional delete action.
- Add desktop docked layout and mobile drawer/sheet behavior.
- Add accessible toggle controls and keyboard reachable note actions.
- Keep the sidebar list independently scrollable.

Acceptance criteria:

- Saved notes appear after creation and after page refresh.
- Sidebar can collapse and reopen without losing transcript/player state.
- No user identity or author metadata is displayed.
- Empty, loading, error, and stale-note states are explicit but quiet.
- Long quotes and note content wrap without overlapping controls.

## Sprint 4: Timestamp Seek Integration

Goal: make note timestamps seek the audio player smoothly.

Tasks:

- Add a parent-owned seek command boundary in `AudioDetailView`.
- Update `StreamingAudioPlayer` to accept external seek requests without exposing its audio ref.
- Wire note timestamp buttons to seek to `anchor.start_ms / 1000`.
- Clamp seek requests to valid duration.
- Publish playback sync updates immediately after external seeks.
- Add focus behavior so keyboard users can seek repeatedly from the notes list.

Acceptance criteria:

- Clicking a note timestamp seeks the visible player to that time.
- Seeking from notes works while audio is playing and paused.
- Karaoke/current-word highlighting updates after seek.
- The sidebar does not import or own audio element refs.

## Sprint 5: Note Thread Backend and Reply API

Goal: support multiple note entries attached to the same highlighted transcript anchor without duplicating annotation anchors.

Tasks:

- Add a durable note-entry persistence model associated with a root note annotation.
- Add the note-entry table and indexes without backfilling old root-level note content.
- Add repository methods for:
  - creating a note entry under a root note annotation,
  - listing note annotations with entries,
  - soft-deleting an entry if delete support is included.
- Add annotation service methods that enforce:
  - the parent annotation belongs to the requested transcription,
  - the parent annotation is `kind: "note"`,
  - empty reply content is rejected,
  - deleted roots cannot receive new entries unless the product explicitly restores them.
- Add typed HTTP routes under the existing annotation boundary, for example:

```http
POST   /api/v1/transcriptions/{id}/annotations/{annotation_id}/entries
PATCH  /api/v1/transcriptions/{id}/annotations/{annotation_id}/entries/{entry_id}
DELETE /api/v1/transcriptions/{id}/annotations/{annotation_id}/entries/{entry_id}
```

- Update API response mappers so note annotations include ordered entries.
- Update OpenAPI/route contract tests.
- Publish annotation invalidation events when entries change.

Acceptance criteria:

- A root note annotation can contain multiple persisted entries.
- New note annotations create a first entry and keep root annotation `content` empty/null.
- Listing annotations returns note entries in stable chronological order.
- Reply creation uses the existing transcription/annotation authorization boundary.
- Backend tests cover success, empty content, wrong transcription, non-note parent, deleted parent, and ordering.

## Sprint 6: Sidebar Reply Composer and Thread Rendering

Goal: make the right-sidebar reply bubble create additional note entries on the same quote and timestamp.

Tasks:

- Add frontend API helpers and TanStack mutations for note entries.
- Render note annotations from their `entries` array.
- Update `TranscriptNoteItem` to render multiple compact bubbles under one quote and timestamp.
- Convert the reply row into a real compact input with submit, loading, Escape cancel, and Cmd/Ctrl+Enter submit behavior.
- Keep the reply composer locally scoped to one note thread at a time.
- Update the count chip to show the number of visible note entries for that thread.
- Invalidate only the active transcription annotation query after entry mutations.
- Keep note entries free of user identity UI.

Acceptance criteria:

- Typing in the reply bubble and submitting adds another note bubble under the same highlighted quote.
- The timestamp and quote are not duplicated for each reply.
- The note count updates immediately after mutation success.
- Replies persist after refresh.
- Empty replies cannot be submitted.
- Playback ticks do not re-render inactive note entry rows.

## Sprint 7: Polish, Performance, and Regression Coverage

Goal: harden the workflow for smooth reading, selecting, saving, listing, and seeking.

Tasks:

- Extract dense pieces from `AudioDetailView.tsx` if needed to keep responsibilities clear.
- Memoize note sorting and sidebar item rendering by annotation data.
- Ensure playback time ticks do not re-render the notes sidebar.
- Add focused regression tests for:
  - note selection payload creation,
  - create-note mutation payload shape,
  - note sorting,
  - note thread grouping and entry ordering,
  - reply mutation payload shape,
  - timestamp formatting,
  - external seek clamping.
- Add component-level coverage for composer submit/cancel, sidebar reply submit, and sidebar timestamp click if the current test setup supports it.
- Verify desktop and mobile layouts in the browser.
- Run type-check, lint, and build.

Acceptance criteria:

- Transcript selection, highlight rendering, note creation, sidebar listing, and audio seeking all work together.
- Multiple entries on the same note thread render compactly and persist.
- Notes remain visible and correct after refresh.
- The UI remains responsive with many transcript segments and dozens of notes.
- No author/user information appears in note UI.
- Tests and build commands pass, with any pre-existing warnings documented in the tracker.

## Git Hygiene

Use small commits aligned to sprint boundaries:

```txt
Add note annotation hooks
Add transcript note composer
Add transcript notes sidebar
Wire note timestamp seeking
Add note thread reply backend
Add sidebar note replies
Polish transcript notes workflow
```

Before each commit:

- Review `git diff` for unrelated changes.
- Keep generated files out unless the command intentionally updates them.
- Run the smallest meaningful verification command for the sprint.
- Update `devnotes/v2.0.0/sprint-trackers/transcript-note-taking-frontend-sprint-tracker.md`.
