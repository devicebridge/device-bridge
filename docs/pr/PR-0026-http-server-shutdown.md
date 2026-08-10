# PR-0026 HTTP Server Graceful Shutdown

## Статус

Merged

---

## Цель

Завершить lifecycle HTTP-сервера — интеграция в общий жизненный цикл приложения.

---

## Решение

- `*http.Server` с явным `ListenAndServe()` и `Shutdown()`.
- Shutdown timeout 5 секунд.
- `http.ErrServerClosed` не считается ошибкой.
- HTTP startup error вызывает завершение всего приложения.

---

## Тесты

| Тест | Проверка |
|---|---|
| `TestHTTPServerGracefulShutdown` | Bridge → Hub.Shutdown → HTTP.Shutdown, без зависаний |

---

## Результат

HTTP Server полностью интегрирован в единый lifecycle приложения.
