---
name: x-unit-test-partial-interface
description: "Применяй при написании юнит-тестов с внешними зависимостями: частично применяемые интерфейсы, локальные контракты потребителя, callback-зависимости и mock workflow"
compatibility: github.com/stretchr/testify v1+, github.com/vektra/mockery v3+
---

# Частично применяемые интерфейсы

Этот skill — **канонический владелец** unit-dependency паттерна:
- локальные consumer-side interfaces
- выбор между интерфейсом и callback alias
- связь с `mockery`

`x-mockery` не должен повторять эти правила, а должен ссылаться сюда.

## Когда применять

- код зависит от внешнего repo/client/adapter, но тест должен остаться unit
- зависимость естественно выражается callback-функцией, а не интерфейсом

Не применяй для:
- integration-тестов с реальными внешними сервисами
- случаев, где нужен технический контракт SQL/HTTP/AMQP, а не unit isolation

## Core workflow

### 1. Объяви зависимость у потребителя

По умолчанию зависимость оформляется как локальный интерфейс рядом с тестируемым кодом:

```go
type itemRepo interface {
    GetItem(ctx context.Context, id int64) (*entity.Item, error)
}
```

Правила:
- интерфейс объявляется у потребителя, а не у поставщика зависимости
- имя обычно неэкспортируемое
- только реально используемые методы

#### Типы в сигнатуре интерфейса

Частично применяемый интерфейс — это и есть граница между потребителем и поставщиком. Из неё вытекает правило импортов: потребитель не импортирует поставщика по горизонтали (sibling→sibling запрещено — см. раздел «Структура пакетов» в `AGENTS.md`). Значит в сигнатуре интерфейса допустимы только:

- общие пакеты проекта: `internal/domain`, `internal/dto`, `internal/config`
- типы из внешних библиотек (`github.com/...`) — они уже готовый контракт и используются напрямую

Не создавай параллельный тип в `internal/domain/` как «тонкий фасад» над типом из внешней библиотеки. Если `*sdk.Foo` закрывает предметную область — это и есть рабочий контракт, и потребитель, и mockery-мок работают с ним напрямую.

`internal/domain/` существует только для того, чего нет во внешней библиотеке: внутренние state-events, value objects, бизнес-правила.

Типичная ловушка: в scaffolding-фазе (Fake-реализация без реальной либы) создаются domain-дубли «ради удобства мокирования». При подключении реальной либы дубли удаляются вместе с Fake — они стали мёртвым слоем.

### 2. Если интерфейс не нужен, выбери callback alias

Если зависимость выражается функцией, не создавай интерфейс только ради теста.

Используй callback alias, когда:
- API уже принимает функцию
- зависимость слишком маленькая для отдельного интерфейса
- мок через `mockery` только усложнит код

```go
type notify = func(state AuthorizationState)
```

Правила:
- используй alias через `=`, а не новый named type
- имя callback — глагол в нижнем регистре
- callback остаётся локальным, если не нужен снаружи

Тестируй callback обычным замыканием:

```go
var received AuthorizationState
mockNotify := func(state AuthorizationState) {
    received = state
}
```

Если callback строится из зависимостей, оформи `NewFunc...` и мокируй уже эти зависимости, а не сам callback.

### 3. Для интерфейса генерируй mock через `x-mockery`

Если выбрана interface-зависимость:
- оставь локальный consumer-side contract
- сгенерируй mock по `x-mockery`
- не мокируй callback-типы через `mockery`

### 4. Выбери форму теста

| Ситуация | Форма |
|---|---|
| Один сценарий | простой тест |
| Несколько наборов входных данных | табличный тест |
| Сложный setup / verify | call-тест |
| Общий fixture для группы тестов | `suite.Suite` |

Если нужен alias для импорта моков, предпочитай понятное имя вроде `repoMock` или `clientMock`, а не общее `m`.

#### Табличный тест с условным EXPECT

