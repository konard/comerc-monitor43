package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("success_with_defaults", func(t *testing.T) {
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_PORT", "5432")
		t.Setenv("DB_NAME", "testdb")
		t.Setenv("DB_USER", "testuser")
		t.Setenv("DB_PASSWORD", "testpassword123")
		t.Setenv("MONITOR_SERVICE_GRPC_ADDRESS", "localhost:5002")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, 8084, cfg.Server.Port)
		assert.Equal(t, 5007, cfg.Server.GRPCPort)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, "testdb", cfg.Database.Name)
		assert.Equal(t, "testuser", cfg.Database.User)
		assert.Equal(t, "reporting-service", cfg.Tracing.ServiceName)
		assert.Equal(t, 90, cfg.Reporting.RetentionDays)
		assert.Equal(t, 50000, cfg.Reporting.MaxExportRows)
	})

	t.Run("success_with_custom_values", func(t *testing.T) {
		t.Setenv("SERVER_PORT", "9090")
		t.Setenv("SERVER_GRPC_PORT", "6000")
		t.Setenv("DB_HOST", "db-host")
		t.Setenv("DB_PORT", "5433")
		t.Setenv("DB_NAME", "mydb")
		t.Setenv("DB_USER", "myuser")
		t.Setenv("DB_PASSWORD", "mypass123")
		t.Setenv("DB_SSL_MODE", "require")
		t.Setenv("OTEL_ENABLED", "false")
		t.Setenv("OTEL_SERVICE_NAME", "my-reporting")
		t.Setenv("METRICS_ENABLED", "false")
		t.Setenv("METRICS_PORT", "9100")
		t.Setenv("MONITOR_SERVICE_GRPC_ADDRESS", "monitor:5002")
		t.Setenv("JWT_SECRET_KEY", "super-secret-key")
		t.Setenv("REPORTING_RETENTION_DAYS", "30")
		t.Setenv("REPORTING_MAX_EXPORT_ROWS", "10000")
		t.Setenv("LOG_LEVEL", "debug")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, 6000, cfg.Server.GRPCPort)
		assert.Equal(t, "db-host", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "mydb", cfg.Database.Name)
		assert.Equal(t, "myuser", cfg.Database.User)
		assert.Equal(t, "require", cfg.Database.SSLMode)
		assert.Equal(t, false, cfg.Tracing.Enabled)
		assert.Equal(t, "my-reporting", cfg.Tracing.ServiceName)
		assert.Equal(t, false, cfg.Metrics.Enabled)
		assert.Equal(t, 9100, cfg.Metrics.Port)
		assert.Equal(t, 30, cfg.Reporting.RetentionDays)
		assert.Equal(t, 10000, cfg.Reporting.MaxExportRows)
		assert.Equal(t, "debug", cfg.LogLevel)
	})

	t.Run("error_invalid_port", func(t *testing.T) {
		t.Setenv("SERVER_PORT", "99999")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_NAME", "testdb")
		t.Setenv("DB_USER", "testuser")
		t.Setenv("DB_PASSWORD", "testpassword123")
		t.Setenv("MONITOR_SERVICE_GRPC_ADDRESS", "localhost:5002")

		_, err := Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config")
	})
}
