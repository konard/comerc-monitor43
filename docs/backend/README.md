# Backend Documentation

## Система Milan - Мониторинг доступности для РФ/СНГ

**Версия:** 1.0 MVP
**Дата:** Март 2026
**Статус:** Архитектурные решения приняты

---

## 📚 Документация

### [Overview](./overview.md)
Обзор архитектуры системы, ключевые решения, high-level diagram.

### [Database Architecture](./database.md)
- PostgreSQL schema (users, monitors, events, alerts, billing)
- Migration strategy (Goose)
- Data retention policies
- Zero-downtime подходы

### [Microservices](./microservices.md)
Детальное описание всех сервисов:
- API Gateway
- Monitor Service
- Alert Service
- Billing Service
- Scheduler Service
- Check Workers

### [Queue System](./queue-system.md)
RabbitMQ архитектура:
- Очереди для v1.0 MVP
- Dead Letter Queues
- Producer/Consumer best practices
- Monitoring

### [Infrastructure](./infrastructure.md)
K3s cluster в Timeweb Cloud:
- Multi-master HA setup
- Storage strategy
- Networking & Ingress
- Autoscaling (HPA)
- CI/CD pipeline

### [Monitoring](./monitoring.md)
Observability стек:
- OpenTelemetry (pure-golang adapters)
- Metrics (Prometheus + Grafana)
- Logging (structured JSON)
- Tracing (Jaeger)
- Health checks

### [Security & Compliance](./security.md)
- 152-ФЗ compliance
- Data encryption (TLS + field-level)
- API security (JWT, RBAC)
- Payment security (PCI DSS)
- Webhook verification

---

## 🏗️ Architecture Summary

### Tech Stack

| Компонент | Технология |
|-----------|------------|
| Backend | Go |
| Database | PostgreSQL 15+ (Timeweb Cloud) |
| Queue | RabbitMQ |
| Cache | Minimal (stateless JWT) |
| Observability | OpenTelemetry (pure-golang adapters) |
| Metrics | Prometheus + Grafana |
| Infrastructure | k3s (3 nodes, multi-master) |

### Services (v1.0 MVP)

1. **API Gateway** - Auth, rate limiting, routing
2. **Monitor Service** - CRUD, state management
3. **Alert Service** - Notifications (Email, Telegram, Webhooks)
4. **Billing Service** - Subscriptions, payments (ЮKassa)
5. **Scheduler Service** - Check scheduling
6. **Check Workers** - HTTP checks execution

### Communication

- **Synchronous:** gRPC (services ↔ workers)
- **Asynchronous:** RabbitMQ (alerts, notifications, payments)

---

## 🚀 Quick Start

### Prerequisites

```bash
# Go 1.21+
go version

# k3s cluster
kubectl version

# PostgreSQL access
psql --version
```

### Local Development

```bash
# Start dependencies
docker-compose up -d postgres rabbitmq

# Run services
go run cmd/api-gateway/main.go
go run cmd/monitor-service/main.go
go run cmd/alert-service/main.go
go run cmd/billing-service/main.go
go run cmd/scheduler-service/main.go
go run cmd/check-worker/main.go
```

### Database Migrations

```bash
# Install goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
goose -dir migrations postgres "user=postgres dbname=milan sslmode=require" up
```

---

## 📊 Key Design Decisions

### 1. Store Only Changes, Not Every Ping
- **Текущее состояние:** Last state в `monitors` table
- **История:** Только события в `monitor_events`
- **Преимущество:** Оптимизация хранилища

### 2. Stateless JWT + Minimal Cache
- **Sessions:** Stateless JWT (access 15min + refresh 7days)
- **Cache:** Minimal approach
- **Преимущество:** Упрощенная архитектура

### 3. RabbitMQ for Critical Operations
- **Очереди:** alerts, notifications, payments
- **Checks:** Local scheduler (no queue for checks)
- **Преимущество:** Баланс надежности и сложности

### 4. k3s on 3 Nodes
- **Setup:** Multi-master HA с embedded etcd
- **Provider:** Timeweb Cloud (152-ФЗ compliance)
- **Преимущество:** Cost-effective, simple operations

### 5. OpenTelemetry (PureGolang Adapters)
- **Observability:** Единый стек для metrics, traces, logs
- **Integration:** Использование существующего репозитория
- **Преимущество:** Переиспользование кода

---

## 🔒 Security & Compliance

### 152-ФЗ Compliance

- ✅ **Data Residency:** Timeweb Cloud (РФ)
- ✅ **Encryption:** TLS 1.3 + Field-level (PDE)
- ✅ **Logging:** 7-90 дней tiered retention
- ✅ **Payments:** ЮKassa (PCI DSS compliant)

### API Security

- **Authentication:** Stateless JWT
- **Authorization:** RBAC (по тарифам)
- **Rate Limiting:** In-memory, per-user
- **Input Validation:** API + Database level

---

## 📈 Scalability

### Current Capacity (MVP)

| Метрика | Target |
|---------|--------|
| Monitors | 10,000+ |
| Checks/min | 100,000+ |
| Users | 5,000+ |
| Uptime | 99.9% |

### Scaling Strategy

- **Horizontal:** K8s HPA (CPU/memory)
- **Vertical:** Bigger instances при росте
- **Database:** Read replicas при growth
- **Queue:** RabbitMQ clustering при необходимости

---

## 🛠️ Operations

### Monitoring

- **Metrics:** Prometheus + Grafana dashboards
- **Logs:** Structured JSON logging
- **Alerts:** PagerDuty/Telegram при critical issues

### Backup

- **PostgreSQL:** Daily automatic backups
- **RabbitMQ:** Weekly volume snapshots
- **Kubernetes:** etcd backups (встроенные в k3s)

### Deployment

- **Strategy:** Rolling update (zero-downtime)
- **CI/CD:** GitLab CI → Docker registry → K8s
- **Rollback:** Automatic на failure

---

## 📝 Next Steps

1. ✅ Архитектурные решения определены
2. 🔄 Детализация API (protobuf definitions)
3. 🔄 Implementation (sprint 1-4)
4. 🔄 Testing (unit, integration, E2E)
5. 🔄 Launch (beta → public)

---

## 📞 Support

**Architecture Questions:** @raul
**Infrastructure:** @devops-team
**Security:** @security-team

---

**Дата:** 2026-03-08
**Версия:** 1.0
