# Scriberr Backend Architecture Design

## Principle

Scriberr is a modular monolith. Keep one Go backend process, but make package boundaries strict enough that ASR engines, storage, queue execution, LLMs, and HTTP can be replaced independently.

Core rule:

```txt
HTTP/API -> services -> repositories/providers/storage -> database/files/engines
```

No inner package imports `internal/api`. No service reaches through another service into its database tables, file paths, or HTTP DTOs.

## Current Target Package Map

Use the existing packages as the migration base:

```txt
cmd/server              process lifecycle: flags, config, listener, signals
internal/app            composition root for repositories, services, API, workers
internal/config         typed configuration loading and validation
internal/database       DB open, migrations, health checks only
internal/models         persistence records and migration compatibility
internal/repository     GORM-backed repository implementations
internal/api            Gin routes, auth boundary, DTO mapping, SSE adapter
internal/account        auth, API keys, user settings commands
internal/profile        transcription profile workflows
internal/files          upload/import metadata, file readiness, audio lookup
internal/automation     post-file-ready automation decisions
internal/transcription  transcription commands and queries
internal/transcription/worker
                         durable queue workers, leases, cancellation, stats
internal/transcription/orchestrator
                         job execution workflow and transcript persistence
internal/transcription/engineprovider
                         ASR provider interface, registry, local provider
internal/recording      browser recording sessions, chunks, finalization
internal/summarization  summary/widget generation workflow
internal/chat           transcript chat workflow
internal/llmprovider    LLM provider settings
internal/llm            LLM client implementations
internal/annotations    highlights and notes workflow
internal/tags           tag workflow
internal/mediaimport    YouTube/video import adapter
internal/web            static frontend serving adapter
```

Future cleanup may introduce `internal/domain` for pure domain types. Do not block current refactors on that rename; first enforce behavior and dependency direction.

## Dependency Direction

Allowed:

```txt
cmd/server -> internal/app, config, logger, and process/runtime packages
internal/app -> every concrete package needed for wiring
internal/api -> service interfaces/concrete services, auth helpers, response DTOs
services -> repositories, provider registries, storage boundaries, domain/persistence models
worker -> repository queue methods, orchestrator processor
orchestrator -> engineprovider registry, repository execution methods
repository -> models, database driver/GORM
database -> models, migrations
providers/llm/storage adapters -> external libraries
```

Forbidden:

```txt
internal/api -> internal/database for production code
internal/api -> database.DB
non-bootstrap packages -> internal/api
non-bootstrap packages -> internal/database
repository -> api
models -> api, services, providers
engineprovider -> api or repository
worker -> api handlers
services -> Gin, HTTP DTOs, or request/response structs
```

Tests may seed databases directly when it is the clearest verification path.

## Composition Root

`internal/app` owns backend construction and bounded application lifecycle:

1. Open database and run migrations.
2. Construct repositories from `database.DB`.
3. Construct provider registries and external adapters.
4. Construct services.
5. Wire event publishers, completion observers, and file-ready handoffs.
6. Build `api.Handler` from explicit `api.HandlerDependencies`.
7. Build routes without starting the HTTP listener.
8. Start durable workers after queue recovery.
9. On shutdown, stop workers, summaries, finalizers, providers, then close DB.

`cmd/server/main.go` owns only process concerns:

1. Parse flags.
2. Initialize logger.
3. Load, validate, and log `config.Config`.
4. Call `app.Build`.
5. Call `App.Start`.
6. Start the HTTP server from `App.Server`.
7. Handle shutdown signals and process exit.

`api.NewHandler` must not create fallback repositories or services.

## HTTP/API Boundary

`internal/api` is an adapter. It may:

- Authenticate and authorize.
- Parse path/query/body data.
- Validate API syntax.
- Translate public IDs such as `tr_`, `file_`, `profile_`, and `rec_`.
- Call one service method per command/query.
- Map service results to public response DTOs.
- Publish small SSE payloads through service event interfaces.

It must not:

- Query `database.DB`.
- Build GORM repositories.
- Decide provider/model selection.
- Construct filesystem paths.
- Run transcription, extraction, summarization, chat, or title generation inline.
- Expose raw local paths, provider stack traces, or persistence structs as permanent API contracts.

## Services

Services own use cases and workflow decisions:

```txt
account.Service         registration, login, refresh tokens, API keys, settings validation
profile.Service         profile CRUD and default-profile invariants
files.Service           uploads, imports, file readiness, audio opening
automation.Service      auto-transcribe and auto-rename decisions after file readiness
transcription.Service   create/list/get/update/delete/cancel/retry/logs/executions
worker.Service          queue enqueue/claim/lease/cancel/recover/stats
orchestrator.Processor  provider execution, diarization, canonical transcript creation
recording.Service       recording session and chunk lifecycle
recording.Finalizer     recording finalization and file-ready handoff
summarization.Service   summary/widget runs and transcript completion observers
chat.Service            chat sessions, context building, LLM calls
```

