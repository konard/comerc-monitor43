---
name: "x-integration-testing"
description: "Применяй когда тест требует внешнего сервиса (БД, брокер, хранилище): testcontainers-go, setup/teardown контейнера, context timeout"
compatibility: github.com/testcontainers/testcontainers-go, github.com/stretchr/testify v1+
---
# Интеграционные тесты

## Паттерн testcontainers-go Suite

`SetupSuite` / `TearDownSuite` для запуска контейнера:

```go
func (s *MySuite) SetupSuite() {
    ctx := context.Background()
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:15-alpine",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_USER":     "postgres",
                "POSTGRES_PASSWORD": "secret",
                "POSTGRES_DB":       "testdb",
            },
            WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
            AutoRemove: true,
        },
        Started: true,
    })
    s.Require().NoError(err)
    s.container = container

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5432")
    s.dsn = fmt.Sprintf("postgres://postgres:secret@%s:%s/testdb?sslmode=disable", host, port.Port())
}

func (s *MySuite) TearDownSuite() {
    if s.container != nil {
        s.Require().NoError(s.container.Terminate(context.Background()))
    }
}
```

## Таймаут контекста в интеграционных тестах
Используй `context.WithTimeout` вместо `context.Background()`:
```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
```

## Стратегии ожидания готовности
```go
// Ждать сообщение в логе (не требует exec — работает быстро)
wait.ForLog("ready to accept connections").WithOccurrence(2)

// Ждать открытия порта (использует exec внутри контейнера — на Mac медленно)
wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute)
```

## Переиспользование setup через `test/support/`

Когда одни и те же контейнеры нужны в нескольких слоях (`test/integration`, `test/bdd/steps`, `test/e2e`), вынеси их запуск в **экспортируемый пакет** `test/support/`:

```go
// test/support/postgres.go (package support, НЕ _test.go)
package support

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Postgres struct {
	DSN       string
	container testcontainers.Container
}

// StartPostgres поднимает postgres контейнер и регистрирует tb.Cleanup для остановки.
func StartPostgres(tb testing.TB) *Postgres {
	tb.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "secret",
				"POSTGRES_DB":       "testdb",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		},
		Started: true,
	})
	if err != nil {
		tb.Fatalf("failed to start postgres: %v", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://postgres:secret@%s:%s/testdb?sslmode=disable", host, port.Port())

	tb.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			tb.Logf("failed to terminate postgres: %v", err)
		}
	})

	return &Postgres{DSN: dsn, container: container}
}
```

**Ключевые моменты:**
- Параметр `testing.TB` — стандартный интерфейс, принимает `*testing.T` и другие реализации.
- `tb.Cleanup` автоматически останавливает контейнер после теста/suite.
- Пакет `support` НЕ попадает в production binary — ни один `main.go` его не импортирует. Build tags не нужны.
- Для `test/e2e` (работа с реальными сервисами, не testcontainers) добавь параллельную функцию `StartPostgresFromEnv(tb testing.TB) *Postgres`, которая берёт DSN из `.env` и не поднимает контейнер.

**Использование в integration-suite:**
```go
func (s *MySuite) SetupSuite() {
    s.pg = support.StartPostgres(s.T())
}
```

**Использование в BDD:** см. `x-bdd-godog`, раздел «Bootstrap и ScenarioContext».

## Смежные скиллы

- `x-testing-conventions` — базовые соглашения: suite-структура, skip-маркер, `t.Cleanup`
- `x-bdd-godog` — бизнес-тесты через godog, переиспользует `test/support`
