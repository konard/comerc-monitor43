package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_defaults(t *testing.T) {
	// Устанавливаем обязательные переменные с валидными значениями
	t.Setenv("DB_PASSWORD", "password123") // минимум 8 символов
	t.Setenv("YOOKASSA_RETURN_URL", "https://example.com/return")
	t.Setenv("YOOKASSA_SECRET_KEY", "12345678901234567890123456789012")    // минимум 32 символа
	t.Setenv("STRIPE_API_KEY", "sk_test_12345678901234567890123456789012") // минимум 32 символа

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 8082, cfg.Server.Port)
	assert.Equal(t, 5003, cfg.Server.GRPCPort)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "billing", cfg.Database.Name)
}

func TestLoad_custom_values(t *testing.T) {
	// Устанавливаем кастомные значения
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_GRPC_PORT", "9091")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_NAME", "test_billing")
	t.Setenv("DB_USER", "test_user")
	t.Setenv("DB_PASSWORD", "mysecretpassword")
	t.Setenv("DB_SSL_MODE", "require")
	t.Setenv("YOOKASSA_RETURN_URL", "https://example.com/return")
	t.Setenv("YOOKASSA_SECRET_KEY", "12345678901234567890123456789012")
	t.Setenv("STRIPE_API_KEY", "sk_test_12345678901234567890123456789012")
	t.Setenv("AUTH_SERVICE_GRPC_ADDRESS", "auth:5001")
	t.Setenv("RABBITMQ_URL", "amqp://user:pass@rabbit:5672/")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 9091, cfg.Server.GRPCPort)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "test_billing", cfg.Database.Name)
	assert.Equal(t, "require", cfg.Database.SSLMode)
}

func TestLoad_invalid_ssl_mode(t *testing.T) {
	// SSL mode невалидный
	t.Setenv("DB_SSL_MODE", "invalid_mode")
	t.Setenv("DB_PASSWORD", "password123")
	t.Setenv("YOOKASSA_RETURN_URL", "https://example.com/return")
	t.Setenv("YOOKASSA_SECRET_KEY", "12345678901234567890123456789012")
	t.Setenv("STRIPE_API_KEY", "sk_test_12345678901234567890123456789012")

	_, err := Load()

	assert.Error(t, err)
}
