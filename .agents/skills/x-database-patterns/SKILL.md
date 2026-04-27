---
name: "x-database-patterns"
description: "Применяй когда проектируешь PostgreSQL-доступ в сервисе или адаптере: выбор `pgx`/`sqlx`, структура `repo`, общий `dbQuerier`, транзакции, named queries и обработка constraint errors"
compatibility: ../adapters
---

# PostgreSQL-паттерны

## Когда применять

Используй этот скилл, когда:
- выбираешь между `db/pg/pgx` и `db/pg/sqlx`
- проектируешь слой `repo`
- реализуешь транзакцию через несколько репозиториев
- обрабатываешь PostgreSQL constraint errors

Не применяй для:
- доменной логики без SQL
- разовых ad-hoc запросов в handler/service без отдельного репозитория

## 1. Сначала выбери драйвер

| Вариант | Когда брать |
|---|---|
| `db/pg/pgx` | новый проект, нативный pgx, pool, высокая нагрузка |
| `db/pg/sqlx` | нужен `database/sql`-совместимый стек, named queries, уже есть код на sqlx |

Для новых проектов по умолчанию выбирай `pgx`.

Не используй `JSONB` как основной способ моделирования прикладных данных. Если данные участвуют в поиске, фильтрации, join или constraint logic, проектируй отдельные таблицы и явную схему.

## 2. Спроектируй слой `repo`

В service-репозиториях ориентируйся на плоский паттерн: один пакет `repo`, несколько файлов по сущностям, общий `dbQuerier`.

Эталонный паттерн:
- один пакет `repo`
- один файл на сущность
- общий `dbQuerier`
- конструкторы без I/O

Если нескольким репозиториям нужна одна транзакция, вложенные подпакеты `repo/foo`, `repo/bar` для сервисного слоя не используй.

```text
internal/
  repo/
    db.go
    room.go
    participant.go
```

Общий контракт:

```go
type dbQuerier interface {
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
```

Такой интерфейс должен принимать и pool, и transaction.

Если репозиторий в сервисе один, допустим и более компактный вариант с одним основным `repo`-типом и общим `db`.

### Naming conventions

| Что | Паттерн |
|---|---|
| файл сущности | `room.go` |
| общий файл контракта | `db.go` или `errors.go` |
| структура репозитория | `RoomRepo` или компактный `repo` |
| конструктор | `NewRoom(db dbQuerier)` |
| tracer | `var roomTracer = otel.Tracer(".../repo/room")` |
| ошибка пакета | `errDBNotConfigured` |
| ошибка сущности | `errRoomNotFound` |

### Когда допустим один репозиторий на пакет

Если сервис маленький и пакет фактически реализует один адаптер хранения, допустим более компактный вариант:
- один пакет
- один основной тип репозитория
- общий адаптер извне

## 3. Конструктор репозитория

```go
type RoomRepo struct {
    db dbQuerier
}

func NewRoom(db dbQuerier) *RoomRepo {
    return &RoomRepo{db: db}
}
```

Конструктор:
- возвращает одно значение
- не делает I/O
- просто сохраняет зависимость

## 4. Транзакции

### `pgx`

Для кросс-репо операций создавай `tx` снаружи и передавай его в новые экземпляры репозиториев:

```go
return pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
    roomRepo := repo.NewRoom(tx)
    participantRepo := repo.NewParticipant(tx)
    // ...
    return nil
})
```

### `sqlx`

Используй `RunTx` из адаптера:

```go
err := db.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
    // операции
    return nil
})
```

`RunTx` должен автоматически откатывать транзакцию при error и panic.

Если несколько репозиториев участвуют в одной операции, они должны принимать одну и ту же транзакционную зависимость, а не открывать соединения сами.

Если пишешь ручной `Rollback`, проверяй `sql.ErrTxDone`, чтобы не считать уже завершённую транзакцию новой ошибкой.

Пример с isolation level:

```go
opts := &sqlx.TxOptions{
    Isolation: sql.LevelRepeatableRead,
}

err := db.RunTx(ctx, opts, func(ctx context.Context, tx *sqlx.Tx) error {
    return nil
})
```

## 5. Обрабатывай constraint errors через хелперы

Для `sqlx`-адаптера используй:

```go
sqlx.IsUniqueViolation(err)
sqlx.IsForeignKeyViolation(err)
sqlx.IsCheckViolation(err)
sqlx.IsNotNullViolation(err)
sqlx.IsConstraintViolation(err)
```

Если нужно проверить обобщённый случай, используй `sqlx.IsConstraintViolation(err)` как верхнеуровневый helper поверх более узких проверок.

## 6. Query timeout и named queries — только там, где они действительно упрощают код

Если в адаптере есть конфигурируемый `QueryTimeout`, оборачивай каждую операцию в `context.WithTimeout`, а не размазывай таймауты по вызывающему коду.

Если у адаптера есть `Config.QueryTimeout`, оборачивай запросы в локальный `context.WithTimeout`:

```go
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
    if timeout <= 0 {
        return ctx, func() {}
    }
    return context.WithTimeout(ctx, timeout)
}
```

Таймаут должен жить рядом с SQL-адаптером, а не размазываться по service-коду.

Если структура данных хорошо ложится на именованные параметры, `sqlx` может быть проще:

```go
type User struct {
    Name string `db:"name"`
    Age  int    `db:"age"`
}

_, err := db.NamedExec(ctx,
    "INSERT INTO users (name, age) VALUES (:name, :age)",
    User{Name: "John", Age: 30},
)
```

### `sqlx.Connect()` стартовый пример

```go
cfg := sqlx.Config{
    Host:           "localhost",
    Port:           5432,
    User:           "postgres",
    Password:       "secret",
    Database:       "app",
    SSLMode:        "disable",
    ConnectTimeout: 5 * time.Second,
    QueryTimeout:   10 * time.Second,
}
```

## Не делай

- не используй `JSONB` как замену нормальной схеме хранения
- не дроби service-репозитории на вложенные `repo/foo`, `repo/bar`, если им нужна общая транзакция
- не создавай репозиторий, который знает о конкретном pool и не принимает `tx`
- не размазывай SQL-доступ по service-слою
