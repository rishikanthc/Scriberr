# Chat With Transcripts API

Chat routes live under `/api/v1/chat` and require the same bearer token or API key authentication as the rest of the v1 API.

## Response Shapes

Collections use:

```json
{ "items": [], "next_cursor": null }
```

Context source responses expose state and metadata only. They do not include raw transcript text, local paths, provider metadata, or compacted snapshot bodies.

Important context fields:

- `status`: `active`, `disabled`, `compacting`, `compacted`, or `failed`
- `compaction_status`: persisted compaction lifecycle state
- `has_plain_text_snapshot`: whether the backend has a stable plaintext snapshot
- `has_compacted_snapshot`: whether the source is currently using compacted context
- `tokens_estimated`: approximate context size for UI budgeting displays

Assistant message responses keep final content and reasoning separate:

```json
{
  "id": "chatmsg_...",
  "role": "assistant",
  "content": "Final Markdown answer.",
  "reasoning_content": "Provider reasoning text.",
  "status": "completed",
  "prompt_tokens": 1200,
  "completion_tokens": 240,
  "reasoning_tokens": 80,
  "total_tokens": 1520
}
```

`content` is the final Markdown body suitable for read-only rendering. `reasoning_content` is never mixed into `content`.

## Streaming Events

`POST /api/v1/chat/sessions/{session_id}/messages:stream` returns `text/event-stream`.

Every run-scoped event includes `session_id`, `run_id`, and `message_id` when applicable.

Event names:

- `chat.run.started`
- `chat.message.created`
- `chat.delta.reasoning`
- `chat.delta.content`
- `chat.run.completed`
- `chat.run.failed`

Delta payloads:

```json
{
  "session_id": "chat_...",
  "run_id": "chatrun_...",
  "message_id": "chatmsg_...",
  "delta": "partial text"
}
```

Completion payload:

```json
{
  "session_id": "chat_...",
  "run_id": "chatrun_...",
  "message_id": "chatmsg_...",
  "status": "completed",
  "assistant_message": {
    "id": "chatmsg_...",
    "role": "assistant",
    "content": "Final Markdown answer.",
    "reasoning_content": "Reasoning text.",
    "status": "completed"
  },
  "usage": {
    "prompt_tokens": 1200,
    "completion_tokens": 240,
    "reasoning_tokens": 80,
    "total_tokens": 1520
  }
}
```

Errors use the standard envelope before streaming starts. Provider failures after streaming starts are emitted as `chat.run.failed` with a sanitized `error` string.
