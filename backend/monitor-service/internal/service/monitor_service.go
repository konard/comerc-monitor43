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
	apperrors "github.com/raul/monitor/backend/monitor-service/pkg/errors"
	"github.com/raul/monitor/backend/monitor-service/pkg/logger"
)

type MonitorService struct {
	monitorRepo interfaces.MonitorRepository
	auditRepo   interfaces.AuditRepository
	authz       *AuthorizationService
	logger      *slog.Logger
}

// NewMonitorService создаёт новый сервис мониторов.
func NewMonitorService(
	monitorRepo interfaces.MonitorRepository,
	auditRepo interfaces.AuditRepository,
) *MonitorService {
	return &MonitorService{
		monitorRepo: monitorRepo,
		auditRepo:   auditRepo,
		authz:       NewAuthorizationService(monitorRepo),
		logger:      logger.DefaultLogger,
	}
}

// CreateMonitor создаёт новый монитор.
func (s *MonitorService) CreateMonitor(ctx context.Context, req *dto.CreateMonitorRequest) (*dto.MonitorResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorService.CreateMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "create_monitor.start", map[string]any{
		"user_id": req.UserID,
		"name":    req.Name,
		"url":     req.URL,
	})

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Проверяем, что монитор с таким именем не существует
	exists, err := s.monitorRepo.ExistsByName(ctx, userID, req.Name)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to check monitor existence")
	}
	if exists {
		err := errors.New("monitor with this name already exists")
		tracing.RecordError(span, err)
		return nil, err
	}

	// Проверяем лимиты мониторов пользователя согласно subscription tier
	if err := s.authz.CanCreateMonitor(ctx, userID, req.Tier); err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Создаём монитор
	monitor, err := domain.NewMonitor(userID, req.Name, req.URL, req.IntervalSeconds)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to create monitor")
	}

	// Устанавливаем дополнительные параметры
	if req.TimeoutSeconds > 0 {
		if req.TimeoutSeconds >= req.IntervalSeconds {
			err := errors.New("timeout must be less than interval")
			tracing.RecordError(span, err)
			return nil, err
		}
		monitor.TimeoutSeconds = req.TimeoutSeconds
	}

	if req.CheckType != "" {
		monitor.CheckType = req.CheckType
	}

	// Парсим рабочие часы
	if req.WorkingHoursStart != "" && req.WorkingHoursEnd != "" {
		workingDays := parseWorkingDays(req.WorkingDays)
		wh, err := domain.NewWorkingHours(req.WorkingHoursStart, req.WorkingHoursEnd, workingDays)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, errors.Wrap(err, "invalid working hours")
		}
		start, end := wh.GetStartEndTime()
		monitor.WorkingHoursStart = start
		monitor.WorkingHoursEnd = end
		monitor.WorkingDays = workingDays
	}

	// Устанавливаем пороги DEGRADED
	if req.DegradedResponseTimeThreshold != nil {
		monitor.DegradedResponseTimeThreshold = req.DegradedResponseTimeThreshold
	}
	if req.DegradedFailureRateThreshold != nil {
		monitor.DegradedFailureRateThreshold = req.DegradedFailureRateThreshold
	}

	// Сохраняем монитор
	if err := s.monitorRepo.Create(ctx, monitor); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to save monitor")
	}

	// Записываем в audit log
	s.createAuditLog(ctx, &interfaces.AuditLogEntry{
		MonitorID: monitor.ID,
		UserID:    userID,
		Action:    interfaces.ActionMonitorCreate,
		NewValues: map[string]any{
			"name":             monitor.Name,
			"url":              monitor.URL,
			"interval_seconds": monitor.IntervalSeconds,
			"timeout_seconds":  monitor.TimeoutSeconds,
			"check_type":       monitor.CheckType,
			"working_days":     req.WorkingDays,
		},
	})

	tracing.AddEvent(ctx, "create_monitor.complete", map[string]any{
		"monitor_id": monitor.ID.String(),
		"status":     string(monitor.Status),
	})
	tracing.SetSuccess(span)

	return s.monitorToResponse(monitor), nil
}

// GetMonitor возвращает монитор по ID.
func (s *MonitorService) GetMonitor(ctx context.Context, id, userID string) (*dto.MonitorResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorService.GetMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "get_monitor.start", map[string]any{
		"monitor_id": id,
		"user_id":    userID,
	})

	monitorID, err := uuid.Parse(id)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid monitor_id")
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Проверяем, что пользователь имеет доступ к монитору
	if err := s.authz.CanAccessMonitor(ctx, uid, monitorID); err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		// Проверяем, что это не прямая ошибка авторизации
		if errors.Is(err, apperrors.ErrMonitorNotFound) ||
			errors.Is(err, apperrors.ErrPermissionDenied) ||
			errors.Is(err, apperrors.ErrUnknownTier) {
			return nil, err // Прямая ошибка авторизации, возвращаем её напрямую
		}
		// Обычные ошибки оборачиваем для сохранения контекста
		return nil, errors.Wrap(err, "failed to get monitor")
	}

	tracing.SetSuccess(span)
	return s.monitorToResponse(monitor), nil
}

