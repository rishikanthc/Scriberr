# Sprint Tracker: Chat With Transcripts Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/chat-with-transcripts-frontend-sprint-plan.md`.

Status: Sprint 4 verification complete; Sprint 5 polish in progress.

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

Status: complete

Progress:

- [x] Stream chat messages over SSE.
- [x] Render user and assistant messages.
- [x] Render assistant Markdown with a single React Markdown path that supports streaming and completed responses.
- [x] Keep reasoning separate and collapsible.
- [x] Refresh persisted queries after run completion/failure.
- [x] Show optimistic user messages and a visible generation status immediately after submit.
- [x] Parse SSE event boundaries for both LF and CRLF streams.

Verification:

- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.
- [x] Browser verified with `gemma4-4B`: prompt appears immediately, generation status appears, and markdown response content updates before completion.

## Sprint 5: Polish, Accessibility, and Verification

Status: partially complete

Progress:

- [ ] Verify desktop layout.
- [ ] Verify mobile layout.
- [ ] Confirm accessible names and keyboard reachability.
- [x] Confirm chat streaming does not disturb transcript/audio hot paths.
- [x] Run final frontend and backend checks.

Verification:

- [x] `npm run type-check` from `web/frontend`.
- [x] `npm run build` from `web/frontend`.
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run TestChatSessionContextAndStreamingLifecycle`
- [x] Browser verified locally on `http://127.0.0.1:5174/audio/file_a317bbdb4e2ff2924368dfff3ac583a3`.
