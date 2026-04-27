# Backend Definition of Done (DoD)

> **Version:** 2.2
> **Date:** 2026-03-27
> **Status:** Active
> **Changes:** Added pure-golang/adapters and pure-golang/platform separation, JSON→TEXT storage rule, YYYYMMDDHHMMSS migrations, TODO/stub checks
> **Purpose:** Quality standards for Go backend implementation in Milan monitoring service

---

## Introduction

This Backend Definition of DoD establishes quality standards for Go backend implementation across all microservices. It complements the feature files DoD and backend architecture documentation, providing specific technical requirements for backend development.

### Scope

- Applies to all Go microservices: API Gateway, Monitor Service, Alert Service, Billing Service, Scheduler Service, Check Workers
- Covers architecture patterns, code standards, testing, database operations, queue systems, observability, security, API design, and configuration
- Serves as a checklist before code commits, pull requests, and releases

### Related Documents

- `docs/backend/` - Backend architecture reference
- `features/DOD.md` - Feature files Definition of Done
- `api/proto/` - gRPC definitions (when implemented)

---

## Architecture Requirements (MUST)

### 1.1 Clean Architecture

Every service MUST follow Clean Architecture principles with clear separation of concerns:

#### Layer Structure

```
service/
├── cmd/                    # Application entry points
│   └── main.go            # Service bootstrap
├── internal/              # Private application code
│   ├── model/              # Domain entities (Business logic models)
│   ├── repository/         # Data access (interfaces + implementations)
│   │   ├── interfaces/    # Repository interfaces (for testability)
│   │   └── postgres/     # PostgreSQL implementations
│   ├── service/            # Business logic / Use cases
│   │   └── dto/           # Data transfer objects
│   ├── handler/            # gRPC handlers
│   ├── config/             # Configuration
│   └── infrastructure/     # External adapters (Redis, JWT, OAuth)
└── pkg/                  # Public packages shared across services
```

#### Layer Responsibilities

- **Model Layer**: Domain entities with business logic methods. No external dependencies.
- **Repository Layer**: Data access with interfaces for testability and implementations.
- **Service Layer**: Use cases implemented as struct methods. Groups related operations.
- **Handler Layer**: gRPC handlers that delegate to service layer and convert protobuf.
- **Infrastructure Layer**: External adapters (Redis, JWT, OAuth). Implements interfaces.
- **DTO Layer**: Data transfer objects for request/response conversion.

#### Dependency Rules

- Dependencies MUST point inward: Infrastructure → Application → Domain
- Domain layer MUST NOT import from infrastructure or interfaces
- Domain layer MUST NOT depend on external frameworks

### 1.2 Go-Way Framework Patterns

Services MUST follow Go-Way framework conventions:

#### Project Structure

- Use `internal/` for private code (cannot be imported from outside)
- Use `pkg/` for public packages that may be reused
- Place service-specific code in `internal/` to enforce encapsulation

#### Conventions

- Context MUST be the first parameter in exported functions
- Errors MUST be the last return value in exported functions
- Interface names: simple verb or noun (e.g., `MonitorRepository`, `QueueProducer`)
- Implementation names: interface + specific implementation (e.g., `PostgresMonitorRepository`, `RabbitMQQueueProducer`)

### 1.3 Directory Structure Per Service

Each service MUST follow this structure:

```
service-name/
├── migrations/              # Database migrations (Goose)
│   ├── 20240314000000_create_monitors_table.sql
│   ├── 20240314000001_create_users_table.sql
│   └── ...
├── cmd/
│   └── service-name/
│       └── main.go
├── internal/
│   ├── model/
│   │   ├── user.go
│   │   ├── session.go
│   │   └── ...
│   ├── repository/
│   │   ├── interfaces/
│   │   │   ├── user_repository.go
│   │   │   ├── session_repository.go
│   │   │   └── ...
│   │   └── postgres/
│   │       ├── user_repository.go
│   │       ├── session_repository.go
│   │       └── ...
│   ├── service/
│   │   ├── auth_service.go        # PasswordLogin, OAuthCallback, Register
│   │   ├── token_service.go       # RefreshToken, ValidateToken
│   │   ├── session_service.go     # ListSessions, RevokeSession
│   │   ├── user_service.go       # GetUser, LockAccount, UnlockAccount
│   │   └── dto/
│   │       ├── auth_dto.go
│   │       ├── session_dto.go
│   │       └── user_dto.go
│   ├── handler/
│   │   ├── auth_handler.go       # Auth-related gRPC methods
│   │   ├── session_handler.go    # Session-related gRPC methods
│   │   ├── user_handler.go       # User-related gRPC methods
│   │   └── server.go            # gRPC server setup
│   ├── config/
│   │   └── config.go
│   └── infrastructure/
│       ├── redis/
│       ├── jwt/
│       └── oauth/
├── pkg/
│   └── shared/
│       └── utils.go
├── .env.example
├── go.mod
├── go.sum
└── Makefile
```

### 1.4 Service Layer (Use Cases)

Service layer implements use cases as struct methods following clean architecture:

#### Service Structure Pattern

```go
// internal/service/auth_service.go
package service

type AuthService struct {
    userRepo    *repository.UserRepository
    tokenService *TokenService
    // ... other dependencies
}

// Constructor with dependency injection
func NewAuthService(
    userRepo *repository.UserRepository,
    tokenService *TokenService,
) *AuthService {
    return &AuthService{
        userRepo:    userRepo,
        tokenService: tokenService,
    }
}

// Use case methods
func (s *AuthService) PasswordLogin(ctx context.Context, req *PasswordLoginRequest, ipAddress, userAgent string) (*AuthResponse, error)
func (s *AuthService) OAuthCallback(ctx context.Context, req *OAuthCallbackRequest, ipAddress, userAgent string) (*AuthResponse, error)
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest, ipAddress, userAgent string) (*AuthResponse, error)
```

