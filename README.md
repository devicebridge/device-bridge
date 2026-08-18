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

Supported source names are `scanner-main` and `scanner-secondary`. Source input adapters are the next integration step; configuring a source starts its lifecycle and keeps the application running until shutdown.

## Быстрый старт

```bash
make build
./device-bridge
```

WebSocket endpoint: `ws://localhost:8080/ws`

## Статус

Проект находится в активной разработке.

Транспортный слой, lifecycle и concurrency hardened.
Следующий этап — интеграция с физическими устройствами.
