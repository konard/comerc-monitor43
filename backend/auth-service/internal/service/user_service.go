package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/pkg/errors"
)

// Локальные частично применяемые интерфейсы — только нужные методы.

type userManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	LockAccount(ctx context.Context, userID uuid.UUID, lockedUntil any) error
	UnlockAccount(ctx context.Context, userID uuid.UUID) error
}

type auditWriter interface {
	Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error)
}

// UserService обрабатывает операции с пользователями.
type UserService struct {
	userRepo  userManager
	auditRepo auditWriter
	tracer    trace.Tracer
}

// NewUserService создаёт новый UserService.
func NewUserService(userRepo userManager, auditRepo auditWriter, tracer trace.Tracer) *UserService {
	return &UserService{
		userRepo:  userRepo,
		auditRepo: auditRepo,
		tracer:    tracer,
	}
}

// GetUser возвращает пользователя по ID.
func (s *UserService) GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.GetUserResponse, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.UserID.String()))

	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		span.AddEvent("user_not_found")
		return nil, errors.UserNotFound()
	}

	return dto.ToGetUserResponse(user), nil
}

// UpdateUser обновляет профиль пользователя (email и full_name).
func (s *UserService) UpdateUser(ctx context.Context, req *dto.UpdateUserRequest) (*dto.GetUserResponse, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.UpdateUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.UserID.String()))

	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		span.AddEvent("user_not_found")
		return nil, errors.UserNotFound()
	}

	// Обновляем только переданные поля, пустые значения игнорируем
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		span.RecordError(err)
		return nil, errors.InternalError(err)
	}

	span.AddEvent("user_updated")

	return dto.ToGetUserResponse(user), nil
}

// LockAccount блокирует учётную запись пользователя.
func (s *UserService) LockAccount(ctx context.Context, req *dto.LockAccountRequest) error {
	ctx, span := s.tracer.Start(ctx, "UserService.LockAccount")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserID.String()),
		attribute.Float64("duration_hours", float64(req.DurationHours)),
	)

	lockedUntil := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)

	if err := s.userRepo.LockAccount(ctx, req.UserID, lockedUntil); err != nil {
		span.RecordError(err)
		return errors.InternalError(err)
	}

	span.AddEvent("account_locked")

	// Log audit event
	log := model.NewAuditLog(&req.UserID, model.EventTypeAccountLocked, "", true, "", "", "")
	if _, err := s.auditRepo.Create(ctx, log); err != nil {
		span.RecordError(err)
		return errors.InternalError(err)
	}

	return nil
}

// UnlockAccount разблокирует учётную запись пользователя.
func (s *UserService) UnlockAccount(ctx context.Context, req *dto.UnlockAccountRequest) error {
	ctx, span := s.tracer.Start(ctx, "UserService.UnlockAccount")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.UserID.String()))

	if err := s.userRepo.UnlockAccount(ctx, req.UserID); err != nil {
		span.RecordError(err)
		return errors.InternalError(err)
	}

	span.AddEvent("account_unlocked")

	// Log audit event
	log := model.NewAuditLog(&req.UserID, model.EventTypeAccountUnlocked, "", true, "", "", "")
	if _, err := s.auditRepo.Create(ctx, log); err != nil {
		span.RecordError(err)
		return errors.InternalError(err)
	}

	return nil
}
