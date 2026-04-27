FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY api/proto /api/proto
COPY backend/reporting-service/go.mod backend/reporting-service/go.sum ./
RUN go mod download

COPY backend/reporting-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o reporting-service ./cmd/reporting-service

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/reporting-service .
COPY --from=builder /app/migrations ./migrations

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /root
USER appuser

EXPOSE 5007
EXPOSE 8084

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8084/health || exit 1

CMD ["./reporting-service"]
