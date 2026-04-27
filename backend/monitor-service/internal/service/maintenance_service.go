package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
	"github.com/raul/monitor/backend/monitor-service/pkg/logger"
)

// MaintenanceWindowConfig содержит конфигурацию для окон обслуживания.
type MaintenanceWindowConfig struct {
	MinDurationMinutes   int
	MaxDurationHours     int
	MaxWindowsFreeTier   int
	MaxWindowsProTier    int
	MaxWindowsEnterprise int
	MaxMonitorsPerWindow int
}

// DefaultMaintenanceWindowConfig возвращает конфигурацию по умолчанию.
func DefaultMaintenanceWindowConfig() MaintenanceWindowConfig {
	return MaintenanceWindowConfig{
		MinDurationMinutes:   1,
		MaxDurationHours:     24,
		MaxWindowsFreeTier:   5,
		MaxWindowsProTier:    20,
		MaxWindowsEnterprise: 100,
		MaxMonitorsPerWindow: 50,
	}
}

type MaintenanceService struct {
	windowRepo  interfaces.MaintenanceWindowRepository
	monitorRepo interfaces.MonitorRepository
	auditRepo   interfaces.AuditRepository
	cfg         MaintenanceWindowConfig
	logger      *slog.Logger
	metrics     *MaintenanceMetrics
}

// NewMaintenanceService создаёт новый сервис окон обслуживания.
func NewMaintenanceService(
	windowRepo interfaces.MaintenanceWindowRepository,
	monitorRepo interfaces.MonitorRepository,
	auditRepo interfaces.AuditRepository,
	cfg MaintenanceWindowConfig,
) *MaintenanceService {
	return &MaintenanceService{
		windowRepo:  windowRepo,
		monitorRepo: monitorRepo,
		auditRepo:   auditRepo,
		cfg:         cfg,
		logger:      logger.DefaultLogger,
		metrics:     NewMaintenanceMetrics(),
	}
}

// CreateMaintenanceWindow создаёт новое окно обслуживания (uc_08_01_01).
func (s *MaintenanceService) CreateMaintenanceWindow(ctx context.Context, req *dto.CreateMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.CreateMaintenanceWindow")
	defer span.End()

	tracing.AddEvent(ctx, "create_maintenance_window.start", map[string]any{
		"user_id": req.UserID,
		"name":    req.Name,
	})

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Проверяем права на создание глобального окна (uc_08_01_06a)
	if req.IsGlobal && req.UserRole != "ADMIN" {
		err := errors.New("INSUFFICIENT_PERMISSIONS: only admins can create global windows")
		s.logger.Warn("attempt to create global window by non-admin",
			"user_id", userID,
			"role", req.UserRole,
		)
		tracing.RecordError(span, err)

		// Записываем в audit log
		s.auditCreateInPastDenied(ctx, userID, req.Name)

		return nil, err
	}

	// Парсим monitor IDs
	var monitorIDs []uuid.UUID
	for _, idStr := range req.MonitorIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, errors.Wrap(err, "invalid monitor_id")
		}
		monitorIDs = append(monitorIDs, id)
	}

	// Проверяем лимит окон для аккаунта (uc_08_01_10)
	maxWindows := s.getMaxWindowsForTier(req.UserTier)
	currentCount, err := s.windowRepo.CountByUserID(ctx, userID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to count windows")
	}

	if currentCount >= maxWindows {
		err := errors.New("MAINTENANCE_WINDOW_LIMIT_REACHED")
		s.logger.Warn("maintenance window limit reached",
			"user_id", userID,
			"current", currentCount,
			"limit", maxWindows,
		)

		s.auditLimitReached(ctx, userID, maxWindows, currentCount)

		tracing.RecordError(span, err)
		return nil, err
	}

	// Создаём окно с валидацией
	cfg := domain.MaintenanceWindowConfig{
		MinDurationMinutes:   s.cfg.MinDurationMinutes,
		MaxDurationHours:     s.cfg.MaxDurationHours,
		MaxMonitorsPerWindow: s.cfg.MaxMonitorsPerWindow,
	}

	recurrence := domain.RecurrenceType(req.Recurrence)
	window, err := domain.NewMaintenanceWindow(
		userID,
		req.Name,
		req.StartTime,
		req.EndTime,
		recurrence,
		req.IsGlobal,
		req.PauseMonitoring,
		req.SuppressAlerts,
		req.SafeMode,
		monitorIDs,
		cfg,
	)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to create maintenance window")
	}

	// Проверяем пересечение с существующими окнами (uc_08_01_07b)
	if len(monitorIDs) > 0 && !req.IsGlobal {
		hasOverlap, err := s.windowRepo.CheckOverlap(ctx, userID, monitorIDs, req.StartTime, req.EndTime, nil)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, errors.Wrap(err, "failed to check overlap")
		}

		if hasOverlap {
			err := errors.New("OVERLAPPING_MAINTENANCE_WINDOWS")
			s.logger.Warn("overlapping maintenance windows detected",
				"user_id", userID,
				"start_time", req.StartTime,
				"end_time", req.EndTime,
			)

			s.auditOverlappingRejected(ctx, userID, req.StartTime, req.EndTime)

			tracing.RecordError(span, err)
			return nil, err
		}
	}

	// Сохраняем в БД
	repoWindow := s.toRepoWindow(window)
	if err := s.windowRepo.Create(ctx, repoWindow); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to save maintenance window")
	}

	// Записываем в audit log (uc_08_01_01)
	s.auditCreate(ctx, userID, window)

	s.logger.Info("maintenance window created",
		"window_id", window.ID,
		"user_id", userID,
		"name", window.Name,
		"start_time", window.StartTime,
		"end_time", window.EndTime,
	)

	tracing.AddEvent(ctx, "create_maintenance_window.success", map[string]any{
		"window_id": window.ID,
	})

	return s.toResponseDTO(window), nil
}

