# Sprint Tracker: Transcript Highlighting Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/transcript-highlighting-frontend-sprint-plan.md`.

Status: Sprint 5 complete. Transcript highlighting frontend sprint set is complete.

## Sprint 1: API Boundary and Query Hooks

Status: complete

Progress:

- [x] Add `features/transcription/api/annotationsApi.ts`.
- [x] Add typed annotation response, list response, create request, and anchor contracts.
- [x] Add `listTranscriptAnnotations` and `createTranscriptAnnotation`.
- [x] Add `features/transcription/hooks/useTranscriptAnnotations.ts`.
- [x] Add query keys and create-highlight mutation.
- [x] Invalidate active transcription annotation queries from annotation SSE events.
- [x] Keep new code off the legacy direct-fetch note hooks.
- [x] Remove unreferenced legacy note and selection hooks that referenced old routes or selection assumptions.

Touched files:

- `web/frontend/src/features/transcription/api/annotationsApi.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptAnnotations.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptionDetailEvents.ts`
- `web/frontend/src/features/transcription/hooks/useSelectionMenu.ts` (removed)
- `web/frontend/src/features/transcription/hooks/useTranscriptSelection.ts` (removed)
- `web/frontend/src/features/transcription/hooks/useTranscriptionNotes.ts` (removed)
- `web/frontend/src/types/note.ts` (removed)
- `devnotes/v2.0.0/sprint-trackers/transcript-highlighting-frontend-sprint-tracker.md`

Verification:

- `npm run type-check` from `web/frontend`

## Sprint 2: Clean Transcript Selection and Anchor Builder

Status: complete

Progress:

- [x] Add a transcript-text-only selection hook.
- [x] Ensure speaker and timestamp UI cannot become valid highlight targets.
- [x] Parse DOM ranges only through `[data-transcript-text]` nodes.
- [x] Compute clean quote, timing, word indexes, char offsets, and text hash.
- [x] Reject metadata-crossing or unreliable selections.

Performance notes:

- Heavy range parsing runs on selection completion events (`mouseup`, `touchend`, `keyup`) rather than every `selectionchange` tick.
- `selectionchange` only performs cheap dismissal checks for collapsed or out-of-transcript selections.
- Typical single-paragraph selections walk the range common-ancestor subtree instead of the full transcript.
- Playback karaoke highlighting remains in the CSS Highlight API path and is not coupled to React selection state.

Touched files:

- `web/frontend/src/features/transcription/hooks/useTranscriptTextSelection.ts`
- `web/frontend/src/features/transcription/components/AudioDetailView.tsx`
- `web/frontend/src/styles/design-system.css`
- `devnotes/v2.0.0/sprint-trackers/transcript-highlighting-frontend-sprint-tracker.md`

Verification:

- `npm run type-check` from `web/frontend`

## Sprint 3: Floating Selection Menu

Status: complete

Progress:

- [x] Add `TranscriptSelectionMenu`.
- [x] Position it centered above selected text, with viewport clamping and below-selection fallback.
- [x] Add accessible icon buttons for highlight and deferred note/comment.
- [x] Wire highlight button to create-highlight mutation.
- [x] Clear selection and menu after successful creation.
- [x] Remove remaining unreferenced frontend hooks that used legacy transcription summary, speaker, or list routes.

Performance notes:

- The menu only mounts for a valid transcript text selection.
- Positioning is computed from the selected range rect and does not subscribe to playback state.
- Highlight creation invalidates only the active transcription annotation query through the Sprint 1 hook.

Touched files:

- `web/frontend/src/features/transcription/components/TranscriptSelectionMenu.tsx`
- `web/frontend/src/features/transcription/components/AudioDetailView.tsx`
- `web/frontend/src/features/transcription/hooks/useTranscriptTextSelection.ts`
- `web/frontend/src/features/transcription/hooks/useAudioFiles.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptionSpeakers.ts` (removed)
- `web/frontend/src/features/transcription/hooks/useTranscriptionSummary.ts` (removed)
- `web/frontend/src/styles/design-system.css`
- `devnotes/v2.0.0/sprint-trackers/transcript-highlighting-frontend-sprint-tracker.md`

Verification:

- `npm run type-check` from `web/frontend`

## Sprint 4: Persisted Highlight Rendering

Status: complete

Progress:

- [x] Fetch annotations for completed transcriptions.
- [x] Render active highlights inline inside transcript text.
- [x] Map annotation anchors to segment-local character ranges.
- [x] Handle overlapping highlights deterministically.
- [x] Keep stale annotations from confusing the reader.
- [x] Preserve transcript reading and karaoke highlighting behavior.

Performance notes:

- Annotation-to-segment range mapping is memoized by transcript segments and annotation response.
- Highlight rendering only splits text for segments with active persisted highlight ranges.
- Overlapping highlights merge into deterministic non-nested ranges.
- Karaoke highlighting now resolves text ranges across descendant text nodes so inline marks do not force React playback updates.
- Character anchors are emitted only when display text maps back to canonical `transcript.text`; otherwise selection falls back to word/time anchors to avoid false stale hashes.

Touched files:

- `web/frontend/src/features/transcription/components/AudioDetailView.tsx`
- `web/frontend/src/features/transcription/hooks/useKaraokeHighlight.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptTextSelection.ts`
- `web/frontend/src/styles/design-system.css`
- `devnotes/v2.0.0/sprint-trackers/transcript-highlighting-frontend-sprint-tracker.md`

Verification:

- `npm run type-check` from `web/frontend`

## Sprint 5: UX, Accessibility, and Regression Coverage

Status: complete

Progress:

- [x] Add focused no-dependency regression checks for hash normalization, menu positioning, overlapping highlight range merging, char-anchor rendering, and word-anchor rendering.
- [x] Check keyboard and assistive behavior for menu buttons.
- [x] Tune CSS so native selection, persistent highlight marks, and karaoke highlight are visually distinct but quiet.
- [x] Remove unused legacy note-selection paths if unreferenced.
- [x] Verify persisted highlight rendering in the in-app browser against the running backend.
- [x] Fix persisted rendering for highlights that save with word anchors but no reliable character anchors.

Notes:

- Browser native drag selection was not reliably reproducible through the automation surface, but persisted highlight rendering was verified in the in-app browser with the real backend.
- A local test highlight was created on the `sample` transcript to verify persisted rendering.

Touched files:

- `web/frontend/package.json`
- `web/frontend/tsconfig.highlighting-tests.json`
- `web/frontend/src/features/transcription/components/TranscriptSelectionMenu.tsx`
- `web/frontend/src/features/transcription/components/AudioDetailView.tsx`
- `web/frontend/src/features/transcription/hooks/useTranscriptTextSelection.ts`
- `web/frontend/src/features/transcription/utils/transcriptHighlighting.ts`
- `web/frontend/src/features/transcription/utils/transcriptHighlighting.regression.ts`
- `web/frontend/src/styles/design-system.css`
- `devnotes/v2.0.0/sprint-trackers/transcript-highlighting-frontend-sprint-tracker.md`

Verification:

- `npm run test:highlighting` from `web/frontend`
- `npm run type-check` from `web/frontend`
- `npm run lint` from `web/frontend` (passes with pre-existing warnings)
- `npm run build` from `web/frontend`
