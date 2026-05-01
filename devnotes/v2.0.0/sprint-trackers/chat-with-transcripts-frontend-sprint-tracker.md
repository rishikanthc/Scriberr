# Sprint Tracker: Chat With Transcripts Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/chat-with-transcripts-frontend-sprint-plan.md`.

Status: Sprint 4 implementation in progress.

## Sprint 1: API Contract and Hook Foundation

Status: complete

Progress:

- [x] Add missing backend route registration needed by frontend chat streaming.
- [x] Add persisted message listing support for reopened sessions.
- [x] Add typed frontend chat API helpers.
- [x] Add chat query keys and hooks.
- [x] Add context and message mutations.
- [x] Add completed-transcript choices for the context picker.

Verification:

- [ ] Backend chat route tests.
- [x] `npm run type-check` from `web/frontend`.

## Sprint 2: Sidebar Shell and Session Navigation

Status: complete

Progress:

- [x] Add functional Chat/Notes tab switching.
- [x] Preserve notes sidebar behavior.
- [x] Add chat panel top bar.
- [x] Add session dropdown grouped by recency.
- [x] Add new chat session action.
- [x] Add provider-disabled state.

Verification:

- [x] `npm run type-check` from `web/frontend`.

## Sprint 3: Composer, Context Picker, and Model Picker

Status: complete

Progress:

- [x] Add plus-icon context picker.
- [x] List only completed transcript audio titles.
- [x] Add selected context bubbles with remove affordance.
- [x] Add model dropdown.
- [x] Disable composer when no provider/model is available.

Verification:

- [x] `npm run type-check` from `web/frontend`.

## Sprint 4: Streaming Messages and Markdown Rendering

Status: implementation complete; browser verification pending

Progress:

- [x] Stream chat messages over SSE.
- [x] Render user and assistant messages.
- [x] Render assistant Markdown with Textforge.
- [x] Keep reasoning separate and collapsible.
- [x] Refresh persisted queries after run completion/failure.

Verification:

- [x] `npm run type-check` from `web/frontend`.

## Sprint 5: Polish, Accessibility, and Verification

Status: pending

Progress:

- [ ] Verify desktop layout.
- [ ] Verify mobile layout.
- [ ] Confirm accessible names and keyboard reachability.
- [ ] Confirm chat streaming does not disturb transcript/audio hot paths.
- [ ] Run final frontend and backend checks.

Verification:

- [ ] `npm run type-check` from `web/frontend`.
- [ ] `npm run build` from `web/frontend`.
- [ ] Backend chat route tests.
