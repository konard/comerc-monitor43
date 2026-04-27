---
name: "x-tdlib"
description: "Применяй когда работаешь с TDLib-адаптером: `clientAdapter`, `telegramRepo` (consumer-side), тестовая стратегия, авторизация и lifecycle"
---

# TDLib-адаптер

Этот skill — канонический владелец контракта TDLib-адаптера и тестовой стратегии взаимодействия с Telegram.

## Когда применять

- добавляешь или меняешь метод в `internal/repo/telegram/`
- работаешь с consumer-side интерфейсом `telegramRepo`
- подключаешь реальный TDLib или меняешь тестовую стратегию
- затрагиваешь авторизационный flow

Не применяй для:
- бизнес-логики handler/service (там свои skills)
- общих тестовых соглашений (→ `x-testing-conventions`)
- общих правил частично применяемых интерфейсов и типов в их сигнатурах (→ `x-unit-test-partial-interface`)

## Архитектура адаптера

### Структура пакета `internal/repo/telegram/`

| Файл | Назначение |
|---|---|
| `doc.go` | Контракт пакета, env-переменные |
| `client_adapter.go` | Интерфейс `clientAdapter` + тонкие обёртки `*Repo` с raw-TDLib сигнатурами |
| `repo.go` | `Repo` struct, lifecycle (`New`, `Start`, `Close`), авторизационный flow, каналы обновлений, композитные операции без аналога в go-tdlib |

Разделение простое и строгое:

- **что есть в go-tdlib** — ложится в `client_adapter.go` как обёртка 1-в-1
- **чего нет в go-tdlib** — живёт в `repo.go` (lifecycle, auth submit, `SendMessageAndWait` и подобные композиты)

### `clientAdapter` — внутренний интерфейс-обёртка над `*client.Client`

`clientAdapter` — это внутренний интерфейс внутри самого `repo/telegram/`-пакета, скрывающий конкретный SDK go-tdlib от остальной части `Repo`.

Не путать с:

- **adapter-пакетом из `x-new-adapter`** — там речь о *пакете* целиком (`doc.go`, `Config`, lifecycle, observability) как единице ownership внешней системы. `clientAdapter` — это одна деталь внутри такого пакета, не самостоятельная единица.
- **частично применяемым интерфейсом из `x-unit-test-partial-interface`** — тот паттерн про consumer-side контракт *между модулями проекта*. `clientAdapter` живёт *внутри одного модуля* и абстрагирует внешний SDK, а не другой внутренний модуль.

Общее между этими техниками только слово «adapter»/«интерфейс» и правило «только реально используемые методы». Ownership и контекст — разные.

Содержимое `clientAdapter`:

- **только методы, которые реально использует `Repo`**, а не полная поверхность `*client.Client`
- это не копия go-tdlib и не обёртка «на будущее»; `*client.Client` выставляет сотни методов, большая их часть проекту не нужна
- метод добавляется в `clientAdapter` только когда в `repo.go` появился живой потребитель этого метода; убирается — когда потребитель исчез

Сигнатуры 1-в-1 с go-tdlib, без контекста и без domain-типов:

```go
type clientAdapter interface {
    SendMessage(*client.SendMessageRequest) (*client.Message, error)
    ForwardMessages(*client.ForwardMessagesRequest) (*client.Messages, error)
    GetMessage(*client.GetMessageRequest) (*client.Message, error)
    // ...только реально используемые методы
    GetListener() *client.Listener
}
```

Обёртки живут на `*Repo` — тонкая пересылка в `r.tdClient.X(req)`:

```go
func (r *Repo) SendMessage(req *client.SendMessageRequest) (*client.Message, error) {
    return r.tdClient.SendMessage(req)
}
```

Статическая проверка:

```go
var _ clientAdapter = (*Repo)(nil)
```

Роль `clientAdapter`:

- фиксирует, что `Repo` выставляет наружу именно TDLib-поверхность, а не параллельную domain-обёртку
- даёт точку мокирования TDLib в юнит-тестах самого `Repo` (редкий случай — обычно мокают уровнем выше, через consumer-side `telegramRepo`)
- в обёртках не делается никакого mapping, логирования запроса, переименования полей — это мёртвый код

Статические пакетные функции go-tdlib (`client.ParseTextEntities`, `client.GetMarkdownText`, `client.GetOption`) не являются методами `*client.Client` и в `clientAdapter` не входят. Они вызываются из `repo.go` напрямую.

### Consumer-side `telegramRepo`

Потребители (handler, facade, transform, auth, term) объявляют частично применяемый интерфейс `telegramRepo` только с нужными им методами. В интерфейсе равноправно сосуществуют два класса методов, потому что оба они принадлежат поверхности `*Repo`:

1. **Обёртки `clientAdapter`** — тонкие делегаты к go-tdlib (`SendMessage`, `ForwardMessages`, `GetMessage`, …). Сигнатуры — raw-TDLib типы.
2. **Собственные методы `Repo`** — то, чего нет в go-tdlib:
   - композиты (`SendMessageAndWait` — `SendMessage` + ожидание `UpdateMessageSendSucceeded`)
   - lifecycle-каналы (`AuthStates() <-chan domain.AuthStateEvent`, `Updates() <-chan client.Type`, `ClientDone() <-chan struct{}`)
   - auth submits (`SubmitPhone`, `SubmitCode`, `SubmitPassword`)
   - управление сессией (`LogOut`, `CleanUp`)

Пример:

```go
// в handler/handler.go
type telegramRepo interface {
    // обёртки clientAdapter
    SendMessage(req *client.SendMessageRequest) (*client.Message, error)
    ForwardMessages(req *client.ForwardMessagesRequest) (*client.Messages, error)

    // собственный метод Repo (композит)
    SendMessageAndWait(ctx context.Context, req *client.SendMessageRequest) (*client.Message, error)
    // ...только методы, реально используемые handler-ом
}
```

Потребителю не важно, какой класс у метода — он просто тащит то, что ему нужно. Разделение существует только внутри пакета `repo/telegram/` (см. «Структура пакета»): что есть в go-tdlib → `client_adapter.go`, чего нет → `repo.go`.

Правила про импорты и типы в сигнатуре частично применяемого интерфейса — в `x-unit-test-partial-interface`. Кратко:

- `*client.Message`, `*client.SendMessageRequest`, `*client.FormattedText` и прочие TDLib-типы — это и есть рабочий контракт
- параллельный `internal/domain/Message` / `internal/domain/InputMessageContent` / `internal/domain/FormattedText` как «тонкий фасад» над TDLib — запрещён
- `internal/domain/` для Telegram оставляет только то, чего в go-tdlib нет: `AuthStateEvent`/`AuthState*`/`WaitPasswordState` (внутренний контракт с `service.auth`) и утилиты вроде `MaskPhoneNumber`

Соглашение по именованию:
- интерфейс: `telegramRepo` (не `telegramGateway`, не `telegramClient`)
- поле/параметр: `telegramRepo telegramRepo` (имя = тип)

## Композитные операции

Операции, которых нет в go-tdlib (требуют комбинации нескольких TDLib-вызовов или ожидания асинхронного update), живут в `repo.go`, а не в `client_adapter.go`.

Пример — `SendMessageAndWait`: `SendMessage` возвращает temporary ID, а permanent ID приходит асинхронно через `UpdateMessageSendSucceeded`. Метод принимает и возвращает TDLib-типы, но внутри делает композит (отправка + listener + таймаут):

```go
func (r *Repo) SendMessageAndWait(
    ctx context.Context,
    req *client.SendMessageRequest,
) (*client.Message, error) { ... }
```

`ctx` здесь уместен: он реально используется (`<-ctx.Done()` при ожидании update). В тонких обёртках `client_adapter.go` контекста нет, потому что TDLib синхронный и ctx не использует.

## Тестовая стратегия

### Unit-тесты

Через mockery-моки consumer-side интерфейса `telegramRepo`:

```go
tg := mocks.NewTelegramRepo(t)
tg.EXPECT().
    SendMessage(mock.MatchedBy(func(r *client.SendMessageRequest) bool {
        return r.ChatId == chatID
    })).
    Return(&client.Message{Id: 42}, nil)
```

Unit-тесты **не зависят** от TDLib. Они проверяют логику компонентов-потребителей, работая напрямую с TDLib-типами.

### BDD-тесты

Работают через живой TDLib с тестовым DC. BDD проверяет бизнес-сценарии на реальном Telegram — форвардинг, фильтрация, трансформация и т.д.

Нюансы поведения TDLib (асинхронность, различия типов чатов, rate limits) проявляются естественно в BDD-тестах и при ручном тестировании. Отдельный слой adapter-тестов для TDLib **не нужен** — это тестирование чужого продукта.

### Fake-реализации TDLib

Fake-реализации TDLib (эмуляция `*client.Client` без живой либы) — это scaffolding раннего этапа. Когда реальный TDLib подключён, Fake удаляется. Он не поддерживается как долгосрочный test double: эмуляция нюансов TDLib создаёт ложное чувство безопасности и требует бесконечной синхронизации с реальным поведением.

Вместе с Fake удаляется всё, что он провоцировал создать — в первую очередь параллельные `domain.*` типы, дублирующие `*client.*` (см. `x-unit-test-partial-interface`).

## Авторизация

Авторизационный flow живёт в `repo.go`, потому что управляет состоянием канала `clientDone` у `Repo`. Flow реализован как state machine: `WaitPhone → WaitCode → WaitPassword → Ready`.

Контракт между TDLib-адаптером и UI авторизации (терминал, бот, веб) выражается через `domain.AuthStateEvent` и `domain.AuthState*` — это тот случай, когда `internal/domain/` оправдан: в go-tdlib нет такого объединённого события (у него внутренний `AuthorizationState` с 12+ вариантами, из которых UI релевантны только 5).

`SubmitPhone`/`SubmitCode`/`SubmitPassword` в `repo.go` передают ввод в authorizer go-tdlib через каналы.

`LogOut` и `CleanUp` — тоже в `repo.go`: `LogOut` комбинирует `client.LogOut()` + сброс внутреннего состояния `Repo`; `CleanUp` удаляет локальные файлы TDLib (нет аналога в go-tdlib).

## Короткий чек-лист

- что есть в go-tdlib → обёртка в `client_adapter.go`
- чего нет в go-tdlib → `repo.go`
- `clientAdapter` содержит только реально используемые методы go-tdlib, а не полную поверхность `*client.Client`
- обёртки в `client_adapter.go` — только пересылка, никакого mapping
- consumer-side `telegramRepo` использует raw-TDLib типы, domain-дубли запрещены
- имя поля/параметра совпадает с именем типа (`telegramRepo telegramRepo`)
- unit-тесты используют mockery-моки поверх TDLib-сигнатур
- BDD работает через живой TDLib
- нюансы TDLib не эмулируются в тестах

## Смежные skills

- `x-unit-test-partial-interface`
- `x-testing-conventions`
- `x-mockery`
- `x-doc-go`
- `x-env-config`
