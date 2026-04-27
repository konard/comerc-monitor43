# Microservices Architecture

## Состав микросервисов для v1.0 MVP

1. **API Gateway** - Единая точка входа
2. **Monitor Service** - Управление мониторами
3. **Alert Service** - Отправка уведомлений
4. **Billing Service** - Подписки и платежи
5. **Scheduler Service** - Планирование проверок
6. **Check Workers** - Выполнение проверок

---

## 1. API Gateway

### Назначение
Единая точка входа для всех client requests. Аутентификация, rate limiting, маршрутизация.

### Ответственности

#### Core Functions
- **Authentication:** JWT validation (stateless)
- **Authorization:** RBAC по тарифам
- **Rate Limiting:** In-memory, per-user
- **Request Routing:** gRPC proxy to services
- **TLS Termination:** HTTPS → HTTP
- **Observability:** Logging, metrics, tracing

#### Endpoints
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout

GET    /api/v1/monitors
POST   /api/v1/monitors
GET    /api/v1/monitors/:id
PUT    /api/v1/monitors/:id
DELETE /api/v1/monitors/:id

GET    /api/v1/alerts/channels
POST   /api/v1/alerts/channels
DELETE /api/v1/alerts/channels/:id

GET    /api/v1/billing/subscription
POST   /api/v1/billing/subscribe
POST   /api/v1/billing/payment

GET    /api/v1/health
```

### Технологии
- **Language:** Go
- **Framework:** Custom gRPC proxy
- **Auth:** JWT (github.com/golang-jwt/jwt)
- **Rate Limiter:** In-memory (uber-go/ratelimit)
- **Observability:** OpenTelemetry (pure-golang adapters)

### Конфигурация
```yaml
server:
  port: 8080
  grpc_port: 9090

jwt:
  secret: ${JWT_SECRET}
  access_duration: 15m
  refresh_duration: 168h  # 7 days

rate_limit:
  requests_per_second: 100
  burst: 200

services:
  monitor: "monitor-service:9090"
  alert: "alert-service:9091"
  billing: "billing-service:9092"
  scheduler: "scheduler-service:9093"
```

---

## 2. Monitor Service

### Назначение
CRUD операции для мониторов, хранение состояний, обработка результатов проверок.

### Ответственности

#### Core Functions
- **Monitor CRUD:** Создание, чтение, обновление, удаление
- **State Management:** Хранение last state
- **Event Logging:** Запись изменений в `monitor_events`
- **Alert Triggering:** Отправка алертов при смене статуса
- **Validation:** Проверка лимитов по тарифу

#### gRPC API
```protobuf
service MonitorService {
  rpc CreateMonitor(CreateMonitorRequest) returns (Monitor);
  rpc GetMonitor(GetMonitorRequest) returns (Monitor);
  rpc ListMonitors(ListMonitorsRequest) returns (ListMonitorsResponse);
  rpc UpdateMonitor(UpdateMonitorRequest) returns (Monitor);
  rpc DeleteMonitor(DeleteMonitorRequest) returns (google.protobuf.Empty);

  rpc ReportCheckResult(ReportCheckResultRequest) returns (google.protobuf.Empty);
  rpc GetMonitorHistory(GetMonitorHistoryRequest) returns (GetMonitorHistoryResponse);
}
```

### Технологии
- **Language:** Go
- **Framework:** gRPC + protobuf
- **Database:** PostgreSQL (github.com/lib/pq)
- **Migration:** Goose
- **Observability:** OpenTelemetry

### Конфигурация
```yaml
server:
  port: 9090

database:
  host: ${DB_HOST}
  port: 5432
  name: milan
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  ssl_mode: require

rabbitmq:
  url: ${RABBITMQ_URL}
  alerts_queue: alerts.queue

monitor_limits:
  free: 25
  starter: 50
  professional: 150
  business: 500
