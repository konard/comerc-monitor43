# Auth Service

Authentication and authorization microservice for the Milan monitoring system.

## Features

- **OAuth Authentication**: Google OAuth 2.0 (extensible for future providers)
- **Password Authentication**: For service accounts
- **JWT Tokens**: Access tokens (15min) and refresh tokens (7 days)
- **Session Management**: Redis-based sessions with expiration
- **Account Lockout**: After 10 failed login attempts (configurable)
- **Audit Logging**: All auth events logged for compliance (152-ФЗ)
- **RBAC**: Authorization by subscription tier

## Architecture

```
cmd/auth-service/          # Application entry point
internal/
  model/                   # Domain entities (User, Session, RefreshToken, OAuthAccount, AuditLog)
  service/                 # Business logic and orchestration
    dto/                   # Data transfer objects
  repository/
    interfaces/            # Repository contracts
    postgres/              # PostgreSQL implementations
  handler/                 # gRPC handlers
  infrastructure/
    config/                # env-based configuration
    jwt/                   # JWT generation/validation
    oauth/                 # OAuth providers (Google)
    postgres/              # DB connection
    redis/                 # Redis client and session store
  publisher/               # RabbitMQ event publishing
  rabbitmq/                # RabbitMQ topology
pkg/
  errors/                  # Error types with codes
  logger/                  # Structured logging
migrations/                # Goose database migrations
test/
  smoke/                   # Smoke tests (docker compose)
```

## Dependencies

- PostgreSQL 15+
- Redis 6+
- RabbitMQ 3.13+ (optional, for event publishing)
- Google OAuth credentials

## Configuration

All configuration is via environment variables — see `.env.example` for the full list.

## Running

```bash
# Install dependencies
go mod download

# Run migrations (goose CLI required)
goose -dir ./migrations postgres "$DATABASE_URL" up

# Run the service
go run ./cmd/auth-service
```

## API

The service exposes both gRPC and HTTP endpoints.

### gRPC (port 5001)

See `api/proto/auth.proto` for full gRPC API definition.

### HTTP (port 8080)

The gRPC gateway provides HTTP endpoints:

- `POST /api/v1/auth/oauth/google` — Google OAuth login
- `POST /api/v1/auth/oauth/google/callback` — OAuth callback
- `POST /api/v1/auth/login` — Password login
- `POST /api/v1/auth/register` — Registration
- `POST /api/v1/auth/refresh` — Refresh token
- `POST /api/v1/auth/logout` — Logout
- `POST /api/v1/auth/validate` — Validate token
- `GET /api/v1/auth/sessions` — List sessions
- `DELETE /api/v1/auth/sessions/{session_id}` — Revoke session
- `GET /api/v1/auth/users/{user_id}` — Get user
- `POST /api/v1/auth/admin/lock` — Lock account
- `POST /api/v1/auth/admin/unlock` — Unlock account

## Development

```bash
# Unit tests
go test -short -race -count=1 ./...

# Smoke tests (docker required)
go test -v -count=1 -tags smoke ./test/smoke/...

# Lint (from this directory)
bash ../../scripts/lint.sh

# Regenerate mocks
mockery
```

BDD-сценарии для этого сервиса живут в корневом `features/` и покрывают межсервисное взаимодействие единым test-stack'ом.

## Database Migrations

Миграции через [Goose](https://github.com/pressly/goose):

```bash
# Apply all pending migrations
goose -dir ./migrations postgres "$DATABASE_URL" up

# Rollback last migration
goose -dir ./migrations postgres "$DATABASE_URL" down

# Reset (rollback all)
goose -dir ./migrations postgres "$DATABASE_URL" reset
```

## License

See project root LICENSE file.
