# Alert Service

Сервис управления алертами и уведомлениями для мониторинга Milan.

## Возможности

- Создание и управление алертами для мониторов
- Множественные каналы уведомлений (Telegram, Email, Webhook)
- Повторная доставка уведомлений с настраиваемыми интервалами
- gRPC API с HTTP Gateway annotations
- Clean Architecture с разделением на слои

## Архитектура

```
alert-service/
├── cmd/server/          # Точка входа
├── internal/
│   ├── model/          # Доменная модель (сущности)
│   ├── repository/     # Доступ к данным
│   │   ├── interfaces/ # Интерфейсы репозиториев
│   │   └── postgres/   # PostgreSQL реализации
│   ├── service/        # Бизнес-логика
│   │   ├── channels/  # Управление каналами
│   │   ├── delivery/  # Доставка уведомлений
│   │   └── triggering/# Триггеры алертов
│   ├── grpc/          # gRPC handlers
│   ├── channels/      # Клиенты каналов (Telegram, Email, Webhook)
│   └── config/        # Конфигурация
├── migrations/         # Миграции БД
├── features/           # Gherkin feature files
├── pkg/logger/        # Структурированное логирование
└── .env.example       # Пример конфигурации
```

## Требования

- Go 1.26+
- PostgreSQL 14+
- (опционально) Docker для testcontainers

## Установка

```bash
# Клонируйте репозиторий
cd /Users/raul/go/src/monitor/backend/alert-service

# Скопируйте .env.example в .env и настройте
cp .env.example .env

# Скачайте зависимости
go mod download

# Соберите бинарник
make build

# Запустите
./server
```

## Конфигурация

Основные переменные окружения (см. `.env.example`):

- `SERVER_PORT` - порт gRPC сервера (по умолчанию 50051)
- `DB_HOST`, `DB_PORT`, `DB_NAME` - параметры подключения к PostgreSQL
- `TELEGRAM_BOT_TOKEN` - токен для Telegram уведомлений
- `SMTP_*` - параметры SMTP для email уведомлений
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP collector для observability

## Тестирование

```bash
# Все тесты
make test

# Только короткие тесты (без интеграционных)
make test-short

# С покрытием
make test-coverage

# Интеграционные тесты (требует Docker)
make integration-test
```

## API

### gRPC Methods

- `CreateAlert` - создать новый алерт
- `GetAlert` - получить алерт по ID
- `ListAlerts` - список алертов с пагинацией
- `UpdateAlert` - обновить алерт
- `DeleteAlert` - удалить алерт
- `EnableAlert` - включить алерт
- `DisableAlert` - отключить алерт
- `GetNotificationChannels` - получить каналы уведомлений пользователя
- `UpdateNotificationChannels` - обновить каналы уведомлений

### HTTP Gateway

Все gRPC методы также доступны через HTTP благодаря gRPC-Gateway annotations.

Пример:
```bash
curl -X POST http://localhost:50051/v1/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "monitor_id": "...",
    "type": "STATUS_CODE",
    "consecutive_failures": 3
  }'
```

## Статус разработки

### ✅ Завершено:
- Clean Architecture с разделением на слои
- Доменная модель (Alert, AlertRule, AlertChannel, DeliveryAttempt)
- PostgreSQL репозитории (AlertRepository, AlertRuleRepository, AlertChannelRepository, DeliveryAttemptRepository)
- gRPC handlers для всех 9 методов
- DTO мапперы proto ↔ domain
- Feature files (20 Gherkin сценариев)
- Миграция БД
- Структурированное логирование (slog)
- Конфигурация через переменные окружения
- Makefile для common operations

### ⏸️ В процессе:
- Реальная интеграция репозиториев в main.go (текущая: dummy сервисы)
- Полная реализация всех методов AlertService
- Unit тесты для сервисов
- OpenTelemetry tracing
- Интеграционные тесты с testcontainers

### 📊 DOD Compliance:
- backend/DOD.md: ~55%
- features/DOD.md: 100% ✨

## Лицензия

Internal project - Milan Monitoring System
