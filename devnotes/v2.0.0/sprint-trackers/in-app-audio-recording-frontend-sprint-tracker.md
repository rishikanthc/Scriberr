# Sprint Tracker: In-App Audio Recording Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/in-app-audio-recording-frontend-sprint-plan.md`.

Status: Sprint 6 static verification complete. Authenticated cross-browser microphone QA remains pending because this environment cannot pass the app login and browser permission boundary.

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

Status: complete

Progress:

- [x] Invalidate file, audio-list, and recording queries after stop/finalization events.
- [x] Add an optimistic active/finalizing row only when the finalized file does not exist yet.
- [x] Replace optimistic state with the normal `file_...` row after finalization.
- [x] Keep tagged pages scoped to tagged server files/transcriptions.
- [x] Surface finalization failure and retry without adding a normal ready row.

Verification:

- [x] Home list reactive path reviewed through provider optimistic summary, recording SSE invalidation, and file-id replacement logic.
- [x] Tagged page behavior remains scoped by excluding local optimistic recording rows when `tagId` is present.
- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.

Artifacts:

- `web/frontend/src/features/home/components/HomePage.tsx`
- `web/frontend/src/features/files/hooks/useFiles.ts`
- `web/frontend/src/features/files/hooks/useFileEvents.ts`
- `web/frontend/src/features/recording/hooks/useRecordingEvents.ts`

Notes:

- `RecordingProvider` now mounts `useRecordingEvents()` so recording events invalidate recording/file/audio-list queries outside the home route.
- The home list prepends an optimistic `rec_...` row for active, paused, finalizing, and failed local recordings while no finalized file row exists.
- The optimistic row is omitted on tagged pages and is removed once the finalized `file_...` appears in the files query.
- `npm run build` passed with existing dependency-data freshness warnings for Browserslist/baseline-browser-mapping.

## Sprint 6: Cross-Browser QA, Accessibility, and Performance

Status: partially complete

Progress:

- [ ] Verify Chrome microphone recording flow.
- [ ] Verify Firefox microphone recording flow.
- [ ] Verify Safari microphone recording flow.
- [x] Review MIME fallback strategy for Chrome, Firefox, and Safari.
- [x] Verify mobile and desktop layouts.
- [x] Verify keyboard and screen-reader accessible controls.
- [x] Verify recording state does not cause broad home/audio rerenders.
- [x] Update tracker with artifacts and residual risks.

Verification:

- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.
- [x] `npm run lint` from `web/frontend`.
- [x] Focused backend API tests if a contract mismatch is found. No new backend contract mismatch was found in this sprint.

Artifacts:

- `web/frontend/src/features/recording/hooks/useRecordingController.ts`
- `web/frontend/src/features/recording/components/RecordingProvider.tsx`
- `web/frontend/src/features/home/components/HomePage.tsx`

Browser QA notes:

- Local Vite server started at `http://127.0.0.1:5174/` for a browser smoke attempt.
- The in-app browser reached the Scriberr frontend and loaded the app title, but the protected route stopped at the sign-in screen.
- Authenticated microphone interaction could not be completed in this environment without entering user credentials and accepting browser microphone permission. Those steps need manual QA in Chrome, Firefox, and Safari.
- Static cross-browser support was reviewed through runtime MIME selection with `MediaRecorder.isTypeSupported()` and supported-constraint negotiation through `navigator.mediaDevices.getSupportedConstraints()`.
- Mobile and desktop layout were reviewed statically through constrained dialog sizing, wrapping controls, and the compact minimized-sidebar entry at `max-width: 760px`.
- Keyboard/accessibility was reviewed statically: dialog uses Radix dialog semantics, title input has a label, controls are real buttons with visible text or explicit labels, and the minimized entry has an accessible name.
- Performance was reviewed and adjusted: the provider context no longer exports live recorder state, so home consumers avoid timer-driven rerenders; only low-churn optimistic summary changes reach the home list.
- `npm run lint` exits successfully. Remaining warnings are pre-existing outside the recording feature: `SettingsPage.tsx`, `AudioTagSection.tsx`, `AudioDetailView.tsx`, `ReadOnlyMarkdown.tsx`, and `TranscriptChatPanel.tsx`.
- `npm run build` passes with existing dependency-data freshness warnings for Browserslist/baseline-browser-mapping.

Residual risks:

- Full page reload during recording cannot preserve the live browser media stream; the implementation should warn before unload.
- Exact MIME output and audio-processing constraints remain browser/device dependent and must be runtime-detected.
- Manual authenticated microphone QA remains required in Chrome, Firefox, and Safari because this environment could not pass the app login and browser permission boundary.