// UpdateMaintenanceWindow обновляет окно обслуживания (uc_08_01_19).
func (s *MaintenanceService) UpdateMaintenanceWindow(ctx context.Context, req *dto.UpdateMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.UpdateMaintenanceWindow")
	defer span.End()

	tracing.AddEvent(ctx, "update_maintenance_window.start", map[string]any{
		"window_id": req.ID,
	})

	windowID, err := uuid.Parse(req.ID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid window_id")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Получаем окно
	repoWindow, err := s.windowRepo.GetByID(ctx, windowID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get maintenance window")
	}

	if repoWindow == nil {
		err := errors.New("maintenance window not found")
		tracing.RecordError(span, err)
		return nil, err
	}

	// Проверяем права доступа
	if repoWindow.UserID != userID {
		err := errors.New("INSUFFICIENT_PERMISSIONS: window belongs to another user")
		tracing.RecordError(span, err)
		return nil, err
	}

	// Конвертируем в domain model
	domainWindow := s.toDomainWindow(repoWindow)

	// Обновляем
	if err := domainWindow.UpdateDetails(req.Name, req.StartTime, req.EndTime, req.Version); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to update window details")
	}

	// Проверяем пересечение
	monitorIDs, err := s.windowRepo.GetWindowMonitors(ctx, windowID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get window monitors")
	}
	if len(monitorIDs) > 0 {
		hasOverlap, err := s.windowRepo.CheckOverlap(ctx, userID, monitorIDs, req.StartTime, req.EndTime, &windowID)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, errors.Wrap(err, "failed to check overlap")
		}

		if hasOverlap {
			err := errors.New("OVERLAPPING_MAINTENANCE_WINDOWS")
			tracing.RecordError(span, err)
			return nil, err
		}
	}

	// Сохраняем
	updatedWindow := s.toRepoWindow(domainWindow)
	if err := s.windowRepo.Update(ctx, updatedWindow); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to update maintenance window")
	}

	// Audit log
	s.auditUpdate(ctx, userID, domainWindow, map[string]any{
		"changed_fields": []string{"start_time", "end_time"},
	})

	s.logger.Info("maintenance window updated",
		"window_id", windowID,
		"user_id", userID,
	)

	tracing.AddEvent(ctx, "update_maintenance_window.success", map[string]any{
		"window_id": windowID,
	})

	return s.toResponseDTO(domainWindow), nil
}

