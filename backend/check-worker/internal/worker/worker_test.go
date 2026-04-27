package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	api "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/raul/monitor/backend/check-worker/internal/executor"
)

type mockSchedulerClient struct {
	connectErr        error
	registerWorkerErr error
	registerWorkerID  string
	heartbeatErr      error
	unregisterErr     error
	scheduledChecks   []*api.ScheduledCheck
	getChecksErr      error
	closed            bool
	connected         bool
	registered        atomic.Bool
	unregistered      atomic.Bool
	heartbeatCnt      atomic.Int64
}

func (m *mockSchedulerClient) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockSchedulerClient) Close() error {
	m.connected = false
	m.closed = true
	return nil
}

func (m *mockSchedulerClient) IsConnected() bool {
	return m.connected
}

func (m *mockSchedulerClient) RegisterWorker(ctx context.Context, name, zone string, metadata map[string]string) (string, error) {
	if m.registerWorkerErr != nil {
		return "", m.registerWorkerErr
	}
	m.registered.Store(true)
	return m.registerWorkerID, nil
}

func (m *mockSchedulerClient) SendHeartbeat(ctx context.Context, status string, checksCompleted, checksFailed int, avgDurationMs float64) error {
	if m.heartbeatErr != nil {
		return m.heartbeatErr
	}
	m.heartbeatCnt.Add(1)
	return nil
}

func (m *mockSchedulerClient) UnregisterWorker(ctx context.Context) error {
	if m.unregisterErr != nil {
		return m.unregisterErr
	}
	m.unregistered.Store(true)
	return nil
}

func (m *mockSchedulerClient) GetScheduledChecks(ctx context.Context) ([]*api.ScheduledCheck, error) {
	if m.getChecksErr != nil {
		return nil, m.getChecksErr
	}
	return m.scheduledChecks, nil
}

type mockMonitorClient struct {
	connectErr      error
	submitResultErr error
	getMonitorErr   error
	monitor         *api.Monitor
	closed          bool
	connected       bool
	submitted       atomic.Int64
}

func (m *mockMonitorClient) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockMonitorClient) Close() error {
	m.connected = false
	m.closed = true
	return nil
}

func (m *mockMonitorClient) IsConnected() bool {
	return m.connected
}

func (m *mockMonitorClient) SubmitCheckResult(ctx context.Context, monitorID string, result *api.CheckResult) error {
	if m.submitResultErr != nil {
		return m.submitResultErr
	}
	m.submitted.Add(1)
	return nil
}

func (m *mockMonitorClient) GetMonitor(ctx context.Context, monitorID string) (*api.Monitor, error) {
	if m.getMonitorErr != nil {
		return nil, m.getMonitorErr
	}
	return m.monitor, nil
}

type mockExecutor struct {
	response *executor.CheckResponse
	err      error
	executed atomic.Int64
}

func (m *mockExecutor) ExecuteCheck(ctx context.Context, req executor.CheckRequest) (*executor.CheckResponse, error) {
	m.executed.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func newTestLogger() *slog.Logger {
	return slog.Default()
}

func TestNewWorker(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}
	exec := &mockExecutor{}

	w := NewWorker(sched, mon, exec, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	assert.NotNil(t, w)
	assert.Equal(t, "worker-01", w.name)
	assert.Equal(t, "default", w.zone)
	assert.False(t, w.IsRunning())
}

func TestWorker_Start(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	assert.True(t, w.IsRunning())
	assert.True(t, sched.registered.Load())
	assert.True(t, sched.connected)
	assert.True(t, mon.connected)
}

func TestWorker_Start_connect_scheduler_error(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{connectErr: fmt.Errorf("connection refused")}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to scheduler")
	assert.False(t, w.IsRunning())
}

func TestWorker_Start_connect_monitor_error(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{connectErr: fmt.Errorf("connection refused")}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to monitor service")
	assert.False(t, w.IsRunning())
}

func TestWorker_Start_register_error(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerErr: fmt.Errorf("registration failed")}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register worker")
}

func TestWorker_Stop(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)

	w.Stop()

	assert.False(t, w.IsRunning())
	assert.True(t, sched.unregistered.Load())
	assert.True(t, sched.closed)
	assert.True(t, mon.closed)
}

