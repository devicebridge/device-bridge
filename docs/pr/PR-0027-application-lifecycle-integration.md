# PR-0027 Application Lifecycle Integration

## Статус

Merged

---

## Цель

Связать все ранее реализованные lifecycle механизмы в единый application lifecycle.

---

## Тесты

| Тест | Проверка |
|---|---|
| `TestFullApplicationLifecycle` | Scanner → Bus → Hub → WebSocket → Client → cancel → shutdown |
| `TestRunApplicationRejectsOccupiedPort` | Ошибка bind HTTP-порта возвращается до запуска runtime |
| `TestRunApplicationStopsOnContextCancellation` | Отмена application context завершает Bridge и HTTP server |

---

## Проверяемый маршрут

```text
Scanner публикует "hello-world"
  ↓
Bus
  ↓
Bridge Runtime
  ↓
Hub
  ↓
WebSocket Client
  ↓
WebSocket connection получает "hello-world"
  ↓
cancel()
  ↓
Bridge.Run() → nil
  ↓
Hub.Shutdown()
  ↓
httptest cleanup
```

---

## Результат

Полный интеграционный тест подтверждает: данные от Source проходят через весь pipeline и доставляются WebSocket-клиенту. Завершение — без зависаний и утечек.

Application orchestration вынесен в `runApplication`, поэтому ошибки bind, cancellation и shutdown можно проверять без запуска отдельного процесса.
