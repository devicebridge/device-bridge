# PR-0022 Runtime Forwarder Lifecycle

## Статус

Merged

---

## Цель

Привести lifecycle горутины `Bus → Hub` к формально завершённой модели. `Bridge.Run()` должен дождаться завершения forwarder перед возвратом.

---

## Проблема

После PR-0020 forwarder `Bus → Hub` завершался асинхронно — `Bridge.Run()` не ждал его. Горутина могла остаться после завершения Runtime.

---

## Решение

- `Bridge.Run()` ожидает `hubDone` после `Bus.Close()`.
- Forwarder завершается естественно: `range` на закрытом канале Bus выходит.
- В случае блокированного клиента `Hub.Shutdown()` (через `Client.Close()`) разблокирует forwarder.

---

## Тесты

| Тест | Проверка |
|---|---|
| `TestForwarderCompletesBeforeRunReturns` | Все сообщения доставлены до возврата Run |
| `TestBlockedDownstreamDoesNotHangShutdown` (обновлён) | Hub.Shutdown вызывает Close до ожидания Run |

---

## Результат

`Bridge.Run()` возвращается только после завершения всех внутренних горутин.
