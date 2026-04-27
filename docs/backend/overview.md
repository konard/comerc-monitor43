# Backend Architecture Overview

## Система Milan - Мониторинг доступности для РФ/СНГ

**Версия:** 1.0 MVP
**Дата:** Март 2026
**Статус:** Архитектурные решения

---

## Ключевые решения

### Архитектурный подход
- **Тип:** Микросервисы
- **Стиль:** API Gateway → Core Services → Workers
- **Коммуникация:** gRPC (синхронный), RabbitMQ (асинхронный)

### Инфраструктура
- **Провайдер:** Timeweb Cloud (соответствие 152-ФЗ)
- **Kubernetes:** k3s cluster на 3 виртуалках (multi-master HA)
- **База данных:** PostgreSQL (Timeweb Cloud)
- **Очереди:** RabbitMQ
- **Кэш:** Minimal cache approach
- **Observability:** OpenTelemetry (pure-golang adapters)

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                            │
│  React/Next.js Dashboard + Management UI                    │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTPS
┌────────────────────▼────────────────────────────────────────┐
│                    API Gateway (Go)                         │
│  - JWT Authentication                                        │
│  - Rate Limiting (in-memory)                                │
│  - Service Routing (gRPC)                                   │
│  - Observability (OpenTelemetry)                            │
└─────┬─────────────┬─────────────┬──────────────┬───────────┘
      │             │             │              │
┌─────▼─────┐ ┌────▼────┐ ┌──────▼─────┐ ┌────▼────────┐
│  Monitor  │ │  Alert  │ │   Billing  │ │  Scheduler  │
│  Service  │ │ Service │ │  Service   │ │   Service   │
│   (gRPC)  │ │ (gRPC)  │ │   (gRPC)   │ │   (gRPC)    │
└─────┬─────┘ └────┬────┘ └──────┬─────┘ └────┬────────┘
      │             │             │              │
      │             │             │              │
┌─────▼─────┐ ┌────▼────┐ ┌──────▼─────┐         │
│PostgreSQL │ │RabbitMQ │ │PostgreSQL  │         │
│ (monitors)│ │(alerts) │ │ (billing)  │         │
└───────────┘ └─────────┘ └────────────┘         │
                                               │
┌──────────────────────────────────────────────▼────────┐
│                   Check Workers (Go)                  │
│  - Pull schedule from Scheduler via gRPC              │
│  - Execute HTTP checks                                │
│  - Report results to Monitor Service                  │
└───────────────────────────────────────────────────────┘
```

---

## Core Services

### 1. API Gateway
**Назначение:** Единая точка входа, аутентификация и маршрутизация

**Ответственности:**
- JWT validation (stateless, access 15min + refresh 7days)
- Rate limiting (in-memory, per-user)
- Request routing to services
- Logging and metrics
- TLS termination

**Технологии:**
- Go + custom gRPC proxy
- OpenTelemetry adapters (pure-golang)
- In-memory rate limiter

### 2. Monitor Service
**Назначение:** Управление мониторами и их состояниями

**Ответственности:**
- CRUD операции для мониторов
- Хранение current state (last_check_time, status, last_error)
- Обработка результатов проверок от workers
- Trigger алертов при смене статуса
- История изменений (events)

**Технологии:**
- Go + gRPC
- PostgreSQL (monitors, monitor_events)
- Redis (optional, minimal cache)

### 3. Alert Service
**Назначение:** Отправка уведомлений пользователям

**Ответственности:**
- Отправка в Email, Telegram, Webhooks
- Rate limiting (hybrid: per-monitor + daily limits)
- Retry logic (3x с exponential backoff)
- Template rendering (Email, Markdown)

**Технологии:**
- Go + gRPC
- RabbitMQ (alerts, notifications queues)
- SMTP client, Telegram Bot API

### 4. Billing Service
**Назначение:** Подписки и платежи

**Ответственности:**
- Управление тарифами и подписками
- Интеграция с ЮKassa (СБП, карты)
- Обработка webhook'ов от ЮKassa
- Проверка лимитов мониторов

**Технологии:**
- Go + gRPC
- PostgreSQL (billing tables)
- ЮKassa REST API

### 5. Scheduler Service
**Назначение:** Планирование проверок мониторов

**Ответственности:**
- Расчет next_check_time для каждого монитора
- Предоставление расписания worker'ам по gRPC
- Управление приоритетами (Business 30сек > Free 5мин)

**Технологии:**
- Go + gRPC
- PostgreSQL (monitors table)

### 6. Check Workers
**Назначение:** Выполнение HTTP проверок

**Ответственности:**
- Опрос Scheduler'а за расписанием
- Выполнение HTTP/HTTPS запросов
- Измерение response time
- Отчет результатов в Monitor Service

**Технологии:**
- Go + gRPC client
- HTTP client with timeouts
- Autoscaling (K8s HPA)

---

## Коммуникация между сервисами

### Синхронная (gRPC)
- API Gateway → Core Services
- Check Workers → Scheduler
- Check Workers → Monitor Service

**Преимущества:**
- Type-safe (protobuf)
- Эффективно (binary protocol)
- Bidirectional streaming

### Асинхронная (RabbitMQ)
- Monitor Service → Alert Service (alerts queue)
- Alert Service → notifications (email/telegram/webhooks)
- ЮKassa → Billing Service (payment webhooks)

**Очереди для MVP:**
1. `checks.queue` - задания на проверки
2. `alerts.queue` - триггеринг алертов
3. `notifications.queue` - отправка уведомлений
4. `payment.webhooks` - платежи от ЮKassa

---

## Масштабируемость

### Horizontal Scaling
- **Stateless services:** API Gateway, core services
- **K8s HPA:** по CPU/memory
- **Workers:** autoscaling по queue depth

### Database Scaling
- **Read replicas** (будет нужно при росте)
- **Connection pooling** (PgBouncer)
- **Partitioning** (monitor_events по времени)

---

## Security & Compliance

### 152-ФЗ Compliance
- **Data residency:** Timeweb Cloud (РФ)
- **Encryption:** TLS 1.3 + Field-level (ПД)
- **Logging:** 7-90 дней tiered retention

### API Security
- **Authentication:** Stateless JWT
- **Authorization:** RBAC (по тарифам)
- **Rate limiting:** In-memory

---

## Следующие шаги

1. ✅ Архитектурные решения определены
2. 🔄 Детализация API (protobuf)
3. 🔄 Database schema design
4. 🔄 CI/CD pipeline
5. 🔄 Monitoring setup

---

**Дата:** 2026-03-08
**Версия:** 1.0
