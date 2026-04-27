package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockPinger struct {
	connected bool
}

func (m *mockPinger) IsConnected() bool {
	return m.connected
}

func TestHealthChecker_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		schedulerConn *mockPinger
		monitorConn   *mockPinger
		wantErr       bool
	}{
		{
			name:          "all_healthy",
			schedulerConn: &mockPinger{connected: true},
			monitorConn:   &mockPinger{connected: true},
			wantErr:       false,
		},
		{
			name:          "scheduler_unhealthy",
			schedulerConn: &mockPinger{connected: false},
			monitorConn:   &mockPinger{connected: true},
			wantErr:       true,
		},
		{
			name:          "monitor_unhealthy",
			schedulerConn: &mockPinger{connected: true},
			monitorConn:   &mockPinger{connected: false},
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hc := NewHealthChecker(tc.schedulerConn, tc.monitorConn)
			err := hc.Check(context.Background())

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHealthChecker_Check_nil_connections(t *testing.T) {
	t.Parallel()

	hc := NewHealthChecker(nil, nil)
	err := hc.Check(context.Background())
	assert.NoError(t, err)
}

func TestHealthChecker_Check_nil_scheduler_monitor_ok(t *testing.T) {
	t.Parallel()

	hc := NewHealthChecker(nil, &mockPinger{connected: true})
	err := hc.Check(context.Background())
	assert.NoError(t, err)
}

func TestHealthChecker_ServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		schedulerConn *mockPinger
		monitorConn   *mockPinger
		wantStatus    int
		wantBody      string
	}{
		{
			name:          "healthy_response",
			schedulerConn: &mockPinger{connected: true},
			monitorConn:   &mockPinger{connected: true},
			wantStatus:    http.StatusOK,
			wantBody:      `{"status":"healthy"}`,
		},
		{
			name:          "unhealthy_response",
			schedulerConn: &mockPinger{connected: false},
			monitorConn:   &mockPinger{connected: true},
			wantStatus:    http.StatusServiceUnavailable,
			wantBody:      `{"status":"unhealthy","error":"scheduler connection unhealthy"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hc := NewHealthChecker(tc.schedulerConn, tc.monitorConn)
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			hc.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.JSONEq(t, tc.wantBody, rec.Body.String())
		})
	}
}
