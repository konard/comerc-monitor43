package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const instrumentationName = "github.com/raul/monitor/backend/check-worker"

type Metrics struct {
	meter           metric.Meter
	checksTotal     metric.Int64Counter
	checkDuration   metric.Float64Histogram
	checksFailed    metric.Int64Counter
	heartbeatSent   metric.Int64Counter
	heartbeatFailed metric.Int64Counter
}

func NewMetrics(serviceName string) (*Metrics, error) {
	meter := otel.GetMeterProvider().Meter(instrumentationName)

	m := &Metrics{meter: meter}

	checksTotal, err := meter.Int64Counter(
		"check.worker.checks.total",
		metric.WithDescription("Total number of checks executed"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		return nil, err
	}
	m.checksTotal = checksTotal

	checkDuration, err := meter.Float64Histogram(
		"check.worker.check.duration",
		metric.WithDescription("Check execution duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	m.checkDuration = checkDuration

	checksFailed, err := meter.Int64Counter(
		"check.worker.checks.failed",
		metric.WithDescription("Number of failed checks"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		return nil, err
	}
	m.checksFailed = checksFailed

	heartbeatSent, err := meter.Int64Counter(
		"check.worker.heartbeat.sent",
		metric.WithDescription("Number of heartbeats sent"),
		metric.WithUnit("{heartbeat}"),
	)
	if err != nil {
		return nil, err
	}
	m.heartbeatSent = heartbeatSent

	heartbeatFailed, err := meter.Int64Counter(
		"check.worker.heartbeat.failed",
		metric.WithDescription("Number of failed heartbeats"),
		metric.WithUnit("{heartbeat}"),
	)
	if err != nil {
		return nil, err
	}
	m.heartbeatFailed = heartbeatFailed

	return m, nil
}

func (m *Metrics) RecordCheck(ctx context.Context, monitorID string, success bool, durationMs float64) {
	m.checksTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("monitor_id", monitorID),
		attribute.Bool("success", success),
	))
	m.checkDuration.Record(ctx, durationMs, metric.WithAttributes(
		attribute.String("monitor_id", monitorID),
	))
	if !success {
		m.checksFailed.Add(ctx, 1, metric.WithAttributes(
			attribute.String("monitor_id", monitorID),
		))
	}
}

func (m *Metrics) RecordHeartbeat(ctx context.Context, success bool) {
	if success {
		m.heartbeatSent.Add(ctx, 1)
	} else {
		m.heartbeatFailed.Add(ctx, 1)
	}
}
