package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/repository/interfaces"
	"github.com/raul/monitor/gateway/internal/service/dto"
)

// tracer используется для OpenTelemetry трассировки операций сервиса аутентификации.
var tracer = otel.Tracer("gateway/service")

// AuthService handles authentication-related business logic
type AuthService struct {
	authRepo interfaces.AuthRepository
}

// NewAuthService creates a new auth service
func NewAuthService(authRepo interfaces.AuthRepository) *AuthService {
	return &AuthService{
		authRepo: authRepo,
	}
}

// ValidateToken validates a token and returns the user context
func (s *AuthService) ValidateToken(ctx context.Context, req *dto.ValidateTokenRequest) (*dto.ValidateTokenResponse, error) {
	ctx, span := tracer.Start(ctx, "AuthService.ValidateToken")
	defer span.End()

	result, err := s.authRepo.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return &dto.ValidateTokenResponse{
			Valid:   false,
			Message: "validation failed",
		}, err
	}

	if !result.Valid {
		return &dto.ValidateTokenResponse{
			Valid:   false,
			Message: "invalid token",
		}, nil
	}

	return dto.NewValidateTokenResponse(true, result.UserID, result.Email, result.Tier), nil
}

// GetUserContext extracts user context from a token
func (s *AuthService) GetUserContext(ctx context.Context, token string) (*model.UserContext, error) {
	ctx, span := tracer.Start(ctx, "AuthService.GetUserContext")
	defer span.End()

	result, err := s.authRepo.ValidateToken(ctx, token)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, err
	}

	if !result.Valid {
		return nil, model.ErrInvalidToken
	}

	return model.NewUserContext(result.UserID, result.Email, result.Tier), nil
}

// CheckAuthHealth checks the health of the auth service
func (s *AuthService) CheckAuthHealth(ctx context.Context) (bool, time.Duration, error) {
	start := time.Now()
	healthy, err := s.authRepo.CheckHealth(ctx)
	latency := time.Since(start)
	return healthy, latency, err
}
