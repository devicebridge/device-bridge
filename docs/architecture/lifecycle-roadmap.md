# Lifecycle Completion Roadmap

## Завершенные этапы

- Bus shutdown and concurrent publish safety.
- Single-owner Bridge runtime lifecycle.
- Source forwarder synchronization before Bus shutdown.
- Thread-safe source Registry.
- Hub client cleanup and idempotent client close.
- Broadcast continuation after partial client failures.
- Bounded WebSocket outbound queues with isolated writer goroutines.
- Testable application orchestration for bind errors and cancellation.
- Logging of Bridge delivery failures.

## Source integration in progress

- Application configuration now accepts `DEVICE_BRIDGE_SOURCES`.
- The application bootstrap registers the supported scanner source names.
- The channel input adapter is available for tests and future transport adapters.
- Application-level adapter → WebSocket E2E coverage is available.
- Basic `/healthz` and `/readyz` endpoints are available.
- `/readyz` reflects runtime readiness instead of being a static liveness response.
- Delivery guarantees and bounded backpressure policy are documented.

## Serial integration in progress

- Cross-platform line-oriented `SerialAdapter` is implemented over `io.ReadCloser`.
- Unit tests cover framing, EOF, port errors, and cancellation.
- OS device opening, serial settings, PTY integration, and reconnect policy remain separate.
- `DEVICE_BRIDGE_SCANNER_PATH` can open a serial-compatible path; PTY/socat manual validation is documented.
- Serial baud/reconnect settings are validated and a cancellable reconnect adapter is covered by unit tests.
- Physical device input and reconnect policy remain a separate block.

## Release criteria

- `go test -race ./...` passes.
- `go vet ./...` passes.
- `go build ./...` passes.
- CI passes on Linux, macOS, and Windows.
- Slow clients cannot block Hub or Bridge shutdown.
- Source, forwarder, dispatcher, and WebSocket writer lifecycle is covered by deterministic tests.

## Remaining hardening

- Add explicit goroutine lifecycle probes if runtime observability is required.
- Add metrics backend only when operational requirements justify a dependency.
