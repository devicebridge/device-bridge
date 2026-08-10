# PR-0023 Source Creation Error Handling

## Статус

Merged

---

## Цель

Устранить потерю ошибок создания Source. Ошибка `Registry.Create()` должна сохраняться и отменять Runtime.

---

## Проблема

Ошибка `Registry.Create(name)` молча игнорировалась через `continue`. Runtime продолжал работу с уже запущенными Sources.

---

## Решение

- Ошибка `Create` сохраняется через `errOnce.Do()` как `firstErr`.
- Runtime context отменяется — уже запущенные Sources получают `ctx.Done()`.
- `nil`-источники пропускаются без паники.

---

## Тесты

| Тест | Проверка |
|---|---|
| `TestNilSourceSkipped` | Nil source не вызывает panic, ok source работает |

---

## Результат

Ошибки создания Source обрабатываются единообразно с ошибками выполнения.
