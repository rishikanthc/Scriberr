# Sprint Run Tracker: ASR Parameter Contract Hardening

Run ID: `ASR-PARAM-CONTRACT`

Status: completed.

This tracker belongs to `devnotes/v2.0.0/sprint-plans/asr-parameter-contract-sprint-plan.md`.

## Run Rules

- Engine descriptors own local parameter facts.
- Backend validation enforces descriptor semantics.
- Frontend renders from descriptors.
- Do not solve `sherpa.model_type` with long-lived frontend key-specific behavior.
- Update this tracker in the same change set as each completed sprint.

## Validation Checklist

- [ ] Engine tests when descriptor code changes.
- [ ] Backend profile/model-card tests when contract mapping changes.
- [ ] Frontend build or focused tests when renderer types change.
- [ ] `git diff --check`.

## ASR-PARAM-CONTRACT-Sprint 0: Contract Design And Engine Descriptor Tests

Status: completed

Planned tasks:

- [x] Add read-only/mutability field to engine `ParameterDescriptor`.
- [x] Add descriptor validation coverage.
- [x] Mark `sherpa.model_type` read-only in Parakeet descriptors.
- [x] Test Parakeet descriptors expose read-only `sherpa.model_type`.
- [x] Confirm other editable parameters remain editable.

Acceptance checks:

- [x] Engine descriptor schema can express read-only parameters.
- [x] Parakeet model type is exposed but not editable by contract.

Verification:

- [x] `GOCACHE=/tmp/scriberr-engine-go-cache go test ./speech/providers ./speech/providers/sherpa/catalog`

Artifacts:

- `references/engine/speech/providers/descriptor.go`
- `references/engine/speech/providers/descriptor_test.go`
- `references/engine/speech/providers/sherpa/catalog/descriptors.go`
- `references/engine/speech/providers/sherpa/catalog/catalog_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-parameter-contract-sprint-tracker.md`

Commit:

- Pending.

## ASR-PARAM-CONTRACT-Sprint 1: Backend Model Card Mapping And Validation

Status: completed

Planned tasks:

- [x] Add read-only field to `asrcontract.ParameterDescriptor`.
- [x] Map read-only metadata in local provider adapter.
- [x] Update schema validation.
- [x] Reject changed read-only values in profile option validation.
- [x] Add tests for omitted/default/changed `sherpa.model_type`.

Acceptance checks:

- [x] Backend model cards expose read-only metadata.
- [x] Backend rejects attempts to change `sherpa.model_type`.
- [x] No profile save path relies on frontend-only enforcement.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/asrcontract ./internal/transcription/engineprovider ./internal/profile`

Artifacts:

- `internal/transcription/asrcontract/types.go`
- `internal/transcription/asrcontract/types_test.go`
- `internal/transcription/engineprovider/local_provider.go`
- `internal/transcription/engineprovider/local_provider_test.go`
- `internal/profile/service_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-parameter-contract-sprint-tracker.md`

Commit:

- Pending.

## ASR-PARAM-CONTRACT-Sprint 2: Frontend Renderer Support

Status: completed

Planned tasks:

- [x] Add read-only field to frontend `ParameterDescriptor` type.
- [x] Defer disabled/metadata rendering to `ASR-PROFILE-FE` dynamic form implementation.
- [x] Defer advanced section placement to `ASR-PROFILE-FE` dynamic form implementation.
- [x] Backend validation prevents changed read-only values from being saved.
- [x] Backend tests cover read-only string parameters.

Acceptance checks:

- [x] Frontend contract types can receive `read_only` descriptors.
- [x] Generic parameter renderer work can use descriptor metadata without key-specific logic in `ASR-PROFILE-FE`.

Verification:

- [x] `npm --prefix web/frontend run build`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract ./internal/profile`
- [x] `git diff --check -- internal/transcription/asrcontract/types.go internal/transcription/asrcontract/types_test.go internal/transcription/engineprovider/local_provider.go internal/transcription/engineprovider/local_provider_test.go internal/profile/service_test.go web/frontend/src/features/settings/api/profilesApi.ts devnotes/v2.0.0/sprint-trackers/asr-parameter-contract-sprint-tracker.md`

Artifacts:

- `web/frontend/src/features/settings/api/profilesApi.ts`
- `devnotes/v2.0.0/sprint-trackers/asr-parameter-contract-sprint-tracker.md`

Commit:

- Pending.
