# Chat With Transcripts Backend Sprint Plan

## Current Assessment

The backend is not yet architecturally ready for the requested chat-with-transcripts feature.

What exists now:

- `chat_sessions` and `chat_messages` persistence records exist, but they are legacy-shaped records embedded in `internal/models/transcription.go`.
- Chat rows are linked to `user_id` and one `transcription_id`, but there is no clean chat repository, chat service, context-source model, run lifecycle, cancellation model, or v2 REST contract.
- Legacy chat routes were intentionally removed from the canonical API pass and are listed as deferred module routes under `/api/v1/chat/sessions`.
- The LLM package has basic non-streaming and streaming chat calls, but streaming returns raw content strings only. It does not expose typed content deltas, reasoning deltas, usage, finish reasons, provider request IDs, or normalized provider errors.
- The summarization service has a plaintext transcript helper, but it strips speaker labels and truncates by approximate characters. That helper is not robust enough for multi-transcript chat context management or transcript compaction.
- The frontend currently contains legacy chat components that call old `/api/v1/chat/...` endpoints and render assistant output with `react-markdown`, not Textforge read-only rendering.

What is missing for the requested backend behavior:

- Clean chat database schema designed around user-owned sessions, parent transcription, message persistence, selected context transcripts, and model response runs.
- Backend-managed context sources that allow adding/removing the parent transcript and other completed transcripts during an existing session.
- A canonical REST and streaming API contract for creating sessions, listing sessions, managing context, sending messages, cancelling generation, and retrieving persisted messages.
- Durable state for in-flight assistant generation. Chat generation can be started by HTTP, but the database must know about the run before provider execution begins.
- Provider streaming abstraction that separates `reasoning` tokens from final `content` tokens and can stream both to the frontend in real time.
- LLM provider readiness checks that return clear API responses when the user has not configured a backend provider.
- Model discovery from the configured LLM provider so clients can choose from currently available chat-capable models.
- Automatic model capability discovery, especially context-window length, from the configured provider endpoint where the endpoint exposes metadata.
- Robust context budgeting and compaction. The current code can truncate transcript text for summaries, but it cannot compact an oversized transcript intelligently, and it cannot compact chat history while retaining full transcript context.
- Token accounting or tokenizer-aware budgeting. The current summarization code uses a rough character estimate only.
- Tests for ownership, context source mutation, streaming event shape, cancellation, provider failures, compaction, and unauthorized transcript injection.

## Architectural Direction

Build chat as a new backend workflow module, not as CRUD over legacy rows.

Target flow:

```txt
HTTP handler -> typed request/auth boundary
chat service -> session/context/run orchestration
chat repository -> durable chat persistence
context builder -> transcript plaintext/context budget/compaction
LLM provider -> typed streaming deltas
stream writer -> SSE or NDJSON response
response mapper -> public API shape
```

New code should live behind these boundaries:

- `internal/chat`: service, context builder, compactor, stream orchestration, provider-facing types.
- `internal/repository`: chat repository interface and implementation.
- `internal/models/chat.go`: chat persistence records only.
- `internal/llm`: provider-neutral streaming events and OpenAI-compatible/Ollama adapters.
- `internal/api/chat_handlers.go`: thin handlers only.

Do not preserve old chat compatibility. Remove dead legacy chat fields, old frontend-only assumptions, and any old chat route shape that conflicts with the v2 API.

## LLM Provider and Model Capability Rules

Chat depends on the authenticated user's active LLM provider configuration. The backend must make this explicit and predictable.

Provider readiness:

- Every chat API entry point that needs model execution must check for an active LLM provider before doing chat-specific work.
- If no provider is configured, return a standard error envelope with a stable code such as `LLM_PROVIDER_NOT_CONFIGURED`, status `409 Conflict`, and a message suitable for the UI: `Configure an LLM provider before starting chat.`
- If a provider is configured but unreachable, return `LLM_PROVIDER_UNAVAILABLE`, status `503 Service Unavailable`, with a sanitized message.
- If a requested model is not in the provider's available model list, return `MODEL_NOT_AVAILABLE`, status `422 Unprocessable Entity`, with `field: "model"`.
- Provider readiness checks should never expose API keys, raw endpoint URLs with embedded credentials, provider stack traces, or full upstream error bodies.

