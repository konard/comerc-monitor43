package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	instrumentationName = "github.com/raul/monitor/backend/alert-service"
)

// Metrics представляет collector для метрик
type Metrics struct {
	meter           metric.Meter
	requestCount    metric.Int64Counter
	requestDuration metric.Float64Histogram
	errorCount      metric.Int64Counter
	dbQueryDuration metric.Float64Histogram
	alertTriggered  metric.Int64Counter
	deliverySuccess metric.Int64Counter
	deliveryFailed  metric.Int64Counter
}

// NewMetrics создаёт новый collector метрик
func NewMetrics(serviceName string) *Metrics {
	meter := otel.GetMeterProvider().Meter(instrumentationName)

	m := &Metrics{meter: meter}

	// Request count by endpoint
	requestCount, err := meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Number of requests processed"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create http.server.request.count metric: %v", err))
	}
	m.requestCount = requestCount

	// Request duration (histogram for latency)
	requestDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("Request processing duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create http.server.duration metric: %v", err))
	}
	m.requestDuration = requestDuration

	// Error count by type
	errorCount, err := meter.Int64Counter(
		"http.server.error.count",
		metric.WithDescription("Number of errors encountered"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create http.server.error.count metric: %v", err))
	}
	m.errorCount = errorCount

	// Database query duration
	dbQueryDuration, err := meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Database query duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create db.query.duration metric: %v", err))
	}
	m.dbQueryDuration = dbQueryDuration

	// Business metrics: alerts triggered
	alertTriggered, err := meter.Int64Counter(
		"alert.triggered.count",
		metric.WithDescription("Number of alerts triggered"),
		metric.WithUnit("{alert}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create alert.triggered.count metric: %v", err))
	}
	m.alertTriggered = alertTriggered

	// Business metrics: delivery success
	deliverySuccess, err := meter.Int64Counter(
		"delivery.success.count",
		metric.WithDescription("Number of successful deliveries"),
		metric.WithUnit("{delivery}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create delivery.success.count metric: %v", err))
	}
	m.deliverySuccess = deliverySuccess

	// Business metrics: delivery failed
	deliveryFailed, err := meter.Int64Counter(
		"delivery.failed.count",
		metric.WithDescription("Number of failed deliveries"),
		metric.WithUnit("{delivery}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create delivery.failed.count metric: %v", err))
	}
	m.deliveryFailed = deliveryFailed

	return m
}

// RecordRequest записывает метрики запроса
func (m *Metrics) RecordRequest(ctx context.Context, method, endpoint string, statusCode int, durationMs float64) {
	// Request count
	m.requestCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("endpoint", endpoint),
			attribute.Int("status_code", statusCode),
		),
	)

	// Request duration
	m.requestDuration.Record(ctx, durationMs,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("endpoint", endpoint),
		),
	)

	// Error count (if status code indicates error)
	if statusCode >= 400 {
		m.errorCount.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("method", method),
				attribute.String("endpoint", endpoint),
				attribute.Int("status_code", statusCode),
				attribute.String("error_type", errorTypeFromStatus(statusCode)),
			),
		)
	}
}

// RecordDBQuery записывает метрики выполнения запроса к БД
func (m *Metrics) RecordDBQuery(ctx context.Context, operation, table string, durationMs float64) {
	m.dbQueryDuration.Record(ctx, durationMs,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("table", table),
		),
	)
}

// RecordAlertTriggered записывает метрику триггера алерта
func (m *Metrics) RecordAlertTriggered(ctx context.Context, alertType string) {
	m.alertTriggered.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("alert_type", alertType),
		),
	)
}

// RecordDeliverySuccess записывает метрику успешной доставки
func (m *Metrics) RecordDeliverySuccess(ctx context.Context, channelType string) {
	m.deliverySuccess.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("channel_type", channelType),
		),
	)
}

// RecordDeliveryFailed записывает метрику неудачной доставки
func (m *Metrics) RecordDeliveryFailed(ctx context.Context, channelType, errorType string) {
	m.deliveryFailed.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("channel_type", channelType),
			attribute.String("error_type", errorType),
		),
	)
}

// errorTypeFromStatus возвращает тип ошибки по HTTP статус коду
func errorTypeFromStatus(statusCode int) string {
	switch {
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}
