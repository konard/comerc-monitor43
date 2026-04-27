---
name: "x-testing-conventions"
description: "Применяй при написании или правке любого теста в проекте: выбор слоя (`unit`/`integration`/`bdd`/`e2e`/`smoke`), `t.Parallel()`, `testing.Short()`, AAA и базовая структура проверки"
---
# Testing Conventions

Этот skill — **канонический владелец** общих правил тестового слоя:
- layer selection и физическое расположение теста
- маркеры слоя (`t.Parallel()`, `testing.Short()`, build tags)
- AAA-структура
- базовые требования к именованию и cleanup

Другие testing-skills не должны переописывать эти правила, а должны ссылаться сюда.

## Когда применять

- пишешь новый тест
- переносишь тест между слоями
- ревьюишь структуру существующего теста

## Core workflow

### 1. Сначала определи слой теста

| Слой | Где живёт | Маркер | Команда запуска |
|---|---|---|---|
| unit | `*_test.go` рядом с кодом | `t.Parallel()` по умолчанию | `task test-short` |
| integration | `test/integration/` | `if testing.Short() { t.Skip(...) }` | `task test` |
| bdd | `test/bdd/steps/` | `if testing.Short() { t.Skip(...) }` | `task test` или `task bdd` |
| e2e | `test/e2e/` | `//go:build e2e` | `task test-e2e` |
| smoke | `test/smoke/` | `//go:build smoke` | `task test-smoke` |

Если тест проверяет систему на языке бизнеса, смотри BDD-skills. Если проверка техническая и использует реальные внешние зависимости, это integration/e2e/smoke, а не unit.

#### Как запускать локально

- `task test-short` — быстрая петля разработки. Прогоняет только unit; integration/bdd скипаются через `testing.Short()`, e2e/smoke отрезаются build tags. Используй по умолчанию при работе над кодом.
- `task test` — полный прогон с integration и bdd. Выполняется долго, требует docker/testcontainers. Запускай перед коммитом или когда менял адаптеры, репозитории или BDD-сценарии.
- `task test-smoke`, `task test-e2e`, `task bdd` — точечный запуск конкретного слоя через build tags.

Маркер слоя должен соответствовать команде: если unit-тест случайно зависит от реального контейнера, `task test-short` начнёт требовать внешние сервисы, и быстрая петля сломается.

### 2. Поставь правильный маркер слоя

- unit-тест обычно начинает с `t.Parallel()`
- если unit-тест меняет process-wide state, `t.Parallel()` не ставь
- integration и bdd маркируются через `testing.Short()`, а не через `t.Parallel()`
- e2e и smoke маркируются build tags

#### Когда unit-тесту не нужен `t.Parallel()`

`t.Parallel()` остаётся нормой для unit-тестов, пока тест изолирован.
Если тест меняет process-wide состояние или делит mutable fixture между несколькими сценариями, параллельность превращается из ускорения в источник паник, race conditions и флейков.

Для integration, bdd, e2e и smoke `t.Parallel()` не является нормой слоя и обычно не нужен.

**Быстрая проверка**

Перед `t.Parallel()` в unit-тесте спроси себя:

- меняет ли тест env процесса
- меняет ли тест текущую рабочую директорию
- трогает ли тест global OpenTelemetry singleton'ы
- делят ли parent test и subtests один и тот же mutable state

Если ответ "да" хотя бы на один пункт, запускай unit-тест последовательно или сначала изолируй fixture.

**1. Изменение env процесса**

`t.Setenv()` несовместим с `t.Parallel()`.
`os.Setenv()` и `os.Unsetenv()` формально не вызывают ту же панику, но меняют то же самое глобальное состояние процесса и создают те же гонки.

```go
func TestConfig(t *testing.T) {
    // без t.Parallel()

    // Arrange
    t.Setenv("APP_PORT", "8080")

    // Act
    cfg, err := LoadConfig()

    // Assert
    require.NoError(t, err)
    assert.Equal(t, 8080, cfg.Port)
}
```

**2. Изменение cwd и загрузка `.env`**

`os.Chdir()` тоже меняет состояние всего процесса.
Если тест проверяет загрузку `.env` из текущей директории, он часто одновременно меняет cwd и косвенно загрязняет env уже загруженными переменными.

Такие тесты:
- не запускай параллельно
- делай явный cleanup cwd и env
- по возможности держи их отдельно от обычных parsing/defaults тестов

**3. Global OpenTelemetry state**

