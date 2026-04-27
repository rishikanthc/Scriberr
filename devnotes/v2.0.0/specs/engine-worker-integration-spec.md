# Engine Worker Integration Spec

## Purpose

Scriberr needs to move from an API-only transcription skeleton to real local
transcription and diarization execution. The default engine will be the custom
Go module under `references/engine`, while the backend architecture must remain
simple enough for local single-user use and flexible enough for future multi-user
and multi-engine deployments.

This spec is implementation-ready. The implementer should not need to choose
queue semantics, engine boundaries, transcript JSON shape, startup behavior, or
testing strategy.

## Current State

- API routes for files, transcriptions, profiles, settings, events, and queue
  stats exist under `/api/v1`.
- Transcription create/submit currently persists queued transcription records but
  does not execute real transcription work.
- `GET /transcriptions/{id}/logs` and `/executions` are still placeholders.
- There is an older Python adapter stack under `internal/transcription/adapters`
  and registry/pipeline code. The next implementation should remove that stack
  and replace it with the new engine-provider architecture.
- `references/engine` is an untracked local Go module with module path
  `scriberr-engine`.
- The engine module uses sherpa-onnx and exposes:
  - `engine.New(engine.Config) (*engine.Engine, error)`
  - `(*Engine).Transcribe(ctx, engine.TranscriptionRequest)`
  - `(*Engine).Diarize(ctx, engine.DiarizationRequest)`
  - `EnsureModel`, `LoadModel`, `UnloadModel`, `ListLoadedModels`,
    `IsModelInstalled`, and `Close`
- The engine serializes operations with an internal `opMu`, so one engine
  instance should be treated as one bounded inference resource.

## Design Goals

- Make first-run setup smooth: a few environment variables, automatic defaults,
  automatic model downloads, clear logs, and no manual Python environment setup.
- Keep the API layer thin. Handlers validate/request-map and call services; they
  never invoke engine code directly.
- Use durable job state in SQLite. In-memory channels may wake workers, but the
  database is the source of truth.
- Preserve local single-user simplicity while including `user_id` in queue and
  execution paths for future multi-user scheduling.
- Support future engines by defining a narrow provider interface now. The local
  Go engine is default; a future Python server becomes another provider behind
  the same interface.
- Persist word-level timestamps when available. Gracefully return empty arrays
  when word timestamps are absent.
- Keep logs/events sanitized. Never expose local filesystem paths in public API
  responses or SSE payloads.

## Engine Module Integration

### Module Wiring

Add the local engine module to Scriberr:

```go
require scriberr-engine v0.0.0
replace scriberr-engine => ./references/engine
```

Import engine packages only from the new orchestration/provider layer, not from
API handlers or repository packages.

### Default Engine Config

Extend `internal/config.Config` with a nested engine config or clearly grouped
fields:

```go
type EngineConfig struct {
    CacheDir     string
    Provider     string
    Threads      int
    MaxLoaded    int
    AutoDownload bool
}

type WorkerConfig struct {
    Workers      int
    PollInterval time.Duration
    LeaseTimeout time.Duration
}
```

Environment variables:

| Env var | Default | Meaning |
| --- | --- | --- |
| `SPEECH_ENGINE_CACHE_DIR` | `data/models` | Persistent model artifact cache. |
| `SPEECH_ENGINE_PROVIDER` | `auto` | `auto`, `cpu`, or `cuda`. |
| `SPEECH_ENGINE_THREADS` | `0` | `0` means engine default. |
| `SPEECH_ENGINE_MAX_LOADED` | `2` | Max resident models per local engine. |
| `SPEECH_ENGINE_AUTO_DOWNLOAD` | `true` | Download missing model artifacts automatically. |
| `TRANSCRIPTION_WORKERS` | `1` | Number of worker goroutines. Keep `1` by default because local engine serializes inference. |
| `TRANSCRIPTION_QUEUE_POLL_INTERVAL` | `2s` | Poll interval when no in-process wake signal fires. |
| `TRANSCRIPTION_LEASE_TIMEOUT` | `10m` | Claim lease timeout for in-progress jobs. |

Parsing rules:

- Invalid numeric/duration values should fail startup with a clear config error.
- Invalid provider values should fail startup with a clear config error.
- `SPEECH_ENGINE_PROVIDER=auto` should resolve to CUDA when the engine runtime
  detects CUDA support, otherwise CPU.