Services should depend on narrow interfaces when a dependency is not local to their package. Prefer small command/query structs over passing HTTP requests or generic maps.

## Persistence Boundary

`internal/database` is only for DB lifecycle and migrations. `internal/repository` owns GORM and SQL shape.

Repository methods should express Scriberr invariants:

```txt
FindFileByIDForUser
ListTranscriptionsByUser
EnqueueTranscription
ClaimNextTranscription
RenewClaim
RecoverOrphanedProcessing
CompleteTranscription
FailTranscription
CancelTranscription
FindDefaultByUser
GetActiveByUser
```

Avoid adding generic query plumbing to services when a query has lifecycle, ownership, or state-machine meaning.

`internal/models` remains the persistence-record package for now. New API DTOs must live in `internal/api` or a dedicated DTO package, not in `models`.

## File And Storage Boundary

Current local storage is implemented inside `internal/files`, `internal/recording`, media import code, and the transcript artifact store in `orchestrator`. The target is one explicit storage boundary before adding S3/MinIO or user-scoped storage.

Rules now:

- File paths are constructed only in file, recording, media import, or transcript storage code.
- Handlers receive opened files or metadata from services.
- Public responses never include local paths.
- Original filenames are sanitized and kept separate from storage object names.
- Temporary video/import/recording artifacts are cleaned up by the owning service.

Future interface:

```go
type Storage interface {
    Save(ctx context.Context, object Object, body io.Reader) error
    Open(ctx context.Context, objectID string) (io.ReadCloser, ObjectInfo, error)
    Delete(ctx context.Context, objectID string) error
}
```

## Transcription Queue

SQLite is the durable source of truth. In-memory channels only wake workers and hold process-local cancel functions.

State flow:

```txt
uploaded file
  -> transcription.Service.Create
  -> worker.Service.Enqueue
  -> repository.EnqueueTranscription
  -> worker.Service.Claim loop
  -> orchestrator.Processor
  -> engineprovider.Provider
  -> canonical transcript artifact
  -> repository terminal update
  -> completion observers and SSE events
```

Queue operations must be repository-owned and atomic:

```txt
enqueue
claim
renew lease
update progress
complete
fail
cancel
recover orphaned processing jobs
stats
```

Jobs are already user-scoped with `TranscriptionJob.UserID`; keep all queue/list/get/update operations user-aware. When multi-user scheduling is expanded, claim policy should add fairness without changing the API or provider boundary.

Target scheduling order:

```txt
priority DESC
per-user concurrency allowance
queued_at ASC
```

## ASR Provider Boundary

`internal/transcription/engineprovider` is the ASR port.

Provider interface:

```go
type Provider interface {
    ID() string
    Capabilities(ctx context.Context) ([]ModelCapability, error)
    Prepare(ctx context.Context) error
    Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error)
    Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error)
    Close() error
}
```

Rules:

- API handlers never call providers.
- Repositories never call providers.
- Provider-native outputs are converted to canonical transcript structs before storage.
- Provider errors are sanitized before public exposure.
- Model cache and runtime details remain inside provider implementations.
- Capabilities are deterministic and testable.

The current local provider is `local`. Future Python or remote engines should be added as new providers behind this interface, not as branches in handlers or repositories.

## Remote Provider Target

Remote providers should be optional adapters:

```txt
Go backend -> engineprovider.Registry -> remote provider client -> Python/remote worker
```

Use HTTP for batch transcription and WebSocket only when true streaming is implemented. Remote providers must advertise capabilities and models before use. The provider registry should select by explicit provider ID first, then by capability matching later.

## Events

Events are notifications, not truth. Durable state lives in the database and transcript artifacts.

Use small events for:

```txt
file.created
file.ready
transcription.queued
transcription.progress
transcription.completed
transcription.failed
transcription.stopped
summary.started/completed/failed
annotation/tag/recording updates
```

Clients must be able to recover by re-fetching REST resources after missed SSE events.

## Public IDs And DTOs

Public IDs are API contracts:

```txt
tr_<uuid>       transcription
file_<uuid>     uploaded/imported file
profile_<id>    transcription profile
rec_<uuid>      recording
chat_<uuid>     chat session
```

Parse public IDs at the API boundary. Services and repositories use internal IDs. Response structs should be explicit DTOs; do not expose persistence structs as a long-term contract.

## Refactor Path

1. Finish removing production `internal/api` imports of `internal/database`.
2. Move remaining handler persistence into account, profile, LLM provider, file, summary, chat, and admin services.
3. Continue centralizing transcript/audio object access behind explicit storage APIs.
4. Keep API DTO mappers as the public contract boundary and avoid exposing persistence structs.
5. Keep capability-based provider selection behind `engineprovider.Registry`.
6. Add remote provider discovery and health checks as adapter work, not core workflow work.
