# Sprint Run Tracker: ASR Parameter Contract Hardening

Run ID: `ASR-PARAM-CONTRACT`

Status: not started.

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

Status: pending

Planned tasks:

- [ ] Add read-only/mutability field to engine `ParameterDescriptor`.
- [ ] Add descriptor validation tests.
- [ ] Mark `sherpa.model_type` read-only in Parakeet descriptors.
- [ ] Test Parakeet v2/v3 expose read-only `sherpa.model_type`.
- [ ] Confirm Whisper editable parameters remain editable.

Acceptance checks:

- [ ] Engine descriptor schema can express read-only parameters.
- [ ] Parakeet model type is exposed but not editable by contract.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-PARAM-CONTRACT-Sprint 1: Backend Model Card Mapping And Validation

Status: pending

Planned tasks:

- [ ] Add read-only field to `asrcontract.ParameterDescriptor`.
- [ ] Map read-only metadata in local provider adapter.
- [ ] Update schema validation.
- [ ] Reject changed read-only values in profile option validation.
- [ ] Add tests for omitted/default/changed `sherpa.model_type`.

Acceptance checks:

- [ ] Backend model cards expose read-only metadata.
- [ ] Backend rejects attempts to change `sherpa.model_type`.
- [ ] No profile save path relies on frontend-only enforcement.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-PARAM-CONTRACT-Sprint 2: Frontend Renderer Support

Status: pending

Planned tasks:

- [ ] Add read-only field to frontend `ParameterDescriptor` type.
- [ ] Render read-only parameters disabled or as metadata.
- [ ] Include read-only parameters in advanced sections.
- [ ] Avoid submitting changed read-only values.
- [ ] Add tests for read-only string parameters.

Acceptance checks:

- [ ] Frontend exposes `sherpa.model_type` while preventing edits.
- [ ] Generic parameter renderer supports read-only descriptors without key-specific logic.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.
