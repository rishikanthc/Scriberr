# Sprint Tracker: Transcript Click-to-Seek

This tracker belongs to `devnotes/v2.0.0/sprint-plans/transcript-click-to-seek-sprint-plan.md`.

Status: planned.

## Sprint 1: Transcript Data and Rendering Audit

Status: pending

Progress:

- [ ] Inspect current transcript payload types and word-level timing availability.
- [ ] Identify the current audio player seek API and owner component boundary.
- [ ] Identify current transcript segment rendering and text normalization.
- [ ] Document any offset-shifting display transformations.
- [ ] Decide the minimal component/hook boundary for event delegation.

Verification:

- [ ] Architecture notes captured in tracker or implementation notes.

## Sprint 2: Word Timing Index Foundation

Status: pending

Progress:

- [ ] Add typed word seek target/index utilities.
- [ ] Build segment-local character ranges from displayed text and word timings.
- [ ] Implement binary search lookup by character offset.
- [ ] Cover whitespace, punctuation, repeated words, and missing timestamps.
- [ ] Add unit tests for index construction and lookup.

Verification:

- [ ] Unit tests for word seek index.
- [ ] `npm run type-check` from `web/frontend`.

## Sprint 3: Browser Caret Hit Testing Utility

Status: pending

Progress:

- [ ] Add Firefox `caretPositionFromPoint` support.
- [ ] Add Chromium/WebKit `caretRangeFromPoint` fallback.
- [ ] Resolve clicked text node, offset, and containing segment.
- [ ] Reject clicks outside transcript text and inside interactive controls.
- [ ] Add targeted tests where practical.

Verification:

- [ ] Browser verification for real caret API behavior.
- [ ] `npm run type-check` from `web/frontend`.

## Sprint 4: Click-to-Seek Hook Integration

Status: pending

Progress:

- [ ] Add `useTranscriptClickSeek` with container-level delegation.
- [ ] Wire hook into transcript view.
- [ ] Seek through the existing player controller/ref boundary.
- [ ] Preserve existing paused/playing behavior.
- [ ] Prevent accidental seeks during text selection or drag gestures.

Verification:

- [ ] Browser verification on audio detail page.
- [ ] `npm run type-check` from `web/frontend`.

## Sprint 5: UX Polish and Hot Path Protection

Status: pending

Progress:

- [ ] Add cursor/clickable affordance only over transcript text.
- [ ] Confirm audio time updates do not rebuild indexes.
- [ ] Confirm rendered DOM scales by segment, not word count.
- [ ] Verify desktop and mobile pointer/tap behavior.
- [ ] Confirm notes, chat, transcript menus, links, and speaker controls remain unaffected.

Verification:

- [ ] Browser verification on desktop.
- [ ] Browser verification on mobile viewport.
- [ ] Performance review for long transcripts.

## Sprint 6: Verification and Cleanup

Status: pending

Progress:

- [ ] Run frontend type-check/build.
- [ ] Run unit tests for seek index and hit-testing utilities.
- [ ] Verify click-to-seek with punctuation, multiple speakers, long segments, and missing word timings.
- [ ] Remove exploratory/debug instrumentation.
- [ ] Update this tracker with completed artifacts and residual risks.

Verification:

- [ ] `npm run type-check` from `web/frontend`.
- [ ] `npm run build` from `web/frontend`.
- [ ] Relevant unit test command.
- [ ] Browser verification summary.
