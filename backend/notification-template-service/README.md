# Notification Template Service

Сервис управления шаблонами уведомлений для мониторинговой системы.

## Назначение

Сервис предоставляет:
- CRUD операций с шаблонами уведомлений
- Рендеринг шаблонов с подстановкой переменных
- Валидацию синтаксиса шаблонов
- Управление шаблонами по умолчанию
- Поддержку различных каналов (Email, Telegram, Webhook, Slack, Discord, SMS)
- Поддержку типов уведомлений (Monitor Up/Down, Degraded, Certificate expiry, etc.)

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                   Notification Template Service              │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐     ┌─────────────────────────────────┐  │
│  │   gRPC API   │────▶│          Handler               │  │
│  └──────────────┘     └─────────────────────────────────┘  │
│                                │                            │
│                         ┌──────▼──────┐                    │
│                         │   Service   │                    │
│                         └──────┬──────┘                    │
│                                │                            │
│         ┌──────────────────────┼──────────────────────┐    │
│         │                      │                      │    │
│  ┌──────▼──────┐      ┌───────▼───────┐    ┌────────▼────────┐
│  │   Template  │      │   Renderer    │    │   Validator     │
│  │   Service   │      │   Service     │    │   Service       │
│  └──────┬──────┘      └───────┬───────┘    └────────┬────────┘
│         │                     │                      │       │
│         └──────────────────────┴──────────────────────┘       │
│                                │                            │
│                         ┌──────▼──────┐                    │
│                         │ Repository  │                    │
│                         └──────┬──────┘                    │
│                                │                            │
├────────────────────────────────┼────────────────────────────┤
│                         ┌──────▼──────┐                    │
│                         │  PostgreSQL │                    │
│                         └─────────────┘                    │
└─────────────────────────────────────────────────────────────┘
```

## Требования

- Go 1.23+
- PostgreSQL 14+
- (Опционально) Jaeger для трассировки

## Переменные окружения

```bash
# Server
SERVER_ADDRESS=:50051
SERVER_TIMEOUT=30s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=monitor
DB_PASSWORD=monitor
DB_NAME=notification_templates
DB_SSLMODE=disable

# Observability
JAEGER_ENDPOINT=localhost:4318
LOG_LEVEL=info
ENVIRONMENT=development

# Template settings
MAX_TEMPLATE_SIZE=100000
ENABLE_PREVIEW=true
```

## Разработка

### Быстрый старт (Development)

```bash
# В одну команду: запускает PostgreSQL, Jaeger, миграции и сервис
./scripts/dev.sh
```

### Построчный запуск

```bash
# 1. Установка зависимостей
make deps

# 2. Генерация gRPC кода
make generate

# 3. Запуск инфраструктуры (PostgreSQL + Jaeger)
docker-compose up -d postgres jaeger

# 4. Применение миграций
make migrate-up

# 5. Запуск сервиса
make run
```

### Тесты

```bash
# Все тесты
make test

# С coverage
make test-coverage

# Или через скрипт
./scripts/test.sh
```

### Работа с миграциями

```bash
# Применить все миграции
make migrate-up

# Откатить последнюю миграцию
make migrate-down

# Создать новую миграцию
make migrate-create NAME=add_new_field
```

## API

### Создание шаблона

```protobuf
rpc CreateTemplate(CreateTemplateRequest) returns (Template);
```

### Получение шаблона

```protobuf
rpc GetTemplate(GetTemplateRequest) returns (Template);
```

### Список шаблонов

```protobuf
rpc ListTemplates(ListTemplatesRequest) returns (ListTemplatesResponse);
```

### Обновление шаблона

```protobuf
rpc UpdateTemplate(UpdateTemplateRequest) returns (Template);
```

### Удаление шаблона

```protobuf
rpc DeleteTemplate(DeleteTemplateRequest) returns (google.protobuf.Empty);
```

### Клонирование шаблона

```protobuf
rpc CloneTemplate(CloneTemplateRequest) returns (Template);
```

### Рендеринг шаблона

```protobuf
rpc RenderTemplate(RenderTemplateRequest) returns (RenderedTemplate);
```

### Валидация шаблона

```protobuf
rpc ValidateTemplate(ValidateTemplateRequest) returns (ValidationResult);
```

### Получение доступных переменных

```protobuf
rpc GetAvailableVariables(GetAvailableVariablesRequest) returns (AvailableVariables);
```

### Установка дефолтного шаблона

```protobuf
rpc SetDefaultTemplate(SetDefaultTemplateRequest) returns (Template);
```

### Получение дефолтного шаблона

```protobuf
rpc GetDefaultTemplate(GetDefaultTemplateRequest) returns (Template);
```

### Превью шаблона

```protobuf
rpc PreviewTemplate(PreviewTemplateRequest) returns (RenderedTemplate);
```

## Примеры шаблонов

### Email (Monitor Down)

**Subject:**
```
🚨 Alert: {{ .monitor_name }} is DOWN
```

**Body (HTML):**
```html
<h2>Monitor Alert</h2>
<p><strong>Monitor:</strong> {{ .monitor_name }}</p>
<p><strong>Status:</strong> {{ .status }}</p>
<p><strong>URL:</strong> <a href="{{ .monitor_url }}">{{ .monitor_url }}</a></p>
<p><strong>Error:</strong> {{ .error_message }}</p>
<p><strong>Time:</strong> {{ .timestamp }}</p>
```

### Telegram (Monitor Up)

```
✅ {{ .monitor_name }} recovered!