// ListMaintenanceWindows возвращает список окон (uc_08_01_20).
func (s *MaintenanceService) ListMaintenanceWindows(ctx context.Context, req *dto.ListMaintenanceWindowsRequest) (*dto.ListMaintenanceWindowsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.ListMaintenanceWindows")
	defer span.End()

	tracing.AddEvent(ctx, "list_maintenance_windows.start", map[string]any{
		"user_id": req.UserID,
	})

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	offset := (req.Page - 1) * req.PageSize

	windows, err := s.windowRepo.GetByUserID(ctx, userID, req.Status, req.PageSize, offset)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to list maintenance windows")
	}

	total, err := s.windowRepo.CountByUserID(ctx, userID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to count maintenance windows")
	}

	response := &dto.ListMaintenanceWindowsResponse{
		Windows:  make([]*dto.MaintenanceWindowResponse, 0, len(windows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	for _, w := range windows {
		domainWindow := s.toDomainWindow(w)
		response.Windows = append(response.Windows, s.toResponseDTO(domainWindow))
	}

	s.auditList(ctx, userID, len(windows))

	tracing.AddEvent(ctx, "list_maintenance_windows.success", map[string]any{
		"count": len(windows),
	})

	return response, nil
}

// CancelMaintenanceWindow отменяет окно (uc_08_01_25a, uc_08_01_25b).
func (s *MaintenanceService) CancelMaintenanceWindow(ctx context.Context, req *dto.CancelMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.CancelMaintenanceWindow")
	defer span.End()

	tracing.AddEvent(ctx, "cancel_maintenance_window.start", map[string]any{
		"window_id": req.ID,
	})

	windowID, err := uuid.Parse(req.ID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid window_id")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Получаем окно
	repoWindow, err := s.windowRepo.GetByID(ctx, windowID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get maintenance window")
	}

	if repoWindow == nil {
		err := errors.New("maintenance window not found")
		tracing.RecordError(span, err)
		return nil, err
	}

	// Проверяем права
	if repoWindow.UserID != userID {
		err := errors.New("INSUFFICIENT_PERMISSIONS")
		tracing.RecordError(span, err)
		return nil, err
	}

	domainWindow := s.toDomainWindow(repoWindow)

	// Отменяем
	if err := domainWindow.Cancel(); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to cancel window")
	}

	// Сохраняем
	updatedWindow := s.toRepoWindow(domainWindow)
	if err := s.windowRepo.Update(ctx, updatedWindow); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to update maintenance window")
	}

	// Audit log
	s.auditCancel(ctx, userID, domainWindow)

	s.logger.Info("maintenance window cancelled",
		"window_id", windowID,
		"user_id", userID,
		"reason", req.CancellationReason,
	)

	tracing.AddEvent(ctx, "cancel_maintenance_window.success", map[string]any{
		"window_id": windowID,
	})

	return s.toResponseDTO(domainWindow), nil
}

// GetMaintenanceWindowHistory возвращает историю окон (uc_08_01_21).
func (s *MaintenanceService) GetMaintenanceWindowHistory(ctx context.Context, req *dto.GetMaintenanceWindowHistoryRequest) (*dto.ListMaintenanceWindowsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.GetMaintenanceWindowHistory")
	defer span.End()

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	var monitorIDs []uuid.UUID
	for _, idStr := range req.MonitorIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, errors.Wrap(err, "invalid monitor_id")
		}
		monitorIDs = append(monitorIDs, id)
	}

	offset := (req.Page - 1) * req.PageSize

	windows, err := s.windowRepo.GetHistory(ctx, userID, req.StartDate, req.EndDate, monitorIDs, req.PageSize, offset)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get maintenance window history")
	}

	response := &dto.ListMaintenanceWindowsResponse{
		Windows:  make([]*dto.MaintenanceWindowResponse, 0, len(windows)),
		Total:    len(windows),
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	for _, w := range windows {
		domainWindow := s.toDomainWindow(w)
		response.Windows = append(response.Windows, s.toResponseDTO(domainWindow))
	}

	return response, nil
}

// DeleteMaintenanceWindow удаляет окно (uc_08_01_24).
func (s *MaintenanceService) DeleteMaintenanceWindow(ctx context.Context, windowID, userID string) error {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.DeleteMaintenanceWindow")
	defer span.End()

	id, err := uuid.Parse(windowID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid window_id")
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid user_id")
	}

	// Получаем окно для audit log
	repoWindow, err := s.windowRepo.GetByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to get maintenance window")
	}

	if repoWindow == nil {
		err := errors.New("maintenance window not found")
		tracing.RecordError(span, err)
		return err
	}

	if repoWindow.UserID != uid {
		err := errors.New("INSUFFICIENT_PERMISSIONS")
		tracing.RecordError(span, err)
		return err
	}

	// Удаляем
	if err := s.windowRepo.Delete(ctx, id); err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to delete maintenance window")
	}

	// Audit log
	s.auditDelete(ctx, uid, repoWindow)

	s.logger.Info("maintenance window deleted",
		"window_id", id,
		"user_id", uid,
	)

	return nil
}

