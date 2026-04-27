# Integration Service

Integration Service для управления webhook интеграциями, API ключами и импортом мониторов из внешних систем.

## Возможности

### Webhook Интеграции (features/07_01_webhooks)
- Создание webhook интеграций для отправки алертов во внешние системы
- Поддержка custom headers и HMAC-SHA256 подписей
- Фильтрация по severity (critical, warning, degraded)
- Retry логика с экспоненциальным backoff
- Priority-based delivery (high, normal, low)
- Автоматическая деактивация после неудачных попыток
- Статистика использования (total sent, success rate, avg response time)

### API Ключи (features/07_02_api_access)
- Создание API ключей с scopes (read_monitors, write_monitors, и т.д.)
- Префиксы: `baku_ro_` (read-only), `baku_rw_` (read-write), `baku_admin_` (admin)
- Истечение срока действия (expiration)
- IP whitelist для ограничения доступа
- Rate limiting (по умолчанию 100 req/min)
- Отслеживание использования (statistics, logs)
- Ротация секретных ключей

### Импорт мониторов (features/07_02_api_access)
- Импорт из UptimeRobot (JSON формат)
- Импорт из Pingdom (JSON формат)
- Импорт из CSV
- Импорт из JSON
- Валидация и обработка ошибок
- Разрешение конфликтов (overwrite или skip)
- История импортов с деталями

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                  Integration Service                        │
├─────────────────────────────────────────────────────────────┤
│  gRPC Handlers                                              │
│  ├── WebhookHandler      │  ├── APIKeyHandler              │
│  └── ImportHandler       │  └── ...                        │
├─────────────────────────────────────────────────────────────┤
│  Services                                                  │
│  ├── WebhookService       │  ├── APIKeyService             │
│  ├── WebhookDeliverySvc   │  ├── ImportService             │
│  ├── HMACService          │  ├── EncryptionService         │
│  └── APIKeyGenerator      │  └── RateLimiter               │
├─────────────────────────────────────────────────────────────┤
│  Repositories               │  Parsers                     │
│  ├── WebhookRepository     │  ├── UptimeRobotParser        │
│  ├── APIKeyRepository      │  ├── PingdomParser            │
│  ├── ImportHistoryRepo     │  ├── CSVParser                │
│  └── ...                   │  └── JSONParser               │
└─────────────────────────────────────────────────────────────┘
```

## Интеграционные точки

### Входящие события (RabbitMQ)
- `alert.triggered` → Integration Service доставляет webhook

### Исходящие вызовы (gRPC)
- **Gateway** → ValidateAPIKey (валидация API ключей)
- **Monitor Service** ← ImportMonitors (создание мониторов при импорте)
- **Billing Service** ← CheckLimits (проверка лимитов webhooks/API keys)
- **Auth Service** ← ValidateUser (валидация пользователя)

## Структура БД

```sql
webhook_integrations      -- Webhook интеграции
webhook_delivery_attempts -- Попытки доставки с retry
api_keys                  -- API ключи
api_key_usage_logs        -- Логи использования API ключей
import_history            -- История импортов
```

## Запуск

### Локально

```bash
# 1. Создать .env файл
cp .env.example .env

# 2. Запустить PostgreSQL и RabbitMQ (docker-compose)
docker-compose up -d postgres rabbitmq

# 3. Применить миграции
go run cmd/integration-service/main.go migrate up

# 4. Запустить сервис
go run cmd/integration-service/main.go
```

### С Docker

```bash
docker-compose up -d integration-service
```

## Конфигурация

Основные environment variables:

| Переменная | Описание | Default |
|------------|----------|---------|
| `SERVER_PORT` | HTTP port | 8084 |
| `SERVER_GRPC_PORT` | gRPC port | 5014 |
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_NAME` | Database name | integration_service |
| `ENCRYPTION_KEY` | Ключ шифрования (32 bytes) | **required** |
| `RABBITMQ_ENABLED` | Включить RabbitMQ | true |
| `AUTH_SERVICE_GRPC_ADDRESS` | Auth service addr | localhost:5001 |
| `MONITOR_SERVICE_GRPC_ADDRESS` | Monitor service addr | localhost:5000 |

## Тестирование

```bash
# Unit tests
go test ./...

# Integration tests (с testcontainers)
go test -tags=integration ./...

# E2E tests (Godog)
godog test/features
```

## API Примеры

### Создание webhook

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "uuid",
    "name": "Slack Integration",
    "url": "https://hooks.slack.com/services/...",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "priority": "high"
  }' \
  localhost:5014 \
  integration.WebhookIntegrationService/CreateWebhook
```

### Создание API ключа

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "uuid",
    "name": "Production Key",
    "key_type": "read_write",
    "scopes": ["read_monitors", "write_monitors"],
    "expires_at": "2026-12-31T23:59:59Z"
  }' \
  localhost:5014 \
  integration.APIKeyService/CreateAPIKey
```

Ответ содержит `full_key` — **сохраните его**, он больше не будет отображаться!

### Импорт мониторов

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "uuid",
    "source": "uptimerobot",
    "file_data": <base64 encoded JSON>,
    "file_name": "uptimerobot_export.json",
    "overwrite_existing": false
  }' \
  localhost:5014 \
  integration.ImportService/ImportMonitors
```

## Мониторинг

### Metrics (Prometheus)
- `integration_webhooks_sent_total` — количество отправленных webhooks
- `integration_webhooks_failed_total` — количество неудачных попыток
- `integration_api_keys_validations_total` — валидации API ключей
- `integration_imports_total` — количество импортов

### Health checks
- `GET /health` — проверка здоровья сервиса
- `GET /ready` — проверка готовности к работе

## Статус разработки

### ✅ Phase 1: Foundation (已完成)
- [x] Service structure
- [x] Database migrations
- [x] Domain models (WebhookIntegration, APIKey, ImportHistory)
- [x] Security services (HMAC, encryption, key generator)
- [x] Config and infrastructure setup
- [x] Proto definitions

### 🔄 Phase 2: Webhooks (в процессе)
- [ ] WebhookRepository implementation
- [ ] WebhookService CRUD
- [ ] WebhookHandler gRPC implementation
- [ ] Unit tests

### ⏳ Phase 3: API Keys
- [ ] APIKeyRepository implementation
- [ ] APIKeyService CRUD + validation
- [ ] RateLimiter implementation
- [ ] APIKeyHandler gRPC implementation
- [ ] Unit tests

### ⏳ Phase 4-7: Остальные фазы
- См. план в `/Users/raul/.claude/plans/curious-giggling-pretzel.md`

## Документация

- [План реализации](../../../.claude/plans/curious-giggling-pretzel.md)
- [Features](../../../features/07_integrations/)
- [AGENTS.md](../../../AGENTS.md)
- [DOD.md](../../../DOD.md)

## Лицензия

MIT
