# Tech Context

## Languages & Frameworks

- **Language:** Go 1.23+
- **API:** gRPC with Protocol Buffers
- **HTTP:** net/http with custom adapters
- **Testing:** standard testing + Godog (E2E) + testcontainers

## Core Dependencies

### Infrastructure Adapters (Pure-Golang)
- `github.com/pure-golang/adapters/httpserver/std` - HTTP server
- `github.com/pure-golang/adapters/grpc/std` - gRPC server
- `github.com/pure-golang/adapters/queue/rabbitmq` - Message broker
- `github.com/pure-golang/adapters/kv/redis` - KV storage
- `github.com/pure-golang/adapters/tracing` - OpenTelemetry integration
- `github.com/pure-golang/adapters/metrics` - Prometheus metrics

### Database & Storage
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/pressly/goose/v3` - Database migrations
- `github.com/jmoiron/sqlx` - SQL extensions

### External Services
- `github.com/telegram-bot-api` - Telegram Bot API
- `github.com/go-redis/redis/v8` - Redis client
- `github.com/rabbitmq/amqp091-go` - RabbitMQ client (deprecated, use adapter)

### Configuration
- `github.com/joho/godotenv` - .env file loading
- Internal config structs with env tags

### Observability
- `go.opentelemetry.io/otel` - OpenTelemetry tracing
- `github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing` - Jaeger

## Development Setup

### Prerequisites
- Go 1.23+
- Docker & Docker Compose
- Make (for build automation)

### Local Development

```bash
# Start infrastructure
docker-compose up -d

# Run migrations (per service)
cd backend/monitor-service && make migrate-up
cd backend/alert-service && make migrate-up
cd backend/auth-service && make migrate-up

# Run services (in separate terminals)
make run-monitor
make run-alert
make run-auth
make run-gateway
```

### Environment Configuration

Services use `.env` files with the following key variables:

**Common:**
- `SERVER_PORT` - HTTP port for health checks
- `SERVER_GRPC_PORT` - gRPC port
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `JAEGER_ENDPOINT` - OTLP endpoint

**Monitor Service:**
- `RABBITMQ_URL` - RabbitMQ connection string
- `CHECK_WORKER_COUNT` - Number of concurrent workers
- `CHECK_QUEUE_SIZE` - Buffer size for check queue

**Alert Service:**
- `RABBITMQ_URL` - RabbitMQ connection string
- `REDIS_URL` - Redis connection string
- `TELEGRAM_BOT_TOKEN` - Telegram bot token
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD` - Email config

**Auth Service:**
- `REDIS_URL` - Redis connection string
- `JWT_SECRET` - JWT signing key
- `OAUTH_PROVIDERS` - Comma-separated list of providers
- `OAUTH_<PROVIDER>_CLIENT_ID`, `OAUTH_<PROVIDER>_CLIENT_SECRET`

## Build & Deployment

### Build Process

```bash
# Build all services
make build-all

# Individual services
cd backend/monitor-service && make build
cd backend/alert-service && make build
cd backend/auth-service && make build
cd backend/gateway && make build
```

### Docker Images

Each service can be built as a Docker image:

```dockerfile
# Example structure (to be implemented)
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN make build

FROM alpine:latest
COPY --from=builder /app/bin/service /usr/local/bin/service
CMD ["service"]
```

## Testing

### Unit Tests
```bash
# Per service
cd backend/monitor-service && make test
cd backend/alert-service && make test
cd backend/auth-service && make test
```

### Integration Tests
```bash
# Uses testcontainers for PostgreSQL, RabbitMQ
make test-integration
```

### E2E Tests
```bash
# Uses Godog with real services
cd backend/monitor-service/test/e2e
make test-e2e
```

## Tooling

### Code Generation
```bash
# Generate Protocol Buffer code
cd api/proto
make generate

# Generate mocks (using mockery)
make generate-mocks
```

### Database Migrations
```bash
# Create new migration
goose create add_new_column sql

# Run migrations
goose postgres "postgres://user:pass@localhost:5432/db?sslmode=disable" up ./migrations

# Rollback
goose postgres "postgres://user:pass@localhost:5432/db?sslmode=disable" down ./migrations
```

## Technical Constraints

### Performance
- **Max concurrent checks per worker:** Configurable (default: 10)
- **Check timeout:** 30s default
- **Worker queue:** In-memory channel (buffered)
- **Database connection pool:** 25 max connections

### Scalability
- **Stateless services:** Can scale horizontally
- **Shared state:** PostgreSQL, Redis, RabbitMQ
- **Scheduler:** Single instance (TODO: distributed locking)
- **Alert delivery:** Can run multiple instances with idempotency

### Security
- **TLS:** Required for production PostgreSQL connections
- **Secrets:** Environment variables only (no hardcoded secrets)
- **JWT expiration:** 15 minutes access token, 30 days refresh token
- **OAuth:** HTTPS callback URLs only