func TestWorker_Stop_double_stop(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)

	w.Stop()
	w.Stop()

	assert.False(t, w.IsRunning())
}

func TestWorker_pollAndExecuteChecks_no_checks(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{
		registerWorkerID: "w-1",
		scheduledChecks:  nil,
	}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	w.pollAndExecuteChecks(context.Background())
}

func TestWorker_pollAndExecuteChecks_get_checks_error(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{
		registerWorkerID: "w-1",
		getChecksErr:     fmt.Errorf("scheduler unavailable"),
	}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	w.pollAndExecuteChecks(context.Background())
}

func TestWorker_executeCheck_monitor_not_found(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{getMonitorErr: fmt.Errorf("monitor not found")}
	exec := &mockExecutor{}

	w := NewWorker(sched, mon, exec, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	check := &api.ScheduledCheck{MonitorId: "mon-nonexistent", Id: "sc-1"}
	w.executeCheck(context.Background(), check)

	assert.Equal(t, int64(1), w.checksFailed.Load())
	assert.Equal(t, int64(0), w.checksCompleted.Load())
}

func TestWorker_executeCheck_executor_error(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{monitor: &api.Monitor{
		Url:       "http://example.com",
		SslVerify: true,
		Headers:   map[string]string{"User-Agent": "check-worker"},
	}}
	exec := &mockExecutor{err: fmt.Errorf("request failed")}

	w := NewWorker(sched, mon, exec, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	check := &api.ScheduledCheck{MonitorId: "mon-1", Id: "sc-1"}
	w.executeCheck(context.Background(), check)

	assert.Equal(t, int64(1), w.checksFailed.Load())
	assert.Equal(t, int64(1), exec.executed.Load())
}

func TestWorker_executeCheck_submit_result_error(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{
		monitor:         &api.Monitor{Url: "http://example.com"},
		submitResultErr: fmt.Errorf("submit failed"),
	}
	exec := &mockExecutor{
		response: &executor.CheckResponse{Success: true, StatusCode: 200, ResponseTimeMs: 50.0},
	}

	w := NewWorker(sched, mon, exec, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	check := &api.ScheduledCheck{MonitorId: "mon-1", Id: "sc-1"}
	w.executeCheck(context.Background(), check)

	assert.Equal(t, int64(1), w.checksCompleted.Load())
}

func TestWorker_executeCheck_success(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{
		monitor: &api.Monitor{
			Url:       "http://example.com",
			SslVerify: true,
			Headers:   map[string]string{"X-Test": "value"},
		},
	}
	exec := &mockExecutor{
		response: &executor.CheckResponse{
			Success:        true,
			StatusCode:     200,
			ResponseTimeMs: 42.5,
			ErrorMessage:   "",
		},
	}

	w := NewWorker(sched, mon, exec, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	check := &api.ScheduledCheck{MonitorId: "mon-1", Id: "sc-1"}
	w.executeCheck(context.Background(), check)

	assert.Equal(t, int64(1), w.checksCompleted.Load())
	assert.Equal(t, int64(0), w.checksFailed.Load())
	assert.Equal(t, int64(1), mon.submitted.Load())
	assert.Equal(t, int64(1), exec.executed.Load())
}

func TestWorker_executeCheck_failed_check(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{
		monitor: &api.Monitor{Url: "http://example.com"},
	}
	exec := &mockExecutor{
		response: &executor.CheckResponse{
			Success:        false,
			StatusCode:     500,
			ResponseTimeMs: 100.0,
			ErrorMessage:   "internal server error",
		},
	}

	w := NewWorker(sched, mon, exec, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	check := &api.ScheduledCheck{MonitorId: "mon-1", Id: "sc-1"}
	w.executeCheck(context.Background(), check)

	assert.Equal(t, int64(0), w.checksCompleted.Load())
	assert.Equal(t, int64(1), w.checksFailed.Load())
}

func TestWorker_heartbeat_success(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Millisecond, 1*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	w.Stop()

	assert.GreaterOrEqual(t, sched.heartbeatCnt.Load(), int64(1))
}

func TestWorker_Stop_unregister_error(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{
		registerWorkerID: "w-1",
		unregisterErr:    fmt.Errorf("unregister failed"),
	}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	err := w.Start(context.Background())
	require.NoError(t, err)

	w.Stop()

	assert.True(t, sched.closed)
	assert.True(t, mon.closed)
}

func TestWorker_SetObservability(t *testing.T) {
	t.Parallel()

	w := NewWorker(
		&mockSchedulerClient{},
		&mockMonitorClient{},
		&mockExecutor{},
		"worker-01", "default",
		10*time.Second, 5*time.Second, 30*time.Second, 5,
		newTestLogger(),
	)

	w.SetObservability(noop.NewTracerProvider().Tracer("test"), nil)

	assert.NotNil(t, w.tracer)
	assert.Nil(t, w.metrics)
}

func TestWorker_startSpan_nil_tracer(t *testing.T) {
	t.Parallel()

	w := NewWorker(
		&mockSchedulerClient{},
		&mockMonitorClient{},
		&mockExecutor{},
		"worker-01", "default",
		10*time.Second, 5*time.Second, 30*time.Second, 5,
		newTestLogger(),
	)

	ctx, span := w.startSpan(context.Background(), "test")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestWorker_Start_with_tracing(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())
	w.SetObservability(noop.NewTracerProvider().Tracer("test"), nil)

	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	assert.True(t, w.IsRunning())
}

func TestWorker_Start_records_error_in_span(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{connectErr: fmt.Errorf("connection refused")}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())
	w.SetObservability(noop.NewTracerProvider().Tracer("test"), nil)

	err := w.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to scheduler")
}

func TestWorker_checkConnections(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	// Запускаем воркер
	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	// Проверяем что checkConnections работает
	w.checkConnections(context.Background())
	assert.True(t, w.IsRunning())
}

func TestWorker_checkConnections_scheduler_disconnected(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	// Запускаем воркер
	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	// Эмулируем отключение scheduler
	sched.connected = false

	// checkConnections должен попытаться переподключиться
	w.checkConnections(context.Background())

	// Ждем завершения goroutine с reconnect
	time.Sleep(100 * time.Millisecond)
}

func TestWorker_reconnectService(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	// Запускаем воркер
	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	// Переподключение scheduler (запускается в фоне)
	w.reconnectService(context.Background(), "scheduler")
	// Ждем завершения
	time.Sleep(50 * time.Millisecond)
	assert.True(t, sched.connected)

	// Переподключение monitor (запускается в фоне)
	w.reconnectService(context.Background(), "monitor")
	// Ждем завершения
	time.Sleep(50 * time.Millisecond)
	assert.True(t, mon.connected)
}

func TestWorker_reconnectService_unknown_service(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	// Запускаем воркер
	err := w.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { w.Stop() })

	// Неизвестный сервис не должен паниковать
	w.reconnectService(context.Background(), "unknown")
	// Ждем завершения
	time.Sleep(50 * time.Millisecond)
}

func TestWorker_connectWithRetry(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	// connectWithRetry должен успешно подключиться к scheduler
	err := w.connectWithRetry(context.Background(), "scheduler")
	assert.NoError(t, err)
	assert.True(t, sched.connected)

	// connectWithRetry должен успешно подключиться к monitor
	err = w.connectWithRetry(context.Background(), "monitor")
	assert.NoError(t, err)
	assert.True(t, mon.connected)
}

func TestWorker_connectWithRetry_scheduler_failure(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{
		connectErr: fmt.Errorf("scheduler connection failed"),
	}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := w.connectWithRetry(ctx, "scheduler")
	assert.Error(t, err)
	assert.False(t, sched.connected)
}

func TestWorker_connectWithRetry_monitor_failure(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{
		connectErr: fmt.Errorf("monitor connection failed"),
	}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := w.connectWithRetry(ctx, "monitor")
	assert.Error(t, err)
	assert.False(t, mon.connected)
}

func TestWorker_connectWithRetry_unknown_service(t *testing.T) {
	t.Parallel()

	sched := &mockSchedulerClient{registerWorkerID: "w-1"}
	mon := &mockMonitorClient{}

	w := NewWorker(sched, mon, &mockExecutor{}, "worker-01", "default", 10*time.Second, 5*time.Second, 30*time.Second, 5, newTestLogger())

	// Неизвестный сервис должен вернуть ошибку
	err := w.connectWithRetry(context.Background(), "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service")
}
