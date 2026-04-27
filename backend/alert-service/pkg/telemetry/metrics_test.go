package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	t.Run("valid_service_name", func(t *testing.T) {
		metrics := NewMetrics("test-service")
		assert.NotNil(t, metrics)
	})

	t.Run("empty_service_name", func(t *testing.T) {
		metrics := NewMetrics("")
		assert.NotNil(t, metrics)
	})
}

func TestMetrics_StructFields(t *testing.T) {
	metrics := NewMetrics("test-service")
	require.NotNil(t, metrics)

	t.Run("all_metrics_initialized", func(t *testing.T) {
		assert.NotNil(t, metrics.meter)
		assert.NotNil(t, metrics.requestCount)
		assert.NotNil(t, metrics.requestDuration)
		assert.NotNil(t, metrics.errorCount)
		assert.NotNil(t, metrics.dbQueryDuration)
		assert.NotNil(t, metrics.alertTriggered)
		assert.NotNil(t, metrics.deliverySuccess)
		assert.NotNil(t, metrics.deliveryFailed)
	})

	t.Run("metrics_have_correct_types", func(t *testing.T) {
		assert.NotNil(t, metrics.requestCount)
		assert.NotNil(t, metrics.requestDuration)
	})
}

func TestMetrics_RecordRequest(t *testing.T) {
	metrics := NewMetrics("test-service")
	require.NotNil(t, metrics)

	ctx := context.Background()

	t.Run("successful_request", func(t *testing.T) {
		metrics.RecordRequest(ctx, "GET", "/test", 200, 150.5)
		assert.NotNil(t, metrics)
	})

	t.Run("client_error_request", func(t *testing.T) {
		metrics.RecordRequest(ctx, "POST", "/test", 404, 250.0)
	})

	t.Run("server_error_request", func(t *testing.T) {
		metrics.RecordRequest(ctx, "PUT", "/test", 500, 125.0)
	})

	t.Run("concurrent_requests", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func() {
				metrics.RecordRequest(ctx, "GET", "/test", 200, 100.0)
			}()
		}
	})
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	metrics := NewMetrics("test-service")
	require.NotNil(t, metrics)

	ctx := context.Background()

	t.Run("fast_query", func(t *testing.T) {
		metrics.RecordDBQuery(ctx, "SELECT", "alerts", 10.5)
	})

	t.Run("slow_query", func(t *testing.T) {
		metrics.RecordDBQuery(ctx, "INSERT", "alerts", 500.0)
	})

	t.Run("concurrent_queries", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func() {
				metrics.RecordDBQuery(ctx, "UPDATE", "alerts", 50.0)
			}()
		}
	})
}

func TestMetrics_RecordAlertTriggered(t *testing.T) {
	metrics := NewMetrics("test-service")
	require.NotNil(t, metrics)

	ctx := context.Background()

	t.Run("trigger_alert", func(t *testing.T) {
		metrics.RecordAlertTriggered(ctx, "status_code")
	})

	t.Run("trigger_multiple_alerts", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			metrics.RecordAlertTriggered(ctx, "status_code")
		}
	})

	t.Run("concurrent_alert_triggers", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func() {
				metrics.RecordAlertTriggered(ctx, "status_code")
			}()
		}
	})
}

func TestMetrics_RecordDeliverySuccess(t *testing.T) {
	metrics := NewMetrics("test-service")
	require.NotNil(t, metrics)

	ctx := context.Background()

	t.Run("successful_delivery", func(t *testing.T) {
		metrics.RecordDeliverySuccess(ctx, "email")
	})

	t.Run("multiple_successful_deliveries", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			metrics.RecordDeliverySuccess(ctx, "email")
		}
	})

	t.Run("different_channel_types", func(t *testing.T) {
		metrics.RecordDeliverySuccess(ctx, "telegram")
		metrics.RecordDeliverySuccess(ctx, "webhook")
	})
}

func TestMetrics_RecordDeliveryFailed(t *testing.T) {
	metrics := NewMetrics("test-service")
	require.NotNil(t, metrics)

	ctx := context.Background()

	t.Run("failed_delivery", func(t *testing.T) {
		metrics.RecordDeliveryFailed(ctx, "email", "timeout_error")
	})

	t.Run("multiple_failed_deliveries", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			metrics.RecordDeliveryFailed(ctx, "telegram", "rate_limit_error")
		}
	})

	t.Run("concurrent_failed_deliveries", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func() {
				metrics.RecordDeliveryFailed(ctx, "webhook", "connection_error")
			}()
		}
	})
}

func TestMetrics_ErrorTypeFromStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected string
	}{
		{"client_error_4xx", 404, "client_error"},
		{"server_error_5xx", 500, "server_error"},
		{"unknown_error", 200, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := errorTypeFromStatus(tt.status)
			assert.Equal(t, tt.expected, errorType)
		})
	}
}

func TestMetrics_ConcurrentRecording(t *testing.T) {
	metrics := NewMetrics("test-service")
	require.NotNil(t, metrics)

	ctx := context.Background()

	t.Run("mixed_operations", func(t *testing.T) {
		go func() {
			metrics.RecordRequest(ctx, "GET", "/test", 200, 100.0)
		}()
		go func() {
			metrics.RecordDBQuery(ctx, "SELECT", "alerts", 50.0)
		}()
		go func() {
			metrics.RecordAlertTriggered(ctx, "status_code")
		}()
		go func() {
			metrics.RecordDeliverySuccess(ctx, "email")
		}()
		go func() {
			metrics.RecordDeliveryFailed(ctx, "telegram", "timeout_error")
		}()
	})

	t.Run("multiple_instances", func(t *testing.T) {
		metrics1 := NewMetrics("service1")
		metrics2 := NewMetrics("service2")

		assert.NotNil(t, metrics1)
		assert.NotNil(t, metrics2)

		ctx := context.Background()
		metrics1.RecordRequest(ctx, "GET", "/test", 200, 100.0)
		metrics2.RecordRequest(ctx, "POST", "/test", 201, 150.0)
	})
}
