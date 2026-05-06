# ASR Provider Backend Architecture

## Purpose

Scriberr needs one stable ASR provider contract that supports the bundled sherpa-onnx engine and future third-party providers without pushing model-specific logic into API handlers, profiles, queue workers, or persistence code.

The default sherpa-onnx provider remains in-process and is called through Go APIs. External providers run as separate containers and are called through a standard REST contract. Both provider types implement the same internal Go interface so the orchestrator can compose, chain, and replace providers without knowing transport details.

## Goals

- Keep Scriberr responsible for queueing, retries, cancellation, audio preprocessing, transcript persistence, authorization, and canonical transcript merging.
- Keep providers responsible for model discovery, model lifecycle, inference, provider-local status, and typed progress events.
- Allow providers to expose richer model cards and optional features such as diarization, speaker identification, token timestamps, custom vocabulary, language detection, translation, and hardware-specific backends.
- Allow providers to expose a model registry with typed parameter schemas so backend validation and frontend ASR profile controls are generated from the same source of truth.
- Keep long-form decode planning, chunking, batching, timestamp stitching, and result normalization owned by the execution runtime that performs model decode.
- Allow a single transcription job to run as a provider pipeline, for example local sherpa transcription followed by remote diarization.
- Make provider development practical for third parties by keeping the external wire contract simple, testable, and language agnostic.

## Non-Goals

- Providers must not own Scriberr's durable queue.
- Providers must not read or write Scriberr database rows.
- Providers must not return final API DTOs.
- Providers must not construct user-visible file paths or public IDs.
- External providers do not need gRPC or WebSocket support for v1 batch transcription.

## Package Shape

Target package boundaries:

```txt
internal/transcription/asrcontract
  Pure provider contract types: provider info, model cards, requests, results, progress, errors.

internal/transcription/preprocess
  Audio normalization to provider-ready artifacts such as 16 kHz mono WAV.

internal/transcription/engineprovider
  Internal Provider interface, registry, model selection, provider status aggregation.

internal/transcription/engineprovider/local
  In-process sherpa-onnx adapter. This is the only ASR path that imports scriberr-engine.

internal/transcription/engineprovider/remote
  REST client adapter for external provider containers.

internal/transcription/orchestrator
  Job pipeline execution, progress mapping, provider chaining, canonical transcript creation.
```

Current code may keep `engineprovider` as the package name during migration. The important rule is that API/profile/orchestrator callers depend on Scriberr contract types, not `scriberr-engine` model metadata.

## Dependency Rules

Allowed:

```txt
orchestrator -> engineprovider, preprocess, repository
engineprovider -> asrcontract
engineprovider/local -> scriberr-engine
engineprovider/remote -> net/http, asrcontract
profile service -> engineprovider registry/model catalog interfaces
api -> profile/transcription services and response DTO mapping
```

Forbidden:

```txt
internal/api -> scriberr-engine
profile handlers -> scriberr-engine
orchestrator -> scriberr-engine
remote providers -> Scriberr database or repository packages
providers -> internal/api DTOs
```

## Internal Provider Interface

The internal Go provider base interface covers model discovery, status, lifecycle, and cleanup:

```go
type Provider interface {
    ID() string

    Inspect(ctx context.Context) (*ProviderInfo, error)
    Models(ctx context.Context) ([]ModelCard, error)
    Status(ctx context.Context) (*ProviderStatus, error)

    LoadModel(ctx context.Context, req LoadModelRequest) error
    UnloadModel(ctx context.Context, req UnloadModelRequest) error
    LoadedModels(ctx context.Context) ([]LoadedModel, error)

    Close() error
}
```

Task execution is split into small task-specific interfaces:

```go
type TranscriptionProvider interface {
    Provider
    Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error)
}

type DiarizationProvider interface {
    Provider
    Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error)
}

type SpeakerIdentificationProvider interface {
    Provider
    IdentifySpeakers(ctx context.Context, req SpeakerIDRequest) (*SpeakerIDResult, error)
}
```

Providers implement only the task interfaces they can actually execute. The registry selects by advertised model capability, and the orchestrator verifies the selected provider implements the required task interface before execution.

