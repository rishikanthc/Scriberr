# ASR Provider Backend Sprint 12: Hardening Notes

Sprint 12 closes the provider-backend implementation pass with contract tests, a minimal example provider server, provider author documentation, and final architecture guards.

## Performance Review

- Model-card lookup remains registry-backed and deterministic. Current providers are queried on demand; add short TTL caching later only if diagnostics/model-list calls become hot under real remote providers.
- Provider status polling is bounded to admin diagnostics and remote job execution. Admin calls use timeouts; remote execution already polls at configured intervals.
- Normalized audio lookup remains per job and file-path based. The preprocessor boundary avoids repeated preprocessing inside provider adapters.
- Queue claiming remains centralized in Scriberr. Providers do not own scheduling state, and provider capacity is used for selection/diagnostics rather than replacing the durable queue.
- Provider output is copied into canonical transcript structures before persistence, keeping provider-specific data out of hot transcript read paths.

## Residual Follow-Up

- Real third-party provider containers should run the contract helper in their own CI.
- Live streaming, automatic Docker discovery, and gRPC/WebSocket transports remain deferred.
