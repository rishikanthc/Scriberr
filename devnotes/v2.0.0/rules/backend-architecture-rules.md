# Scriberr Backend Architecture Rules

These are the backend rules for the revamped Scriberr codebase. They describe the target architecture, not the shape of legacy code. The only compatibility requirement is database migration: older installations must migrate cleanly into the new schema. Legacy routes, legacy handler patterns, and transitional APIs should not constrain new design.

## 1. Build Around Domain Workflows, Not CRUD Files

Scriberr's backend is organized around workflows: upload/import media, create transcription jobs, run durable workers, stream progress, inspect transcripts, summarize, chat, manage profiles, and configure engines.

Use CRUD only where the domain is actually CRUD-like. Transcription is not CRUD; it is a stateful workflow.

Preferred flow:

```txt
HTTP handler -> typed request and auth boundary
service/use-case -> domain decision and orchestration
repository -> durable persistence operation
worker/orchestrator/provider -> long-running execution
event publisher -> small progress/cache notification
response mapper -> public API shape
```

Handlers should be thin. Services should own decisions. Repositories should own database shape. Providers should own engine details.

## 2. The HTTP Layer Is an Adapter

`internal/api` should translate HTTP into domain commands and translate domain results back into responses. It should not own queue policy, transcription execution, profile resolution policy, filesystem layout, or database invariants.

Handlers may:

- Authenticate and authorize.
- Bind and validate request syntax.
- Parse public IDs.
- Call one service method.
- Map returned domain state to a response.
- Emit the standard error envelope.

Handlers should not:

- Reach directly into `database.DB` for new feature work.
- Run transcription, extraction, summarization, or chat generation inline.
- Scatter status updates.
- Construct filesystem paths outside storage services.
- Decide which provider/model should execute a job.

## 3. Public Contracts Are Separate From Internal State

Public API IDs and response shapes are contracts. Database primary keys, file paths, transcript artifact paths, provider errors, and model cache paths are internal.

Rules:

- Parse public IDs at the API boundary.
- Pass internal IDs through services and repositories.
- Emit public IDs only from response/event mappers.
- Keep API request/response types separate from GORM models.
- Never leak raw local paths, provider stack traces, or model cache details to clients.

If a domain model needs to change, the API contract should change deliberately, with route contract tests and docs updated in the same change.

## 4. Durable Work Belongs in the Queue

Any operation that can outlive an HTTP request belongs in a durable workflow: transcription, media extraction, model-backed summarization, long chat responses, and future batch operations.

Durable workflows must have:

- A persisted record before execution begins.
- Explicit states and terminal states.
- Idempotent start/command behavior where clients may retry.
- Cancellation semantics.
- Progress or status observability.
- Recovery behavior after process restart.
- Tests for state conflicts and terminal behavior.

Do not add "quick" background goroutines from handlers for user-visible work. If the user cares about the result, the database should know about it.

## 5. State Transitions Have One Owner

Job state changes must flow through domain-specific repository/service methods such as enqueue, claim, renew, progress, complete, fail, and cancel. Avoid scattered `Update("status", ...)` calls.

When changing a lifecycle:

- Document valid source and target states.
- Persist the relevant timestamp and error/progress fields.
- Publish one meaningful event for user-visible transitions.
- Return state conflicts explicitly.
- Test both successful and invalid transitions.

Queue hot paths must stay atomic. Claiming work should use indexed candidate selection plus a conditional update in a transaction. Cancellation must account for both queued and actively running jobs.

## 6. Repositories Speak Domain Persistence

Repositories should hide GORM and expose persistence operations that mean something in Scriberr.

Good repository methods:

```txt
ClaimNextTranscription
RenewClaim
RecoverOrphanedProcessing
CountStatusesByUser
FindLatestCompletedExecution
ListExecutions
UpdateProgress
```

Avoid generic query plumbing in services when a query carries a domain invariant. Also avoid turning repositories into business services: they should not choose providers, publish events, or interpret user commands.

## 7. Models Are Persistence Records

`internal/models` is for durable records, schema tags, persisted enums, and migration helpers. Models should not become the domain service layer.

Models may own:

- Table names, GORM tags, and schema-facing defaults.
- JSON column serialization needed by persistence.
- Persistent enum definitions.
- Migration helpers that prepare older rows for the new schema.

Models should not own:

- HTTP response shapes.
- Queue orchestration.
- Provider selection.
- File serving policy.
- Event publishing.

## 8. Migrations Are the Only Legacy Contract

Because the codebase is being revamped, new code should optimize for the target model. Do not preserve legacy routes, compatibility response aliases, or old internal naming unless they still match the desired product API.