`ProgressSink` is a Scriberr-owned callback. Local providers call it directly. Remote providers translate remote job progress into the same callback.

## Provider Registry

The registry owns provider discovery and selection.

Responsibilities:

- Register the bundled local sherpa provider when enabled.
- Register remote providers from explicit configuration.
- Probe remote provider health and metadata.
- Cache model cards with a short TTL.
- Select providers by explicit provider ID, model ID, required capabilities, load state, and health.
- Expose aggregated model/provider status to services.
- Derive selectable `ModelCapability` projections from `Models()` instead of requiring a second provider-level capability API.

Selection rules:

1. If a pipeline step pins provider and model, use that exact pair or fail.
2. If a step pins provider only, choose that provider's default model satisfying required capabilities.
3. If a step pins model only, choose any healthy provider exposing that model.
4. If neither is pinned, use Scriberr defaults.
5. If a requested feature is unavailable, fail before queue execution when possible.

### Registry V2 Model Descriptors

Model cards are evolving into provider-owned model descriptors. They remain JSON-friendly wire objects, but they must carry enough metadata for Scriberr to validate profiles, plan execution, and render model-specific frontend controls.

Required descriptor groups:

- identity: provider ID, model ID, family, display name, version, license, and install/load state
- tasks: transcription, diarization, speaker identification, audio tagging, translation, or language detection
- language support: supported languages, fixed language, automatic detection, or runtime language parameter
- output capabilities: word timestamps, segment timestamps, token timestamps, speaker labels, tags, emotions, language spans, and confidence when available
- runtime capabilities: CPU/CUDA/CoreML support, thread support, batching support, memory class, and load/reload behavior
- chunking capabilities: whether Scriberr may chunk audio, whether provider-owned chunking is supported, preferred chunking mode, recommended chunk size, max chunk size, and batching limits
- parameter schema: typed, bounded, frontend-renderable parameters with defaults

The provider registry is descriptive. The orchestrator execution planner remains responsible for choosing the final plan after applying user profile settings, global safety limits, and measured defaults.

### Parameter Schema Contract

Profile settings should be stored as generic provider/model parameters validated against each model descriptor, not as permanent provider-specific backend structs.

Parameter descriptors should include:

```json
{
  "key": "runtime.num_threads",
  "label": "CPU threads",
  "type": "integer",
  "default": 4,
  "min": 1,
  "max": 16,
  "scope": "runtime",
  "advanced": true,
  "requires_reload": true
}
```

Allowed scopes:

```txt
model
runtime
decoding
chunking
vad
output
postprocess
```

Rules:

- Common concepts should use common keys across providers.
- Provider-specific keys must be namespaced, bounded, and documented.
- Frontend-generated controls are convenience only; the backend must validate every submitted parameter.
- Parameters that affect recognizer construction, such as sherpa thread count or model provider, must be marked `requires_reload`.
- Runtime paths, secrets, raw URLs, and credentials are not allowed in profile parameter values.

## Execution Planning, Chunking, And Batching

Scriberr backend owns job-level orchestration, not model-level chunk planning:

```txt
profile + provider registry + model descriptor + audio metadata
  -> validate parameters
  -> ensure/preload model when needed
  -> preprocess audio once
  -> call selected provider task with pipeline step options
  -> merge typed task results into canonical transcript
```

For the bundled local sherpa provider, `scriberr-engine` owns the model-level execution plan: fixed-window chunking, VAD chunking, batching, timestamp offsetting, provider result stitching, and decode metrics. The backend passes full normalized audio and descriptor-keyed `pipeline[].options` through the direct Go adapter.

For external REST providers, the provider service owns any long-form segmentation needed for its model. The backend does not split audio into spans for remote providers in this contract.

Chunking modes:

```txt
none      send full audio when model/provider can handle it safely
fixed     split by fixed duration with optional overlap
vad       split by speech spans from Scriberr-owned VAD planning
provider  send full audio and let provider plan chunks internally
```

Batching belongs to the execution runtime plan, not the backend transcript result contract. A provider can advertise batch decode support while recommending `batch_size: 1` when measured CPU performance is better without batching.

