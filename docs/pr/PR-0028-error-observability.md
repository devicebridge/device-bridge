# PR-0028 Runtime Error and Shutdown Observability

## Статус

Merged

---

## Цель

Сделать причины завершения Runtime различимыми без внедрения полноценного logging framework.

---

## Решение

Сообщения об ошибках чётко идентифицируют источник:

| Сообщение | Причина |
|---|---|
| `config error` | Ошибка загрузки конфигурации |
| `http server startup error` | HTTP listen/serve failure |
| `runtime error during shutdown` | Ошибка Source во время остановки |
| `http graceful shutdown error` | Ошибка graceful shutdown HTTP |
| `application exited with error` | Итоговая ошибка приложения |

Используется стандартный `log`, без внешних зависимостей.

---

## Результат

Причины завершения приложения однозначно идентифицируются по логам.
