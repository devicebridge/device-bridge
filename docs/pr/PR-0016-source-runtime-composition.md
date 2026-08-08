# PR-0016 Source Runtime Composition

## Статус

Merged

---

## Цель

Устранить расхождение между утверждённой архитектурой и текущей реализацией `Source`.

После PR-0016 Bridge становится центром runtime-композиции: Source публикует сообщения в выходной канал, а Runtime доставляет их в Bus.

---

## Исходная проблема

Архитектурный контракт предполагает, что Source публикует сообщения через переданный канал:

```go
type Source interface {
    Run(chan<- message.Message) error
}
```

Однако Scanner напрямую получал `*bus.Bus` и вызывал `bus.Publish()` самостоятельно. Схема выглядела как `Scanner → Bus` вместо `Scanner → output channel → Bus`.

Кроме того, Bridge не отвечал за запуск зарегистрированных Sources — композиция выполнялась вручную через `Bridge.Bus()`.

---

## Архитектурное решение

Source больше не зависит от Bus. Source публикует сообщения через переданный в `Run()` канал:

```go
func (s *Scanner) Run(out chan<- message.Message) error {
    // ...
    out <- msg
    // ...
}
```

Bridge Runtime отвечает за:
1. Получение зарегистрированных Sources из Registry.
2. Запуск каждого Source в отдельной горутине.
3. Передачу сообщений из выходного канала Source в общий Bus.
4. Чтение сообщений из Bus и доставку в Hub.

---

## Выполненные изменения

### Scanner

- Удалена зависимость от `*bus.Bus`.
- Конструктор: `New(sourceID string, input <-chan Input)` — только sourceID и канал ввода.
- `Run(out chan<- message.Message)` использует выходной канал для публикации.

### Registry

- Добавлен метод `Names() []string` для перечисления зарегистрированных источников.

### Bridge

- `Bus()` удалён — прямой доступ к Bus через Bridge больше не нужен.
- Добавлен `Registry() *source.Registry` для регистрации Sources.
- `Run()` запускает все зарегистрированные Sources и соединяет их с Bus.
- Добавлен `connectSource()` — создаёт выходной канал, запускает Source, перенаправляет сообщения в Bus.

### Тесты

- Тесты Scanner обновлены: используется выходной канал вместо `bus.Subscribe()`.
- E2E-тесты обновлены: Scanner регистрируется через Registry, Bridge запускает источники.
- Интеграционный тест (`TestBridgeHTTPIntegration`) не изменён — он проверяет Hub→WebSocket и не зависит от Source.

---

## Роль Registry

Registry является единственным источником истины для зарегистрированных Sources. Bridge получает список имён через `Names()` и создаёт каждый Source через `Create()`.

Factory использует closure для захвата входного канала:

```go
b.Registry().Register("scanner-main", func() source.Source {
    return scanner.New("scanner-main", input)
})
```

---

## Итоговая схема

```
Source
  ↓
output channel
  ↓
Bus
  ↓
Bridge Runtime
  ↓
Hub
  ↓
WebSocket
```

---

## Совместимость

Изменений, нарушающих обратную совместимость, нет.

Дополнительные внешние зависимости не добавлены.

---

## Ограничения

- Ошибка Source не обрабатывается Runtime (заглушается `_`). Полноценная система error supervision — отдельная задача.
- Политика автоматического перезапуска Source не реализована.
- Lifecycle API (`Start`/`Stop`/`Shutdown`) не формализован.

---

## Что намеренно НЕ входит

- Реализация error supervision / restart policy.
- Формализация lifecycle API.
- Изменение Message format / Bus / Hub / WebSocket.
- Новые абстракции (broker, event bus, DI framework).

---

## Результат

Ключевой архитектурный принцип соблюдён:

> Source знает только о своём входе и выходном канале сообщений. Bus, Hub и WebSocket являются ответственностью Runtime-композиции и не проникают внутрь Source.