Initial measured default for sherpa Parakeet v3 on CPU:

```txt
chunking.mode: fixed
chunking.chunk_seconds: 30
runtime.num_threads: 4
batching.batch_size: 1
```

VAD remains available but should be selected by profile or workload. It is useful for silence-heavy audio; fixed-window chunking gave the best accuracy in the current dense-speech experiments.

## Task-Oriented Provider Capabilities

Avoid one large provider method surface that assumes every provider supports every future feature. Capability execution is task-oriented:

```txt
ASR/transcription
diarization
speaker identification
audio tagging
language detection
translation
```

Each task should have:

- input contract
- output contract
- capability flags
- parameter schema scopes
- progress stages
- result normalization rules

This keeps future speaker identification and audio tagging support from contaminating the ASR-only hot path.

## Audio Preprocessing

Scriberr owns audio preprocessing before provider execution.

Provider-ready audio contract:

```txt
sample_rate: 16000
channels: 1
format: wav initially
path_mode: mounted_file
```

Flow:

```txt
source upload/import/recording
  -> file ownership and readiness checks
  -> normalized audio artifact by file hash/job id
  -> provider receives mounted normalized path
```

Rules:

- Public API responses never expose normalized artifact paths.
- Provider paths are internal execution details and must be sanitized from errors/logs.
- Normalized artifacts may be cached by source file hash.
- Providers may declare additional accepted formats, but v1 providers must accept normalized WAV.
- External provider containers must mount the same normalized audio root at a configured provider-visible path.

## Provider Progress

Progress is discrete and sporadic. It does not need low-latency streaming, but it must be structured.

Standard stages:

```txt
accepted
preprocessing
loading_model
transcribing
diarizing
identifying_speakers
postprocessing
completed
failed
canceled
```

Provider progress event:

```json
{
  "stage": "loading_model",
  "progress": 0.2,
  "message": "Loading parakeet-v3",
  "operation": "transcription",
  "model": "parakeet-v3",
  "timestamp": "2026-05-03T12:00:00Z"
}
```

`progress` is optional but must be between `0` and `1` when present. Scriberr maps provider stages into job progress/events and may add higher-level stages such as `saving` and `merging`.

## Provider Status And Capacity

Each provider processes at most one job at a time in v1. Scriberr still owns the durable queue and decides when to call a provider.

Provider status:

```json
{
  "state": "busy",
  "active_job": {
    "id": "job_123",
    "operation": "diarization",
    "model": "pyannote-3.1",
    "stage": "diarizing",
    "progress": 0.62
  },
  "loaded_models": [
    {
      "id": "pyannote-3.1",
      "loaded_at": "2026-05-03T12:00:00Z",
      "memory_mb": 4200
    }
  ],
  "capacity": {
    "max_concurrent_jobs": 1,
    "available_slots": 0
  }
}
```

Valid provider states:

```txt
starting
idle
busy
degraded
unhealthy
stopping
```

If a remote provider is busy, it must reject new work with `PROVIDER_BUSY`. Scriberr can retry later or select another provider.

## Model Lifecycle

Providers expose explicit model load and unload operations.

Load policy on execution requests:

```txt
auto      provider may load the model if needed
require   fail if the model is not already loaded
reload    reload the model before execution when supported
```

Rules:

- `LoadModel` returns after the model is resident or fails with a typed error.
- `UnloadModel` must fail with `PROVIDER_BUSY` if unloading would disrupt an active job.
- Providers may evict models according to their own memory policy, but must report actual loaded state.
- Scriberr may preload defaults during startup or before claiming a job, but startup must not require all models to be loaded.

## Model Cards

Model cards are the source of truth for profile validation and frontend controls.

Minimum model card:

