package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	api "github.com/raul/monitor/api/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/check-worker/internal/client"
	"github.com/raul/monitor/backend/check-worker/internal/executor"
	"github.com/raul/monitor/backend/check-worker/pkg/telemetry"
)

type schedulerService interface {
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool
	RegisterWorker(ctx context.Context, name, zone string, metadata map[string]string) (string, error)
	SendHeartbeat(ctx context.Context, status string, checksCompleted, checksFailed int, avgDurationMs float64) error
	UnregisterWorker(ctx context.Context) error
	GetScheduledChecks(ctx context.Context) ([]*api.ScheduledCheck, error)
}

type monitorService interface {
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool
	SubmitCheckResult(ctx context.Context, monitorID string, result *api.CheckResult) error
	GetMonitor(ctx context.Context, monitorID string) (*api.Monitor, error)
}

type checkExecutor interface {
	ExecuteCheck(ctx context.Context, req executor.CheckRequest) (*executor.CheckResponse, error)
}

type Worker struct {
	schedulerClient schedulerService
	monitorClient   monitorService
	executor        checkExecutor
	logger          *slog.Logger
	tracer          trace.Tracer
	metrics         *telemetry.Metrics

	name              string
	zone              string
	heartbeatInterval time.Duration
	pollInterval      time.Duration
	maxConcurrent     int
	checkTimeout      time.Duration

	workerID     string
	running      atomic.Bool
	activeChecks sync.WaitGroup
	sem          chan struct{}

	checksCompleted atomic.Int64
	checksFailed    atomic.Int64
	totalDurationMs atomic.Int64

	cancel       context.CancelFunc
	stateMachine *ConnectionStateMachine
	reconnectMgr *ReconnectManager
}

func safeProtoInt32(v int, field string) (int32, error) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, fmt.Errorf("%s out of int32 range: %d", field, v)
	}
	return int32(v), nil
}

func NewWorker(
	schedulerClient schedulerService,
	monitorClient monitorService,
	exec checkExecutor,
	name, zone string,
	heartbeatInterval, pollInterval, checkTimeout time.Duration,
	maxConcurrent int,
	logger *slog.Logger,
) *Worker {
	// Создаём state machine для управления подключениями
	stateMachine := NewConnectionStateMachine(logger)

	// Создаём reconnect manager с конфигурацией по умолчанию
	reconnectConfig := DefaultReconnectConfig()
	reconnectMgr := NewReconnectManager(reconnectConfig, logger)

	return &Worker{
		schedulerClient:   schedulerClient,
		monitorClient:     monitorClient,
		executor:          exec,
		logger:            logger,
		name:              name,
		zone:              zone,
		heartbeatInterval: heartbeatInterval,
		pollInterval:      pollInterval,
		maxConcurrent:     maxConcurrent,
		checkTimeout:      checkTimeout,
		sem:               make(chan struct{}, maxConcurrent),
		stateMachine:      stateMachine,
		reconnectMgr:      reconnectMgr,
	}
}

func (w *Worker) SetObservability(tracer trace.Tracer, metrics *telemetry.Metrics) {
	w.tracer = tracer
	w.metrics = metrics
}

func (w *Worker) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)

	ctx, span := w.startSpan(ctx, "Worker.Start")
	defer span.End()

	// Подключаемся к scheduler (без автоматического retry при старте)
	if err := w.schedulerClient.Connect(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to connect to scheduler: %w", err)
	}
	w.stateMachine.OnSchedulerConnected(ctx)

	// Подключаемся к monitor service (без автоматического retry при старте)
	if err := w.monitorClient.Connect(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to connect to monitor service: %w", err)
	}
	w.stateMachine.OnMonitorConnected(ctx)

	// Регистрируем worker
	workerID, err := w.schedulerClient.RegisterWorker(ctx, w.name, w.zone, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to register worker: %w", err)
	}
	w.workerID = workerID
	w.running.Store(true)

	w.logger.InfoContext(ctx, "worker started",
		"worker_id", w.workerID,
		"name", w.name,
		"zone", w.zone,
		"state", w.stateMachine.GetState(),
	)

	// Запускаем фоновые процессы
	go w.runHeartbeatLoop(ctx)
	go w.runPollLoop(ctx)
	go w.monitorConnections(ctx)

	return nil
}

func (w *Worker) Stop() {
	if !w.running.Load() {
		return
	}

	w.running.Store(false)

	// Останавливаем reconnect manager
	w.reconnectMgr.Stop()

	// Shutdown state machine
	w.stateMachine.Shutdown()

	if w.cancel != nil {
		w.cancel()
	}

	w.logger.Info("waiting for active checks to complete")
	w.activeChecks.Wait()

	ctx := context.Background()
	if err := w.schedulerClient.UnregisterWorker(ctx); err != nil {
		w.logger.Error("failed to unregister worker", "error", err)
	}

	if err := w.schedulerClient.Close(); err != nil {
		w.logger.Error("failed to close scheduler connection", "error", err)
	}
	if err := w.monitorClient.Close(); err != nil {
		w.logger.Error("failed to close monitor connection", "error", err)
	}

	w.logger.Info("worker stopped",
		"worker_id", w.workerID,
		"checks_completed", w.checksCompleted.Load(),
		"checks_failed", w.checksFailed.Load(),
	)
}

