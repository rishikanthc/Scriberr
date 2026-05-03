# Scriberr Backend Rules

1. `internal/api` is an HTTP adapter only: authenticate, validate, call one service method, map the response.

2. Production API code must not import `internal/database`, call `database.DB`, construct repositories, or run GORM queries.

3. Long-running work never runs inside handlers; transcription, import extraction, summaries, chat generation, recording finalization, and future automation go through durable services or workers.

4. State transitions have one owner. Queue and transcription status changes must use repository/service methods like enqueue, claim, renew, progress, complete, fail, cancel, and recover.

5. Repositories own persistence shape. Services ask for domain operations; GORM structs, raw rows, SQL details, and schema compatibility stay behind `internal/repository` and `internal/models`.

6. ASR, LLMs, media extraction, storage, auth tokens, and webhooks stay behind narrow interfaces so tests can fake them and adapters can change.

7. File paths are internal. Handlers and public responses must not expose or construct local paths; use file, recording, media import, or transcript storage services.

8. Every user-owned operation is scoped by `user_id`, including files, transcriptions, profiles, summaries, chat, recordings, API keys, queue stats, and automation.

9. Cross-user operations are admin-only service use cases. Do not let normal product services accept target user IDs from request bodies; use the authenticated principal, and use separate admin commands for user management and global settings.

10. Authorization is explicit. Admin routes must require an authenticated user with an admin role, and background/system paths must not be reused as public authorization shortcuts.

11. Events are small notifications, not source of truth. Persist durable state first, publish after, filter subscriptions by authorized audience, and make clients able to recover by re-fetching REST resources.

12. Database design must use relational constraints for core state. Prefer typed columns, foreign keys, composite unique indexes, partial unique indexes for per-user defaults, and query-driven composite indexes over unindexed JSON blobs or ad hoc string conventions.

13. New durable tables with ownership must include `user_id NOT NULL`, an index beginning with `user_id` for user-facing access paths, and foreign-key behavior that matches the lifecycle. New code must not rely on `user_id = 1` defaults except in explicit legacy migration paths.

14. Secrets are never stored raw. Passwords use password hashing, refresh tokens and API keys use one-way hashes, provider credentials are encrypted or otherwise protected before persistence, and public DTOs/logs never expose secret values.

15. The transcription queue is shared infrastructure. Scheduler policy is configured by admin-only settings, implemented behind a scheduler boundary, and must preserve user isolation in stats, events, cancellation, logs, and result reads.

16. Queue state transitions have one owner. Enqueue, claim, renew, progress, complete, fail, cancel, recover, and scheduler-policy selection must remain repository/service operations, not handler logic.

17. Configuration is loaded once, validated at startup, and injected. Runtime code must not read environment variables directly or silently create missing dependencies.

18. `internal/app` is the only backend composition root. `cmd/server` owns process concerns only, and non-bootstrap packages must not import `internal/api` or `internal/database`.
