package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	model "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// IncidentDetector детектирует и закрывает инциденты.
type IncidentDetector struct {
	monitorRepo  interfaces.MonitorRepository
	incidentRepo interfaces.IncidentRepository

	// Configuration
	resolveThreshold int // Количество последовательных UP для закрытия инцидента
}

// NewIncidentDetector создаёт новый IncidentDetector.
func NewIncidentDetector(
	monitorRepo interfaces.MonitorRepository,
	incidentRepo interfaces.IncidentRepository,
	resolveThreshold int,
) *IncidentDetector {
	return &IncidentDetector{
		monitorRepo:      monitorRepo,
		incidentRepo:     incidentRepo,
		resolveThreshold: resolveThreshold,
	}
}

// HandleStatusChange обрабатывает изменение статуса монитора для детекции инцидентов.
func (d *IncidentDetector) HandleStatusChange(
	ctx context.Context,
	monitorID uuid.UUID,
	oldStatus, newStatus model.MonitorStatus,
) error {
	ctx, span := tracing.StartSpan(ctx, "IncidentDetector.HandleStatusChange")
	defer span.End()

	tracing.AddEvent(ctx, "handle_status_change", map[string]any{
		"monitor_id": monitorID,
		"old_status": oldStatus,
		"new_status": newStatus,
	})

	defer func() {
		if span.IsRecording() {
			tracing.SetSuccess(span)
		}
	}()

	// При переходе в DOWN -> создаём инцидент
	if newStatus == model.StatusDown && oldStatus != model.StatusDown {
		err := d.createIncident(ctx, monitorID)
		if err != nil {
			tracing.RecordError(span, err)
			return err
		}
		return nil
	}

	// При переходе из DOWN -> UP/DEGRADED -> закрываем активные инциденты
	if oldStatus == model.StatusDown && newStatus != model.StatusDown {
		err := d.resolveIncidents(ctx, monitorID)
		if err != nil {
			tracing.RecordError(span, err)
			return err
		}
		return nil
	}

	return nil
}

// createIncident создаёт новый инцидент.
func (d *IncidentDetector) createIncident(ctx context.Context, monitorID uuid.UUID) error {
	// Проверяем, есть ли уже активный инцидент
	activeIncidents, err := d.incidentRepo.GetActiveByMonitorID(ctx, monitorID)
	if err != nil {
		return errors.Wrap(err, "failed to get active incidents")
	}

	if len(activeIncidents) > 0 {
		// Активный инцидент уже существует - не создаём новый
		return nil
	}

	// Создаём новый инцидент
	incident := model.NewIncident(monitorID)
	if err := d.incidentRepo.Create(ctx, incident); err != nil {
		return errors.Wrap(err, "failed to create incident")
	}

	return nil
}

// resolveIncidents закрывает активные инциденты монитора.
func (d *IncidentDetector) resolveIncidents(ctx context.Context, monitorID uuid.UUID) error {
	// Получаем активные инциденты
	activeIncidents, err := d.incidentRepo.GetActiveByMonitorID(ctx, monitorID)
	if err != nil {
		return errors.Wrap(err, "failed to get active incidents")
	}

	if len(activeIncidents) == 0 {
		return nil
	}

	// Закрываем все активные инциденты
	endTime := time.Now()
	for _, incident := range activeIncidents {
		if err := d.incidentRepo.Resolve(ctx, incident.ID, endTime); err != nil {
			return errors.Wrap(err, "failed to resolve incident")
		}
	}

	return nil
}

// GetIncidents возвращает инциденты монитора.
func (d *IncidentDetector) GetIncidents(ctx context.Context, req *dto.GetIncidentsRequest) (*dto.GetIncidentsResponse, error) {
	monitorID, err := uuid.Parse(req.MonitorID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid monitor_id")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Проверяем, что монитор принадлежит пользователю
	monitor, err := d.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get monitor")
	}
	if monitor.UserID != userID {
		return nil, errors.New("monitor not found")
	}

	// Получаем инциденты за период
	var incidents []*model.Incident
	if !req.From.IsZero() && !req.To.IsZero() {
		incidents, err = d.incidentRepo.GetByMonitorIDAndPeriod(ctx, monitorID, req.From, req.To)
	} else {
		incidents, err = d.incidentRepo.GetByMonitorID(ctx, monitorID, req.Limit, req.Offset)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to get incidents")
	}

	// Конвертируем в DTO
	incidentResponses := make([]*dto.IncidentResponse, len(incidents))
	for i, inc := range incidents {
		incidentResponses[i] = d.incidentToResponse(inc)
	}

	// Получаем общее количество
	total, err := d.incidentRepo.CountByMonitorID(ctx, monitorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count incidents")
	}

	return &dto.GetIncidentsResponse{
		Incidents: incidentResponses,
		Total:     int(total),
		Limit:     req.Limit,
		Offset:    req.Offset,
	}, nil
}

// incidentToResponse конвертирует model.Incident в dto.IncidentResponse.
func (d *IncidentDetector) incidentToResponse(inc *model.Incident) *dto.IncidentResponse {
	return &dto.IncidentResponse{
		ID:              inc.ID.String(),
		MonitorID:       inc.MonitorID.String(),
		StartTime:       inc.StartTime,
		EndTime:         inc.EndTime,
		DurationSeconds: inc.DurationSeconds,
		Status:          string(inc.Status),
		CreatedAt:       inc.CreatedAt,
	}
}
