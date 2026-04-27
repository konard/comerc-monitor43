package grpc

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raul/monitor/gateway/internal/model"
)

func TestNewServiceRepository(t *testing.T) {
	repo := NewServiceRepository(30*time.Second, map[string]string{
		"monitor": "http://localhost:5002",
	})

	if repo == nil {
		t.Error("expected repository to be created")
	}
}

func TestServiceRepository_CheckHealth(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		healthy    bool
	}{
		{
			name:       "healthy service",
			statusCode: http.StatusOK,
			healthy:    true,
		},
		{
			name:       "unhealthy service",
			statusCode: http.StatusServiceUnavailable,
			healthy:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			repo := NewServiceRepository(5*time.Second, map[string]string{
				"test": server.URL,
			})

			healthy, err := repo.CheckHealth(context.Background(), "test")

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if healthy != tt.healthy {
				t.Errorf("expected healthy %v, got %v", tt.healthy, healthy)
			}
		})
	}
}

func TestServiceRepository_CheckHealth_ServiceNotFound(t *testing.T) {
	repo := NewServiceRepository(5*time.Second, map[string]string{})

	_, err := repo.CheckHealth(context.Background(), "non-existent")

	if err == nil {
		t.Error("expected error for non-existent service")
	}
}

func TestServiceRepository_ProxyRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	repo := NewServiceRepository(5*time.Second, map[string]string{
		"test": server.URL,
	})

	resp, err := repo.ProxyRequest(context.Background(), "test", "/api/test", nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if string(resp.Body) != `{"status":"ok"}` {
		t.Errorf("expected body '{\"status\":\"ok\"}', got '%s'", string(resp.Body))
	}
}

func TestServiceRepository_ProxyRequest_Error(t *testing.T) {
	repo := NewServiceRepository(5*time.Second, map[string]string{})

	resp, err := repo.ProxyRequest(context.Background(), "non-existent", "/api/test", nil)

	// Should return error response, not an error (graceful degradation)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error to be set in response")
	}
}

func TestNewServiceHealthChecker(t *testing.T) {
	mockRepo := &mockServiceRepoForHealthChecker{
		healthResults: map[string]bool{
			"auth":    true,
			"monitor": true,
			"alert":   false,
		},
	}

	checker := NewServiceHealthChecker(mockRepo, []string{"auth", "monitor", "alert"})

	if checker == nil {
		t.Error("expected checker to be created")
	}

	results := checker.CheckAllServices(context.Background())

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

// mockServiceRepoForHealthChecker is a mock for health checker tests
type mockServiceRepoForHealthChecker struct {
	healthResults map[string]bool
}

func (m *mockServiceRepoForHealthChecker) ProxyRequest(ctx context.Context, serviceName, path string, body io.Reader) (*model.ProxyResponse, error) {
	return nil, nil
}

func (m *mockServiceRepoForHealthChecker) CheckHealth(ctx context.Context, serviceName string) (bool, error) {
	healthy, ok := m.healthResults[serviceName]
	if !ok {
		return false, nil
	}
	return healthy, nil
}

func TestServiceHealthChecker_CheckAllServices(t *testing.T) {
	mockRepo := &mockServiceRepoForHealthChecker{
		healthResults: map[string]bool{
			"auth":    true,
			"monitor": true,
			"alert":   false,
		},
	}

	checker := NewServiceHealthChecker(mockRepo, []string{"auth", "monitor", "alert"})
	results := checker.CheckAllServices(context.Background())

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Check auth
	authFound := false
	for _, r := range results {
		if r.Name == "auth" {
			authFound = true
			if !r.Healthy {
				t.Error("expected auth to be healthy")
			}
		}
	}
	if !authFound {
		t.Error("auth not found in results")
	}

	// Check alert
	alertFound := false
	for _, r := range results {
		if r.Name == "alert" {
			alertFound = true
			if r.Healthy {
				t.Error("expected alert to be unhealthy")
			}
		}
	}
	if !alertFound {
		t.Error("alert not found in results")
	}
}
