package mute

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	"github.com/raul/monitor/backend/alert-service/internal/service/audit"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

// MuteService управляет заглушениями алертов
type MuteService struct {
	muteRepo     repository.AlertMuteRepository
	auditService *audit.AuditService
	tracer       trace.Tracer
	metrics      *apptelemetry.Metrics
}

// NewMuteService создаёт новый сервис заглушений
func NewMuteService(
	muteRepo repository.AlertMuteRepository,
	auditService *audit.AuditService,
	tracer trace.Tracer,
	metrics *apptelemetry.Metrics,
) *MuteService {
	return &MuteService{
		muteRepo:     muteRepo,
		auditService: auditService,
		tracer:       tracer,
		metrics:      metrics,
	}
}

// MuteForUser создаёт пользовательское заглушение для монитора
func (s *MuteService) MuteForUser(ctx context.Context, userID, monitorID uuid.UUID, mutedUntil *time.Time) error {
	ctx, span := s.tracer.Start(ctx, "MuteService.MuteForUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("monitor_id", monitorID.String()),
	)

	mute := &model.AlertMute{
		ID:         uuid.New(),
		UserID:     userID,
		MonitorID:  &monitorID,
		Scope:      model.MuteScopeUser,
		MutedUntil: mutedUntil,
		CreatedAt:  time.Now(),
		CreatedBy:  userID,
	}

	if err := s.muteRepo.Create(ctx, mute); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create user mute: %v", err)
	}

	span.AddEvent("mute_created", trace.WithAttributes(
		attribute.String("mute_id", mute.ID.String()),
		attribute.String("scope", string(mute.Scope)),
	))

	if s.auditService != nil {
		if err := s.auditService.Log(ctx, model.AuditActionAlertMutedGlobal, "alert_mute", mute.ID.String(), userID, map[string]any{
			"scope":       string(mute.Scope),
			"monitor_id":  monitorID.String(),
			"muted_until": mutedUntil,
		}); err != nil {
			slog.Default().Warn("failed to record audit event", "error", err)
		}
	}

	return nil
}

// MuteGlobally создаёт глобальное заглушение для монитора (только для ADMIN)
func (s *MuteService) MuteGlobally(ctx context.Context, adminID, monitorID uuid.UUID, mutedUntil *time.Time) error {
	ctx, span := s.tracer.Start(ctx, "MuteService.MuteGlobally")
	defer span.End()

	if err := auth.RequireAdmin(ctx); err != nil {
		return model.ErrForbidden
	}

	span.SetAttributes(
		attribute.String("admin_id", adminID.String()),
		attribute.String("monitor_id", monitorID.String()),
	)

	mute := &model.AlertMute{
		ID:         uuid.New(),
		UserID:     adminID,
		MonitorID:  &monitorID,
		Scope:      model.MuteScopeGlobal,
		MutedUntil: mutedUntil,
		CreatedAt:  time.Now(),
		CreatedBy:  adminID,
	}

	if err := s.muteRepo.Create(ctx, mute); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create global mute: %v", err)
	}

	span.AddEvent("global_mute_created", trace.WithAttributes(
		attribute.String("mute_id", mute.ID.String()),
		attribute.String("monitor_id", monitorID.String()),
	))

	if s.auditService != nil {
		if err := s.auditService.Log(ctx, model.AuditActionAlertMutedGlobal, "alert_mute", mute.ID.String(), adminID, map[string]any{
			"scope":       string(mute.Scope),
			"monitor_id":  monitorID.String(),
			"muted_until": mutedUntil,
		}); err != nil {
			slog.Default().Warn("failed to record audit event", "error", err)
		}
	}

	return nil
}

// Unmute удаляет заглушение по ID
func (s *MuteService) Unmute(ctx context.Context, muteID string) error {
	ctx, span := s.tracer.Start(ctx, "MuteService.Unmute")
	defer span.End()

	span.SetAttributes(attribute.String("mute_id", muteID))

	if err := s.muteRepo.Delete(ctx, muteID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete mute: %v", err)
	}

	span.AddEvent("mute_deleted", trace.WithAttributes(
		attribute.String("mute_id", muteID),
	))

	return nil
}

// IsMuted проверяет наличие активного заглушения для пользователя и монитора
func (s *MuteService) IsMuted(ctx context.Context, userID, monitorID string) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "MuteService.IsMuted")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("monitor_id", monitorID),
	)

	muted, err := s.muteRepo.IsMuted(ctx, userID, monitorID)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to check mute status: %v", err)
	}

	span.AddEvent("mute_check_result", trace.WithAttributes(
		attribute.Bool("is_muted", muted),
	))

	return muted, nil
}

// UnmuteForUser удаляет все пользовательские заглушения для монитора
func (s *MuteService) UnmuteForUser(ctx context.Context, userID, monitorID uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "MuteService.UnmuteForUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("monitor_id", monitorID.String()),
	)

	if err := s.muteRepo.DeleteByUserIDAndMonitorID(ctx, userID.String(), monitorID.String()); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unmute for user: %v", err)
	}

	span.AddEvent("user_mute_deleted", trace.WithAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("monitor_id", monitorID.String()),
	))

	return nil
}

// ProcessExpiredMutes удаляет истёкшие заглушения (вызывается фоновым воркером)
func (s *MuteService) ProcessExpiredMutes(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "MuteService.ProcessExpiredMutes")
	defer span.End()

	deleted, err := s.muteRepo.DeleteExpired(ctx)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete expired mutes: %v", err)
	}

	if deleted > 0 {
		span.AddEvent("expired_mutes_deleted")
	}
	return nil
}

// GetActiveMutes возвращает все активные заглушения для пользователя
func (s *MuteService) GetActiveMutes(ctx context.Context, userID string) ([]*model.AlertMute, error) {
	ctx, span := s.tracer.Start(ctx, "MuteService.GetActiveMutes")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	mutes, err := s.muteRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get active mutes: %v", err)
	}

	span.AddEvent("active_mutes_found", trace.WithAttributes(
		attribute.Int("count", len(mutes)),
	))

	return mutes, nil
}
