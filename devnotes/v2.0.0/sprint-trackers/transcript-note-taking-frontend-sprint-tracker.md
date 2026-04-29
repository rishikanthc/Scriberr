# Sprint Tracker: Transcript Note Taking Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/transcript-note-taking-frontend-sprint-plan.md`.

Status: Sprint 6 complete.

## Sprint 1: Note API Helpers and Hook Mutations

Status: complete

Progress:

- [x] Add `CreateTranscriptNoteRequest` and note helper types.
- [x] Add note create mutation through the canonical annotation route.
- [x] Defer update mutation because edit support is not included in Sprint 1.
- [x] Generalize or add note delete mutation.
- [x] Add note selector for all non-deleted note annotations returned by the API.
- [x] Keep annotation query invalidation scoped to the active transcription.

Verification:

- [x] `npm run type-check` from `web/frontend`

## Sprint 2: Floating Note Composer

Status: complete

Progress:

- [x] Enable the note action in `TranscriptSelectionMenu`.
- [x] Add `TranscriptNoteComposer`.
- [x] Freeze selection payload when the composer opens.
- [x] Position composer with clamping and flip behavior.
- [x] Add submit, cancel, keyboard shortcuts, and loading state.
- [x] Save `kind: "note"` annotations.
- [x] Clear selection after successful save.
- [x] Defer auto-opening notes sidebar until the sidebar is added in Sprint 3.

Verification:

- [x] `npm run type-check` from `web/frontend`

## Sprint 3: Collapsible Notes Sidebar

Status: complete

Progress:

- [x] Add `TranscriptNotesSidebar`.
- [x] Add `TranscriptNoteItem`.
- [x] Add parent open/collapsed sidebar state.
- [x] Render notes from annotation query data.
- [x] Show quote, note content, and timestamp.
- [x] Defer timestamp seek behavior to Sprint 4.
- [x] Ensure no user identity or author metadata is displayed.
- [x] Add desktop docked layout and mobile fixed-panel behavior.
- [x] Correct sidebar ownership so it renders as a layout-level right rail beside the full audio detail view, not inside the transcript tab.

Verification:

- [x] `npm run type-check` from `web/frontend`
- [x] `npm run build` from `web/frontend`

## Sprint 4: Timestamp Seek Integration

Status: complete

Progress:

- [x] Add parent-owned seek command boundary.
- [x] Update `StreamingAudioPlayer` to consume external seek requests.
- [x] Wire note timestamp buttons to `anchor.start_ms`.
- [x] Clamp seek requests to known duration.
- [x] Publish playback sync after external seeks.
- [x] Verify seek works from the notes sidebar.

Verification:

- [x] `npm run type-check` from `web/frontend`
- [x] Browser verification: clicking a note timestamp updated the visible player time from `0:00` to `2:04`.

## Sprint 5: Note Thread Backend and Reply API

Status: complete

Progress:

- [x] Add durable note-entry persistence model for root note annotations.
- [x] Add note-entry table and indexes without backfilling old root-level note content.
- [x] Add repository methods for creating, listing, updating, and deleting note entries.
- [x] Add annotation service validation for note-entry commands.
- [x] Add entry routes under the existing transcription annotation boundary.
- [x] Update response mappers so note annotations include ordered entries.
- [x] Update OpenAPI and route contract tests.
- [x] Publish annotation invalidation events when entries change.
- [x] Cover success, empty content, wrong transcription, non-note parent, deleted parent, and ordering.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/repository ./internal/annotations`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestAnnotation|TestAnnotations|TestCanonicalRouteRegistration|TestAPIDocsContainOnlyCanonicalRoutes|TestSSEReceivesAnnotationEvents'`
- [x] `npm run type-check` from `web/frontend`

## Sprint 6: Sidebar Reply Composer and Thread Rendering

Status: complete

Progress:

- [x] Add frontend API helpers and mutations for note entries.
- [x] Render note annotations from their `entries` array.
- [x] Render multiple compact bubbles under one quote and timestamp.
- [x] Convert reply row into a real compact input.
- [x] Add submit, loading, Escape cancel, and Cmd/Ctrl+Enter behavior.
- [x] Keep reply composition scoped to one note thread at a time.
- [x] Update count chip from visible note-entry count.
- [x] Invalidate only the active transcription annotation query after entry mutation.
- [x] Ensure no user identity or author metadata is displayed for replies.

Verification:

- [x] `npm run type-check` from `web/frontend`
- [x] Browser verification for adding multiple notes to one highlighted timestamp.

## Sprint 7: Polish, Performance, and Regression Coverage

Status: pending

Progress:

- [ ] Extract dense `AudioDetailView.tsx` pieces if needed.
- [ ] Memoize note sorting and sidebar rendering.
- [ ] Confirm playback ticks do not re-render sidebar note rows.
- [ ] Add focused regression tests for payloads, sorting, timestamp formatting, and seek clamping.
- [ ] Add focused regression tests for note thread grouping, entry ordering, and reply payloads.
- [ ] Verify desktop layout.
- [ ] Verify mobile layout.
- [ ] Run type-check, lint, and build.

Verification:

- [ ] `npm run type-check` from `web/frontend`
- [ ] `npm run lint` from `web/frontend`
- [ ] `npm run build` from `web/frontend`
