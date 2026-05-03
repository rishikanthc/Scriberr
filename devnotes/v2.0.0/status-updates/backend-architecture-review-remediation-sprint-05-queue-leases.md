# Backend Architecture Review Remediation Sprint 05 Queue Leases

Date: 2026-05-03

## Scope

Sprint 5 addresses:

- Finding 3: queue terminal updates were not claim-owned.
- Finding 4: recovery requeued every processing job.

## Changes

- Recovery now requeues only processing jobs with missing or expired leases.
- Added claim-owned terminal repository methods for complete, fail, and cancel transitions.
- Terminal writes now require:
  - `status = processing`;
  - current `claimed_by`;
  - matching `latest_execution_id`.
- Added `repository.ErrQueueClaimConflict` for stale worker/lost-claim conflicts.
- Updated the worker flow to use claim-owned terminal methods when the processor reports an execution ID.
- Updated the orchestrator processor to return the execution ID it creates.

## TDD Evidence

The new queue tests failed before implementation because claim-owned terminal methods did not exist:

```txt
repo.CompleteTranscriptionClaimed undefined
repo.FailTranscriptionClaimed undefined
```

After implementation, the new recovery and stale-worker tests passed.

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/orchestrator ./internal/api -run 'TestCapabilitiesQueue|TestEngineWorker'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/orchestrator
git diff --check
```

## Notes

The worker keeps a fallback to legacy terminal methods when a processor returns no execution ID. The real orchestrator now returns the execution ID; the fallback keeps narrower worker fakes and non-orchestrator test processors compatible.