func (w *Worker) IsRunning() bool {
	return w.running.Load()
}

func (w *Worker) runHeartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx, span := w.startSpan(ctx, "Worker.Heartbeat")
			status := "IDLE"
			if len(w.sem) > 0 {
				status = "BUSY"
			}

			completed := int(w.checksCompleted.Load())
			failed := int(w.checksFailed.Load())
			total := w.totalDurationMs.Load()
			var avgDuration float64
			if completed > 0 {
				avgDuration = float64(total) / float64(completed)
			}

			err := w.schedulerClient.SendHeartbeat(ctx, status, completed, failed, avgDuration)
			if err != nil {
				span.RecordError(err)
				w.logger.Error("failed to send heartbeat", "error", err)
				if w.metrics != nil {
					w.metrics.RecordHeartbeat(ctx, false)
				}
				span.End()
				continue
			}
			if w.metrics != nil {
				w.metrics.RecordHeartbeat(ctx, true)
			}
			span.End()
		}
	}
}

func (w *Worker) runPollLoop(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !w.running.Load() {
				return
			}
			w.pollAndExecuteChecks(ctx)
		}
	}
}

func (w *Worker) pollAndExecuteChecks(ctx context.Context) {
	ctx, span := w.startSpan(ctx, "Worker.PollAndExecute")
	defer span.End()

	checks, err := w.schedulerClient.GetScheduledChecks(ctx)
	if err != nil {
		span.RecordError(err)
		w.logger.Error("failed to get scheduled checks", "error", err)
		return
	}

	if len(checks) == 0 {
		w.logger.Debug("no pending checks")
		return
	}

	span.SetAttributes(attribute.Int("checks.count", len(checks)))
	w.logger.Debug("received scheduled checks", "count", len(checks))

	for _, check := range checks {
		if !w.running.Load() {
			return
		}
		w.activeChecks.Add(1)
		go func(scheduledCheck *api.ScheduledCheck) {
			defer w.activeChecks.Done()
			w.executeCheck(ctx, scheduledCheck)
		}(check)
	}
}

func (w *Worker) executeCheck(ctx context.Context, scheduledCheck *api.ScheduledCheck) {
	monitorID := scheduledCheck.MonitorId
	spanName := fmt.Sprintf("Worker.ExecuteCheck/%s", monitorID)
	ctx, span := w.startSpan(ctx, spanName)
	defer span.End()

	span.SetAttributes(
		attribute.String("monitor_id", monitorID),
		attribute.String("scheduled_check_id", scheduledCheck.Id),
		attribute.String("worker_state", w.stateMachine.GetState().String()),
	)

	// Graceful degradation: проверяем можно ли выполнять проверки
	if !w.stateMachine.CanExecuteChecks() {
		span.SetAttributes(attribute.Bool("skipped", true))
		w.logger.Warn("check skipped due to worker state",
			"monitor_id", monitorID,
			"state", w.stateMachine.GetState(),
			"can_execute", w.stateMachine.CanExecuteChecks(),
		)
		w.checksFailed.Add(1)
		if w.metrics != nil {
			w.metrics.RecordCheck(ctx, monitorID, false, 0)
		}
		return
	}

	w.sem <- struct{}{}
	defer func() { <-w.sem }()

	monitor, err := w.monitorClient.GetMonitor(ctx, monitorID)
	if err != nil {
		span.RecordError(err)
		w.logger.Error("failed to get monitor", "monitor_id", monitorID, "error", err)
		w.checksFailed.Add(1)
		if w.metrics != nil {
			w.metrics.RecordCheck(ctx, monitorID, false, 0)
		}
		return
	}

	headers := make(map[string]string)
	if monitor.Headers != nil {
		headers = monitor.Headers
	}

	timeout := w.checkTimeout
	if monitor.Timeout != nil {
		timeout = monitor.Timeout.AsDuration()
	}

	req := executor.CheckRequest{
		MonitorID:           monitorID,
		URL:                 monitor.Url,
		ExpectedStatusCode:  monitor.ExpectedStatusCode,
		ExpectedBodyPattern: monitor.ExpectedBodyPattern,
		Timeout:             timeout,
		SSLVerify:           monitor.SslVerify,
		Headers:             headers,
	}

	resp, err := w.executor.ExecuteCheck(ctx, req)
	if err != nil {
		span.RecordError(err)
		w.logger.Error("failed to execute check", "monitor_id", monitorID, "error", err)
		w.checksFailed.Add(1)
		if w.metrics != nil {
			w.metrics.RecordCheck(ctx, monitorID, false, 0)
		}
		return
	}

	if resp.Success {
		w.checksCompleted.Add(1)
	} else {
		w.checksFailed.Add(1)
	}
	w.totalDurationMs.Add(int64(resp.ResponseTimeMs))

	span.SetAttributes(
		attribute.Bool("success", resp.Success),
		attribute.Int("status_code", resp.StatusCode),
		attribute.Float64("response_time_ms", resp.ResponseTimeMs),
	)

	statusCode, err := safeProtoInt32(resp.StatusCode, "status_code")
	if err != nil {
		span.RecordError(err)
		w.logger.Error("invalid status code for proto conversion", "monitor_id", monitorID, "error", err)
		w.checksFailed.Add(1)
		if w.metrics != nil {
			w.metrics.RecordCheck(ctx, monitorID, false, 0)
		}
		return
	}

	checkResult := client.BuildCheckResultProto(
		resp.Success,
		statusCode,
		resp.ResponseTimeMs,
		resp.ErrorMessage,
	)

	if err := w.monitorClient.SubmitCheckResult(ctx, monitorID, checkResult); err != nil {
		span.RecordError(err)
		w.logger.Error("failed to submit check result", "monitor_id", monitorID, "error", err)
		return
	}

	w.logger.InfoContext(ctx, "check completed",
		"monitor_id", monitorID,
		"success", resp.Success,
		"status_code", resp.StatusCode,
		"response_time_ms", resp.ResponseTimeMs,
	)

	if w.metrics != nil {
		w.metrics.RecordCheck(ctx, monitorID, resp.Success, resp.ResponseTimeMs)
	}
}

