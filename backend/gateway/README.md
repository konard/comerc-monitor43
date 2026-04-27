# API Gateway

HTTP API Gateway для платформы мониторинга Milan. Принимает клиентские запросы, проверяет токены через Auth Service по gRPC и проксирует трафик к нижестоящим сервисам.

## Возможности

- Проверка Bearer-токенов через gRPC (Auth Service)
- Проксирование запросов к нижестоящим HTTP-сервисам
- Health-check эндпоинты (`/health`, `/ready`, `/live`)
- Структурированное логирование запросов
- OpenTelemetry трассировка ключевых операций
- Пул соединений gRPC и HTTP

## Архитектура

```
gateway/
├── cmd/api-gateway/     # Точка входа
├── internal/
│   ├── adapter/
│   │   ├── grpc/        # Пул gRPC соединений
│   │   └── http/        # Пул HTTP соединений
│   ├── handler/         # HTTP обработчики и middleware
│   ├── infrastructure/
│   │   ├── config/      # Конфигурация из переменных окружения
│   │   ├── health/      # Health-checker
│   │   └── logger/      # Структурированное логирование
│   ├── model/           # Доменные сущности
│   ├── repository/
│   │   ├── grpc/        # Репозитории (gRPC auth + HTTP proxy)
│   │   └── interfaces/  # Интерфейсы репозиториев
│   └── service/
│       ├── dto/         # DTO запросов и ответов
│       ├── auth_service.go
│       └── gateway_service.go
└── pkg/errors/          # Типы ошибок
```

## Требования

- Go 1.25+
- Auth Service (gRPC, по умолчанию `localhost:5001`)

## Установка

```bash
cd /Users/raul/go/src/monitor/backend/gateway

# Скачать зависимости
go mod download

# Собрать бинарник
go build -o api-gateway ./cmd/api-gateway

# Запустить
./api-gateway
```

## Конфигурация

Переменные окружения:

| Переменная | По умолчанию | Описание |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP порт сервера |
| `SERVER_READ_TIMEOUT` | `10s` | Таймаут чтения запроса |
| `SERVER_WRITE_TIMEOUT` | `10s` | Таймаут записи ответа |
| `AUTH_SERVICE_ADDR` | `localhost:5001` | Адрес Auth Service (gRPC) |
| `AUTH_SERVICE_TIMEOUT` | `5s` | Таймаут запроса к Auth Service |
| `SERVICE_TIMEOUT` | `30s` | Таймаут к нижестоящим сервисам |
| `MONITOR_SERVICE_ADDR` | `http://localhost:5002` | Адрес Monitor Service |
| `ALERT_SERVICE_ADDR` | `http://localhost:5003` | Адрес Alert Service |
| `BILLING_SERVICE_ADDR` | `http://localhost:5004` | Адрес Billing Service |
| `SCHEDULER_SERVICE_ADDR` | `http://localhost:5005` | Адрес Scheduler Service |
| `LOG_LEVEL` | `info` | Уровень логирования |
| `ENABLE_TRACING` | `false` | Включить OTel трассировку |
| `METRICS_ENABLED` | `false` | Включить метрики |
| `METRICS_PORT` | `9090` | Порт для метрик |

## Тестирование

```bash
# Все тесты
go test ./...

# С покрытием
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Только быстрые тесты
go test ./... -short
```

## HTTP API

### Публичные эндпоинты (без авторизации)

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/health` | Комплексная проверка состояния сервисов |
| `GET` | `/ready` | Readiness probe (200 если всё здорово) |
| `GET` | `/live` | Liveness probe (200 если gateway запущен) |

### Защищённые эндпоинты (требуют `Authorization: Bearer <token>`)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/v1/gateway/validate` | Проверка токена |
| `*` | `/api/v1/{service}/*` | Проксирование к нижестоящему сервису |

### Пример ответа `/health`

```json
{
  "status": "healthy",
  "timestamp": "2026-04-04T10:00:00Z",
  "services": [
    { "name": "auth-service", "healthy": true, "latency_ms": 3 },
    { "name": "monitor",      "healthy": true, "latency_ms": 5 }
  ],
  "gateway": {
    "name": "api-gateway",
    "version": "1.0.0",
    "started_at": "2026-04-04T09:00:00Z"
  }
}
```

## Observability

- **Трассировка**: OpenTelemetry spans на операциях `authRepository.ValidateToken`, `authRepository.CheckHealth`, `serviceRepository.ProxyRequest`, `serviceRepository.CheckHealth`, `AuthService.*`, `GatewayService.*`
- **Логирование**: структурированный slog через `infrastructure/logger`
- **Health checks**: `/health` агрегирует состояние всех зарегистрированных сервисов
