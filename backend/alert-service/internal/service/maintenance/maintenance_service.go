package maintenance

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
	maxWindowDuration = 24 * time.Hour
	minWindowDuration = 1 * time.Minute
)

// CreateMaintenanceWindowRequest содержит параметры для создания окна обслуживания
type CreateMaintenanceWindowRequest struct {
	UserID          uuid.UUID
	UserRole        string
	Name            string
	StartsAt        time.Time
	EndsAt          time.Time
	Recurrence      model.RecurrenceType
	IsGlobal        bool
	PauseMonitoring bool
	SuppressAlerts  bool
	SafeMode        bool
	MonitorIDs      []string
}

// UpdateMaintenanceWindowRequest содержит параметры для обновления окна обслуживания
type UpdateMaintenanceWindowRequest struct {
	ID       string
	UserID   uuid.UUID
	Name     string
	StartsAt *time.Time
	EndsAt   *time.Time
	Version  int32
}

// ListMaintenanceWindowsRequest содержит параметры для получения списка окон обслуживания
type ListMaintenanceWindowsRequest struct {
	UserID uuid.UUID
	Filter model.MaintenanceWindowFilter
}

// MaintenanceService управляет окнами обслуживания
type MaintenanceService struct {
	mwRepo       repository.MaintenanceWindowRepository
	auditService *audit.AuditService
	tracer       trace.Tracer
	metrics      *apptelemetry.Metrics
}

// NewMaintenanceService создаёт новый сервис окон обслуживания
func NewMaintenanceService(
	mwRepo repository.MaintenanceWindowRepository,
	tracer trace.Tracer,
	metrics *apptelemetry.Metrics,
) *MaintenanceService {
	return &MaintenanceService{
		mwRepo:  mwRepo,
		tracer:  tracer,
		metrics: metrics,
	}
}

// WithAuditService добавляет сервис аудита
func (s *MaintenanceService) WithAuditService(auditSvc *audit.AuditService) *MaintenanceService {
	s.auditService = auditSvc
	return s
}

// CreateMaintenanceWindow создаёт новое окно обслуживания с валидацией
func (s *MaintenanceService) CreateMaintenanceWindow(ctx context.Context, req CreateMaintenanceWindowRequest) (*model.MaintenanceWindow, error) {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.CreateMaintenanceWindow")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID.String()),
		attribute.Bool("is_global", req.IsGlobal),
	)

	if err := s.validateTimeRange(req.StartsAt, req.EndsAt); err != nil {
		return nil, err
	}

	// Глобальные окна доступны только для администраторов
	if req.IsGlobal && req.UserRole != "ADMIN" {
		s.logAudit(ctx, model.AuditActionGlobalMaintenanceWindowDenied, "", req.UserID, map[string]any{
			"required_role": "ADMIN",
			"current_role":  req.UserRole,
		})
		return nil, model.ErrInsufficientPermissions
	}

	// Проверяем пересечения для не-глобальных окон
	if !req.IsGlobal && len(req.MonitorIDs) > 0 {
		overlaps, existing, err := s.mwRepo.CheckOverlapping(ctx, req.MonitorIDs, req.StartsAt, req.EndsAt, "")
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to check overlapping windows: %v", err)
		}
		if overlaps {
			s.logAudit(ctx, model.AuditActionMaintenanceWindowOverlapping, "", req.UserID, map[string]any{
				"existing_window_id": existing.ID.String(),
			})
			return nil, model.ErrOverlappingMaintenanceWindows
		}
	}

	recurrence := req.Recurrence
	if recurrence == "" {
		recurrence = model.RecurrenceTypeOnce
	}

	now := time.Now().UTC()
	mw := &model.MaintenanceWindow{
		ID:              uuid.New(),
		UserID:          req.UserID,
		Name:            req.Name,
		Status:          model.MaintenanceStatusScheduled,
		Recurrence:      recurrence,
		IsGlobal:        req.IsGlobal,
		PauseMonitoring: req.PauseMonitoring,
		SuppressAlerts:  req.SuppressAlerts,
		SafeMode:        req.SafeMode,
		MonitorIDs:      req.MonitorIDs,
		StartsAt:        req.StartsAt,
		EndsAt:          req.EndsAt,
		Version:         0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.mwRepo.Create(ctx, mw); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create maintenance window: %v", err)
	}

	span.AddEvent("maintenance_window_created", trace.WithAttributes(
		attribute.String("window_id", mw.ID.String()),
	))

	auditAction := model.AuditActionMaintenanceWindowCreated
	if req.IsGlobal {
		auditAction = model.AuditActionGlobalMaintenanceWindowCreated
	}

	s.logAudit(ctx, auditAction, mw.ID.String(), req.UserID, map[string]any{
		"window_name":   mw.Name,
		"recurrence":    string(mw.Recurrence),
		"monitor_count": len(req.MonitorIDs),
		"is_global":     mw.IsGlobal,
	})

	return mw, nil
}

