package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raul/monitor/gateway/internal/service/dto"
)

// mockGatewayService is a mock implementation of GatewayService
type mockGatewayService struct {
	healthResponse  *dto.HealthResponse
	checkAuthHealth func(ctx context.Context) (bool, time.Duration, error)
}

func (m *mockGatewayService) CheckHealth(ctx context.Context) *dto.HealthResponse {
	if m.healthResponse != nil {
		return m.healthResponse
	}
	return dto.NewHealthResponse("healthy", time.Now().Format(time.RFC3339))
}

func (m *mockGatewayService) CheckAuthHealth(ctx context.Context) (bool, time.Duration, error) {
	if m.checkAuthHealth != nil {
		return m.checkAuthHealth(ctx)
	}
	return true, 10 * time.Millisecond, nil
}

func (m *mockGatewayService) ProxyRequest(ctx context.Context, serviceName, path string, body []byte) (*dto.ProxyResponse, error) {
	return &dto.ProxyResponse{
		StatusCode: http.StatusOK,
		Message:    "Proxy not implemented",
	}, nil
}

func (m *mockGatewayService) GetUptime() time.Duration {
	return time.Minute
}

func TestNewGatewayHandler(t *testing.T) {
	gatewayService := &mockGatewayService{}
	handler := NewGatewayHandler(gatewayService)

	if handler.gatewayService == nil {
		t.Error("expected gatewayService to be set")
	}
}

func TestGatewayHandler_Health(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		healthStatus   string
		expectedStatus int
	}{
		{
			name:           "healthy",
			method:         http.MethodGet,
			healthStatus:   "healthy",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "degraded",
			method:         http.MethodGet,
			healthStatus:   "degraded",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid method",
			method:         http.MethodPost,
			healthStatus:   "healthy",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gatewayService := &mockGatewayService{
				healthResponse: dto.NewHealthResponse(tt.healthStatus, time.Now().Format(time.RFC3339)),
			}
			handler := NewGatewayHandler(gatewayService)

			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			handler.Health(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGatewayHandler_Readiness(t *testing.T) {
	tests := []struct {
		name           string
		healthStatus   string
		expectedStatus int
	}{
		{
			name:           "ready",
			healthStatus:   "healthy",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not ready",
			healthStatus:   "degraded",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "not ready unhealthy",
			healthStatus:   "unhealthy",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gatewayService := &mockGatewayService{
				healthResponse: dto.NewHealthResponse(tt.healthStatus, time.Now().Format(time.RFC3339)),
			}
			handler := NewGatewayHandler(gatewayService)

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			w := httptest.NewRecorder()

			handler.Readiness(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGatewayHandler_Liveness(t *testing.T) {
	gatewayService := &mockGatewayService{}
	handler := NewGatewayHandler(gatewayService)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "live",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid method",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/live", nil)
			w := httptest.NewRecorder()

			handler.Liveness(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGatewayHandler_Proxy(t *testing.T) {
	gatewayService := &mockGatewayService{}
	handler := NewGatewayHandler(gatewayService)

	// Note: PathValue is available in Go 1.22+
	// For Go 1.21, we'll just test the error case
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxy/", nil)
	w := httptest.NewRecorder()

	handler.Proxy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGatewayHandler_respondJSON(t *testing.T) {
	gatewayService := &mockGatewayService{}
	handler := NewGatewayHandler(gatewayService)

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

func TestGatewayHandler_respondError(t *testing.T) {
	gatewayService := &mockGatewayService{}
	handler := NewGatewayHandler(gatewayService)

	w := httptest.NewRecorder()

	handler.respondError(w, "test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Check for JSON error response
	if w.Body.Len() == 0 {
		t.Error("expected response body")
	}
}
