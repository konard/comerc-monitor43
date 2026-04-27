---
name: "x-bdd-godog"
description: "Применяй при написании бизнес-тестов через godog: структура feature-файлов по каноническому Gherkin, нумерация сценариев, step definitions, связь с testcontainers через test/support"
compatibility: github.com/cucumber/godog, github.com/stretchr/testify v1+, github.com/testcontainers/testcontainers-go
---
# BDD на godog

Этот skill — **канонический владелец** правил BDD-слоя про:
- layout `test/bdd/**`
- именование epic/story/scenario
- wiring `godog.TestSuite`, `TestMain`, `State`
- размещение и регистрацию step definitions

`x-bdd-product-workflow`, `x-bdd-dev-workflow` и `bdd-reviewer` не должны переописывать эти правила, а должны ссылаться сюда.

## Когда применять

- Создаёшь или меняешь `.feature` в `test/bdd/NN_epic/`.
- Подключаешь или перестраиваешь `godog`-слой в `test/bdd/suite/`.
- Проверяешь, соответствует ли BDD-слой проектному layout.

Не применяй для:
- продуктового пути от PRD до готового `.feature` → `x-bdd-product-workflow`
- RGB-реализации готового `.feature` → `x-bdd-dev-workflow`
- извлечения сценариев из legacy → `x-bdd-knowledge-harvest`

## Core workflow

### 1. Подтверди, что задача относится к BDD-слою

BDD здесь описывает поведение, видимое аналитику или заказчику. Если проверка формулируется на языке системы, а не бизнеса, это другой слой тестирования.

### 2. Собери файловую структуру по проектному layout

#### Физическая структура

```text
{project-root}/
└── test/
    └── bdd/
        ├── suite/
        │   ├── state.go
        │   ├── stack.go
        │   ├── prefix.go
        │   ├── steps_common.go
        │   ├── helpers.go
        │   ├── runner.go
        │   └── steps_NN_<name>.go   ← по одному на эпик
        ├── 01_delivery/
        │   ├── bdd_test.go          ← package delivery_test
        │   ├── 01_copy.feature
        │   └── 02_forward.feature
        └── 02_billing/
            ├── bdd_test.go
            └── 01_subscription.feature
```

`.feature` файлы живут **рядом с `bdd_test.go`** своего эпика, а не в отдельном `features/` поддереве. godog ищет только `*.feature`, `.go` файлы в той же директории он игнорирует.

#### Иерархия сущностей

| Уровень | Где живёт | Что означает |
|---|---|---|
| Epic | директория `NN_name/` | крупное продуктовое направление |
| Feature | файл `NN_name.feature` | одна user story |
| Scenario | `Scenario: NN_slug` | одна бизнес-проверка |

#### Naming rules

- Директория эпика: `NN_name`
- Файл feature: `NN_name.feature`
- Номер файла задаёт порядок story внутри эпика и не дублирует номер директории
- Файл шагов эпика: `suite/steps_NN_<name>.go` (с номером, совпадающим с номером директории)
- Имя сценария: `NN_slug` или `NNA_slug`
- Смысловая часть имени после номера пишется на английском в `snake_case`
- Формат совпадает с соглашением для имён кейсов в табличных тестах и удобен для поиска в выводе `go test`
- Регекс сценария: `^\d{2}[A-Z]?_[a-z0-9_]+$`

#### Numbering rules

- Базовые happy-path и основные use cases используют `01`, `02`, `03`
- Варианты того же кейса используют `01A`, `01B`, `02A`
- Уникальность номера обязательна только в рамках файла
- Пропуски допустимы: не перенумеровывай файл после удаления сценария
- Внешняя ссылка строится как `эпик/файл#номер`, например `01_delivery/01_copy.feature#02A`

#### Минимальный feature template

```gherkin
Feature: Вход пользователя

  Как зарегистрированный пользователь
  Я хочу войти по email и паролю
  Чтобы получить доступ к своему кабинету

  Scenario: 01_successful_login_with_valid_credentials
    Given зарегистрирован пользователь "user@example.com"
    When пользователь отправляет запрос на вход с email "user@example.com" и паролем "correct"
    Then пользователь получает доступ к личному кабинету
```

#### Антипаттерны

- `.feature` вне директории своего эпика (`test/bdd/NN_epic/`)
- файл вида `01_01_login.feature` внутри `01_auth/`
- несколько `Feature:` в одном файле
- отсутствие блока `Как / Я хочу / Чтобы`
- duplicate `Scenario: 01_...` в одном файле
- технические детали в шагах вместо бизнес-языка

### 3. Подключи `godog` без локальной самодеятельности

Раскладка BDD-кода: **по одному test-пакету на эпик** плюс общий helper-пакет `suite`.

```
test/bdd/
  suite/                  ← package suite; обычные .go файлы, не _test.go
    state.go              ← type State, методы Reset, ApplyRuleSet
    stack.go              ← GetOrCreateStack, liveStack (package var)
    prefix.go             ← GeneratePrefix, scenarioSeq
    steps_common.go       ← RegisterCommonSteps, RegisterAllSteps
    helpers.go            ← TextContent, MessageCaption, HasTMeEntity, …
    runner.go             ← RunEpic, featurePaths
    steps_NN_<name>.go    ← Register<Name>Steps (по одному на эпик)
  01_delivery/
    bdd_test.go           ← package delivery_test; одна Test01Delivery функция
    *.feature             ← feature-файлы эпика
  02_filters/
    bdd_test.go
    *.feature
  …
  06_auto/
    bdd_test.go
    *.feature
```

