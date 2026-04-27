# Check Worker

Сервис для выполнения HTTP/HTTPS проверок мониторинга в рамках системы Milan Monitoring.

## Описание

Check Worker — это распределенный исполнитель проверок, который:
- Регистрируется в scheduler-service при запуске
- Получает назначения на выполнения проверок
- Выполняет HTTP/HTTPS запросы к целевым URL
- Отправляет результаты в monitor-service
- Поддерживает graceful shutdown с доработкой активных проверок
- Автоматически переподключается при потере связи с сервисами
- Проверяет окна обслуживания и пропускает проверки во время обслуживания

## Архитектура

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Scheduler      │────▶│  Check Worker    │────▶│    Monitor      │
│   Service       │     │                  │     │    Service      │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                              │
                              ▼
                       ┌─────────────┐
                       │ HTTP/HTTPS  │
                       │   Targets   │
                       └─────────────┘
```

### Компоненты

- **Worker**: Главный lifecycle manager, управляет регистрацией, heartbeat, опросом заданий
- **Executor**: HTTP исполнитель с поддержкой HEAD, GET, Basic Auth, redirects, error classification
- **SchedulerClient**: gRPC клиент для взаимодействия с scheduler-service
- **MonitorClient**: gRPC клиент для отправки результатов в monitor-service
- **ResultBatcher**: Пакетная отправка результатов с дедупликацией
- **ReconnectManager**: Автоматическое переподключение с exponential backoff
- **StateMachine**: Управление состояниями воркера (CONNECTED, DEGRADED, RECONNECTING, SHUTDOWN)
- **MaintenanceChecker**: Проверка окон обслуживания
- **Retry**: Exponential backoff с jitter для сетевых операций
- **Queue**: Локальная очередь результатов при недоступности monitor-service
- **Telemetry**: OpenTelemetry tracing и Prometheus metrics

## Возможности

### Выполнение проверок
- HTTP/HTTPS методы (GET, HEAD)
- Настройка таймаута
- Basic аутентификация
- Поддержка redirects (до 10 по умолчанию)
- Ограничение размера ответа
- Классификация ошибок (DNS, SSL, timeout, connection refused, etc.)

### Надежность
- Автоматическое переподключение к scheduler/monitor сервисам
- Graceful degradation при временных сбоях
- Локальная очередь результатов при недоступности monitor-service
- Пакетная отправка результатов с дедупликацией
- Exponential backoff с jitter для retry

### Обслуживание
- Проверка окон обслуживания перед выполнением проверок
- Пропуск проверок во время активного обслуживания
- Интеграция с monitor-service для получения окон обслуживания

### Observability
- OpenTelemetry distributed tracing
- Prometheus метрики:
  - Количество выполненных проверок
  - Время выполнения проверок
  - Успешность heartbeat
  - Состояние воркера
  - Размер очереди результатов
- Structured JSON logging
- Health check endpoint

## Установка и запуск

### Требования
- Go 1.25+
- Доступ к scheduler-service (порт 9094)
- Доступ к monitor-service (порт 9090)
- OpenTelemetry collector (опционально)

### Переменные окружения

```bash
# Основные настройки
CHECK_WORKER_SCHEDULER_ADDRESS=localhost:9094  # Адрес scheduler-service
CHECK_WORKER_MONITOR_ADDRESS=localhost:9091    # Адрес monitor-service
CHECK_WORKER_NAME=worker-01                    # Уникальное имя воркера
CHECK_WORKER_ZONE=default                       # Зона доступности

# Производительность
CHECK_WORKER_MAX_CONCURRENT_CHECKS=10          # Макс. параллельных проверок
CHECK_WORKER_CHECK_TIMEOUT=30s                 # Таймаут HTTP запроса
CHECK_WORKER_POLL_INTERVAL=5s                  # Интервал опроса заданий
CHECK_WORKER_HEARTBEAT_INTERVAL=10s            # Интервал heartbeat

# Retry
CHECK_WORKER_RETRY_MAX_ATTEMPTS=5              # Макс. попыток retry
CHECK_WORKER_RETRY_BASE_DELAY=2s               # Базовая задержка retry
CHECK_WORKER_RETRY_MAX_DELAY=60s               # Макс. задержка retry

