# Sprint Tracker: In-App Audio Recording Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/in-app-audio-recording-frontend-sprint-plan.md`.

Status: Sprint 4 complete. Sprint 5 has not started.

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

Status: complete

Progress:

- [x] Add `useBrowserRecorder` state machine.
- [x] Add runtime MIME fallback selection.
- [x] Add supported audio constraint negotiation.
- [x] Add sequential chunk upload queue and retry handling.
- [x] Add monotonic active-duration timer.
- [x] Add unload protection while recording or chunks are unsent.

Verification:

- [x] Hook/state machine implementation reviewed at the API boundary; browser smoke is deferred until Sprint 3 mounts the hook in the recorder dialog.
- [x] `npm run type-check` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/recording/hooks/useBrowserRecorder.ts`
- `web/frontend/src/features/recording/utils/mediaRecorderSupport.ts`
- `web/frontend/src/features/recording/utils/recordingDuration.ts`

Notes:

- `useBrowserRecorder` owns the local browser workflow state, stream lifecycle, `MediaRecorder` instance, chunk queue, retry queue, timer, unload guard, and backend create/upload/stop/cancel mutations.
- MIME type and audio constraints are runtime-selected so Chrome, Firefox, and Safari can choose their supported encoder and device features.
- Cancel/unmount disables chunk acceptance before stopping tracks so explicit cancellation does not upload a trailing `dataavailable` blob.

## Sprint 3: Recorder Dialog and Header Entry

Status: complete

Progress:

- [x] Wire the revamped home top bar Record button.
- [x] Add feature-owned recording dialog.
- [x] Add default `recording-YYYYMMDD-HHmmss` title generation.
- [x] Add start, pause/resume, stop, cancel, and retry controls.
- [x] Make outside click and escape minimize active recordings.
- [x] Add accessible names/tooltips and keyboard behavior.

Verification:

- [x] Static desktop/mobile layout review through responsive classes and constrained dialog sizing.
- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/recording/components/RecordingDialog.tsx`
- `web/frontend/src/features/home/components/HomePage.tsx`

Notes:

- Dismissing an active recording now hides the dialog without stopping the browser recorder. Sprint 4 will add the visible minimized left-sidebar entry for reopening it from navigation.
- The dialog uses feature-owned recorder state, keeps title fallback local, and exposes explicit start, pause, resume, stop, cancel, retry, and done actions.
- `npm run build` passed with existing dependency-data freshness warnings for Browserslist/baseline-browser-mapping.

## Sprint 4: Background Recording and Sidebar Minimize

Status: complete

Progress:

- [x] Add provider-level recording workflow state.
- [x] Add minimized recording item in the left sidebar.
- [x] Reopen the same dialog from the sidebar item.
- [x] Keep timer live while minimized.
- [x] Preserve recording across in-app navigation.
- [x] Support cancel/retry from minimized or reopened states.

Verification:

- [x] Provider placement reviewed against route boundaries so recorder state survives route changes under `ProtectedRoute`.
- [x] Static sidebar minimize/reopen behavior reviewed; full browser interaction smoke remains in Sprint 6.
- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/recording/components/RecordingProvider.tsx`
- `web/frontend/src/features/recording/components/RecordingSidebarItem.tsx`
- `web/frontend/src/features/home/components/HomePage.tsx`

Notes:

- `RecordingProvider` now owns the recorder hook, dialog open state, and minimized entry outside route components.
- The home top-bar Record button only calls `openDialog()`, avoiding timer-driven home-page rerenders.
- The minimized recording entry is fixed to the left side and remains available across route changes while recording, paused, stopping, finalizing, or failed.
- `npm run build` passed with existing dependency-data freshness warnings for Browserslist/baseline-browser-mapping.

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
