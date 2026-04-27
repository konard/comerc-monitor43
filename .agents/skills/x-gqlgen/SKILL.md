---
name: "x-gqlgen"
description: "Применяй когда работаешь с GraphQL через gqlgen: `gqlgen.yml`, раскладка generated vs resolvers, тонкие резолверы, привязка к domain-типам и интеграция с `transport/http/`"
---

# gqlgen

Этот skill — канонический владелец проектной раскладки gqlgen и правил написания резолверов.

## Когда применять

- добавляешь или меняешь GraphQL-схему в `internal/transport/http/graph/*.graphqls`
- пишешь или правишь резолверы
- настраиваешь или меняешь `gqlgen.yml`
- интегрируешь GraphQL-handler в HTTP-роутер

Не применяй для:
- общих правил `doc.go` и package contract (→ `x-doc-go`)
- частично применяемых интерфейсов и типов в их сигнатурах (→ `x-unit-test-partial-interface`)
- тестовых соглашений (→ `x-testing-conventions`)
- консьюмерских моков (→ `x-mockery`)

## Ключевой принцип раскладки

gqlgen-пакет с generated-кодом — это **контракт, а не место для логики**. Он играет ту же роль, что `internal/transport/grpc/pb/`: держит схему и сгенерированный код, читается только автогенератором, правится руками только в части `.graphqls`.

Имплементация резолверов живёт в **отдельном пакете**, как обычный внутренний код проекта, и подчиняется общим правилам: частично применяемые интерфейсы, тонкая оркестрация, нет горизонтальных импортов между siblings.

## Раскладка пакетов

```
internal/transport/http/
  transport.go                   — HTTP-сервер, монтирует /graphql и /playground
  graph/                         — контракт (аналог grpc/pb/)
    doc.go
    schema.graphqls              — source of truth
    generated.go                 — gqlgen exec, single-file layout
    model/
      models_gen.go              — сгенерированные типы моделей
  resolvers/                     — имплементация
    doc.go
    resolver.go                  — Resolver struct, конструктор, partial interfaces
    <name>.resolvers.go          — генерируемые стабы, заполняются вручную
```

Пакеты: `graph` для контракта, `model` для моделей, `resolvers` для имплементации. Имя `resolvers` намеренно отличается от `graph`, чтобы при импорте было очевидно, что лежит в каждом пакете.

## gqlgen.yml

`gqlgen.yml` лежит в корне репозитория.

```yaml
schema:
  - internal/transport/http/graph/*.graphqls

exec:
  package: graph
  layout: single-file
  filename: internal/transport/http/graph/generated.go

model:
  filename: internal/transport/http/graph/model/models_gen.go
  package: model

resolver:
  package: resolvers
  layout: follow-schema
  dir: internal/transport/http/resolvers
  filename_template: "{name}.resolvers.go"

# Привязка GraphQL-типов к существующим Go-типам.
# Первая линия обороны против жирных резолверов: если domain/dto тип совпадает
# со схемой по полям — биндим напрямую, резолвер возвращает его без маппинга.
models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.ID
      - github.com/99designs/gqlgen/graphql.Int64
  # Пример привязки к проектному domain-типу:
  # Status:
  #   model: some-project-url/internal/domain.Status
```

### Federation

Секция `federation` подключается, только если сервис участвует в Apollo Federation как subgraph. Признаки:

- рядом живёт router/gateway, собирающий supergraph из нескольких subgraph'ов
- схема использует federation-директивы (`@key`, `@external`, `@requires`, `@provides`, `@shareable`)
- есть потребность в entity resolvers, которые тянут данные этого сервиса по ссылке из другого subgraph'а

Если ничего из этого нет — федерацию **не включай**: она добавляет `federation.go`, `_Entity` union, `_service` query и расширяет resolver-контракт, это лишний сгенерированный шум.

Когда включаешь, минимальная секция:

```yaml
federation:
  filename: internal/transport/http/graph/federation.go
  package: graph
  version: 2
```

`version: 2` — текущий дефолт Apollo Federation. `version: 1` только если gateway старый и не умеет v2.

## Тонкие резолверы

Резолвер — это граница между GraphQL и use case слоем, не место для бизнес-логики.

### Правило конструктора

```go
// internal/transport/http/resolvers/resolver.go
package resolvers

import (
    "context"

    "some-project-url/internal/domain"
)

type statusService interface {
    GetStatus(ctx context.Context) (*domain.Status, error)
}

// Resolver реализует graph.ResolverRoot.
type Resolver struct {
    statusService statusService
}

// New создаёт резолвер.
func New(statusService statusService) *Resolver {
    return &Resolver{statusService: statusService}
}
```

- зависимости — частично применяемые интерфейсы, объявленные здесь же
- сигнатуры интерфейсов работают с **domain-типами**, а не с gqlgen `model.*` — это защищает сервисный слой от знания о GraphQL
- конструктор не делает I/O, ничего не стартует

### Правило метода резолвера

Каждый генерированный стаб в `<name>.resolvers.go` заполняется так:

```go
func (r *queryResolver) Status(ctx context.Context) (*model.Status, error) {
    st, err := r.statusService.GetStatus(ctx)
    if err != nil {
        return nil, err
    }
    return mapDomainStatus(st), nil
}
```

- распаковка аргументов → вызов сервиса через интерфейс → маппинг → возврат
- если domain-тип забинден через `models:` — маппинг не нужен, возвращай `st` напрямую
- маппинг-функции живут тут же в пакете `resolvers`, приватные (`mapDomainStatus`), не тащатся наружу
- ошибки возвращай как есть; превращение в `*gqlerror.Error` с кодами — только если нужна клиент-ориентированная семантика (например, `NOT_FOUND`), и тогда оборачивай явно

### Анти-паттерны

- импорт конкретных service-пакетов из `resolvers` — нет, только интерфейсы
- бизнес-логика в резолвере (валидация правил, оркестрация нескольких сервисов, транзакции) — выноси в use case слой проекта
- прямой доступ из резолвера в `repo/`, `adapters/` — нет, всё через service
- хранение состояния в `Resolver` между запросами (кеши, in-memory аккумуляторы) — нет, резолвер создаётся один раз и должен быть stateless

## Интеграция с transport/http/

Резолвер инстанцируется в `cmd/*/main.go` и передаётся в `transport/http/`:

```go
// cmd/<name>/main.go
resolver := resolvers.New(statusService, ...)
httpTransport := httptransport.New(authService, resolver)
```

`transport/http/transport.go` монтирует GraphQL-маршруты через `handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))`:

```go
import (
    "github.com/99designs/gqlgen/graphql/handler"
    "github.com/99designs/gqlgen/graphql/playground"

    "some-project-url/internal/transport/http/graph"
    "some-project-url/internal/transport/http/resolvers"
)

type Transport struct {
    authService authService
    resolver    *resolvers.Resolver
}

func (t *Transport) EnrichRoutes(mux *http.ServeMux) {
    // ... REST routes
    if t.resolver != nil {
        srv := handler.NewDefaultServer(
            graph.NewExecutableSchema(graph.Config{Resolvers: t.resolver}),
        )
        mux.Handle("POST /graphql", srv)
        mux.Handle("GET /playground", playground.Handler("GraphQL", "/graphql"))
    }
}
```

Ручной JSON-парсинг запроса и самописный роутинг по полям `query` запрещены — всё идёт через `handler.NewDefaultServer`.

## Re-generation workflow

- `Taskfile.yml` содержит таск `gqlgen` → `go run github.com/99designs/gqlgen generate`
- запуск: `task gqlgen`
- после смены схемы: отредактировать `schema.graphqls` → `task gqlgen` → заполнить новые стабы в `<name>.resolvers.go` → прогнать тесты
- стабы новых полей выглядят как `panic(fmt.Errorf("not implemented: ..."))` — оставлять panic в коммите нельзя, это сразу падающий тест
- сгенерированные файлы (`generated.go`, `models_gen.go`, `*.resolvers.go` в части верхних комментариев) правятся только автогеном; ручные правки стабов в `*.resolvers.go` сохраняются между запусками — gqlgen не перезаписывает уже реализованные методы

## Когда переключаться на manual-режим

По умолчанию секция `resolver` в `gqlgen.yml` включена — gqlgen поддерживает стабы и ловит drift между схемой и кодом. Это путь по умолчанию.

Секция `resolver` **целиком комментируется** только в одном случае: когда стандартный layout не ложится на архитектуру — например, если понадобится split на resolver-per-domain с разными инъекциями зависимостей в разные подструктуры, или если схема вырастет настолько, что автогенерация стабов станет шумом в diff'ах.

Пока резолверов мало и они однородные — не комментировать. Переключение на manual-режим обсуждается отдельным архитектурным решением, не делается «на всякий случай».

## Тестовая стратегия

Unit-тесты резолверов — уровнем пакета `resolvers`, через мок частично применяемого интерфейса (`statusService` и подобные):

```go
func TestResolver_Status(t *testing.T) {
    t.Parallel()

    mockService := mocks.NewStatusService(t)
    mockService.EXPECT().GetStatus(mock.Anything).Return(&domain.Status{...}, nil)

    r := resolvers.New(mockService)
    got, err := r.Query().Status(context.Background())
    // ...
}
```

Интеграция через `handler.NewDefaultServer` с реальной схемой — уровень integration или bdd, не unit. Моки генерируются через mockery, правила — в `x-mockery`.

## Что не принадлежит этому skill

- правила `doc.go` и package contract → `x-doc-go`
- частично применяемые интерфейсы и требования к типам в их сигнатурах → `x-unit-test-partial-interface`
- test layers, `t.Parallel()`, AAA → `x-testing-conventions`
- генерация моков → `x-mockery`
- ошибки и `%q`-форматирование → `x-errors`
- logging в резолверах → `x-log` (обычно логирование делается уровнем handler middleware, не в резолверах)