func (w *Worker) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if w.tracer != nil {
		return w.tracer.Start(ctx, name)
	}
	return ctx, trace.SpanFromContext(ctx)
}

// connectWithRetry подключается к сервису с использованием reconnect manager
func (w *Worker) connectWithRetry(ctx context.Context, service string) error {
	var connectFunc func(ctx context.Context) error

	switch service {
	case "scheduler":
		connectFunc = w.schedulerClient.Connect
	case "monitor":
		connectFunc = w.monitorClient.Connect
	default:
		return fmt.Errorf("unknown service: %s", service)
	}

	// Пробуем подключиться один раз без retry
	err := connectFunc(ctx)
	if err == nil {
		return nil
	}

	// Если не удалось, используем reconnect manager
	w.logger.Warn("initial connection failed, starting reconnect",
		"service", service,
		"error", err,
	)

	return w.reconnectMgr.StartReconnect(ctx, connectFunc)
}

// monitorConnections отслеживает состояния подключений и запускает reconnect
func (w *Worker) monitorConnections(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkConnections(ctx)
		}
	}
}

// checkConnections проверяет состояния подключений и предпринимает действия
func (w *Worker) checkConnections(ctx context.Context) {
	// Проверяем scheduler connection
	if w.schedulerClient.IsConnected() {
		if !w.stateMachine.IsConnected() && !w.stateMachine.IsDegraded() {
			w.stateMachine.OnSchedulerConnected(ctx)
		}
	} else {
		if w.stateMachine.IsConnected() || w.stateMachine.IsDegraded() {
			w.stateMachine.OnSchedulerDisconnected(ctx, fmt.Errorf("connection lost"))
			// Запускаем reconnect в фоне
			go w.reconnectService(ctx, "scheduler")
		}
	}

	// Проверяем monitor connection
	if w.monitorClient.IsConnected() {
		if !w.stateMachine.IsConnected() && !w.stateMachine.IsDegraded() {
			w.stateMachine.OnMonitorConnected(ctx)
		}
	} else {
		if w.stateMachine.IsConnected() || w.stateMachine.IsDegraded() {
			w.stateMachine.OnMonitorDisconnected(ctx, fmt.Errorf("connection lost"))
			// Запускаем reconnect в фоне
			go w.reconnectService(ctx, "monitor")
		}
	}
}

// reconnectService переподключается к указанному сервису
func (w *Worker) reconnectService(ctx context.Context, service string) {
	w.logger.Info("starting reconnect for service",
		"service", service,
		"state", w.stateMachine.GetState(),
	)

	var connectFunc func(ctx context.Context) error
	switch service {
	case "scheduler":
		connectFunc = w.schedulerClient.Connect
	case "monitor":
		connectFunc = w.monitorClient.Connect
	default:
		w.logger.Error("unknown service for reconnect", "service", service)
		return
	}

	// Создаём context с timeout для reconnect (максимум 5 минут)
	reconnectCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Используем reconnect manager
	err := w.reconnectMgr.StartReconnect(reconnectCtx, connectFunc)
	if err != nil {
		w.logger.Error("reconnect failed",
			"service", service,
			"error", err,
		)
		return
	}

	// Успешное переподключение
	if service == "scheduler" {
		w.stateMachine.OnSchedulerConnected(ctx)
	}
	if service == "monitor" {
		w.stateMachine.OnMonitorConnected(ctx)
	}

	w.logger.Info("reconnect successful",
		"service", service,
		"state", w.stateMachine.GetState(),
	)
}

// GetState возвращает текущее состояние Worker
func (w *Worker) GetState() ConnectionState {
	return w.stateMachine.GetState()
}

// GetConnectionInfo возвращает информацию о подключениях
func (w *Worker) GetConnectionInfo() (scheduler, monitor ConnectionInfo) {
	return w.stateMachine.GetConnectionInfo()
}
