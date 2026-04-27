package monitor

import (
	"context"
	"log/slog"
	"time"

	"github.com/raul/monitor/backend/scheduler-service/internal/circuitbreaker"
)

// CircuitBreakerAdapter оборачивает Adapter в circuit breaker.
//
// При достижении порога ошибок gRPC-вызовы к monitor-service
// перестают выполняться (fail-fast) до восстановления.
//
// Использование:
//
//	adapter := monitor.NewCircuitBreakerAdapter(inner, cb, logger)
//	monitors, err := adapter.ListActiveMonitors(ctx, 100)
type CircuitBreakerAdapter struct {
	inner   *Adapter
	cb      *circuitbreaker.CircuitBreaker
	timeout time.Duration
	logger  *slog.Logger
}

func NewCircuitBreakerAdapter(inner *Adapter, cb *circuitbreaker.CircuitBreaker, logger *slog.Logger) *CircuitBreakerAdapter {
	return &CircuitBreakerAdapter{inner: inner, cb: cb, timeout: 30 * time.Second, logger: logger}
}

func (a *CircuitBreakerAdapter) GetMonitor(ctx context.Context, monitorID string) (*MonitorInfo, error) {
	var result *MonitorInfo
	err := a.cb.Execute(ctx, func(ctx context.Context) error {
		var fnErr error
		result, fnErr = a.inner.GetMonitor(ctx, monitorID)
		return fnErr
	})
	if err != nil {
		if circuitbreaker.IsErrCircuitOpen(err) {
			a.logOpenState("GetMonitor")
			return nil, circuitbreaker.FormatErrCircuitOpen("monitor-service")
		}
		return nil, err
	}
	return result, nil
}

func (a *CircuitBreakerAdapter) ListActiveMonitors(ctx context.Context, pageSize int) ([]*MonitorInfo, error) {
	var result []*MonitorInfo
	err := a.cb.Execute(ctx, func(ctx context.Context) error {
		var fnErr error
		result, fnErr = a.inner.ListActiveMonitors(ctx, pageSize)
		return fnErr
	})
	if err != nil {
		if circuitbreaker.IsErrCircuitOpen(err) {
			a.logOpenState("ListActiveMonitors")
			return nil, circuitbreaker.FormatErrCircuitOpen("monitor-service")
		}
		return nil, err
	}
	return result, nil
}

func (a *CircuitBreakerAdapter) IsMonitorPaused(ctx context.Context, monitorID string) (bool, error) {
	var result bool
	err := a.cb.Execute(ctx, func(ctx context.Context) error {
		var fnErr error
		result, fnErr = a.inner.IsMonitorPaused(ctx, monitorID)
		return fnErr
	})
	if err != nil {
		if circuitbreaker.IsErrCircuitOpen(err) {
			a.logOpenState("IsMonitorPaused")
			return false, circuitbreaker.FormatErrCircuitOpen("monitor-service")
		}
		return false, err
	}
	return result, nil
}

func (a *CircuitBreakerAdapter) logOpenState(method string) {
	a.logger.WarnContext(context.Background(), "circuit breaker open, request rejected",
		"method", method,
		"state", a.cb.State(),
		"next_retry", time.Now().Add(a.timeout),
	)
}