```go
func TestService_GetItem(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        id      int64
        prepare func(repo *mocks.ItemRepo)
        wantErr error
        wantNil bool
        wantID  int64
    }{
        {
            name: "success_case",
            id:   1,
            prepare: func(repo *mocks.ItemRepo) {
                repo.EXPECT().
                    GetItem(mock.Anything, int64(1)).
                    Return(&entity.Item{ID: 1}, nil)
            },
            wantID: 1,
        },
        {
            name: "repo_error",
            id:   2,
            prepare: func(repo *mocks.ItemRepo) {
                repo.EXPECT().
                    GetItem(mock.Anything, int64(2)).
                    Return(nil, assert.AnError)
            },
            wantErr: assert.AnError,
            wantNil: true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            // Arrange
            repo := mocks.NewItemRepo(t)
            tt.prepare(repo)
            service := New(repo)

            // Act
            got, err := service.MethodName(context.Background(), tt.id)

            // Assert
            assert.ErrorIs(t, err, tt.wantErr)
            if tt.wantNil {
                assert.Nil(t, got)
                return
            }
            require.NotNil(t, got)
            assert.Equal(t, tt.wantID, got.ID)
        })
    }
}
```

#### Call-test с моками

```go
func TestService_Create(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name   string
        setup  func(t *testing.T) *Service
        verify func(t *testing.T, got *entity.Item, err error)
    }{
        {
            name: "success_case",
            setup: func(t *testing.T) *Service {
                t.Helper()

                repo := mocks.NewItemRepo(t)
                repo.EXPECT().
                    Save(mock.Anything, mock.Anything).
                    Return(nil)

                return New(repo)
            },
            verify: func(t *testing.T, got *entity.Item, err error) {
                t.Helper()
                require.NoError(t, err)
                require.NotNil(t, got)
            },
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            // Arrange
            service := tt.setup(t)

            // Act
            got, err := service.Create(context.Background())

            // Assert
            tt.verify(t, got, err)
        })
    }
}
```

#### `suite.Suite`

```go
type ServiceSuite struct {
    suite.Suite
    repo *mocks.ItemRepo
    service *Service
}

func (s *ServiceSuite) SetupTest() {
    s.repo = mocks.NewItemRepo(s.T())
    s.service = New(s.repo)
}

func (s *ServiceSuite) TestMethodName() {
    // Arrange
    s.repo.EXPECT().
        GetItem(mock.Anything, int64(1)).
        Return(&entity.Item{ID: 1}, nil)

    // Act
    got, err := s.service.MethodName(context.Background(), 1)

    // Assert
    s.Require().NoError(err)
    s.Require().NotNil(got)
}

func TestServiceSuite(t *testing.T) {
    t.Parallel()
    suite.Run(t, new(ServiceSuite))
}
```

#### Error case

```go
func TestMethodName_error(t *testing.T) {
    t.Parallel()

    // Arrange
    repo := mocks.NewItemRepo(t)
    repo.EXPECT().
        GetItem(mock.Anything, mock.Anything).
        Return(nil, assert.AnError)

    service := New(repo)

    // Act
    got, err := service.MethodName(context.Background(), 1)

    // Assert
    assert.Error(t, err)
    assert.ErrorIs(t, err, assert.AnError)
    assert.Nil(t, got)
}
```

#### `errors.As`

```go
func TestMethodName_domainError(t *testing.T) {
    t.Parallel()

    // Arrange
    repo := mocks.NewItemRepo(t)
    repo.EXPECT().
        GetItem(mock.Anything, mock.Anything).
        Return(nil, &DomainError{Code: "not_found"})

    service := New(repo)

    // Act
    _, err := service.MethodName(context.Background(), 1)

    // Assert
    var domainErr *DomainError
    require.Error(t, err)
    require.ErrorAs(t, err, &domainErr)
    assert.Equal(t, "not_found", domainErr.Code)
}
```

## Короткий чек-лист

- dependency contract объявлен у потребителя
- в интерфейс не попали методы "на всякий случай"
- callback-case не насилуется через `mockery`
- если тест стал тяжёлым, это сигнал к обсуждению архитектуры, а не к молчаливому рефакторингу

## Архитектурный сигнал

Если unit-тест трудно написать без громоздкого setup, большого числа моков или копирования почти всего внешнего контракта, это признак архитектурной проблемы. Обычно это означает, что зависимость слишком крупная или сервис знает о слишком многих деталях.
Если увидел такой сигнал, не рефактори архитектуру молча только ради теста: сначала обсуди это с пользователем и зафиксируй, что именно стоит упростить в контракте.

## Смежные skills

- `x-testing-conventions`
- `x-mockery`
