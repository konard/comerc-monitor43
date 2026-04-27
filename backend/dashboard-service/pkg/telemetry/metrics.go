package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const instrumentationName = "github.com/raul/monitor/backend/dashboard-service"

type Metrics struct {
	meter           metric.Meter
	requestCount    metric.Int64Counter
	requestDuration metric.Float64Histogram
	errorCount      metric.Int64Counter
	dbQueryDuration metric.Float64Histogram
	dashboardViews  metric.Int64Counter
	wsConnections   metric.Int64Counter
	eventsProcessed metric.Int64Counter
}

func NewMetrics(serviceName string) (*Metrics, error) {
	meter := otel.GetMeterProvider().Meter(instrumentationName)

	m := &Metrics{meter: meter}

	requestCount, err := meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Number of requests processed"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}
	m.requestCount = requestCount

	requestDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("Request processing duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	m.requestDuration = requestDuration

	errorCount, err := meter.Int64Counter(
		"http.server.error.count",
		metric.WithDescription("Number of errors encountered"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}
	m.errorCount = errorCount

	dbQueryDuration, err := meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Database query duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	m.dbQueryDuration = dbQueryDuration

	dashboardViews, err := meter.Int64Counter(
		"dashboard.views.count",
		metric.WithDescription("Number of dashboard views"),
		metric.WithUnit("{view}"),
	)
	if err != nil {
		return nil, err
	}
	m.dashboardViews = dashboardViews

	wsConnections, err := meter.Int64Counter(
		"ws.connections.count",
		metric.WithDescription("Number of WebSocket connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}
	m.wsConnections = wsConnections

	eventsProcessed, err := meter.Int64Counter(
		"events.processed.count",
		metric.WithDescription("Number of events processed"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, err
	}
	m.eventsProcessed = eventsProcessed

	return m, nil
}

func (m *Metrics) RecordRequest(ctx context.Context, method, endpoint string, statusCode int, durationMs float64) {
	m.requestCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("endpoint", endpoint),
			attribute.Int("status_code", statusCode),
		),
	)

	m.requestDuration.Record(ctx, durationMs,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("endpoint", endpoint),
		),
	)

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

func (m *Metrics) RecordDBQuery(ctx context.Context, operation, table string, durationMs float64) {
	m.dbQueryDuration.Record(ctx, durationMs,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("table", table),
		),
	)
}

func (m *Metrics) RecordDashboardView(ctx context.Context, dashboardID string) {
	m.dashboardViews.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("dashboard_id", dashboardID),
		),
	)
}

func (m *Metrics) RecordWSConnection(ctx context.Context, action string) {
	m.wsConnections.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("action", action),
		),
	)
}

func (m *Metrics) RecordEventProcessed(ctx context.Context, eventType string) {
	m.eventsProcessed.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
		),
	)
}

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