```json
{
  "id": "parakeet-v3",
  "display_name": "NVIDIA Parakeet TDT v3",
  "provider": "local-sherpa",
  "family": "nemo_transducer",
  "version": "0.6b-v3",
  "installed": true,
  "loaded": false,
  "default": false,
  "tasks": ["transcribe"],
  "languages": ["en"],
  "capabilities": {
    "transcription": true,
    "diarization": false,
    "speaker_identification": false,
    "translation": false,
    "word_timestamps": true,
    "segment_timestamps": true,
    "token_timestamps": false,
    "streaming": false,
    "custom_vocabulary": false,
    "initial_prompt": false,
    "language_detection": false
  },
  "limits": {
    "max_audio_duration_sec": null,
    "recommended_chunk_sec": 30
  },
  "resource_requirements": {
    "backends": ["cpu", "cuda"],
    "recommended_vram_mb": null,
    "recommended_ram_mb": null
  },
  "parameter_schema": {},
  "license": "",
  "source_url": ""
}
```

Use typed capability fields for stable features. Experimental/provider-specific features may live under `extensions`.

## Profiles And Pipelines

Profiles should evolve from a single model/options bag into an ordered provider pipeline.

Example:

```json
{
  "name": "Parakeet + external diarization",
  "pipeline": [
    {
      "kind": "transcription",
      "provider": "local-sherpa",
      "model": "parakeet-v3",
      "options": {
        "decoding.method": "greedy_search",
        "chunking.mode": "fixed",
        "chunking.chunk_seconds": 30,
        "output.word_timestamps": true,
        "output.timestamps": true
      }
    },
    {
      "kind": "diarization",
      "provider": "pyannote-remote",
      "model": "pyannote-3.1",
      "inputs": ["audio"]
    },
    {
      "kind": "speaker_identification",
      "provider": "speaker-id-remote",
      "model": "ecapa",
      "inputs": ["audio", "diarization_segments"]
    }
  ]
}
```

Common options should use provider-neutral descriptor keys inside the owning step's `options` map:

```txt
runtime.num_threads
decoding.method
chunking.mode
chunking.chunk_seconds
output.word_timestamps
output.timestamps
vad.threshold
```

Provider-specific options belong in the same `options` map with a namespace and must be validated against the selected model card's `parameter_schema`.

## Orchestrator Flow

Target execution flow:

```txt
worker claims durable job
  -> orchestrator resolves profile pipeline
  -> preprocess source audio once
  -> for each pipeline step:
       select provider/model
       check provider status/capacity
       call provider operation with ProgressSink
       collect typed artifact
  -> merge artifacts into canonical transcript
  -> save transcript JSON/artifacts
  -> update execution/job terminal state
```

The orchestrator owns chaining. Providers return typed partial artifacts:

```txt
TranscriptionResult: text, segments, words, language, metadata
DiarizationResult: speaker segments, speaker labels, metadata
SpeakerIDResult: speaker labels, speaker embeddings or identities when available
```

Providers must not return final Scriberr transcript JSON. Canonical transcript construction stays in Scriberr.

## External REST Contract

External provider containers implement REST endpoints.

Control plane:

```http
GET    /v1/health
GET    /v1/provider
GET    /v1/models
GET    /v1/status
GET    /v1/models/loaded
POST   /v1/models/{model_id}:load
POST   /v1/models/{model_id}:unload
```

Execution plane:

```http
POST   /v1/jobs
GET    /v1/jobs/{job_id}
GET    /v1/jobs/{job_id}/events
DELETE /v1/jobs/{job_id}
```

`POST /v1/jobs` creates one ephemeral provider job. This is not a durable queue. Providers may persist enough in-memory state to report progress and result while the request is active, but Scriberr remains the durable source of truth.

Job request:

```json
{
  "request_id": "tr_abc_exec_001_step_1",
  "operation": "transcription",
  "audio": {
    "path": "/provider-input/audio/job-123.wav",
    "sample_rate": 16000,
    "channels": 1,
    "format": "wav",
    "duration_sec": 412.7
  },
  "model": "parakeet-v3",
  "load_policy": "auto",
  "task": "transcribe",
  "language": "en",
  "features": {
    "word_timestamps": true,
    "segment_timestamps": true
  },
  "options": {}
}
```

Job status:

```json
{
  "id": "job_123",
  "request_id": "tr_abc_exec_001_step_1",
  "state": "running",
  "stage": "transcribing",
  "progress": 0.48,
  "result": null,
  "error": null
}
```

