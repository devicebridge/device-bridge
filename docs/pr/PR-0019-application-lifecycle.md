# PR-0019 Application Lifecycle / OS Signals

## Статус

Merged

---

## Цель

Связать жизненный цикл процесса приложения с OS signals и существующим lifecycle Bridge Runtime.

После PR-0019 приложение реагирует на `SIGINT`/`SIGTERM`, корректно завершает Bridge Runtime и различает штатное завершение от ошибки Source.

---

## Исходная проблема

После PR-0018 `Bridge.Run(ctx)` был управляемым через `context.Context`, но `main()` использовал:

```go
go b.Run(context.Background())
```

`context.Background()` невозможно отменить, OS signals не обрабатывались, ошибка Runtime не доходила до уровня приложения.

---

## Новая модель lifecycle

```
process start
     │
     ▼
signal.NotifyContext(SIGINT, SIGTERM)
     │
     ▼
Bridge.Run(ctx) + HTTP Server
     │
     │
     ▼
OS signal
     │
     ▼
context cancellation
     │
     ▼
Bridge: Sources → Bus → Hub
     │
     ▼
Hub.Shutdown()
     │
     ▼
HTTP Server.Shutdown()
     │
     ▼
main() exits
```

---

## Реализованные изменения

- `cmd/device-bridge/main.go`:
  - `signal.NotifyContext(os.Interrupt, syscall.SIGTERM)` для signal-aware context.
  - `Bridge.Run(ctx)` с этим context вместо `context.Background()`.
  - `bridgeDone` channel — ожидание результата Runtime.
  - Ошибка Bridge проверяется: `context.Canceled` при нормальном shutdown не считается ошибкой.
  - Реальная ошибка Source/Runtime выводится в лог и приводит к `os.Exit(1)`.

- Тесты:
  - `TestApplicationNormalShutdown` — отмена контекста → Bridge завершается → Hub/HTTP shutdown.
  - `TestApplicationErrorPropagation` — ошибка Source доходит до application level.
  - `TestContextCancellationIsNotError` — `context.Canceled` не передаётся как ошибка Runtime.

---

## Обработка ошибок

| Сценарий | Результат |
|---|---|
| `SIGINT`/`SIGTERM` | Bridge.Run → nil (штатное завершение) |
| `context.Canceled` | Не считается ошибкой |
| Source error | Лог ошибки → `os.Exit(1)` |

---

## Ограничения

- Полноценный graceful shutdown HTTP Server и WebSocket connections не является основной задачей этого PR (существующий механизм PR-0018 сохранён).
- Signal handling находится только на уровне `cmd/device-bridge`, не в `internal`.
- Auto-restart Sources не реализован.
- Health checks / metrics / structured logging не добавлены.

---

## Результат

Application lifecycle связан на всех уровнях:

```
OS signal (SIGINT/SIGTERM)
    ↓
Application context cancellation
    ↓
Bridge.Runtime(ctx)
    ├── Sources
    ├── Bus
    └── Hub
    ↓
Hub.Shutdown()
    ↓
HTTP Server.Shutdown()
    ↓
main() exits
```

`main()` больше не запускает Runtime без контроля и корректно обрабатывает ошибки Sources.
