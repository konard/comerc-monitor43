---
name: "x-queue-rabbitmq"
description: "Применяй когда проектируешь publisher/subscriber для RabbitMQ через API `../adapters/queue/rabbitmq`: `Definitions`, `NewPublisher2`, `NewSubscriber2`, retry через `x-death`, DLQ и `MultiQueueSubscriber`"
compatibility: ../adapters
---

# RabbitMQ

## Когда применять

Используй этот скилл, когда работаешь с очередями, routing key, DLQ и topology через `../adapters/queue/rabbitmq`.

Не применяй для:
- Kafka topic/group сценариев
- логики, где retry должен управляться возвращаемым `bool` из handler

## 1. Зафиксируй topology

Для устойчивого deploy/test паттерна используй `Definitions`.
Это основной способ описать:
- exchanges
- queues
- bindings
- retry/DLQ topology

`Definitions` годится и для:
- JSON-файла через `definitions.JSON()` под management API / `load_definitions`
- прямого применения topology в тестах

### Базовый retry/DLQ паттерн

Нужны:
- основная очередь
- retry очередь с `x-message-ttl`
- DLQ
- binding'и и, при необходимости, отдельный DLX exchange

Идея:
- обработчик падает
- subscriber публикует сообщение в retry queue
- по TTL RabbitMQ возвращает его в основную очередь
- после исчерпания `MaxRetries` сообщение уходит в DLQ

### `Definitions` как JSON

`Definitions` полезны не только для тестов, но и как переносимый topology artifact:

```go
payload, err := definitions.JSON()
```

Дальше этот JSON можно:
- загрузить через management API
- использовать через `load_definitions`
- импортировать через `rabbitmqctl import_definitions`

## 2. Publisher: предпочитай `NewPublisher2`

```go
pub := rabbitmq.NewPublisher2(rabbitmq.PublisherConfig{
    Exchange:          "my-exchange",
    RoutingKey:        "my.routing.key",
    Encoder:           encoders.JSON{},
    DeliveryMode:      amqp.Persistent,
    PublisherConfirms: 10,
}, dialer)
```

`NewPublisher` оставлен для обратной совместимости, но не как основной путь.

### Publisher confirms

Используй `PublisherConfirms`, когда цена потери сообщения высока.

| Режим | Когда использовать |
|---|---|
| `0` | максимум throughput, fire-and-forget |
| `1` | подтверждение на каждое сообщение |
| `N` | батчевое подтверждение |

## 3. Subscriber: предпочитай `NewSubscriber2`

```go
sub := rabbitmq.NewSubscriber2(rabbitmq.SubscriberConfig{
    QueueName:      "my-queue",
    MaxRetries:     3,
    RetryQueueName: "my-queue.retry",
    PrefetchCount:  1,
    MessageTimeout: 30 * time.Second,
}, dialer)
defer sub.Close()
```

Дефолты, которые нужно помнить:
- `RetryQueueName == ""` → обычно используется `QueueName + ".retry"`
- `PrefetchCount == 0` → адаптер подставляет безопасный дефолт, но для читаемого контракта лучше указывать явно
- `MessageTimeout == 0` → сообщение обрабатывается без per-message timeout

### Retry semantics

Retry строится через RabbitMQ DLX-механику и `x-death`.

Логика:
- ошибка + `x-death < MaxRetries` → публикация в retry queue
- ошибка + `x-death >= MaxRetries` → `Nack(requeue=false)` и уход в DLQ
- ошибка публикации в retry → fallback `Nack(requeue=true)`

Важно:
- `handler` возвращает `(bool, error)`, но `bool` здесь не участвует в retry-логике
- retry определяется только фактом ошибки и состоянием `x-death`

### Что проверить до запуска subscriber

- существует ли `RetryQueueName`
- есть ли route обратно в основную очередь
- настроен ли DLQ path
- совпадает ли `RoutingKey` publisher с topology

## 4. MultiQueueSubscriber

Используй `MultiQueueSubscriber`, когда:
- нужно читать несколько очередей
- хочется один канал и один последовательный обработчик
- нежелателен параллелизм между очередями

```go
sub := rabbitmq.NewMultiQueueSubscriber(dialer, rabbitmq.MultiQueueOptions{
    PrefetchCount: 10,
    MaxRetries:    3,
})
```

Важная деталь: `MultiQueueSubscriber` использует один канал и общий `Qos(..., global=true)` для fan-in обработки нескольких очередей.

Не брать его по инерции, если обработка может идти независимо и параллельно.

## 5. Практические детали

- для durable/persistent очередей фиксируй hostname у RabbitMQ node
- topology и retry queue должны быть созданы до старта subscriber
- в интеграционных тестах topology удобнее применять напрямую через `Definitions`
- recovery соединения не заменяет проверку topology: после reconnect убедись, что exchange/queue/bindings всё ещё соответствуют ожидаемому контракту

### Hostname caveat для durable node

Для persistent RabbitMQ node hostname должен быть стабильным: имя узла выглядит как `rabbit@<hostname>`. Если hostname меняется между перезапусками контейнера, ожидания по сохранённому состоянию и recovery могут не совпасть с фактическим поведением.

Минимальный ориентир для compose:

```yaml
services:
  rabbitmq:
    hostname: rabbitmq
```

### Recovery behaviour

Recovery соединения полезен, но не отменяет явную проверку topology и retry path после reconnect. Если topology drift недопустим, проверяй его отдельно, а не считай reconnect достаточной гарантией.

## Не делай

- не используй устаревшие `NewPublisher` / `NewSubscriber` как основной выбор
- не рассчитывай на `bool` из handler как на источник retry-поведения
- не запускай subscriber без подготовленной DLX/retry topology, если рассчитываешь на retry