// ActivateScheduledWindows активирует окна с наступившим start_time (uc_08_01_27, uc_08_01_30c).
func (s *MaintenanceService) ActivateScheduledWindows(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.ActivateScheduledWindows")
	defer span.End()

	windows, err := s.windowRepo.GetWindowsRequiringActivation(ctx, time.Now())
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to get windows requiring activation")
	}

	for _, repoWindow := range windows {
		domainWindow := s.toDomainWindow(repoWindow)
		oldStatus := string(domainWindow.Status)

		if err := domainWindow.Activate(); err != nil {
			s.logger.Warn("failed to activate window",
				"window_id", domainWindow.ID,
				"error", err,
			)
			continue
		}

		updated := s.toRepoWindow(domainWindow)
		if err := s.windowRepo.Update(ctx, updated); err != nil {
			s.logger.Error("failed to update window status to ACTIVE",
				"window_id", domainWindow.ID,
				"error", err,
			)
			continue
		}

		s.auditStatusChange(ctx, domainWindow.UserID, domainWindow, oldStatus, "ACTIVE", 0)

		s.logger.Info("maintenance window activated",
			"window_id", domainWindow.ID,
			"user_id", domainWindow.UserID,
		)
	}

	return nil
}

// CompleteActiveWindows завершает окна с истёкшим end_time (uc_08_01_28, uc_08_01_30d).
func (s *MaintenanceService) CompleteActiveWindows(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceService.CompleteActiveWindows")
	defer span.End()

	windows, err := s.windowRepo.GetWindowsRequiringCompletion(ctx, time.Now())
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to get windows requiring completion")
	}

	for _, repoWindow := range windows {
		domainWindow := s.toDomainWindow(repoWindow)
		oldStatus := string(domainWindow.Status)
		durationMinutes := int(domainWindow.Duration().Minutes())

		if err := domainWindow.Complete(); err != nil {
			s.logger.Warn("failed to complete window",
				"window_id", domainWindow.ID,
				"error", err,
			)
			continue
		}

		updated := s.toRepoWindow(domainWindow)
		if err := s.windowRepo.Update(ctx, updated); err != nil {
			s.logger.Error("failed to update window status to COMPLETED",
				"window_id", domainWindow.ID,
				"error", err,
			)
			continue
		}

		s.auditStatusChange(ctx, domainWindow.UserID, domainWindow, oldStatus, "COMPLETED", durationMinutes)

		s.logger.Info("maintenance window completed",
			"window_id", domainWindow.ID,
			"user_id", domainWindow.UserID,
			"duration_minutes", durationMinutes,
		)
	}

	return nil
}

// Helper methods

func (s *MaintenanceService) getMaxWindowsForTier(tier string) int {
	switch tier {
	case "Pro":
		return s.cfg.MaxWindowsProTier
	case "Enterprise":
		return s.cfg.MaxWindowsEnterprise
	default:
		return s.cfg.MaxWindowsFreeTier
	}
}

func (s *MaintenanceService) toRepoWindow(window *domain.MaintenanceWindow) *interfaces.MaintenanceWindow {
	monitorIDs := make([]string, len(window.MonitorIDs))
	for i, id := range window.MonitorIDs {
		monitorIDs[i] = id.String()
	}

	return &interfaces.MaintenanceWindow{
		ID:              window.ID,
		UserID:          window.UserID,
		Name:            window.Name,
		StartTime:       window.StartTime,
		EndTime:         window.EndTime,
		Status:          string(window.Status),
		Recurrence:      string(window.Recurrence),
		IsGlobal:        window.IsGlobal,
		PauseMonitoring: window.PauseMonitoring,
		SuppressAlerts:  window.SuppressAlerts,
		SafeMode:        window.SafeMode,
		CreatedAt:       window.CreatedAt,
		UpdatedAt:       window.UpdatedAt,
		ActivatedAt:     window.ActivatedAt,
		CompletedAt:     window.CompletedAt,
		Version:         window.Version,
	}
}

func (s *MaintenanceService) toDomainWindow(window *interfaces.MaintenanceWindow) *domain.MaintenanceWindow {
	return &domain.MaintenanceWindow{
		ID:              window.ID,
		UserID:          window.UserID,
		Name:            window.Name,
		StartTime:       window.StartTime,
		EndTime:         window.EndTime,
		Status:          domain.MaintenanceWindowStatus(window.Status),
		Recurrence:      domain.RecurrenceType(window.Recurrence),
		IsGlobal:        window.IsGlobal,
		PauseMonitoring: window.PauseMonitoring,
		SuppressAlerts:  window.SuppressAlerts,
		SafeMode:        window.SafeMode,
		CreatedAt:       window.CreatedAt,
		UpdatedAt:       window.UpdatedAt,
		ActivatedAt:     window.ActivatedAt,
		CompletedAt:     window.CompletedAt,
		Version:         window.Version,
	}
}

