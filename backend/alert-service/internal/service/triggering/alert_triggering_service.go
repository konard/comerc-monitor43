package triggering

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

// defaultCooldownPeriod — fallback cooldown when throttling service не задан.
const defaultCooldownPeriod = 15 * time.Minute

// AlertTriggeringService управляет триггерами алертов
type AlertTriggeringService struct {
	alertRepo         repository.AlertRepository
	ruleRepo          repository.AlertRuleRepository
	throttlingService ThrottleChecker
	cooldownPeriod    time.Duration
	tracer            trace.Tracer
	metrics           *apptelemetry.Metrics
}

// ThrottleChecker определяет интерфейс для проверки throttling алертов
type ThrottleChecker interface {
	CheckThrottle(ctx context.Context, userID, monitorID uuid.UUID, status string) (*ThrottleResult, error)
}

// ThrottleResult представляет результат проверки throttling
type ThrottleResult struct {
	Action     string
	Reason     string
	DeliverAt  *time.Time
	StormGroup bool
}

// successStatuses содержит статусы монитора, которые считаются успешными
var successStatuses = map[string]bool{
	"success": true,
	"up":      true,
	"ok":      true,
	"healthy": true,
}

// NewAlertTriggeringService создаёт новый сервис триггеринга
func NewAlertTriggeringService(
	alertRepo repository.AlertRepository,
	ruleRepo repository.AlertRuleRepository,
	tracer trace.Tracer,
	metrics *apptelemetry.Metrics,
	opts ...AlertTriggeringOption,
) *AlertTriggeringService {
	svc := &AlertTriggeringService{
		alertRepo:      alertRepo,
		ruleRepo:       ruleRepo,
		cooldownPeriod: defaultCooldownPeriod,
		tracer:         tracer,
		metrics:        metrics,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// AlertTriggeringOption задаёт опциональные параметры для AlertTriggeringService
type AlertTriggeringOption func(*AlertTriggeringService)

// WithThrottlingService устанавливает сервис throttling
func WithThrottlingService(tc ThrottleChecker) AlertTriggeringOption {
	return func(s *AlertTriggeringService) {
		s.throttlingService = tc
	}
}

// WithCooldownPeriod устанавливает fallback cooldown для случая, когда throttling-сервис не задан.
func WithCooldownPeriod(period time.Duration) AlertTriggeringOption {
	return func(s *AlertTriggeringService) {
		if period > 0 {
			s.cooldownPeriod = period
		}
	}
}

// ProcessMonitorCheck обрабатывает результат проверки монитора
func (s *AlertTriggeringService) ProcessMonitorCheck(
	ctx context.Context,
	userID, monitorID uuid.UUID,
	status string,
	consecutiveFailures int,
) (*model.Alert, error) {
	ctx, span := s.tracer.Start(ctx, "AlertTriggeringService.ProcessMonitorCheck")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.String("user_id", userID.String()),
			attribute.String("monitor_id", monitorID.String()),
			attribute.String("status", status),
			attribute.Int("consecutive_failures", consecutiveFailures),
		}...,
	)

	if successStatuses[strings.ToLower(status)] && consecutiveFailures == 0 {
		activeAlerts, err := s.alertRepo.GetActiveAlertsForMonitor(ctx, monitorID.String())
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		for _, alert := range activeAlerts {
			alert.Status = model.AlertStatusResolved
			alert.UpdatedAt = time.Now()
			if err := s.alertRepo.Update(ctx, alert); err != nil {
				span.RecordError(err)
				return nil, err
			}

			span.AddEvent("alert_auto_resolved", trace.WithAttributes(
				attribute.String("alert_id", alert.ID.String()),
			))
		}

		return nil, nil
	}

	// Получаем правило алерта для монитора
	rule, err := s.ruleRepo.GetByUserIDAndMonitorID(ctx, userID.String(), monitorID.String())
	if err != nil {
		span.RecordError(err)
		// Если правило не найдено, просто возвращаем nil (без алерта)
		if errors.Is(err, model.ErrAlertRuleNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// Проверяем, что правило включено
	if !rule.Enabled {
		span.AddEvent("rule_disabled", trace.WithAttributes(
			attribute.String("rule_id", rule.ID.String()),
		))
		return nil, nil
	}

	// Проверяем, достигнуто ли пороговое количество последовательных неудач
	if consecutiveFailures < rule.ConsecutiveFailures {
		span.AddEvent("threshold_not_reached", trace.WithAttributes(
			attribute.Int("consecutive_failures", consecutiveFailures),
			attribute.Int("threshold", rule.ConsecutiveFailures),
		))
		return nil, nil
	}

	// Проверяем throttling (storm detection, rate limit, cooldown)
	if s.throttlingService != nil {
		throttleResult, err := s.throttlingService.CheckThrottle(ctx, userID, monitorID, status)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		switch throttleResult.Action {
		case "suppress":
			span.AddEvent("alert_suppressed", trace.WithAttributes(
				attribute.String("reason", throttleResult.Reason),
			))
			return nil, nil
		case "storm_group":
			span.AddEvent("alert_storm_grouped", trace.WithAttributes(
				attribute.String("reason", throttleResult.Reason),
			))
		case "queue":
			span.AddEvent("alert_queued", trace.WithAttributes(
				attribute.String("reason", throttleResult.Reason),
			))
		}
	} else {
		// Fallback: проверяем cooldown без throttling-сервиса
		lastAlert, err := s.alertRepo.GetLastAlertTimeAnyStatus(ctx, monitorID.String())
		if err != nil && err != model.ErrAlertNotFound {
			span.RecordError(err)
			return nil, err
		}

		if lastAlert != nil {
			cooldownPeriod := s.cooldownPeriod
			if cooldownPeriod <= 0 {
				cooldownPeriod = defaultCooldownPeriod
			}
			if time.Since(lastAlert.CreatedAt) < cooldownPeriod {
				span.AddEvent("alert_too_recent", trace.WithAttributes(
					attribute.String("last_alert_id", lastAlert.ID.String()),
					attribute.String("last_alert_created_at", lastAlert.CreatedAt.String()),
				))
				return nil, nil
			}
		}
	}

	// Создаём новый алерт
	alert := &model.Alert{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		AlertRuleID:         &rule.ID,
		Status:              model.AlertStatusTriggered,
		Type:                model.AlertTypeStatusCode, // Тип алерта на основе статуса монитора
		Enabled:             true,
		ConsecutiveFailures: consecutiveFailures,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := s.alertRepo.Create(ctx, alert); err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.AddEvent("alert_created", trace.WithAttributes(
		attribute.String("alert_id", alert.ID.String()),
		attribute.String("rule_id", rule.ID.String()),
		attribute.Int("consecutive_failures", consecutiveFailures),
	))

	// Записываем метрику триггера алерта
	if s.metrics != nil {
		s.metrics.RecordAlertTriggered(ctx, string(alert.Type))
	}

	return alert, nil
}

// CheckAlertRules проверяет правила алертов для монитора
func (s *AlertTriggeringService) CheckAlertRules(
	ctx context.Context,
	userID, monitorID uuid.UUID,
) ([]*model.AlertRule, error) {
	ctx, span := s.tracer.Start(ctx, "AlertTriggeringService.CheckAlertRules")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.String("user_id", userID.String()),
			attribute.String("monitor_id", monitorID.String()),
		}...,
	)

	// Получаем все правила алертов для пользователя
	rules, err := s.ruleRepo.List(ctx, userID.String())
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Фильтруем правила для конкретного монитора
	var monitorRules []*model.AlertRule
	for _, rule := range rules {
		if rule.MonitorID == monitorID {
			monitorRules = append(monitorRules, rule)
		}
	}

	span.AddEvent("rules_found", trace.WithAttributes(
		attribute.Int("total_rules", len(rules)),
		attribute.Int("monitor_rules", len(monitorRules)),
	))

	return monitorRules, nil
}

// CreateAlert создаёт новый алерт
func (s *AlertTriggeringService) CreateAlert(
	ctx context.Context,
	alert *model.Alert,
) (*model.Alert, error) {
	ctx, span := s.tracer.Start(ctx, "AlertTriggeringService.CreateAlert")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.String("alert_id", alert.ID.String()),
			attribute.String("user_id", alert.UserID.String()),
			attribute.String("monitor_id", alert.MonitorID.String()),
			attribute.String("status", string(alert.Status)),
		}...,
	)

	if err := s.alertRepo.Create(ctx, alert); err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.AddEvent("alert_created", trace.WithAttributes(
		attribute.String("alert_id", alert.ID.String()),
	))

	if s.metrics != nil {
		s.metrics.RecordAlertTriggered(ctx, string(alert.Type))
	}

	return alert, nil
}

// CleanupOldResolvedAlerts удаляет старые алерты в статусе resolved
func (s *AlertTriggeringService) CleanupOldResolvedAlerts(ctx context.Context, retentionDays int) error {
	ctx, span := s.tracer.Start(ctx, "AlertTriggeringService.CleanupOldResolvedAlerts")
	defer span.End()

	span.SetAttributes(
		attribute.Int("retention_days", retentionDays),
	)

	retention := time.Duration(retentionDays) * 24 * time.Hour

	if err := s.alertRepo.DeleteResolvedOlderThan(ctx, retention); err != nil {
		span.RecordError(err)
		return err
	}

	span.AddEvent("resolved_alerts_cleaned_up", trace.WithAttributes(
		attribute.Int("retention_days", retentionDays),
	))

	return nil
}
