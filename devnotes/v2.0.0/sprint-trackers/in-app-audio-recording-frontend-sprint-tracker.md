# Sprint Tracker: In-App Audio Recording Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/in-app-audio-recording-frontend-sprint-plan.md`.

Status: Sprint 1 complete. Sprint 2 has not started.

## Sprint 1: API Contract and Query Foundation

Status: complete

Progress:

- [x] Add typed recording API helpers.
- [x] Add recording response/status types.
- [x] Add recording query keys and mutations.
- [x] Add recording event invalidation hook.
- [x] Review focused API/helper test need; no dedicated frontend API test harness exists for this slice, so coverage is currently type-check plus integration-oriented follow-up sprints.

Verification:

- [x] `npm run type-check` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/recording/api/recordingsApi.ts`
- `web/frontend/src/features/recording/hooks/useRecordingSession.ts`
- `web/frontend/src/features/recording/hooks/useRecordingEvents.ts`

Notes:

- The cancel command currently returns a minimal `{ id, status }` response from the backend, so the frontend updates known session cache state and invalidates the list instead of treating cancel as a full session response.
- Recording events update known recording caches and invalidate `filesQueryKey` plus `["audioFiles"]` when finalization produces a file signal.

## Sprint 2: Browser Recorder Engine

Status: pending

Progress:

- [ ] Add `useBrowserRecorder` state machine.
- [ ] Add runtime MIME fallback selection.
- [ ] Add supported audio constraint negotiation.
- [ ] Add sequential chunk upload queue and retry handling.
- [ ] Add monotonic active-duration timer.
- [ ] Add unload protection while recording or chunks are unsent.

Verification:

- [ ] Hook/state machine tests or browser smoke notes.
- [ ] `npm run type-check` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/recording/hooks/useBrowserRecorder.ts`
- `web/frontend/src/features/recording/utils/mediaRecorderSupport.ts`
- `web/frontend/src/features/recording/utils/recordingDuration.ts`

## Sprint 3: Recorder Dialog and Header Entry

Status: pending

Progress:

- [ ] Wire the revamped home top bar Record button.
- [ ] Add feature-owned recording dialog.
- [ ] Add default `recording-YYYYMMDD-HHmmss` title generation.
- [ ] Add start, pause/resume, stop, cancel, and retry controls.
- [ ] Make outside click and escape minimize active recordings.
- [ ] Add accessible names/tooltips and keyboard behavior.

Verification:

- [ ] Desktop dialog smoke test.
- [ ] Mobile dialog smoke test.
- [ ] `npm run type-check` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/recording/components/RecordingDialog.tsx`
- `web/frontend/src/features/home/components/HomePage.tsx`

## Sprint 4: Background Recording and Sidebar Minimize

Status: pending

Progress:

- [ ] Add provider-level recording workflow state.
- [ ] Add minimized recording item in the left sidebar.
- [ ] Reopen the same dialog from the sidebar item.
- [ ] Keep timer live while minimized.
- [ ] Preserve recording across in-app navigation.
- [ ] Support cancel/retry from minimized or reopened states.

Verification:

- [ ] Route-navigation recording smoke test.
- [ ] Sidebar minimize/reopen smoke test.
- [ ] `npm run type-check` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/recording/components/RecordingProvider.tsx`
- `web/frontend/src/features/recording/components/RecordingSidebarItem.tsx`
- `web/frontend/src/features/home/components/HomePage.tsx`

## Sprint 5: Reactive Home List Integration

Status: pending

Progress:

- [ ] Invalidate file, audio-list, and recording queries after stop/finalization events.
- [ ] Add an optimistic active/finalizing row only when the finalized file does not exist yet.
- [ ] Replace optimistic state with the normal `file_...` row after finalization.
- [ ] Keep tagged pages scoped to tagged server files/transcriptions.
- [ ] Surface finalization failure and retry without adding a normal ready row.

Verification:

- [ ] Home list updates without refresh after recording finalization.
- [ ] Tagged page behavior verified.
- [ ] `npm run type-check` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/home/components/HomePage.tsx`
- `web/frontend/src/features/files/hooks/useFiles.ts`
- `web/frontend/src/features/files/hooks/useFileEvents.ts`
- `web/frontend/src/features/recording/hooks/useRecordingEvents.ts`

## Sprint 6: Cross-Browser QA, Accessibility, and Performance

Status: pending

Progress:

- [ ] Verify Chrome microphone recording flow.
- [ ] Verify Firefox microphone recording flow.
- [ ] Verify Safari microphone recording flow.
- [ ] Verify MIME fallback in each browser.
- [ ] Verify mobile and desktop layouts.
- [ ] Verify keyboard and screen-reader accessible controls.
- [ ] Verify recording state does not cause broad home/audio rerenders.
- [ ] Update tracker with artifacts and residual risks.

Verification:

- [ ] `npm run type-check` from `web/frontend`.
- [ ] `npm run build` from `web/frontend`.
- [ ] Focused backend API tests if a contract mismatch is found.

Artifacts:

- Browser QA notes to be added under this sprint.

Residual risks:

- Full page reload during recording cannot preserve the live browser media stream; the implementation should warn before unload.
- Exact MIME output and audio-processing constraints remain browser/device dependent and must be runtime-detected.
