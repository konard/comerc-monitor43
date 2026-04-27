package handler

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/service/dto"
)

type mockAuthMiddlewareService struct {
	getUserContextFunc func(ctx context.Context, token string) (*model.UserContext, error)
}

func (m *mockAuthMiddlewareService) ValidateToken(ctx context.Context, req *dto.ValidateTokenRequest) (*dto.ValidateTokenResponse, error) {
	return nil, nil
}

func (m *mockAuthMiddlewareService) GetUserContext(ctx context.Context, token string) (*model.UserContext, error) {
	if m.getUserContextFunc != nil {
		return m.getUserContextFunc(ctx, token)
	}
	return model.NewUserContext("user123", "test@example.com", "premium"), nil
}

func (m *mockAuthMiddlewareService) CheckAuthHealth(ctx context.Context) (bool, time.Duration, error) {
	return true, 10 * time.Millisecond, nil
}

func TestNewAuthMiddleware(t *testing.T) {
	authService := &mockAuthMiddlewareService{}
	middleware := NewAuthMiddleware(authService)

	if middleware.authService == nil {
		t.Error("expected authService to be set")
	}
}

func TestAuthMiddleware_Middleware(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		authHeader         string
		expectedStatusCode int
		expectedUserID     string
	}{
		{
			name:               "no auth header on protected path",
			path:               "/api/v1/gateway/validate",
			authHeader:         "",
			expectedStatusCode: http.StatusUnauthorized,
			expectedUserID:     "",
		},
		{
			name:               "valid token",
			path:               "/api/v1/gateway/validate",
			authHeader:         "Bearer valid-token",
			expectedStatusCode: http.StatusOK,
			expectedUserID:     "user123",
		},
		{
			name:               "invalid token format",
			path:               "/api/v1/gateway/validate",
			authHeader:         "InvalidFormat token",
			expectedStatusCode: http.StatusUnauthorized,
			expectedUserID:     "",
		},
		{
			name:               "empty bearer",
			path:               "/api/v1/gateway/validate",
			authHeader:         "Bearer ",
			expectedStatusCode: http.StatusUnauthorized,
			expectedUserID:     "",
		},
		{
			name:               "no auth header on public path",
			path:               "/health",
			authHeader:         "",
			expectedStatusCode: http.StatusOK,
			expectedUserID:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authService := &mockAuthMiddlewareService{}
			middleware := NewAuthMiddleware(authService)

			var capturedUserID string
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUserID = GetUserIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			middleware.Middleware(nextHandler).ServeHTTP(w, req)

			if w.Code != tt.expectedStatusCode {
				t.Errorf("expected status %d, got %d", tt.expectedStatusCode, w.Code)
			}

			if capturedUserID != tt.expectedUserID {
				t.Errorf("expected user ID %q, got %q", tt.expectedUserID, capturedUserID)
			}
		})
	}
}

func TestAuthMiddleware_Middleware_InvalidToken(t *testing.T) {
	authService := &mockAuthMiddlewareService{
		getUserContextFunc: func(ctx context.Context, token string) (*model.UserContext, error) {
			return nil, model.ErrInvalidToken
		},
	}
	middleware := NewAuthMiddleware(authService)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	middleware.Middleware(nextHandler).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_Middleware_UnhealthyUserContext(t *testing.T) {
	authService := &mockAuthMiddlewareService{
		getUserContextFunc: func(ctx context.Context, token string) (*model.UserContext, error) {
			return &model.UserContext{Valid: false}, nil
		},
	}
	middleware := NewAuthMiddleware(authService)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	middleware.Middleware(nextHandler).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestNewLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	middleware := NewLoggingMiddleware(logger)

	if middleware.logger == nil {
		t.Error("expected logger to be set")
	}
}

func TestLoggingMiddleware_Middleware(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	middleware := NewLoggingMiddleware(logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.Middleware(nextHandler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	logOutput := buf.String()
	if !contains(logOutput, "GET") || !contains(logOutput, "/test") {
		t.Errorf("expected log to contain request info, got %s", logOutput)
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"valid bearer", "Bearer token123", "token123"},
		{"bearer with space", "Bearer token with spaces", "token with spaces"},
		{"no bearer prefix", "token123", ""},
		{"empty string", "", ""},
		{"only prefix", "Bearer ", ""},
		{"lowercase bearer", "bearer token123", ""},
		{"bearer in middle", "Prefix Bearer token123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBearerToken(tt.header)
			if got != tt.expected {
				t.Errorf("extractBearerToken(%q) = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("with user context", func(t *testing.T) {
		userCtx := &model.UserContext{UserID: "user123", Valid: true}
		ctx := context.WithValue(context.Background(), userContextKey, userCtx)

		userID := GetUserIDFromContext(ctx)
		if userID != "user123" {
			t.Errorf("expected user ID 'user123', got '%s'", userID)
		}
	})

	t.Run("without user context", func(t *testing.T) {
		userID := GetUserIDFromContext(context.Background())
		if userID != "" {
			t.Errorf("expected empty user ID, got '%s'", userID)
		}
	})

	t.Run("with invalid type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userContextKey, "not a user context")
		userID := GetUserIDFromContext(ctx)
		if userID != "" {
			t.Errorf("expected empty user ID for invalid type, got '%s'", userID)
		}
	})
}

func TestGetUserContext(t *testing.T) {
	t.Run("with user context", func(t *testing.T) {
		userCtx := &model.UserContext{UserID: "user123", Email: "test@example.com", Valid: true}
		ctx := context.WithValue(context.Background(), userContextKey, userCtx)

		got := GetUserContext(ctx)
		if got == nil {
			t.Fatal("expected user context, got nil")
		}

		if got.UserID != "user123" {
			t.Errorf("expected user ID 'user123', got '%s'", got.UserID)
		}

		if got.Email != "test@example.com" {
			t.Errorf("expected email 'test@example.com', got '%s'", got.Email)
		}
	})

	t.Run("without user context", func(t *testing.T) {
		got := GetUserContext(context.Background())
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
