package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables to test defaults
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_SSL_MODE")
	os.Unsetenv("DB_MAX_OPEN_CONNS")
	os.Unsetenv("DB_MAX_IDLE_CONNS")
	os.Unsetenv("DB_MAX_LIFETIME")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_SERVICE_NAME")
	os.Unsetenv("OTEL_LOG_LEVEL")
	os.Setenv("JWT_SECRET", "test-secret-key-that-is-at-least-32-chars!") // JWT_SECRET не имеет дефолтного значения — обязательный параметр
	defer os.Unsetenv("JWT_SECRET")
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")
	os.Unsetenv("SMTP_USER")
	os.Unsetenv("SMTP_PASSWORD")
	os.Unsetenv("SMTP_FROM")
	os.Unsetenv("WEBHOOK_TIMEOUT")
	os.Unsetenv("DELIVERY_MAX_RETRIES")
	os.Unsetenv("DELIVERY_RETRY_INTERVAL")

	cfg, err := Load()
	require.NoError(t, err)

	// Check default values
	assert.Equal(t, 50051, cfg.ServerPort)
	assert.Equal(t, "0.0.0.0", cfg.ServerHost)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "postgres", cfg.DBUser)
	assert.Equal(t, "postgres", cfg.DBPassword)
	assert.Equal(t, "milan", cfg.DBName)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, 25, cfg.DBMaxOpenConns)
	assert.Equal(t, 5, cfg.DBMaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.DBMaxLifetime)
	assert.Equal(t, "http://localhost:4318", cfg.OTELExporterOTLPEndpoint)
	assert.Equal(t, "alert-service", cfg.OTELServiceName)
	assert.Equal(t, "info", cfg.OTELLogLevel)
	assert.Equal(t, "test-secret-key-that-is-at-least-32-chars!", cfg.JWTSecret)
	assert.Equal(t, "smtp.gmail.com", cfg.SMTPHost)
	assert.Equal(t, 587, cfg.SMTPPort)
	assert.Equal(t, "noreply@monitor.example.com", cfg.SMTPFrom)
	assert.Equal(t, 30*time.Second, cfg.WebhookTimeout)
	assert.Equal(t, 3, cfg.DeliveryMaxRetries)
	assert.Equal(t, 5*time.Minute, cfg.DeliveryRetryInterval)
}

func TestLoad_CustomValues(t *testing.T) {
	// Set custom environment variables
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("SERVER_HOST", "192.168.1.1")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_SSL_MODE", "require")
	os.Setenv("DB_MAX_OPEN_CONNS", "50")
	os.Setenv("DB_MAX_IDLE_CONNS", "10")
	os.Setenv("DB_MAX_LIFETIME", "10m")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4318")
	os.Setenv("OTEL_SERVICE_NAME", "custom-service")
	os.Setenv("OTEL_LOG_LEVEL", "debug")
	os.Setenv("JWT_SECRET", "custom-secret-key-that-is-at-least-32-chars!")
	os.Setenv("TELEGRAM_BOT_TOKEN", "bot-token")
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("SMTP_PORT", "2525")
	os.Setenv("SMTP_USER", "smtpuser")
	os.Setenv("SMTP_PASSWORD", "smtppass")
	os.Setenv("SMTP_FROM", "noreply@example.com")
	os.Setenv("WEBHOOK_TIMEOUT", "60s")
	os.Setenv("DELIVERY_MAX_RETRIES", "5")
	os.Setenv("DELIVERY_RETRY_INTERVAL", "10m")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSL_MODE")
		os.Unsetenv("DB_MAX_OPEN_CONNS")
		os.Unsetenv("DB_MAX_IDLE_CONNS")
		os.Unsetenv("DB_MAX_LIFETIME")
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		os.Unsetenv("OTEL_SERVICE_NAME")
		os.Unsetenv("OTEL_LOG_LEVEL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USER")
		os.Unsetenv("SMTP_PASSWORD")
		os.Unsetenv("SMTP_FROM")
		os.Unsetenv("WEBHOOK_TIMEOUT")
		os.Unsetenv("DELIVERY_MAX_RETRIES")
		os.Unsetenv("DELIVERY_RETRY_INTERVAL")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	// Check custom values are applied
	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, "192.168.1.1", cfg.ServerHost)
	assert.Equal(t, "db.example.com", cfg.DBHost)
	assert.Equal(t, 5433, cfg.DBPort)
	assert.Equal(t, "testuser", cfg.DBUser)
	assert.Equal(t, "testpass", cfg.DBPassword)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "require", cfg.DBSSLMode)
	assert.Equal(t, 50, cfg.DBMaxOpenConns)
	assert.Equal(t, 10, cfg.DBMaxIdleConns)
	assert.Equal(t, 10*time.Minute, cfg.DBMaxLifetime)
	assert.Equal(t, "http://jaeger:4318", cfg.OTELExporterOTLPEndpoint)
	assert.Equal(t, "custom-service", cfg.OTELServiceName)
	assert.Equal(t, "debug", cfg.OTELLogLevel)
	assert.Equal(t, "custom-secret-key-that-is-at-least-32-chars!", cfg.JWTSecret)
	assert.Equal(t, "bot-token", cfg.TelegramBotToken)
	assert.Equal(t, "smtp.example.com", cfg.SMTPHost)
	assert.Equal(t, 2525, cfg.SMTPPort)
	assert.Equal(t, "smtpuser", cfg.SMTPUser)
	assert.Equal(t, "smtppass", cfg.SMTPPassword)
	assert.Equal(t, "noreply@example.com", cfg.SMTPFrom)
	assert.Equal(t, 60*time.Second, cfg.WebhookTimeout)
	assert.Equal(t, 5, cfg.DeliveryMaxRetries)
	assert.Equal(t, 10*time.Minute, cfg.DeliveryRetryInterval)
}

func TestConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "default DSN",
			config: Config{
				DBHost:     "localhost",
				DBPort:     5432,
				DBUser:     "postgres",
				DBPassword: "postgres",
				DBName:     "milan",
				DBSSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=postgres dbname=milan sslmode=disable",
		},
		{
			name: "custom DSN",
			config: Config{
				DBHost:     "db.example.com",
				DBPort:     5433,
				DBUser:     "testuser",
				DBPassword: "testpass",
				DBName:     "testdb",
				DBSSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=testuser password=testpass dbname=testdb sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DSN()
			assert.Equal(t, tt.expected, dsn)
		})
	}
}

func TestConfig_ServerAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "default server address",
			config: Config{
				ServerHost: "0.0.0.0",
				ServerPort: 50051,
			},
			expected: "0.0.0.0:50051",
		},
		{
			name: "custom server address",
			config: Config{
				ServerHost: "192.168.1.1",
				ServerPort: 8080,
			},
			expected: "192.168.1.1:8080",
		},
		{
			name: "localhost with port",
			config: Config{
				ServerHost: "localhost",
				ServerPort: 9090,
			},
			expected: "localhost:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := tt.config.ServerAddress()
			assert.Equal(t, tt.expected, address)
		})
	}
}

func TestLoad_MultipleCalls(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-that-is-at-least-32-chars!")
	defer os.Unsetenv("JWT_SECRET")

	// Test that multiple Load calls work correctly
	cfg1, err1 := Load()
	require.NoError(t, err1)
	assert.NotNil(t, cfg1)

	cfg2, err2 := Load()
	require.NoError(t, err2)
	assert.NotNil(t, cfg2)

	// Both configs should have the same structure
	assert.Equal(t, cfg1.ServerPort, cfg2.ServerPort)
	assert.Equal(t, cfg1.DBHost, cfg2.DBHost)
}

func TestLoad_EmptyEnvVars(t *testing.T) {
	// Test with empty environment variables (should use defaults)
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_USER")
	os.Setenv("JWT_SECRET", "test-secret-key-that-is-at-least-32-chars!")
	defer os.Unsetenv("JWT_SECRET")

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 50051, cfg.ServerPort)   // default value
	assert.Equal(t, "localhost", cfg.DBHost) // default value
}

