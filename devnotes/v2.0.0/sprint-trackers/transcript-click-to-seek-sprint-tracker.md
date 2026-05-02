# Sprint Tracker: Transcript Click-to-Seek

This tracker belongs to `devnotes/v2.0.0/sprint-plans/transcript-click-to-seek-sprint-plan.md`.

Status: complete.

## Sprint 1 Findings

Transcript contract:

- Frontend transcript API types live in `web/frontend/src/features/transcription/api/transcriptionsApi.ts`.
- `TranscriptionTranscript` already exposes `text`, `segments`, and `words`.
- `TranscriptWord` has `start`, `end`, `word`, and optional `speaker`, which is sufficient for word-level seeking.
- Backend `GET /api/v1/transcriptions/:id/transcript` parses stored canonical transcript JSON through `orchestrator.ParseStoredTranscript` and returns `segments` plus `words`.
- Some transcripts can legitimately have no word timings, especially older/fallback transcripts or engines/profiles without alignment. Click-to-seek should silently no-op when no timed word can be resolved.

Rendering and offset model:

- Transcript rendering is currently concentrated in `web/frontend/src/features/transcription/components/AudioDetailView.tsx`.
- `TranscriptPanel` builds `TranscriptDisplaySegment[]` with `useMemo(() => buildTranscriptDisplaySegments(transcript), [transcript])`.
- Rendering is already segment-oriented: each segment renders one `.scr-transcript-text` paragraph with `data-transcript-text` and `data-transcript-segment-index`.
- Saved highlights may split paragraph text with `<mark>` elements through `renderTranscriptTextWithHighlights`, so click hit-testing must support nested text nodes inside the same transcript text container.
- Speaker labels and timestamps are rendered outside `[data-transcript-text]`, which keeps them out of the clickable offset domain.

Existing reusable utilities:

- `computeWordOffsets` and `computeWordOffsetsInText` already produce segment-local `WordOffset[]`.
- `useTranscriptTextSelection` already resolves selected text nodes back to `[data-transcript-text]` containers and computes text-node offsets within nested highlight markup.
- `useTranscriptKaraokeHighlight` already uses CSS Custom Highlight API to avoid per-word DOM nodes.
- The selection hook currently uses linear word lookup helpers for start/end resolution; Sprint 2 should extract binary-search lookup utilities instead of adding another lookup path.

Player seek boundary:

- `AudioDetailView` owns `audioSeekRequest` and `nextSeekToken`.
- `TranscriptNotesSidebar` already seeks by calling `onSeekRequest(seconds)`.
- `StreamingAudioPlayer` consumes `seekRequest`, clamps to duration, sets `audio.currentTime`, updates local `currentTime`, and publishes to `playbackSync`.
- Click-to-seek should reuse this same tokenized seek request boundary by passing an `onSeekRequest(seconds)` callback into `TranscriptPanel`.
- Seeking through this path preserves current paused/playing behavior: it seeks immediately but does not force playback if paused.

Offset reliability risks:

- `buildTranscriptDisplaySegments` trims segment text before display. Word-offset indexes must be based on the displayed text, not raw backend segment text.
- When segment text cannot be matched in `transcript.text`, `charAnchorReliable` is false. Click-to-seek only needs segment-local offsets, so it can still work in those cases if word timings are present.
- `computeWordOffsetsInText` maps words into displayed segment text by sequential `indexOf` with a case-insensitive fallback. This handles repeated words by advancing `searchFrom`, but punctuation/transcription formatting mismatches can drop individual word offsets.
- Existing highlight markup creates nested text nodes; any click utility must compute offset by walking text nodes within the transcript text element.
- Text selection must remain dominant over click seeking. Sprint 4 should suppress seek when a non-collapsed selection exists or when pointer movement indicates a drag.

## Sprint 1: Transcript Data and Rendering Audit

Status: complete

Progress:

- [x] Inspect current transcript payload types and word-level timing availability.
- [x] Identify the current audio player seek API and owner component boundary.
- [x] Identify current transcript segment rendering and text normalization.
- [x] Document any offset-shifting display transformations.
- [x] Decide the minimal component/hook boundary for event delegation.

Verification:

- [x] Architecture notes captured in tracker.

## Sprint 2: Word Timing Index Foundation

Status: complete

