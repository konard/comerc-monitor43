# System Patterns

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway                             │
│                   (gRPC Reverse Proxy)                       │
└─────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
        ┌───────▼──────┐ ┌───▼────┐ ┌─────▼──────┐
        │   Monitor    │ │ Alert  │ │    Auth    │
        │   Service    │ │Service │ │  Service   │
        └───────┬──────┘ └───┬────┘ └─────┬──────┘
                │            │             │
        ┌───────▼────────────▼─────────────▼──────┐
        │         Infrastructure Layer             │
        │  PostgreSQL | Redis | RabbitMQ | Jaeger │
        └──────────────────────────────────────────┘
```

## Service Boundaries

### Monitor Service
- **Responsibility:** Проверка доступности HTTP/HTTPS эндпоинтов
- **Input:** gRPC commands (CreateMonitor, UpdateMonitor, etc.)
- **Output:** Events to RabbitMQ (check results, incidents)
- **Storage:** PostgreSQL (monitors, check_results, incidents)

### Alert Service
- **Responsibility:** Управление алертами и доставкой уведомлений
- **Input:** RabbitMQ events from Monitor Service
- **Output:** Notifications to channels (Telegram, Email, Webhook)
- **Storage:** PostgreSQL (alerts, channels, deliveries)

### Auth Service
- **Responsibility:** OAuth авторизация и управление пользователями
- **Input:** OAuth callbacks, token refresh requests
- **Output:** JWT tokens, session data
- **Storage:** PostgreSQL (users, oauth_accounts, refresh_tokens)
- **Cache:** Redis (sessions)

## Communication Patterns

### Synchronous: gRPC
- Used for: Command/Query operations between Gateway and Services
- Protocols: HTTP/2 with Protocol Buffers
- Examples: CreateMonitor, GetAlert, OAuthLogin

### Asynchronous: RabbitMQ
- Used for: Event propagation between services
- Pattern: Publisher-Subscriber with durable queues
- Examples: CheckResult → AlertTrigger, IncidentCreated → AlertDelivery

## Data Flow

### Monitoring Flow
```
1. User creates monitor via gRPC
2. Scheduler queues periodic check
3. Worker executes HTTP check
4. Result saved to PostgreSQL
5. Event published to RabbitMQ
6. Alert Service consumes event
7. Trigger logic evaluates alert conditions
8. If triggered → delivery to channels
```

### Alerting Flow
```
1. Alert Service receives CheckResult event
2. Trigger Service evaluates conditions
   - Consecutive failures?
   - Flapping detection?
3. If triggered → Delivery Service
4. Rate limiting check
5. Channel client sends notification
6. Delivery result saved (success/retry)
```

## Database Patterns

### Schema Design
- **Separate databases per service** - isolation
- **Migration files timestamped** - `YYYYMMDDHHMMSS_description.sql`
- **Audit logs** - separate tables for tracking changes
- **Soft deletes** - status fields instead of DELETE

### Transaction Boundaries
- **Monitor Service:** Single check result = single transaction
- **Alert Service:** Alert triggering + delivery creation = transaction
- **Auth Service:** User creation + OAuth account = transaction

## Error Handling Patterns

### Retry Logic
- **RabbitMQ:** x-death header for retry counting
- **HTTP Clients:** Exponential backoff for transient failures
- **Channel Delivery:** Configurable retry attempts per channel

### Circuit Breaking
- **HTTP Checker:** Timeout per check, fail fast
- **Channel Clients:** Don't block queue on failing channel
- **Database:** Connection pool limits, query timeouts

## Security Patterns

### Authentication
- **OAuth 2.0:** Third-party identity providers
- **JWT:** Stateless tokens with expiration
- **Refresh Tokens:** Long-lived credentials in database
- **Sessions:** Redis-based for quick lookup

### Authorization
- **Gateway:** gRPC interceptor for JWT validation
- **Service-to-Service:** (TODO) mTLS or API keys

## Observability Patterns

### Distributed Tracing
- **Jaeger:** OTLP endpoint on port 4318
- **Context Propagation:** gRPC metadata, RabbitMQ headers
- **Span Naming:** `service.operation` format

### Metrics
- **Prometheus format:** Exposed on `/metrics` HTTP endpoint
- **Key metrics:**
  - Check duration (histogram)
  - Check success rate (counter)
  - Alert triggers (counter)
  - Channel delivery success (counter)

### Logging
- **Structured logging:** slog with JSON output
- **Log levels:** DEBUG, INFO, WARN, ERROR
- **Context inclusion:** Request ID, trace ID in all logs