func TestConfig_StructFields(t *testing.T) {
	cfg := &Config{
		ServerPort:               8080,
		ServerHost:               "localhost",
		DBHost:                   "db.example.com",
		DBPort:                   5432,
		DBUser:                   "user",
		DBPassword:               "pass",
		DBName:                   "dbname",
		DBSSLMode:                "disable",
		DBMaxOpenConns:           25,
		DBMaxIdleConns:           5,
		DBMaxLifetime:            5 * time.Minute,
		OTELExporterOTLPEndpoint: "http://otel:4318",
		OTELServiceName:          "service",
		OTELLogLevel:             "info",
		JWTSecret:                "secret",
		TelegramBotToken:         "token",
		SMTPHost:                 "smtp.example.com",
		SMTPPort:                 587,
		SMTPUser:                 "user",
		SMTPPassword:             "pass",
		SMTPFrom:                 "from@example.com",
		WebhookTimeout:           30 * time.Second,
		DeliveryMaxRetries:       3,
		DeliveryRetryInterval:    5 * time.Minute,
	}

	// Test all fields are set correctly
	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, "localhost", cfg.ServerHost)
	assert.Equal(t, "db.example.com", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "user", cfg.DBUser)
	assert.Equal(t, "pass", cfg.DBPassword)
	assert.Equal(t, "dbname", cfg.DBName)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, 25, cfg.DBMaxOpenConns)
	assert.Equal(t, 5, cfg.DBMaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.DBMaxLifetime)
	assert.Equal(t, "http://otel:4318", cfg.OTELExporterOTLPEndpoint)
	assert.Equal(t, "service", cfg.OTELServiceName)
	assert.Equal(t, "info", cfg.OTELLogLevel)
	assert.Equal(t, "secret", cfg.JWTSecret)
	assert.Equal(t, "token", cfg.TelegramBotToken)
	assert.Equal(t, "smtp.example.com", cfg.SMTPHost)
	assert.Equal(t, 587, cfg.SMTPPort)
	assert.Equal(t, "user", cfg.SMTPUser)
	assert.Equal(t, "pass", cfg.SMTPPassword)
	assert.Equal(t, "from@example.com", cfg.SMTPFrom)
	assert.Equal(t, 30*time.Second, cfg.WebhookTimeout)
	assert.Equal(t, 3, cfg.DeliveryMaxRetries)
	assert.Equal(t, 5*time.Minute, cfg.DeliveryRetryInterval)
}

func TestConfig_Validate_InvalidPort(t *testing.T) {
	os.Setenv("SERVER_PORT", "0")
	defer os.Unsetenv("SERVER_PORT")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SERVER_PORT")
}

func TestConfig_Validate_InvalidDBPort(t *testing.T) {
	os.Setenv("DB_PORT", "99999")
	defer os.Unsetenv("DB_PORT")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_PORT")
}

func TestConfig_Validate_EmptyDBName(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 9090, DBHost: "localhost", DBPort: 5432, DBUser: "u", JWTSecret: "s"}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_NAME")
}

func TestConfig_Validate_EmptyDBUser(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 9090, DBHost: "localhost", DBPort: 5432, DBName: "db", JWTSecret: "s"}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_USER")
}

func TestConfig_Validate_EmptyJWTSecret(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 9090, DBHost: "localhost", DBPort: 5432, DBName: "db", DBUser: "u"}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	// Сохраняем предыдущее значение если было
	prev := os.Getenv("JWT_SECRET")
	defer func() {
		if prev != "" {
			os.Setenv("JWT_SECRET", prev)
		}
	}()

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestConfig_Validate_NegativeRetries(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 9090, DBHost: "localhost", DBPort: 5432, DBName: "db", DBUser: "u", JWTSecret: "test-secret-key-that-is-at-least-32-chars!", DeliveryMaxRetries: -1}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DELIVERY_MAX_RETRIES")
}

func TestConfig_Validate_NegativeRetryInterval(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 9090, DBHost: "localhost", DBPort: 5432, DBName: "db", DBUser: "u", JWTSecret: "test-secret-key-that-is-at-least-32-chars!", DeliveryRetryInterval: -1}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DELIVERY_RETRY_INTERVAL")
}

func TestConfig_Validate_NegativeWebhookTimeout(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 9090, DBHost: "localhost", DBPort: 5432, DBName: "db", DBUser: "u", JWTSecret: "test-secret-key-that-is-at-least-32-chars!", WebhookTimeout: -1}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WEBHOOK_TIMEOUT")
}

func TestConfig_Validate_InvalidMetricsPort(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 0, DBHost: "localhost", DBPort: 5432, DBName: "db", DBUser: "u", JWTSecret: "s"}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "METRICS_PORT")
}

func TestConfig_Validate_EmptyDBHost(t *testing.T) {
	cfg := &Config{ServerPort: 8080, MetricsPort: 9090, DBPort: 5432, DBName: "db", DBUser: "u", JWTSecret: "s"}
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_HOST")
}

func TestAlertThrottlingConfigFromEnv_Defaults(t *testing.T) {
	cfg := Config{
		AlertCooldownPeriod:  15 * time.Minute,
		AlertRateLimitPeriod: 5 * time.Minute,
		AlertStormWindow:     5 * time.Minute,
		AlertStormThreshold:  5,
		AlertStormDuration:   10 * time.Minute,
	}

	result := AlertThrottlingConfigFromEnv(cfg)

	assert.Equal(t, 15*time.Minute, result.CooldownPeriod)
	assert.Equal(t, 5*time.Minute, result.RateLimitPeriod)
	assert.Equal(t, 5*time.Minute, result.StormWindow)
	assert.Equal(t, 5, result.StormThreshold)
	assert.Equal(t, 10*time.Minute, result.StormDuration)
}
