package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/service/dto"
	"github.com/raul/monitor/gateway/pkg/errors"
)

// mockAuthService is a mock implementation of AuthService
type mockAuthService struct {
	validateTokenFunc func(ctx context.Context, req *dto.ValidateTokenRequest) (*dto.ValidateTokenResponse, error)
	validateError     error
}

func (m *mockAuthService) ValidateToken(ctx context.Context, req *dto.ValidateTokenRequest) (*dto.ValidateTokenResponse, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, req)
	}
	if m.validateError != nil {
		return nil, m.validateError
	}
	return dto.NewValidateTokenResponse(true, "user123", "test@example.com", "premium"), nil
}

func (m *mockAuthService) GetUserContext(ctx context.Context, token string) (*model.UserContext, error) {
	return model.NewUserContext("user123", "test@example.com", "premium"), nil
}

func (m *mockAuthService) CheckAuthHealth(ctx context.Context) (bool, time.Duration, error) {
	return true, 10 * time.Millisecond, nil
}

func TestNewAuthHandler(t *testing.T) {
	authService := &mockAuthService{}
	handler := NewAuthHandler(authService)

	if handler.authService == nil {
		t.Error("expected authService to be set")
	}
}

func TestAuthHandler_ValidateToken(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           any
		expectedStatus int
		checkResponse  func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:           "valid token",
			method:         http.MethodPost,
			body:           map[string]string{"access_token": "valid-token"},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var result dto.ValidateTokenResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !result.Valid {
					t.Error("expected valid response")
				}
			},
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var result map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if result["error"] != "Method not allowed" {
					t.Errorf("expected 'Method not allowed' error, got %v", result)
				}
			},
		},
		{
			name:           "missing token",
			method:         http.MethodPost,
			body:           map[string]string{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var result map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if result["error"] != "access_token is required" {
					t.Errorf("expected 'access_token is required' error, got %v", result)
				}
			},
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var result map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if result["error"] != "Invalid request body" {
					t.Errorf("expected 'Invalid request body' error, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authService := &mockAuthService{}
			handler := NewAuthHandler(authService)

			var body []byte
			if tt.body != nil {
				var err error
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("failed to marshal body: %v", err)
				}
			} else if strBody, ok := tt.body.(string); ok {
				body = []byte(strBody)
			}

			req := httptest.NewRequest(tt.method, "/api/v1/gateway/validate", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ValidateToken(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandler_ValidateToken_ServiceError(t *testing.T) {
	authService := &mockAuthService{
		validateError: errors.ErrServiceUnavailable,
	}
	handler := NewAuthHandler(authService)

	body := map[string]string{"access_token": "valid-token"}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/validate", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.ValidateToken(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
}

func TestAuthHandler_respondJSON(t *testing.T) {
	authService := &mockAuthService{}
	handler := NewAuthHandler(authService)

	data := map[string]string{"key": "value"}
	w := httptest.NewRecorder()

	handler.respondJSON(w, data, http.StatusOK)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected content-type application/json, got %s", contentType)
	}
}

func TestAuthHandler_respondError(t *testing.T) {
	authService := &mockAuthService{}
	handler := NewAuthHandler(authService)

	w := httptest.NewRecorder()

	handler.respondError(w, "test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["error"] != "test error" {
		t.Errorf("expected error 'test error', got %v", result)
	}
}
