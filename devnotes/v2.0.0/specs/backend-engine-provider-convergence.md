# Backend Engine Provider Convergence Spec

## Intent

Scriberr backend and `scriberr-engine` should behave like one coherent system while keeping ownership boundaries clean. The local sherpa-onnx engine is an in-process Go dependency. It is not a REST provider. REST is reserved for future external providers.

## Final Ownership

Engine owns:

- local model registry descriptors
- typed parameter schema validation
- runtime defaults and reload-required metadata
- local chunking, VAD, batching, timestamp offsets, and stitching
- sherpa-onnx config mapping and decode behavior
- local transcript results, metrics, and sanitized execution plan summaries

Backend owns:

- durable transcription jobs and executions
- profile storage and validation against provider descriptors
- queueing, claims, retries, cancellation, and progress persistence
- audio preprocessing to provider-ready files
- provider pipeline ordering across transcription, diarization, speaker identification, and future tasks
- canonical transcript persistence and public API DTO mapping

## Required Direction

```txt
api -> transcription/profile services
profile service -> provider model catalog interface
orchestrator -> engineprovider registry
engineprovider/local -> scriberr-engine Go API
engineprovider/remote -> REST client for future external providers
```

Forbidden:

```txt
api -> scriberr-engine
profile -> scriberr-engine
orchestrator -> scriberr-engine
local sherpa -> REST
backend planner -> local fixed/VAD/batch chunk planning
```

## Local Provider Shape

The local provider adapter should be thin:

```go
type TranscriptionRequest struct {
    JobID      string
    UserID     uint
    AudioPath  string
    ModelID    string
    Parameters map[string]any
    Progress   ProgressSink
}
```

It should map directly to:

```go
speechengine.TranscriptionRequest{
    RequestID:  req.JobID,
    ModelID:    req.ModelID,
    AudioPath:  req.AudioPath,
    Parameters: req.Parameters,
    Progress:   localProgressSink{...},
}
```

No local provider code should set `chunking.mode`, `chunking.chunk_seconds`, `batching.batch_size`, `runtime.num_threads`, or model-specific sherpa parameters unless those values came from validated profile/job options or global engine config.

## Descriptor Flow

```txt
engine descriptor
  -> local provider adapter mechanical mapping
  -> backend asrcontract model card
  -> profile validation
  -> frontend dynamic controls
  -> job pipeline options
  -> engine request parameters
  -> engine execution plan
```

The backend must not recreate descriptor facts from model family strings.

## Chunking And VAD

For local engine execution:

- Backend passes `chunking.*`, `vad.*`, and `batching.*` as parameters only.
- Engine validates support and chooses the execution plan.
- Engine emits progress and metrics for chunks/batches.
- Backend persists plan summary and progress, but does not compute chunk plans.

For remote providers:

- Backend still validates parameters against remote model descriptors.
- Remote provider may own chunking if its descriptor says provider-owned chunking is required.
- Backend should not assume remote provider internals.

## Metrics And Execution Metadata

Engine metrics are authoritative for local execution:

- audio duration
- decode duration
- real-time factor
- chunk count
- batch size
- hypothesis words

Backend execution metadata should store those values plus sanitized plan summary. Backend should not infer local decode metrics from word/segment timestamps.

## Extensibility Rule

Adding a local model should usually require:

- engine model descriptor
- engine artifact requirements/downloads
- engine adapter config mapping when needed
- optional engine regression check

It should not require backend schema/default/chunking edits.

Adding a new provider should require:

- a provider adapter implementing model registry and advertised task interfaces
- descriptor-backed validation
- task result mapping into canonical transcript/task outputs

It should not require API/profile/orchestrator model-family conditionals.
