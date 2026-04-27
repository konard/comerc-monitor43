package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics_success(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")

	require.NoError(t, err)
	assert.NotNil(t, m)
}

func TestMetrics_RecordRequest_does_not_panic(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		m.RecordRequest(context.Background(), "POST", "/api/v1/test", 200, 42.5)
	})
}

func TestMetrics_RecordRequest_error_statuses(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	for _, code := range []int{400, 404, 500, 503} {
		t.Run("status_code", func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				m.RecordRequest(context.Background(), "GET", "/test", code, 10.0)
			})
		})
	}
}

func TestMetrics_RecordDBQuery_does_not_panic(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		m.RecordDBQuery(context.Background(), "select", "monitor_statuses", 5.0)
	})
}

func TestMetrics_RecordDashboardView_does_not_panic(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		m.RecordDashboardView(context.Background(), "dashboard-123")
	})
}

func TestMetrics_RecordWSConnection_does_not_panic(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		m.RecordWSConnection(context.Background(), "connect")
		m.RecordWSConnection(context.Background(), "disconnect")
	})
}

func TestMetrics_RecordEventProcessed_does_not_panic(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		m.RecordEventProcessed(context.Background(), "monitor.created")
	})
}

func TestErrorTypeFromStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		code     int
		wantType string
	}{
		{400, "client_error"},
		{404, "client_error"},
		{499, "client_error"},
		{500, "server_error"},
		{503, "server_error"},
		{200, "unknown"},
		{301, "unknown"},
	}

	for _, tc := range cases {
		t.Run("status", func(t *testing.T) {
			t.Parallel()

			result := errorTypeFromStatus(tc.code)
			assert.Equal(t, tc.wantType, result)
		})
	}
}

func TestNewTracer_success(t *testing.T) {
	t.Parallel()

	tr, err := NewTracer("test-service", "http://localhost:4318")

	require.NoError(t, err)
	assert.NotNil(t, tr)
}

func TestTracer_StartSpan_does_not_panic(t *testing.T) {
	t.Parallel()

	tr, err := NewTracer("test-service", "")
	require.NoError(t, err)

	ctx, span := tr.StartSpan(context.Background(), "test-span")

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestTracer_GetTracer_returns_tracer(t *testing.T) {
	t.Parallel()

	tr, err := NewTracer("test-service", "")
	require.NoError(t, err)

	tracer := tr.GetTracer()

	assert.NotNil(t, tracer)
}

func TestTracer_Shutdown_does_not_error(t *testing.T) {
	t.Parallel()

	tr, err := NewTracer("test-service", "")
	require.NoError(t, err)

	err = tr.Shutdown(context.Background())

	assert.NoError(t, err)
}
