# Dashboard Service

Сервис read model для дашборда мониторинга. Реализует CQRS-паттерн: потребляет события из RabbitMQ и предоставляет gRPC API и WebSocket для real-time обновлений.

## Возможности

- Агрегированный дашборд статусов мониторов с фильтрацией
- История проверок и инцидентов с метриками за период
- Real-time обновления через WebSocket
- Экспорт данных (CSV/JSON)
- gRPC API с JWT аутентификацией
- OpenTelemetry трейсинг и метрики

## Архитектура

```
dashboard-service/
├── cmd/server/              # Точка входа
├── internal/
│   ├── model/               # Доменные модели
│   ├── repository/postgres/ # PostgreSQL реализации
│   ├── service/realtime/    # WebSocket Hub и клиенты
│   ├── adapters/            # Связка репозиториев с сервисным слоем
│   ├── grpc/                # gRPC обработчики
│   ├── consumer/            # RabbitMQ consumer (события → read model)
│   ├── infrastructure/auth/ # JWT аутентификация
│   ├── config/              # Конфигурация
│   └── ws/                  # WebSocket upgrade handler
├── migrations/              # Миграции БД
└── pkg/                     # logger, telemetry
```

## Требования

- Go 1.21+
- PostgreSQL 14+
- RabbitMQ 3.11+

## Установка

```bash
cp .env.example .env   # настроить переменные окружения
go mod download
make build
./server
```

## Конфигурация

| Переменная | По умолчанию | Описание |
|---|---|---|
| `SERVER_PORT` | `9095` | Порт gRPC сервера |
| `HTTP_PORT` | `8095` | Порт HTTP/WebSocket сервера |
| `DB_HOST` | `localhost` | Хост PostgreSQL |
| `DB_PORT` | `5432` | Порт PostgreSQL |
| `DB_NAME` | `dashboard` | Имя базы данных |
| `DB_USER` | `postgres` | Пользователь БД |
| `DB_PASSWORD` | `postgres` | Пароль БД |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL RabbitMQ |
| `JWT_SECRET` | — | Секретный ключ JWT (обязателен) |
| `OTEL_SERVICE_NAME` | `dashboard-service` | Имя сервиса для трейсинга |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP collector endpoint |

## Тестирование

```bash
make test            # все тесты
make test-short      # без интеграционных
make test-coverage   # с отчётом покрытия
```

## API

### gRPC Methods

- `GetDashboard` — список статусов мониторов с фильтрацией и пагинацией
- `GetCheckHistory` — история проверок монитора
- `GetIncidents` — инциденты монитора
- `GetIncidentDetails` — детали инцидента
- `GetPeriodMetrics` — агрегированные метрики за период
- `ExportDashboard` — экспорт дашборда (CSV/JSON)
- `ExportHistory` — экспорт истории проверок

### WebSocket

```
ws://host:8095/ws?token=<JWT>
```

Клиент получает real-time события при изменении статусов мониторов, завершении проверок и изменении инцидентов.

### Health

- `GET /healthz` — liveness probe
- `GET /readyz` — readiness probe (проверяет БД и RabbitMQ)

## События RabbitMQ

Сервис подписывается на следующие события:

- `monitor.created`, `monitor.updated`, `monitor.deleted`, `monitor.paused`, `monitor.resumed`
- `check.completed`, `check.failed`
- `incident.detected`, `incident.resolved`
- `alert.triggered`, `alert.resolved`
