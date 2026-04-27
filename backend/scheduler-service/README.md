# Scheduler Service

Сервис для планирования и распределения проверок мониторов между воркерами в разных зонах доступности.

## Обзор

Scheduler Service является частью системы мониторинга Milan и отвечает за:
- Регистрацию и управление жизненным циклом check-worker'ов
- Планирование проверок мониторов по расписанию
- Распределение проверок между доступными воркерами
- Обработку heartbeat и обнаружение offline воркеров
- Переназначение проверок при потере воркера

## Архитектура

Сервис построен по принципам Clean Architecture:

```
scheduler-service/
├── cmd/scheduler-service/          # Точка входа
├── internal/
│   ├── model/                      # Доменные сущности
│   │   ├── worker.go              # Воркер с зоной и метриками
│   │   └── scheduled_check.go     # Запланированная проверка
│   ├── service/                    # Бизнес-логика
│   │   ├── worker_service.go      # Управление воркерами
│   │   ├── check_scheduler.go     # Планирование проверок
│   │   ├── scheduling_loop.go     # Фоновое планирование
│   │   └── dto/                   # Data Transfer Objects
│   ├── handler/                    # gRPC обработчики
│   │   └── scheduler_handler.go   # Конвертация proto
│   ├── repository/                 # Доступ к данным
│   │   ├── interfaces/            # Интерфейсы репозиториев
│   │   └── postgres/              # PostgreSQL реализации
│   ├── monitor/                    # Интеграция с monitor-service
│   │   ├── client.go              # gRPC клиент
│   │   └── circuit_breaker.go     # Circuit breaker
│   ├── config/                     # Конфигурация
│   └── circuitbreaker/            # Circuit breaker паттерн
└── migrations/                     # Миграции БД
```

## Основные функции

### Управление воркерами

- **Регистрация воркера** (`RegisterWorker`):
  - Уникальное имя (макс. 255 символов)
  - Зона доступности (moscow, spb, etc.)
  - Метаданные (версия, OS, max_concurrent)

- **Heartbeat** (`WorkerHeartbeat`):
  - Обновление статуса (IDLE, BUSY)
  - Метрики: completed, failed, avg_duration_ms
  - Timeout: 30s → OFFLINE

- **Graceful отключение** (`UnregisterWorker`):
  - Переназначение незавершённых проверок
  - Смена статуса на OFFLINE

- **Обнаружение offline воркеров**:
  - Автоматическая отметка по heartbeat timeout
  - Переназначение проверок другим воркерам
  - Cleanup через 10 минут

### Планирование проверок

- **Планирование по расписанию**:
  - Интервалы от 30s до 24h
  - Приоритеты: LOW, NORMAL, HIGH, CRITICAL
  - Предотвращение дублирования

- **Распределение по воркерам**:
  - Приоритет воркеров в той же зоне
  - Выбор воркера с наименьшей нагрузкой
  - Очередь ожидания при отсутствии IDLE воркеров

- **Ручной запуск** (`TriggerScheduledCheck`):
  - Приоритет HIGH
  - Немедленное выполнение

- **Обработка просроченных проверок** (`ReassignOverdueChecks`):
  - Порог: 5 минут
  - Повышение приоритета до HIGH
  - Переназначение другому воркеру

## API

### gRPC Эндпоинты

#### Worker Management
```protobuf
rpc RegisterWorker(RegisterWorkerRequest) returns (Worker);
rpc WorkerHeartbeat(HeartbeatRequest) returns (Empty);
rpc UnregisterWorker(UnregisterWorkerRequest) returns (Empty);
rpc GetWorkerStatus(GetWorkerStatusRequest) returns (Worker);
rpc ListWorkers(ListWorkersRequest) returns (ListWorkersResponse);
```

#### Check Scheduling
```protobuf
rpc GetScheduledChecks(GetScheduledChecksRequest) returns (ScheduledChecks);
rpc GetNextCheck(GetNextCheckRequest) returns (ScheduledCheck);
rpc TriggerScheduledCheck(TriggerScheduledCheckRequest) returns (Empty);
rpc CompleteCheck(CompleteCheckRequest) returns (Empty);
rpc ReassignOverdueChecks(ReassignOverdueChecksRequest) returns (Empty);
```

### HTTP Gateway

Все gRPC методы доступны через HTTP API:

```bash
# Регистрация воркера
POST /api/v1/scheduler/workers
{
  "name": "worker-msk-01",
  "zone": "moscow",
  "metadata": {
    "version": "1.0.0",
    "os": "linux"
  }
}

# Heartbeat
POST /api/v1/scheduler/workers/{worker_id}/heartbeat

# Получение следующей проверки
GET /api/v1/scheduler/monitors/{monitor_id}/next-check

# Ручной запуск проверки
POST /api/v1/scheduler/monitors/{monitor_id}/trigger

# Завершение проверки
POST /api/v1/scheduler/checks/{check_id}/complete
{
  "failed": false,
  "error_message": ""
}

# Переназначение просроченных проверок
POST /api/v1/scheduler/checks/reassign-overdue
{
  "threshold_time": "2026-03-29T10:00:00Z"
}
```

## Запуск

### Локально

```bash
# Копирование конфигурации
cp .env.example .env

# Настройка переменных
vim .env

# Запуск PostgreSQL и RabbitMQ
docker-compose up -d postgres rabbitmq jaeger

# Миграции
goose -dir migrations postgres "user=scheduler password=scheduler host=localhost port=5432 dbname=monitor sslmode=disable" up

# Запуск сервиса
go run cmd/scheduler-service/main.go
```

### Docker

```bash
# Сборка образа
docker build -t scheduler-service .

# Запуск через docker-compose
docker-compose up -d

# Просмотр логов
docker-compose logs -f scheduler-service
```

