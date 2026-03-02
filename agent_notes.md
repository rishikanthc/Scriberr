# Speaker Segment Persistence & Error Handling Fixes

## Changes Summary

1.  **Backend Models**: Added `SpeakerSegment` model to `internal/models/transcription.go` to persist timestamped audio segments for each identified speaker. Added to GORM auto-migration.
2.  **Database Layer**:
    - Updated `JobRepository` interface in `internal/repository/implementations.go` with `SaveSpeakerSegments` and `GetSegmentsBySpeakerID`.
    - Implemented these methods in `jobRepository`.
3.  **Transcription Pipeline**:
    - Updated `UnifiedTranscriptionService.saveTranscriptionResults` in `internal/transcription/unified_service.go` to automatically extract and save speaker segments after successful transcription.
4.  **API Layer**:
    - Added `GET /api/v1/speakers/:id/segments` endpoint in `internal/api/speaker_handlers.go`.
    - Registered the new route in `internal/api/router.go`.
5.  **Speaker Management Fixes**:
    - Corrected a Go-style syntax error (`func` instead of `def`) in `internal/transcription/adapters/py/nvidia/titanet_manage.py`.
    - Enhanced `TitanetAdapter` to capture and return `stderr` from Python commands for better diagnostics.
    - Updated API handlers to return these descriptive error messages to the frontend.
6.  **Frontend Enhancements**:
    - Updated `web/frontend/src/lib/speakersApi.ts` to include the `getSegments` method and improved error parsing from API responses.
    - Updated `AudioFilesTable.tsx` to display speaker names in the table view.
7.  **Tests**: Updated `MockJobRepository` in test suites to match the new interface; all `internal/transcription` tests passed.

## Environment Resolution
- The `uv run` issue in `data/whisperx-env/parakeet/` was resolved by running `uv lock` (performed by user), fixing dependency resolution for the private registry.
- Syntax error in `titanet_manage.py` was manually patched in both the source and the active environment.


* * *
# Speaker Persistence Implementation

Implemented global speaker identity tracking with high-dimensional embedding storage.

## Backend Changes
- Added `SpeakerSegment` and `SpeakerJobCentroid` models in SQLite.
- Updated `UnifiedTranscriptionService` to save reference segments and job-level centroids.
- Enhanced `titanet_identify.py` to extract and return segment-level embeddings and the calculated centroid.
- Added `SaveSpeakerJobCentroids` to `JobRepository`.
- Updated API routes and handlers for speaker management (Rename, List, Delete).
- Fixed build errors in `unified_service.go` related to variable scope and function signatures.

## Frontend Changes
- Created a "Speakers" tab in the Settings page.
- Implemented an `AudioChip` component that plays speaker voice samples using browser-side seeking.
- Added global speaker renaming and deletion capabilities.
- Optimized API calls to handle large transcript payloads by removing redundant preloads in the segments endpoint.

## Format & Consistency
- Standardized speaker IDs in the database (supporting multiple prefix formats like `Speaker-` and `Spk-`).
- Implemented trailing slash consistency for Gin routing.


* * *
