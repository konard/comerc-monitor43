# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY api/proto /api/proto
COPY backend/alert-service/go.mod backend/alert-service/go.sum ./
RUN go mod download

# Copy source code
COPY backend/alert-service/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o alert-service ./cmd/alert-service

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget && \
    wget -qO /usr/local/bin/grpc_health_probe \
      https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.19/grpc_health_probe-linux-amd64 && \
    echo "0f46d50fb7220dcf4cbf8861b394be9ebdfba5d7712f3900f0f4d6c38843cbcf  /usr/local/bin/grpc_health_probe" | sha256sum -c - && \
    chmod +x /usr/local/bin/grpc_health_probe

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/alert-service .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Use non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /root
USER appuser

# Expose gRPC port
EXPOSE 50051

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD grpc_health_probe -addr=:50051 || exit 1

# Run binary
CMD ["./alert-service"]
