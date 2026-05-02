# In-App Audio Recording Frontend Sprint Plan

This plan integrates the completed in-app audio recording backend with the revamped home frontend.

Related plans:

- `devnotes/v2.0.0/sprint-plans/in-app-audio-recording-backend-sprint-plan.md`
- `devnotes/v2.0.0/sprint-trackers/in-app-audio-recording-backend-sprint-tracker.md`

Architecture rules:

- `devnotes/v2.0.0/rules/react-architecture-rules.md`
- `devnotes/v2.0.0/rules/design-system-philosophy.md`

Browser API references:

- `MediaRecorder.isTypeSupported()` should be used to pick the best recording MIME type at runtime because supported containers/codecs differ across Chrome, Firefox, and Safari: https://developer.mozilla.org/en-US/docs/Web/API/MediaRecorder/isTypeSupported_static
- `navigator.mediaDevices.getSupportedConstraints()` should gate optional audio processing constraints such as noise suppression, echo cancellation, and automatic gain control: https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getSupportedConstraints
- `MediaTrackSettings.echoCancellation` can verify which constraints the browser actually applied: https://developer.mozilla.org/en-US/docs/Web/API/MediaTrackSettings/echoCancellation

## Product Goal

Users can start a high-quality microphone recording from the home header, continue using Scriberr while the recording continues in the background, and see the finalized recording appear in the home audio list without refreshing.

Required behavior:

- Add a Record button next to the existing Import button in the revamped home top bar.
- Clicking Record opens a recorder dialog with title, live timer, start, pause/resume, and stop controls.
- Default titles use a stable `recording-YYYYMMDD-HHmmss` pattern.
- Recording continues when the dialog is dismissed by outside click or escape.
- Dismissing while recording minimizes the recorder to the left sidebar instead of stopping it.
- Clicking the minimized sidebar item reopens the recorder dialog with the same live recording state.
- Stop completes the recording, uploads the final chunk metadata, requests backend finalization, and shows progress while finalization creates the normal Scriberr audio file.
- The home audio list updates reactively when the new file is created or becomes ready.
- Permission, unsupported-browser, denied-permission, upload-retry, and finalization-failed states are explicit.
- Recording should work in current Chrome, Firefox, and Safari where `getUserMedia` and `MediaRecorder` are available.
- Prefer high-quality browser encoding and enable echo cancellation, noise suppression, and automatic gain control when supported by the browser/device.

Out of scope:

- System/tab audio recording. Browser support is uneven and should remain separate from this microphone workflow.
- Live transcription while recording.
- Waveform editing, trimming, or post-recording cleanup.
- Offline persistence across full page reloads while a recording is active.
- Multiple simultaneous recordings.

## Target Frontend Model

Keep browser recording under a dedicated recording feature boundary:

```txt
features/recording/api/recordingsApi.ts
features/recording/hooks/useBrowserRecorder.ts
features/recording/hooks/useRecordingSession.ts
features/recording/hooks/useRecordingEvents.ts
features/recording/components/RecordingDialog.tsx
features/recording/components/RecordingSidebarItem.tsx
features/recording/components/RecordingProvider.tsx
features/home/components/HomePage.tsx
```

Feature placement:

- Browser media lifecycle, timer state, chunk buffering, pause/resume, MIME selection, and permission handling live in `features/recording`.
- Backend recording requests live in `features/recording/api`.
- TanStack Query mutations and invalidation live in `features/recording/hooks`.
- The home page owns only entry points: opening the dialog from the top bar and rendering the minimized sidebar item.
- Shared UI primitives come from `components/ui` or `shared/ui`; avoid new root-level recorder components.

Server state rules:

- Recording sessions are backend state and use typed API helpers plus TanStack Query mutations.
- The in-progress `MediaRecorder` instance is client workflow state and belongs to a provider/hook with complete cleanup.
- Chunk uploads use idempotent indexes and optional SHA-256 headers.
- Recording events and file events invalidate the smallest useful query keys: recording session, `filesQueryKey`, and `["audioFiles"]`.
- Queries remain the source of truth after refresh; optimistic UI is only for the active local recording and upload/finalization progress.

## Backend Contract

Use the completed canonical recording endpoints:

- `POST /api/v1/recordings` creates a session.
- `PUT /api/v1/recordings/{id}/chunks/{chunk_index}` uploads each MediaRecorder chunk.
- `POST /api/v1/recordings/{id}:stop` declares the final chunk and starts durable finalization.
- `POST /api/v1/recordings/{id}:cancel` cancels abandoned local recordings.
- `GET /api/v1/recordings/{id}` polls or refreshes finalization state when needed.

The frontend should not upload completed recordings through legacy `/api/v1/transcription/upload`. Finalized backend recording sessions create normal Scriberr file rows, which the existing home list can display through the files query.

## Browser Recording Strategy

MIME selection:

- Build a preference list and choose the first supported type with `MediaRecorder.isTypeSupported`.
- Prefer Opus-based WebM where supported, then Ogg Opus for Firefox if available, then MP4/AAC-oriented choices for Safari, then browser default.
- Persist the exact `mediaRecorder.mimeType || selectedMimeType || blob.type` on the created session and each chunk.

Audio constraints:

- Start from `navigator.mediaDevices.getSupportedConstraints()`.
- Request only supported constraints to avoid browser-specific constraint failures.
- Prefer:
  - `echoCancellation: true`
  - `noiseSuppression: true`
  - `autoGainControl: true`
  - `channelCount: 1`
  - `sampleRate: 48000` when supported