#### Controller/Handler Pattern

Handlers delegate to services and handle protocol conversion:

```go
// internal/handler/auth_handler.go
package handler

type AuthHandler struct {
    authService *service.AuthService
    // ... other services
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
    return &AuthHandler{
        authService: authService,
    }
}

// gRPC method that delegates to service
func (h *AuthHandler) PasswordLogin(ctx context.Context, req *pb.PasswordLoginRequest) (*pb.AuthResponse, error) {
    dtoReq := &service.PasswordLoginRequest{
        Email:    req.Email,
        Password: req.Password,
    }

    resp, err := h.authService.PasswordLogin(ctx, dtoReq, ipAddress, userAgent)
    // Convert to protobuf
    return &pb.AuthResponse{
        AccessToken:  resp.AccessToken,
        User:        toProtoUser(resp.User),
    }, nil
}
```

#### Key Principles

- **Service as Use Case Container**: Each service groups related operations
- **Constructor DI**: All dependencies injected via constructor
- **Repository Interface + Implementation**: Interfaces for testability, implementations in repository/postgres/
- **Handler → Service Delegation**: Handler only converts protobuf and calls service

---

## Dependency Injection Patterns

### 1.5 PureGolang Adapters Repository

All external dependencies MUST use pure-golang adapters when available:

#### Adapters Repository

Location: `git@git@github.com:pure-golang/adapters.git`

Contains:
- Database adapters (PostgreSQL with connection pooling)
- RabbitMQ adapters (producers, consumers with retry)
- Redis adapters (cache, session storage)
- HTTP server adapters (gRPC, HTTP middleware)
- HTTP client adapters (retry, observability)
- Other external service connections and ports

#### Platform Repository

Location: `git@git@github.com:pure-golang/platform.git`

Contains:
- Monitoring and observability tools
- OpenTelemetry integrations (traces, metrics, logs)
- Prometheus instrumentation
- Jaeger tracing exporters
- Structured logging frameworks
- Health check utilities
- Alert rule definitions

#### Adapter Usage Rules

- **First check adapters**: Before implementing any infrastructure code, check if adapter exists in pure-golang/adapters
- **Check platform for monitoring**: For observability, metrics, tracing - check pure-golang/platform
- **Configuration**: Use adapters/platform Config structures and env parsing utilities
- **Instances**: Use adapters/platform constructors and factory functions
- **Environment variables**: Follow adapters/platform env naming conventions
- **No duplication**: Do NOT copy adapter code to infrastructure/ unless absolutely necessary

#### Infrastructure Directory Purpose

The `internal/infrastructure/` directory is for:
- Service-specific configuration wiring
- Adapters/platform initialization
- Interface implementations specific to this service
- NOT for re-implementing existing adapters

```go
// GOOD: Use adapters
import "github.com/pure-golang/adapters/database/postgres"
import "github.com/pure-golang/adapters/rabbitmq"

// GOOD: Use platform for monitoring
import "githib.com/pure-golang/newbackend/platform/observability"

// BAD: Re-implement in infrastructure/
// (unless adapters/platform don't have what you need)
```

### 1.6 Dependency Injection Patterns

Services MUST use dependency injection for testability:

#### Constructor Pattern

```go
func NewMonitorService(
    repo domain.MonitorRepository,
    queueProducer queue.Producer,
    logger logger.Logger,
) *MonitorService {
    return &MonitorService{
        repo:         repo,
        queueProducer: queueProducer,
        logger:       logger,
    }
}
```

#### Interface-Based Dependencies

- Define interfaces in domain layer
- Implement in infrastructure layer
- Inject via constructors
- Enable mocking in tests

---

## Code Standards (MUST)

### 2.1 Go Naming Conventions

#### Package Names

- Lowercase, single word when possible
- No underscores or mixed caps
- Descriptive but concise
- Examples: `monitor`, `alert`, `billing`

#### Function Names

- Exported: PascalCase (e.g., `CreateMonitor`)
- Unexported: camelCase (e.g., `createMonitor`)
- Descriptive verbs for actions (Create, Get, Update, Delete)
- Nouns for accessors (e.g., `ID()`, `Name()`, `URL()`)

#### Variable Names

- camelCase for local variables
- Short names for common patterns (ctx, err, db, req, resp)
- Descriptive names for business logic (monitor, subscription, webhook)

#### Constants

- UpperCamelCase for exported constants
- lowerCamelCase for unexported constants
- Group related constants

```go
const (
    DefaultInterval = 5 * time.Minute
    MaxRetries      = 3
)
```

### 2.2 Error Handling

#### Error Wrapping

- Use `fmt.Errorf` with `%v` verb for error formatting
- Preserve error context through call stack
- Avoid error wrapping at multiple levels (format once per layer)

```go
func (s *MonitorService) CreateMonitor(ctx context.Context, req *CreateMonitorRequest) (*Monitor, error) {
    if err := validateRequest(req); err != nil {
        return nil, fmt.Errorf("invalid request: %v", err)
    }

    monitor, err := s.repo.Create(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to create monitor: %v", err)
    }

    return monitor, nil
}
```

#### Context Propagation

- Context MUST be passed through all exported functions
- Context MUST be used for cancellation and timeouts
- Context MUST be propagated to external calls (database, HTTP, queues)

```go
func (s *MonitorService) GetMonitor(ctx context.Context, id string) (*Monitor, error) {
    monitor, err := s.repo.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get monitor: %v", err)
    }
    return monitor, nil
}
```

#### Error Codes

- Error responses MUST use uppercase snake_case codes
- Error codes MUST be consistent across services
- Error responses MUST include technical details (not localized text)
- Error responses MUST include request_id for tracing

```json
{
  "error": {
    "code": "MONITOR_LIMIT_REACHED",
    "message": "Technical description of the error",
    "details": {
      "limit": 25,
      "current": 26,
      "tier": "Free"
    },
    "request_id": "req_1234567890"
  }
}
```

