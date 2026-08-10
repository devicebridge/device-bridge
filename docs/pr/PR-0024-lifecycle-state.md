# PR-0024 Runtime Lifecycle State Model

## Статус

Merged

---

## Цель

Проверить, что конкурентные и повторные операции lifecycle не приводят к паникам, гонкам или зависаниям.

---

## Проверенные сценарии

| Сценарий | Тест |
|---|---|
| Многократный cancel | `TestMultipleCancellation` — 3 вызова, без паники |
| Конкурентный cancel + ошибка Source | `TestConcurrentCancelAndSourceError` — ошибка сохранена |
| Уже отменённый context | `TestApplicationNormalShutdown` |
| Штатное завершение | `TestNormalShutdown`, `TestContextCancellationIsNotError` |
| Ошибка Source | `TestSourceError`, `TestApplicationErrorPropagation` |

---

## Результат

Runtime lifecycle устойчив к конкурентным и повторным операциям. State machine не потребовался.
