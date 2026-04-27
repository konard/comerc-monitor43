# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY api/proto /api/proto
COPY backend/billing-service/go.mod backend/billing-service/go.sum ./
RUN go mod download

COPY backend/billing-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o billing-service ./cmd/billing-service

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget && \
    wget -qO /usr/local/bin/grpc_health_probe \
      https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.19/grpc_health_probe-linux-amd64 && \
    echo "0f46d50fb7220dcf4cbf8861b394be9ebdfba5d7712f3900f0f4d6c38843cbcf  /usr/local/bin/grpc_health_probe" | sha256sum -c - && \
    chmod +x /usr/local/bin/grpc_health_probe

WORKDIR /root/

COPY --from=builder /app/billing-service .
COPY --from=builder /app/migrations ./migrations

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /root
USER appuser

EXPOSE 5003

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD grpc_health_probe -addr=:5003 || exit 1

CMD ["./billing-service"]
