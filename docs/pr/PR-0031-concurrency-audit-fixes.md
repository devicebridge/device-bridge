# PR-0031 Concurrency Audit Fixes

## Статус

In progress

---

## Исправления

- `Bus` получил идемпотентный shutdown-сигнал и безопасную конкурентную публикацию.
- `Bridge` допускает только один запуск runtime.
- Bus → Hub dispatcher завершает ожидание новых сообщений по `Bus.Done()` и дренирует накопленный буфер.
- `Registry` защищает factories через `sync.RWMutex`.

## Проверки

- Конкурентные операции `Registry` покрыты тестом `TestConcurrentRegistryAccess`.
- Полный suite и race detector должны проходить перед переводом документа в статус `Merged`.
