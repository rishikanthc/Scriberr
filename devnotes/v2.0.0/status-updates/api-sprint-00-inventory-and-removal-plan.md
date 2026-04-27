# Sprint 0 API Inventory and Removal Plan

## Decision Summary

The current API surface is a legacy, mixed-responsibility Gin API. Sprint 1 should replace it with a clean `/api/v1` foundation from `devnotes/new-api-spec.md`, remove old route registrations and generated old docs, and rebuild only the first-pass canonical API modules.

Do not preserve old route names as the primary design. Add temporary legacy aliases only if the frontend or CLI must keep working during the revamp, and route those aliases through the new canonical services.

## API-Owned Files

Current API implementation files:

- `internal/api/router.go`
- `internal/api/handlers.go`
- `internal/api/types.go`
- `internal/api/constants.go`
- `internal/api/cli_auth_handlers.go`
- `internal/api/cli_install_handlers.go`
- `internal/api/log_handler.go`
- `internal/api/openai_handler.go`
- `internal/api/chat_handlers.go`
- `internal/api/notes_handlers.go`
- `internal/api/speaker_mapping_handlers.go`
- `internal/api/summarize_handlers.go`
- `internal/api/summary_handlers.go`

Current API-adjacent middleware/logging files:

- `pkg/middleware/auth.go`
- `pkg/middleware/compression.go`
- `pkg/logger/logger.go`

Generated or static old API docs:

- `api-docs/API_DOCS.md`
- `api-docs/docs.go`
- `api-docs/swagger.json`
- `api-docs/swagger.yaml`

## Compile-Sensitive Dependencies

`cmd/server/main.go` currently constructs `api.NewHandler(...)` with many concrete dependencies, then calls `api.SetupRoutes(handler, authService)`.

Sprint 1 must either:

- keep those function names while replacing their internals, or
- update `cmd/server/main.go` narrowly enough to compile against the new API constructor.

Known current constructor dependencies:

- config
- auth service
- user service
- file service
- job repository
- API key repository
- profile repository
- user repository
- LLM config repository
- summary repository
- chat repository
- note repository
- speaker mapping repository
- refresh token repository
- task queue
- unified transcription processor
- quick transcription service
- multi-track processor
- SSE broadcaster

The clean API should reduce this surface over time. For Sprint 1, preserve only dependencies needed for health, auth hooks, structured errors, and placeholder route registration.

Static web serving is attached from `internal/web/static.go` inside the current router. Do not modify web/frontend code during the API revamp.

CLI coupling:

- `internal/cli/client.go` uploads to `/api/v1/transcription/upload`.
- The new API uses `POST /api/v1/files` or `POST /api/v1/transcriptions:submit`.
- Updating CLI internals is outside API scope unless the user explicitly approves it. If CLI support must continue during API revamp, add a temporary legacy alias in API code and mark it for removal.

## Route Removal Matrix

