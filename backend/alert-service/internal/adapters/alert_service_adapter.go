package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	"github.com/raul/monitor/backend/alert-service/internal/service/audit"
	applogger "github.com/raul/monitor/backend/alert-service/pkg/logger"
)

// AlertServiceAdapter адаптирует репозитории к интерфейсу AlertService
type AlertServiceAdapter struct {
	alertRepo    repository.AlertRepository
	ruleRepo     repository.AlertRuleRepository
	channelRepo  repository.AlertChannelRepository
	auditService *audit.AuditService
	logger       *applogger.Logger
}

// NewAlertServiceAdapter создаёт новый адаптер
func NewAlertServiceAdapter(
	alertRepo repository.AlertRepository,
	ruleRepo repository.AlertRuleRepository,
	channelRepo repository.AlertChannelRepository,
	logger *applogger.Logger,
) *AlertServiceAdapter {
	if logger == nil {
		logger = applogger.New("info")
	}
	return &AlertServiceAdapter{
		alertRepo:   alertRepo,
		ruleRepo:    ruleRepo,
		channelRepo: channelRepo,
		logger:      logger,
	}
}

// WithAuditService добавляет сервис аудита
func (a *AlertServiceAdapter) WithAuditService(auditSvc *audit.AuditService) *AlertServiceAdapter {
	a.auditService = auditSvc
	return a
}

// CreateAlert создаёт новый алерт
func (a *AlertServiceAdapter) CreateAlert(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	if err := a.alertRepo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %v", err)
	}
	if a.auditService != nil {
		if err := a.auditService.Log(ctx, model.AuditActionAlertCreated, "alert", alert.ID.String(), alert.UserID, map[string]any{
			"monitor_id": alert.MonitorID.String(),
			"alert_type": string(alert.Type),
		}); err != nil {
			a.logger.Warn("failed to record audit event", "error", err)
		}
	}
	return alert, nil
}

// GetAlert возвращает алерт по ID
func (a *AlertServiceAdapter) GetAlert(ctx context.Context, alertID uuid.UUID) (*model.Alert, error) {
	alert, err := a.alertRepo.GetByID(ctx, alertID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrAlertNotFound) {
			return nil, model.ErrAlertNotFound
		}
		return nil, fmt.Errorf("failed to get alert: %v", err)
	}
	return alert, nil
}

// ListAlerts возвращает список алертов с фильтрацией и пагинацией
func (a *AlertServiceAdapter) ListAlerts(ctx context.Context, userID uuid.UUID, filter model.AlertFilter) ([]*model.Alert, int, error) {
	alerts, total, err := a.alertRepo.List(ctx, userID.String(), filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list alerts: %v", err)
	}
	return alerts, total, nil
}

// UpdateAlert обновляет алерт
func (a *AlertServiceAdapter) UpdateAlert(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	_, err := a.alertRepo.GetByID(ctx, alert.ID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrAlertNotFound) {
			return nil, model.ErrAlertNotFound
		}
		return nil, fmt.Errorf("failed to get alert for update: %v", err)
	}

	if err := a.alertRepo.Update(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to update alert: %v", err)
	}
	if a.auditService != nil {
		action := model.AuditActionAlertUpdated
		if alert.Status == model.AlertStatusResolved {
			action = model.AuditActionAlertResolved
		}
		if err := a.auditService.Log(ctx, action, "alert", alert.ID.String(), alert.UserID, map[string]any{
			"status":  string(alert.Status),
			"enabled": alert.Enabled,
		}); err != nil {
			a.logger.Warn("failed to record audit event", "error", err)
		}
	}
	return alert, nil
}

// DeleteAlert удаляет алерт
func (a *AlertServiceAdapter) DeleteAlert(ctx context.Context, alertID uuid.UUID) error {
	err := a.alertRepo.Delete(ctx, alertID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrAlertNotFound) {
			return model.ErrAlertNotFound
		}
		return fmt.Errorf("failed to delete alert: %v", err)
	}
	if a.auditService != nil {
		if err := a.auditService.Log(ctx, model.AuditActionAlertDeleted, "alert", alertID.String(), uuid.Nil, map[string]any{
			"alert_id": alertID.String(),
		}); err != nil {
			a.logger.Warn("failed to record audit event", "error", err)
		}
	}
	return nil
}

// EnableAlert включает алерт
func (a *AlertServiceAdapter) EnableAlert(ctx context.Context, alertID uuid.UUID) error {
	alert, err := a.alertRepo.GetByID(ctx, alertID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrAlertNotFound) {
			return model.ErrAlertNotFound
		}
		return fmt.Errorf("failed to get alert: %v", err)
	}

	alert.Enabled = true
	if err := a.alertRepo.Update(ctx, alert); err != nil {
		return fmt.Errorf("failed to enable alert: %v", err)
	}
	if a.auditService != nil {
		if err := a.auditService.Log(ctx, model.AuditActionAlertEnabled, "alert", alertID.String(), alert.UserID, map[string]any{
			"alert_id": alertID.String(),
		}); err != nil {
			a.logger.Warn("failed to record audit event", "error", err)
		}
	}
	return nil
}

// DisableAlert отключает алерт
func (a *AlertServiceAdapter) DisableAlert(ctx context.Context, alertID uuid.UUID) error {
	alert, err := a.alertRepo.GetByID(ctx, alertID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrAlertNotFound) {
			return model.ErrAlertNotFound
		}
		return fmt.Errorf("failed to get alert: %v", err)
	}

	alert.Enabled = false
	if err := a.alertRepo.Update(ctx, alert); err != nil {
		return fmt.Errorf("failed to disable alert: %v", err)
	}
	if a.auditService != nil {
		if err := a.auditService.Log(ctx, model.AuditActionAlertDisabled, "alert", alertID.String(), alert.UserID, map[string]any{
			"alert_id": alertID.String(),
		}); err != nil {
			a.logger.Warn("failed to record audit event", "error", err)
		}
	}
	return nil
}

// AcknowledgeAlert подтверждает алерт пользователем с проверкой владельца и статуса
func (a *AlertServiceAdapter) AcknowledgeAlert(ctx context.Context, alertID, userID uuid.UUID) error {
	alert, err := a.alertRepo.GetByID(ctx, alertID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrAlertNotFound) {
			return model.ErrAlertNotFound
		}
		return fmt.Errorf("failed to get alert: %v", err)
	}

	if alert.UserID != userID {
		return model.ErrForbidden
	}

	if alert.Status == model.AlertStatusAcknowledged {
		return model.ErrAlertAlreadyAcknowledged
	}

	if alert.Status != model.AlertStatusTriggered {
		return model.ErrAlertNotAcknowledgeable
	}

	if err := a.alertRepo.AcknowledgeAlert(ctx, alertID.String(), userID.String()); err != nil {
		return fmt.Errorf("failed to acknowledge alert: %v", err)
	}

	a.logger.Info("alert acknowledged",
		"alert_id", alertID.String(),
		"user_id", userID.String(),
	)

	if a.auditService != nil {
		if err := a.auditService.Log(ctx, model.AuditActionAlertAcknowledged, "alert", alertID.String(), userID, map[string]any{
			"alert_id": alertID.String(),
		}); err != nil {
			a.logger.Warn("failed to record audit event", "error", err)
		}
	}

	return nil
}
