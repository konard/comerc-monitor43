# monitor

Go-монорепозиторий системы мониторинга.

## Что внутри

- `api/` — общие Go-пакеты и protobuf-контракты (`api/proto`).
- `backend/` — сервисы: `gateway`, `auth-service`, `monitor-service`, `alert-service`, `billing-service`, `dashboard-service`, `integration-service`, `notification-template-service`, `reporting-service`, `scheduler-service`, `check-worker`.
- `backend/testinfra` и `backend/tests` — тестовая инфраструктура и кросс-сервисные интеграционные тесты.
- `frontend/` — SPA на Vite + React + TypeScript + TailwindCSS + Ant Design. Детальная документация в `frontend/README.md`.
- `docs/` — документация.
- `features/` — описание продуктовых фич.

## Запуск

Нужно: Go `1.26.1`, Docker и Docker Compose.

Установка зависимостей:

```bash
make
```

Доступные команды:

```bash
task
```

### Полный стек

Поднять весь стек одной командой из корня репозитория (12 backend-сервисов + Postgres/Redis/RabbitMQ). Корневой compose подключает `backend/docker-compose.yml` через `include`, отдельной команды для backend больше не требуется:

```bash
docker compose up -d                              # базовый стек
docker compose --profile observability up -d     # + jaeger / prometheus / grafana
```

### Только инфраструктура

Когда сервисы запускаются нативно через `go run`, достаточно поднять зависимости из корня:

```bash
docker compose up -d postgres redis rabbitmq
```

При первом запуске Postgres создаёт все БД через `backend/init.sql`. Для сброса: `docker compose down -v`.

### Нативный запуск сервиса

```bash
cd backend/monitor-service
go run cmd/monitor-service/main.go
```

## Frontend

SPA на Vite + React + TypeScript + TailwindCSS + Ant Design в `frontend/`.

```bash
cd frontend
npm install
npm run dev   # http://localhost:5173
npm run build
```

Dev-прокси `/api` направлен на `http://localhost:8080` (api-gateway). Подробности — `frontend/README.md`.

## Миграции БД

Миграции выполняются через [`goose`](https://github.com/pressly/goose) и применяются **автоматически** при старте каждого сервиса. Ручное вмешательство не требуется — достаточно поднять сервис.

### Соглашения

- Каждый сервис хранит миграции в `backend/<service>/migrations/`.
- Несколько сервисов используют общую БД `monitor`, поэтому каждый сервис ведёт собственную таблицу версий goose:

  | Сервис | БД | Таблица версий |
  |---|---|---|
  | `monitor-service` | `monitor` | `goose_db_version` |
  | `auth-service` | `monitor` | `goose_auth_version` |
  | `alert-service` | `monitor` | `goose_alert_version` |
  | `scheduler-service` | `monitor` | `goose_scheduler_version` |
  | `reporting-service` | `monitor` | `goose_reporting_version` |
  | `billing-service` | `billing` | `goose_db_version` |
  | `dashboard-service` | `dashboard` | `goose_db_version` |
  | `integration-service` | `integration_service` | `goose_db_version` |
  | `notification-template-service` | `notification_templates` | `goose_db_version` |

### Ручное управление

Автоматический `Up` при старте покрывает типичный сценарий. Для rollback и сброса в контейнере доступны CLI-команды:

```bash
# Откатить последнюю миграцию
docker exec monitor-<service> ./<service> migrate down

# Полный сброс (down до версии 0)
docker exec monitor-<service> ./<service> migrate reset
```

### Roundtrip-тест

Интеграционный тест `backend/tests/migration/migration_roundtrip_test.go` проверяет, что `Up` → `Reset` возвращает каждую БД в исходное состояние (после `Reset` версия = 0, application-таблицы не остаются).

Запуск (требует живой Postgres из корневого `docker compose`):

```bash
go test -tags integration ./backend/tests/migration/...
```

Настройки подключения — через env vars (дефолты под docker-compose):
- `TEST_DB_HOST` (default: `localhost`)
- `TEST_DB_PORT` (default: `5432`)
- `TEST_DB_USER` (default: `monitor`)
- `TEST_DB_PASSWORD` (default: `monitor_p`)

## Тесты

Прогнать тесты конкретного сервиса:

```bash
cd backend/monitor-service
go test -race -count=1 ./...
```

Прогнать линтёр конкретного сервиса: 

```bash
cd backend/monitor-service
../../script/lint.sh
```

## Порты

### Инфраструктура

| Сервис | Порт |
|---|---|
| Postgres | 5432 |
| Redis | 6379 |
| RabbitMQ | 5672 (AMQP), 15672 (UI) |
| Jaeger UI | 16686 (только `--profile observability`) |
| Prometheus | 9090 (только `--profile observability`) |
| Grafana | 3000 (только `--profile observability`) |

### Backend-сервисы

Наружу публикуются только:

| Сервис | Порт | Назначение |
|---|---|---|
| `api-gateway` | 8080 | единственная HTTP-точка входа для frontend и внешних клиентов |
| `dashboard-service` | 8095, 9095 | WebSocket для real-time обновлений дашборда |

Остальные backend-сервисы доступны только внутри Docker-сети `monitor-network` и общаются между собой по внутренним gRPC/HTTP-портам (см. env-переменные `*_GRPC_PORT` / `SERVER_PORT` в `backend/docker-compose.yml`).

## Сервисы

| Сервис | Что делает | Где детали |
|---|---|---|
| `gateway` | HTTP API gateway: проверяет токены через `auth-service` и проксирует запросы дальше | `backend/gateway/README.md` |
| `auth-service` | Аутентификация и авторизация | `backend/auth-service/README.md` |
| `monitor-service` | Мониторы, проверки и окна обслуживания | `backend/monitor-service/README.md` |
| `alert-service` | Правила алертов и их обработка | `backend/alert-service/README.md` |
| `billing-service` | Тарифы, подписки, платежи и billing webhooks | `backend/billing-service/README.md` |
| `dashboard-service` | Read model для дашборда и real-time обновления | `backend/dashboard-service/README.md` |
| `integration-service` | Webhooks, API-ключи и внешние интеграции | `backend/integration-service/README.md` |
| `notification-template-service` | Шаблоны уведомлений и их рендеринг | `backend/notification-template-service/README.md` |
| `reporting-service` | SLA, аналитика и экспорт отчётов | `backend/reporting-service/README.md` |
| `scheduler-service` | Планирование проверок и управление воркерами | `backend/scheduler-service/README.md` |
| `check-worker` | Выполняет проверки и отправляет результаты в `monitor-service` | `backend/check-worker/README.md` |
