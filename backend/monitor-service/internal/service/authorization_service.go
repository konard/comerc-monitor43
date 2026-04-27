package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	apperrors "github.com/raul/monitor/backend/monitor-service/pkg/errors"
)

// AuthorizationService предоставляет сервис авторизации и RBAC.
//
// Проверяет права доступа и лимиты по subscription tier:
//   - CanCreateMonitor: проверяет лимит мониторов для tier
//   - CanAccessMonitor: проверяет владение монитором
type AuthorizationService struct {
	monitorRepo interfaces.MonitorRepository
	tierLimits  map[string]int
}

// NewAuthorizationService создаёт новый AuthorizationService.
func NewAuthorizationService(monitorRepo interfaces.MonitorRepository) *AuthorizationService {
	return &AuthorizationService{
		monitorRepo: monitorRepo,
		tierLimits: map[string]int{
			"Free":       5,
			"Basic":      25,
			"Pro":        100,
			"Enterprise": -1, // -1 означает unlimited
		},
	}
}

// CanCreateMonitor проверяет лимит мониторов для tier.
//
// Возвращает ошибку если лимит превышен или tier неизвестен.
func (s *AuthorizationService) CanCreateMonitor(ctx context.Context, userID uuid.UUID, tier string) error {
	ctx, span := tracing.StartSpan(ctx, "AuthorizationService.CanCreateMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "can_create_monitor", map[string]any{
		"user_id": userID,
		"tier":    tier,
	})

	// Если tier не задан, применяем ограничения Free tier как самого ограничительного
	if tier == "" {
		tier = "Free"
	}

	limit, exists := s.tierLimits[tier]
	if !exists {
		tracing.RecordError(span, apperrors.New("unknown subscription tier"))
		return apperrors.ErrUnknownTier
	}

	// -1 означает unlimited
	if limit == -1 {
		tracing.SetSuccess(span)
		return nil
	}

	// Получаем текущее количество мониторов
	count, err := s.monitorRepo.CountByUserID(ctx, userID)
	if err != nil {
		tracing.RecordError(span, err)
		return apperrors.Wrap(err, "failed to count monitors")
	}

	// Проверяем лимит
	if count >= limit {
		tracing.RecordError(span, fmt.Errorf("monitor limit reached for %s tier: %d/%d", tier, count, limit))
		return fmt.Errorf("monitor limit reached for %s tier: %d/%d", tier, count, limit)
	}

	tracing.SetSuccess(span)
	return nil
}

// CanAccessMonitor проверяет владение монитором.
//
// Возвращает ошибку если монитор не принадлежит пользователю.
func (s *AuthorizationService) CanAccessMonitor(ctx context.Context, userID, monitorID uuid.UUID) error {
	ctx, span := tracing.StartSpan(ctx, "AuthorizationService.CanAccessMonitor")
	defer span.End()

	tracing.AddEvent(ctx, "can_access_monitor", map[string]any{
		"user_id":    userID,
		"monitor_id": monitorID,
	})

	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		if errors.Is(err, apperrors.ErrMonitorNotFound) {
			tracing.RecordError(span, apperrors.ErrMonitorNotFound)
			return apperrors.ErrMonitorNotFound
		}
		tracing.RecordError(span, err)
		return apperrors.Wrap(err, "failed to get monitor")
	}

	if monitor.UserID != userID {
		tracing.RecordError(span, apperrors.ErrPermissionDenied)
		return apperrors.ErrPermissionDenied
	}

	tracing.SetSuccess(span)
	return nil
}
