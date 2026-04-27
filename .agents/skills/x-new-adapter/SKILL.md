---
name: "x-new-adapter"
description: "Применяй когда добавляешь новый инфраструктурный адаптер: выбор места в дереве, `doc.go`, `Config`, конструктор, lifecycle-контракт, observability и тесты"
compatibility: ../adapters
---

# Новый адаптер

Этот skill собирает workflow создания нового adapter package и ссылается на смежные skills, а не пересказывает их.

## Когда применять

- создаёшь новый adapter package вроде `queue/...`, `storage/...`, `db/...`, `executor/...`

Не применяй для:
- service/repo/controller пакетов прикладного сервиса
- доменной логики без внешней зависимости

## Core workflow

### 1. Выбери место в дереве пакетов

Если нужна совместимость с legacy-экосистемой `../adapters`, используй её только как карту текущего API.

### 2. Не дублируй domain внешней библиотеки

Типы из внешней библиотеки (SDK, gRPC client, go-tdlib и т.п.) — это и есть рабочий контракт адаптера и его потребителей. Не создавай параллельный `internal/domain/*` как «тонкий фасад» над `*sdk.Foo`.

`internal/domain/` существует только для того, чего нет во внешней библиотеке: внутренние state-events, value objects, бизнес-правила.

Правила про типы в частично применяемых интерфейсах у сервисов-потребителей адаптера и типичную ловушку со scaffolding-фазой — в `x-unit-test-partial-interface`.

### 3. Оформи package contract

- прочитай ближайший родительский `doc.go`
- создай или обнови `doc.go` нового пакета
- детали структуры `doc.go` бери из `x-doc-go`

### 4. Оформи `Config` и lifecycle

- `Config` через env workflow из `x-env-config`
- конструктор без I/O
- явные `Start()/Connect()/Run()/Close()`, если lifecycle нужен

### 5. Подключи observability

- logging policy → `x-log`
- tracing/metrics/bootstrap → `x-observability`

### 6. Покрой тестами

- unit рядом с кодом
- integration через `x-integration-testing`, если нужен реальный внешний сервис

## Короткий чек-лист

- пакет лежит в правильном месте дерева
- `doc.go` описывает пакет
- `Config` и lifecycle оформлены явно
- конструктор не делает I/O
- тестовый слой выбран осознанно

## References

- `assets/adapter-package/` — прозрачный стартовый каркас нового пакета

## Смежные skills

- `x-doc-go`
- `x-env-config`
- `x-log`
- `x-observability`
- `x-integration-testing`
- `x-unit-test-partial-interface`
