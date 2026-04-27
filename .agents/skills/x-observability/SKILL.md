---
name: "x-observability"
description: "Применяй когда добавляешь новую внешнюю операцию, adapter package или server middleware: bootstrap monitoring через `../platform/monitoring`, tracer/meter в пакете, метрики и порядок HTTP/gRPC/GraphQL observability"
compatibility: ../platform, ../adapters
---

# Observability

Этот skill — **канонический владелец** observability policy:
- app bootstrap через `monitoring.InitDefault`
- package-level tracer/meter
- span naming и технические атрибуты
- meaningful metrics
- порядок HTTP/gRPC/GraphQL middleware/interceptors

Логирование как таковое принадлежит `x-log`.

## Когда применять

- добавляешь новый adapter или внешний вызов
- инструментируешь HTTP/gRPC/GraphQL слой
- настраиваешь monitoring bootstrap

## Core workflow

### 1. Инициализируй monitoring на уровне приложения

Ориентируйся на `../platform/monitoring`:

```go
closeMonitoring := monitoring.InitDefault(cfg)
defer func() {
    if err := closeMonitoring(); err != nil {
        slog.Default().Warn("failed to close monitoring", slog.Any("err", err))
    }
}()
```

Сбой telemetry не должен останавливать приложение.

### 2. В пакете объявляй один tracer и один meter

```go
var tracer = otel.Tracer("github.com/pure-golang/project/internal/repo")
var meter = otel.Meter("github.com/pure-golang/project/internal/repo")
```

Span name: `packageName.Operation`, например `sqlx.Get`, `S3.Put`, `rabbitmq.Publish`.

### 3. Добавляй только meaningful сигналы

- span для внешней операции
- технические атрибуты там, где они реально помогают расследованию
- метрики только там, где пакет владеет meaningful request/error/latency signal

### 4. Соблюдай порядок server observability

- HTTP: monitoring внешний, recovery после него
- gRPC: следуй project middleware order из `../adapters/grpc/middleware`
- GraphQL: не теряй parent HTTP span, если GraphQL сидит поверх HTTP

#### Самопроверка по HTTP, gRPC и GraphQL

##### HTTP

- `Monitoring` ставь снаружи `Recovery`
- `Monitoring` должен видеть финальный HTTP status
- если нужен trace ID в ответе, добавляй его на HTTP-слое, а не в бизнес-логике
- исключения для health/metrics paths принимай осознанно, а не по инерции
- `/graphql` не исключай, если GraphQL observability должна строиться как child span от HTTP span
- если операция внутри handler сама владеет внешним вызовом, дочерний span именуй как `packageName.Operation`
- `Monitoring` — whitelist путей; без аргументов мониторинг не применяется ни к одному запросу

Минимальная цепочка (GraphQL + REST):

```go
handler := amiddleware.Chain(
    mux,
    amiddleware.Monitoring("/graphql", "/api/*"),
    amiddleware.Recovery,
)
```

##### gRPC

- используй `SetupMonitoring` из `adapters/grpc/middleware` или при ручной сборке держи порядок `tracing -> metrics -> recovery -> logging`
- observability (tracing/metrics) снаружи recovery — чтобы видеть ошибки от паник
- если добавляешь свой interceptor, проверь, не ломает ли он propagation `context` и span attributes

Минимальная цепочка:

```go
grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        tracing.UnaryServerInterceptor(),
        metrics.UnaryServerInterceptor(),
        recovery.UnaryServerInterceptor(),
        logging.UnaryServerInterceptor(),
    ),
)
```

##### GraphQL

- GraphQL observability должна дополнять уже существующий HTTP span
- operation name/type полезнее писать и в span, и в logger context
- child span на operation — нормальный дефолт
- не тащи transport-level поля в business resolvers только ради логов

##### DB spans

- для SQL-пакетов используй `packageName.Operation` как имя span
- пиши `db.system`, `db.operation`, `db.transaction`, если пакет сам владеет этими сигналами
- `db.statement` используй только для безопасного нормализованного SQL
- имя запроса удобнее выносить в отдельный атрибут вроде `db.query_name`
- не пиши чувствительные данные в span attributes

Ориентиры в экосистеме:
- `../platform/monitoring`
- `../adapters/httpserver/middleware`
- `../adapters/grpc/middleware`
- `../adapters/graphql/interceptor/observability.go`

### 5. Помни о process-wide state в тестах

Если unit-тест меняет global OpenTelemetry state, не ставь `t.Parallel()`. Общие правила test-layer принадлежат `x-testing-conventions`.

## Короткий чек-лист

- monitoring bootstrap собран через `monitoring.InitDefault`
- tracer/meter не создаются на каждый вызов
- span name согласован с `packageName.Operation`
- метрики добавлены только там, где сигнал действительно принадлежит пакету
- middleware order не придуман вручную

## Смежные skills

- `x-log`
- `x-testing-conventions`
