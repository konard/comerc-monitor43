FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY api/proto /api/proto
COPY backend/dashboard-service/go.mod backend/dashboard-service/go.sum ./
RUN go mod download

COPY backend/dashboard-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dashboard-service ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/dashboard-service .
COPY --from=builder /app/migrations ./migrations

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app
USER appuser

EXPOSE 9095 8095

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8095/healthz || exit 1

CMD ["./dashboard-service"]