Model discovery:

- `GET /api/v1/chat/models` must read the active backend LLM provider configuration and query the configured provider endpoint.
- Responses should include currently available models and model capability metadata when known.
- The endpoint must return a typed empty/error state when the provider is not configured instead of pretending chat is available.
- Model discovery should be bounded by a short timeout and may use a small per-user cache to avoid repeated model-list calls on every render.
- Cached capability data must be invalidated when the user updates LLM provider settings.

Recommended model list response:

```json
{
  "provider": "openai_compatible",
  "configured": true,
  "models": [
    {
      "id": "qwen3.5-4B",
      "display_name": "qwen3.5-4B",
      "context_window": 32768,
      "context_window_source": "provider",
      "supports_streaming": true,
      "supports_reasoning": true
    }
  ]
}
```

Recommended provider-not-configured response:

```json
{
  "error": {
    "code": "LLM_PROVIDER_NOT_CONFIGURED",
    "message": "Configure an LLM provider before starting chat.",
    "field": null,
    "request_id": "req_123"
  }
}
```

Context-window discovery:

- Resolve the context window for the selected model before creating a generation run or compacting context.
- Prefer provider-exposed model metadata from the configured endpoint.
- For OpenAI-compatible providers, support common metadata shapes from `/v1/models`, including direct fields like `context_window`, `context_length`, `max_context_length`, `max_position_embeddings`, or provider-specific metadata objects.
- For Ollama-style endpoints, use model detail endpoints such as `/api/show` where available and map fields like `num_ctx`, `context_length`, or model metadata into the common capability shape.
- If the endpoint cannot provide context length, fall back to a conservative known-model registry and then a safe default.
- Every resolved context window must include a source value: `provider`, `provider_metadata`, `known_model`, `configured_default`, or `safe_default`.
- Store the resolved `context_window` and source on `chat_generation_runs` metadata so later debugging and compaction decisions are explainable.
- Do not let clients supply arbitrary context windows in chat requests. Clients choose models; the backend resolves capabilities.

## Target Backend Model

Recommended tables:

```txt
chat_sessions
id                         string primary key
user_id                    uint not null indexed
parent_transcription_id    string not null indexed
title                      string not null
provider                   string not null
model                      string not null
system_prompt              text null
status                     string not null default active
context_policy_json        json not null default {}
last_message_at            timestamp null
created_at                 timestamp
updated_at                 timestamp
deleted_at                 soft delete
```

```txt
chat_context_sources
id                         string primary key
user_id                    uint not null indexed
chat_session_id            string not null indexed
transcription_id           string not null indexed
kind                       string not null: parent_transcript | transcript
enabled                    bool not null default true
position                   integer not null default 0
plain_text_snapshot        text null
snapshot_hash              string null
source_version             string null
compacted_snapshot         text null
compaction_status          string not null default none
metadata_json              json not null default {}
created_at                 timestamp
updated_at                 timestamp
```

```txt
chat_messages
id                         string primary key
user_id                    uint not null indexed
chat_session_id            string not null indexed
role                       string not null: user | assistant | system | tool
content                    text not null default ''
reasoning_content          text not null default ''
status                     string not null default completed
provider                   string null
model                      string null
run_id                     string null indexed
prompt_tokens              integer null
completion_tokens          integer null
reasoning_tokens           integer null
total_tokens               integer null
metadata_json              json not null default {}
created_at                 timestamp
updated_at                 timestamp
```

```txt
chat_generation_runs
id                         string primary key
user_id                    uint not null indexed
chat_session_id            string not null indexed
assistant_message_id       string null indexed
status                     string not null: pending | streaming | completed | failed | canceled
provider                   string not null
model                      string not null
context_window             integer not null
context_window_source      string not null default safe_default
context_tokens_estimated   integer not null default 0
compaction_applied         bool not null default false
error_message              text null
started_at                 timestamp null
completed_at               timestamp null
failed_at                  timestamp null
created_at                 timestamp
updated_at                 timestamp
```

