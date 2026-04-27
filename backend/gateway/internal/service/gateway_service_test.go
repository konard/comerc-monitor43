package service

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/service/dto"
)

// mockAuthRepoForGateway is a mock implementation of AuthRepository
type mockAuthRepoForGateway struct {
	checkHealthFunc func(ctx context.Context) (bool, error)
}

func (m *mockAuthRepoForGateway) ValidateToken(ctx context.Context, token string) (*model.TokenValidationResult, error) {
	return model.NewTokenValidationResult(true, "user123", "test@example.com", "premium"), nil
}

func (m *mockAuthRepoForGateway) CheckHealth(ctx context.Context) (bool, error) {
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(ctx)
	}
	return true, nil
}

// mockServiceRepoForGateway is a mock implementation of ServiceRepository
type mockServiceRepoForGateway struct {
	checkHealthFunc func(ctx context.Context, serviceName string) (bool, error)
}

func (m *mockServiceRepoForGateway) ProxyRequest(ctx context.Context, serviceName, path string, body io.Reader) (*model.ProxyResponse, error) {
	return model.NewProxyResponse(http.StatusOK, []byte("OK")), nil
}

func (m *mockServiceRepoForGateway) CheckHealth(ctx context.Context, serviceName string) (bool, error) {
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(ctx, serviceName)
	}
	return true, nil
}

// mockServiceHealthChecker is a mock implementation of ServiceHealthChecker
type mockServiceHealthChecker struct {
	results []model.ServiceHealth
}

func (m *mockServiceHealthChecker) CheckAllServices(ctx context.Context) []model.ServiceHealth {
	if len(m.results) > 0 {
		return m.results
	}
	return []model.ServiceHealth{
		{Name: "monitor", Healthy: true, LatencyMS: 10},
		{Name: "alert", Healthy: true, LatencyMS: 5},
	}
}

func TestNewGatewayService(t *testing.T) {
	authRepo := &mockAuthRepoForGateway{}
	serviceRepo := &mockServiceRepoForGateway{}
	healthChecker := &mockServiceHealthChecker{}
	services := []string{"monitor", "alert"}

	service := NewGatewayService(authRepo, serviceRepo, healthChecker, services)

	if service.authRepo == nil {
		t.Error("expected authRepo to be set")
	}
	if service.serviceRepo == nil {
		t.Error("expected serviceRepo to be set")
	}
	if service.healthChecker == nil {
		t.Error("expected healthChecker to be set")
	}
}

func TestGatewayService_CheckHealth(t *testing.T) {
	tests := []struct {
		name           string
		authHealthy    bool
		serviceResults []model.ServiceHealth
		expectedStatus string
	}{
		{
			name:        "all healthy",
			authHealthy: true,
			serviceResults: []model.ServiceHealth{
				{Name: "monitor", Healthy: true, LatencyMS: 10},
				{Name: "alert", Healthy: true, LatencyMS: 5},
			},
			expectedStatus: "healthy",
		},
		{
			name:        "auth unhealthy",
			authHealthy: false,
			serviceResults: []model.ServiceHealth{
				{Name: "monitor", Healthy: true, LatencyMS: 10},
			},
			expectedStatus: "unhealthy",
		},
		{
			name:        "one service degraded",
			authHealthy: true,
			serviceResults: []model.ServiceHealth{
				{Name: "monitor", Healthy: true, LatencyMS: 10},
				{Name: "alert", Healthy: false, LatencyMS: 5},
			},
			expectedStatus: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authRepo := &mockAuthRepoForGateway{
				checkHealthFunc: func(ctx context.Context) (bool, error) {
					return tt.authHealthy, nil
				},
			}
			serviceRepo := &mockServiceRepoForGateway{}
			healthChecker := &mockServiceHealthChecker{results: tt.serviceResults}

			service := NewGatewayService(authRepo, serviceRepo, healthChecker, []string{"monitor", "alert"})

			resp := service.CheckHealth(context.Background())

			if resp.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, resp.Status)
			}

			// Check auth service health
			authFound := false
			for _, svc := range resp.Services {
				if svc.Name == "auth-service" {
					authFound = true
					if svc.Healthy != tt.authHealthy {
						t.Errorf("expected auth healthy %v, got %v", tt.authHealthy, svc.Healthy)
					}
				}
			}
			if !authFound {
				t.Error("auth-service not found in services")
			}
		})
	}
}

func TestGatewayService_CheckAuthHealth(t *testing.T) {
	tests := []struct {
		name           string
		healthy        bool
		expectedHealth bool
	}{
		{
			name:           "healthy",
			healthy:        true,
			expectedHealth: true,
		},
		{
			name:           "unhealthy",
			healthy:        false,
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authRepo := &mockAuthRepoForGateway{
				checkHealthFunc: func(ctx context.Context) (bool, error) {
					return tt.healthy, nil
				},
			}
			serviceRepo := &mockServiceRepoForGateway{}
			healthChecker := &mockServiceHealthChecker{}

			service := NewGatewayService(authRepo, serviceRepo, healthChecker, []string{})

			healthy, _, err := service.CheckAuthHealth(context.Background())

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if healthy != tt.expectedHealth {
				t.Errorf("expected health %v, got %v", tt.expectedHealth, healthy)
			}
		})
	}
}

func TestGatewayService_ProxyRequest(t *testing.T) {
	authRepo := &mockAuthRepoForGateway{}
	serviceRepo := &mockServiceRepoForGateway{}
	healthChecker := &mockServiceHealthChecker{}

	service := NewGatewayService(authRepo, serviceRepo, healthChecker, []string{})

	resp, err := service.ProxyRequest(context.Background(), "monitor", "/api/v1/monitors", nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	if resp.Message == "" {
		t.Error("expected message to be set")
	}
}

func TestGatewayService_GetUptime(t *testing.T) {
	authRepo := &mockAuthRepoForGateway{}
	serviceRepo := &mockServiceRepoForGateway{}
	healthChecker := &mockServiceHealthChecker{}

	service := NewGatewayService(authRepo, serviceRepo, healthChecker, []string{})

	uptime := service.GetUptime()

	if uptime == 0 {
		t.Error("expected non-zero uptime")
	}

	time.Sleep(10 * time.Millisecond)

	newUptime := service.GetUptime()
	if newUptime <= uptime {
		t.Error("expected uptime to increase")
	}
}

func TestNewHealthResponse(t *testing.T) {
	resp := dto.NewHealthResponse("healthy", "2024-01-01T00:00:00Z")

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", resp.Status)
	}

	if resp.Timestamp != "2024-01-01T00:00:00Z" {
		t.Errorf("expected timestamp '2024-01-01T00:00:00Z', got '%s'", resp.Timestamp)
	}

	if resp.Gateway.Name != "api-gateway" {
		t.Errorf("expected gateway name 'api-gateway', got '%s'", resp.Gateway.Name)
	}

	if resp.Gateway.Version != "1.0.0" {
		t.Errorf("expected gateway version '1.0.0', got '%s'", resp.Gateway.Version)
	}
}
