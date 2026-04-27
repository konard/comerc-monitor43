package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/pkg/logger"
)

// CheckExecutor выполняет проверки мониторов.
type CheckExecutor struct {
	monitorRepo     interfaces.MonitorRepository
	checkResultRepo interfaces.CheckResultRepository
	maintenanceRepo interfaces.MaintenanceWindowRepository
	auditRepo       interfaces.AuditRepository
	httpClient      *http.Client
	config          CheckExecutorConfig
	meter           metric.Meter

	checksTotal    metric.Int64Counter
	checksDuration metric.Float64Histogram
	checksStatus   metric.Int64Counter
}

// CheckExecutorConfig конфигурация CheckExecutor.
type CheckExecutorConfig struct {
	DefaultTimeout              time.Duration
	MaxRecentResultsForStats    int
	DefaultDegradedResponseTime int
	DefaultDegradedFailureRate  int
	MaxResponseBodySize         int64
	Treat4xxAsDown              bool
	MaxRetries                  int
	RetryBaseDelay              time.Duration
	CheckResultRetentionDays    int
}

// NewCheckExecutor создаёт новый CheckExecutor.
func NewCheckExecutor(
	monitorRepo interfaces.MonitorRepository,
	checkResultRepo interfaces.CheckResultRepository,
	maintenanceRepo interfaces.MaintenanceWindowRepository,
	auditRepo interfaces.AuditRepository,
	config CheckExecutorConfig,
) *CheckExecutor {
	meter := otel.Meter("monitor-service")

	checksTotal, err := meter.Int64Counter(
		"monitor.checks.total",
		metric.WithDescription("Total number of checks executed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.DefaultLogger.Warn("failed to create checks_total counter", "error", err)
	}

	checksDuration, err := meter.Float64Histogram(
		"monitor.checks.duration",
		metric.WithDescription("Duration of checks in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		logger.DefaultLogger.Warn("failed to create checks_duration histogram", "error", err)
	}

	checksStatus, err := meter.Int64Counter(
		"monitor.checks.status",
		metric.WithDescription("Check results by status"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.DefaultLogger.Warn("failed to create checks_status counter", "error", err)
	}

	if config.MaxResponseBodySize == 0 {
		config.MaxResponseBodySize = 10 * 1024 * 1024
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBaseDelay == 0 {
		config.RetryBaseDelay = time.Minute
	}

	return &CheckExecutor{
		monitorRepo:     monitorRepo,
		checkResultRepo: checkResultRepo,
		maintenanceRepo: maintenanceRepo,
		auditRepo:       auditRepo,
		httpClient: &http.Client{
			Timeout: config.DefaultTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
		config:         config,
		meter:          meter,
		checksTotal:    checksTotal,
		checksDuration: checksDuration,
		checksStatus:   checksStatus,
	}
}

// ExecuteCheck выполняет проверку монитора.
func (e *CheckExecutor) ExecuteCheck(ctx context.Context, monitorID string) (*domain.CheckResult, error) {
	ctx, span := tracing.StartSpan(ctx, "CheckExecutor.ExecuteCheck")
	defer span.End()

	tracing.AddEvent(ctx, "execute_check.start", map[string]any{
		"monitor_id": monitorID,
	})

	startTime := time.Now()

	monitorUUID, err := uuid.Parse(monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid monitor_id")
	}

	monitor, err := e.monitorRepo.GetByID(ctx, monitorUUID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get monitor")
	}

	if !monitor.Status.IsActive() {
		err := errors.New("monitor is paused")
		tracing.RecordError(span, err)
		return nil, err
	}

	if !monitor.ShouldCheckNow() {
		err := errors.New("monitor is outside working hours")
		tracing.RecordError(span, err)
		return nil, err
	}

	if e.maintenanceRepo != nil {
		activeWindows, err := e.maintenanceRepo.GetActiveWindowsForMonitor(ctx, monitorUUID, time.Now())
		if err != nil {
			logger.DefaultLogger.WarnContext(ctx, "failed to check maintenance windows",
				"monitor_id", monitorID,
				"error", err,
			)
		} else if len(activeWindows) > 0 {
			logger.DefaultLogger.InfoContext(ctx, "check skipped: active maintenance window",
				"monitor_id", monitorID,
				"maintenance_windows", len(activeWindows),
			)
			skipped := domain.NewCheckResult(monitor.ID, domain.StatusPaused).
				WithErrorCode(domain.ErrorCodeCheckSkippedMaintenance)
			return skipped, nil
		}
	}

	result := e.executeHTTPCheck(ctx, monitor)

	for retry := 0; retry < e.config.MaxRetries && result.IsFailure(); retry++ {
		delay := e.config.RetryBaseDelay * time.Duration(1<<uint(retry))
		logger.DefaultLogger.InfoContext(ctx, "retrying check",
			"monitor_id", monitorID,
			"attempt", retry+2,
			"delay", delay,
		)

		select {
		case <-ctx.Done():
			return result, nil
		case <-time.After(delay):
		}

		result = e.executeHTTPCheck(ctx, monitor)
	}

	if err := e.checkResultRepo.Create(ctx, result); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to save check result")
	}

	monitor.UpdateLastCheck()
	if err := e.monitorRepo.Update(ctx, monitor); err != nil {
		logger.DefaultLogger.WarnContext(ctx, "failed to update monitor last_check time",
			"monitor_id", monitorID,
			"error", err,
		)
	}

	e.recordAuditLog(ctx, monitor, result)

	duration := time.Since(startTime).Seconds()
	e.recordMetrics(ctx, monitor, result, duration)

	tracing.AddEvent(ctx, "execute_check.complete", map[string]any{
		"monitor_id":    monitorID,
		"status":        string(result.Status),
		"response_time": result.ResponseTimeMs,
	})
	tracing.SetSuccess(span)

	return result, nil
}

// executeHTTPCheck выполняет HTTP проверку и определяет статус.
func (e *CheckExecutor) executeHTTPCheck(ctx context.Context, monitor *domain.Monitor) *domain.CheckResult {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", monitor.URL, nil)
	if err != nil {
		return e.createErrorResult(monitor.ID, err, domain.ErrorCodeNetworkError)
	}

	client := e.httpClient
	client.Timeout = time.Duration(monitor.TimeoutSeconds) * time.Second

	resp, err := client.Do(req)
	if err != nil {
		return e.classifyNetworkError(monitor.ID, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.DefaultLogger.WarnContext(ctx, "failed to close check response body",
				"monitor_id", monitor.ID,
				"error", closeErr,
			)
		}
	}()

	responseTime := time.Since(startTime)

	result := domain.NewCheckResult(monitor.ID, domain.StatusUp)
	result = result.WithResponseTime(int(responseTime.Milliseconds()))
	result = result.WithStatusCode(resp.StatusCode)

	if resp.StatusCode >= 500 {
		result.Status = domain.StatusDown
		result = result.WithError(fmt.Errorf("server error: %d", resp.StatusCode)).
			WithErrorCode(domain.ErrorCodeServerError)
		return result
	}

	if resp.StatusCode >= 400 {
		if e.config.Treat4xxAsDown {
			result.Status = domain.StatusDown
			result = result.WithError(fmt.Errorf("client error: %d", resp.StatusCode)).
				WithErrorCode(domain.ErrorCodeClientError)
		}
		return result
	}

	if resp.ContentLength > e.config.MaxResponseBodySize {
		if _, copyErr := io.Copy(io.Discard, resp.Body); copyErr != nil {
			logger.DefaultLogger.WarnContext(ctx, "failed to discard oversized response body",
				"monitor_id", monitor.ID,
				"error", copyErr,
			)
		}
		result.Status = domain.StatusDown
		result = result.WithError(
			fmt.Errorf("response body too large: %d bytes (max %d)", resp.ContentLength, e.config.MaxResponseBodySize),
		).
			WithErrorCode(domain.ErrorCodeResponseTooLarge)
		return result
	}

	if _, copyErr := io.Copy(io.Discard, resp.Body); copyErr != nil {
		logger.DefaultLogger.WarnContext(ctx, "failed to discard response body",
			"monitor_id", monitor.ID,
			"error", copyErr,
		)
	}

	status, err := e.determineStatus(ctx, monitor, int(responseTime.Milliseconds()))
	if err != nil {
		logger.DefaultLogger.WarnContext(ctx, "failed to determine status, defaulting to UP",
			"monitor_id", monitor.ID,
			"error", err,
		)
	} else {
		result.Status = status
		if status == domain.StatusDegraded {
			result = e.enrichDegradedResult(result, monitor, int(responseTime.Milliseconds()))
		}
	}

	return result
}

// classifyNetworkError определяет тип сетевой ошибки.
func (e *CheckExecutor) classifyNetworkError(monitorID uuid.UUID, err error) *domain.CheckResult {
	var dnsErr *net.DNSError
	var tlsErr *tls.CertificateVerificationError
	var opErr *net.OpError

	switch {
	case errors.As(err, &dnsErr):
		return e.createErrorResult(monitorID, err, domain.ErrorCodeDNSResolutionFailed)

	case errors.As(err, &tlsErr):
		return e.createErrorResult(monitorID, err, domain.ErrorCodeSSLCertificateExpired)

	case errors.As(err, &opErr) && opErr.Op == "dial":
		return e.createErrorResult(monitorID, err, domain.ErrorCodeConnectionTimeout)

	case strings.Contains(err.Error(), "stopped after 10 redirects"):
		return e.createErrorResult(monitorID, err, domain.ErrorCodeTooManyRedirects)

	case strings.Contains(err.Error(), "timeout"):
		return e.createErrorResult(monitorID, err, domain.ErrorCodeConnectionTimeout)

	default:
		return e.createErrorResult(monitorID, err, domain.ErrorCodeNetworkError)
	}
}

// determineStatus определяет статус на основе текущего результата и истории.
func (e *CheckExecutor) determineStatus(
	ctx context.Context,
	monitor *domain.Monitor,
	responseTimeMs int,
) (domain.MonitorStatus, error) {
	responseTimeThreshold, failureRateThreshold := monitor.GetDegradedThresholds()

	if responseTimeThreshold == 0 {
		responseTimeThreshold = e.config.DefaultDegradedResponseTime
	}
	if failureRateThreshold == 0 {
		failureRateThreshold = e.config.DefaultDegradedFailureRate
	}

	degraded := responseTimeMs > responseTimeThreshold

	recentResults, err := e.checkResultRepo.GetLatestByMonitorID(
		ctx,
		monitor.ID,
		e.config.MaxRecentResultsForStats,
	)
	if err == nil && len(recentResults) > 0 {
		results := make([]domain.CheckResult, len(recentResults))
		for i, r := range recentResults {
			results[i] = *r
		}
		failureRate := domain.CheckResultSlice(results).CalculateFailureRate()
		if failureRate > float64(failureRateThreshold) {
			degraded = true
		}
	}

	if degraded {
		return domain.StatusDegraded, nil
	}

	return domain.StatusUp, nil
}

// enrichDegradedResult добавляет причину DEGRADED статуса.
func (e *CheckExecutor) enrichDegradedResult(
	result *domain.CheckResult,
	monitor *domain.Monitor,
	responseTimeMs int,
) *domain.CheckResult {
	responseTimeThreshold, _ := monitor.GetDegradedThresholds()

	if responseTimeThreshold == 0 {
		responseTimeThreshold = e.config.DefaultDegradedResponseTime
	}

	if responseTimeMs > responseTimeThreshold {
		msg := fmt.Sprintf("response time exceeded threshold (%dms > %dms)", responseTimeMs, responseTimeThreshold)
		result = result.WithError(fmt.Errorf("%s", msg)).
			WithErrorCode(domain.ErrorCodeDegradedSlowResponse)
	}

	return result
}

// createErrorResult создаёт результат с ошибкой и кодом.
func (e *CheckExecutor) createErrorResult(monitorID uuid.UUID, err error, code domain.CheckErrorCode) *domain.CheckResult {
	result := domain.NewCheckResult(monitorID, domain.StatusDown)
	result = result.WithError(err).WithErrorCode(code)
	return result
}

// recordMetrics записывает метрики проверки.
func (e *CheckExecutor) recordMetrics(ctx context.Context, monitor *domain.Monitor, result *domain.CheckResult, duration float64) {
	commonAttrs := attribute.NewSet(
		attribute.String("monitor_id", monitor.ID.String()),
		attribute.String("monitor_url", monitor.URL),
	)

	if e.checksTotal != nil {
		e.checksTotal.Add(ctx, 1, metric.WithAttributeSet(commonAttrs))
	}

	if e.checksDuration != nil {
		e.checksDuration.Record(ctx, duration, metric.WithAttributeSet(commonAttrs))
	}

	if e.checksStatus != nil {
		statusAttrs := attribute.NewSet(
			attribute.String("monitor_id", monitor.ID.String()),
			attribute.String("monitor_url", monitor.URL),
			attribute.String("status", string(result.Status)),
		)
		e.checksStatus.Add(ctx, 1, metric.WithAttributeSet(statusAttrs))
	}
}

// recordAuditLog записывает audit log для результата проверки.
func (e *CheckExecutor) recordAuditLog(ctx context.Context, monitor *domain.Monitor, result *domain.CheckResult) {
	if e.auditRepo == nil {
		return
	}

	action := interfaces.ActionCheckCompleted
	if result.IsFailure() {
		action = interfaces.ActionCheckFailed
	}

	entry := &interfaces.AuditLogEntry{
		MonitorID: monitor.ID,
		UserID:    monitor.UserID,
		Action:    action,
		NewValues: map[string]any{
			"monitor_url":   monitor.URL,
			"status":        string(result.Status),
			"response_time": result.ResponseTimeMs,
			"status_code":   result.StatusCode,
		},
	}

	if result.ErrorCode != nil {
		entry.NewValues["error_code"] = string(*result.ErrorCode)
	}
	if result.ErrorMessage != nil {
		entry.NewValues["error_message"] = *result.ErrorMessage
	}

	if err := e.auditRepo.Create(ctx, entry); err != nil {
		logger.DefaultLogger.WarnContext(ctx, "failed to create audit log for check result",
			"monitor_id", monitor.ID,
			"error", err,
		)
	}
}
