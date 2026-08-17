# PR-0029 Final Concurrency and Lifecycle Hardening

## Статус

Merged

---

## Цель

Финальная техническая стабилизация concurrency/lifecycle слоя.

---

## Области проверки

| Компонент | Проверки |
|---|---|
| Source | cancellation, blocked output, error, creation error |
| Bus | publish, publish during shutdown, close, repeated close, blocked publish |
| Bridge | cancellation, source error, multiple sources, forwarder, WaitGroup, shutdown ordering |
| Hub | concurrent Register/Unregister, Broadcast, Shutdown, Broadcast during Shutdown |
| WebSocket Client | concurrent Send, Send+Close, repeated Close, blocked Send, Send after Close |
| Handler | connection, disconnect, shutdown, concurrent shutdown |

---

## Результаты стресс-тестов (`-race -count=100`)

| Пакет | Статус |
|---|---|
| `internal/bus` | PASS |
| `internal/source` | PASS |
| `internal/bridge` | PASS |
| `internal/hub` | PASS |
| `internal/websocket` | PASS |

Убран последний `time.Sleep` (заменён на channel-based ready signal).

---

## Последующее исправление Bus

Аудит выявил, что закрытие канала сообщений одновременно с публикацией могло привести к `panic: send on closed channel`, а повторный `Close()` был небезопасен. Bus переведен на отдельный idempotent shutdown-сигнал:

- `Close()` закрывает `Done()` ровно один раз;
- `Publish()` разблокируется при shutdown;
- `PublishCtx()` возвращает `ErrClosed` после shutdown;
- канал сообщений не закрывается напрямую, что устраняет race между send и close.

Потребители должны завершать чтение по `Bus.Done()`, а не ожидать закрытия канала сообщений.
