# PR-0021 WebSocket Client Shutdown Robustness

## Статус

Merged

---

## Цель

Устранить потенциальный deadlock между `Client.Send()` и `Client.Close()` при блокирующей записи WebSocket.

---

## Проблема

Синхронный `Send()` мог блокировать Hub внутри `conn.WriteMessage()`, если клиент перестал читать сообщения. Это удерживало путь рассылки и осложняло shutdown.

---

## Решение

- `Client` использует bounded outbound queue размером 16 сообщений.
- `Send()` только сериализует и помещает сообщение в очередь, не выполняя сетевую запись синхронно.
- Отдельная writer goroutine выполняет `WriteMessage()`.
- Переполненная очередь возвращает `ErrQueueFull`; Hub удаляет и закрывает такого клиента.
- `Close()` останавливает writer и закрывает WebSocket-соединение.
- `sync.Once` гарантирует идемпотентность shutdown.

---

## Тесты

| Тест | Проверка |
|---|---|
| `TestCloseIdempotent` | Повторный Close без паники |
| `TestSendAfterClose` | Send после Close возвращает ошибку |
| `TestConcurrentSendAndClose` | Конкурентные Send + Close без паники и гонок |
| `TestSendReturnsWhenOutboundQueueIsFull` | Переполненная очередь не блокирует Send |

---

## Результат

Медленный WebSocket-клиент больше не блокирует Hub: запись выполняется в отдельной goroutine, а переполнение bounded queue приводит к удалению клиента.
