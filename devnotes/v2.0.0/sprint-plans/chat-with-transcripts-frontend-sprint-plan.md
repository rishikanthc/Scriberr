# Chat With Transcripts Frontend Sprint Plan

This plan integrates the completed chat-with-transcripts backend into the transcript detail right sidebar.

Related plans:

- `devnotes/v2.0.0/sprint-plans/chat-with-transcripts-backend-sprint-plan.md`
- `devnotes/v2.0.0/sprint-plans/transcript-note-taking-frontend-sprint-plan.md`

Architecture rules:

- `devnotes/v2.0.0/rules/react-architecture-rules.md`
- `devnotes/v2.0.0/rules/design-system-philosophy.md`

## Product Goal

Users can chat with one or more completed transcripts from the transcript detail right sidebar. Chat should feel like a quiet reading companion: session switching at the top, streamed answers in the middle, and compact context/model controls inside the composer.

Required behavior:

- Chat lives in the existing right sidebar next to Notes.
- If no LLM provider is configured, chat is disabled and shows a clear settings message.
- The top-left control opens a chat-session dropdown.
- The top-right control creates a new chat session.
- The composer has a plus icon context picker, selected transcript context bubbles, a model dropdown, and a send button.
- The plus picker lists only completed transcriptions, using audio titles as display labels.
- Selecting a transcript adds it as context but never displays raw transcript text in the chat UI.
- Selected contexts render as removable bubbles with an `x` icon on hover/focus.
- Assistant responses render read-only Markdown through Textforge.
- The screenshot logo/icon next to the session name is intentionally excluded.

Out of scope:

- Displaying raw transcript context in the chat session.
- Prompt templates, tool calls, citations, or multi-user chat.
- Separate full-page chat history management.
- Editing message content.

## Target Frontend Model

Keep chat under the transcription feature boundary:

```txt
features/transcription/api/chatApi.ts
features/transcription/hooks/useTranscriptChat.ts
features/transcription/components/TranscriptChatPanel.tsx
features/transcription/components/TranscriptNotesSidebar.tsx
features/transcription/components/AudioDetailView.tsx
```

Use existing shared primitives:

- `components/ui` buttons, popovers, command, dropdowns, and tooltips.
- Lucide icons for plus, new chat, remove, send, copy, search, and chevrons.
- Existing `ReadOnlyMarkdown` Textforge wrapper for assistant output.

Server state rules:

- Chat sessions, messages, context sources, models, and completed transcript choices use typed API helpers plus TanStack Query hooks.
- Mutations invalidate the smallest useful chat query key.
- The streaming send flow uses local optimistic state for the active run and invalidates persisted messages at completion.
- Components do not assemble chat URLs inline.

## Sprint Runs

### Sprint 1: API Contract and Hook Foundation

- Add any missing backend route registration needed for the completed chat handlers.
- Add a message-list route if reopening a session needs persisted messages.
- Add typed frontend chat API helpers.
- Add chat query keys and hooks for models, sessions, messages, and context.
- Add mutations for create session, update session title/status, add/remove context, and stream message.
- Add a completed-transcript selector derived from transcription and file queries.

### Sprint 2: Sidebar Shell and Session Navigation

- Convert the right sidebar tab bar from Notes-only to Chat/Notes.
- Preserve notes rendering and sidebar resize/collapse behavior.
- Add the chat panel top bar.
- Implement the session dropdown grouped by recency.
- Implement new chat session creation using the selected model.
- Handle loading, empty, and provider-disabled states.

### Sprint 3: Composer, Context Picker, and Model Picker

- Add the composer matching the screenshot structure.
- Replace the `@` context trigger with a Lucide plus icon.
- Add searchable completed-transcript context picker.
- Render selected context bubbles and remove controls.
- Replace Advanced with a model dropdown.
- Disable composer controls when provider configuration or model availability is missing.

### Sprint 4: Streaming Messages and Markdown Rendering

- Send messages through the SSE stream endpoint.
- Render user bubbles, assistant Markdown, reasoning disclosure, and copy/delete affordances.
- Keep reasoning separate from final content.
- Use Textforge read-only Markdown rendering for assistant responses.
- Maintain scroll behavior without interrupting users reading older responses.
- Refresh session/message/context queries after completion or failure.

### Sprint 5: Polish, Accessibility, and Verification

- Tighten spacing, truncation, hover/focus behavior, and mobile layout.
- Ensure icon-only controls have accessible labels and tooltips.
- Confirm transcript/audio hot paths are not re-rendered by chat streaming.
- Run frontend type-check/build and backend tests for touched chat routes.
- Update tracker with completed artifacts and verification.
