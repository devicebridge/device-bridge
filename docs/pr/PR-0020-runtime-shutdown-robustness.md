# PR-0020 Runtime Shutdown Robustness

## Статус

Merged

---

## Цель

Укрепить lifecycle Runtime: устранить потенциальные зависания при завершении приложения из-за блокирующей публикации в Bus и заблокированного downstream, а также корректно обрабатывать ошибку запуска HTTP Server.

---

## Исходная проблема

### Блокирующая публикация

`Bus.Publish(msg)` выполняет безусловную блокирующую отправку:

```go
b.ch <- msg
```

Если downstream (Hub → WebSocket client) заблокирован, Bus может заполниться, forwarder Source → Bus зависает навсегда, `WaitGroup` никогда не завершается, `Bridge.Run()` не возвращается, shutdown зависает.

### Hub forwarder блокировка

Hub forwarder (`for msg := range bus.Subscribe { hub.Broadcast(msg) }`) блокируется на `Broadcast`, если клиент не читает сообщения. После `Bus.Close()` горутина не может завершиться, блокируя возврат `Bridge.Run()`.

### HTTP Server error

Ошибка запуска HTTP Server (занятый порт) только логировалась, приложение продолжало работать.

---

## Новая модель shutdown

```
context cancellation
        ↓
runtime cancellation
        ↓
forwarder: PublishCtx detects ctx.Done() → returns
source: select detects ctx.Done() → returns
        ↓
WaitGroup complete
        ↓
Bus.Close()
        ↓
Bridge.Run() returns (hub forwarder continues asynchronously)
        ↓
Hub.Shutdown() → closes clients → unblocks hub forwarder
        ↓
HTTP Server.Shutdown()
        ↓
application exits
```

---

## Изменения

### Bus
- Добавлен `PublishCtx(ctx, msg) error` — контекстно-зависимая публикация через `select` с `ctx.Done()`. Блокирующий `Publish` сохранён для обратной совместимости.

### Bridge Runtime
- Forwarder Source → Bus использует `PublishCtx` вместо `Publish`. При отмене контекста forwarder немедленно завершается.
- `Bridge.Run()` больше не ждёт завершения hub forwarder goroutine — после `Bus.Close()` и возврата Run, goroutine завершится асинхронно при `Hub.Shutdown()`.

### main.go
- HTTP-ошибка запуска (`ListenAndServe`) больше не теряется. При ошибке вызывается `stop()` (отмена signal context), Bridge завершается, приложение выходит с ошибкой.

---

## Тестирование

| Тест | Проверка |
|---|---|
| `TestPublishCtx` | Успешная публикация с контекстом |
| `TestPublishCtxCancelled` | Отменённый контекст → ошибка |
| `TestBlockedDownstreamDoesNotHangShutdown` | Заблокированный клиент не препятствует завершению Run() |
| Обновлён `TestMultipleSources` | Polling под мьютексом для ожидания всех сообщений |

---

## Что намеренно НЕ входит

- Не добавляются новые внешние зависимости.
- Hub forwarder не имеет контекстной отмены для Broadcast.
- Архитектура Source → Bus → Hub не изменена.

---

## Результат

Runtime имеет bounded shutdown path: cancellation немедленно прерывает блокирующую публикацию, forwarder завершается, Sources останавливаются, Bus закрывается, `Bridge.Run()` возвращается. Заблокированный downstream не препятствует завершению Runtime.

---

## Fix: Deadlock Broadcast ↔ Shutdown

После PR-0020 обнаружен потенциальный deadlock между `Hub.Broadcast()` и `Hub.Shutdown()`.

### Проблема

`Broadcast()` удерживал `RLock()` во время `client.Send()`. Если `Send()` блокируется, `Shutdown()` не может получить `Lock()` и закрыть клиента.

### Исправление

`Broadcast()` больше не удерживает mutex во время `client.Send()`:

```text
RLock → snapshot clients → RUnlock → Send() без блокировки
```

`Shutdown()` может свободно получить `Lock()` и закрыть клиентов через `client.Close()`.

### Тесты

- `TestBlockedDownstreamDoesNotHangShutdown` расширен: проверяет, что `Close()` был вызван и `Send()` был разблокирован после `Hub.Shutdown()`.
- `TestMultipleSources` переписан на канальную синхронизацию вместо polling с `time.Sleep`.
- Добавлен `chanClient` для детерминированного ожидания сообщений.