// UpdateMaintenanceWindow обновляет окно обслуживания с оптимистичной блокировкой
func (s *MaintenanceService) UpdateMaintenanceWindow(ctx context.Context, req UpdateMaintenanceWindowRequest) (*model.MaintenanceWindow, error) {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.UpdateMaintenanceWindow")
	defer span.End()

	span.SetAttributes(attribute.String("window_id", req.ID))

	mw, err := s.mwRepo.GetByID(ctx, req.ID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if mw.Status != model.MaintenanceStatusScheduled {
		return nil, model.ErrMaintenanceWindowNotScheduled
	}

	if req.Name != "" {
		mw.Name = req.Name
	}
	if req.StartsAt != nil {
		mw.StartsAt = *req.StartsAt
	}
	if req.EndsAt != nil {
		mw.EndsAt = *req.EndsAt
	}

	if err := s.validateTimeRange(mw.StartsAt, mw.EndsAt); err != nil {
		return nil, err
	}

	mw.Version = int(req.Version)
	mw.UpdatedAt = time.Now().UTC()

	if err := s.mwRepo.Update(ctx, mw); err != nil {
		span.RecordError(err)
		return nil, err
	}

	s.logAudit(ctx, model.AuditActionMaintenanceWindowUpdated, mw.ID.String(), req.UserID, map[string]any{
		"window_name": mw.Name,
	})

	// Получаем обновлённую запись с актуальным version
	updated, err := s.mwRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload updated window: %v", err)
	}

	return updated, nil
}

// ListMaintenanceWindows возвращает список окон обслуживания пользователя
func (s *MaintenanceService) ListMaintenanceWindows(ctx context.Context, req ListMaintenanceWindowsRequest) ([]*model.MaintenanceWindow, int, error) {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.ListMaintenanceWindows")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", req.UserID.String()))

	windows, total, err := s.mwRepo.List(ctx, req.UserID.String(), req.Filter)
	if err != nil {
		span.RecordError(err)
		return nil, 0, fmt.Errorf("failed to list maintenance windows: %v", err)
	}

	return windows, total, nil
}

// DeleteMaintenanceWindow удаляет запланированное окно обслуживания
func (s *MaintenanceService) DeleteMaintenanceWindow(ctx context.Context, id string, userID uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.DeleteMaintenanceWindow")
	defer span.End()

	span.SetAttributes(attribute.String("window_id", id))

	mw, err := s.mwRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}

	if mw.Status != model.MaintenanceStatusScheduled {
		return model.ErrMaintenanceWindowNotScheduled
	}

	if err := s.mwRepo.Delete(ctx, id); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete maintenance window: %v", err)
	}

	s.logAudit(ctx, model.AuditActionMaintenanceWindowDeleted, id, userID, map[string]any{
		"window_name": mw.Name,
	})

	return nil
}

// CancelMaintenanceWindow отменяет активное или запланированное окно обслуживания
func (s *MaintenanceService) CancelMaintenanceWindow(ctx context.Context, id string, userID uuid.UUID, reason string) (*model.MaintenanceWindow, error) {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.CancelMaintenanceWindow")
	defer span.End()

	span.SetAttributes(attribute.String("window_id", id))

	mw, err := s.mwRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if mw.Status != model.MaintenanceStatusScheduled && mw.Status != model.MaintenanceStatusActive {
		return nil, model.ErrMaintenanceWindowNotScheduled
	}

	now := time.Now().UTC()
	mw.Status = model.MaintenanceStatusCancelled
	mw.CompletedAt = &now
	mw.UpdatedAt = now

	if err := s.mwRepo.Update(ctx, mw); err != nil {
		span.RecordError(err)
		return nil, err
	}

	s.logAudit(ctx, model.AuditActionMaintenanceWindowCancelled, id, userID, map[string]any{
		"window_name": mw.Name,
		"reason":      reason,
	})

	updated, err := s.mwRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reload cancelled window: %v", err)
	}

	return updated, nil
}