## Конфигурация

Основные переменные окружения:

```bash
# Server
SERVER_PORT=8084
SERVER_GRPC_PORT=9094

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=monitor
DB_USER=scheduler
DB_PASSWORD=<change-me>
DB_SSL_MODE=disable

# RabbitMQ
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_ENABLED=true

# Monitor Service
MONITOR_SERVICE_GRPC_ADDRESS=localhost:9091

# Timeouts
HEARTBEAT_TIMEOUT=30s
OFFLINE_CLEANUP_INTERVAL=10m
OVERDUE_CHECK_THRESHOLD=5m
SCHEDULING_INTERVAL=10s

# Circuit Breaker
CB_FAILURE_THRESHOLD=5
CB_TIMEOUT=30s
CB_SUCCESS_THRESHOLD=1

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=scheduler-service
OTEL_LOG_LEVEL=info
METRICS_ENABLED=true
METRICS_PORT=9104
```

## Тестирование

```bash
# Все тесты
go test ./...

# Только unit тесты (быстро)
go test -short ./...

# Интеграционные тесты (требует Docker)
go test ./internal/repository/postgres/test/...

# С покрытием
go test -cover ./...

# Benchmarks
go test -bench=. ./...
```

## Observability

### Metrics

Prometheus метрики доступны на порту `9104`:

```bash
curl http://localhost:9104/metrics
```

Основные метрики:
- `scheduler_workers_total` - общее число воркеров
- `scheduler_workers_idle` - IDLE воркеры
- `scheduler_workers_busy` - BUSY воркеры
- `scheduler_workers_offline` - OFFLINE воркеры
- `scheduler_checks_scheduled_total` - запланированные проверки
- `scheduler_checks_completed_total` - завершённые проверки
- `scheduler_checks_failed_total` - неудачные проверки
- `scheduler_checks_overdue_total` - просроченные проверки

### Tracing

OpenTelemetry traces экспортируются в Jaeger:

```bash
http://localhost:16687
```

### Logging

Структурированные логи в JSON формате:

```json
{
  "time": "2026-03-29T18:00:00Z",
  "level": "INFO",
  "msg": "worker registered",
  "worker_id": "123e4567-e89b-12d3-a456-426614174000",
  "worker_name": "worker-msk-01",
  "zone": "moscow"
}
```

## Использование с PureGolang Adapters

Сервис использует адаптеры из `github.com/pure-golang/adapters`:

- **gRPC сервер**: `github.com/pure-golang/adapters/grpc/std`
- **HTTP сервер**: `github.com/pure-golang/adapters/httpserver/std`
- **Logger**: `github.com/pure-golang/adapters/logger`
- **Metrics**: `github.com/pure-golang/adapters/metrics`
- **Tracing**: `github.com/pure-golang/adapters/tracing/jaeger`

## Database Schema

### scheduler_workers
```sql
CREATE TABLE scheduler_workers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    zone VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'IDLE',
    last_heartbeat TIMESTAMP NOT NULL DEFAULT NOW(),
    checks_completed INTEGER NOT NULL DEFAULT 0,
    checks_failed INTEGER NOT NULL DEFAULT 0,
    avg_check_duration_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### scheduled_checks
```sql
CREATE TABLE scheduled_checks (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL,
    worker_id UUID,
    priority VARCHAR(20) NOT NULL DEFAULT 'NORMAL',
    scheduled_at TIMESTAMP NOT NULL,
    executed_at TIMESTAMP,
    completed_at TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### scheduler_audit_log
```sql
CREATE TABLE scheduler_audit_log (
    id UUID PRIMARY KEY,
    action VARCHAR(100) NOT NULL,
    worker_id UUID,
    check_id UUID,
    monitor_id UUID,
    details TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Production Deployment

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scheduler-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: scheduler-service
  template:
    metadata:
      labels:
        app: scheduler-service
    spec:
      containers:
      - name: scheduler-service
        image: scheduler-service:latest
        ports:
        - containerPort: 9094
        env:
        - name: DB_HOST
          valueFrom:
            configMapKeyRef:
              name: db-config
              key: host
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          grpc:
            port: 9094
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          grpc:
            port: 9094
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Мониторинг и алертинг

### Критические алерты

- **Scheduler Workers Offline**:
  ```yaml
  - alert: SchedulerWorkersOffline
    expr: scheduler_workers_offline > 0
    for: 5m
    annotations:
      summary: "Offline workers detected"
      description: "{{ $value }} workers are offline"
  ```

- **Overdue Checks High**:
  ```yaml
  - alert: OverdueChecksHigh
    expr: rate(scheduler_checks_overdue_total[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High overdue check rate"
      description: "{{ $value }} checks/sec are overdue"
  ```

## Troubleshooting

### Воркеры не регистрируются

1. Проверьте подключение к БД:
   ```bash
   psql -h localhost -U scheduler -d monitor
   ```

2. Проверьте логи scheduler-service:
   ```bash
   docker-compose logs -f scheduler-service
   ```

3. Проверьте gRPC connectivity:
   ```bash
   grpcurl -plaintext localhost:9094 list
   ```

### Проверки не назначаются

1. Проверьте наличие IDLE воркеров:
   ```bash
   grpcurl -plaintext -d '{"status": "IDLE"}' \
     localhost:9094 scheduler/SchedulerService/ListWorkers
   ```

2. Проверьте интеграцию с monitor-service:
   ```bash
   curl http://localhost:9091/health
   ```

3. Проверьте circuit breaker状态:
   ```bash
   curl http://localhost:9104/metrics | grep circuit_breaker
   ```

## Лицензия

MIT License

## Контакты

- Backend Team: backend@milan.monitor
- Documentation: https://docs.milan.monitor/scheduler-service
