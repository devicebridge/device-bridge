# PR-0031 Concurrency Audit Fixes

## Статус

Merged

---

## Исправления

- `Bus` получил идемпотентный shutdown-сигнал и безопасную конкурентную публикацию.
- `Bridge` допускает только один запуск runtime.
- Bus → Hub dispatcher завершает ожидание новых сообщений по `Bus.Done()` и дренирует накопленный буфер.
- `Registry` защищает factories через `sync.RWMutex`.
- `Hub.Shutdown()` очищает реестр клиентов до вызова `Client.Close()`.
- `Hub.Broadcast()` удаляет и закрывает клиента, если его `Send()` завершился ошибкой, чтобы неисправный клиент не блокировал последующие рассылки.
- Закрытие клиента дедуплицируется внутри Hub, поэтому конкурентные `Broadcast()` и `Shutdown()` не вызывают `Client.Close()` повторно.
- `Hub.Broadcast()` продолжает рассылку здоровым клиентам и возвращает агрегированную ошибку после обработки всех клиентов.
- WebSocket Client использует bounded outbound queue и отдельную writer goroutine; медленный клиент не блокирует Hub.
- Ошибки рассылки из Bridge dispatcher логируются вместо безусловного игнорирования.
- `Bridge` больше не вызывает `WaitGroup.Add` из запущенного coordinator; счетчик устанавливается до `go`, что устраняет race `Add`/`Wait`.
- `Bridge.Run()` дожидается source → Bus forwarders перед `Bus.Close()`, устраняя потерю сообщений при быстром завершении source.

## Проверки

- Конкурентные операции `Registry` покрыты тестом `TestConcurrentRegistryAccess`.
- Полный suite и race detector проходят после всех исправлений.