- After stream creation, inspect `track.getSettings()` for applied settings and surface only meaningful status/errors to users.

Chunking and durability:

- Use a fixed MediaRecorder timeslice, likely 5 seconds, to avoid holding a long recording entirely in memory.
- Queue chunk uploads sequentially to preserve chunk order and simplify retry behavior.
- Keep chunk bytes only until upload succeeds.
- Stop waits for the final `dataavailable` event, uploads the final chunk, then calls `:stop` with final chunk index and duration.
- Failed chunk uploads keep the recorder in an explicit recoverable state until retry or cancel.

Timer:

- Use monotonic elapsed time from `performance.now()` plus accumulated active segments, not a naive interval counter.
- Pause freezes active elapsed time and resume continues from the accumulated duration.
- The visible timer updates while the dialog is open and while minimized.

Dialog/minimize behavior:

- Closing while idle or completed closes normally.
- Closing while recording or paused minimizes to the sidebar and leaves the stream/recorder alive.
- Stop/cancel are explicit controls; outside click and escape never stop an active recording.
- A `beforeunload` guard warns while active recording or unsent chunks exist.

## UI and Design Direction

The recorder should feel like an operational tool, not a marketing surface:

- Compact dialog using existing dialog/button/input primitives.
- Clear title field, status line, timer, and three primary controls.
- Start, pause/resume, and stop use Lucide icons with accessible labels.
- Stop is visually distinct but not oversized.
- Minimized sidebar item shows a small recording indicator, title, elapsed duration, and current state.
- Avoid waveform-heavy rendering in the first pass unless it is cheap and isolated.
- Do not add nested cards or decorative gradients.
- Mobile layout keeps controls reachable and avoids fixed widths.

## Sprint Runs

### Sprint 1: API Contract and Query Foundation

- Add typed recording API helpers for create, upload chunk, stop, cancel, get, and list.
- Add response/status string unions matching backend DTOs.
- Add `recordingsQueryKey` and mutations with scoped invalidation.
- Add a small recording event hook that handles `recording.*` events and invalidates recording/file queries.
- Add unit tests for API request construction and error parsing where frontend test infrastructure allows.

### Sprint 2: Browser Recorder Engine

- Add `useBrowserRecorder` for permission request, stream lifecycle, MIME selection, MediaRecorder setup, pause/resume/stop, timer calculation, and cleanup.
- Add supported-constraint negotiation for echo cancellation, noise suppression, AGC, channel count, and sample rate.
- Add chunk upload queue with sequential retry and checksum calculation.
- Add unload protection for active recordings and unsent chunks.
- Add explicit states for unsupported browser, permission denied, idle, permission-ready, recording, paused, stopping, finalizing, ready, failed, and canceled.

### Sprint 3: Recorder Dialog and Header Entry

- Wire the home top bar Record button to open the feature-owned recorder dialog.
- Build `RecordingDialog` with title, timer, permission state, start, pause/resume, stop, cancel, and retry controls.
- Generate the default `recording-YYYYMMDD-HHmmss` title when the dialog opens.
- Make outside click and escape minimize instead of stopping when active.
- Keep idle close behavior normal.
- Ensure controls are keyboard reachable and icon-only buttons have accessible labels/tooltips.

### Sprint 4: Background Recording and Sidebar Minimize

- Add `RecordingProvider` at the home shell level so recording state survives navigation within the app shell.
- Add `RecordingSidebarItem` to the left sidebar when a recording is active, paused, stopping, finalizing, or failed.
- Reopen the same dialog state from the sidebar item.
- Keep the timer live while minimized.
- Preserve active recording while users open audio detail, settings, or tag pages.
- Handle cancel and failed upload cleanup from minimized state.

### Sprint 5: Reactive Home List Integration

- Invalidate `filesQueryKey`, `["audioFiles"]`, and relevant recording queries after `:stop`, `recording.ready`, and `file.ready` events.
- Add an optimistic local row only when needed to show active/finalizing recordings before the file row exists.
- Replace the optimistic row with the finalized `file_...` row once file events/queries return it.
- Ensure tagged audio pages do not show active untagged recording rows.
- Make finalization failure visible and retryable without polluting the normal audio list.

### Sprint 6: Cross-Browser QA, Accessibility, and Performance

- Verify Chrome, Firefox, and Safari microphone permission, recording, pause/resume, stop, and finalization flows.
- Verify MIME fallback behavior in each browser.
- Verify outside click minimize, sidebar reopen, route navigation during recording, and unload guard.
- Verify mobile and desktop layouts.
- Run `npm run type-check` and `npm run build` from `web/frontend`.
- Run focused backend API tests if frontend integration exposes a contract mismatch.
- Update the sprint tracker with artifacts, browser results, and residual risks.

## Implementation Risks

- Safari MIME support differs from Chromium/Firefox, so the MIME selection must be runtime-detected.
- Some audio constraints are browser/device dependent; unsupported constraints must be omitted rather than forced.
- Browser background throttling can affect UI timer intervals; elapsed time should be derived from monotonic timestamps.
- Full page reloads cannot preserve a live `MediaRecorder`; warn before unload and keep this limitation explicit.
- Current legacy recorder components under `src/components` should not become the integration target; they can be retired or left untouched until follow-up cleanup.