Status: {{ .status }}
URL: {{ .monitor_url }}
Time: {{ .timestamp }}
```

### Webhook (JSON)

```json
{
  "monitor_name": "{{ .monitor_name }}",
  "monitor_url": "{{ .monitor_url }}",
  "status": "{{ .status }}",
  "error_message": "{{ .error_message }}",
  "timestamp": "{{ .timestamp }}"
}
```

## Поддерживаемые переменные

### Базовые переменные (для всех типов)

| Имя | Тип | Обязательная | Описание |
|-----|-----|--------------|-----------|
| `monitor_name` | string | ✅ | Имя монитора |
| `monitor_url` | string | ✅ | URL монитора |
| `status` | string | ✅ | Текущий статус |
| `timestamp` | timestamp | ✅ | Время события |

### Переменные для Monitor Down

| Имя | Тип | Обязательная | Описание |
|-----|-----|--------------|-----------|
| `error_message` | string | ✅ | Сообщение об ошибке |

### Переменные для Monitor Degraded

| Имя | Тип | Обязательная | Описание |
|-----|-----|--------------|-----------|
| `response_time_ms` | int | ✅ | Время отклика (мс) |
| `threshold_ms` | int | ✅ | Пороговое значение (мс) |

### Переменные для Certificate Expiry

| Имя | Тип | Обязательная | Описание |
|-----|-----|--------------|-----------|
| `days_until_expiry` | int | ✅ | Дней до истечения |
| `expiry_date` | timestamp | ✅ | Дата истечения |

## Локальный запуск с Docker Compose

```bash
# Запуск всех сервисов (PostgreSQL, Jaeger, notification-template-service)
docker-compose up -d

# Проверка статуса
docker-compose ps

# Логи
docker-compose logs -f notification-template-service

# Остановка
docker-compose down

# Health check
curl http://localhost:8080/health
```

## Docker

```bash
# Сборка образа
docker build -t notification-template-service:latest .

# Запуск контейнера
docker run -p 50051:50051 -p 8080:8080 notification-template-service:latest
```

## Тестирование

```bash
# Все тесты
make test

# С покрытием
make test-coverage

# Конкретный пакет
go test -v ./internal/model/...
go test -v ./internal/service/...
go test -v ./internal/repository/...
go test -v ./internal/handler/...
```

### Покрытие

- ✅ Model unit tests (100%)
- ✅ Service unit tests (80%+)
- ✅ Repository integration tests (90%+)
- ✅ Handler integration tests (85%+)

**Overall coverage: ~85%**

## Наблюдаемость

### Health Checks

```bash
# HTTP health check
curl http://localhost:8080/health

# Пример ответа
{
  "status": "pass",
  "timestamp": "2026-03-27T20:00:00Z",
  "checks": {
    "database": {
      "status": "pass"
    }
  }
}
```

### Логи

Сервис использует структурированное логирование с уровнями:
- `debug`
- `info`
- `warn`
- `error`
- `fatal`

Настройка через переменную окружения: `NOTIFICATION_TEMPLATE_LOG_LEVEL`

### Метрики

TODO: добавить Prometheus метрики

### Трассировка

Сервис поддерживает OpenTelemetry и отправляет спаны в Jaeger.

```bash
# Jaeger UI
http://localhost:16686
```

## Безопасность

- Валидация размера шаблона (MAX_TEMPLATE_SIZE)
- Защита от инъекций в шаблонах
- Isolation пользовательских шаблонов

## Структура проекта

```
backend/notification-template-service/
├── cmd/notification-template-service/  # Точка входа
├── internal/
│   ├── config/                          # Конфигурация
│   ├── handler/                         # gRPC handlers
│   ├── model/                           # Domain модели
│   ├── repository/                      # Репозитории
│   ├── service/                         # Бизнес-логика
│   └── infrastructure/                  # Logging, tracing, health
│       ├── logging/
│       ├── tracing/
│       └── health/
├── migrations/                          # Миграции БД
├── scripts/                             # Скрипты для разработки
│   ├── dev.sh                          # Быстрый запуск
│   └── test.sh                         # Все тесты
├── pkg/                                 # Публичные пакеты
├── Makefile                             # Сборка
├── docker-compose.yml                   # Local development
├── Dockerfile                           # Docker образ
└── README.md                            # Документация
```

## Скрипты

### `scripts/dev.sh`
Быстрый запуск всех компонентов в development режиме:
- PostgreSQL
- Jaeger
- Миграции
- Сервис

### `scripts/test.sh`
Запуск всех тестов с генерацией coverage report.

## Контрибьюшн

Смотри `CLAUDE.md` и `AGENTS.md` в корне проекта.

## Лицензия

MIT