# Queue
CHECK_WORKER_RESULT_QUEUE_MAX_SIZE=1000        # Макс. размер очереди
CHECK_WORKER_RESULT_QUEUE_TTL=24h              # TTL результатов в очереди
CHECK_WORKER_QUEUE_FLUSH_INTERVAL=30s          # Интервал flush очереди

# Observability
CHECK_WORKER_METRICS_ENABLED=true              # Включить метрики
CHECK_WORKER_METRICS_PORT=9105                 # Порт metrics endpoint
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318  # OTEL endpoint
OTEL_SERVICE_NAME=check-worker                 # Имя сервиса для OTEL
OTEL_LOG_LEVEL=info                            # Уровень логирования
```

### Запуск

#### Локально
```bash
# Клонировать репозиторий
git clone https://github.com/raul/monitor.git
cd monitor/backend/check-worker

# Загрузить зависимости
go mod download

# Создать .env файл
cp .env.example .env
# Редактировать .env по необходимости

# Запустить
go run cmd/check-worker/main.go
```

#### С помощью Docker
```bash
# Собрать образ
docker build -t check-worker .

# Запустить контейнер
docker run -d \
  --name check-worker \
  -p 9105:9105 \
  -e CHECK_WORKER_SCHEDULER_ADDRESS=scheduler-service:9094 \
  -e CHECK_WORKER_MONITOR_ADDRESS=monitor-service:9090 \
  -e CHECK_WORKER_NAME=worker-01 \
  check-worker
```

#### С помощью docker-compose
```bash
# Запустить все сервисы
docker-compose up -d

# Просмотреть логи
docker-compose logs -f check-worker

# Остановить
docker-compose down
```

## Использование

### Health Check
```bash
curl http://localhost:9105/health
```

### Metrics
```bash
curl http://localhost:9105/metrics
```

### Пример метрик
```
# HELP check_worker_checks_total Total number of checks executed
# TYPE check_worker_checks_total counter
check_worker_checks_total{status="success"} 142
check_worker_checks_total{status="failure"} 3

# HELP check_worker_check_duration_seconds Duration of check execution
# TYPE check_worker_check_duration_seconds histogram
check_worker_check_duration_seconds_bucket{le="0.1"} 120
check_worker_check_duration_seconds_bucket{le="0.5"} 140
check_worker_check_duration_seconds_bucket{le="+Inf"} 145

# HELP check_worker_heartbeat_total Total number of heartbeats sent
# TYPE check_worker_heartbeat_total counter
check_worker_heartbeat_total{status="success"} 500
```

## Тестирование

### Запуск всех тестов
```bash
go test ./... -v
```

### Запуск с покрытием
```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Интеграционные тесты
```bash
go test ./... -tags=integration
```

### Benchmarks
```bash
go test ./... -bench=. -benchmem
```

## Структура проекта

```
check-worker/
├── cmd/
│   └── check-worker/
│       └── main.go              # Точка входа
├── internal/
│   ├── client/                  # gRPC клиенты
│   │   ├── scheduler_client.go  # Клиент scheduler-service
│   │   ├── monitor_client.go    # Клиент monitor-service
│   │   └── result_batcher.go    # Пакетная отправка результатов
│   ├── config/                  # Конфигурация
│   │   └── config.go
│   ├── executor/                # Исполнитель проверок
│   │   ├── check_executor.go    # HTTP исполнитель
│   │   ├── http_tracer.go       # HTTP трассировка
│   │   ├── check_classifiers.go # Классификация ошибок
│   │   └── maintenance_checker.go # Проверка окон обслуживания
│   ├── health/                  # Health checks
│   │   └── health.go
│   ├── queue/                   # Локальная очередь
│   │   └── result_queue.go
│   ├── retry/                   # Retry логика
│   │   └── retry.go
│   └── worker/                  # Главный lifecycle manager
│       ├── worker.go
│       ├── state_machine.go
│       └── reconnect_manager.go
├── pkg/
│   └── telemetry/               # OpenTelemetry
│       ├── tracer.go
│       └── metrics.go
├── .env.example                 # Пример конфигурации
├── docker-compose.yml           # Docker compose
├── Dockerfile                   # Docker образ
├── go.mod
├── go.sum
└── README.md
```