```txt
chat_context_summaries
id                         string primary key
user_id                    uint not null indexed
chat_session_id            string not null indexed
summary_type               string not null: transcript | session
source_transcription_id    string null indexed
source_message_through_id  string null indexed
content                    text not null
model                      string not null
provider                   string not null
input_tokens_estimated     integer not null default 0
output_tokens_estimated    integer not null default 0
created_at                 timestamp
updated_at                 timestamp
```

Required indexes:

- `chat_sessions(user_id, parent_transcription_id, updated_at DESC)`
- `chat_context_sources(chat_session_id, enabled, position)`
- `chat_context_sources(user_id, transcription_id)`
- `chat_messages(chat_session_id, created_at ASC)`
- `chat_generation_runs(chat_session_id, created_at DESC)`
- `chat_generation_runs(status, created_at ASC)`

## Transcript Context Rules

- Transcript context must be assembled from completed transcript records owned by the authenticated user.
- Transcript input must be plaintext, never JSON record dumps.
- Segment text should be concatenated in transcript order.
- Timing fields, word timestamps, raw JSON, paths, provider metadata, and execution metadata must be excluded.
- Speaker labels should be included only when available and useful:

```txt
Speaker 1: First segment text.
Speaker 2: Second segment text.
```

- Repeated adjacent segments from the same speaker may be joined to reduce tokens.
- The context builder should support both a live parse from `transcript_text` and a persisted `plain_text_snapshot` for stable repeat prompts.
- Snapshot hashes should make it possible to detect transcript changes and refresh or compact context safely.

## Context Budgeting and Compaction Rules

Use explicit context sections:

```txt
System instructions
Active transcript contexts
Session summary, if any
Recent conversation messages
Current user message
```

Budget priorities:

1. System/developer safety and product instructions.
2. Active transcript contexts, including full parent transcript when it fits.
3. Current user message.
4. Recent un-compacted chat messages.
5. Session summary of older chat messages.

Oversized transcript behavior:

- If a transcript alone cannot fit, create a transcript-specific compacted representation with a larger model when available.
- Persist that compacted transcript context in `chat_context_summaries`.
- Mark the corresponding `chat_context_sources.compaction_status`.
- Surface compaction metadata in the session context API and stream headers/events.

Growing session behavior:

- When estimated context reaches a configurable threshold, compact only the chat history that is older than the recent-message window.
- Do not include original transcript context in session-history compaction.
- Retain active transcript context separately and rebuild it after compaction.
- Persist the session summary and the message boundary it summarizes.

Initial defaults:

- Context threshold: 80% of the resolved model context window.
- Completion reserve: model-specific if known, otherwise conservative.
- Recent message window: last 8-12 messages, adjusted by budget.
- Fallback token estimate can start at 4 chars/token, but sprint work should isolate this behind an estimator interface so a tokenizer can replace it.
- Never calculate chat context against an unknown or client-provided context window. Always use backend-resolved model capabilities.

## Streaming Contract

Prefer SSE for chat generation because it supports named events and is already used in the API.

Recommended endpoint:

```http
POST /api/v1/chat/sessions/{session_id}/messages:stream
Accept: text/event-stream
```

Event names:

```txt
chat.run.started
chat.message.created
chat.delta.reasoning
chat.delta.content
chat.context.compacted
chat.run.completed
chat.run.failed
chat.run.canceled
```

Example delta payloads:

```json
{
  "run_id": "chatrun_...",
  "message_id": "chatmsg_...",
  "delta": "partial text"
}
```

Reasoning deltas must be streamed separately from final response deltas. If a provider exposes reasoning in a native field, use that. If a provider only emits `<think>...</think>` in content, normalize it in the provider adapter and do not store those tags in assistant `content`.

Assistant messages should persist:

- `content`: final markdown response only.
- `reasoning_content`: reasoning/thinking text only.
- token usage fields when available.
- run status and sanitized provider error on failure.

## REST API

Recommended canonical routes:

```http
GET    /api/v1/chat/models
GET    /api/v1/chat/sessions?parent_transcription_id=tr_...
POST   /api/v1/chat/sessions
GET    /api/v1/chat/sessions/{session_id}
PATCH  /api/v1/chat/sessions/{session_id}
DELETE /api/v1/chat/sessions/{session_id}

GET    /api/v1/chat/sessions/{session_id}/context
POST   /api/v1/chat/sessions/{session_id}/context/transcripts
PATCH  /api/v1/chat/sessions/{session_id}/context/transcripts/{context_source_id}
DELETE /api/v1/chat/sessions/{session_id}/context/transcripts/{context_source_id}

POST   /api/v1/chat/sessions/{session_id}/messages:stream
POST   /api/v1/chat/runs/{run_id}:cancel
POST   /api/v1/chat/sessions/{session_id}/title:generate
```

Developer UX rules:

- Keep resources predictable: sessions, messages, context sources, and runs should have explicit URLs and typed response bodies.
- Use command suffixes only for non-CRUD operations: `messages:stream`, `runs/{id}:cancel`, and `title:generate`.
- Accept and return public IDs only. Clients should never need to know whether database IDs are UUIDs, strings, or integers.
- Prefer `parent_transcription_id` over ambiguous `transcription_id` when filtering sessions, because sessions can include multiple context transcripts.
- Use idempotency keys on session creation, context-source creation, and message streaming starts so clients can retry after network failures.
- Return the created user message, assistant message placeholder, and run ID early in the stream so clients can reconcile optimistic UI state.
- Make stream events self-contained: every event should include `session_id`, `run_id`, and `message_id` when applicable.
- Keep event names stable and documented. Add fields instead of changing meanings.
- Use cursor pagination for session and message lists. Default limits should be useful for UI use and bounded for API clients.
- Provide `include=` expansion for developer convenience where it does not create expensive responses, for example `include=latest_message,context_summary`.
- Return typed context status rather than opaque booleans, for example `active`, `disabled`, `stale`, `compacting`, `compacted`, `failed`.
- Put context/token metadata in response bodies and stream events, not only headers, so browser and CLI clients behave consistently.
- Return model capability metadata from model-list responses so clients can show meaningful model choices without hard-coded provider assumptions.
- Keep provider-not-configured responses consistent across model list, session creation, and message streaming.
- Use the standard error envelope with stable `code` values and a `field` pointer for validation errors.
- Keep request and response field names consistent with the rest of v2: snake_case JSON, collection envelope `{ "items": [], "next_cursor": null }`, RFC3339 timestamps.
- Document examples for curl, browser `EventSource`/fetch streaming, and TypeScript response types during the implementation sprint.

Response rules:

- Use public IDs with prefixes such as `chat_`, `chatmsg_`, `chatctx_`, and `chatrun_`.
- Collection responses should use `{ "items": [], "next_cursor": null }`.
- Errors should use the standard v2 error envelope.
- No local paths, raw transcript JSON, API keys, provider stack traces, or database-only IDs in responses.

Recommended create-session request:

```json
{
  "parent_transcription_id": "tr_abc123",
  "title": "Research questions",
  "model": "qwen3.5-4B",
  "include_parent_transcript": true
}
```

Recommended create-session response:

```json
{
  "id": "chat_abc123",
  "parent_transcription_id": "tr_abc123",
  "title": "Research questions",
  "provider": "openai_compatible",
  "model": "qwen3.5-4B",
  "model_capabilities": {
    "context_window": 32768,
    "context_window_source": "provider",
    "supports_streaming": true,
    "supports_reasoning": true
  },
  "status": "active",
  "context": {
    "items": [
      {
        "id": "chatctx_parent",
        "transcription_id": "tr_abc123",
        "kind": "parent_transcript",
        "status": "active",
        "enabled": true
      }
    ]
  },
  "created_at": "2026-05-01T12:00:00Z",
  "updated_at": "2026-05-01T12:00:00Z"
}
```

Recommended stream request:

```json
{
  "content": "What are the main objections raised in this transcript?",
  "client_message_id": "tmp_msg_123"
}
```

Recommended stream event sequence:

