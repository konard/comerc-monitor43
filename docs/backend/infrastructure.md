# Infrastructure

## Провайдер и конфигурация

**Cloud Provider:** Timeweb Cloud (соответствие 152-ФЗ)
**Kubernetes:** k3s cluster (multi-master HA)
**Nodes:** 3 виртуальные машины

---

## Kubernetes Cluster (k3s)

### Архитектура

**Multi-master HA setup:**
- 3 master nodes (embedded etcd)
- External PostgreSQL
- Node monitoring
- Automatic failover

### Почему k3s?

✅ **Легковесный:** Binary < 100MB
✅ **Простой:** Easy setup и maintenance
✅ **Эффективный:** Optimized for small clusters
✅ **Built-in:** Traefik ingress, embedded SQLite

### Потенциальные проблемы и решения

| Проблема | Решение |
|---------|---------|
| Single point of failure | Multi-master + etcd cluster |
| Нет auto-healing | Node monitoring + recovery |
| Limited scalability | 3 nodes достаточно для MVP (~5000 monitors) |
| Manual updates | Regular backup + update procedures |

---

## Node Configuration

### Specifications

```
Node 1: Master + etcd
- CPU: 4 cores
- RAM: 8 GB
- Disk: 50 GB SSD

Node 2: Master + etcd
- CPU: 4 cores
- RAM: 8 GB
- Disk: 50 GB SSD

Node 3: Master + etcd
- CPU: 4 cores
- RAM: 8 GB
- Disk: 50 GB SSD
```

### k3s Installation

```bash
# На всех нодах
curl -sfL https://get.k3s.io | sh -

# Первая нода (master)
curl -sfL https://get.k3s.io | K3S_TOKEN=secret sh -s - server \
  --cluster-init \
  --tls-san k3s.example.com

# Вторая и третья ноды
curl -sfL https://get.k3s.io | K3S_TOKEN=secret sh -s - server \
  --server https://node1.example.com:6443
```

---

## Storage Strategy

### PostgreSQL

**Provider:** Timeweb Cloud Managed PostgreSQL

**Configuration:**
- Version: PostgreSQL 15+
- Instance: 2 vCPU, 4 GB RAM, 50 GB SSD
- HA: Read replica (будет добавлено при росте)
- Backup: Daily automatic backups + Point-in-time recovery

**Connection String:**
```
postgres://user:password@postgres.timeweb.cloud:5432/milan?sslmode=require
```

### Persistent Volumes

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: rabbitmq-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: timeweb-ssd
```

---

## Networking

### Service Mesh

**Решение:** K8s Ingress (Traefik) + API Gateway

**Почему без Service Mesh для MVP?**
- Избыточно для small cluster
- Добавляет сложность
- Traefik Ingress достаточно для routing

### Ingress Configuration

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: milan-ingress
  annotations:
    traefik.ingress.kubernetes.io/frontend-entry-points: http,https
    traefik.ingress.kubernetes.io/redirect-entry-point: https
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - milan.example.com
    secretName: milan-tls
  rules:
  - host: milan.example.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: api-gateway
            port:
              number: 8080
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend
            port:
              number: 3000
```

---

## Secrets Management

### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
type: Opaque
stringData:
  username: milan_user
  password: ${POSTGRES_PASSWORD}
  database: milan
  host: postgres.timeweb.cloud
  port: "5432"
```

### Environment Variables

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: api-gateway-config
data:
  JWT_SECRET: ${JWT_SECRET}
  RABBITMQ_URL: "amqp://user:pass@rabbitmq:5672/"
  POSTGRES_HOST: "postgres.timeweb.cloud"
  POSTGRES_PORT: "5432"
  POSTGRES_DB: "milan"
```

---

## Autoscaling

### Horizontal Pod Autoscaler (HPA)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Check Workers HPA

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

---

## Deployment Strategy

### Rolling Update

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: milan/api-gateway:v1.0.0
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

### Blue-Green Deployment (для critical updates)

```yaml
# Дублируем deployment с новой версией
# Переключаем Ingress на новые pods
# Rollback если проблемы
```

---

## Monitoring & Logging

### Prometheus + Grafana

**Prometheus:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:v2.45.0
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: prometheus-config
          mountPath: /etc/prometheus
        - name: prometheus-data
          mountPath: /prometheus
      volumes:
      - name: prometheus-config
        configMap:
          name: prometheus-config
      - name: prometheus-data
        persistentVolumeClaim:
          claimName: prometheus-pvc
```

**Grafana:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:10.0.0
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-secret
              key: admin-password
        volumeMounts:
        - name: grafana-data
          mountPath: /var/lib/grafana
      volumes:
      - name: grafana-data
        persistentVolumeClaim:
          claimName: grafana-pvc
```

### OpenTelemetry Integration

**Using pure-golang adapters:**
```go
import (
    "github.com/pure-golang/adapters/observability"
)

func main() {
    // Initialize OpenTelemetry
    observability.Init(observability.Config{
        ServiceName:    "api-gateway",
        ServiceVersion: "1.0.0",
        Endpoint:       "http://jaeger:14268/api/traces",
    })

    // Automatic instrumentation
    // Metrics, traces, logging
}
```

---

## Backup & Disaster Recovery

### Backup Strategy

**PostgreSQL:**
- Daily automatic backups (Timeweb Cloud)
- Point-in-time recovery (7 days)
- Manual backup перед миграциями

**RabbitMQ:**
- Volume snapshot weekly
- Message retention 7 days in DLQ

**Kubernetes:**
- Regular etcd backups (встроенные в k3s)
- Velero для cluster backup

### Recovery Procedures

```bash
# PostgreSQL recovery
pg_restore -d milan milan_backup.dump

# RabbitMQ recovery
kubectl exec rabbitmq-0 -- rabbitmqctl restore

# Kubernetes recovery
velero restore create --from-backup milan-backup-20260308
```

---

## CI/CD Pipeline

### GitLab CI Example

```yaml
stages:
  - build
  - test
  - deploy

build:
  stage: build
  script:
    - docker build -t milan/api-gateway:$CI_COMMIT_SHA .
    - docker push registry.example.com/milan/api-gateway:$CI_COMMIT_SHA

test:
  stage: test
  script:
    - go test ./...
    - go test -race ./...

deploy:
  stage: deploy
  script:
    - kubectl set image deployment/api-gateway api-gateway=registry.example.com/milan/api-gateway:$CI_COMMIT_SHA
    - kubectl rollout status deployment/api-gateway
  only:
    - main
```

---

## Security

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: milan-network-policy
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: milan
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
```

### Pod Security Policies

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: milan-psp
spec:
  privileged: false
  runAsUser:
    rule: MustRunAsNonRoot
  seLinux:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  volumes:
  - configMap
  - persistentVolumeClaim
  - secret
```

---

## Cost Optimization

### Resource Limits

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: milan-resource-quota
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    limits.cpu: "20"
    limits.memory: 40Gi
    persistentvolumeclaims: "10"
```

### Node Auto-scaling

**Future:** Cluster autoscaler при росте нагрузки

---

**Дата:** 2026-03-08
**Версия:** 1.0