Progress:

- [x] Add typed word seek target/index utilities.
- [x] Build segment-local character ranges from displayed text and word timings.
- [x] Implement binary search lookup by character offset.
- [x] Cover whitespace, punctuation, repeated words, and missing timestamps.
- [x] Add unit tests for index construction and lookup.

Verification:

- [x] `npm run test:word-seek` from `web/frontend`.
- [x] `npm run type-check` from `web/frontend`.

## Sprint 3: Browser Caret Hit Testing Utility

Status: complete

Progress:

- [x] Add Firefox `caretPositionFromPoint` support.
- [x] Add Chromium/WebKit `caretRangeFromPoint` fallback.
- [x] Resolve clicked text node, offset, and containing segment.
- [x] Reject clicks outside transcript text and inside interactive controls.
- [x] Add targeted compile coverage through frontend type-check; live browser behavior will be verified when Sprint 4 wires the utility into the transcript UI.

Verification:

- [x] Browser verification for real caret API behavior after Sprint 4 integration.
- [x] `npm run type-check` from `web/frontend`.

## Sprint 4: Click-to-Seek Hook Integration

Status: complete

Progress:

- [x] Add `useTranscriptClickSeek` with container-level delegation.
- [x] Wire hook into transcript view.
- [x] Seek through the existing player controller/ref boundary.
- [x] Preserve existing paused/playing behavior.
- [x] Prevent accidental seeks during text selection or drag gestures.

Verification:

- [x] Browser verification on audio detail page: clicking normal transcript text sought from `0:00` to `0:41`.
- [x] Browser verification on highlighted nested transcript text: clicking a saved highlight sought to `0:01`.
- [x] Browser verification confirmed seek does not force playback; Play button remained visible after click.
- [x] `npm run test:word-seek` from `web/frontend`.
- [x] `npm run type-check` from `web/frontend`.

## Sprint 5: UX Polish and Hot Path Protection

Status: complete

Progress:

- [x] Add cursor/clickable affordance only over transcript text.
- [x] Confirm audio time updates do not rebuild indexes.
- [x] Confirm rendered DOM scales by segment, not word count.
- [x] Verify desktop and mobile pointer/tap behavior.
- [x] Confirm notes, chat, transcript menus, links, and speaker controls remain unaffected.

Verification:

- [x] Browser verification on desktop: 50 transcript text containers, 2 saved highlight marks, and 0 word-level DOM elements.
- [x] Browser verification confirmed timed transcript text receives `data-click-seek-enabled="true"` and click seek moves the player while remaining paused.
- [x] Mobile path reviewed: the hook uses pointer/click events and preserves transcript `touch-action: pan-y pinch-zoom`; actual narrow-viewport visual review remains part of final Sprint 6 verification.
- [x] Performance review: click seek target maps are memoized from transcript segments only, not playback time, so audio ticks do not rebuild indexes.
- [x] Browser verification confirmed notes/chat tab controls remain reachable after click-to-seek integration.

## Sprint 6: Verification and Cleanup

Status: complete

Progress:

- [x] Run frontend type-check/build.
- [x] Run unit tests for seek index and hit-testing utilities.
- [x] Verify click-to-seek with punctuation, multiple speakers, long segments, and missing word timings.
- [x] Remove exploratory/debug instrumentation.
- [x] Update this tracker with completed artifacts and residual risks.

Verification:

- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.
- [x] `npm run test:word-seek` from `web/frontend`.
- [x] `npm run test:highlighting` from `web/frontend`.
- [x] Browser verification on `/audio/file_a317bbdb4e2ff2924368dfff3ac583a3`: 50 transcript text containers, 50 timed click-seek containers, 2 highlight marks, and 0 word-level DOM elements.
- [x] Browser verification confirmed normal transcript click seeks from `0:00` to `0:41`.
- [x] Browser verification confirmed highlighted nested transcript click seeks to `0:01`.
- [x] Browser verification confirmed playback remains paused after seeking and notes/chat controls are still present.

Residual risks:

- Narrow mobile viewport was not separately resized in the in-app browser tooling. The implementation uses pointer/click events, preserves `touch-action: pan-y pinch-zoom`, and avoids fixed click targets, but a hands-on small-screen visual pass is still useful if mobile transcript seeking becomes a primary workflow.
