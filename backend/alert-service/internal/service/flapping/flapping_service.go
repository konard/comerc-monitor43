package flapping

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
	FlappingWindow     = 10 * time.Minute
	FlappingThreshold  = 5
	FlappingExitPeriod = 15 * time.Minute
)

// FlappingCheckResult содержит результат проверки флаппинга
type FlappingCheckResult struct {
	IsFlapping bool
	FlapCount  int
	Threshold  int
	Window     time.Duration
}

// FlappingService управляет обнаружением флаппинга мониторов
type FlappingService struct {
	statusChangeRepo repository.MonitorStatusChangeRepository
	alertRepo        repository.AlertRepository
	auditService     *audit.AuditService
	window           time.Duration
	threshold        int
	exitPeriod       time.Duration
	tracer           trace.Tracer
	metrics          *apptelemetry.Metrics
}

// FlappingOption задаёт опциональные параметры FlappingService.
type FlappingOption func(*FlappingService)

// WithFlappingWindow переопределяет окно подсчёта смен статусов.
func WithFlappingWindow(d time.Duration) FlappingOption {
	return func(s *FlappingService) {
		if d > 0 {
			s.window = d
		}
	}
}

// WithFlappingThreshold переопределяет порог числа смен статусов.
func WithFlappingThreshold(n int) FlappingOption {
	return func(s *FlappingService) {
		if n > 0 {
			s.threshold = n
		}
	}
}

// WithFlappingExitPeriod переопределяет период выхода из состояния флаппинга.
func WithFlappingExitPeriod(d time.Duration) FlappingOption {
	return func(s *FlappingService) {
		if d > 0 {
			s.exitPeriod = d
		}
	}
}

// NewFlappingService создаёт новый сервис обнаружения флаппинга
func NewFlappingService(
	statusChangeRepo repository.MonitorStatusChangeRepository,
	alertRepo repository.AlertRepository,
	auditService *audit.AuditService,
	tracer trace.Tracer,
	metrics *apptelemetry.Metrics,
	opts ...FlappingOption,
) *FlappingService {
	s := &FlappingService{
		statusChangeRepo: statusChangeRepo,
		alertRepo:        alertRepo,
		auditService:     auditService,
		window:           FlappingWindow,
		threshold:        FlappingThreshold,
		exitPeriod:       FlappingExitPeriod,
		tracer:           tracer,
		metrics:          metrics,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RecordStatusChange записывает смену статуса и проверяет флаппинг
func (s *FlappingService) RecordStatusChange(ctx context.Context, userID, monitorID uuid.UUID, oldStatus, newStatus string) error {
	ctx, span := s.tracer.Start(ctx, "FlappingService.RecordStatusChange")
	defer span.End()

	span.SetAttributes(
		attribute.String("monitor_id", monitorID.String()),
		attribute.String("old_status", oldStatus),
		attribute.String("new_status", newStatus),
	)

	change := &model.MonitorStatusChange{
		ID:        uuid.New(),
		MonitorID: monitorID,
		UserID:    userID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		CreatedAt: time.Now(),
	}

	if err := s.statusChangeRepo.Create(ctx, change); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to record monitor status change: %v", err)
	}

	result, err := s.CheckFlapping(ctx, monitorID.String())
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to check flapping after status change: %v", err)
	}

	if result.IsFlapping {
		span.AddEvent("flapping_detected", trace.WithAttributes(
			attribute.Int("flap_count", result.FlapCount),
			attribute.String("window", result.Window.String()),
		))

		if err := s.createFlappingAlert(ctx, userID, monitorID, result.FlapCount); err != nil {
			span.RecordError(err)
			// Не останавливаем выполнение — ошибка создания алерта не критична
		}

		if s.auditService != nil {
			if err := s.auditService.Log(ctx, model.AuditActionFlappingDetected, "monitor", monitorID.String(), userID, map[string]any{
				"flap_count": result.FlapCount,
				"window":     result.Window.String(),
			}); err != nil {
				slog.Default().Warn("failed to record audit event", "error", err)
			}
		}
	}

	return nil
}

// createFlappingAlert создаёт алерт типа flapping при обнаружении флаппинга
func (s *FlappingService) createFlappingAlert(ctx context.Context, userID, monitorID uuid.UUID, flapCount int) error {
	now := time.Now()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    userID,
		MonitorID: monitorID,
		Status:    model.AlertStatusTriggered,
		Type:      model.AlertTypeFlapping,
		Enabled:   true,
		Config: map[string]any{
			"flap_count": flapCount,
			"window":     s.window.String(),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.alertRepo.Create(ctx, alert); err != nil {
		return fmt.Errorf("failed to create flapping alert: %v", err)
	}

	return nil
}

// CheckFlapping проверяет, находится ли монитор в состоянии флаппинга
func (s *FlappingService) CheckFlapping(ctx context.Context, monitorID string) (*FlappingCheckResult, error) {
	ctx, span := s.tracer.Start(ctx, "FlappingService.CheckFlapping")
	defer span.End()

	span.SetAttributes(attribute.String("monitor_id", monitorID))

	count, err := s.statusChangeRepo.CountInWindow(ctx, monitorID, s.window)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count status changes in window: %v", err)
	}

	result := &FlappingCheckResult{
		IsFlapping: count >= s.threshold,
		FlapCount:  count,
		Threshold:  s.threshold,
		Window:     s.window,
	}

	span.AddEvent("flapping_check_result", trace.WithAttributes(
		attribute.Bool("is_flapping", result.IsFlapping),
		attribute.Int("flap_count", count),
	))

	return result, nil
}

// CheckExitFlapping проверяет, можно ли выйти из состояния флаппинга
func (s *FlappingService) CheckExitFlapping(ctx context.Context, monitorID string) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "FlappingService.CheckExitFlapping")
	defer span.End()

	span.SetAttributes(attribute.String("monitor_id", monitorID))

	latest, err := s.statusChangeRepo.GetLatestByMonitorID(ctx, monitorID)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to get latest status change: %v", err)
	}

	canExit := time.Since(latest.CreatedAt) >= s.exitPeriod

	span.AddEvent("exit_flapping_check", trace.WithAttributes(
		attribute.Bool("can_exit", canExit),
		attribute.String("last_change", latest.CreatedAt.String()),
	))

	return canExit, nil
}
