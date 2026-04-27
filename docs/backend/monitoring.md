# Monitoring & Observability

## Стек технологий

**Observability:** OpenTelemetry (pure-golang adapters)
**Metrics:** Prometheus + Grafana
**Logs:** Structured JSON logging
**Traces:** Jaeger (опционально)

---

## OpenTelemetry Integration

### PureGolang Adapters

**Repository:** https://github.com/pure-golang/adapters

### Installation

```bash
go get github.com/pure-golang/adapters/observability
```

### Initialization

```go
package main

import (
    "github.com/pure-golang/adapters/observability"
)

func main() {
    // Initialize OpenTelemetry
    observability.Init(observability.Config{
        ServiceName:    "api-gateway",
        ServiceVersion: "1.0.0",
        Environment:    "production",

        // Tracing
        TracingEnabled: true,
        TracingEndpoint: "http://jaeger:14268/api/traces",

        // Metrics
        MetricsEnabled: true,
        MetricsEndpoint: "http://prometheus:9090",

        // Logging
        LoggingEnabled: true,
        LoggingLevel: "info",
        LoggingFormat: "json",
    })

    // Your application code
    runAPIGateway()
}
```

### Automatic Instrumentation

```go
// HTTP middleware
func (s *Server) middleware(next http.Handler) http.Handler {
    return observability.HTTPMiddleware(next)
}

// gRPC interceptor
func (s *Server) intercept() grpc.ServerOption {
    return observability.GRPCServerInterceptor()
}
```

---

## Metrics

### Business Metrics

#### 1. MTTD (Mean Time To Detect)
```go
var mttdHistogram = prometheus.NewHistogram(
    prometheus.HistogramOpts{
        Name: "milan_mttd_seconds",
        Help: "Time from incident to alert",
        Buckets: []float64{60, 180, 300, 600, 1800}, // 1m, 3m, 5m, 10m, 30m
    },
)
```

**Target:** < 10 мин (Launch), < 5 мин (Growth), < 3 мин (Scale)

#### 2. Alert Delivery Time
```go
var alertDeliveryDuration = prometheus.NewHistogram(
    prometheus.HistogramOpts{
        Name: "milan_alert_delivery_seconds",
        Help: "Time from trigger to delivery",
        Buckets: []float64{5, 10, 30, 60, 120}, // 5s, 10s, 30s, 1m, 2m
    },
)
```

**Target:** < 30 секунд

#### 3. Check Accuracy
```go
var checkAccuracy = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "milan_check_results_total",
        Help: "Total check results by status",
    },
    []string{"status"}, // up, down, error
)
```

**Target:** > 99% accuracy (false positives < 5%)

### Application Metrics

#### 4. Request Duration
```go
var httpRequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "milan_http_request_duration_seconds",
        Help: "HTTP request duration",
        Buckets: prometheus.DefBuckets,
    },
    []string{"method", "endpoint", "status"},
)
```

#### 5. Error Rate
```go
var httpErrorsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "milan_http_errors_total",
        Help: "Total HTTP errors",
    },
    []string{"method", "endpoint", "status"},
)
```

#### 6. Queue Depth
```go
var queueDepth = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "milan_queue_depth",
        Help: "Number of messages in queue",
    },
    []string{"queue_name"},
)
```

### System Metrics

#### 7. Resource Usage
```go
// CPU, Memory, Disk (collected by Prometheus node_exporter)
```

#### 8. Worker Lag
```go
var workerLag = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "milan_worker_lag_seconds",
        Help: "Time lag for workers behind schedule",
    },
    []string{"worker_id"},
)
```

---

## Dashboards

### Grafana Dashboard: Overview

```
┌─────────────────────────────────────────────────────┐
│ Milan Monitoring - Overview                         │
├─────────────────────────────────────────────────────┤
│                                                      │
│ MTTD: 5.2 min        Alert Delivery: 18.5 sec      │
│ [Histogram]          [Histogram]                   │
│                                                      │
│ Total Monitors: 1,234    Checks/min: 4,100         │
│ [Gauge]               [Gauge]                     │
│                                                      │
│ Request Rate (P95): 150ms    Error Rate: 0.12%      │
│ [Graph]              [Graph]                     │
│                                                      │
│ Queue Depths:                                      │
│ - alerts.queue: 15                                 │
│ - notifications.queue: 42                          │
│ - payment.webhooks: 0                              │
└─────────────────────────────────────────────────────┘
```

### Grafana Dashboard: Per-Service

```
┌─────────────────────────────────────────────────────┐
│ API Gateway                                        │
├─────────────────────────────────────────────────────┤
│                                                      │
│ Request Rate: 450 req/s                            │
│ [Time series graph]                                 │
│                                                      │
│ P50: 45ms    P95: 150ms    P99: 320ms             │
│ [Histogram]                                         │
│                                                      │
│ Error Rate: 0.05%                                  │
│ [Graph]                                            │
│                                                      │
│ Rate Limiting: 23 blocked req/s                    │
│ [Graph]                                            │
└─────────────────────────────────────────────────────┘
```

---

## Logging

