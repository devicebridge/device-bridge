# Device Bridge

Device Bridge — легковесный транспортный слой между локальными источниками данных и web-приложениями.

## Архитектура

```
Source
  ↓
Bus
  ↓
Bridge Runtime
  ↓
Hub
  ↓
Client (WebSocket)
  ↓
Web-клиент
```

### Source

Поставщик данных. Получает данные от внешнего устройства (сканер, датчик), формирует `Message` и публикует в общий Bus. Не знает о Hub, WebSocket или HTTP.

### Bus

Транспорт сообщений внутри Runtime. Поддерживает Publish/Subscribe. Context-aware публикация (`PublishCtx`) для корректного завершения при shutdown.

### Bridge Runtime

Центральный координатор. Запускает зарегистрированные Sources, управляет их lifecycle, связывает Source output с Bus, доставляет сообщения из Bus в Hub.

### Hub

Широковещательная рассылка сообщений подключённым клиентам. Поддерживает Register/Unregister клиентов, Broadcast сообщений и Shutdown.

### WebSocket Client

Транспортный клиент для доставки сообщений web-клиентам через WebSocket. Сериализует Message в JSON.

## Формат сообщения

```json
{
  "source": "scanner-main",
  "timestamp": 1785472345123,
  "payload": "..."
}
```

## Lifecycle

### Штатное завершение

```
SIGINT / SIGTERM
      ↓
context cancellation
      ↓
Bridge.Run(ctx)
  ├── Sources останавливаются
  ├── Bus закрывается
  └── Hub forwarder завершается
      ↓
Hub.Shutdown() — закрытие WebSocket клиентов
      ↓
HTTP Server.Shutdown()
      ↓
процесс завершается
```

### Ошибка Source

```
Source → ошибка
      ↓
сохраняется первая ошибка
      ↓
runtime context отменяется
      ↓
остальные Sources останавливаются
      ↓
Bus закрывается
      ↓
Bridge.Run() возвращает ошибку
      ↓
приложение завершается с ненулевым кодом
```

## Конфигурация

| Переменная | Назначение | По умолчанию |
|---|---|---|
| `DEVICE_BRIDGE_HTTP_HOST` | HTTP listen host | `0.0.0.0` |
| `DEVICE_BRIDGE_HTTP_PORT` | HTTP listen port | `8080` |
| `DEVICE_BRIDGE_SOURCES` | Comma-separated source names | empty |
| `DEVICE_BRIDGE_SCANNER_PATH` | Serial-compatible scanner path | empty |
| `DEVICE_BRIDGE_HID_PATH` | Linux input event scanner path | empty |
| `DEVICE_BRIDGE_SCANNER_BAUD` | Configured serial baud rate | `9600` |
| `DEVICE_BRIDGE_SCANNER_RECONNECT_SECONDS` | Reconnect delay after port failure | `1` |

Supported source names are `scanner-main` and `scanner-secondary`. Each configured scanner uses a bounded transport-neutral input adapter. Physical device adapters can publish into the same scanner input contract.

Serial integration uses a line-oriented `io.ReadCloser` boundary. The adapter supports `LF`, `CRLF`, final values before `EOF`, and cancellation of blocking reads. Device opening and serial port configuration are platform-specific and are not part of the cross-platform core.

When `DEVICE_BRIDGE_SCANNER_PATH` is set, `scanner-main` reads from that path. In the development VM it can point to a PTY endpoint created with `socat`. Alternatively, on Linux, `DEVICE_BRIDGE_HID_PATH` can point to `/dev/input/eventN` for a keyboard-like scanner. Serial and HID paths are mutually exclusive. The cross-platform core validates baud and reconnect settings; OS-specific port configuration remains in the device opener.

The HID keyboard mapping supports US-layout letters, digits, punctuation, Shift, Caps Lock, keypad digits, Space, Backspace, and Enter. GS/FNC1 replacement remains scanner configuration and application-level parsing policy.

## Быстрый старт

```bash
make build
./device-bridge
```

WebSocket endpoint: `ws://localhost:8080/ws`

Для установки на Debian 12 как systemd-службы см. [`docs/deployment/debian12.md`](docs/deployment/debian12.md).

Health endpoints:

- `GET /healthz` — process health;
- `GET /readyz` — application readiness; returns `503` until runtime startup and during shutdown.

Delivery is at-most-once. WebSocket clients have bounded outbound queues; a slow client is disconnected when its queue is full rather than blocking other clients.

## Статус

Проект находится в активной разработке.

Транспортный слой, lifecycle и concurrency hardened.
Следующий этап — интеграция с физическими устройствами.