### 2.3 Structured Logging

All logging MUST be structured JSON format:

#### Log Levels

- `debug`: Detailed diagnostic information
- `info`: General informational messages
- `warn`: Warning messages (not errors)
- `error`: Error messages with context
- `fatal`: Fatal errors causing service termination

#### Log Structure

```go
logger.Info("monitor created",
    "monitor_id", monitor.ID,
    "user_id", monitor.UserID,
    "url", monitor.URL,
    "interval", monitor.Interval,
)
```

#### Log Context

- Include request_id in all request-related logs
- Include user_id in all user-related logs
- Include monitor_id in all monitor-related logs
- Include error details in error logs
- Use consistent field names across services

#### PureGolang Platform for Logging

- Use pure-golang/platform OpenTelemetry logging adapters
- Configure structured logging with consistent format
- Include trace_id and span_id from OpenTelemetry context

### 2.4 gRPC Patterns

#### Service Definition

- Define services in protobuf files in `api/proto/`
- Use proto3 syntax
- Include comments referencing use cases from feature files directly above RPC methods

```protobuf
// epic=01_monitoring, us=01_monitor_management
service MonitorService {
  // uc_01_01_01: Create monitor
  rpc CreateMonitor(CreateMonitorRequest) returns (Monitor);
}
```

#### Request/Response Messages

- Use snake_case for field names
- Include google.protobuf.Empty for void responses
- Use google.api.http for HTTP gateway annotations
- Use proper message types (repeated fields, enums)

```protobuf
message CreateMonitorRequest {
  string name = 1;
  string url = 2;
  int64 interval_seconds = 3;
  google.protobuf.Duration timeout = 4;
}
```

#### HTTP Gateway

Every gRPC endpoint MUST have HTTP gateway mapping:

```protobuf
// epic=01_monitoring, us=01_monitor_management
service MonitorService {
  // uc_01_01_01: Create monitor
  rpc CreateMonitor(CreateMonitorRequest) returns (Monitor) {
    option (google.api.http) = {
      post: "/api/v1/monitors"
      body: "*"
    };
  }

  // uc_01_01_02: Get monitor details
  rpc GetMonitor(GetMonitorRequest) returns (Monitor) {
    option (google.api.http) = {
      get: "/api/v1/monitors/{id}"
    };
  }
}
```

### 2.5 API Design

#### RESTful Conventions

- Use standard HTTP methods (GET, POST, PUT, DELETE)
- Use plural nouns for resource collections (e.g., `/monitors`)
- Use singular nouns for individual resources (e.g., `/monitors/{id}`)
- Use query parameters for filtering and pagination

#### Versioning

- All API endpoints MUST be versioned (e.g., `/api/v1/`)
- Use URL path versioning, not headers
- Maintain backward compatibility within major version

#### Feature File Use Case Comments

Every RPC endpoint MUST reference use case from feature files:

- **Service level**: epic and us (user story) comments
- **Method level**: uc (use case) comments directly above RPC methods

```protobuf
// epic=01_monitoring, us=01_monitor_management
service MonitorService {
  // uc_01_01_01: Create monitor
  rpc CreateMonitor(CreateMonitorRequest) returns (Monitor);

  // uc_01_01_02: Get monitor details
  rpc GetMonitor(GetMonitorRequest) returns (Monitor);

  // uc_01_01_03: List all monitors
  rpc ListMonitors(ListMonitorsRequest) returns (ListMonitorsResponse);
}
```

#### Error Response Format

All error responses MUST follow consistent format:

```json
{
  "error": {
    "code": "UPPERCASE_SNAKE_CASE",
    "message": "Technical description",
    "details": {
      "field": "value"
    },
    "request_id": "uuid"
  }
}
```

---

## Testing Requirements (MUST)

### 3.1 Unit Tests

Every service MUST have unit tests with >80% coverage:

#### Test Organization

```
service-name/
└── internal/
    ├── domain/
    │   ├── entities/
    │   │   └── monitor.go
    │   │   └── monitor_test.go
    │   ├── repositories/
    │   │   └── monitor_repository_test.go
    │   └── services/
    │       └── monitor_service_test.go
    ├── application/
    │   └── usecases/
    │       └── create_monitor_test.go
    └── infrastructure/
        ├── postgres/
        │   └── monitor_repository_impl_test.go
        └── rabbitmq/
            └── alert_producer_test.go
```

#### Test Structure

```go
func TestMonitorService_CreateMonitor(t *testing.T) {
    tests := []struct {
        name    string
        req     *CreateMonitorRequest
        want    *Monitor
        wantErr bool
        errCode string
    }{
        {
            name: "success",
            req:  &CreateMonitorRequest{Name: "Test", URL: "https://example.com"},
            want: &Monitor{Name: "Test", URL: "https://example.com"},
        },
        {
            name:    "invalid url",
            req:     &CreateMonitorRequest{Name: "Test", URL: "invalid-url"},
            wantErr: true,
            errCode: "INVALID_URL",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockRepo := &MockMonitorRepository{}
            service := NewMonitorService(mockRepo, nil, logger)

            // Execute
            got, err := service.CreateMonitor(context.Background(), tt.req)

            // Assert
            if tt.wantErr {
                require.Error(t, err)
                require.Equal(t, tt.errCode, GetErrorCode(err))
            } else {
                require.NoError(t, err)
                require.Equal(t, tt.want, got)
            }
        })
    }
}
```

#### Coverage Requirements

- Overall coverage: >80%
- Domain layer coverage: >90%
- Application layer coverage: >85%
- Infrastructure layer coverage: >75%
- Interfaces layer coverage: >70%

### 3.2 Integration Tests

Each service MUST have integration tests:

#### Database Integration

