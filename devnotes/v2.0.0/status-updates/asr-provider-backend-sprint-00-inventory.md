# ASR Provider Backend Sprint 00 Inventory

Date: 2026-05-03

Related plan:

- `devnotes/v2.0.0/sprint-plans/asr-provider-backend-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Related spec:

- `devnotes/v2.0.0/specs/asr-provider-backend-architecture.md`

## Goal

Establish the ASR provider architecture baseline before runtime changes begin. Sprint 0 adds guardrails and documentation only. It does not change transcription behavior.

## Worktree Baseline

Unrelated local/untracked workspace entries present before Sprint 0 implementation:

```txt
.playwright-mcp/
.tmp/
DM_Sans,Nunito.zip
DM_Sans,Nunito/
references/
test-audio/
```

Sprint-owned documents already present from planning:

```txt
devnotes/v2.0.0/specs/asr-provider-backend-architecture.md
devnotes/v2.0.0/sprint-plans/asr-provider-backend-sprint-plan.md
devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md
```

Sprint-owned changes in this sprint:

```txt
internal/api/architecture_test.go
devnotes/v2.0.0/status-updates/asr-provider-backend-sprint-00-inventory.md
devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md
```

## Current ASR Coupling

Current production `scriberr-engine` imports:

```txt
internal/api/profile_handlers.go
internal/transcription/engineprovider/local_provider.go
```

The local provider import is intended for the in-process sherpa adapter. The profile handler import is a known compatibility exception: profile validation currently uses `scriberr-engine/speech/models`. ASRP-Sprint 3 must move profile validation behind a provider-backed model catalog and remove this API dependency.

Current provider capability surface:

```txt
internal/transcription/engineprovider/types.go
  ModelCapability uses free-form capability strings.

internal/transcription/engineprovider/registry.go
  Selects by provider, model, and required free-form capabilities.

internal/api/admin_handlers.go
  Maps ModelCapability into GET /api/v1/models/transcription responses.

internal/api/profile_handlers.go
  Validates and normalizes profile models through scriberr-engine model metadata.
```

Current orchestrator behavior:

```txt
internal/transcription/orchestrator/processor.go
  Resolves provider mostly by job.EngineID.
  Defaults transcription model to whisper-base.
  Defaults diarization model to diarization-default.
  Passes job.AudioPath directly to Transcribe and Diarize.
  Publishes coarse progress stages owned by the orchestrator.
```

Current audio/path behavior:

```txt
models.TranscriptionJob.AudioPath persists source_file_path.
orchestrator validates the source path with os.Stat.
local provider forwards AudioPath into scriberr-engine.
files/recording/mediaimport services own current source path construction.
API response and event tests already guard against source_file_path and raw path leakage.
```

## Compatibility Map

Existing `models.WhisperXParams` remains the compatibility profile/job parameter structure until later sprints introduce provider pipeline persistence.

Compatibility expectations:

- Existing profile create/update/list/get API JSON shape remains stable.
- Existing single-model profiles continue to resolve to one transcription step.
- `Parameters.Model` continues to default to `whisper-base`.
- `Parameters.Diarize` continues to create an optional diarization step.
- `Parameters.DiarizeModel` continues to default to `diarization-default`.
- Existing transcription jobs and execution rows remain readable.
- Public transcript JSON remains canonical `text`, `segments`, `words`, and `engine`.

Deferred compatibility migration:

- ASRP-Sprint 3 removes API-level sherpa model validation.
- ASRP-Sprint 8 introduces internal pipeline execution.
- ASRP-Sprint 9 persists ordered profile pipeline steps while preserving legacy fields.

## Route And API Impact Matrix

```txt
GET /api/v1/models/transcription
  Current: returns ModelCapability items from provider registry.
  Target: returns model-card-backed items without breaking current fields.

POST/PATCH /api/v1/profiles
  Current: handler validates model through scriberr-engine metadata.
  Target: service validates through provider model catalog.

GET /api/v1/profiles
GET /api/v1/profiles/{id}
  Current: returns legacy profile option fields.
  Target: keep compatible fields; pipeline data can be additive or versioned later.

POST /api/v1/transcriptions
  Current: resolves profile parameters into TranscriptionJob.Parameters.
  Target: create jobs from profile pipeline compatibility without changing queue ownership.

GET /api/v1/transcriptions/{id}/events
  Current: orchestrator emits coarse progress events.
  Target: provider progress maps into durable progress and small path-free events.

GET /api/v1/transcriptions/{id}/logs
  Current: returns sanitized job logs.
  Target: include provider-safe messages only, no paths/provider URLs/tokens.

GET /api/v1/transcriptions/{id}/executions
  Current: returns provider/model/status/timing/error.
  Target: store provider step metadata and error codes internally while preserving response compatibility unless intentionally changed.
```

## Guards Added

`internal/api/architecture_test.go` now includes ASR-specific guard coverage:

- Current `scriberr-engine` import inventory is fixed to `api/profile_handlers.go` and `transcription/engineprovider/local_provider.go`.
- Profile service production code must not import `scriberr-engine`.
- ASR provider production code must not import `internal/api` or `internal/repository`.

The import inventory is intentionally strict around today's known exception. New `scriberr-engine` imports fail tests. ASRP-Sprint 3 should remove the `api/profile_handlers.go` exception and tighten the expected list to only the local provider adapter.

## Verification

Run during Sprint 0:

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASREngineImportInventory|TestProfileServiceDoesNotImportSherpaEngine|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection'
git diff --check -- internal/api/architecture_test.go devnotes/v2.0.0/status-updates/asr-provider-backend-sprint-00-inventory.md devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md
```

Result: passed.

## Next Sprint

ASRP-Sprint 1 should introduce pure ASR contract types under `internal/transcription/asrcontract` with standard-library-only imports and JSON round-trip tests. No provider runtime behavior should change in Sprint 1.
