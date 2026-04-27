# Queue System (RabbitMQ)

## Технологический выбор

**Queue Technology:** RabbitMQ
**Provider:** Self-hosted в k3s cluster
**Version:** 3.12+

### Почему RabbitMQ?

✅ **Reliability:** Сообщения не теряются при сбоях
✅ **Features:** DLQ, acknowledgments, publisher confirms
✅ **Flexibility:** Multiple routing patterns
✅ **Maturity:** Enterprise-grade, battle-tested
✅ **Monitoring:** Built-in UI and metrics

---

## Очереди для v1.0 MVP

### 1. `alerts.queue`

**Назначение:** Триггеринг и отправка алертов

**Producer:** Monitor Service
**Consumer:** Alert Service

#### Message Format
```json
{
  "alert_id": "uuid",
  "monitor_id": "uuid",
  "monitor_name": "My API",
  "event_type": "status_change",
  "old_status": "up",
  "new_status": "down",
  "timestamp": "2026-03-08T10:30:00Z",
  "metadata": {
    "response_time_ms": 1250,
    "error_message": "Connection timeout",
    "check_url": "https://api.example.com"
  }
}
```

#### Configuration
```yaml
alerts.queue:
  durable: true
  auto_delete: false
  arguments:
    x-max-length: 10000
    x-dead-letter-exchange: "alerts.dlx"
    x-dead-letter-routing-key: "alerts.dlq"
```

---

### 2. `notifications.queue`

**Назначение:** Отправка уведомлений (Email, Telegram, Webhooks)

**Producer:** Alert Service
**Consumer:** Notification Workers

#### Message Format
```json
{
  "notification_id": "uuid",
  "alert_id": "uuid",
  "channel_type": "telegram",
  "channel_config": {
    "chat_id": "123456789",
    "bot_token": "***"
  },
  "message": {
    "title": "⚠️ Monitor DOWN",
    "body": "Monitor 'My API' is DOWN\n\nStatus: 502 Bad Gateway\nResponse time: 1250ms\nTime: 2026-03-08 10:30:00",
    "format": "markdown"
  },
  "retry_count": 0,
  "max_retries": 3
}
```

#### Configuration
```yaml
notifications.queue:
  durable: true
  auto_delete: false
  arguments:
    x-max-priority: 10
    x-dead-letter-exchange: "notifications.dlx"
```

---

### 3. `payment.webhooks`

**Назначение:** Обработка платежей от ЮKassa

**Producer:** ЮKassa (external webhook)
**Consumer:** Billing Service

#### Message Format
```json
{
  "event": "payment.succeeded",
  "payment": {
    "id": "2b764ec0-000f-5000-8000-15c48aa77ac1",
    "status": "succeeded",
    "amount": {
      "value": "500.00",
      "currency": "RUB"
    },
    "metadata": {
      "user_id": "uuid",
      "subscription_id": "uuid"
    }
  }
}
```

#### Configuration
```yaml
payment.webhooks:
  durable: true
  auto_delete: false
  arguments:
    x-max-length: 1000
    x-dead-letter-exchange: "payment.dlx"
```

---

## Dead Letter Queues (DLQ)

### Назначение
Обработка невалидных или failed сообщений.

### DLQ Configuration

#### `alerts.dlq`
```yaml
alerts.dlq:
  durable: true
  arguments:
    x-message-ttl: 604800000  # 7 days
```

#### `notifications.dlq`
```yaml
notifications.dlq:
  durable: true
  arguments:
    x-message-ttl: 604800000  # 7 days
```

#### `payment.dlq`
```yaml
payment.dlq:
  durable: true
  arguments:
    x-message-ttl: 2592000000  # 30 days (payment retention)
```

---

## Producer Best Practices

### 1. Publisher Confirms
```go
channel.Confirm(false)

// Publish message
err := channel.PublishWithContext(
    ctx,
    "alerts.exchange",   // exchange
    "alerts.routing",    // routing key
    false,               // mandatory
    false,               // immediate
    amqp.Publishing{
        ContentType:  "application/json",
        DeliveryMode: amqp.Persistent,  // durable
        Body:         messageBody,
    },
)

// Wait for confirmation
confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

if confirmed := <-confirms; !confirmed.Ack {
    // Message not delivered, retry
}
```

### 2. Message Idempotency
```go
message := map[string]interface{}{
    "idempotency_key": uuid.New().String(),  // unique key
    "message_id":      uuid.New().String(),
    "timestamp":       time.Now().Unix(),
    "payload":         payload,
}
```

---

## Consumer Best Practices

### 1. Acknowledgments
```go
msgs, err := channel.Consume(
    "alerts.queue",
    "alert-consumer",
    false,  // autoAck = false (manual ack)
    false,  // exclusive
    false,  // noLocal
    false,  // noWait
    nil,    // args
)

for msg := range msgs {
    // Process message
    if err := processMessage(msg); err != nil {
        // Failed processing
        msg.Nack(false, true)  // requeue=true
        continue
    }

    // Success
    msg.Ack(false)
}
```