```

---

## 3. Alert Service

### Назначение
Отправка уведомлений пользователям через разные каналы.

### Ответственности

#### Core Functions
- **Channel Management:** CRUD для Email, Telegram, Webhooks
- **Alert Delivery:** Отправка уведомлений
- **Rate Limiting:** Hybrid (per-monitor + daily limits)
- **Retry Logic:** 3x с exponential backoff
- **Template Rendering:** HTML для Email, Markdown для Telegram

#### gRPC API
```protobuf
service AlertService {
  rpc CreateAlertChannel(CreateAlertChannelRequest) returns (AlertChannel);
  rpc ListAlertChannels(ListAlertChannelsRequest) returns (ListAlertChannelsResponse);
  rpc DeleteAlertChannel(DeleteAlertChannelRequest) returns (google.protobuf.Empty);

  rpc SendAlert(SendAlertRequest) returns (SendAlertResponse);
  rpc TestAlertChannel(TestAlertChannelRequest) returns (TestAlertChannelResponse);
}
```

#### RabbitMQ Consumers
```go
// Alerts from Monitor Service
func (s *AlertService) ConsumeAlerts() {
    // 1. Parse alert
    // 2. Check rate limits
    // 3. Send to channels
    // 4. Retry on failure
    // 5. Log delivery
}
```

### Технологии
- **Language:** Go
- **Framework:** gRPC + RabbitMQ (github.com/rabbitmq/amqp091-go)
- **Email:** SMTP client
- **Telegram:** Bot API HTTP client
- **Webhooks:** HTTP client with retry

### Конфигурация
```yaml
server:
  port: 9091

rabbitmq:
  url: ${RABBITMQ_URL}
  alerts_queue: alerts.queue
  notifications_queue: notifications.queue

email:
  smtp_host: ${SMTP_HOST}
  smtp_port: 587
  smtp_user: ${SMTP_USER}
  smtp_password: ${SMTP_PASSWORD}

telegram:
  bot_token: ${TELEGRAM_BOT_TOKEN}
  api_url: https://api.telegram.org

rate_limits:
  per_monitor: 1 per 5 minutes
  daily_per_user: 100

retry:
  max_attempts: 3
  initial_delay: 1s
  max_delay: 60s
```

---

## 4. Billing Service

### Назначение
Управление подписками, платежами, интеграция с ЮKassa.

### Ответственности

#### Core Functions
- **Subscription Management:** Создание, обновление, отмена
- **Payment Processing:** ЮKassa integration
- **Webhook Handling:** Обработка платежей от ЮKassa
- **Limit Enforcement:** Проверка лимитов мониторов
- **Invoice Generation:** Выставление счетов

#### gRPC API
```protobuf
service BillingService {
  rpc CreateSubscription(CreateSubscriptionRequest) returns (Subscription);
  rpc GetSubscription(GetSubscriptionRequest) returns (Subscription);
  rpc CancelSubscription(CancelSubscriptionRequest) returns (google.protobuf.Empty);

  rpc CreatePayment(CreatePaymentRequest) returns (Payment);
  rpc GetPayment(GetPaymentRequest) returns (Payment);

  rpc HandleYandexWebhook(HandleYandexWebhookRequest) returns (google.protobuf.Empty);
  rpc CheckUserLimits(CheckUserLimitsRequest) returns (CheckUserLimitsResponse);
}
```

### ЮKassa Integration

#### Payment Methods
- **СБП:** QR code на мобильное приложение
- **Банковские карты:** Visa, Mastercard, МИР
- **Платежные поручения:** Для B2B

#### API Endpoints
```
POST https://api.yookassa.ru/v3/payments
Authorization: Basic <shop_id>:<secret_key>
Content-Type: application/json

{
  "amount": {
    "value": "500.00",
    "currency": "RUB"
  },
  "payment_method_data": {
    "type": "sbp"
  },
  "confirmation": {
    "type": "qr",
    "return_url": "https://milan.ru/payment/success"
  }
}
```

#### Webhook Verification
```go
func VerifyWebhook(signature string, payload []byte) bool {
    // 1. Extract signature from header
    // 2. Compute HMAC-SHA256 with secret key
    // 3. Compare signatures
}
```

### Технологии
- **Language:** Go
- **Framework:** gRPC
- **Payment:** ЮKassa REST API
- **Database:** PostgreSQL

### Конфигурация
```yaml
server:
  port: 9092

yandex_kassa:
  shop_id: ${YANDEX_SHOP_ID}
  secret_key: ${YANDEX_SECRET_KEY}
  api_url: https://api.yookassa.ru/v3
  webhook_secret: ${YANDEX_WEBHOOK_SECRET}

subscriptions:
  free:
    monitors: 25
    interval_min: 300
    price: 0
  starter:
    monitors: 50
    interval_min: 120
    price: 50000  # 500₽ в копейках
  professional:
    monitors: 150
    interval_min: 60
    price: 250000  # 2500₽
  business:
    monitors: 500
    interval_min: 30
    price: 750000  # 7500₽