// ListMonitors возвращает список мониторов пользователя.
func (s *MonitorService) ListMonitors(ctx context.Context, req *dto.ListMonitorsRequest) (*dto.ListMonitorsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorService.ListMonitors")
	defer span.End()

	tracing.AddEvent(ctx, "list_monitors.start", map[string]any{
		"user_id":          req.UserID,
		"status":           req.Status,
		"limit":            req.Limit,
		"offset":           req.Offset,
		"skip_user_filter": req.SkipUserFilter,
	})

	if req.Limit <= 0 {
		req.Limit = 100
	}

	// Сервисный вызов (например, scheduler-service): возвращаем все активные мониторы
	// без фильтра по user_id и без попытки распарсить UserID как UUID.
	if req.SkipUserFilter {
		monitors, err := s.monitorRepo.ListActive(ctx)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, errors.Wrap(err, "failed to list active monitors")
		}

		responses := make([]*dto.MonitorResponse, len(monitors))
		for i, m := range monitors {
			responses[i] = s.monitorToResponse(m)
		}

		tracing.AddEvent(ctx, "list_monitors.complete", map[string]any{
			"total":    len(responses),
			"returned": len(responses),
		})
		tracing.SetSuccess(span)

		return &dto.ListMonitorsResponse{
			Monitors: responses,
			Total:    len(responses),
			Limit:    req.Limit,
			Offset:   req.Offset,
		}, nil
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	var monitors []*domain.Monitor
	if req.Status != "" {
		status := domain.MonitorStatus(req.Status)
		if !status.IsValid() {
			err := errors.New("invalid status")
			tracing.RecordError(span, err)
			return nil, err
		}
		monitors, err = s.monitorRepo.ListByUserIDAndStatus(ctx, userID, status)
	} else {
		monitors, err = s.monitorRepo.ListByUserID(ctx, userID, req.Limit, req.Offset)
	}

	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to list monitors")
	}

	total, err := s.monitorRepo.CountByUserID(ctx, userID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to count monitors")
	}

	responses := make([]*dto.MonitorResponse, len(monitors))
	for i, m := range monitors {
		responses[i] = s.monitorToResponse(m)
	}

	tracing.AddEvent(ctx, "list_monitors.complete", map[string]any{
		"total":    total,
		"returned": len(responses),
	})
	tracing.SetSuccess(span)

	return &dto.ListMonitorsResponse{
		Monitors: responses,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}, nil
}

// UpdateMonitor обновляет монитор.
func (s *MonitorService) UpdateMonitor(ctx context.Context, req *dto.UpdateMonitorRequest) (*dto.MonitorResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorService.UpdateMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "update_monitor.start", map[string]any{
		"monitor_id": req.ID,
		"user_id":    req.UserID,
	})

	monitorID, err := uuid.Parse(req.ID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid monitor_id")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Проверяем, что пользователь имеет доступ к монитору
	if err := s.authz.CanAccessMonitor(ctx, userID, monitorID); err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Получаем монитор (CanAccessMonitor уже вызывает GetByID, но нам нужны данные для audit log)
	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		// Проверяем, что это не прямая ошибка авторизации
		if errors.Is(err, apperrors.ErrMonitorNotFound) ||
			errors.Is(err, apperrors.ErrPermissionDenied) ||
			errors.Is(err, apperrors.ErrUnknownTier) {
			return nil, err // Прямая ошибка авторизации, не оборачиваем
		}
		return nil, errors.Wrap(err, "failed to get monitor")
	}

	// Сохраняем старые значения для audit log
	oldValues := map[string]any{
		"name":             monitor.Name,
		"url":              monitor.URL,
		"interval_seconds": monitor.IntervalSeconds,
		"timeout_seconds":  monitor.TimeoutSeconds,
	}

	// Применяем обновления
	if req.Name != nil {
		monitor.Name = *req.Name
	}
	if req.URL != nil {
		monitor.URL = *req.URL
	}
	if req.IntervalSeconds != nil {
		monitor.IntervalSeconds = *req.IntervalSeconds
	}
	if req.TimeoutSeconds != nil {
		monitor.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.WorkingHoursStart != nil && req.WorkingHoursEnd != nil {
		workingDays := parseWorkingDays(req.WorkingDays)
		wh, err := domain.NewWorkingHours(*req.WorkingHoursStart, *req.WorkingHoursEnd, workingDays)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, errors.Wrap(err, "invalid working hours")
		}
		start, end := wh.GetStartEndTime()
		monitor.WorkingHoursStart = start
		monitor.WorkingHoursEnd = end
		monitor.WorkingDays = workingDays
	}
	if req.DegradedResponseTimeThreshold != nil {
		monitor.DegradedResponseTimeThreshold = req.DegradedResponseTimeThreshold
	}
	if req.DegradedFailureRateThreshold != nil {
		monitor.DegradedFailureRateThreshold = req.DegradedFailureRateThreshold
	}

	// Сохраняем обновления
	if err := s.monitorRepo.Update(ctx, monitor); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to update monitor")
	}

	// Записываем в audit log
	newValues := map[string]any{
		"name":             monitor.Name,
		"url":              monitor.URL,
		"interval_seconds": monitor.IntervalSeconds,
		"timeout_seconds":  monitor.TimeoutSeconds,
	}

	s.createAuditLog(ctx, &interfaces.AuditLogEntry{
		MonitorID: monitor.ID,
		UserID:    userID,
		Action:    interfaces.ActionMonitorUpdate,
		OldValues: oldValues,
		NewValues: newValues,
	})

	tracing.SetSuccess(span)
	return s.monitorToResponse(monitor), nil
}