Глобальные setter'ы OpenTelemetry влияют на весь процесс:
- `otel.SetTracerProvider`
- `otel.SetMeterProvider`
- `otel.SetTextMapPropagator`

Если helper вроде `SetupMonitoring` или `monitoring.InitDefault` внутри вызывает такие setter'ы, рассматривай его точно так же: тест должен быть последовательным.

```go
func TestSetupMonitoring(t *testing.T) {
    // без t.Parallel()

    // Arrange
    cleanup := SetupMonitoring(...)
    t.Cleanup(func() { _ = cleanup() })

    // Act
    // ...

    // Assert
    // ...
}
```

**4. Shared mutable state в parent/subtests**

Проблема не в самом `t.Run`, а в разделяемом mutable fixture.
Если parent test создаёт slice, logger, mock, context wrapper или структуру, которую потом меняют несколько parallel subtests, это риск гонки.

Плохо:

```go
func TestLogger(t *testing.T) {
    t.Parallel()

    records := []string{}

    t.Run("a", func(t *testing.T) {
        t.Parallel()
        records = append(records, "a")
    })
}
```

Лучше:

```go
func TestLogger(t *testing.T) {
    t.Run("a", func(t *testing.T) {
        t.Parallel()

        // Arrange
        records := []string{}

        // Act
        records = append(records, "a")

        // Assert
        assert.Equal(t, []string{"a"}, records)
    })
}
```

Правило:
- immutable table cases делить можно
- mutable fixture должен принадлежать конкретному subtest
- если изоляция дорогая или невозможна, убери `t.Parallel()`

**5. Граница применения**

- unit-тесты обычно начинают с `t.Parallel()`, если fixture изолирован
- integration / bdd тесты не нужно "ускорять" через `t.Parallel()`; их маркер слоя другой
- e2e / smoke тесты тоже не должны опираться на `t.Parallel()` как на соглашение

**6. Что делать вместо слепого запрета**

- оставь `t.Parallel()` только там, где тест реально изолирован
- разнеси env/cwd тесты отдельно от обычных unit-кейсов
- перенеси mutable fixture внутрь каждого `t.Run`
- для time/concurrency логики используй `synctest` (см. раздел «Специальные техники»), но не путай это с безопасностью process-wide state

### 3. Держи тест в форме AAA

Каждый тест и каждый `t.Run` содержит:
- `// Arrange`
- `// Act`
- `// Assert`

AAA — обязательный project convention.

#### Самопроверка теста

Перед завершением теста спроси себя:

- выбран ли правильный слой тестирования
- есть ли явные `// Arrange`, `// Act`, `// Assert`
- не меняет ли тест process-wide state через env, cwd или global OpenTelemetry
- не маскируется ли проблема через `t.Skip()`
- вынесен ли cleanup в `t.Cleanup`

#### Call-test / setup-verify форма

Большие unit-тесты со многими сценариями часто используют call-test стиль:

- `setup func(t *testing.T) *Service`
- `verify func(t *testing.T, got Result, err error)`

Это полезно, когда:

- сценариев много
- Arrange очень разный
- табличный тест с inline logic уже нечитаем

Минимальный каркас:

```go
func TestService_Do(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name   string
        setup  func(t *testing.T) *Service
        verify func(t *testing.T, got Result, err error)
    }{
        {
            name: "success_case",
            setup: func(t *testing.T) *Service {
                t.Helper()
                return New()
            },
            verify: func(t *testing.T, got Result, err error) {
                t.Helper()
                require.NoError(t, err)
                assert.Equal(t, expected, got)
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            // Arrange
            service := tt.setup(t)

            // Act
            got, err := service.Do(context.Background())

            // Assert
            tt.verify(t, got, err)
        })
    }
}
```

#### Типовые ошибки

- AAA есть только в одном примере скилла, а в остальных примерах исчезает
- integration-тест лежит рядом с unit-кодом без явного маркера слоя
- табличный тест уходит в inline-логику и перестаёт читаться
- `verify` начинает сам заново делать Arrange вместо проверки результата

### 4. Соблюдай минимальные naming/cleanup правила

- имена кейсов в таблицах — `snake_case` или lowercase-with-hyphen
- `t.Cleanup(...)` предпочитай `defer` для test-owned ресурсов
- integration/bdd/e2e/smoke не используют `t.Parallel()` как маркер слоя
- `t.Skip()` не используется для сокрытия падающего теста; допустим только слой-маркер через `testing.Short()`
- для проверок используй `require`/`assert` из testify, а не ручные `if ... { t.Fatalf(...) }`: `require.NoError(t, err)`, `require.Equal(t, want, got)` и т.д.

