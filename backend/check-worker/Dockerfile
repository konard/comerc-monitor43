FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY api/proto /api/proto
COPY backend/check-worker/go.mod backend/check-worker/go.sum ./
RUN go mod download

COPY backend/check-worker/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o check-worker ./cmd/check-worker

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/check-worker .

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /root
USER appuser

EXPOSE 9105

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9105/health || exit 1

CMD ["./check-worker"]