// DeleteMonitor удаляет монитор.
func (s *MonitorService) DeleteMonitor(ctx context.Context, id, userID string) error {
	ctx, span := tracing.StartSpan(ctx, "MonitorService.DeleteMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "delete_monitor.start", map[string]any{
		"monitor_id": id,
		"user_id":    userID,
	})

	monitorID, err := uuid.Parse(id)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid monitor_id")
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid user_id")
	}

	// Проверяем, что пользователь имеет доступ к монитору
	if err := s.authz.CanAccessMonitor(ctx, uid, monitorID); err != nil {
		tracing.RecordError(span, err)
		return err
	}

	// Получаем монитор для audit log
	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		// Проверяем, что это не прямая ошибка авторизации
		if errors.Is(err, apperrors.ErrMonitorNotFound) ||
			errors.Is(err, apperrors.ErrPermissionDenied) ||
			errors.Is(err, apperrors.ErrUnknownTier) {
			return err // Прямая ошибка авторизации, не оборачиваем
		}
		return errors.Wrap(err, "failed to get monitor")
	}

	// Удаляем монитор
	if err := s.monitorRepo.Delete(ctx, monitorID); err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to delete monitor")
	}

	// Записываем в audit log
	s.createAuditLog(ctx, &interfaces.AuditLogEntry{
		MonitorID: monitor.ID,
		UserID:    uid,
		Action:    interfaces.ActionMonitorDelete,
		OldValues: map[string]any{
			"name": monitor.Name,
			"url":  monitor.URL,
		},
	})

	tracing.SetSuccess(span)
	return nil
}

// PauseMonitor приостанавливает монитор.
func (s *MonitorService) PauseMonitor(ctx context.Context, req *dto.PauseMonitorRequest) error {
	ctx, span := tracing.StartSpan(ctx, "MonitorService.PauseMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "pause_monitor.start", map[string]any{
		"monitor_id": req.ID,
		"user_id":    req.UserID,
	})

	monitorID, err := uuid.Parse(req.ID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid monitor_id")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid user_id")
	}

	// Проверяем, что пользователь имеет доступ к монитору
	if err := s.authz.CanAccessMonitor(ctx, userID, monitorID); err != nil {
		tracing.RecordError(span, err)
		return err
	}

	// Получаем монитор
	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		// Проверяем, что это не прямая ошибка авторизации
		if errors.Is(err, apperrors.ErrMonitorNotFound) ||
			errors.Is(err, apperrors.ErrPermissionDenied) ||
			errors.Is(err, apperrors.ErrUnknownTier) {
			return err // Прямая ошибка авторизации, не оборачиваем
		}
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to get monitor")
	}
	// Сохраняем старый статус для audit log
	oldStatus := monitor.Status

	// Обновляем статус
	if err := monitor.UpdateStatus(domain.StatusPaused); err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to update monitor status")
	}

	// Сохраняем обновления
	if err := s.monitorRepo.Update(ctx, monitor); err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to update monitor")
	}

	// Записываем в audit log
	s.createAuditLog(ctx, &interfaces.AuditLogEntry{
		MonitorID: monitor.ID,
		UserID:    userID,
		Action:    interfaces.ActionMonitorPause,
		OldValues: map[string]any{
			"status": string(oldStatus),
		},
		NewValues: map[string]any{
			"status": "PAUSED",
		},
	})

	tracing.SetSuccess(span)
	return nil
}