func (s *MaintenanceService) toResponseDTO(window *domain.MaintenanceWindow) *dto.MaintenanceWindowResponse {
	monitorIDs := make([]string, len(window.MonitorIDs))
	for i, id := range window.MonitorIDs {
		monitorIDs[i] = id.String()
	}

	return &dto.MaintenanceWindowResponse{
		ID:              window.ID.String(),
		UserID:          window.UserID.String(),
		Name:            window.Name,
		StartTime:       window.StartTime,
		EndTime:         window.EndTime,
		Status:          string(window.Status),
		Recurrence:      string(window.Recurrence),
		IsGlobal:        window.IsGlobal,
		PauseMonitoring: window.PauseMonitoring,
		SuppressAlerts:  window.SuppressAlerts,
		SafeMode:        window.SafeMode,
		MonitorIDs:      monitorIDs,
		CreatedAt:       window.CreatedAt,
		UpdatedAt:       window.UpdatedAt,
		ActivatedAt:     window.ActivatedAt,
		CompletedAt:     window.CompletedAt,
		Version:         window.Version,
	}
}

// Audit log methods

func (s *MaintenanceService) auditCreate(ctx context.Context, userID uuid.UUID, window *domain.MaintenanceWindow) {
	entry := &interfaces.AuditLogEntry{
		ID:     uuid.New(),
		UserID: userID,
		Action: "maintenance_window.created",
		NewValues: map[string]any{
			"window_id":     window.ID.String(),
			"window_name":   window.Name,
			"recurrence":    string(window.Recurrence),
			"monitor_count": len(window.MonitorIDs),
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditUpdate(ctx context.Context, userID uuid.UUID, window *domain.MaintenanceWindow, details map[string]any) {
	entry := &interfaces.AuditLogEntry{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    "maintenance_window.updated",
		NewValues: details,
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditDelete(ctx context.Context, userID uuid.UUID, window *interfaces.MaintenanceWindow) {
	entry := &interfaces.AuditLogEntry{
		ID:     uuid.New(),
		UserID: userID,
		Action: "maintenance_window.deleted",
		OldValues: map[string]any{
			"window_id":     window.ID.String(),
			"window_status": window.Status,
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditCancel(ctx context.Context, userID uuid.UUID, window *domain.MaintenanceWindow) {
	entry := &interfaces.AuditLogEntry{
		ID:     uuid.New(),
		UserID: userID,
		Action: "maintenance_window.cancelled",
		OldValues: map[string]any{
			"old_status": string(window.Status),
		},
		NewValues: map[string]any{
			"new_status": "CANCELLED",
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditList(ctx context.Context, userID uuid.UUID, count int) {
	entry := &interfaces.AuditLogEntry{
		ID:     uuid.New(),
		UserID: userID,
		Action: "maintenance_windows.listed",
		NewValues: map[string]any{
			"window_count": count,
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditCreateInPastDenied(ctx context.Context, userID uuid.UUID, name string) {
	entry := &interfaces.AuditLogEntry{
		ID:     uuid.New(),
		UserID: userID,
		Action: "global_maintenance_window_creation_denied",
		NewValues: map[string]any{
			"window_name": name,
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditLimitReached(ctx context.Context, userID uuid.UUID, limit, current int) {
	entry := &interfaces.AuditLogEntry{
		ID:     uuid.New(),
		UserID: userID,
		Action: "maintenance_window_limit_reached",
		NewValues: map[string]any{
			"limit":   limit,
			"current": current,
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditOverlappingRejected(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time) {
	entry := &interfaces.AuditLogEntry{
		ID:     uuid.New(),
		UserID: userID,
		Action: "overlapping_maintenance_windows_rejected",
		NewValues: map[string]any{
			"start_time": startTime.Format(time.RFC3339),
			"end_time":   endTime.Format(time.RFC3339),
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}

func (s *MaintenanceService) auditStatusChange(ctx context.Context, userID uuid.UUID, window *domain.MaintenanceWindow, oldStatus, newStatus string, durationMinutes int) {
	values := map[string]any{
		"window_id":    window.ID.String(),
		"old_status":   oldStatus,
		"new_status":   newStatus,
		"trigger_type": "AUTOMATIC",
		"initiator":    "SYSTEM",
	}
	if durationMinutes > 0 {
		values["duration_minutes"] = durationMinutes
	}

	entry := &interfaces.AuditLogEntry{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    "maintenance_window.status_changed",
		NewValues: values,
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", "error", err)
	}
}
