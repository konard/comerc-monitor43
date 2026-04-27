package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	ajwt "github.com/raul/monitor/backend/auth-service/internal/infrastructure/jwt"
	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/pkg/errors"
)

// Локальные частично применяемые интерфейсы — только нужные методы.

type refreshTokenStore interface {
	Create(ctx context.Context, token *model.RefreshToken) (*model.RefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error
}

type sessionStore interface {
	GetByID(ctx context.Context, sessionID string) (*model.Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type userGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type tokenJWT interface {
	GenerateTokenPair(ctx context.Context, userID uuid.UUID, email, tier string) (string, string, error)
	ValidateAccessToken(ctx context.Context, tokenStr string) (*ajwt.Claims, error)
}

type auditLogger interface {
	Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error)
}

// TokenService обрабатывает операции с токенами.
type TokenService struct {
	jwtService       tokenJWT
	refreshTokenRepo refreshTokenStore
	sessionRepo      sessionStore
	userRepo         userGetter
	auditRepo        auditLogger
	tracer           trace.Tracer
}

// NewTokenService создаёт новый TokenService.
func NewTokenService(
	jwtService tokenJWT,
	refreshTokenRepo refreshTokenStore,
	sessionRepo sessionStore,
	userRepo userGetter,
	auditRepo auditLogger,
	tracer trace.Tracer,
) *TokenService {
	return &TokenService{
		jwtService:       jwtService,
		refreshTokenRepo: refreshTokenRepo,
		sessionRepo:      sessionRepo,
		userRepo:         userRepo,
		auditRepo:        auditRepo,
		tracer:           tracer,
	}
}

// RefreshToken обновляет пару токенов по refresh-токену.
func (s *TokenService) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "TokenService.RefreshToken")
	defer span.End()

	span.SetAttributes(
		attribute.String("auth.ip_address", ipAddress),
		attribute.String("auth.user_agent", userAgent),
	)

	// Find refresh token
	refreshTokenHash := model.HashToken(req.RefreshToken)
	token, err := s.refreshTokenRepo.GetByTokenHash(ctx, refreshTokenHash)
	if err != nil {
		span.AddEvent("invalid_refresh_token")
		s.logAuditEvent(ctx, nil, model.EventTypeTokenRefresh, "", false, ipAddress, userAgent, "invalid refresh token")
		return nil, errors.InvalidRefreshToken()
	}

	span.SetAttributes(attribute.String("user.id", token.UserID.String()))

	// Check if token is valid
	if !token.IsValid() {
		span.AddEvent("refresh_token_expired_or_revoked")
		s.logAuditEvent(ctx, &token.UserID, model.EventTypeTokenRefresh, "", false, ipAddress, userAgent, "refresh token expired or revoked")
		return nil, errors.InvalidRefreshToken()
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &token.UserID, model.EventTypeTokenRefresh, "", false, ipAddress, userAgent, "user not found")
		return nil, errors.InvalidRefreshToken()
	}

	// Generate new tokens
	accessToken, newRefreshToken, err := s.jwtService.GenerateTokenPair(ctx, user.ID, user.Email, user.Tier)
	if err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &token.UserID, model.EventTypeTokenRefresh, "", false, ipAddress, userAgent, err.Error())
		return nil, errors.InternalError(err)
	}

	// Revoke old refresh token
	if err := s.refreshTokenRepo.Revoke(ctx, token.ID); err != nil {
		span.RecordError(err)
		return nil, errors.InternalError(err)
	}

	// Store new refresh token
	newRefreshTokenHash := model.HashToken(newRefreshToken)
	newToken := model.NewRefreshToken(token.UserID, newRefreshTokenHash, "refresh", ipAddress, 7*24*time.Hour)
	if _, err := s.refreshTokenRepo.Create(ctx, newToken); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &token.UserID, model.EventTypeTokenRefresh, "", true, ipAddress, userAgent, "failed to store new refresh token")
		return nil, errors.InternalError(err)
	}

	// Log audit event
	s.logAuditEvent(ctx, &token.UserID, model.EventTypeTokenRefresh, "", true, ipAddress, userAgent, "")

	span.AddEvent("token_refreshed")

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User:         dto.ToUserDTO(user),
	}, nil
}