The compatibility obligation is database migration:

- Older database files must open and migrate forward.
- Migrations must be deterministic and covered by tests.
- New required columns need sensible backfills.
- Data transformations should be explicit and reversible in reasoning, even if not reversible in code.
- Indexes and constraints should be created as part of the schema step that needs them.

Do not keep awkward runtime compatibility hooks forever when a one-time migration can normalize the data.

## 9. Provider Boundaries Stay Narrow

Speech engines and model runtimes are expensive, platform-sensitive, and likely to evolve. The rest of Scriberr should know only about provider capabilities and typed requests/results.

Provider rules:

- Keep engine-specific setup in provider implementations.
- Convert provider-native outputs into Scriberr canonical transcript structures before storage.
- Sanitize provider errors before they reach services or API responses.
- Keep model download/cache policy in engine/provider configuration.
- Make capability listing deterministic.
- Make provider interfaces easy to fake in tests.

The orchestrator coordinates providers; handlers should never call engine code directly.

## 10. Optimize the Real Hot Paths

Scriberr's hot paths are queue polling/claiming, progress updates, list views, event streaming, file/audio streaming, transcript retrieval, and provider capability checks.

Performance rules:

- Queue claim queries must stay indexed by status and queue time.
- Progress updates should update only necessary columns.
- List endpoints must use bounded limits or cursor pagination.
- SSE events should be small cache invalidation/status messages, not large transcript payloads.
- Audio and transcript endpoints should stream or load the specific artifact needed.
- List screens should read metadata columns, not parse transcript files or stat many files in a loop.
- Add indexes when a query enters a polling, queue, or frequently refreshed path.

Performance work should start with the known workflow shape, not broad micro-optimization.

## 11. Context and Cancellation Are Correctness

Every request, repository operation, queue operation, provider call, and long filesystem operation should accept or derive from `context.Context`.

Rules:

- Request-scoped work uses request context.
- Claimed jobs use worker/job context.
- Cancellation is a valid terminal path, not a generic failure.
- Shutdown should stop accepting work, cancel running jobs, wait with a bounded timeout, then close providers and the database.

Ignoring context in transcription and media paths wastes user hardware and leaves misleading job states.

## 12. Events Are Notifications, Not Source of Truth

SSE should make the UI feel live, but durable truth remains in the database and transcript artifacts.

Use events for:

- Progress and terminal status.
- Created/updated/deleted notifications.
- Cache invalidation.
- Small user-facing status payloads.

Do not depend on event delivery for correctness. Clients can reconnect or miss messages. A full page refresh must reconstruct accurate state from persisted records.

## 13. Storage Is an Explicit Boundary

Uploads, imported media, extracted audio, transcript JSON, temporary files, and model caches are operational state. Keep storage policy centralized.

Rules:

- Sanitize original filenames.
- Separate user-visible titles from stored filenames.
- Store durable artifacts under configured directories.
- Prevent path traversal before serving any file.
- Use restrictive permissions for transcript artifacts.
- Keep temporary artifacts scoped and cleaned up.
- Do not expose local paths in API responses.

Filesystem access should read like a storage operation, not ad hoc string manipulation inside handlers.

## 14. Configuration Fails Early

Configuration should be loaded, validated, logged, and injected at startup. Runtime code should not read environment variables directly.

When adding config:

- Add a typed field to `Config`.
- Add default loading and validation.
- Add tests for invalid values.
- Log important operational settings at startup.
- Inject it into the service/provider that needs it.

Startup may initialize providers, but it should not unexpectedly download large models unless configuration explicitly says to do so.

## 15. Tests Protect Contracts and State Machines

Prioritize tests around failures that are hard to see manually:

- Route registration and API contract shape.
- Auth, request IDs, and structured errors.
- Idempotent create/import/command behavior.
- Queue claim, lease renewal, recovery, cancellation, and terminal updates.
- Database migration from older schemas.
- Index and constraint invariants.
- Transcript canonicalization and provider result merging.
- Security regressions for path traversal, sensitive errors, and unauthorized access.

Use fake providers and repositories where possible. Only use real engine integration tests for explicitly marked integration paths.

## 16. The Backend Should Read Like Scriberr

Use domain terms consistently: file, transcription, profile, execution, provider, queue, transcript, summary, note, speaker, model.

Good backend code in this repo should be:

```txt
thin at the HTTP edge
explicit in domain services
durable for long-running work
transactional around state transitions
narrow at provider interfaces
indexed on queue/list/event hot paths
sanitized before public output
migration-safe for old databases
testable without real engines
```