- `bdd_test.go` в каждом пакете — тонкий runner: одна `Test<Name>` функция, которая вызывает `suite.RunEpic`.
- Все `Register<Name>Steps` живут в `suite/steps_NN_*.go` и регистрируются вместе, потому что feature-файлы разных эпиков могут переиспользовать шаги друг друга (например, `03_transform` использует «пользователь отправляет сообщение с текстом» из `02_filters`). `_test.go` пакеты в Go не импортируемы, поэтому шаги нельзя перенести в per-epic пакеты.
- `State`, `GetOrCreateStack`, `GeneratePrefix`, common Given-шаги живут в `suite` и экспортированы; приватные хелперы конкретного эпика остаются файл-local в `suite/steps_NN_<name>.go`.

**Зачем per-package:** `go test` кэширует по пакету. Один общий пакет = одна запись кэша — правка в любом step-файле инвалидирует все эпики. Шесть пакетов = шесть записей: правка в `05_sync` не трогает кэш `01_delivery..04_media,06_auto`. Дополнительно даёт clearer output и точечный re-run через `-run Test<Name> ./test/bdd/<NN_name>/...`.

**Ограничения:**
- Шесть test-бинарников = шесть TDLib-логинов (но сессия кэширована в БД, warm-up ~5–10с каждый).
- Одна TDLib-сессия → параллельный запуск пакетов конфликтует на session-lock. `task bdd` / `task cover` должны использовать `-p 1`.
- `scenarioSeq` per-binary: каждый пакет начинает с 0. Коллизии префиксов между пакетами отсекает `isFresh` (по `msg.Date`) в `LiveStack`.

#### Минимальный `bdd_test.go`

Весь runner (chdir, `godog.TestSuite`, initScenario, `BDD_PATHS` override) живёт в
`suite.RunEpic`. В per-epic пакете остаётся только имя эпика:

```go
package delivery_test

import (
	"testing"

	"github.com/pure-golang/budva-claude/test/bdd/suite"
)

func Test(t *testing.T) { suite.RunEpic(t) }
```

`TestMain` не нужен: `suite.RunEpic` сам делает chdir через `sync.Once`
и определяет имя эпика из пути вызывающего файла через `runtime.Caller(1)`.

**`BDD_PATHS` для дебага:** указывает на конкретную директорию или `.feature`-файл (несколько через запятую), нужен только при локальном прогоне; ответственность за сочетание c правильным `-run` и целевой директорией пакета — на пользователе:
```bash
BDD_PATHS=test/bdd/01_delivery/01_copy.feature \
  go test -run Test01Delivery ./test/bdd/01_delivery/...
```

#### Bootstrap через `TestMain`

```go
func TestMain(m *testing.M) {
	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()
	pg = support.StartPostgresBG(ctx)
	defer pg.Stop(ctx)

	amqp = support.StartRabbitMQBG(ctx)
	defer amqp.Stop(ctx)

	os.Exit(m.Run())
}
```

Не поднимай testcontainers прямо в step-файлах. Shared setup принадлежит `test/support`.

#### `State`

```go
type State struct {
	client   *httpclient.Client
	lastResp *http.Response
	lastErr  error
}

func (s *State) Reset() {
	s.lastResp = nil
	s.lastErr = nil
}

func initScenario(ctx *godog.ScenarioContext) {
	s := &State{client: httpclient.New(apiURL)}

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		s.Reset()
		return ctx, nil
	})

	register01AuthSteps(ctx, s)
}
```

#### Регистрация шагов

- Один файл шагов на эпик: `suite/steps_01_auth.go`, `suite/steps_02_billing.go`
- Методы шагов живут на `State`
- Текст шагов остаётся на бизнес-языке
- Функции шагов возвращают только `error`

```go
func register01AuthSteps(ctx *godog.ScenarioContext, s *State) {
	ctx.Step(`^зарегистрирован пользователь "([^"]*)"$`, s.userRegistered)
	ctx.Step(`^пользователь получает доступ к личному кабинету$`, s.userHasAccess)
}
```

#### Полезные команды

```bash
go test -count=1 -p 1 ./test/bdd/01_delivery/...
go test -count=1 -p 1 ./test/bdd/01_delivery/... -godog.name="^01[A-Z]?_message_is_forwarded"
task bdd
task bdd-pending
```

### 4. Держи границы ответственности

- layout, naming, numbering и wiring принадлежат этому skill
- продуктовые вопросы и состав сценариев принадлежат `x-bdd-product-workflow`
- порядок red → green → blue принадлежит `x-bdd-dev-workflow`

Если правило уже описано в другом skill как его owner-area, здесь оставляй только ссылку.

## Короткий чек-лист

- `.feature` лежит в `test/bdd/NN_epic/NN_story.feature`
- в файле ровно один `Feature:` и user story блок
- `Scenario:` имеют стабильные имена `NN_slug` или `NNA_slug` на английском
- `godog.TestSuite` использует `Strict: true`
- `State.Reset()` вызывается в `Before`
- testcontainers bootstrap не размазан по step-файлам

## Смежные skills

- `x-bdd-product-workflow`
- `x-bdd-dev-workflow`
- `x-bdd-knowledge-harvest`
- `x-integration-testing`
- `x-testing-conventions`
