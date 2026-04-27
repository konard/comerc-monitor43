# Reporting Service

Сервис генерации SLA отчётов, аналитики response time и экспорта данных для мониторинга Milan.

## Возможности

- Генерация SLA отчётов по мониторам за произвольный период
- Аналитика response time: среднее, медиана, P95, P99
- Отчёты по инцидентам и статистике проверок
- Экспорт отчётов в CSV и PDF форматы
- JWT аутентификация через gRPC middleware
- OpenTelemetry tracing и Prometheus метрики

## Архитектура

```
reporting-service/
├── cmd/server/              # Точка входа
├── internal/
│   ├── config/              # Конфигурация из переменных окружения
│   ├── model/               # Доменные модели (SLAReport)
│   ├── repository/
│   │   ├── interfaces/      # Интерфейсы репозиториев
│   │   └── postgres/        # PostgreSQL реализации
│   ├── service/             # Бизнес-логика (SLA, Analytics, Export)
│   ├── handler/             # gRPC handlers
│   └── infrastructure/
│       ├── auth/            # JWT middleware
│       ├── export/          # CSV и PDF экспортёры
│       ├── monitor_client/  # gRPC клиент monitor-service
│       └── tracing/         # OpenTelemetry helpers
├── migrations/              # Миграции БД
└── .env.example             # Пример конфигурации
```

## Требования

- Go 1.22+
- PostgreSQL 14+
- monitor-service (gRPC, для получения данных о проверках)
- (опционально) Docker для интеграционных тестов

## Установка

```bash
cd /Users/raul/go/src/monitor/backend/reporting-service

cp .env.example .env
# Отредактируйте .env

go mod download
make build
./server
```

## Конфигурация

| Переменная | По умолчанию | Описание |
|---|---|---|
| `SERVER_PORT` | `8084` | Порт HTTP сервера |
| `SERVER_GRPC_PORT` | `5007` | Порт gRPC сервера |
| `DB_HOST` | `localhost` | Хост PostgreSQL |
| `DB_PORT` | `5432` | Порт PostgreSQL |
| `DB_NAME` | `monitor` | Имя базы данных |
| `DB_USER` | `monitor` | Пользователь БД |
| `DB_PASSWORD` | `monitor` | Пароль БД |
| `JWT_SECRET_KEY` | — | Секрет для валидации JWT |
| `MONITOR_SERVICE_GRPC_ADDRESS` | `localhost:5002` | Адрес monitor-service |
| `OTEL_ENABLED` | `true` | Включить трейсинг |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | OTLP collector |
| `METRICS_ENABLED` | `true` | Включить метрики |
| `METRICS_PORT` | `9097` | Порт Prometheus метрик |
| `REPORTING_RETENTION_DAYS` | `90` | Хранение отчётов (дней) |
| `REPORTING_MAX_EXPORT_ROWS` | `50000` | Лимит строк при экспорте |

## Тестирование

```bash
# Все тесты
make test

# Только короткие тесты (без интеграционных)
make test-short

# С покрытием
make test-coverage
```

## API

### gRPC Methods (`reporting.ReportingService`)

| Метод | Описание |
|---|---|
| `GenerateSLAReport` | Сгенерировать SLA отчёт за период |
| `ListSLAReports` | Список сохранённых SLA отчётов |
| `GetSLAReport` | Получить SLA отчёт по ID |
| `GetResponseTimeMetrics` | Метрики response time (avg, p95, p99) |
| `GetIncidentsReport` | Отчёт по инцидентам |
| `GetCheckCountByStatus` | Статистика проверок по статусам |
| `ExportReportCSV` | Экспорт отчёта в CSV |
| `ExportReportPDF` | Экспорт отчёта в PDF |

### Health Check

```
GET /health  →  {"status":"healthy"}
```

## Лицензия

Internal project — Milan Monitoring System
