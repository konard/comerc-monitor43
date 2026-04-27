# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY api/proto /api/proto
COPY backend/notification-template-service/go.mod backend/notification-template-service/go.sum ./
RUN go mod download

# Copy source code
COPY backend/notification-template-service/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o notification-template-service ./cmd/notification-template-service

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget && \
    wget -qO /usr/local/bin/grpc_health_probe \
      https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.19/grpc_health_probe-linux-amd64 && \
    chmod +x /usr/local/bin/grpc_health_probe

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/notification-template-service .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD grpc_health_probe -addr=:50051 || exit 1

# Run the binary
CMD ["./notification-template-service"]