### 5. Для специальных техник подключай только нужное

- unit dependencies, local interfaces, callback-style зависимости → `x-unit-test-partial-interface`
- testcontainers/shared setup → `x-integration-testing`
- BDD lifecycle `red -> green -> blue` → `x-bdd-dev-workflow`

#### Специальные техники тестирования

##### Snapshot

Используй snapshot только когда значение:
- слишком громоздкое для ручного `assert.Equal`
- стабильно по форме
- удобно ревьюить по diff

Практические правила:
- снапшоты коммить в git
- нестабильные поля нормализуй до снапшота
- не подменяй снапшотом точечные assertions на важные поля

##### `synctest`

Используй `synctest`, когда код зависит от времени, таймеров или горутин.

Практические правила:
- оставляй внешний защитный timeout
- всё time/concurrency-sensitive создавай внутри `synctest.Test`
- после действия вызывай `synctest.Wait()`
- не передавай каналы и таймеры, созданные вне sandbox, внутрь `synctest.Test`

### 6. Smoke-тесты для тонкого main

Smoke проверяет, что собранный бинарь стартует и корректно останавливается. Не проверяет бизнес-логику.

**Что проверяет smoke:**
- `run()` не паникует при валидном конфиге
- граф зависимостей собирается
- healthcheck/liveness эндпоинты отвечают 200
- graceful shutdown завершается без ошибок

**Чего smoke НЕ проверяет:**
- бизнес-сценарии (→ bdd)
- контракты адаптеров (→ integration)
- корректность алгоритмов (→ unit)

**Маркер слоя:** `//go:build smoke`

**Паттерн тонкого main:**

```go
func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}
```

Вся сборка графа и lifecycle — в `run()`. Smoke-тест поднимает бинарь (или docker-compose стек) и проверяет liveness.

**Инфраструктура:** `test/smoke/`, docker-compose с зависимостями, testcontainers-go для оркестрации.

## Inline-first: без переиспользуемых helper-функций

Тест читается сверху вниз без прыжков в внешние функции. Дублирование Arrange между кейсами допустимо, если альтернатива — helper, который заставляет читателя открывать другой файл или функцию.

### Граница inline и абстракции

- **inline** — function literal живёт внутри одного теста: тело `t.Run`, поле `tests[]`, замыкание `setup`/`verify` конкретного кейса. Это часть AAA данного теста, не внешняя абстракция.
- **запрещённая абстракция** — именованная функция (`newTestService`, `assertUserEqual`, `setupDB` и т.п.), которую вызывают из нескольких тестов или из другого файла.

Пример inline, который правилом **не** запрещён:

```go
tests := []struct {
    name   string
    setup  func(t *testing.T) *Service
    verify func(t *testing.T, got Result, err error)
}{
    {
        name: "success_case",
        setup: func(t *testing.T) *Service {
            t.Helper()
            return New()
        },
        // ...
    },
}
```

`setup` и `verify` здесь — inline literals конкретного кейса, а не переиспользуемая между тестами функция.

### Исключения

Helper оправдан только когда выносимая логика не относится к проверяемому поведению:
- cleanup инфраструктуры (`t.Cleanup` над контейнером, закрытие соединения)
- оркестрация testcontainers и shared setup в `test/support/`
- установка process-wide state, который всё равно должен быть одинаков во всех тестах файла

В таких случаях helper должен иметь очевидное имя и не содержать assertions.

### Типовые ошибки

- `newTestFoo(t)`, который собирает объект с моками и вызывается из 5 тестов — читатель вынужден реконструировать Arrange мысленно
- `assertFooEqual(t, got, want)`, подменяющий `require.Equal` — скрывает, какие поля реально важны
- общий `setupDB(t)` в unit-пакете, вместо явной inline сборки фикстуры

## Короткий чек-лист

- выбран правильный слой теста
- маркер слоя соответствует слою
- AAA присутствует
- `t.Parallel()` не конфликтует с process-wide state
- cleanup оформлен явно
- нет переиспользуемых между тестами helper-функций

## Смежные skills

- `x-integration-testing`
- `x-bdd-godog`
- `x-bdd-dev-workflow`
- `x-unit-test-partial-interface`
- `x-test-matrix`
