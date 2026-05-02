# Transcript Click-to-Seek Sprint Plan

This plan adds precise click-to-seek support for word-level transcripts without rendering every word as its own DOM element.

Related rules:

- `devnotes/v2.0.0/rules/react-architecture-rules.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/transcript-click-to-seek-sprint-tracker.md`

## Product Goal

Users can click a visible transcript word and the audio player seeks to that word's start timestamp. The interaction should feel immediate, accurate, and lightweight even for long transcripts.

Required behavior:

- Clicking transcript body text seeks the audio player to the clicked word's timestamp.
- Transcript rendering remains segment-oriented, not word-oriented.
- Word timing metadata is indexed outside the DOM.
- Hit testing uses browser caret/range APIs to map pointer position to text offset.
- Word lookup uses binary search over precomputed character ranges.
- The transcript remains selectable, accessible, searchable, and efficient.
- The implementation must avoid DOM explosion, per-word React components, and per-word event handlers.

Out of scope:

- Rendering every word as a `<span>` or clickable element.
- Canvas-based transcript rendering.
- Approximate x/y word estimation.
- Karaoke-style per-word highlighting changes beyond what already exists.
- Backend transcript schema changes unless the current payload lacks word-level timestamps.

## Target Interaction Model

Render transcript text at the segment level:

```txt
Transcript container
  Segment row
    Speaker/timestamp metadata
    Plain transcript text node
  Segment row
    Speaker/timestamp metadata
    Plain transcript text node
```

Keep the timing index outside the rendered text:

```ts
type WordSeekTarget = {
  segmentId: string;
  startChar: number;
  endChar: number;
  startMs: number;
  endMs?: number;
};
```

Click flow:

```txt
pointer click
  -> locate nearest text caret/range from x/y
  -> resolve containing segment
  -> convert DOM text offset to segment-local transcript text offset
  -> binary search segment word timing index
  -> seek player to word.startMs
  -> preserve existing playback state
```

Browser API strategy:

- Use `document.caretPositionFromPoint(x, y)` when available.
- Fall back to `document.caretRangeFromPoint(x, y)` for Chromium/WebKit.
- Guard against clicks on speaker labels, buttons, timestamps, menus, notes, links, and non-transcript UI.
- Keep the segment text used for display identical to the text used for character offset indexing.

## Frontend Architecture

Feature-owned implementation should live under `features/transcription`.

Preferred placement:

```txt
features/transcription/utils/wordSeekIndex.ts
features/transcription/utils/caretHitTesting.ts
features/transcription/hooks/useTranscriptClickSeek.ts
features/transcription/components/TranscriptSegmentText.tsx
```

Responsibilities:

- API/hook layer owns transcript normalization and exposes stable segment text plus word timings.
- `wordSeekIndex.ts` builds segment-local character ranges and exposes binary-search lookup.
- `caretHitTesting.ts` wraps browser-specific caret/range APIs behind a typed utility.
- `useTranscriptClickSeek` owns event delegation and calls the existing audio/player seek API.
- Segment text components render plain text with stable `data-segment-id` attributes.

Avoid:

- Direct `fetch` or server state in click-to-seek components.
- Global state for click offsets or hovered words.
- Recomputing indexes on every render or audio time tick.
- Adding listeners to each segment if container-level delegation is enough.
- Mixing speaker label text into the clickable transcript text offset domain.

## Performance Requirements

- DOM node count should scale with segments, not words.
- Index build should run only when transcript content changes.
- Click lookup should be near O(log n) within a segment.
- Current audio time updates must not rebuild seek indexes.
- Event handlers should be stable and scoped to the transcript container.
- Long transcripts should remain compatible with future virtualization/windowing.

Implementation notes:

- Prefer per-segment word arrays over one global array so lookup starts from the clicked segment.
- Store indexes in memoized maps keyed by stable segment IDs.
- Keep primitive props on segment render components.
- Use refs for imperative container/player integration.
- Add small instrumentation during development if needed to confirm index build time and click lookup cost.

## UX and Accessibility

- Cursor should indicate click-to-seek only over clickable transcript text.
- Text selection should remain possible; avoid seeking when the click is part of a drag selection.
- Keyboard transcript navigation can remain existing behavior for this sprint, but clickable text must not break focusable controls.
- Seek should be immediate and should not force playback if the audio was paused unless existing player behavior already does so.
- Mobile tap should use the same hit-testing path where supported.
- If a word has no valid timestamp, clicking it should do nothing silently.

## Sprint Runs

### Sprint 1: Transcript Data and Rendering Audit

- Inspect current transcript payload types and word-level timing availability.
- Identify the current audio player seek API and the component boundary that owns it.
- Identify where transcript segments are rendered and whether displayed text exactly matches transcript text.
- Document any normalization that could shift character offsets.
- Decide the minimal component/hook boundary for event delegation.

### Sprint 2: Word Timing Index Foundation

- Add typed word seek target/index utilities under `features/transcription`.
- Build segment-local character ranges from displayed segment text and word timing data.
- Implement binary search lookup by character offset.
- Cover whitespace, punctuation, repeated words, and missing timestamps.
- Add unit tests for index construction and lookup.

### Sprint 3: Browser Caret Hit Testing Utility

- Add a typed caret hit-testing utility that supports Firefox and Chromium/WebKit APIs.
- Resolve clicked text node, offset, and containing transcript segment.
- Reject clicks outside transcript text or inside interactive child controls.
- Handle nested harmless text nodes without requiring per-word spans.
- Add targeted tests where practical and keep browser-level verification for real caret APIs.

### Sprint 4: Click-to-Seek Hook Integration

- Add `useTranscriptClickSeek` with container-level event delegation.
- Wire the hook into the transcript view without moving server state out of existing query hooks.
- Seek through the existing player controller/ref boundary.
- Preserve paused/playing state according to current player behavior.
- Ensure text selection and drag gestures do not trigger accidental seeks.

### Sprint 5: UX Polish and Hot Path Protection

- Add subtle clickable affordance through cursor/style on transcript text only.
- Confirm audio time updates do not re-render the full transcript or rebuild indexes.
- Confirm long transcripts avoid word-level DOM growth.
- Verify desktop and mobile pointer/tap behavior.
- Check that notes, chat, transcript menus, links, and speaker controls remain unaffected.

### Sprint 6: Verification and Cleanup

- Run frontend type-check/build.
- Run unit tests for seek index and click-seek utilities.
- Browser-verify Firefox-style and Chromium-style hit testing where available.
- Verify click-to-seek on transcripts with punctuation, multiple speakers, long segments, and missing word timings.
- Remove any exploratory/debug instrumentation.
- Update the sprint tracker with completed work and residual risks.
