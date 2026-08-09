# PR-0018 Graceful HTTP/WebSocket Shutdown

## Статус

Merged

---

## Цель

Обеспечить корректное завершение HTTP Server и активных WebSocket-соединений при остановке Bridge Runtime.

После PR-0017 Runtime имел управляемый lifecycle через `context.Context`, но HTTP Server и WebSocket connections не были его частью.

---

## Исходная проблема

- HTTP Server использовал `http.ListenAndServe(...)` без управляемого lifecycle.
- Активные WebSocket connections не закрывались при остановке Runtime.
- Завершение Runtime и завершение HTTP/WebSocket существовали независимо.

---

## Новая модель lifecycle

```
Bridge.Run() returns
        │
        ▼
Hub.Shutdown()
        │
        ├── all WebSocket clients receive Close frame
        │
        ▼
HTTP Server.Shutdown()
        │
        ▼
all connections closed
        │
        ▼
application exits
```

---

## Выполненные изменения

### Hub
- Добавлен канал `closed` — сигнал завершения Hub.
- `Register()` не добавляет клиентов после `Shutdown()`.
- `Broadcast()` возвращает `nil` после `Shutdown()`.
- `Shutdown()` — идемпотентная остановка: закрывает канал, обходит всех клиентов и вызывает `client.Close()`.
- `Done() <-chan struct{}` — канал для наблюдения за состоянием Hub.

### Client интерфейс
- Добавлен метод `Close()`.

### WebSocket Client
- `Close()`: отправляет WebSocket close frame (`CloseGoingAway`), закрывает соединение.

### WebSocket Handler
- Проверка `hub.Done()` после регистрации клиента — если Hub завершается, соединение закрывается без входа в read-луп.

### HTTP Server
- `main.go` использует `*http.Server` с `ListenAndServe()` и `Shutdown()`.
- `http.ErrServerClosed` не считается ошибкой.
- Shutdown timeout: 5 секунд.

### main.go
- Создаёт `*http.Server`.
- Запускает Bridge Runtime и HTTP Server.
- Ожидает завершения Bridge.
- Вызывает `Hub.Shutdown()` + `HTTP Server.Shutdown()`.
- Корректно обрабатывает ошибки.

---

## Порядок shutdown

```text
Bridge.Run() returns
        │
        ├── Hub.Shutdown()
        │     ├── close(closed)
        │     ├── for each client: client.Close()
        │     │   └── write WebSocket Close frame
        │     │   └── close TCP connection
        │     └── handler goroutine:
        │           ReadMessage → error → break → defer Unregister
        │
        ├── HTTP Server.Shutdown(shutdownCtx)
        │     ├── stop listener (no new connections)
        │     └── wait for active connections to close
        │
        ▼
application exits
```

---

## Тестирование

| Тест | Проверка |
|---|---|
| `TestShutdown` (hub) | Hub.Shutdown закрывает клиентов, Done канал сигналит |
| `TestShutdownIdempotent` (hub) | Повторный Shutdown не паникует |
| `TestRegisterAfterShutdown` (hub) | Register не добавляет клиентов после shutdown |
| `TestBroadcastAfterShutdown` (hub) | Broadcast возвращает nil после shutdown |
| `TestWebSocketShutdown` | WebSocket соединение закрывается при shutdown Hub |
| `TestWebSocketShutdownMultipleClients` | Множественные клиенты закрываются при shutdown |
| `TestShutdownIsIdempotent` (ws) | Повторный shutdown Hub не паникует с WebSocket |
| `TestBroadcastDuringShutdown` | Гонка Broadcast + Shutdown не вызывает panic/race |

---

## Совместимость

- Публичный API не изменён.
- Дополнительные внешние зависимости не добавлены.
- Интерфейс `hub.Client` расширен методом `Close()` — все реализации обновлены.

---

## Что намеренно НЕ входит

- OS signal handling (`signal.Notify`).
- Graceful shutdown HTTP через OS signals.
- Health checks / metrics.
- Динамическое управление Sources.
- Изменение WebSocket protocol / Message format.

---

## Результат

Device Bridge получил единый управляемый lifecycle включая HTTP Server и WebSocket connections. При завершении Bridge Runtime все активные WebSocket-соединения корректно закрываются, HTTP Server выполняет graceful shutdown.
