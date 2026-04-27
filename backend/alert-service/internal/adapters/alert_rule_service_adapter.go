package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
)

// AlertRuleServiceAdapter адаптирует репозиторий к интерфейсу RuleService
type AlertRuleServiceAdapter struct {
	ruleRepo repository.AlertRuleRepository
}

// NewAlertRuleServiceAdapter создаёт новый адаптер
func NewAlertRuleServiceAdapter(ruleRepo repository.AlertRuleRepository) *AlertRuleServiceAdapter {
	return &AlertRuleServiceAdapter{
		ruleRepo: ruleRepo,
	}
}

// GetRuleByMonitorID возвращает правило алерта для монитора
func (a *AlertRuleServiceAdapter) GetRuleByMonitorID(ctx context.Context, userID, monitorID uuid.UUID) (*model.AlertRule, error) {
	rule, err := a.ruleRepo.GetByUserIDAndMonitorID(ctx, userID.String(), monitorID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrAlertRuleNotFound
		}
		return nil, fmt.Errorf("failed to get alert rule by monitor: %v", err)
	}
	return rule, nil
}

// GetRuleByID возвращает правило алерта по ID
func (a *AlertRuleServiceAdapter) GetRuleByID(ctx context.Context, id uuid.UUID) (*model.AlertRule, error) {
	rule, err := a.ruleRepo.GetByID(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrAlertRuleNotFound
		}
		return nil, fmt.Errorf("failed to get alert rule by id: %v", err)
	}
	return rule, nil
}

// CreateRule создаёт новое правило алерта
func (a *AlertRuleServiceAdapter) CreateRule(ctx context.Context, rule *model.AlertRule) (*model.AlertRule, error) {
	if err := a.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create alert rule: %v", err)
	}
	return rule, nil
}

// DeleteRule удаляет правило алерта для монитора, принадлежащего пользователю
func (a *AlertRuleServiceAdapter) DeleteRule(ctx context.Context, userID, monitorID uuid.UUID) error {
	rule, err := a.ruleRepo.GetByUserIDAndMonitorID(ctx, userID.String(), monitorID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrAlertRuleNotFound) {
			return model.ErrAlertRuleNotFound
		}
		return fmt.Errorf("failed to get alert rule for deletion: %v", err)
	}

	if err := a.ruleRepo.Delete(ctx, rule.ID.String()); err != nil {
		return fmt.Errorf("failed to delete alert rule: %v", err)
	}
	return nil
}