```txt
event: chat.run.started
data: {"session_id":"chat_abc123","run_id":"chatrun_123","user_message_id":"chatmsg_u1","assistant_message_id":"chatmsg_a1"}

event: chat.delta.reasoning
data: {"session_id":"chat_abc123","run_id":"chatrun_123","message_id":"chatmsg_a1","delta":"I need to compare the objections..."}

event: chat.delta.content
data: {"session_id":"chat_abc123","run_id":"chatrun_123","message_id":"chatmsg_a1","delta":"The main objections are"}

event: chat.run.completed
data: {"session_id":"chat_abc123","run_id":"chatrun_123","message_id":"chatmsg_a1","usage":{"prompt_tokens":1200,"completion_tokens":240,"reasoning_tokens":80,"total_tokens":1520}}
```

## Sprint Run 1: Remove Legacy Chat and Add Clean Schema

Goal: replace legacy chat persistence with a v2 chat schema that supports user ownership, parent transcription, context sources, messages, and generation runs.

Scope:

- Move chat persistence models out of `internal/models/transcription.go` into `internal/models/chat.go`.
- Replace legacy `ChatSession` and `ChatMessage` fields with the target model.
- Add `ChatContextSource`, `ChatGenerationRun`, and `ChatContextSummary`.
- Register new schema models and indexes.
- Remove legacy chat migration/backfill code and tests that assert the old chat shape.
- Add fresh schema and cascade/delete tests.

Acceptance criteria:

- Fresh databases create the new chat tables and indexes.
- Chat sessions are always user-scoped and parented to a transcription.
- Deleting a chat session cascades messages, context sources, runs, and context summaries.
- Old compatibility fields such as `job_id`, `is_active`, metadata-only message count, and numeric chat message IDs are gone.

Verification:

- `go test ./internal/database ./internal/repository`

## Sprint Run 2: Chat Repository and Context Builder

Goal: build domain persistence and transcript plaintext assembly before any HTTP endpoint streams model output.

Scope:

- Add `ChatRepository` with domain methods for sessions, messages, context sources, runs, and summaries.
- Add `internal/chat.ContextBuilder`.
- Add transcript ownership checks through `JobRepository.FindTranscriptionByIDForUser`.
- Add plaintext transcript assembly with speaker labels and no timestamps/metadata.
- Add context source add/remove/reorder/enable-disable operations.
- Add context budgeting primitives and token-estimator interface.
- Add model capability structs to the chat domain so context building accepts backend-resolved model capabilities rather than raw client values.

Acceptance criteria:

- A chat session can include its parent transcript and additional transcripts owned by the same user.
- A user cannot add another user's transcript to a chat context.
- Transcript plaintext output contains only speaker labels and text.
- Context source mutations are persisted and reflected in the next context build.
- Context building requires a resolved model context window and records the source of that value.
- No HTTP handler reads GORM directly for chat.

Verification:

- `go test ./internal/chat ./internal/repository`

## Sprint Run 3: Provider Streaming and Reasoning Deltas

Goal: normalize provider model capabilities and streaming into typed data suitable for frontend model choice and real-time rendering.

Scope:

- Add provider-neutral model capability types:
  - model ID
  - display name
  - context window
  - context window source
  - streaming support
  - reasoning support
- Add provider capability discovery through the configured LLM endpoint.
- Parse context-window metadata from OpenAI-compatible and Ollama-style provider responses.
- Add a conservative known-model fallback registry for providers that only return model IDs.
- Replace string-only `ChatCompletionStream` with a provider-neutral typed stream event.
- Support event kinds: `content_delta`, `reasoning_delta`, `usage`, `done`, `error`.
- Update OpenAI-compatible streaming parser for content deltas and known reasoning fields.
- Add fallback parsing for `<think>...</think>` content streams in an adapter layer.
- Update Ollama streaming parser if it remains supported.
- Sanitize provider errors before they reach API responses or persisted messages.

Acceptance criteria:

- Model list and context-window resolution come from the configured provider endpoint when metadata is available.
- Unknown models without provider metadata use a conservative backend fallback and expose the fallback source.
- Provider-not-configured and provider-unavailable states return stable typed errors.
- Provider adapters never expose raw provider stack traces to API callers.
- Reasoning and final content are separable during streaming and after persistence.
- Existing summarization code still works through a non-streaming provider method or a compatibility adapter.