| Current route | Handler | Disposition |
| --- | --- | --- |
| `GET /health` | `HealthCheck` | Keep, replace response with new spec shape if needed. |
| `GET /install.sh` | `GetInstallScript` | Delete from canonical API. Consider temporary compatibility only if install flow is in scope. |
| `GET /install-cli.sh` | `GetInstallScript` | Delete from canonical API. Consider temporary compatibility only if install flow is in scope. |
| `GET /api/v1/auth/registration-status` | `GetRegistrationStatus` | Replace with canonical contract. |
| `POST /api/v1/auth/register` | `Register` | Replace with canonical contract. |
| `POST /api/v1/auth/login` | `Login` | Replace with canonical contract. |
| `POST /api/v1/auth/refresh` | `Refresh` | Replace with canonical contract. |
| `POST /api/v1/auth/logout` | `Logout` | Replace with canonical contract. |
| `POST /api/v1/auth/change-password` | `ChangePassword` | Replace with canonical contract. |
| `POST /api/v1/auth/change-username` | `ChangeUsername` | Replace with canonical contract. |
| `GET /api/v1/auth/cli/authorize` | `AuthorizeCLI` | Delete or defer. CLI auth is not in Sprint 1 spec. |
| `POST /api/v1/auth/cli/authorize` | `ConfirmCLIAuthorization` | Delete or defer. CLI auth is not in Sprint 1 spec. |
| `GET /api/v1/cli/download` | `DownloadCLIBinary` | Delete or defer. Not in new API spec. |
| `GET /api/v1/cli/install` | `GetInstallScript` | Delete or defer. Not in new API spec. |
| `GET /api/v1/api-keys/` | `ListAPIKeys` | Replace with canonical `GET /api/v1/api-keys`. |
| `POST /api/v1/api-keys/` | `CreateAPIKey` | Replace with canonical `POST /api/v1/api-keys`. |
| `DELETE /api/v1/api-keys/:id` | `DeleteAPIKey` | Replace with canonical `DELETE /api/v1/api-keys/{id}`. |
| `POST /api/v1/transcription/upload` | `UploadAudio` | Replace with `POST /api/v1/files`; optional temporary alias for CLI/frontend only. |
| `POST /api/v1/transcription/upload-video` | `UploadVideo` | Delete. Video extraction is deferred; canonical file upload may accept video later. |
| `POST /api/v1/transcription/upload-multitrack` | `UploadMultiTrack` | Delete. Multi-track is out of scope. |
| `GET /api/v1/transcription/:id/audio` | `GetAudioFile` | Replace with `GET /api/v1/files/{id}/audio` and `GET /api/v1/transcriptions/{id}/audio`. |
| `POST /api/v1/transcription/youtube` | `DownloadFromYouTube` | Replace with `POST /api/v1/files:import-youtube` placeholder. |
| `POST /api/v1/transcription/submit` | `SubmitJob` | Replace with `POST /api/v1/transcriptions` and `POST /api/v1/transcriptions:submit`. |
| `POST /api/v1/transcription/:id/start` | `StartTranscription` | Delete. Creation should enqueue work. |
| `POST /api/v1/transcription/:id/kill` | `KillJob` | Replace with `POST /api/v1/transcriptions/{id}:cancel`. |
| `GET /api/v1/transcription/:id/logs` | `GetJobLogs` | Replace with `GET /api/v1/transcriptions/{id}/logs`. Backend may be placeholder. |
| `GET /api/v1/transcription/:id/status` | `GetJobStatus` | Replace with `GET /api/v1/transcriptions/{id}`. |
| `GET /api/v1/transcription/:id/transcript` | `GetTranscript` | Replace with `GET /api/v1/transcriptions/{id}/transcript`. |
| `GET /api/v1/transcription/:id/execution` | `GetJobExecutionData` | Replace with `GET /api/v1/transcriptions/{id}/executions`. |
| `GET /api/v1/transcription/:id/merge-status` | `GetMergeStatus` | Delete. Multi-track/merge is out of scope. |
| `GET /api/v1/transcription/:id/track-progress` | `GetTrackProgress` | Delete. Multi-track is out of scope. |
| `PUT /api/v1/transcription/:id/title` | `UpdateTranscriptionTitle` | Replace with `PATCH /api/v1/transcriptions/{id}`. |
| `GET /api/v1/transcription/:id/summary` | `GetSummaryForTranscription` | Delete/defer. Summaries are deferred module. |
| `GET /api/v1/transcription/:id` | `GetTranscriptionJob` | Replace with `GET /api/v1/transcriptions/{id}`. |
| `DELETE /api/v1/transcription/:id` | `DeleteTranscriptionJob` | Replace with `DELETE /api/v1/transcriptions/{id}`. |
| `GET /api/v1/transcription/list` | `ListTranscriptionJobs` | Replace with `GET /api/v1/transcriptions`. |
| `GET /api/v1/transcription/models` | `GetSupportedModels` | Replace with `GET /api/v1/models/transcription`. |
| `GET /api/v1/transcription/:id/notes` | `ListNotes` | Delete/defer. Notes are deferred module. |
| `POST /api/v1/transcription/:id/notes` | `CreateNote` | Delete/defer. Notes are deferred module. |
| `GET /api/v1/transcription/:id/speakers` | `GetSpeakerMappings` | Delete/defer. Speaker editing is deferred module. |
| `POST /api/v1/transcription/:id/speakers` | `UpdateSpeakerMappings` | Delete/defer. Speaker editing is deferred module. |
| `POST /api/v1/transcription/quick` | `SubmitQuickTranscription` | Delete/defer. Not in first clean API pass. |
| `GET /api/v1/transcription/quick/:id` | `GetQuickTranscriptionStatus` | Delete/defer. Not in first clean API pass. |
| `GET /api/v1/profiles/` | `ListProfiles` | Replace with canonical `GET /api/v1/profiles`. |
| `POST /api/v1/profiles/` | `CreateProfile` | Replace with canonical `POST /api/v1/profiles`. |
| `GET /api/v1/profiles/:id` | `GetProfile` | Replace with canonical `GET /api/v1/profiles/{id}`. |
| `PUT /api/v1/profiles/:id` | `UpdateProfile` | Replace with `PATCH /api/v1/profiles/{id}`. |
| `DELETE /api/v1/profiles/:id` | `DeleteProfile` | Replace with canonical `DELETE /api/v1/profiles/{id}`. |
| `POST /api/v1/profiles/:id/set-default` | `SetDefaultProfile` | Replace with `POST /api/v1/profiles/{id}:set-default`. |
| `GET /api/v1/user/default-profile` | `GetUserDefaultProfile` | Replace with settings/profile default contract. |
| `POST /api/v1/user/default-profile` | `SetUserDefaultProfile` | Replace with settings/profile default contract. |
| `GET /api/v1/user/settings` | `GetUserSettings` | Replace with `GET /api/v1/settings`. |
| `PUT /api/v1/user/settings` | `UpdateUserSettings` | Replace with `PATCH /api/v1/settings`. |
| `GET /api/v1/admin/queue/stats` | `GetQueueStats` | Replace with `GET /api/v1/admin/queue`. |
| `GET /api/v1/llm/config` | `GetLLMConfig` | Delete/defer. LLM config is not in first clean API pass. |
| `POST /api/v1/llm/config` | `SaveLLMConfig` | Delete/defer. LLM config is not in first clean API pass. |
| `GET /api/v1/summaries/` | `ListSummaryTemplates` | Delete/defer. Summaries are deferred module. |
| `POST /api/v1/summaries/` | `CreateSummaryTemplate` | Delete/defer. Summaries are deferred module. |
| `GET /api/v1/summaries/:id` | `GetSummaryTemplate` | Delete/defer. Summaries are deferred module. |
| `PUT /api/v1/summaries/:id` | `UpdateSummaryTemplate` | Delete/defer. Summaries are deferred module. |
| `DELETE /api/v1/summaries/:id` | `DeleteSummaryTemplate` | Delete/defer. Summaries are deferred module. |
| `GET /api/v1/summaries/settings` | `GetSummarySettings` | Delete/defer. Summaries are deferred module. |
| `POST /api/v1/summaries/settings` | `SaveSummarySettings` | Delete/defer. Summaries are deferred module. |
| `GET /api/v1/chat/models` | `GetChatModels` | Delete/defer. Chat is deferred module. |
| `POST /api/v1/chat/sessions` | `CreateChatSession` | Delete/defer. Chat is deferred module. |
| `GET /api/v1/chat/transcriptions/:transcription_id/sessions` | `GetChatSessions` | Delete/defer. Chat is deferred module. |
| `GET /api/v1/chat/sessions/:session_id` | `GetChatSession` | Delete/defer. Chat is deferred module. |
| `POST /api/v1/chat/sessions/:session_id/messages` | `SendChatMessage` | Delete/defer. Chat is deferred module. |
| `PUT /api/v1/chat/sessions/:session_id/title` | `UpdateChatSessionTitle` | Delete/defer. Chat is deferred module. |
| `POST /api/v1/chat/sessions/:session_id/title/auto` | `AutoGenerateChatTitle` | Delete/defer. Chat is deferred module. |
| `DELETE /api/v1/chat/sessions/:session_id` | `DeleteChatSession` | Delete/defer. Chat is deferred module. |
| `GET /api/v1/notes/:note_id` | `GetNote` | Delete/defer. Notes are deferred module. |
| `PUT /api/v1/notes/:note_id` | `UpdateNote` | Delete/defer. Notes are deferred module. |
| `DELETE /api/v1/notes/:note_id` | `DeleteNote` | Delete/defer. Notes are deferred module. |
| `POST /api/v1/summarize/` | `Summarize` | Delete/defer. Summaries are deferred module. |
| `POST /api/v1/config/openai/validate` | `ValidateOpenAIKey` | Delete/defer. Provider config is not in first clean API pass. |
| `GET /api/v1/events/` | `Events` | Replace with `GET /api/v1/events`; backend may be placeholder. |

