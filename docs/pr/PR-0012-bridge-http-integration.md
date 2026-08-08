# PR-0012 Bridge ↔ HTTP Integration

## Статус

Merged

---

## Цель

Подключить существующий WebSocket Adapter к HTTP Server и Bridge Hub.

После завершения PR HTTP-сервер Device Bridge должен обслуживать WebSocket endpoint, через который подключённые клиенты получают сообщения из существующего `Hub`.

PR выполняет только интеграцию уже существующих компонентов. Новая бизнес-логика не добавляется.

---

## Выполненные изменения

Изменены:

- `internal/bridge/bridge.go` — добавлен метод `Hub()` для доступа к `hub.Hub`.
- `cmd/device-bridge/main.go` — точка сборки компонентов и запуск HTTP-сервера.

Добавлены:

- `internal/bridge/bridge_integration_test.go` — интеграционный тест полного цикла.
- `docs/pr/PR-0012-bridge-http-integration.md` — настоящий документ.

---

## Принятые решения

- `Bridge` предоставляет доступ к `Hub` через минимальный метод `Hub()`.
- WebSocket Handler регистрируется через `server.Handle()`.
- Endpoint WebSocket: `/ws`.
- Точка сборки приложения (`main.go`) связывает `Bridge`, `WebSocket Handler` и `HTTP Server`.
- `Bridge` запускается в горутине независимо от HTTP-сервера.
- Ошибка `ListenAndServe` обрабатывается через `log.Fatal`.

---

## Архитектурные решения

Зафиксированы следующие архитектурные решения:

- WebSocket Adapter остаётся транспортным слоем.
- HTTP Server отвечает только за HTTP routing.
- Bridge является владельцем Hub.
- Точка сборки связывает Bridge, WebSocket Handler и HTTP Server.
- Второй Hub не создаётся.
- `websocket` не создаёт Bridge и Hub.
- `server` не знает о Bridge и не импортирует `websocket`.
- `Hub` не знает о HTTP.
- `Bridge` не знает детали WebSocket-протокола.

---

## Тестирование

Добавлен интеграционный тест полного цикла:

```
Bridge.Hub
    ↓
WebSocket Handler
    ↓
HTTP Server
    ↓
httptest.Server
    ↓
WebSocket client
```

Тест проверяет:

1. HTTP Server регистрирует WebSocket endpoint.
2. WebSocket-клиент может подключиться к endpoint.
3. Сообщение, отправленное через `Bridge.Hub.Broadcast()`, доставляется клиенту.
4. Все поля `message.Message` совпадают.
5. После закрытия WebSocket-соединения клиент удаляется из Hub (проверяется косвенно: повторный `Broadcast` после закрытия не возвращает ошибку).

Тест использует `httptest.Server` и `websocket.DefaultDialer`. Mock-объекты WebSocket не используются.

---

## Совместимость

Изменений, нарушающих обратную совместимость, нет.

Дополнительные внешние зависимости не добавлены.

---

## Что намеренно НЕ входит

В данный PR не входят:

- конфигурация;
- CLI flags;
- переменные окружения;
- authentication;
- authorization;
- CORS;
- heartbeat;
- ping/pong policy;
- reconnect logic;
- graceful shutdown;
- logging subsystem;
- metrics;
- REST API;
- новые Source;
- Scanner Source;
- изменение формата Message;
- изменение Hub API;
- изменение WebSocket протокола;
- health-check endpoint;
- дополнительные HTTP endpoints.

---

## Результат

Проект получил работающий путь от Bridge.Hub до WebSocket-клиента через HTTP Server:

```
Bridge.Hub
    │
    ▼
WebSocket Handler
    │
    ▼
HTTP Server
    │
    ▼
/ws
    │
    ▼
WebSocket client
```
