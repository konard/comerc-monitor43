package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	validBase := func() *Config {
		return &Config{
			ServerPort:     8084,
			ServerGRPCPort: 5014,
			DBHost:         "localhost",
			DBPort:         5432,
			DBName:         "testdb",
			DBUser:         "testuser",
			DBPassword:     "testpass",
			DBSSLMode:      "disable",
			EncryptionKey:  "12345678901234567890123456789012", // 32 bytes
			JWTSecret:      "test-jwt-secret",
		}
	}

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		assert.NoError(t, cfg.Validate())
	})

	t.Run("invalid server port zero", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.ServerPort = 0
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid server port too large", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.ServerPort = 99999
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid grpc port zero", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.ServerGRPCPort = 0
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid grpc port too large", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.ServerGRPCPort = 70000
		assert.Error(t, cfg.Validate())
	})

	t.Run("empty db host", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.DBHost = ""
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid db port", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.DBPort = 0
		assert.Error(t, cfg.Validate())
	})

	t.Run("empty db name", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.DBName = ""
		assert.Error(t, cfg.Validate())
	})

	t.Run("empty db user", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.DBUser = ""
		assert.Error(t, cfg.Validate())
	})

	t.Run("empty db password", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.DBPassword = ""
		assert.Error(t, cfg.Validate())
	})

	t.Run("empty encryption key", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.EncryptionKey = ""
		assert.Error(t, cfg.Validate())
	})

	t.Run("short encryption key", func(t *testing.T) {
		t.Parallel()
		cfg := validBase()
		cfg.EncryptionKey = "short"
		assert.Error(t, cfg.Validate())
	})
}

func TestConfigDSN(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "mydb",
		DBUser:     "myuser",
		DBPassword: "mypass",
		DBSSLMode:  "disable",
	}

	dsn := cfg.DSN()
	assert.Contains(t, dsn, "localhost")
	assert.Contains(t, dsn, "5432")
	assert.Contains(t, dsn, "mydb")
	assert.Contains(t, dsn, "myuser")
	assert.Contains(t, dsn, "mypass")
	assert.Contains(t, dsn, "disable")
}

func TestConfigRabbitMQURL(t *testing.T) {
	t.Parallel()

	t.Run("disabled rabbitmq returns empty string", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			RabbitMQEnabled: false,
		}
		assert.Equal(t, "", cfg.RabbitMQURL())
	})

	t.Run("enabled rabbitmq returns url", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			RabbitMQEnabled:  true,
			RabbitMQUser:     "guest",
			RabbitMQPassword: "guest",
			RabbitMQHost:     "localhost",
			RabbitMQPort:     5672,
			RabbitMQVHost:    "/",
		}
		url := cfg.RabbitMQURL()
		assert.Contains(t, url, "amqp://")
		assert.Contains(t, url, "guest:guest@localhost:5672")
	})
}

func TestConfigRabbitMQExchangeName(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RabbitMQExchange: "monitor",
	}
	assert.Equal(t, "monitor", cfg.RabbitMQExchangeName())
}

func TestGetEnv(t *testing.T) {
	// t.Setenv не совместим с t.Parallel

	t.Run("existing env var", func(t *testing.T) {
		t.Setenv("TEST_GET_ENV_EXISTING", "hello")
		result := getEnv("TEST_GET_ENV_EXISTING", "default")
		assert.Equal(t, "hello", result)
	})

	t.Run("missing env var returns default", func(t *testing.T) {
		result := getEnv("TEST_GET_ENV_MISSING_XYZ_12345", "mydefault")
		assert.Equal(t, "mydefault", result)
	})
}

func TestGetEnvInt(t *testing.T) {
	// t.Setenv не совместим с t.Parallel

	t.Run("valid int env var", func(t *testing.T) {
		t.Setenv("TEST_INT_ENV", "42")
		result := getEnvInt("TEST_INT_ENV", 0)
		assert.Equal(t, 42, result)
	})

	t.Run("invalid int env var returns default", func(t *testing.T) {
		t.Setenv("TEST_INT_ENV_INVALID", "not-an-int")
		result := getEnvInt("TEST_INT_ENV_INVALID", 99)
		assert.Equal(t, 99, result)
	})

	t.Run("missing env var returns default", func(t *testing.T) {
		result := getEnvInt("TEST_INT_ENV_MISSING_XYZ", 55)
		assert.Equal(t, 55, result)
	})
}

func TestGetEnvBool(t *testing.T) {
	// t.Setenv не совместим с t.Parallel

	t.Run("true value", func(t *testing.T) {
		t.Setenv("TEST_BOOL_ENV_TRUE", "true")
		result := getEnvBool("TEST_BOOL_ENV_TRUE", false)
		assert.True(t, result)
	})

	t.Run("false value", func(t *testing.T) {
		t.Setenv("TEST_BOOL_ENV_FALSE", "false")
		result := getEnvBool("TEST_BOOL_ENV_FALSE", true)
		assert.False(t, result)
	})

	t.Run("invalid bool returns default", func(t *testing.T) {
		t.Setenv("TEST_BOOL_ENV_INVALID", "not-a-bool")
		result := getEnvBool("TEST_BOOL_ENV_INVALID", true)
		assert.True(t, result)
	})

	t.Run("missing env var returns default", func(t *testing.T) {
		result := getEnvBool("TEST_BOOL_ENV_MISSING_XYZ", true)
		assert.True(t, result)
	})
}

func TestLoadConfigWithValidEnv(t *testing.T) {
	// t.Setenv не совместим с t.Parallel

	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_HOST", "localhost")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "12345678901234567890123456789012", cfg.EncryptionKey)
}
