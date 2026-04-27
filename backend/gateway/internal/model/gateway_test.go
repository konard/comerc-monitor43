package model

import (
	"testing"
)

func TestNewHealthStatus(t *testing.T) {
	status := NewHealthStatus("healthy")

	if status.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", status.Status)
	}

	if status.Gateway.Name != "api-gateway" {
		t.Errorf("expected gateway name 'api-gateway', got '%s'", status.Gateway.Name)
	}

	if status.Gateway.Version != "1.0.0" {
		t.Errorf("expected gateway version '1.0.0', got '%s'", status.Gateway.Version)
	}

	if len(status.Services) != 0 {
		t.Errorf("expected no services, got %d", len(status.Services))
	}
}

func TestHealthStatus_AddService(t *testing.T) {
	status := NewHealthStatus("healthy")

	status.AddService("auth-service", true, 10, "OK")
	status.AddService("monitor-service", false, 20, "Connection failed")

	if len(status.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(status.Services))
	}

	// Check first service
	if status.Services[0].Name != "auth-service" {
		t.Errorf("expected first service name 'auth-service', got '%s'", status.Services[0].Name)
	}
	if !status.Services[0].Healthy {
		t.Error("expected first service to be healthy")
	}
	if status.Services[0].LatencyMS != 10 {
		t.Errorf("expected first service latency 10ms, got %d", status.Services[0].LatencyMS)
	}

	// Check second service
	if status.Services[1].Name != "monitor-service" {
		t.Errorf("expected second service name 'monitor-service', got '%s'", status.Services[1].Name)
	}
	if status.Services[1].Healthy {
		t.Error("expected second service to be unhealthy")
	}
	if status.Services[1].Message != "Connection failed" {
		t.Errorf("expected second service message 'Connection failed', got '%s'", status.Services[1].Message)
	}
}

func TestHealthStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		services       []ServiceHealth
		expectedHealth bool
	}{
		{
			name:           "healthy status, no services",
			status:         "healthy",
			services:       []ServiceHealth{},
			expectedHealth: true,
		},
		{
			name:   "healthy status, all services healthy",
			status: "healthy",
			services: []ServiceHealth{
				{Name: "auth", Healthy: true},
				{Name: "monitor", Healthy: true},
			},
			expectedHealth: true,
		},
		{
			name:   "unhealthy status",
			status: "unhealthy",
			services: []ServiceHealth{
				{Name: "auth", Healthy: true},
			},
			expectedHealth: false,
		},
		{
			name:   "degraded status",
			status: "degraded",
			services: []ServiceHealth{
				{Name: "auth", Healthy: true},
			},
			expectedHealth: false,
		},
		{
			name:   "one service unhealthy",
			status: "healthy",
			services: []ServiceHealth{
				{Name: "auth", Healthy: true},
				{Name: "monitor", Healthy: false},
			},
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewHealthStatus(tt.status)
			for _, svc := range tt.services {
				status.AddService(svc.Name, svc.Healthy, svc.LatencyMS, svc.Message)
			}

			got := status.IsHealthy()
			if got != tt.expectedHealth {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.expectedHealth)
			}
		})
	}
}

func TestServiceHealth(t *testing.T) {
	sh := ServiceHealth{
		Name:      "test-service",
		Healthy:   true,
		LatencyMS: 42,
		Message:   "All good",
	}

	if sh.Name != "test-service" {
		t.Errorf("expected name 'test-service', got '%s'", sh.Name)
	}

	if !sh.Healthy {
		t.Error("expected healthy to be true")
	}

	if sh.LatencyMS != 42 {
		t.Errorf("expected latency 42ms, got %d", sh.LatencyMS)
	}

	if sh.Message != "All good" {
		t.Errorf("expected message 'All good', got '%s'", sh.Message)
	}
}

func TestGateway(t *testing.T) {
	gw := Gateway{
		Name:      "api-gateway",
		Version:   "1.0.0",
		StartedAt: "2024-01-01T00:00:00Z",
	}

	if gw.Name != "api-gateway" {
		t.Errorf("expected name 'api-gateway', got '%s'", gw.Name)
	}

	if gw.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", gw.Version)
	}

	if gw.StartedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("expected start time '2024-01-01T00:00:00Z', got '%s'", gw.StartedAt)
	}
}
