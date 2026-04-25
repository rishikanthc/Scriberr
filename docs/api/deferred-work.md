# Deferred API Work

This API pass intentionally leaves engine/backend work out of scope.

Deferred items:

- Actual transcription execution and queue integration for created transcription resources.
- Durable transcription logs backend for `GET /api/v1/transcriptions/{id}/logs`.
- Durable transcription executions backend for `GET /api/v1/transcriptions/{id}/executions`.
- Durable idempotency persistence across process restarts or multiple server processes.
- SSE event replay for `Last-Event-ID`.

Current placeholders keep authenticated route shapes and structured error responses so clients can distinguish unsupported backend work from missing endpoints.
