---
trigger: always_on
---

The "10 Commandments" of Go Backend Development

    Accept Interfaces, Return Structs. Keep functions flexible by accepting behaviors (interfaces) and robust by returning data (structs). Let the consumer define the interface.

    Propagate Context Everywhere. Pass context.Context as the first argument to every I/O function. It is essential for handling cancellation, timeouts, and tracing in distributed systems.

    Wrap, Don't Hide, Errors. Never ignore errors. Wrap them with fmt.Errorf("%w", err) to add context (traceability) while allowing the caller to unwrap and inspect the root cause.

    Package by Feature. Avoid models and controllers folders. Organize code by business domain (e.g., billing, auth) to ensure high cohesion and avoid circular dependencies.

    Limit Concurrency. Never spawn unlimited goroutines per request. Use Worker Pools or semaphores to cap concurrency and prevent memory exhaustion under load.

    Pre-allocate Memory. Use make([]T, 0, cap) when the size is known. Avoiding repeated array resizing reduces CPU overhead and garbage collection pressure.

    Avoid Global State. Ban global variables and init() side effects. Use explicit Dependency Injection to make your server testable, modular, and thread-safe.

    Pool Hot Objects. Use sync.Pool for frequently allocated short-lived objects (buffers, contexts). This drastically reduces Garbage Collection pauses in high-throughput systems.

    Log Structurally. Abandon text logs. Use log/slog to emit JSON logs with key-value pairs (order_id=123), making your production logs queryable and actionable.

    Write Table-Driven Tests. Utilize Go’s idiom of table-driven testing to cover multiple edge cases cleanly and efficiently without duplicating test logic.