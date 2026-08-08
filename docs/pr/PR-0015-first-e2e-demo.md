# PR-0015 First End-to-End Demo

## Статус

Merged

---

## Цель

Создать первый воспроизводимый End-to-End сценарий, демонстрирующий прохождение сообщения через весь Device Bridge.

Сообщение, сформированное Scanner Source, достигает WebSocket-клиента без прямой связи между Scanner и WebSocket.

---

## Архитектурный маршрут

```
Scanner
   ↓
Bus
   ↓
Bridge
   ↓
Hub
   ↓
WebSocket
   ↓
Client
```

---

## Сценарий

1. Создание компонентов: Bridge, HTTP Server, WebSocket Handler.
2. Запуск Bridge runtime.
3. Подключение WebSocket-клиента к `/ws`.
4. Ожидание регистрации клиента в Hub (`Hub.Count() == 1`).
5. Запуск тестового Scanner с каналом `Input`.
6. Передача тестового значения через `Input`.
7. Получение Message WebSocket-клиентом (`ReadMessage`).
8. Проверка `source`, `payload`, `timestamp`.

---

## Выполненные изменения

Изменён:

- `internal/bridge/bridge.go` — добавлен метод `Bus()` для доступа к `bus.Bus`.

Добавлены:

- `internal/bridge/bridge_e2e_test.go` — два E2E-теста.
- `docs/pr/PR-0015-first-e2e-demo.md` — настоящий документ.

---

## Принятые решения

- `Bridge.Bus()` предоставляет доступ к шине для подключения Source.
- E2E-тесты инициируют событие на уровне Scanner, а не через прямое `Hub.Broadcast()`.
- Регистрация клиента ожидается через `Hub.Count()` с deadline 5 секунд.
- Каждое `ReadMessage` защищено `SetReadDeadline`.
- Физический сканер не требуется — используется существующий тестовый механизм `Input`.
- `Bridge.Bus()` гарантирует, что Scanner и Bridge используют одну шину.

---

## Тестирование

| Тест | Проверка |
|---|---|
| `TestE2ESingleMessage` | Одно сообщение: Scanner → Bus → Bridge → Hub → WebSocket → Client |
| `TestE2EMultipleMessages` | Три последовательных сообщения с проверкой порядка, source и payload |

Проверяется полный pipeline:

```
Scanner input
    ↓
Scanner
    ↓
Bus
    ↓
Bridge
    ↓
Hub
    ↓
WebSocket Handler
    ↓
WebSocket connection
    ↓
ReadMessage
```

---

## Что доказано

Данные от Source могут пройти через транспортный слой Device Bridge и быть доставлены подключённому WebSocket-клиенту.

---

## Совместимость

Изменений, нарушающих обратную совместимость, нет.

Дополнительные внешние зависимости не добавлены.

---

## Ограничения

Это demonstration/integration test, а не production UI и не интеграция с конкретным физическим сканером.

---

## Что намеренно НЕ входит

В данный PR не входят:

- frontend / browser application;
- физический сканер (USB/HID/Serial/COM/Bluetooth);
- бизнес-логика (1С, Wildberries, Честный знак);
- database / persistence;
- authentication / authorization / TLS;
- production deployment / Docker;
- metrics / logging subsystem;
- reconnect policy / message history / acknowledgement.

---

## Результат

Тестовый Scanner публикует значение, а независимый WebSocket-клиент получает то же значение через полный pipeline Device Bridge. Базовая архитектура Device Bridge получает первый полноценный End-to-End proof.
