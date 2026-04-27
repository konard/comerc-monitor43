package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSN(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "testdb",
		DBUser:     "user",
		DBPassword: "password",
		DBSSLMode:  "disable",
	}

	dsn := cfg.DSN()

	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "user=user")
	assert.Contains(t, dsn, "password=password")
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestRedisAddr_without_password(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RedisHost: "localhost",
		RedisPort: 6379,
	}

	addr := cfg.RedisAddr()

	assert.Equal(t, "localhost:6379", addr)
}

func TestRedisAddr_with_password(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RedisHost:     "redis-host",
		RedisPort:     6380,
		RedisPassword: "secret",
	}

	addr := cfg.RedisAddr()

	assert.Equal(t, ":secret@redis-host:6380", addr)
}

func TestRabbitMQURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RabbitMQUser:     "guest",
		RabbitMQPassword: "guest",
		RabbitMQHost:     "localhost",
		RabbitMQPort:     5672,
		RabbitMQVHost:    "/",
	}

	url := cfg.RabbitMQURL()

	assert.Equal(t, "amqp://guest:guest@localhost:5672/", url)
}

func TestLoadConfig_defaults(t *testing.T) {
	t.Parallel()

	// LoadConfig требует обязательные поля — проверяем, что без env возвращает ошибку валидации
	_, err := LoadConfig()
	require.Error(t, err)
}

func TestLoadConfig_with_required_env(t *testing.T) {
	// t.Setenv несовместим с t.Parallel()
	t.Setenv("DB_PASSWORD", "secure-password-123")
	t.Setenv("JWT_SECRET", "super-secret-key-min-32-chars-ok")
	t.Setenv("GOOGLE_CLIENT_ID", "google-client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "google-client-secret")
	t.Setenv("GOOGLE_REDIRECT_URI", "https://example.com/callback")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, 5001, cfg.ServerGRPCPort)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "super-secret-key-min-32-chars-ok", cfg.JWTSecret)
}