```go
func TestPostgresMonitorRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup test database
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    // Run migrations
    runMigrations(t, db)

    // Test repository
    repo := NewPostgresMonitorRepository(db)

    // Create
    monitor, err := repo.Create(context.Background(), &Monitor{...})
    require.NoError(t, err)

    // Read
    got, err := repo.Get(context.Background(), monitor.ID)
    require.NoError(t, err)
    require.Equal(t, monitor, got)
}
```

#### Queue Integration

```go
func TestRabbitMQProducer_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup test RabbitMQ
    conn := setupTestRabbitMQ(t)
    defer teardownTestRabbitMQ(t, conn)

    // Test producer
    producer := NewRabbitMQProducer(conn, "test.queue")

    // Publish
    err := producer.Publish(context.Background(), []byte("test message"))
    require.NoError(t, err)

    // Consume and verify
    msg, err := consumeMessage(t, conn, "test.queue")
    require.NoError(t, err)
    require.Equal(t, "test message", string(msg))
}
```

### 3.3 E2E Tests (godog scenarios)

Critical user journeys MUST be tested with godog:

#### Scenario Definition

```gherkin
@use_case=uc_01_01_01
@critical
Scenario: Create monitor and verify status
  Given user is authenticated
  And user has subscription "Free"
  When user creates monitor with:
    | name | Test Monitor |
    | url  | https://example.com |
  Then monitor is created successfully
  And monitor has status "PENDING"
```

#### Step Implementation

```go
func InitializeScenario(ctx *godog.ScenarioContext) {
    ctx.Given(`^user is authenticated$`, userIsAuthenticated)
    ctx.When(`^user creates monitor with:$`, userCreatesMonitor)
    ctx.Then(`^monitor is created successfully$`, monitorIsCreated)
}
```

### 3.4 Mock Strategies

#### Interface Mocking

- Use interfaces for all dependencies
- Use `gomock` or `testify/mock` for mocks
- Mock external dependencies (database, queue, HTTP clients)
- Avoid mocking domain logic

```go
//go:generate mockgen -source=monitor_repository.go -destination=mock_monitor_repository.go

type MockMonitorRepository struct {
    mock.Mock
}

func (m *MockMonitorRepository) Create(ctx context.Context, monitor *Monitor) (*Monitor, error) {
    args := m.Called(ctx, monitor)
    return args.Get(0).(*Monitor), args.Error(1)
}
```

#### Test Fixtures

- Define test fixtures for common test data
- Use factory pattern for creating test entities
- Keep fixtures in `testdata/` directory

```go
// testdata/monitor_fixtures.go
package testdata

func NewTestMonitor(opts ...func(*Monitor)) *Monitor {
    monitor := &Monitor{
        ID:   uuid.New().String(),
        Name: "Test Monitor",
        URL:  "https://example.com",
    }

    for _, opt := range opts {
        opt(monitor)
    }

    return monitor
}

func WithName(name string) func(*Monitor) {
    return func(m *Monitor) { m.Name = name }
}

func WithURL(url string) func(*Monitor) {
    return func(m *Monitor) { m.URL = url }
}
```

---

## Database Requirements (MUST)

### 4.1 Data Types

#### JSON Storage

- **DO NOT use JSONB**: This project uses TEXT type for JSON storage
- Use TEXT column to store JSON data
- Validate JSON at application layer before storage
- Document JSON schema in code comments
- Use Go structs for JSON marshaling/unmarshaling

```sql
-- GOOD: TEXT for JSON storage
CREATE TABLE monitors (
    id UUID PRIMARY KEY,
    config TEXT NOT NULL  -- JSON stored as TEXT
);

-- BAD: JSONB - NOT USED IN THIS PROJECT
CREATE TABLE monitors (
    id UUID PRIMARY KEY,
    config JSONB NOT NULL  -- DON'T USE
);
```

### 4.2 Goose Migrations

All database schema changes MUST use Goose migrations:

#### Migration Directory Structure

```
service-name/
└── infrastructure/
    └── postgres/
        └── migrations/
            ├── 20240314000000_create_monitors_table.sql
            ├── 20240314000001_create_users_table.sql
            └── ...
```

#### Migration Format

```sql
-- +goose Up
CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitors_user_id ON monitors(user_id);
CREATE INDEX idx_monitors_status ON monitors(status);

-- +goose Down
DROP INDEX IF EXISTS idx_monitors_status;
DROP INDEX IF EXISTS idx_monitors_user_id;
DROP TABLE IF EXISTS monitors;
```

#### Migration Rules

- Every migration MUST have Up and Down sections
- Use YYYYMMDDHHMMSS timestamps in migration filenames
- Run migrations in transaction for atomicity
- Add indexes for frequently queried columns
- Use foreign keys with appropriate actions (CASCADE, RESTRICT)

### 4.3 Transaction Handling

Database operations MUST use transactions appropriately:

#### Transaction Pattern

```go
func (r *PostgresMonitorRepository) CreateWithAlert(ctx context.Context, monitor *Monitor, alert *Alert) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer func() {
        if err != nil {
            tx.Rollback()
        }
    }()

    // Create monitor
    if err := r.createInTx(ctx, tx, monitor); err != nil {
        return err
    }

    // Create alert
    if err := r.createAlertInTx(ctx, tx, alert); err != nil {
        return err
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %v", err)
    }

    return nil
}
```

#### Transaction Scenarios

- Multiple related writes (e.g., create monitor + create events)
- Updates that must be atomic
- Complex business logic with multiple tables
- Cross-service operations (sagas pattern)

### 4.4 Connection Pooling

All services MUST use pure-golang/adapters database adapters with built-in connection pooling:

#### Adapter Configuration

- Use pure-golang PostgreSQL adapter from `git@github.com:pure-golang/adapters.git`
- Connection pool settings are pre-configured in the adapter
- Adapter handles connection pooling automatically
- Configure connection string and adapter options