// ValidateToken проверяет access-токен и возвращает данные пользователя.
func (s *TokenService) ValidateToken(ctx context.Context, accessToken string) (*dto.ValidateTokenResponse, error) {
	ctx, span := s.tracer.Start(ctx, "TokenService.ValidateToken")
	defer span.End()

	// Validate token
	claims, err := s.jwtService.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		span.AddEvent("token_invalid")
		return &dto.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	userID := claims.UserID
	tier := claims.Tier

	span.SetAttributes(attribute.String("user.id", userID.String()))

	// Get user to verify it still exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		span.AddEvent("user_not_found")
		return &dto.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	// Log audit event
	log := model.NewAuditLog(&user.ID, model.EventTypeTokenValidate, "", true, "", "", "")
	if _, err := s.auditRepo.Create(ctx, log); err != nil {
		span.RecordError(err)
	}

	span.AddEvent("token_valid")

	var expiresAt *time.Time
	if claims.ExpiresAt != nil {
		t := claims.ExpiresAt.Time
		expiresAt = &t
	}

	return &dto.ValidateTokenResponse{
		Valid:     true,
		UserID:    userID.String(),
		Email:     user.Email,
		Tier:      tier,
		ExpiresAt: expiresAt,
	}, nil
}

// Logout отзывает refresh-токен и удаляет сессию.
func (s *TokenService) Logout(ctx context.Context, userID uuid.UUID, refreshToken, sessionID, ipAddress, userAgent string) error {
	ctx, span := s.tracer.Start(ctx, "TokenService.Logout")
	defer span.End()

	span.SetAttributes(
		attribute.String("auth.ip_address", ipAddress),
		attribute.String("auth.user_agent", userAgent),
	)
	if userID != uuid.Nil {
		span.SetAttributes(attribute.String("user.id", userID.String()))
	}

	// Find and revoke refresh token if provided
	var tokenUserID *uuid.UUID
	if refreshToken != "" {
		refreshTokenHash := model.HashToken(refreshToken)
		token, err := s.refreshTokenRepo.GetByTokenHash(ctx, refreshTokenHash)
		if err == nil {
			if revokeErr := s.refreshTokenRepo.Revoke(ctx, token.ID); revokeErr != nil {
				span.RecordError(revokeErr)
			} else {
				tokenUserID = &token.UserID
				span.AddEvent("refresh_token_revoked")
			}
		}
	}

	// Delete session if provided
	if sessionID != "" {
		if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
			span.RecordError(err)
		} else {
			span.AddEvent("session_deleted", trace.WithAttributes(attribute.String("session_id", sessionID)))
		}
	}

	// Log audit event - use token's user ID if userID is nil
	auditUserID := tokenUserID
	if auditUserID == nil && userID != uuid.Nil {
		auditUserID = &userID
	}
	log := model.NewAuditLog(auditUserID, model.EventTypeLogout, "", true, ipAddress, userAgent, "")
	if _, err := s.auditRepo.Create(ctx, log); err != nil {
		span.RecordError(err)
	}

	span.AddEvent("logout_complete")

	return nil
}

// logAuditEvent logs an audit event
func (s *TokenService) logAuditEvent(ctx context.Context, userID *uuid.UUID, eventType, provider string, success bool, ipAddress, userAgent, errorMessage string) {
	log := model.NewAuditLog(userID, eventType, provider, success, ipAddress, userAgent, errorMessage)
	if _, err := s.auditRepo.Create(ctx, log); err != nil {
		trace.SpanFromContext(ctx).RecordError(err)
	}
}
