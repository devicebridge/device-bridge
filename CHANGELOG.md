# Changelog

## Lifecycle — завершён (PR-0017 … PR-0029)

### PR-0017 Runtime Lifecycle
`Bridge.Run(ctx context.Context) error` — управляемый lifecycle через context. Source ошибки сохраняются, отменяют runtime.

### PR-0018 Graceful HTTP/WebSocket Shutdown
Hub.Shutdown(), Client.Close(), HTTP Server.Shutdown() — корректное завершение WebSocket соединений.

### PR-0019 Application Lifecycle
`signal.NotifyContext(SIGINT, SIGTERM)` — OS signals управляют application lifecycle.

### PR-0020 Runtime Shutdown Robustness
`Bus.PublishCtx(ctx, msg)` — прерывание блокирующей публикации при cancellation. HTTP startup error вызывает shutdown.

### PR-0021 WebSocket Client Shutdown
`Client.Close()` использует `TryLock` — не ждёт блокированный `Send()`. `conn.Close()` разблокирует запись.

### PR-0022 Forwarder Lifecycle
`<-hubDone` — Bridge.Run ждёт завершения Bus→Hub forwarder перед возвратом.

### PR-0023 Source Creation Errors
Ошибки `Registry.Create` сохраняются как runtime error, отменяют остальные Sources.

### PR-0024 Lifecycle State
Многократный cancel, конкурентный cancel+source error — без паник, без гонок.

### PR-0025 WebSocket Handler Lifecycle
Подключение после shutdown, handler goroutine exits — детерминированное поведение.

### PR-0026 HTTP Server Graceful Shutdown
Интеграция HTTP shutdown в общий lifecycle.

### PR-0027 Application Lifecycle Integration
Полный интеграционный тест: scanner → bus → hub → websocket → client → shutdown.

### PR-0028 Error Observability
Различимые сообщения об ошибках для config, http, runtime, shutdown.

### PR-0029 Concurrency Hardening
Стресс-тесты 100× с race detector для всех пакетов. Убран последний `time.Sleep`.
