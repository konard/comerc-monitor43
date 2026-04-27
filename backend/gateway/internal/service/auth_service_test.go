package service

import (
	"context"
	"testing"
	"time"

	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/service/dto"
)

// mockAuthRepository is a mock implementation of AuthRepository
type mockAuthRepository struct {
	validateTokenFunc func(ctx context.Context, token string) (*model.TokenValidationResult, error)
	checkHealthFunc   func(ctx context.Context) (bool, error)
}

func (m *mockAuthRepository) ValidateToken(ctx context.Context, token string) (*model.TokenValidationResult, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, token)
	}
	return model.NewTokenValidationResult(true, "user123", "test@example.com", "premium"), nil
}

func (m *mockAuthRepository) CheckHealth(ctx context.Context) (bool, error) {
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(ctx)
	}
	return true, nil
}

func TestNewAuthService(t *testing.T) {
	repo := &mockAuthRepository{}
	service := NewAuthService(repo)

	if service.authRepo == nil {
		t.Error("expected authRepo to be set")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		validateFunc   func(ctx context.Context, token string) (*model.TokenValidationResult, error)
		expectedValid  bool
		expectedUserID string
		expectedError  bool
	}{
		{
			name:  "valid token",
			token: "valid-token",
			validateFunc: func(ctx context.Context, token string) (*model.TokenValidationResult, error) {
				return model.NewTokenValidationResult(true, "user123", "test@example.com", "premium"), nil
			},
			expectedValid:  true,
			expectedUserID: "user123",
			expectedError:  false,
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			validateFunc: func(ctx context.Context, token string) (*model.TokenValidationResult, error) {
				return model.NewTokenValidationResult(false, "", "", ""), nil
			},
			expectedValid:  false,
			expectedUserID: "",
			expectedError:  false,
		},
		{
			name:  "repository error",
			token: "error-token",
			validateFunc: func(ctx context.Context, token string) (*model.TokenValidationResult, error) {
				return nil, model.ErrInvalidToken
			},
			expectedValid:  false,
			expectedUserID: "",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepository{
				validateTokenFunc: tt.validateFunc,
			}
			service := NewAuthService(repo)

			req := &dto.ValidateTokenRequest{AccessToken: tt.token}
			resp, err := service.ValidateToken(context.Background(), req)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if resp.Valid != tt.expectedValid {
				t.Errorf("expected valid %v, got %v", tt.expectedValid, resp.Valid)
			}

			if resp.UserID != tt.expectedUserID {
				t.Errorf("expected user ID %s, got %s", tt.expectedUserID, resp.UserID)
			}
		})
	}
}

func TestAuthService_GetUserContext(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		validateFunc   func(ctx context.Context, token string) (*model.TokenValidationResult, error)
		expectedError  bool
		expectedUserID string
	}{
		{
			name:  "valid token",
			token: "valid-token",
			validateFunc: func(ctx context.Context, token string) (*model.TokenValidationResult, error) {
				return model.NewTokenValidationResult(true, "user123", "test@example.com", "premium"), nil
			},
			expectedError:  false,
			expectedUserID: "user123",
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			validateFunc: func(ctx context.Context, token string) (*model.TokenValidationResult, error) {
				return model.NewTokenValidationResult(false, "", "", ""), nil
			},
			expectedError:  true,
			expectedUserID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepository{
				validateTokenFunc: tt.validateFunc,
			}
			service := NewAuthService(repo)

			userCtx, err := service.GetUserContext(context.Background(), tt.token)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if userCtx == nil {
				t.Fatal("expected user context, got nil")
			}

			if userCtx.UserID != tt.expectedUserID {
				t.Errorf("expected user ID %s, got %s", tt.expectedUserID, userCtx.UserID)
			}
		})
	}
}

func TestAuthService_CheckAuthHealth(t *testing.T) {
	tests := []struct {
		name           string
		healthy        bool
		latency        time.Duration
		expectedHealth bool
	}{
		{
			name:           "healthy service",
			healthy:        true,
			latency:        10 * time.Millisecond,
			expectedHealth: true,
		},
		{
			name:           "unhealthy service",
			healthy:        false,
			latency:        5 * time.Millisecond,
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepository{
				checkHealthFunc: func(ctx context.Context) (bool, error) {
					time.Sleep(tt.latency)
					return tt.healthy, nil
				},
			}
			service := NewAuthService(repo)

			healthy, latency, err := service.CheckAuthHealth(context.Background())

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if healthy != tt.expectedHealth {
				t.Errorf("expected health %v, got %v", tt.expectedHealth, healthy)
			}

			if latency < tt.latency {
				t.Errorf("expected latency at least %v, got %v", tt.latency, latency)
			}
		})
	}
}