// ResumeMonitor возобновляет монитор.
func (s *MonitorService) ResumeMonitor(ctx context.Context, req *dto.ResumeMonitorRequest) error {
	ctx, span := tracing.StartSpan(ctx, "MonitorService.ResumeMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "resume_monitor.start", map[string]any{
		"monitor_id": req.ID,
		"user_id":    req.UserID,
	})

	monitorID, err := uuid.Parse(req.ID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid monitor_id")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "invalid user_id")
	}

	// Проверяем, что пользователь имеет доступ к монитору
	if err := s.authz.CanAccessMonitor(ctx, userID, monitorID); err != nil {
		tracing.RecordError(span, err)
		return err
	}

	// Получаем монитор
	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to get monitor")
	}
	// Сохраняем старый статус для audit log
	oldStatus := monitor.Status

	// Обновляем статус на UP (при восстановлении монитор активен)
	if err := monitor.UpdateStatus(domain.StatusUp); err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to update monitor status")
	}

	// Сохраняем обновления
	if err := s.monitorRepo.Update(ctx, monitor); err != nil {
		tracing.RecordError(span, err)
		return errors.Wrap(err, "failed to update monitor")
	}

	// Записываем в audit log
	s.createAuditLog(ctx, &interfaces.AuditLogEntry{
		MonitorID: monitor.ID,
		UserID:    userID,
		Action:    interfaces.ActionMonitorResume,
		OldValues: map[string]any{
			"status": string(oldStatus),
		},
		NewValues: map[string]any{
			"status": "UP",
		},
	})

	tracing.SetSuccess(span)
	return nil
}

// monitorToResponse конвертирует domain.Monitor в dto.MonitorResponse.
func (s *MonitorService) monitorToResponse(m *domain.Monitor) *dto.MonitorResponse {
	resp := &dto.MonitorResponse{
		ID:              m.ID.String(),
		UserID:          m.UserID.String(),
		Name:            m.Name,
		URL:             m.URL,
		CheckType:       m.CheckType,
		IntervalSeconds: m.IntervalSeconds,
		TimeoutSeconds:  m.TimeoutSeconds,
		Status:          string(m.Status),
		WorkingDays:     workingDaysToStrings(m.WorkingDays),
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	if m.WorkingHoursStart != nil {
		s := m.WorkingHoursStart.Format("15:04")
		resp.WorkingHoursStart = &s
	}
	if m.WorkingHoursEnd != nil {
		s := m.WorkingHoursEnd.Format("15:04")
		resp.WorkingHoursEnd = &s
	}
	if m.DegradedResponseTimeThreshold != nil {
		resp.DegradedResponseTimeThreshold = m.DegradedResponseTimeThreshold
	}
	if m.DegradedFailureRateThreshold != nil {
		resp.DegradedFailureRateThreshold = m.DegradedFailureRateThreshold
	}
	if m.LastCheckAt != nil {
		resp.LastCheckAt = m.LastCheckAt
	}

	return resp
}

// createAuditLog создаёт запись в audit log.
func (s *MonitorService) createAuditLog(ctx context.Context, entry *interfaces.AuditLogEntry) {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		// Log error but don't fail the operation
		s.logger.ErrorContext(ctx, "failed to create audit log",
			"error", err,
			"monitor_id", entry.MonitorID.String(),
			"user_id", entry.UserID.String(),
			"action", entry.Action,
		)
	}
}

// parseWorkingDays парсит рабочие дни из []string.
func parseWorkingDays(days []string) []time.Weekday {
	if len(days) == 0 {
		return nil
	}

	dayMap := map[string]time.Weekday{
		"Sunday":    time.Sunday,
		"Monday":    time.Monday,
		"Tuesday":   time.Tuesday,
		"Wednesday": time.Wednesday,
		"Thursday":  time.Thursday,
		"Friday":    time.Friday,
		"Saturday":  time.Saturday,
	}

	result := make([]time.Weekday, len(days))
	for i, d := range days {
		result[i] = dayMap[d]
	}
	return result
}

// workingDaysToStrings конвертирует []time.Weekday в []string.
func workingDaysToStrings(days []time.Weekday) []string {
	if len(days) == 0 {
		return nil
	}

	result := make([]string, len(days))
	for i, d := range days {
		result[i] = d.String()
	}
	return result
}
