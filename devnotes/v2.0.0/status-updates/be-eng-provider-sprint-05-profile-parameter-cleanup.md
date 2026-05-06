# BE-ENG-PROVIDER Sprint 5: Profile Parameter Cleanup

Status: complete.

## Changes

- Profile API options are now pipeline-only.
- Model/runtime/chunking/decoding values must live under the owning pipeline step `options` map.
- Legacy top-level profile knobs such as `language`, `threads`, `chunking_strategy`, and `decoding_method` are rejected when present.
- Profile normalization validates step options against provider model descriptor schemas through the model catalog.
- Profile persistence derives display columns from the normalized transcription step instead of copying flat `ASRParams` fields.
- The orchestrator no longer fabricates a default execution pipeline from flat job fields.

## Verification

- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/profile`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/orchestrator`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/...`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/models`

## Notes

- `ASRParams` still contains historical flat fields for migration/runtime cleanup follow-up, but profile creation, profile response, and orchestrator execution no longer depend on them.
- Frontend profile controls should source field definitions from provider model descriptors and write values into `options.pipeline[].options`.
