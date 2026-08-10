# PR-0021 WebSocket Client Shutdown Robustness

## Статус

Merged

---

## Цель

Устранить потенциальный deadlock между `Client.Send()` и `Client.Close()` при блокирующей записи WebSocket.

---

## Проблема

`Close()` использовал `mu.Lock()` — тот же mutex, что и `Send()`. Если `Send()` блокировался на `conn.WriteMessage()`, `Close()` не мог получить mutex и закрыть соединение.

---

## Решение

- `Close()` использует `sync.Mutex.TryLock()` — если mutex свободен, отправляет Close frame.
- Если mutex занят — пропускает Close frame и сразу вызывает `conn.Close()`.
- `conn.Close()` разблокирует зависший `WriteMessage()` в `Send()`.
- `sync.Once` гарантирует идемпотентность.

---

## Тесты

| Тест | Проверка |
|---|---|
| `TestCloseDoesNotWaitForMutex` | Close не блокируется при удержании mutex |
| `TestCloseIdempotent` | Повторный Close без паники |
| `TestSendAfterClose` | Send после Close возвращает ошибку |
| `TestConcurrentSendAndClose` | Конкурентные Send + Close без паники и гонок |

---

## Результат

`Client.Close()` больше не ждёт завершения блокированного `Send()` — закрывает соединение напрямую.
