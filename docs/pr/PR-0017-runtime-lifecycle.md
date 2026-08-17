# PR-0017 Runtime Lifecycle

## Статус

Merged

---

## Цель

Ввести управляемый жизненный цикл Bridge Runtime через `context.Context`, обеспечить корректное завершение Sources, Bus и Runtime, предотвращать зависание goroutine при остановке.

---

## Исходная проблема

После PR-0016 Bridge Runtime не имел управляемого lifecycle:
- `Bridge.Run()` не принимал context.
- Ошибки Source терялись через `_ = src.Run()`.
- Не было механизма остановки Runtime.
- При ошибке одного Source остальные продолжали работу.
- Bus не закрывался.

---

## Новая модель lifecycle

### Интерфейс Source

```go
type Source interface {
    Run(ctx context.Context, out chan<- message.Message) error
}
```

### Bridge.Run

```go
func (b *Bridge) Run(ctx context.Context) error
```

`Bridge` является одноразовым runtime-объектом: первый вызов `Run` атомарно занимает lifecycle, а повторный или конкурентный вызов возвращает `ErrAlreadyRunning`.

---

## Порядок shutdown

### Штатное завершение

```
ctx cancellation
      ↓
runtime cancel
      ↓
Sources stop
      ↓
wait all Sources (WaitGroup)
      ↓
close Bus
      ↓
Bus → Hub stops
      ↓
Bridge.Run() → nil
```

### Аварийное завершение

```
Source A → error
      ↓
save first error
      ↓
cancel runtime
      ↓
other Sources stop
      ↓
wait all Sources (WaitGroup)
      ↓
close Bus
      ↓
Bus → Hub stops
      ↓
Bridge.Run() → source error
```

---

## Правила обработки ошибок

- Ошибка Source больше не игнорируется.
- Ошибка одного Source инициирует остановку Runtime.
- Остальные Sources получают `ctx.Done()`.
- Bridge ждёт завершения всех Sources через `sync.WaitGroup`.
- Bus закрывается после завершения всех Sources.
- Штатная отмена контекста возвращает `nil`.
- `context.Canceled` не считается ошибкой Runtime.
- Сохраняется первая полученная ошибка Source.
- Один экземпляр `Bridge` нельзя запускать повторно или одновременно из нескольких goroutine.

---

## Выполненные изменения

### Source интерфейс
- `Run(ctx context.Context, out chan<- message.Message) error`

### Scanner
- `Run()` использует `select` с `ctx.Done()` для реакции на cancellation.
- При штатной отмене возвращает `ctx.Err()` (`context.Canceled`).

### Bus
- Добавлен метод `Close()` для закрытия канала.

### Bridge Runtime
- `Run(ctx context.Context) error` — создаёт дочерний `runtimeCtx`.
- Запускает все Sources из Registry.
- Каждый Source: отдельная горутина, выходной канал подключён к Bus через forwarder.
- `sync.WaitGroup` ожидает завершения всех Sources.
- При ошибке Source: сохраняется первая ошибка, `runtimeCtx` отменяется.
- Bus закрывается после завершения Sources.
- Обработка `Bus → Hub` завершается при закрытии Bus.

---

## Тестирование

| Тест | Проверка |
|---|---|
| `TestRun` | Сообщение от Source доходит до Hub |
| `TestNormalShutdown` | Отмена контекста → `Run()` возвращает `nil` |
| `TestSourceError` | Ошибка Source возвращается из `Run()` |
| `TestSourceErrorStopsOthers` | Ошибка Source A → Source B получает cancellation |
| `TestMultipleSources` | Два Source публикуют сообщения, оба доходят до Hub |
| `TestFirstErrorPreserved` | При ошибках нескольких Source возвращается одна ошибка |
| `TestCancellation` (scanner) | Scanner реагирует на `ctx.Done()` |

Все E2E-тесты обновлены для работы с `context.Context`.

---

## HTTP/WebSocket

Graceful shutdown HTTP Server и WebSocket в данный PR не входит — это отдельная задача.

---

## Что намеренно НЕ входит

- Graceful shutdown HTTP Server.
- Shutdown WebSocket connections.
- OS signal handling.
- Автоматический restart Sources.
- Health checks / metrics / logging framework.
- Динамическое добавление/удаление Sources.

---

## Результат

Runtime Device Bridge получил управляемый жизненный цикл. При отмене контекста все Sources корректно завершаются, Bus закрывается, Runtime корректно завершает свои основные goroutine при штатной отмене и ошибке Source. Ошибки Sources больше не теряются.