#### Pool Settings

Connection pool is managed by pure-golang adapter with optimized settings:
- Automatic connection pool management
- Connection validation on checkout
- Graceful connection handling
- Configurable pool size limits

### 4.5 Query Optimization

Queries MUST be optimized for performance:

#### Indexing Strategy

- Add indexes on foreign keys
- Add indexes on frequently filtered columns (status, user_id)
- Add composite indexes for multi-column queries
- Use EXPLAIN ANALYZE to verify query plans

#### Query Patterns

- Use parameterized queries (no string concatenation)
- Use prepared statements for frequently executed queries
- Limit result sets with LIMIT
- Use pagination for large result sets

```go
func (r *PostgresMonitorRepository) List(ctx context.Context, userID string, limit, offset int) ([]*Monitor, error) {
    query := `
        SELECT id, name, url, status, created_at, updated_at
        FROM monitors
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

    rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to list monitors: %v", err)
    }
    defer rows.Close()

    var monitors []*Monitor
    for rows.Next() {
        var m Monitor
        if err := rows.Scan(&m.ID, &m.Name, &m.URL, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan monitor: %v", err)
        }
        monitors = append(monitors, &m)
    }

    return monitors, nil
}
```

---

## Queue Requirements (MUST)

### 5.1 PureGolang RMQ Adapters

All RabbitMQ operations MUST use pure-golang/adapters:

#### Adapter Features

- Connection management with automatic retry and backoff
- Built-in observability (metrics, traces, logging)
- Structured logging integration
- Message persistence and delivery guarantees
- Dead letter queue support
- Connection pooling and health checks

#### Adapter Configuration

- Use pure-golang adapters from `git@github.com:pure-golang/adapters.git`
- Configure connection retries and backoff
- Use observability integration (metrics, traces)
- Use structured logging adapter

### 5.2 Producer/Consumer Patterns

All queue operations MUST use pure-golang/adapters RMQ adapters:

#### Producer Pattern

- Use pure-golang RMQ producer for message publishing
- Configure queue and routing key
- Set message persistence and headers
- Adapter handles retries and error handling

#### Consumer Pattern

- Use pure-golang RMQ consumer for message processing
- Configure queue and dead letter queue
- Implement message handler function
- Adapter handles acknowledgment and reconnection

### 5.3 Dead Letter Queue

Failed messages MUST be handled with dead letter queue:

#### DLQ Configuration

- Configure DLQ in pure-golang/adapters RMQ adapter settings
- Set max retry count and TTL
- Configure queue routing for failed messages

#### DLQ Monitoring

- Log DLQ size periodically
- Alert ops team when DLQ size > 1000
- Provide admin interface for replay from DLQ
- Investigate DLQ messages for systemic issues

### 5.4 Error Handling

Queue error handling MUST follow these rules:

#### Retryable Errors

- Connection timeouts
- Network errors
- Service unavailable
- Rate limit exceeded

#### Non-Retryable Errors

- Invalid message format
- Validation errors
- Authentication errors
- Permission errors

#### Retry Logic

```go
func (c *Consumer) handleMessage(ctx context.Context, msg rmq.Message) error {
    maxRetries := 3
    backoff := time.Second

    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := c.service.Process(ctx, msg)
        if err == nil {
            return nil
        }

        if isNonRetryable(err) {
            c.logger.Error("non-retryable error", "error", err)
            return err
        }

        c.logger.Warn("retryable error",
            "attempt", attempt,
            "max_retries", maxRetries,
            "error", err,
        )

        if attempt < maxRetries {
            time.Sleep(backoff)
            backoff *= 2
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

---

## Observability (MUST)

### 6.1 PureGolang Platform for Observability

Use pure-golang/platform for monitoring and observability:

#### Platform Repository

Location: `git@git@github.com:pure-golang/platform.git`

#### Available Monitoring Tools

Check platform for:
- OpenTelemetry integrations (traces, metrics, logs)
- Prometheus middleware and instrumentation
- Jaeger tracing exporters
- Structured logging adapters
- Health check frameworks
- Alert rule definitions

#### Usage Pattern

```go
// GOOD: Use platform monitoring
import "githib.com/pure-golang/newbackend/platform/observability"

// Use platform middleware, adapters, instrumentation
```

#### Custom Monitoring

Only implement custom monitoring when:
- Platform doesn't provide required functionality
- Service-specific metrics needed
- Business logic requires custom instrumentation

### 6.2 Observability Middleware

All observability MUST use pure-golang/platform and pure-golang/adapters with built-in middleware:

#### Platform Features

Use pure-golang/platform for:
- Automatic OpenTelemetry integration (traces, metrics, logs)
- Structured JSON logging with trace context propagation
- Prometheus metrics with standard naming conventions
- Distributed tracing with Jaeger/OTLP export

#### Adapters Features

Use pure-golang/adapters for:
- Built-in middleware for gRPC and HTTP servers
- Request/response logging with trace ID
- Metrics collection (request count, latency, error rate)
- Error handling and recovery
- Timeout and deadline enforcement

#### Middleware Integration

Services MUST use pure-golang/platform and pure-golang/adapters middleware:

**gRPC Middleware:**
- Request logging with trace ID
- Metrics collection (request count, latency, error rate)
- Error handling and recovery
- Timeout and deadline enforcement

**HTTP Middleware:**
- Request/response logging
- Metrics collection
- CORS handling
- Error handling and recovery
- Rate limiting support

### 6.3 Metrics

All services MUST collect metrics:

#### Required Metrics

- Request count by endpoint and status code
- Request latency (p50, p95, p99)
- Database query count and duration
- Queue message count and processing time
- Error rate by type
- Active connections
- Memory and CPU usage

#### Metric Naming

- Use dot notation: `monitor.requests.total`
- Use consistent units: seconds, bytes, count
- Include labels for dimensions: `monitor.requests.total{status="200", method="GET"}`

```go
func (s *Service) recordMetrics(ctx context.Context, status string, duration time.Duration) {
    s.meter.RecordCounter(ctx, "monitor.requests.total",
        otel.WithAttributes("status", status),
    )

    s.meter.RecordHistogram(ctx, "monitor.request.duration",
        duration.Seconds(),
        otel.WithAttributes("status", status),
    )
}
```

### 6.4 Tracing

All requests MUST be traced end-to-end:

#### Trace Propagation

- Extract trace context from incoming requests (gRPC metadata)
- Propagate trace context through service calls (outgoing gRPC, database, queues)
- Add spans for significant operations (database queries, HTTP calls, queue operations)

```go
func (s *Service) CreateMonitor(ctx context.Context, req *CreateMonitorRequest) (*Monitor, error) {
    ctx, span := s.tracer.Start(ctx, "MonitorService.CreateMonitor")
    defer span.End()

    // Database operation
    monitor, err := s.repo.Create(ctx, req)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // Queue operation
    err = s.queueProducer.Publish(ctx, monitor)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    return monitor, nil
}
```

#### Span Attributes

- Include operation name
- Include request/response attributes
- Include error information on failures
- Add custom attributes for business logic

### 6.5 Health Checks

All services MUST implement health checks:

#### Health Check Endpoint

```go
type HealthChecker struct {
    db    *sql.DB
    rmq   rmq.Connection
    tracer otel.Tracer
}

func (h *HealthChecker) Check(ctx context.Context) error {
    ctx, span := h.tracer.Start(ctx, "HealthCheck.Check")
    defer span.End()

    // Check database
    if err := h.db.Ping(); err != nil {
        span.RecordError(err)
        return fmt.Errorf("database unhealthy: %v", err)
    }

    // Check RabbitMQ
    if err := h.rmq.Check(); err != nil {
        span.RecordError(err)
        return fmt.Errorf("rabbitmq unhealthy: %v", err)
    }

    return nil
}
```

#### Liveness and Readiness

- Liveness: Service is running (no deadlocks)
- Readiness: Service is ready to accept requests (dependencies connected)
- Both checks MUST use timeouts (5 seconds)

---

## Security Requirements (MUST)

### 7.1 Input Validation

All inputs MUST be validated:

#### Request Validation

```go
func validateCreateMonitorRequest(req *CreateMonitorRequest) error {
    if req.Name == "" {
        return errors.New("name is required")
    }

    if len(req.Name) > 255 {
        return errors.New("name too long")
    }

    if _, err := url.ParseRequestURI(req.URL); err != nil {
        return fmt.Errorf("invalid URL: %v", err)
    }

    if req.IntervalSeconds < 30 || req.IntervalSeconds > 86400 {
        return errors.New("interval must be between 30s and 24h")
    }

    return nil
}
```

#### SQL Injection Prevention

- Use parameterized queries (no string concatenation)
- Use prepared statements
- Validate and sanitize inputs
- Use ORMs with built-in protection

```go
// Safe: parameterized query
query := `SELECT * FROM monitors WHERE user_id = $1 AND status = $2`
rows, err := db.Query(query, userID, status)

// Unsafe: string concatenation
query := fmt.Sprintf("SELECT * FROM monitors WHERE user_id = '%s'", userID) // DON'T DO THIS
```

### 7.2 Authentication and Authorization

#### JWT Authentication

```go
func (a *Authenticator) ValidateToken(ctx context.Context, token string) (*Claims, error) {
    // Parse token
    claims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return a.secret, nil
    })

    if err != nil {
        return nil, fmt.Errorf("invalid token: %v", err)
    }

    // Check expiration
    if claims.ExpiresAt.Time.Before(time.Now()) {
        return nil, errors.New("token expired")
    }

    return claims, nil
}
```

#### RBAC Authorization

```go
func (a *Authorizer) CanCreateMonitor(user *User) error {
    // Check user tier limits
    limits := a.tierLimits[user.Tier]
    if user.MonitorCount >= limits.MaxMonitors {
        return fmt.Errorf("monitor limit exceeded: %d/%d", user.MonitorCount, limits.MaxMonitors)
    }

    return nil
}
```

### 7.3 Secrets Management

All secrets MUST be managed securely:

#### Environment Variables

- Use environment variables for secrets
- Never hardcode secrets in code
- Use .env.example for required variables
- Keep .env in .gitignore

#### Secret Storage

- Use Kubernetes secrets for production
- Rotate secrets regularly
- Use secret management service (HashiCorp Vault, AWS Secrets Manager)

---

## Configuration Requirements (MUST)

### 8.1 .env.example Per Service

Each service MUST have .env.example with all required variables:

#### .env.example Template

```env
# Server
SERVER_PORT=8080
SERVER_GRPC_PORT=9090

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=milan
DB_USER=postgres
DB_PASSWORD=password
DB_SSL_MODE=require

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE=monitors
RABBITMQ_QUEUE=alerts.queue
RABBITMQ_DLQ=alerts.dlq

# JWT
JWT_SECRET=your-secret-key-here
JWT_ACCESS_DURATION=15m
JWT_REFRESH_DURATION=168h

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=monitor-service
OTEL_LOG_LEVEL=info

# External Services
YANDEX_SHOP_ID=your-shop-id
YANDEX_SECRET_KEY=your-secret-key
YANDEX_WEBHOOK_SECRET=your-webhook-secret

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password

TELEGRAM_BOT_TOKEN=your-bot-token
```

### 8.2 Root .env File

Project root MUST have .env file:

```env
# Shared environment variables
DB_HOST=localhost
DB_PORT=5432
DB_NAME=milan
DB_USER=postgres
DB_PASSWORD=postgres

RABBITMQ_URL=amqp://guest:guest@localhost:5672/

JWT_SECRET=super-secret-key-change-in-production

OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

### 8.3 Dockerfile Per Service

Each service MUST have Dockerfile for Kubernetes deployment:

#### Dockerfile Requirements

- Multi-stage build for minimal image size
- Use official Go base image
- Copy only necessary files (no .git, test files, docs)
- Set non-root user
- Configure health checks
- Expose gRPC port

#### Dockerfile Template

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service-name ./cmd/service-name

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/service-name .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Use non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /root
USER appuser

# Expose gRPC port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD grpc_health_probe -addr=:9090 || exit 1

# Run binary
CMD ["./service-name"]
```

#### Docker Build Requirements

- Use `grpc_health_probe` for health checks (install in Dockerfile or use HTTP endpoint)
- Include `migrations/` directory for database setup
- Support configuration via environment variables
- Use semantic versioning for image tags

### 8.4 Service-Level docker-compose.yml

Each service MUST have docker-compose.yml for local debugging:

#### Requirements

- Include service container built from Dockerfile
- Include all necessary infrastructure (postgres, rabbitmq, redis, etc.)
- Exclude other application services from the project
- Support environment variables from .env file
- Configure volume mounts for live reload (optional)
- Set up health checks

#### docker-compose.yml Template

```yaml
version: '3.8'

services:
  # Service container
  monitor-service:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: monitor-service
    ports:
      - "9090:9090"  # gRPC
      - "8080:8080"  # HTTP (if applicable)
    environment:
      # Server
      SERVER_PORT: 8080
      SERVER_GRPC_PORT: 9090

      # Database
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: milan
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_SSL_MODE: disable

      # RabbitMQ
      RABBITMQ_URL: amqp://guest:guest@rabbitmq:5672/
      RABBITMQ_EXCHANGE: monitors
      RABBITMQ_QUEUE: alerts.queue

      # Observability
      OTEL_EXPORTER_OTLP_ENDPOINT: http://jaeger:4318
      OTEL_SERVICE_NAME: monitor-service
      OTEL_LOG_LEVEL: debug
    depends_on:
      postgres:
        condition: service_healthy
      rabbitmq:
        condition: service_healthy
    networks:
      - milan-network

  # PostgreSQL database
  postgres:
    image: postgres:15-alpine
    container_name: milan-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: milan
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - milan-network

  # RabbitMQ message broker
  rabbitmq:
    image: rabbitmq:3-management-alpine
    container_name: milan-rabbitmq
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    ports:
      - "5672:5672"
      - "15672:15672"
    volumes:
      - rabbitmq-data:/var/lib/rabbitmq
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - milan-network

  # Jaeger tracing
  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: milan-jaeger
    ports:
      - "16686:16686"  # UI
      - "14250:14250"  # gRPC
    environment:
      COLLECTOR_OTLP_ENABLED: true
    networks:
      - milan-network

  # Redis (if needed)
  redis:
    image: redis:7-alpine
    container_name: milan-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5
    networks:
      - milan-network

volumes:
  postgres-data:
  rabbitmq-data:
  redis-data:

networks:
  milan-network:
    driver: bridge
```

#### Usage

```bash
# Build and start all services
docker-compose up --build

# Start in detached mode
docker-compose up -d

# View logs
docker-compose logs -f monitor-service

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### 8.5 Root docker-compose.yml

Project root MUST have docker-compose.yml for local development with shared infrastructure:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: milan-postgres
    environment:
      POSTGRES_USER: ${DB_USER:-postgres}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-postgres}
      POSTGRES_DB: ${DB_NAME:-milan}
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-postgres}"]
      interval: 10s
      timeout: 5s
      retries: 5

  rabbitmq:
    image: rabbitmq:3-management-alpine
    container_name: milan-rabbitmq
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    ports:
      - "5672:5672"
      - "15672:15672"
    volumes:
      - rabbitmq-data:/var/lib/rabbitmq
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: milan-jaeger
    ports:
      - "16686:16686"  # UI
      - "14250:14250"  # gRPC
    environment:
      COLLECTOR_OTLP_ENABLED: true

volumes:
  postgres-data:
  rabbitmq-data:
```

### 8.6 Environment Variable Naming

Environment variables MUST follow naming convention:

- Use UPPER_CASE with underscores
- Prefix with service name for service-specific variables
- Use descriptive names for clarity
- Group related variables with common prefix

```env
# Good
MONITOR_SERVICE_PORT=9090
MONITOR_SERVICE_DB_HOST=localhost
MONITOR_SERVICE_RABBITMQ_URL=amqp://...

# Bad
port=9090
db_host=localhost
url=amqp://...
```

### 8.7 Configuration Validation

All services MUST validate configuration on startup:

```go
type Config struct {
    ServerPort      int    `env:"SERVER_PORT" envDefault:"8080" validate:"required,min=1,max=65535"`
    DBHost          string `env:"DB_HOST" envDefault:"localhost" validate:"required"`
    DBPort          int    `env:"DB_PORT" envDefault:"5432" validate:"required,min=1,max=65535"`
    DBName          string `env:"DB_NAME" envDefault:"milan" validate:"required"`
    DBUser          string `env:"DB_USER" envDefault:"postgres" validate:"required"`
    DBPassword      string `env:"DB_PASSWORD" validate:"required,min=8"`
    RabbitMQURL     string `env:"RABBITMQ_URL" envDefault:"amqp://guest:guest@localhost:5672/" validate:"required,url"`
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}

    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %v", err)
    }

    if err := validator.New().Struct(cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %v", err)
    }

    return cfg, nil
}
```

---

## Performance Requirements (SHOULD)

### 9.1 Benchmarks

Critical operations SHOULD have benchmarks:

```go
func BenchmarkMonitorService_CreateMonitor(b *testing.B) {
    service := setupMonitorService(b)

    req := &CreateMonitorRequest{
        Name: "Benchmark Monitor",
        URL:  "https://example.com",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.CreateMonitor(context.Background(), req)
    }
}
```

### 9.2 Response Time Targets

API endpoints SHOULD meet response time targets:

- GET requests: < 100ms (p95)
- POST requests: < 200ms (p95)
- PUT requests: < 200ms (p95)
- DELETE requests: < 100ms (p95)

### 9.3 Resource Limits

Services SHOULD have resource limits:

```yaml
apiVersion: v1
kind: Deployment
metadata:
  name: monitor-service
spec:
  template:
    spec:
      containers:
      - name: monitor-service
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

---

## Quality Metrics

### 10.1 Code Coverage

Target coverage metrics:

| Layer | Target | Minimum |
|-------|--------|---------|
| Domain | 90% | 85% |
| Application | 85% | 80% |
| Infrastructure | 75% | 70% |
| Interfaces | 70% | 65% |
| Overall | 80% | 75% |

### 10.2 Linting

All code MUST pass linting:

```bash
# Run golangci-lint
golangci-lint run ./...

# Enable linters
- gofmt
- govet
- staticcheck
- errcheck
- gosimple
- ineffassign
- unused
```

### 10.3 Security Scanning

All code MUST pass security scanning:

```bash
# Run gosec
gosec ./...

# Check for common vulnerabilities:
- SQL injection
- Command injection
- Cross-site scripting (XSS)
- Path traversal
- Insecure cryptography
```

---

## Checklists

### 11.1 Before Commit

- [ ] Code follows clean architecture patterns
- [ ] Code uses pure-golang adapters for external connectors
- [ ] gRPC endpoints reference use cases from feature files
- [ ] HTTP gateway annotations present (google.api.http)
- [ ] Error responses use UPPERCASE_SNAKE_CASE codes
- [ ] All tests pass (unit, integration, E2E)
- [ ] Code coverage > 80%
- [ ] Linting passes (golangci-lint)
- [ ] Security scan passes (gosec)
- [ ] Logging is structured JSON format
- [ ] Tracing is implemented with OpenTelemetry
- [ ] Configuration is validated on startup
- [ ] .env.example is updated with new variables
- [ ] Dockerfile is present and tested
- [ ] docker-compose.yml is present and tested
- [ ] No TODO comments or stub functions in code

### 11.2 Before Pull Request

- [ ] All before commit items checked
- [ ] PR description references related issues
- [ ] PR includes unit tests for new functionality
- [ ] PR includes integration tests for database/queue operations
- [ ] PR includes E2E tests for user-facing features
- [ ] PR documentation is clear and complete
- [ ] CI/CD pipeline passes
- [ ] Code review approved by at least one reviewer
- [ ] Performance benchmarks run and documented
- [ ] Security review completed for sensitive changes
- [ ] Dockerfile builds successfully
- [ ] docker-compose starts all services correctly
- [ ] No TODO code or stubs in changed files

### 11.3 Before Release

- [ ] All before PR items checked
- [ ] Release notes prepared
- [ ] Breaking changes documented
- [ ] Migration scripts tested
- [ ] Database migrations reviewed
- [ ] Monitoring dashboards updated
- [ ] Alert rules configured
- [ ] Rollback plan documented
- [ ] Load testing completed
- [ ] Security audit completed
- [ ] Compliance review completed (152-ФЗ)
- [ ] Docker images pushed to registry
- [ ] docker-compose tested for local debugging
- [ ] No TODO code or stubs remaining

---

## Notes

### PureGolang Adapters Repository

Location: `git@git@github.com:pure-golang/adapters.git`

**Primary source for infrastructure adapters**:
- Database connections (PostgreSQL, Redis)
- Message queues (RabbitMQ)
- HTTP servers and clients
- External service integrations
- Connection pooling and retry logic

**Always check adapters before implementing**:
- Database repositories
- Queue producers/consumers
- HTTP middleware
- Service connections and ports

### PureGolang Platform Repository

Location: `git@git@github.com:pure-golang/platform.git`

**Primary source for monitoring and observability**:
- OpenTelemetry integration (traces, metrics, logs)
- Prometheus instrumentation
- Jaeger tracing exporters
- Structured logging frameworks
- Health check utilities
- Alert rule definitions

**Always check platform before implementing**:
- Custom logging
- Metrics collection
- Distributed tracing
- Health check endpoints
- Monitoring dashboards

### Infrastructure Directory Purpose

The `internal/infrastructure/` directory is for:
- Service-specific configuration wiring
- Adapters/platform initialization and wiring
- Interface implementations specific to this service
- NOT for re-implementing existing adapters or platform components

**Usage**: Always check pure-golang/adapters and pure-golang/platform before implementing custom solutions.

### Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 2.2 | 2026-03-27 | Added pure-golang/adapters and pure-golang/platform separation, JSON→TEXT storage rule, YYYYMMDDHHMMSS migrations, TODO/stub checks | User & Claude |
| 2.1 | 2026-03-27 | Added Dockerfile for k8s and service-level docker-compose for local debugging | User & Claude |
| 2.0 | 2026-03-15 | Refactored structure to model/service/handler pattern (kopru/calls style) | User & Claude |
| 1.3 | 2026-03-14 | Moved migrations/ directory to service root for easier management | Claude & User |
| 1.2 | 2026-03-14 | Removed code examples, added middleware integration, removed 152-ФЗ Compliance section, simplified connection pooling with adapters | Claude & User |
| 1.1 | 2026-03-14 | Fixed error wrapping (%w → %v), corrected use case comment placement (method level, service level epic/us) | Claude & User |
| 1.0 | 2026-03-14 | Initial Backend DoD for Milan monitoring service | Claude & User |

---

**Document Location:** `/Users/raul/conductor/workspaces/monitor/vancouver/backend/DOD.md`
**Last Updated:** 2026-03-27
**Next Review:** 2026-06-27
