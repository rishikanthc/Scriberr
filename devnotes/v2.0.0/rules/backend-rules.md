# Scriberr Backend Rules

1. `internal/api` is an HTTP adapter only: authenticate, validate, call one service method, map the response.

2. Production API code must not import `internal/database`, call `database.DB`, construct repositories, or run GORM queries.

3. Long-running work never runs inside handlers; transcription, import extraction, summaries, chat generation, recording finalization, and future automation go through durable services or workers.

4. State transitions have one owner. Queue and transcription status changes must use repository/service methods like enqueue, claim, renew, progress, complete, fail, cancel, and recover.

5. Repositories own persistence shape. Services ask for domain operations; GORM structs, raw rows, SQL details, and schema compatibility stay behind `internal/repository` and `internal/models`.

6. ASR, LLMs, media extraction, storage, auth tokens, and webhooks stay behind narrow interfaces so tests can fake them and adapters can change.

7. File paths are internal. Handlers and public responses must not expose or construct local paths; use file, recording, media import, or transcript storage services.

8. Every user-owned operation is scoped by `user_id`, including files, transcriptions, profiles, summaries, chat, recordings, API keys, queue stats, and automation.

9. Events are small notifications, not source of truth. Persist durable state first, publish after, and make clients able to recover by re-fetching REST resources.

10. Configuration is loaded once, validated at startup, and injected. Runtime code must not read environment variables directly or silently create missing dependencies.

11. `internal/app` is the only backend composition root. `cmd/server` owns process concerns only, and non-bootstrap packages must not import `internal/api` or `internal/database`.
