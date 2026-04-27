---
name: x-mockery
description: "Применяй когда в unit-тесте нужен мок интерфейсной зависимости: объяви частично применяемый интерфейс у потребителя, добавь пакет в корневой `.mockery.yml` и сгенерируй мок через mockery v3 + testify"
compatibility: github.com/vektra/mockery v3+, github.com/stretchr/testify v1+
---

## Смежные скиллы

- `x-unit-test-partial-interface` — как применять consumer-side interfaces и callback dependencies в unit-тестах

## Когда применять

Используй этот скилл, когда зависимость должна мокаться через `testify/mock`.

Этот скилл описывает проектный путь работы с `mockery v3`, а не полный справочник возможностей `mockery`.

Перед запуском этого скилла обязательно проверь, доступен ли skill `context7-cli`.
Это обязательное условие запуска: текущий workflow опирается на возможность быстро проверить актуальную документацию `mockery` и `testify`.

Если `context7-cli` недоступен:
- остановись
- явно сообщи пользователю, что обязательный skill для `x-mockery` не подключён
- не продолжай стандартный сценарий "по памяти"

Если задача выходит за рамки описанного здесь паттерна:
- сначала сохраняй проектные инварианты из этого скилла
- используй `context7-cli` для проверки актуальной документации `mockery` и `testify`

Не используй `mockery` для:
- колбеков (`type alias = func(...)`) — см. `x-unit-test-partial-interface`, раздел про callback alias
- автогенерируемых интерфейсов, которые проект не хранит под контролем вручную

## Объявление интерфейса

Объявляй интерфейс в пакете, который его потребляет. Обычно это частично применяемый локальный интерфейс рядом с тестируемым кодом:

```go
type pointRepo interface {
    GetByID(ctx context.Context, id string) (*domain.Point, error)
    Save(ctx context.Context, point domain.Point) error
}
```

Правила:
- предпочитай минимальный интерфейс под потребителя, а не общий "god interface"
- если интерфейс нужен только внутри пакета, оставляй имя неэкспортируемым
- не добавляй `//go:generate mockery ...` над интерфейсом: для `v3` генерация централизована

## Конфиг mockery v3

Корневой `.mockery.yml` — источник правды для генерации.

Базовый паттерн с областью пакета:

```yaml
filename: "{{.InterfaceName | snakecase}}.go"
dir: "{{.InterfaceDir}}/mocks"
structname: "{{.InterfaceName | firstUpper}}"
pkgname: mocks
template: testify
template-data:
  unroll-variadic: true
packages:
  your/module/internal/service:
    config:
      all: true
```

Это проектный дефолт:

- сгенерированный файл лежит рядом с пакетом интерфейса в `mocks/`
- конструктор создаётся как `New<Type>(t)`
- `EXPECT()` даётся шаблоном `testify`

Правила:
- следуй имени конфиг-файла, принятому в проекте; в текущем паттерне используется `.mockery.yml`
- `template: testify` обязателен: он генерирует `EXPECT()`, конструктор `New<Type>(t)` и типизированные методы для вызовов
- `template-data.unroll-variadic: true` держи включённым, если проект уже использует этот паттерн
- `packages:` перечисляет пакеты, где разрешена генерация моков
- если пакет уже указан с `all: true`, новый интерфейс подхватится автоматически
- если интерфейс добавлен в новый пакет, сначала добавь этот пакет в `.mockery.yml`
- не смешивай этот подход с точечными CLI-флагами на отдельных интерфейсах: источник правды должен оставаться в `.mockery.yml`

### Когда добавлять пакет в `.mockery.yml`

Если интерфейс появился в новом пакете, сначала добавь пакет в `packages:`.
Если пакет уже указан с `all: true`, отдельный интерфейс туда дописывать не нужно.

## Генерация

Для быстрого post-check после генерации можно использовать `scripts/check-mockery-targets.sh`.

Пример вызова:

```bash
bash .agents/skills/x-mockery/scripts/check-mockery-targets.sh internal/service PointRepo .mockery.yml
```

Предпочитай проектную задачу:

```bash
task mock
```

Если в репозитории ещё нет задачи, эквивалентный вызов для `v3`:

```bash
go run github.com/vektra/mockery/v3@latest
```

Результат обычно появляется рядом с пакетом интерфейса:

```text
internal/service/mocks/point_repo.go
```

НЕ РЕДАКТИРУЙ сгенерированные файлы вручную.

## Создание мока в тесте

```go
repo := mocks.NewPointRepo(t)
```

`mockery v3` с шаблоном `testify` регистрирует `t.Cleanup(...)` внутри конструктора, поэтому ожидания проверяются автоматически в конце теста.

## EXPECT() API

Пример ожиданий:

```go
repo.EXPECT().GetByID(mock.Anything, "p1").Return(&domain.Point{ID: "p1"}, nil)
repo.EXPECT().Save(mock.Anything, mock.Anything).Return(assert.AnError)
repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Times(3)
```

Один вызов — дефолтное поведение. `.Once()` избыточен, не пиши его.

Если для контракта теста важна кратность больше одного вызова, указывай её явно через `.Times(n)`.

`.Maybe()` не используй без явного запроса пользователя. В финальном тесте это почти всегда признак размытого контракта.

## RunAndReturn

`RunAndReturn` особенно полезен, когда мок должен вернуть результат, зависящий от аргументов вызова:

```go
repo.EXPECT().
    GetByID(mock.Anything, "p1").
    RunAndReturn(func(ctx context.Context, id string) (*domain.Point, error) {
        return &domain.Point{ID: id}, nil
    })
```

Используй `RunAndReturn`, когда нужно:
- вычислить результат от входных аргументов
- проверить типизированные аргументы без ручной распаковки `mock.Arguments`
- удобно работать с variadic-методами, если включён `unroll-variadic: true`

Для простых кейсов предпочитай обычный `.Return(...)`.

## Проверка после генерации

После `task mock` проверь:
- мок появился в `mocks/` рядом с пакетом интерфейса
- имя файла в `snake_case`, имя структуры в `PascalCase`
- конструктор `New<Type>(t)` и метод `EXPECT()` сгенерировались
- тесты компилируются без ручных правок сгенерированного файла
- пакет присутствует в `.mockery.yml`
- сгенерированный файл не редактируется вручную

## Не делай

- не объявляй интерфейс в пакете поставщика зависимости только ради теста потребителя
- не используй `.Once()` для одиночного вызова
- не редактируй сгенерированные `mocks/*.go` вручную
- не держи параллельно несколько источников правды для генерации (`.mockery.yml` и точечные `//go:generate`)
- не пиши inline mock-структуры в тестах; если пакет ещё не в `.mockery.yml`, сначала добавь его, сгенерируй моки, затем используй `mocks.NewX(t)` + `EXPECT()` — inline-моки обходят auto-cleanup и создают неконсистентность
- не используй `.On("Method", ...)` напрямую; используй `.EXPECT().Method(...)` — это type-safe API, проектный стандарт

## Полезные ресурсы

- `scripts/check-mockery-targets.sh` — проверка `.mockery.yml`, `mocks/*.go`, `EXPECT()` и `New<Type>(t)` после генерации
