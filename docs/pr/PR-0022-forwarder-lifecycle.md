# PR-0022 Runtime Forwarder Lifecycle

## Статус

Merged (Revisited: PR Corrective — `<-hubDone` removed)

---

## Цель

Привести lifecycle горутины `Bus → Hub` к формально завершённой модели. `Bridge.Run()` должен дождаться завершения forwarder перед возвратом.

---

## Проблема

После PR-0020 forwarder `Bus → Hub` завершался асинхронно — `Bridge.Run()` не ждал его. Горутина могла остаться после завершения Runtime.

---

## Решение (оригинальное, PR-0022)

- `Bridge.Run()` ожидает `hubDone` после `Bus.Close()`.
- Forwarder завершается естественно: `range` на закрытом канале Bus выходит.
- В случае блокированного клиента `Hub.Shutdown()` (через `Client.Close()`) разблокирует forwarder.

---

## Статус (Revisited)

Модель PR-0022 создавала потенциальный deadlock: при заблокированном downstream-клиенте `Hub.Shutdown()` должен быть вызван ДО возврата `Bridge.Run()`, но внешний lifecycle вызывает `Hub.Shutdown()` ПОСЛЕ.

**Corrective PR** (после `3c8d2e3`) убирает `<-hubDone`. Новый контракт:

```text
Bus.Close()
    ↓
Bridge.Run() returns
    ↓
Hub.Shutdown() (внешний lifecycle)
    ↓
hub forwarder terminates
```

---

## Тесты

| Тест | Проверка |
|---|---|
| `TestForwarderCompletesBeforeRunReturns` | Все сообщения доставлены до возврата Run |
| `TestBlockedDownstreamDoesNotHangShutdown` | Run возвращается ДО Hub.Shutdown при блокированном downstream |
| `TestRunReturnsBeforeHubShutdown` | Новый regression test: подтверждает новый контракт |

---

## Результат

`Bridge.Run()` возвращается после закрытия Bus, не дожидаясь завершения hub forwarder. Downstream shutdown — ответственность внешнего lifecycle через `Hub.Shutdown()`.