- `SPEECH_ENGINE_PROVIDER=cuda` should fail clearly if CUDA is unavailable.
- `SPEECH_ENGINE_AUTO_DOWNLOAD=false` should allow startup but fail jobs with a
  model-unavailable error if required artifacts are missing.

### Startup Flow

`cmd/server/main.go` should initialize in this order:

1. Initialize logger.
2. Load config.
3. Initialize DB and migrations.
4. Initialize auth/repositories.
5. Initialize engine provider registry:
   - Construct local engine provider from `EngineConfig`.
   - Do not download models at startup.
   - Log cache dir, requested provider, resolved provider if available, threads,
     max-loaded, and auto-download setting.
6. Initialize queue scheduler and worker service.
7. Start queue workers after DB recovery has marked expired jobs as queued or
   failed.
8. Initialize API handler with queue service, model service, and event publisher.
9. Start HTTP server.
10. On shutdown, stop workers, cancel running jobs, close engine providers, then
    close DB.

Startup must not depend on Python, `WHISPERX_ENV`, or legacy adapter bootstrap.
Remove old Python environment setup from the server path.

### CUDA Handling

Do not invent a CUDA installer in this sprint. CUDA should be handled by:

- `auto`: engine uses CUDA if detectable, otherwise CPU.
- `cpu`: force CPU.
- `cuda`: require CUDA and fail startup or first engine initialization with a
  clear message.

Docs and logs should explain:

- Docker CUDA users still need NVIDIA runtime/container toolkit.
- Binary users need CUDA-enabled runtime libraries available to the process.
- CPU fallback is automatic only for `auto`.

## Engine Provider Architecture

Create a new package, recommended path:

```txt
internal/transcription/engineprovider
```

Core interfaces:

```go
type Provider interface {
    ID() string
    Capabilities(ctx context.Context) ([]ModelCapability, error)
    Prepare(ctx context.Context) error
    Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error)
    Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error)
    Close() error
}

type Registry interface {
    DefaultProvider() Provider
    Provider(id string) (Provider, bool)
    Capabilities(ctx context.Context) ([]ModelCapability, error)
}
```

Use internal result/request structs rather than leaking the `scriberr-engine`
types outside the provider package. This keeps future Python-server engines easy
to add.

Recommended internal request shape:

```go
type TranscriptionRequest struct {
    JobID     string
    UserID    uint
    AudioPath string
    ModelID   string
    Language  string
    Task      string
    Threads   int
}

type DiarizationRequest struct {
    JobID       string
    UserID      uint
    AudioPath   string
    ModelID     string
    NumSpeakers int
    MinSpeakers *int
    MaxSpeakers *int
}
```

Recommended internal result shape:

```go
type TranscriptionResult struct {
    Text     string
    Language string
    Words    []TranscriptWord
    Segments []TranscriptSegment
    ModelID  string
    EngineID string
}

type DiarizationResult struct {
    Segments []DiarizationSegment
    ModelID  string
    EngineID string
}
```

Local provider behavior:

- `ID()` returns `local`.
- Default transcription model is `whisper-base`.
- Default diarization model is `diarization-default`.
- Map provider requests to `scriberr-engine/speech/engine` requests.
- Force token timestamps on local transcription unless explicitly impossible.
- Do not expose local audio/model paths in returned errors sent to API clients.
- `Capabilities` should use the engine model registry and include install state
  via `IsModelInstalled`.

## Queue Architecture

### Source Of Truth

SQLite is the durable source of truth. In-memory structures are only for:

- Waking workers immediately after enqueue.
- Tracking cancel funcs for currently running jobs.
- Exposing process-local running counts.

### Schema Additions

Add these columns to `transcriptions`:

| Column | Type | Purpose |
| --- | --- | --- |
| `queued_at` | timestamp nullable | When job became queued. |
| `started_at` | timestamp nullable | First current attempt start time. |
| `failed_at` | timestamp nullable | Current terminal failure time. |
| `progress` | real not null default 0 | Current progress from 0.0 to 1.0. |
| `progress_stage` | varchar nullable | Current stage label. |
| `claimed_by` | varchar nullable | Worker ID that owns current lease. |
| `claim_expires_at` | timestamp nullable | Lease expiry for recovery. |
| `engine_id` | varchar nullable | Provider selected for this run, e.g. `local`. |