## Состояния воркера

- **CONNECTED**: Воркер подключен к scheduler и monitor сервисам, работает нормально
- **DEGRADED**: Временные проблемы с сетью, воркер продолжает работу с ограничениями
- **RECONNECTING**: Потеря связи, попытка переподключения
- **SHUTDOWN**: Остановка воркера, graceful shutdown

## Обработка ошибок

### Классификация ошибок
- `DNS_RESOLUTION_FAILED`: Ошибка резолвинга DNS
- `SSL_CERTIFICATE_EXPIRED`: Просроченный SSL сертификат
- `SSL_CERTIFICATE_INVALID`: Недействительный SSL сертификат
- `TIMEOUT`: Таймаут запроса
- `CONNECTION_REFUSED`: Соединение отклонено
- `TOO_MANY_REDIRECTS`: Слишком много redirects
- `RESPONSE_TOO_LARGE`: Ответ превышает лимит размера
- `SLOW_RESPONSE`: Медленный ответ

### Retry стратегия
- Exponential backoff с множителем 2
- Jitter для предотвращения thundering herd
- Максимум 5 попыток (настраивается)
- Базовая задержка 2s (настраивается)
- Максимальная задержка 60s (настраивается)

## Observability

### Tracing
Каждая проверка создает span с атрибутами:
- `check.id`: ID проверки
- `check.url`: URL проверки
- `check.status`: Статус проверки
- `check.duration`: Длительность проверки
- `check.error`: Ошибка (если есть)

### Logging
Структурированные JSON логи с полями:
- `level`: Уровень логирования
- `msg`: Сообщение
- `component`: Компонент (worker, executor, client)
- `check_id`: ID проверки
- `error`: Ошибка
- `worker_id`: ID воркера
- `zone`: Зона доступности

## Мониторинг

### Ключевые метрики для мониторинга
- `check_worker_checks_total{status="failure"}`: Количество неудачных проверок
- `check_worker_check_duration_seconds`: P95, P99 latency
- `check_worker_state`: Состояние воркера
- `check_worker_result_queue_size`: Размер очереди результатов
- `check_worker_heartbeat_total{status="failure"}`: Проблемы с heartbeat

### Алерты
- Высокий процент неудачных проверок (>5%)
- Длительное пребывание в состоянии DEGRADED
- Большой размер очереди результатов (>100)
- Пропуски heartbeat

## Производительность

### Рекомендации
- Настроить `MAX_CONCURRENT_CHECKS` в зависимости от ресурсов
- Использовать несколько воркеров в разных зонах для высокой доступности
- Настроить `CHECK_TIMEOUT` в соответствии с SLA
- Мониторить queue size для обнаружения проблем с monitor-service

### Лимиты
- Максимальное параллельных проверок: 1000
- Максимальный размер очереди: 10000
- Максимальный размер ответа: 10MB

## Troubleshooting

### Воркер не регистрируется в scheduler
- Проверить доступность scheduler-service
- Проверить `CHECK_WORKER_SCHEDULER_ADDRESS`
- Проверить логи на наличие ошибок подключения

### Результаты не отправляются в monitor-service
- Проверить доступность monitor-service
- Проверить `CHECK_WORKER_MONITOR_ADDRESS`
- Проверить размер очереди результатов
- Проверить логи batcher

### Высокий процент неудачных проверок
- Проверить доступность целевых URL
- Проверить таймауты
- Проверить сетевое подключение
- Проверить классификацию ошибок в логах

### Воркер в состоянии DEGRADED
- Проверить подключения к scheduler/monitor сервисам
- Проверить логи reconnect manager
- Проверить сетевое подключение

## Development

### Добавление новых типов проверок
1. Реализовать новый executor в `internal/executor/`
2. Добавить обработку в `Worker.executeCheck()`
3. Добавить тесты
4. Обновить документацию

### Добавление новых метрик
1. Добавить метрику в `pkg/telemetry/metrics.go`
2. Использовать в коде через `metrics.Record...()`
3. Добавить тесты
4. Обновить документацию

## Лицензия

MIT License

## Контакты

- Project: Milan Monitoring System
- Repository: github.com/raul/monitor
