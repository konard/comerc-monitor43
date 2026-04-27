# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY api/proto /api/proto
COPY backend/integration-service/go.mod backend/integration-service/go.sum ./
RUN go mod download

# Copy source code
COPY backend/integration-service/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o integration-service ./cmd/integration-service

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/integration-service .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Use non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /root
USER appuser

# Expose gRPC and HTTP ports
EXPOSE 5014 8084

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8084/health || exit 1

# Run binary
CMD ["./integration-service"]