Verification:

- `go test ./internal/llm ./internal/summarization`

## Sprint Run 4: Context Compaction

Goal: handle oversized transcripts and growing chat sessions robustly.

Scope:

- Add transcript-specific compaction for transcripts that cannot fit in the target model context.
- Add session-history compaction when context crosses the configured threshold.
- Ensure session-history compaction excludes original transcript text.
- Persist compaction summaries and message boundaries.
- Publish/stream compaction metadata so the frontend can show context state.
- Add configuration fields for threshold, reserve tokens, recent message window, and compaction model preference.

Acceptance criteria:

- Oversized transcript context is compacted before chat generation instead of blindly truncated.
- Oversized transcript detection uses the selected model's backend-resolved context window.
- Session-history compaction retains transcript context separately and only summarizes older conversation messages.
- Context builder can reconstruct a prompt after restart using persisted sources and summaries.
- Compaction failure produces a controlled error or fallback according to service policy.

Verification:

- `go test ./internal/chat`

## Sprint Run 5: REST API and Streaming Handler

Goal: expose the canonical v2 chat API with thin handlers and streaming responses.

Scope:

- Register `/api/v1/chat` routes.
- Implement model list using active user LLM provider settings and provider capability discovery.
- Implement session CRUD.
- Implement context source list/add/update/remove endpoints.
- Implement `messages:stream` as SSE.
- Persist the user message, create a generation run, stream provider deltas, update assistant message incrementally or at completion, then mark run terminal.
- Implement run cancellation.
- Add route contract tests and API tests for auth, ownership, idempotency, pagination, validation fields, error envelopes, and stream event shape.
- Add developer-facing examples to the API docs for session creation, context-source mutation, streaming chat, and cancellation.

Acceptance criteria:

- Chat sessions are persisted under the authenticated user and parent transcription.
- Chat features return `LLM_PROVIDER_NOT_CONFIGURED` when no backend provider is configured.
- Users can choose any currently available model returned by `GET /api/v1/chat/models`.
- Session creation and message streaming reject unavailable models with `MODEL_NOT_AVAILABLE`.
- Generation runs persist the selected model, resolved context window, and context-window source.
- The backend streams reasoning and content separately in real time.
- A refresh after a completed stream reconstructs the full session from the database.
- Canceled and failed runs leave durable, inspectable state.
- Session and message list endpoints use bounded cursor pagination and stable collection envelopes.
- Create and stream endpoints support `Idempotency-Key` for retry-safe clients.
- Validation errors identify actionable fields, for example `parent_transcription_id`, `context.transcription_id`, or `content`.
- Stream events are self-contained enough for a client to recover optimistic UI state without an extra immediate fetch.
- API docs include request/response examples and event examples.

Verification:

- `go test ./internal/api ./internal/chat ./internal/repository`

## Sprint Run 6: Frontend Integration Contract Support

Goal: align the backend contract with the frontend requirements without implementing broad frontend redesign in the backend sprint.

Scope:

- Document the exact SSE event contract for the frontend.
- Ensure assistant `content` is Markdown-only final output.
- Ensure assistant `reasoning_content` is returned separately.
- Return context state suitable for controls that add/remove transcript sources.
- Include enough response metadata for Textforge read-only rendering and expandable thinking sections.
- Coordinate replacing legacy frontend calls to old chat endpoints with the canonical v2 routes.

Acceptance criteria:

- The frontend can render reasoning and content separately without parsing model-specific tags.
- The frontend can add/remove transcripts mid-session and the next generation uses the updated context.
- The final assistant output is ready for Textforge read-only Markdown rendering.

Verification:

- Backend API contract tests plus frontend build after frontend sprint begins.

## Open Product Questions

- Should the default new session include the parent transcript in context automatically, or should it start with no transcript selected and let the user opt in?
- For transcript compaction, should the backend prefer the configured large model, the active chat model, or a separate future compaction model setting?
- Should removing a transcript from context delete its stored snapshot/summary, or disable it so it can be re-enabled without recomputation?