Indexes:

- `idx_transcriptions_queue_claim` on `(status, queued_at)`
- `idx_transcriptions_claim_expires_at` on `claim_expires_at`
- Existing `user_id` indexes remain required.

Add execution fields if not already present:

- Ensure `transcription_executions` can store `provider`, `model_name`,
  `model_family`, `started_at`, `completed_at`, `failed_at`, `error_message`,
  `output_json_path`, and request/config JSON.

Do not add a separate jobs table in this sprint. The existing `transcriptions`
table already represents the async job.

### Queue Service Interface

Create a service package, recommended path:

```txt
internal/transcription/worker
```

Public service interface:

```go
type QueueService interface {
    Enqueue(ctx context.Context, jobID string) error
    Cancel(ctx context.Context, userID uint, jobID string) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Stats(ctx context.Context, userID uint) (QueueStats, error)
}
```

Handlers should depend on this interface, not `internal/queue.TaskQueue`.

### Claiming Semantics

Worker loop:

1. Wait for wake signal or poll interval.
2. Try to claim one queued job.
3. If no job, wait again.
4. If claimed, process with a context tied to cancel map and server shutdown.
5. Renew lease periodically while processing.
6. On completion/failure/cancel, clear claim and write terminal state.

Claim query rules:

- Only claim `status = queued`.
- Order by `queued_at ASC, created_at ASC, id ASC`.
- Claim in a transaction.
- Set `status=processing`, `started_at=now`, `progress_stage=preparing`,
  `progress=0.05`, `claimed_by=workerID`, `claim_expires_at=now+leaseTimeout`.
- Future multi-user fairness:
  - Keep claim code isolated behind `SchedulerPolicy`.
  - Initial policy is FIFO for current single-user mode.
  - Include `user_id` in the claimed row and stats paths.

Startup recovery:

- `queued` jobs remain queued.
- `processing` jobs with expired/missing claims become queued with
  `progress_stage=recovered`, `claimed_by=NULL`, `claim_expires_at=NULL`.
- `processing` jobs with non-expired claims from a prior process should still be
  recovered on startup because process-local owners are gone.

Cancellation:

- If queued, set `status=canceled`, `progress_stage=canceled`,
  `claim_expires_at=NULL`.
- If running in this process, call cancel func; processing code should observe
  context cancellation and persist `status=canceled`.
- If marked processing but not running locally, mark canceled in DB and clear
  claim.
- Return `409` for completed/failed/canceled jobs.

## Orchestration

Recommended package:

```txt
internal/transcription/orchestrator
```

Main service:

```go
type Processor struct {
    Jobs repository.JobRepository
    Providers engineprovider.Registry
    Events EventPublisher
    Logs JobLogger
}
```

Processing stages and progress:

| Stage | Progress | Event |
| --- | ---: | --- |
| `queued` | 0.00 | `transcription.queued` |
| `preparing` | 0.05 | `transcription.progress` |
| `transcribing` | 0.20 | `transcription.progress` |
| `diarizing` | 0.70 | `transcription.progress` |
| `merging` | 0.85 | `transcription.progress` |
| `saving` | 0.95 | `transcription.progress` |
| `completed` | 1.00 | `transcription.completed` |
| `failed` | current | `transcription.failed` |
| `canceled` | current | `transcription.canceled` |

Execution flow:

1. Load job by ID and user ID from claim result.
2. Create a `transcription_executions` row with:
   - `status=processing`
   - `started_at=now`
   - `provider=local`
   - selected model IDs in request/config JSON
3. Validate source audio path exists internally.
4. Resolve provider and model IDs:
   - Use job/profile model option when present.
   - Otherwise local transcription default `whisper-base`.
   - Diarization only when job parameters request it.
5. Update progress `transcribing`.
6. Call provider `Transcribe`.
7. If diarization enabled, update progress `diarizing` and call provider
   `Diarize`.
8. Update progress `merging`; assign speakers to words/segments.
9. Update progress `saving`; persist transcript JSON and optional output file.
10. Mark job and execution completed together.
11. Publish final events.

Failure flow:

- Engine/model/download errors mark job failed with sanitized `last_error`.
- Context cancellation from user cancel marks job canceled, not failed.
- Internal logs may include detailed errors; API errors must not leak paths,
  tokens, model cache locations, or command output.

## Transcript Data Contract

Persist transcript JSON in `transcriptions.transcript_text`. Also write the same
JSON to `data/transcripts/{jobID}/transcript.json` and store
`output_json_path` internally.

Canonical JSON:

```json
{
  "text": "Full transcript text",
  "language": "en",
  "segments": [
    {
      "id": "seg_000001",
      "start": 0.0,
      "end": 4.2,
      "speaker": "SPEAKER_00",
      "text": "Hello world."
    }
  ],
  "words": [
    {
      "start": 0.0,
      "end": 0.4,
      "word": "Hello",
      "speaker": "SPEAKER_00"
    }
  ],
  "engine": {
    "provider": "local",
    "transcription_model": "whisper-base",
    "diarization_model": "diarization-default"
  }
}
```

Rules:

- `words` must always be an array in API responses.
- If the engine returns no words, store and return `words: []`.
- If the engine returns words but no segments, generate simple segments:
  - Prefer one segment for the full text if no better segmentation exists.
  - Segment boundaries should not invent word timestamps; only use known word
    times.
- If diarization is not requested or unavailable, omit speaker fields.
- Speaker labels are public-safe strings: `SPEAKER_00`, `SPEAKER_01`, etc.
- Existing plain-text or older JSON transcript rows must continue to render:
  - Plain text becomes `{ text, segments: [], words: [] }`.
  - Missing words becomes `words: []`.

## API Changes

### Create/Submit/Retry

- After creating a transcription row, call `QueueService.Enqueue`.
- If enqueue fails because queue is shutting down, return `503`.
- The job remains durable; a later recovery can still process queued jobs.

### Get Transcription

Add fields:

```json
{
  "progress": 0.42,
  "progress_stage": "transcribing",
  "started_at": "...",
  "completed_at": null,
  "failed_at": null
}
```

### Transcript Endpoint

`GET /api/v1/transcriptions/{id}/transcript` should parse canonical transcript
JSON and return:

```json
{
  "transcription_id": "tr_...",
  "text": "...",
  "segments": [],
  "words": []
}
```

### Events

Progress event shape:

```json
{
  "id": "tr_...",
  "status": "processing",
  "progress": 0.42,
  "stage": "transcribing"
}
```

Do not include local file paths, model cache paths, raw errors, or tokens.

### Executions Endpoint

Implement `GET /api/v1/transcriptions/{id}/executions`:

```json
{
  "items": [
    {
      "id": "exec_...",
      "transcription_id": "tr_...",
      "status": "completed",
      "provider": "local",
      "model": "whisper-base",
      "started_at": "...",
      "completed_at": "...",
      "failed_at": null,
      "processing_duration_ms": 12345,
      "error": null
    }
  ],
  "next_cursor": null
}
```

### Logs Endpoint

Implement `GET /api/v1/transcriptions/{id}/logs` as authenticated plain text.

Initial implementation may derive logs from persisted execution/job events if a
dedicated log file is not present. Logs must be sanitized:

- No absolute local paths.
- No API keys/tokens.
- No raw command output from external tools.
- No model cache directory leakage.

### Models Endpoint

`GET /api/v1/models/transcription` should return local engine models:

```json
{
  "items": [
    {
      "id": "whisper-base",
      "name": "Whisper Base",
      "provider": "local",
      "installed": true,
      "default": true,
      "capabilities": ["transcription", "word_timestamps"]
    }
  ]
}
```

Include diarization model capabilities either in this response or a future
`/models/diarization`; for this sprint, it is acceptable to include
`diarization-default` in the same list with `capabilities: ["diarization"]`.

## Repository Changes

Add repository methods rather than scattering `gorm.DB` updates:

```go
EnqueueTranscription(ctx, jobID string, now time.Time) error
ClaimNextTranscription(ctx, workerID string, leaseUntil time.Time) (*TranscriptionJob, error)
RenewClaim(ctx, jobID, workerID string, leaseUntil time.Time) error
UpdateProgress(ctx, jobID string, progress float64, stage string) error
CompleteTranscription(ctx, jobID string, transcriptJSON string, outputPath *string, completedAt time.Time) error
FailTranscription(ctx, jobID string, message string, failedAt time.Time) error
CancelTranscription(ctx, jobID string, canceledAt time.Time) error
ListExecutions(ctx, jobID string) ([]TranscriptionJobExecution, error)
```