Terminal job status includes either `result` or `error`. `GET /v1/jobs/{job_id}/events` may return recent events as JSON array in v1. SSE can be added later without changing the internal provider interface.

## Error Contract

Provider errors must be typed and sanitized.

```json
{
  "error": {
    "code": "MODEL_NOT_INSTALLED",
    "message": "Model parakeet-v3 is not installed.",
    "retryable": false,
    "details": {
      "model": "parakeet-v3"
    }
  }
}
```

Standard codes:

```txt
INVALID_REQUEST
UNSUPPORTED_OPERATION
UNSUPPORTED_FEATURE
UNSUPPORTED_MODEL
MODEL_NOT_INSTALLED
AUDIO_NOT_FOUND
AUDIO_INVALID
INSUFFICIENT_RESOURCES
PROVIDER_BUSY
PROVIDER_UNHEALTHY
INFERENCE_FAILED
CANCELED
TIMEOUT
```

Scriberr maps provider errors to queue failure/cancel/retry behavior. API responses and logs must not expose provider stack traces, local filesystem roots, tokens, or container internals.

## Configuration

Initial configuration should be explicit.

Example:

```txt
ASR_LOCAL_SHERPA_ENABLED=true
ASR_DEFAULT_PROVIDER=local-sherpa
ASR_REMOTE_PROVIDERS=pyannote=http://scriberr-asr-pyannote:8081,speaker-id=http://scriberr-speaker-id:8081
ASR_NORMALIZED_AUDIO_DIR=data/asr-normalized
ASR_PROVIDER_AUDIO_MOUNT=/provider-input/audio
```

Avoid automatic Docker discovery in v1. Static configuration is easier to audit and debug.

## Security And Isolation

- Remote provider URLs are admin/system configuration, not user input.
- Provider requests must include only provider-visible mounted paths, not public URLs or raw storage internals.
- Provider auth can start as private Docker network only, but the contract should allow an optional shared token header later.
- Provider responses must be size-limited and decoded with strict JSON limits.
- Provider `parameter_schema` controls validation, but Scriberr must still enforce global safety limits such as max duration, max payload size, and allowed feature names.
- Provider progress messages are treated as untrusted text and sanitized before API exposure.

## API Surface Impact

Existing public API can remain mostly stable initially:

```txt
GET /api/v1/models/transcription
POST/PATCH /api/v1/profiles
POST /api/v1/transcriptions
GET /api/v1/transcriptions/{id}/events
GET /api/v1/transcriptions/{id}/executions
```

Planned changes:

- `/models/transcription` should return model cards aggregated from the provider registry.
- Profile validation should use registry/model-card data, not `scriberr-engine`.
- Execution rows should store provider step details, model IDs, operation kind, sanitized options, and provider error codes.
- Profile options should be stored as `pipeline[].options`, validated against selected model descriptors.

## Migration Plan

1. Introduce pure contract types and replace free-form `ModelCapability` with `ModelCard`.
2. Move profile/model validation out of API handlers and into profile/transcription services using the provider registry.
3. Add `ProgressSink`, provider status, model lifecycle, and operation-specific result types to the internal provider interface.
4. Adapt the existing local sherpa provider to the new interface while keeping direct Go calls.
5. Add audio preprocessing that produces normalized provider-ready artifacts before provider execution.
6. Add the remote REST provider client and a fake/test remote provider.
7. Add pipeline execution in the orchestrator, initially supporting transcription plus optional diarization.
8. Expand profile persistence to store ordered pipeline steps with descriptor-keyed options.
9. Add contract tests and a minimal provider SDK/example container.

## Verification

Required tests:

- Local sherpa provider still works without REST.
- Remote provider client maps model cards, status, progress, results, and typed errors.
- Provider registry selects by provider ID, model ID, capability, health, and busy state.
- Profile validation rejects unsupported provider/model/feature combinations.
- Orchestrator can chain local transcription and remote diarization.
- Cancellation propagates to local providers through context and to remote providers through `DELETE /v1/jobs/{job_id}`.
- Provider errors are sanitized in API responses, logs, and execution records.
- Normalized audio paths are never returned from public APIs.

Opt-in real provider tests should remain gated so CI can run fast fake-provider tests by default.
