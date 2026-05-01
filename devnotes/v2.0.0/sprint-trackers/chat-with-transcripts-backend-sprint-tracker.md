# Sprint Tracker: Chat With Transcripts Backend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/chat-with-transcripts-backend-sprint-plan.md`.

Status: completed through Sprint Run 6.

## Sprint Run 1: Remove Legacy Chat and Add Clean Schema

Status: completed

Completed tasks:

- Remove old chat persistence shape from `internal/models/transcription.go`.
- Add clean chat persistence records in `internal/models/chat.go`.
- Register new schema models and indexes.
- Remove old chat migration/backfill assumptions.
- Add database tests for fresh schema, indexes, ownership columns, and cascade behavior.

Artifacts:

- `internal/models/chat.go`
- `internal/models/transcription.go`
- `internal/database/schema.go`
- `internal/database/migrate.go`
- `internal/database/legacy.go`
- `internal/database/database_test.go`
- `internal/repository/implementations.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/database ./internal/repository`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server`

## Sprint Run 2: Chat Repository and Context Builder

Status: completed

Completed tasks:

- Add `ChatRepository`.
- Add `internal/chat.ContextBuilder`.
- Add plaintext transcript assembly with speaker labels and no timestamps or metadata.
- Add context source mutation methods.
- Add token-estimator and budget primitives.

Artifacts:

- `internal/repository/implementations.go`
- `internal/repository/chat_repository_test.go`
- `internal/chat/context_builder.go`
- `internal/chat/transcript.go`
- `internal/chat/budget.go`
- `internal/chat/context_builder_test.go`
- `internal/models/chat.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/database ./internal/chat ./internal/repository`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...`

## Sprint Run 3: Provider Streaming and Reasoning Deltas

Status: completed

Completed tasks:

- Add typed provider stream events.
- Normalize content and reasoning deltas.
- Update OpenAI-compatible and Ollama adapters.
- Keep summarization compatible with non-streaming generation.
- Add provider stream parser tests.

Artifacts:

- `internal/llm/service.go`
- `internal/llm/stream.go`
- `internal/llm/openai.go`
- `internal/llm/ollama.go`
- `internal/llm/stream_test.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/llm ./internal/summarization`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...`

## Sprint Run 4: Context Compaction

Status: completed

Completed tasks:

- Add oversized transcript compaction.
- Add session-history compaction that excludes transcript context.
- Persist context summaries and boundaries.
- Add configurable thresholds and reserve budgets.
- Add compaction tests.

Artifacts:

- `internal/chat/compactor.go`
- `internal/chat/compactor_test.go`
- `internal/repository/implementations.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/chat ./internal/repository`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...`

## Sprint Run 5: REST API and Streaming Handler

Status: completed

Completed tasks:

- Register canonical `/api/v1/chat` routes.
- Implement session, context, streaming message, run cancellation, and title-generation endpoints.
- Stream chat generation over SSE.
- Persist messages and run state.
- Add route contract, ownership, and stream event tests.

Artifacts:

- `internal/api/chat_handlers.go`
- `internal/api/chat_handlers_test.go`
- `internal/api/router.go`
- `internal/api/middleware.go`
- `internal/repository/implementations.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/repository`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...`

## Sprint Run 6: Frontend Integration Contract Support

Status: completed

Completed tasks:

- Document SSE payloads and response shapes.
- Return reasoning separately from content.
- Return context state for transcript add/remove controls.
- Ensure final assistant output is Markdown content suitable for Textforge read-only rendering.

Artifacts:

- `docs/api/chat.md`
- `docs/api/openapi.json`
- `internal/api/chat_handlers.go`
- `internal/api/chat_handlers_test.go`
- `internal/api/route_contract_test.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/repository`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...`