## Test Disposition

Remove or replace these old API tests during Sprint 1:

- `tests/api_handlers_test.go`
- `tests/api_chat_test.go`
- `tests/api_summary_test.go`
- `tests/api_user_test.go`
- `tests/cli_handlers_test.go`
- old API route coverage inside `tests/security_test.go`

Keep non-API test files unless implementation changes require a targeted update:

- `tests/auth_service_test.go`
- `tests/database_test.go`
- `tests/queue_test.go`
- `tests/llm_test.go`
- `tests/adapter_registration_test.go`
- `tests/test_helpers.go` only if still needed by non-API tests

New API tests should prefer `internal/api/**/*_test.go` so they can test the new HTTP layer directly without carrying the old broad integration suite.

## Minimal API Interfaces for Sprint 1

Sprint 1 only needs enough internal shape to compile and test the API foundation:

- `AuthVerifier`: validate JWT bearer tokens and resolve a single-user auth context.
- `APIKeyVerifier`: validate hashed API keys and resolve auth context.
- `ReadinessChecker`: check database readiness for `/api/v1/ready`.
- `Logger`: Zap-backed structured logger with configurable level.
- Placeholder route services for files, transcriptions, profiles, settings, events, models, and admin queue.

Concrete resource services can be introduced in later sprints:

- `FileService`
- `TranscriptionService`
- `ProfileService`
- `SettingsService`
- `EventService`
- `ModelCapabilityService`
- `AdminQueueService`

## Sprint 1 First Actions

1. Write failing tests for request ID propagation, health/readiness, structured errors, panic recovery, auth guard behavior, and placeholder `501` route shape.
2. Replace route registration with the new canonical `/api/v1` groups.
3. Remove legacy handler files and generated docs that only describe deleted routes.
4. Add Zap-backed API logging and log-level config.
5. Run a narrow API test suite, then a compile check.