### Structured JSON Logging

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Check completed",
    zap.String("monitor_id", monitorID),
    zap.String("url", url),
    zap.String("status", "up"),
    zap.Int("response_time_ms", 125),
    zap.Int("status_code", 200),
)
```

### Log Levels

```go
logger.Debug("Detailed debug info")
logger.Info("Normal operation")
logger.Warn("Warning condition")
logger.Error("Error occurred", zap.Error(err))
```

### Log Retention

**По тарифам:**
- Free: 7 дней
- Starter: 30 дней
- Professional: 90 дней
- Business: 1 год

**Implementation:**
```bash
# Cron job для удаления старых логов
0 2 * * * /usr/bin/find /var/log/milan -name "*.log" -mtime +7 -delete
```

---

## Tracing

### Distributed Tracing

```go
import (
    "context"
    "go.opentelemetry.io/otel"
)

func (s *Service) CreateMonitor(ctx context.Context, req *CreateMonitorRequest) (*Monitor, error) {
    ctx, span := otel.Tracer("monitor-service").Start(ctx, "CreateMonitor")
    defer span.End()

    // Trace database query
    monitor, err := s.db.CreateMonitor(ctx, req)

    return monitor, err
}
```

### Trace Storage

**Option 1: Jaeger**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
      - name: jaeger
        image: jaegertracing/all-in-one:1.50
        ports:
        - containerPort: 16686  # UI
        - containerPort: 14268  # HTTP collector
        env:
        - name: COLLECTOR_OTLP_ENABLED
          value: "true"
```

**Option 2: Yandex Trace (managed)**

---

## Health Checks

### Liveness Probe

```go
func (s *Service) Liveness() error {
    // Basic check: is service running?
    return nil
}
```

### Readiness Probe

```go
func (s *Service) Readiness() error {
    // Check dependencies
    if err := s.db.Ping(); err != nil {
        return err
    }

    if err := s.rabbitmq.Check(); err != nil {
        return err
    }

    return nil
}
```

### Kubernetes Configuration

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

## Alerting Rules

### Prometheus AlertRules

```yaml
groups:
- name: milan_alerts
  rules:
  # High MTTD
  - alert: HighMTTD
    expr: milan_mttd_seconds > 600
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High MTTD detected"
      description: "MTTD is {{ $value }}s (target: < 600s)"

  # Slow alert delivery
  - alert: SlowAlertDelivery
    expr: milan_alert_delivery_seconds > 30
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Slow alert delivery"
      description: "Alert delivery time is {{ $value }}s (target: < 30s)"

  # High error rate
  - alert: HighErrorRate
    expr: rate(milan_http_errors_total[5m]) > 0.05
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High error rate"
      description: "Error rate is {{ $value }}%"

  # Queue buildup
  - alert: QueueBuildup
    expr: milan_queue_depth{queue="alerts.queue"} > 1000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Queue buildup detected"
      description: "alerts.queue depth is {{ $value }}"

  # Worker lag
  - alert: WorkerLag
    expr: milan_worker_lag_seconds > 60
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Workers behind schedule"
      description: "Worker lag is {{ $value }}s"
```

---

## Error Tracking

### Panic Recovery

```go
import (
    "runtime/debug"
    "github.com/gin-gonic/gin"
)

func PanicRecoveryMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                stack := debug.Stack()

                // Log panic
                logger.Error("Panic recovered",
                    zap.Any("error", err),
                    zap.String("stack", string(stack)),
                )

                // Send to error tracking service
                // (Sentry, etc.)

                c.JSON(500, gin.H{"error": "Internal server error"})
            }
        }()

        c.Next()
    }
}
```

### Error Logging

```go
func (s *Service) handleError(err error, context map[string]interface{}) {
    logger.Error("Error occurred",
        zap.Error(err),
        zap.Any("context", context),
    )

    // Send to error tracking service
    // Track error rates for alerting
}
```

---

## Performance Monitoring

### Response Time Percentiles

```yaml
# Prometheus recording rules
groups:
- name: milan_performance
  rules:
  - record: milan_http_request_duration_seconds_p50
    expr: histogram_quantile(0.5, rate(milan_http_request_duration_seconds_bucket[5m]))

  - record: milan_http_request_duration_seconds_p95
    expr: histogram_quantile(0.95, rate(milan_http_request_duration_seconds_bucket[5m]))

  - record: milan_http_request_duration_seconds_p99
    expr: histogram_quantile(0.99, rate(milan_http_request_duration_seconds_bucket[5m]))
```

### Database Query Performance

```go
var dbQueryDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "milan_db_query_duration_seconds",
        Help: "Database query duration",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
    },
    []string{"query_name"},
)
```

---

## Production Readiness

### Graceful Shutdown

```go
func (s *Service) Shutdown(ctx context.Context) error {
    logger.Info("Shutting down service...")

    // Stop accepting new requests
    s.server.Shutdown(ctx)

    // Drain queues
    s.rabbitmq.DrainQueues(ctx)

    // Close connections
    s.db.Close()
    s.rabbitmq.Close()

    logger.Info("Service stopped")
    return nil
}
```

### Queue Draining

```go
func (c *RabbitMQClient) DrainQueues(ctx context.Context) error {
    logger.Info("Draining queues...")

    // Wait for messages to be processed
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            depth, _ := c.queueDepth("alerts.queue")
            if depth == 0 {
                return nil
            }
            time.Sleep(1 * time.Second)
        }
    }
}
```

---

**Дата:** 2026-03-08
**Версия:** 1.0