### 2. Retry Logic
```go
func processMessage(msg amqp.Delivery) error {
    var notification Notification
    if err := json.Unmarshal(msg.Body, &notification); err != nil {
        return err  // Don't retry invalid JSON
    }

    // Extract retry count
    retryCount := msg.Headers["x-retry-count"].(int)
    maxRetries := 3

    // Process notification
    if err := sendNotification(notification); err != nil {
        if retryCount < maxRetries {
            // Increment retry count
            msg.Headers["x-retry-count"] = retryCount + 1

            // Requeue with delay
            time.Sleep(time.Duration(math.Pow(2, float64(retryCount))) * time.Second)
            msg.Nack(false, true)
            return nil
        }

        // Max retries exceeded, send to DLQ
        return err
    }

    return nil
}
```

### 3. Prefetch Count
```go
// Limit unacknowledged messages
err := channel.Qos(
    10,    // prefetch count
    0,     // prefetch size
    false, // global
)
```

---

## Exchange Types

### 1. Direct Exchange
```go
// alerts.exchange
err := channel.ExchangeDeclare(
    "alerts.exchange",
    "direct",  // direct routing
    true,      // durable
    false,     // auto-delete
    false,     // internal
    false,     // no-wait
    nil,       // arguments
)

// Bind queue to exchange
channel.QueueBind(
    "alerts.queue",
    "alerts.routing",
    "alerts.exchange",
    false,
    nil,
)
```

### 2. Topic Exchange (для future)
```go
// notifications.exchange (для приоритетов)
err := channel.ExchangeDeclare(
    "notifications.exchange",
    "topic",   // topic routing
    true,
    false,
    false,
    false,
    nil,
)

// Routing patterns
// "notifications.telegram.*"
// "notifications.email.*"
// "notifications.webhook.*"
```

---

## Monitoring & Alerting

### RabbitMQ Management UI
```
URL: http://rabbitmq:15672
Username: admin
Password: ${RABBITMQ_PASSWORD}
```

### Key Metrics
- Queue depth (messages waiting)
- Message rate (publish/deliver)
- Consumer lag
- DLQ size

### Alerting
```
Alert if alerts.queue depth > 1000 for 5min
Alert if DLQ size > 100 for 1min
Alert if consumer lag > 1000
```

---

## Configuration

### RabbitMQ Configuration
```yaml
# rabbitmq.conf
listeners.tcp.default = 5672
management.tcp.port = 15672

# Memory
vm_memory_high_watermark.relative = 0.6

# Disk
disk_free_limit.relative = 2.0

# Clustering (future)
cluster_formation.peer_discovery_backend = rabbitmq_peer_discovery_k8s
cluster_formation.k8s.host = kubernetes.default.svc.cluster.local
cluster_formation.k8s.address_type = hostname
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: rabbitmq
spec:
  serviceName: rabbitmq
  replicas: 1
  selector:
    matchLabels:
      app: rabbitmq
  template:
    metadata:
      labels:
        app: rabbitmq
    spec:
      containers:
      - name: rabbitmq
        image: rabbitmq:3.12-management
        ports:
        - containerPort: 5672
        - containerPort: 15672
        env:
        - name: RABBITMQ_DEFAULT_USER
          value: admin
        - name: RABBITMQ_DEFAULT_PASS
          valueFrom:
            secretKeyRef:
              name: rabbitmq-secret
              key: password
        volumeMounts:
        - name: rabbitmq-data
          mountPath: /var/lib/rabbitmq
  volumeClaimTemplates:
  - metadata:
      name: rabbitmq-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

---

## Connection Management

### Go Connection Pool
```go
type RabbitMQClient struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

func NewRabbitMQClient(url string) (*RabbitMQClient, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, err
    }

    channel, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, err
    }

    return &RabbitMQClient{
        conn:    conn,
        channel: channel,
    }, nil
}

func (c *RabbitMQClient) Close() {
    if c.channel != nil {
        c.channel.Close()
    }
    if c.conn != nil {
        c.conn.Close()
    }
}

func (c *RabbitMQClient) Reconnect(url string) error {
    c.Close()
    return NewRabbitMQClient(url)
}
```

### Connection Error Handling
```go
func (c *RabbitMQClient) MonitorConnection() {
    errChan := make(chan *amqp.Error)
    c.conn.NotifyClose(errChan)

    go func() {
        for err := range errChan {
            log.Printf("Connection closed: %v", err)

            // Reconnect
            for {
                time.Sleep(5 * time.Second)
                if err := c.Reconnect(c.url); err == nil {
                    log.Println("Reconnected to RabbitMQ")
                    break
                }
            }
        }
    }()
}
```

---

**Дата:** 2026-03-08
**Версия:** 1.0
