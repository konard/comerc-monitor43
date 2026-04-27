FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY api/proto /api/proto
COPY backend/scheduler-service/go.mod backend/scheduler-service/go.sum ./
RUN go mod download

COPY backend/scheduler-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scheduler-service ./cmd/scheduler-service

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget && \
    wget -qO /usr/local/bin/grpc_health_probe \
      https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.19/grpc_health_probe-linux-amd64 && \
    chmod +x /usr/local/bin/grpc_health_probe

WORKDIR /root/

COPY --from=builder /app/scheduler-service .
COPY --from=builder /app/migrations ./migrations

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /root
USER appuser

EXPOSE 9094

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD grpc_health_probe -addr=:9094 || exit 1

CMD ["./scheduler-service"]