// GetMaintenanceWindowHistory возвращает историю окон обслуживания (завершённые и отменённые)
func (s *MaintenanceService) GetMaintenanceWindowHistory(ctx context.Context, req ListMaintenanceWindowsRequest) ([]*model.MaintenanceWindow, int, error) {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.GetMaintenanceWindowHistory")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", req.UserID.String()))

	// История включает только завершённые и отменённые окна
	if req.Filter.Status == "" {
		req.Filter.Status = model.MaintenanceStatusCompleted
	}

	windows, total, err := s.mwRepo.List(ctx, req.UserID.String(), req.Filter)
	if err != nil {
		span.RecordError(err)
		return nil, 0, fmt.Errorf("failed to get maintenance window history: %v", err)
	}

	return windows, total, nil
}

// CreateWindow создаёт окно обслуживания (устаревший метод, сохранён для обратной совместимости)
func (s *MaintenanceService) CreateWindow(ctx context.Context, userID, monitorID uuid.UUID, startsAt, endsAt time.Time, reason *string) error {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.CreateWindow")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("monitor_id", monitorID.String()),
	)

	name := "maintenance"
	if reason != nil {
		name = *reason
	}

	mw := &model.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     userID,
		MonitorID:  &monitorID,
		MonitorIDs: []string{monitorID.String()},
		Name:       name,
		Status:     model.MaintenanceStatusScheduled,
		Recurrence: model.RecurrenceTypeOnce,
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		Reason:     reason,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := s.mwRepo.Create(ctx, mw); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create maintenance window: %v", err)
	}

	span.AddEvent("maintenance_window_created", trace.WithAttributes(
		attribute.String("window_id", mw.ID.String()),
		attribute.String("monitor_id", monitorID.String()),
	))

	return nil
}

// IsUnderMaintenance проверяет, находится ли монитор в окне обслуживания
func (s *MaintenanceService) IsUnderMaintenance(ctx context.Context, monitorID string) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.IsUnderMaintenance")
	defer span.End()

	span.SetAttributes(attribute.String("monitor_id", monitorID))

	underMaintenance, err := s.mwRepo.IsUnderMaintenance(ctx, monitorID)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to check maintenance status: %v", err)
	}

	span.AddEvent("maintenance_check_result", trace.WithAttributes(
		attribute.Bool("is_under_maintenance", underMaintenance),
	))

	return underMaintenance, nil
}

// GetActiveWindow возвращает активное окно обслуживания для монитора
func (s *MaintenanceService) GetActiveWindow(ctx context.Context, monitorID string) (*model.MaintenanceWindow, error) {
	ctx, span := s.tracer.Start(ctx, "MaintenanceService.GetActiveWindow")
	defer span.End()

	span.SetAttributes(attribute.String("monitor_id", monitorID))

	mw, err := s.mwRepo.GetActiveByMonitorID(ctx, monitorID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get active maintenance window: %v", err)
	}

	return mw, nil
}

// validateTimeRange проверяет корректность временного диапазона окна
func (s *MaintenanceService) validateTimeRange(startsAt, endsAt time.Time) error {
	if endsAt.Before(startsAt) || endsAt.Equal(startsAt) {
		return model.ErrEndTimeBeforeStartTime
	}

	// Допуск 30 секунд для компенсации задержки сети и сериализации запроса
	if startsAt.Before(time.Now().Add(-30 * time.Second)) {
		return model.ErrMaintenanceWindowInPast
	}

	duration := endsAt.Sub(startsAt)
	if duration < minWindowDuration {
		return model.ErrMaintenanceWindowDurationTooShort
	}
	if duration > maxWindowDuration {
		return model.ErrMaintenanceWindowDurationExceeded
	}

	return nil
}

// logAudit записывает событие в журнал аудита; ошибки логирования не критичны и только пишутся в slog.
func (s *MaintenanceService) logAudit(ctx context.Context, action model.AuditAction, resourceID string, userID uuid.UUID, fields map[string]any) {
	if s.auditService == nil {
		return
	}
	if err := s.auditService.Log(ctx, action, "maintenance_window", resourceID, userID, fields); err != nil {
		slog.Default().Warn("failed to record audit event", "action", string(action), "error", err)
	}
}
