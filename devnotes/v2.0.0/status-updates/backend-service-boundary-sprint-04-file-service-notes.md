# Backend Service Boundary Sprint 04 File Service Notes

Sprint 4 is in progress. This checkpoint moves file route persistence and durable upload storage layout behind an injected file service.

## Completed

- Added `internal/files.Service` for upload, list, get, update, delete, and audio lookup workflows.
- Moved direct upload storage path construction and file writes out of `internal/api`.
- Moved direct upload record creation into the file service through repository methods.
- Moved video extraction completion/failure persistence into the file service through repository methods.
- Injected the YouTube media import service from the composition root instead of constructing a repository in the handler.
- Removed `file_handlers.go` from the production API database import inventory.

## Still Pending

- Introduce one shared file-ready handoff for direct upload, video extraction, YouTube import completion, and recording finalization.
- Use that handoff as the future entry point for post-file automation.

## Verification

- `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestRecording|TestFile|TestProductionAPIDatabaseAccessInventory'`
- `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/files ./internal/mediaimport ./internal/recording ./internal/repository ./cmd/server`