```

---

## 5. Scheduler Service

### Назначение
Планирование проверок мониторов, предоставление расписания worker'ам.

### Ответственности

#### Core Functions
- **Schedule Calculation:** next_check_time для каждого монитора
- **Priority Management:** Business > Pro > Starter > Free
- **Load Balancing:** Распределение проверок во времени
- **gRPC Endpoint:** Предоставление расписания worker'ам

#### gRPC API
```protobuf
service SchedulerService {
  rpc GetSchedule(GetScheduleRequest) returns (GetScheduleResponse);
  rpc RegisterWorker(RegisterWorkerRequest) returns (google.protobuf.Empty);
  rpc Heartbeat(HeartbeatRequest) returns (google.protobuf.Empty);
}
```

#### Algorithm
```go
// Calculate next check time based on:
// 1. Monitor interval (30s, 1m, 2m, 5m, 10m, 30m)
// 2. Current time
// 3. User tier (priority)
// 4. Worker availability

func (s *Scheduler) GetSchedule(req *GetScheduleRequest) (*GetScheduleResponse, error) {
    // 1. Query DB for due monitors
    // 2. Filter by worker capabilities
    // 3. Sort by priority (tier, interval)
    // 4. Return batch of monitors to check
}
```

### Технологии
- **Language:** Go
- **Framework:** gRPC
- **Database:** PostgreSQL

### Конфигурация
```yaml
server:
  port: 9093

database:
  host: ${DB_HOST}
  port: 5432
  name: milan
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  ssl_mode: require

schedule:
  batch_size: 100
  look_ahead_minutes: 5

priorities:
  business: 1
  professional: 2
  starter: 3
  free: 4
```

---

## 6. Check Workers

### Назначение
Выполнение HTTP/HTTPS проверок мониторов.

### Ответственности

#### Core Functions
- **Schedule Polling:** Запрос расписания у Scheduler
- **HTTP Checks:** Выполнение запросов
- **Response Analysis:** Проверка status codes, response time
- **Result Reporting:** Отчет в Monitor Service

#### Workflow
```go
func (w *Worker) Run() {
    for {
        // 1. Get schedule from Scheduler
        schedule := w.GetSchedule()

        // 2. Process each monitor
        for _, monitor := range schedule.Monitors {
            result := w.CheckMonitor(monitor)

            // 3. Report result to Monitor Service
            w.ReportResult(monitor.ID, result)
        }

        // 4. Wait before next poll
        time.Sleep(10 * time.Second)
    }
}

func (w *Worker) CheckMonitor(monitor *Monitor) *CheckResult {
    start := time.Now()

    // Execute HTTP request
    resp, err := w.httpClient.Get(monitor.URL)

    // Measure response time
    duration := time.Since(start)

    // Analyze response
    status := w.AnalyzeResponse(resp, err, monitor)

    return &CheckResult{
        Status:           status,
        ResponseTime:     duration,
        StatusCode:       resp.StatusCode,
        Error:            err,
        Timestamp:        time.Now(),
    }
}
```

### Autoscaling
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: check-workers-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: check-workers
  minReplicas: 2
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Технологии
- **Language:** Go
- **HTTP Client:** net/http with timeouts
- **gRPC Client:** Connection to Scheduler & Monitor Service

### Конфигурация
```yaml
scheduler:
  address: scheduler-service:9093

monitor_service:
  address: monitor-service:9090

check:
  timeout_seconds: 10
  max_redirects: 5
  user_agent: "Milon Monitor/1.0"

poll_interval: 10s
```

---

## Communication Patterns

### Synchronous (gRPC)
```
API Gateway → Monitor Service
API Gateway → Alert Service
API Gateway → Billing Service
Check Worker → Scheduler Service
Check Worker → Monitor Service
```

### Asynchronous (RabbitMQ)
```
Monitor Service → Alert Service (alerts.queue)
Alert Service → Notifications (notifications.queue)
Yandex Kassa → Billing Service (payment.webhooks)
```

---

## Service Discovery

### Kubernetes DNS
```
monitor-service:9090
alert-service:9091
billing-service:9092
scheduler-service:9093
```

### Health Checks
```go
// K8s liveness & readiness probes
func (s *Service) HealthCheck() error {
    // Check database connection
    if err := s.db.Ping(); err != nil {
        return err
    }

    // Check RabbitMQ connection
    if err := s.rabbitmq.Check(); err != nil {
        return err
    }

    return nil
}
```

---

**Дата:** 2026-03-08
**Версия:** 1.0
