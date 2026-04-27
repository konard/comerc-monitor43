package service

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/pkg/errors"
)

// Локальный частично применяемый интерфейс — только нужные методы.

type sessionLister interface {
	GetByID(ctx context.Context, sessionID string) (*model.Session, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Session, int, error)
	Delete(ctx context.Context, sessionID string) error
}

// SessionService обрабатывает операции с сессиями.
type SessionService struct {
	sessionRepo sessionLister
	tracer      trace.Tracer
}

// NewSessionService создаёт новый SessionService.
func NewSessionService(sessionRepo sessionLister, tracer trace.Tracer) *SessionService {
	return &SessionService{
		sessionRepo: sessionRepo,
		tracer:      tracer,
	}
}

// ListSessions возвращает все сессии пользователя.
func (s *SessionService) ListSessions(ctx context.Context, req *dto.ListSessionsRequest) (*dto.ListSessionsResponse, error) {
	ctx, span := s.tracer.Start(ctx, "SessionService.ListSessions")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserID.String()),
		attribute.Int("limit", req.Limit),
		attribute.Int("offset", req.Offset),
	)

	sessions, total, err := s.sessionRepo.GetByUserID(ctx, req.UserID, req.Limit, req.Offset)
	if err != nil {
		span.RecordError(err)
		return nil, errors.InternalError(err)
	}

	span.SetAttributes(attribute.Int("sessions.total", total))

	return &dto.ListSessionsResponse{
		Sessions: dto.ToSessionDTOs(sessions, req.CurrentSessionID),
		Total:    total,
	}, nil
}

// RevokeSession отзывает конкретную сессию, проверяя принадлежность владельцу.
func (s *SessionService) RevokeSession(ctx context.Context, req *dto.RevokeSessionRequest) error {
	ctx, span := s.tracer.Start(ctx, "SessionService.RevokeSession")
	defer span.End()

	span.SetAttributes(attribute.String("session.id", req.SessionID))

	// Получаем сессию для проверки существования и принадлежности
	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		span.AddEvent("session_not_found")
		return errors.SessionNotFound()
	}

	// Проверяем принадлежность сессии запрашивающему пользователю
	if req.UserID != session.UserID {
		span.AddEvent("session_ownership_mismatch", trace.WithAttributes(
			attribute.String("requesting_user_id", req.UserID.String()),
			attribute.String("session_owner_id", session.UserID.String()),
		))
		return errors.SessionNotFound()
	}

	if err := s.sessionRepo.Delete(ctx, req.SessionID); err != nil {
		span.RecordError(err)
		return errors.InternalError(err)
	}

	span.AddEvent("session_revoked")

	return nil
}
