package escalation

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	"github.com/raul/monitor/backend/alert-service/internal/service/audit"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

const (
	DefaultEscalationTimeout = 30 * time.Minute
)

// EscalationService управляет эскалациями алертов
type EscalationService struct {
	escalationRepo repository.AlertEscalationRepository
	alertRepo      repository.AlertRepository
	channelRepo    repository.AlertChannelRepository
	auditService   *audit.AuditService
	timeout        time.Duration
	tracer         trace.Tracer
	metrics        *apptelemetry.Metrics
}

// EscalationOption задаёт опциональные параметры EscalationService.
type EscalationOption func(*EscalationService)

// WithEscalationTimeout переопределяет таймаут авто-эскалации.
func WithEscalationTimeout(d time.Duration) EscalationOption {
	return func(s *EscalationService) {
		if d > 0 {
			s.timeout = d
		}
	}
}

// NewEscalationService создаёт новый сервис эскалаций
func NewEscalationService(
	escalationRepo repository.AlertEscalationRepository,
	alertRepo repository.AlertRepository,
	channelRepo repository.AlertChannelRepository,
	auditService *audit.AuditService,
	tracer trace.Tracer,
	metrics *apptelemetry.Metrics,
	opts ...EscalationOption,
) *EscalationService {
	s := &EscalationService{
		escalationRepo: escalationRepo,
		alertRepo:      alertRepo,
		channelRepo:    channelRepo,
		auditService:   auditService,
		timeout:        DefaultEscalationTimeout,
		tracer:         tracer,
		metrics:        metrics,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CheckAutoEscalation находит все неподтверждённые алерты с истёкшим таймаутом
// и создаёт записи эскалации для каждого из них
func (s *EscalationService) CheckAutoEscalation(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "EscalationService.CheckAutoEscalation")
	defer span.End()

	threshold := time.Now().Add(-s.timeout)

	alerts, err := s.alertRepo.ListActiveByMonitorID(ctx, "")
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to list active alerts for auto escalation: %v", err)
	}

	escalated := 0
	for _, alert := range alerts {
		if alert.Status != model.AlertStatusTriggered {
			continue
		}

		if alert.CreatedAt.After(threshold) {
			continue
		}

		if err := s.Escalate(ctx, alert.ID.String(), "auto escalation: acknowledgement timeout exceeded"); err != nil {
			span.RecordError(err)
			continue
		}
		escalated++
	}

	span.AddEvent("auto_escalation_completed", trace.WithAttributes(
		attribute.Int("escalated_count", escalated),
	))

	return nil
}

// GetEscalationLevel возвращает текущий уровень эскалации для алерта
func (s *EscalationService) GetEscalationLevel(ctx context.Context, alertID string) (int, error) {
	ctx, span := s.tracer.Start(ctx, "EscalationService.GetEscalationLevel")
	defer span.End()

	span.SetAttributes(attribute.String("alert_id", alertID))

	latest, err := s.escalationRepo.GetLatestByAlertID(ctx, alertID)
	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get latest escalation: %v", err)
	}

	span.AddEvent("escalation_level", trace.WithAttributes(
		attribute.Int("level", latest.Level),
	))

	return latest.Level, nil
}

// Escalate создаёт запись эскалации для алерта
func (s *EscalationService) Escalate(ctx context.Context, alertID, reason string) error {
	ctx, span := s.tracer.Start(ctx, "EscalationService.Escalate")
	defer span.End()

	span.SetAttributes(
		attribute.String("alert_id", alertID),
		attribute.String("reason", reason),
	)

	currentLevel, err := s.GetEscalationLevel(ctx, alertID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get current escalation level: %v", err)
	}

	escalation := &model.AlertEscalation{
		ID:             uuid.New(),
		AlertID:        uuid.MustParse(alertID),
		Level:          currentLevel + 1,
		Reason:         reason,
		TimeoutMinutes: int(s.timeout.Minutes()),
		CreatedAt:      time.Now(),
	}

	if err := s.escalationRepo.Create(ctx, escalation); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create escalation record: %v", err)
	}

	span.AddEvent("escalation_created", trace.WithAttributes(
		attribute.String("escalation_id", escalation.ID.String()),
		attribute.Int("level", escalation.Level),
	))

	if s.auditService != nil {
		if err := s.auditService.Log(ctx, model.AuditActionAlertRetriggered, "alert_escalation", escalation.ID.String(), uuid.Nil, map[string]any{
			"alert_id": alertID,
			"level":    escalation.Level,
			"reason":   reason,
		}); err != nil {
			slog.Default().Warn("failed to record audit event", "error", err)
		}
	}

	return nil
}
