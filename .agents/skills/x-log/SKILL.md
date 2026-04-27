---
name: "x-log"
description: "Применяй при добавлении или изменении логирования в сервисе, адаптере, репозитории, клиенте, resolver или handler: `*slog.Logger` в структуре, `module`, `logger.FromContext(ctx)` и уровни логирования"
compatibility: ../platform, ../adapters
---
# Логирование

Этот skill — **канонический владелец** logging policy:
- `*slog.Logger` как поле структуры
- `module`
- выбор между `s.logger` и `logger.FromContext(ctx)`
- уровни логирования

Bootstrap monitoring/tracing/metrics принадлежит `x-observability`, а не этому skill.

## Когда применять

- добавляешь логирование в package-level компонент
- переносишь прямые вызовы `slog.*` в project path
- проверяешь naming `module`

## Core workflow

### 1. Конструктор берёт `slog.Default()` сам — не принимай logger параметром

`monitoring.InitDefault` вызывает `slog.SetDefault(...)` один раз при старте приложения. После этого `slog.Default()` в любом пакете возвращает корректно настроенный логгер. Поэтому конструктор сам вызывает `slog.Default().With("module", "...")` и не требует `*slog.Logger` в сигнатуре.

Антипаттерн — прокидывание logger через параметр:

```go
// Плохо: лишний параметр, boilerplate в main
func New(logger *slog.Logger) *Service {
    return &Service{
        logger: logger.With("module", "service.album"),
    }
}
```

Правильно:

```go
// Хорошо: конструктор самодостаточен
func New() *Service {
    return &Service{
        logger: slog.Default().With("module", "service.album"),
    }
}
```

Это убирает цепочку `slog.Default()` в `main.go` и делает конструкторы самодостаточными.

### 2. Добавляй logger только туда, где он реально используется

Если структура не логирует, поле `logger` не нужно.

### 3. Внутри методов не используй `slog` напрямую

- сервисы, репозитории, клиенты используют `s.logger`
- HTTP/gRPC handlers и middleware используют `logger.FromContext(ctx)`

### 4. Именуй `module` согласованно

Эталонные значения:

- `internal/service/room` → `service.room`
- `internal/repo/user` → `repo.user`
- `internal/graph` → `graph`
- `pkg/fcm` → `pkg.fcm`
- `cmd/app` → `main`

Примеры инициализации logger с полем `module`:

```go
type Client struct {
    logger *slog.Logger
}

func NewClient(cfg config.LiveKitConfig) *Client {
    return &Client{
        logger: slog.Default().With("module", "pkg.livekit"),
    }
}
```

```go
func New(...) *Service {
    return &Service{
        logger: slog.Default().With("module", "service.room"),
    }
}
```

### 5. Выбирай уровень и поля осознанно

- `Debug` — детали отладки
- `Info` — значимые события
- `Warn` — нештатное, но не ошибка
- `Error` — ошибка

Правила:
- сообщения логов на английском
- ключи атрибутов в `snake_case`
- ключ для ошибки — `"err"`, а не `"error"`: `slog.Any("err", err)`
- не дублируй одну и ту же ошибку и в логе, и в return-path без необходимости
- используй типизированные `slog.*` атрибуты

## Когда брать `logger.FromContext(ctx)`

Хендлер-слой уже получает обогащённый логгер через контекст.
Эталонный API в скилле — `logger.FromContext(ctx)`.
В реальном коде алиас импорта может отличаться (`logger`, `alogger`), но сам приём остаётся тем же.

```go
func (c *Controller) ready(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    if err := pinger.Ping(ctx); err != nil {
        logger.FromContext(ctx).Error("dependency not ready", slog.Any("err", err))
    }
}
```

Правило:

- сервис / репозиторий / клиент → `s.logger`
- HTTP / gRPC handler → `logger.FromContext(ctx)`

## Когда уместен `slog.InfoContext(ctx, ...)`

`slog.InfoContext(ctx, ...)` и `slog.ErrorContext(ctx, ...)` не являются основным project path для handler flow, но остаются уместными как самостоятельный паттерн, когда код живёт вне request-handling слоя и готового project logger в контексте нет.

Типичный кейс:
- фоновая goroutine
- короткая служебная операция с `context.Context`
- код, где нужно пронести trace-aware context в запись лога, но `logger.FromContext(ctx)` там не инициализируется проектной middleware

```go
func runWarmup(ctx context.Context) {
    slog.InfoContext(ctx, "starting cache warmup")
}
```

Если код исполняется внутри HTTP / gRPC handler или middleware-цепочки, предпочитай `logger.FromContext(ctx)`, чтобы не потерять обогащение логгера полями запроса.

## Инициализация приложения

`monitoring.InitDefault` внутри вызывает `slog.SetDefault(...)`, подменяя глобальный логгер. Поэтому `logger` в `main` создавай **сразу после** `InitDefault` — иначе получишь логгер без structured handler.

```go
closeMonitoring := monitoring.InitDefault(cfg.Monitoring)
logger := slog.Default().With("module", "main")
defer func() {
    if err := closeMonitoring(); err != nil {
        logger.Error("Failed to close monitoring", slog.Any("err", err))
    }
}()

logger.Info("Connecting dependencies")
```

Не используй `slog.Default()` напрямую после создания `logger` — все логи в `main` пиши через `logger`.

## Что особенно легко испортить

- поле `logger` есть, но ни один метод его не использует
- `module` не соответствует пути пакета
- в handler используется `s.logger`, и теряются поля из контекста запроса
- один и тот же `error` и логируется, и сразу возвращается вверх без добавочной ценности
- конструктор принимает `*slog.Logger` параметром вместо вызова `slog.Default()` внутри

## Короткий чек-лист

- logger живёт в структуре, а не в прямых `slog.*` вызовах
- `module` назван по project path
- request-aware flow использует `logger.FromContext(ctx)`
- атрибуты типизированы

## Смежные skills

- `x-observability`