Completion/failure methods should update the job and latest execution
consistently. Prefer transactions for terminal state changes.

## Deleting Legacy Adapter Stack

Remove or stop compiling:

- `internal/transcription/adapters/**`
- legacy registry/pipeline abstractions that only serve Python adapters
- server startup calls that register Python adapters or prepare Python envs
- obsolete tests tied only to Python adapter behavior

Keep or replace:

- `internal/transcription/queue_integration.go` only if it becomes a thin
  compatibility wrapper over the new worker processor.
- Existing API tests should remain and be updated around the new queue service.

## Documentation And Setup UX

Update docs in the same implementation sprint:

- README configuration table:
  - Remove or de-emphasize `WHISPERX_ENV`.
  - Add `SPEECH_ENGINE_*` and `TRANSCRIPTION_*` variables.
- Docker compose:
  - Persist `data/models` or map `SPEECH_ENGINE_CACHE_DIR`.
  - CPU config should work with no model pre-download.
  - CUDA config should keep NVIDIA runtime setup and use provider `auto`.
- Troubleshooting:
  - Missing `ffmpeg`.
  - Model download failure.
  - Forced CUDA unavailable.
  - First job slow because models are being downloaded.

## Test Strategy

### Unit Tests

- Config parsing:
  - Defaults.
  - Invalid provider.
  - Invalid duration/integer.
  - Auto-download default.
- Engine provider:
  - Maps Scriberr requests to engine requests.
  - Returns words when engine returns words.
  - Handles empty words.
  - Sanitizes errors.
- Transcript mapper:
  - Words present.
  - Words absent.
  - Diarization present.
  - Plain-text legacy transcript fallback.
  - Older JSON without `words`.
- Speaker merge:
  - Word overlap assignment.
  - Segment overlap assignment.
  - No diarization leaves speakers absent.
- Queue:
  - Claim returns oldest queued job.
  - Concurrent claims do not duplicate jobs.
  - Lease renewal updates only owning worker.
  - Startup recovery requeues orphaned processing jobs.
  - Cancel queued and cancel running.

### API Tests

- Create transcription enqueues and returns `queued`.
- Submit upload creates file, creates transcription, and enqueues.
- Fake engine worker completes a job and transcript endpoint returns text,
  segments, and words.
- Fake engine without words returns `words: []`.
- Events stream receives progress and completed events.
- Executions endpoint returns execution metadata.
- Logs endpoint returns sanitized text.
- Models endpoint returns engine model capabilities.
- No response/event leaks upload path, model cache path, or temp path.

### Integration Tests

- Default CI uses fake engine provider.
- Real engine tests are opt-in with env, for example:
  - `SCRIBERR_ENGINE_ITEST=1`
  - `SPEECH_ENGINE_AUTO_DOWNLOAD=true`
- Real tests should use a short fixture and may be skipped if runtime/model
  dependencies are unavailable.

## Implementation Order

1. Add this spec file and commit.
2. Add config fields/env parsing and tests.
3. Add engine provider abstraction and local provider wrapper with fake-provider
   tests.
4. Add queue schema migration and repository methods.
5. Implement durable worker service with fake processor tests.
6. Implement orchestrator and transcript mapping.
7. Wire API create/submit/retry/cancel/models/executions/logs to services.
8. Remove legacy Python adapter stack and startup bootstrap.
9. Update docs and Docker env examples.
10. Run full backend validation and commit in focused chunks.

## Acceptance Criteria

- Fresh install starts without manual model setup.
- First transcription job downloads missing model artifacts automatically when
  auto-download is enabled.
- A queued transcription transitions through processing to completed or failed.
- Completed transcript includes text, segments, and word timestamps when
  available.
- Missing word timestamps are handled as `words: []`.
- Diarization assigns speaker labels when requested and available.
- API events provide status/progress updates.
- Executions and logs endpoints are implemented.
- Queue survives restart and recovers orphaned processing jobs.
- No multi-track code returns.
- No legacy Python adapter bootstrap remains in server startup.
