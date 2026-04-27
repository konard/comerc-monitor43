---
name: "x-env-config"
description: "Применяй когда нужно добавить или изменить конфигурацию через env: `Config` struct, теги `envconfig`, вложенные adapter/platform config, загрузка через `env.InitConfig`, `.env`, `doc.go` и тесты конфигурации"
compatibility: ../adapters
---

# Конфигурация через env

## Когда применять

Используй этот скилл, когда:
- добавляешь новый `Config` в адаптере или пакете инфраструктуры
- расширяешь существующую конфигурацию новыми env-переменными
- переносишь хардкод в конфиг

Не применяй для:
- значений, которые живут только в runtime одного запроса
- параметров, которые удобнее передавать обычным аргументом функции

## Workflow

Если нужно быстро проверить согласованность `Config` / `doc.go` / `.env`, используй `scripts/check-env-config.py`.

Пример вызова:

```bash
python3 .agents/skills/x-env-config/scripts/check-env-config.py internal/config/config.go internal/config/doc.go .env
```

### 1. Объяви `Config`

```go
type Config struct {
    Host     string        `envconfig:"REDIS_HOST" default:"localhost"`
    Port     int           `envconfig:"REDIS_PORT" default:"6379"`
    Password string        `envconfig:"REDIS_PASSWORD"`
    Timeout  time.Duration `envconfig:"REDIS_TIMEOUT" default:"10s"`
}
```

Правила:
- имя переменной должно быть стабильным и привязанным к компоненту: `REDIS_*`, `S3_*`, `PG_*`
- безопасные локальные значения можно задавать через `default`
- секреты не делай `required:"true"` автоматически: сначала проверь, существует ли безопасный noop/local режим
- длительности задавай строкой (`"5s"`, `"1m"`)

#### Корневой `Config` с вложенными adapter/platform-конфигами

```go
type Config struct {
    Environment string        `envconfig:"ENVIRONMENT" default:"development"`
    Monitoring  monitoring.Config
    HTTPServer  ahttp.Config
    GRPCServer  agrpc.Config
    Database    apgx.Config
}
```

Правило:

- корневой `Config` собирает настройки приложения верхнего уровня
- вложенные зависимости остаются вложенными типами, не расплющивай их вручную без причины

### 2. Загрузи конфиг через `env.InitConfig`

```go
var cfg Config
if err := env.InitConfig(&cfg); err != nil {
    return fmt.Errorf("failed to load redis config: %w", err)
}
```

Что делает `env.InitConfig` в `../adapters/env`:
- пытается загрузить `.env` из текущей директории
- не падает, если `.env` нет
- парсит переменные окружения в структуру через `envconfig`

### 3. Обнови `.env` и пример значений

```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_TIMEOUT=10s
```

### 4. Задокументируй переменные

Все поля `Config` перечисляй в секции `Конфигурация:` файла `doc.go`.
Для формата комментария пакета см. `x-doc-go`.

Если корневой `Config` содержит вложенные adapter/platform-конфиги, перечисли верхнеуровневые переменные явно и укажи, где смотреть остальные.

#### `doc.go` не обязан дублировать вообще всё

- верхнеуровневые переменные перечислены явно
- для остальных есть ссылка на вложенные конфиги `monitoring`, `logger`, `metrics`, `tracing`

Это хороший баланс между полнотой и шумом.

#### Когда полезен полный список env

- если пакет сам владеет большим числом прикладных переменных
- если эти переменные действительно важны как контракт пакета

Тогда допустим длинный перечислительный `doc.go`.

#### Legacy-совместимость

- можно явно документировать legacy env migration
- нужно указать, какие новые переменные имеют приоритет
- это часть контракта, если приложение реально поддерживает legacy aliases

### 5. Проверь edge cases

Проверь минимум:
- загрузку значений по умолчанию
- ошибку на обязательном поле, если оно действительно обязательно
- корректный парсинг `time.Duration`, `bool`, `int`
- поведение unit-теста с `t.Setenv` без `t.Parallel()`
- поведение теста с `.env` и `os.Chdir()` без параллельного запуска, если проверяешь загрузку из cwd

Если тест на конфиг меняет env процесса или рабочую директорию, это уже process-wide state.
Не ускоряй такой тест через `t.Parallel()`: сначала обеспечь cleanup, а для общих правил см. `x-testing-conventions`.

#### Unit-тесты конфигурации

- app/runtime код загружает конфиг через `env.InitConfig`
- unit-тесты часто проверяют `envconfig.Process("", &cfg)` напрямую
- отдельные правила по `t.Setenv`, `t.Parallel()` и AAA лежат в `x-testing-conventions`

## Не делай

- не смешивай env-настройки и пост-конструкторную мутацию полей
- не используй разные префиксы для одного и того же компонента
- не прячь обязательные env-переменные в коде без отражения в `doc.go`
- не дублируй полный список env из вложенных зависимостей, если достаточно сослаться на их конфиг

## Полезные ресурсы

- `scripts/check-env-config.py` — проверка согласованности `envconfig` тегов, `doc.go` и `.env`
