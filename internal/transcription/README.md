# Transcription Runtime

Scriberr's default transcription path is the local Go engine worker stack.

## Active Packages

- `engineprovider`: provider boundary around local speech engines. This is the only package that imports `scriberr-engine`.
- `orchestrator`: turns claimed jobs into canonical transcript JSON, creates execution rows, publishes progress events, and writes transcript artifacts.
- `worker`: durable queue workers backed by SQLite claims, leases, cancellation, recovery, and stats.

## Runtime Flow

1. API handlers create durable transcription rows.
2. The queue service enqueues the row and wakes workers.
3. Workers claim jobs from SQLite and renew leases while running.
4. The orchestrator resolves provider/model/options, calls the provider, maps canonical transcript JSON, and returns a worker result.
5. The worker marks jobs completed, failed, or canceled through repository terminal updates.

SQLite is the source of truth. In-memory state is used only for worker wakeups and process-local cancellation.

## Canonical Transcript JSON

Completed jobs store canonical JSON in `transcriptions.transcript_text` and write the same JSON to `data/transcripts/{jobID}/transcript.json`.

The API must always expose `words` as an array. Older plain-text rows and older JSON without `words` are parsed through the compatibility mapper in `orchestrator`.

## Legacy Python Stack

The old Python adapter registry, pipeline, unified service, quick transcription service, and adapter tests are behind the `legacy_python` build tag. They are not part of normal server startup.

Do not add new runtime dependencies on:

- `internal/transcription/adapters`
- `internal/transcription/registry`
- `internal/transcription/pipeline`
- `internal/transcription/unified_service.go`
- `internal/transcription/queue_integration.go`
- `internal/transcription/quick_transcription.go`

## Verification

Default fake-provider validation:

```bash
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...
```

Opt-in real engine smoke:

```bash
SCRIBERR_ENGINE_ITEST=1 SPEECH_ENGINE_AUTO_DOWNLOAD=true GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider -run 'TestRealEngine'
```
