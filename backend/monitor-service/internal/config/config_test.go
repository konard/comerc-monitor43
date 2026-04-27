package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig_Success тестирует успешную загрузку конфигурации с defaults.
func TestLoadConfig_Success(t *testing.T) {
	// Устанавливаем минимальные необходимые env vars
	os.Setenv("DB_PASSWORD", "test_password_123")
	defer os.Unsetenv("DB_PASSWORD")

	cfg, err := LoadConfig()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Проверяем значения по умолчанию
	assert.Equal(t, 8081, cfg.ServerPort)
	assert.Equal(t, 5002, cfg.ServerGRPCPort)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "monitor", cfg.DBName)
	assert.Equal(t, "monitor", cfg.DBUser)
	assert.Equal(t, "test_password_123", cfg.DBPassword)
	assert.Equal(t, "disable", cfg.DBSSLMode)

	// RabbitMQ defaults
	assert.Equal(t, "localhost", cfg.RabbitMQHost)
	assert.Equal(t, 5672, cfg.RabbitMQPort)
	assert.Equal(t, "guest", cfg.RabbitMQUser)
	assert.Equal(t, "guest", cfg.RabbitMQPassword)
	assert.Equal(t, "/", cfg.RabbitMQVHost)
	assert.True(t, cfg.RabbitMQEnabled)

	// Metrics defaults
	assert.True(t, cfg.MetricsEnabled)
	assert.Equal(t, 9091, cfg.MetricsPort)

	// Auth Service
	assert.Equal(t, "localhost:5001", cfg.AuthServiceGRPCAddress)

	// Check Configuration defaults
	assert.Equal(t, 5*time.Minute, cfg.DefaultCheckInterval)
	assert.Equal(t, 30*time.Second, cfg.DefaultCheckTimeout)
	assert.Equal(t, 100, cfg.MaxConcurrentChecks)
	assert.Equal(t, 1000, cfg.CheckQueueSize)
	assert.Equal(t, 100, cfg.MaxRecentResultsForStats)

	// Degraded Threshold defaults
	assert.Equal(t, 1000, cfg.DefaultDegradedResponseTime)
	assert.Equal(t, 50, cfg.DefaultDegradedFailureRate)

	// Incident Detection defaults
	assert.True(t, cfg.IncidentDetectionEnabled)
	assert.Equal(t, 1, cfg.IncidentResolveThreshold)

	// Data Retention defaults
	assert.Equal(t, 90, cfg.CheckResultRetentionDays)
	assert.Equal(t, 365, cfg.IncidentRetentionDays)
	assert.Equal(t, 365, cfg.AuditLogRetentionDays)

	// Observability defaults
	assert.Equal(t, "http://localhost:4318", cfg.OTELExporterOTLPEndpoint)
	assert.Equal(t, "monitor-service", cfg.OTELServiceName)
	assert.Equal(t, "info", cfg.OTELLogLevel)
}

// TestLoadConfig_WithCustomEnvVars тестирует загрузку с кастомными env vars.
func TestLoadConfig_WithCustomEnvVars(t *testing.T) {
	// Устанавливаем кастомные значения
	envVars := map[string]string{
		"SERVER_PORT":                    "9090",
		"SERVER_GRPC_PORT":               "6000",
		"DB_HOST":                        "db.example.com",
		"DB_PORT":                        "5433",
		"DB_NAME":                        "monitor_prod",
		"DB_USER":                        "monitor_user",
		"DB_PASSWORD":                    "secure_password_123",
		"DB_SSL_MODE":                    "require",
		"RABBITMQ_HOST":                  "rabbitmq.example.com",
		"RABBITMQ_PORT":                  "5671",
		"RABBITMQ_USER":                  "rmq_user",
		"RABBITMQ_PASSWORD":              "rmq_pass",
		"RABBITMQ_VHOST":                 "/monitor",
		"RABBITMQ_ENABLED":               "false",
		"METRICS_ENABLED":                "false",
		"METRICS_PORT":                   "9092",
		"AUTH_SERVICE_GRPC_ADDRESS":      "auth-service:5001",
		"DEFAULT_CHECK_INTERVAL":         "10m",
		"DEFAULT_CHECK_TIMEOUT":          "60s",
		"MAX_CONCURRENT_CHECKS":          "200",
		"CHECK_QUEUE_SIZE":               "2000",
		"MAX_RECENT_RESULTS_FOR_STATS":   "200",
		"DEFAULT_DEGRADED_RESPONSE_TIME": "2000",
		"DEFAULT_DEGRADED_FAILURE_RATE":  "75",
		"INCIDENT_DETECTION_ENABLED":     "false",
		"INCIDENT_RESOLVE_THRESHOLD":     "3",
		"CHECK_RESULT_RETENTION_DAYS":    "180",
		"INCIDENT_RETENTION_DAYS":        "730",
		"AUDIT_LOG_RETENTION_DAYS":       "730",
		"OTEL_EXPORTER_OTLP_ENDPOINT":    "http://otel-collector:4318",
		"OTEL_SERVICE_NAME":              "monitor-service-prod",
		"OTEL_LOG_LEVEL":                 "debug",
	}

	// Устанавливаем все env vars
	for k, v := range envVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k) //nolint:revive // defer внутри цикла намеренно: cleanup нужен после завершения теста
	}

	cfg, err := LoadConfig()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Проверяем все кастомные значения
	assert.Equal(t, 9090, cfg.ServerPort)
	assert.Equal(t, 6000, cfg.ServerGRPCPort)
	assert.Equal(t, "db.example.com", cfg.DBHost)
	assert.Equal(t, 5433, cfg.DBPort)
	assert.Equal(t, "monitor_prod", cfg.DBName)
	assert.Equal(t, "monitor_user", cfg.DBUser)
	assert.Equal(t, "secure_password_123", cfg.DBPassword)
	assert.Equal(t, "require", cfg.DBSSLMode)

	assert.Equal(t, "rabbitmq.example.com", cfg.RabbitMQHost)
	assert.Equal(t, 5671, cfg.RabbitMQPort)
	assert.Equal(t, "rmq_user", cfg.RabbitMQUser)
	assert.Equal(t, "rmq_pass", cfg.RabbitMQPassword)
	assert.Equal(t, "/monitor", cfg.RabbitMQVHost)
	assert.False(t, cfg.RabbitMQEnabled)

	assert.False(t, cfg.MetricsEnabled)
	assert.Equal(t, 9092, cfg.MetricsPort)

	assert.Equal(t, "auth-service:5001", cfg.AuthServiceGRPCAddress)

	assert.Equal(t, 10*time.Minute, cfg.DefaultCheckInterval)
	assert.Equal(t, 60*time.Second, cfg.DefaultCheckTimeout)
	assert.Equal(t, 200, cfg.MaxConcurrentChecks)
	assert.Equal(t, 2000, cfg.CheckQueueSize)
	assert.Equal(t, 200, cfg.MaxRecentResultsForStats)

	assert.Equal(t, 2000, cfg.DefaultDegradedResponseTime)
	assert.Equal(t, 75, cfg.DefaultDegradedFailureRate)

	assert.False(t, cfg.IncidentDetectionEnabled)
	assert.Equal(t, 3, cfg.IncidentResolveThreshold)

	assert.Equal(t, 180, cfg.CheckResultRetentionDays)
	assert.Equal(t, 730, cfg.IncidentRetentionDays)
	assert.Equal(t, 730, cfg.AuditLogRetentionDays)

	assert.Equal(t, "http://otel-collector:4318", cfg.OTELExporterOTLPEndpoint)
	assert.Equal(t, "monitor-service-prod", cfg.OTELServiceName)
	assert.Equal(t, "debug", cfg.OTELLogLevel)
}

// TestLoadConfig_ValidationErrors тестирует ошибки валидации.
func TestLoadConfig_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name    string
		envVars map[string]string
		errMsg  string
	}{
		{
			name:    "missing DB_PASSWORD",
			envVars: map[string]string{},
			errMsg:  "invalid config",
		},
		{
			name: "DB_PASSWORD too short",
			envVars: map[string]string{
				"DB_PASSWORD": "short",
			},
			errMsg: "invalid config",
		},
		{
			name: "invalid SERVER_PORT",
			envVars: map[string]string{
				"DB_PASSWORD": "test_password_123",
				"SERVER_PORT": "70000",
			},
			errMsg: "invalid config",
		},
		{
			name: "invalid DB_PORT",
			envVars: map[string]string{
				"DB_PASSWORD": "test_password_123",
				"DB_PORT":     "0",
			},
			errMsg: "invalid config",
		},
		{
			name: "invalid MAX_CONCURRENT_CHECKS",
			envVars: map[string]string{
				"DB_PASSWORD":           "test_password_123",
				"MAX_CONCURRENT_CHECKS": "0",
			},
			errMsg: "invalid config",
		},
		{
			name: "invalid DEFAULT_DEGRADED_FAILURE_RATE",
			envVars: map[string]string{
				"DB_PASSWORD":                   "test_password_123",
				"DEFAULT_DEGRADED_FAILURE_RATE": "101",
			},
			errMsg: "invalid config",
		},
		{
			name: "invalid CHECK_RESULT_RETENTION_DAYS",
			envVars: map[string]string{
				"DB_PASSWORD":                 "test_password_123",
				"CHECK_RESULT_RETENTION_DAYS": "0",
			},
			errMsg: "invalid config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Очищаем все env vars
			for k := range tc.envVars {
				defer os.Unsetenv(k)
			}

			cfg, err := LoadConfig()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
			assert.Nil(t, cfg)
		})
	}
}

// TestLoadConfig_InvalidDuration тестирует обработку невалидных duration значений.
func TestLoadConfig_InvalidDuration(t *testing.T) {
	os.Setenv("DB_PASSWORD", "test_password_123")
	os.Setenv("DEFAULT_CHECK_INTERVAL", "invalid")
	defer os.Unsetenv("DB_PASSWORD")
	defer os.Unsetenv("DEFAULT_CHECK_INTERVAL")

	cfg, err := LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
	assert.Nil(t, cfg)
}

// TestConfig_DSN тестирует генерацию DSN.
func TestConfig_DSN(t *testing.T) {
	testCases := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "default config",
			config: &Config{
				DBHost:     "localhost",
				DBPort:     5432,
				DBName:     "monitor",
				DBUser:     "monitor",
				DBPassword: "password",
				DBSSLMode:  "disable",
			},
			expected: "host=localhost port=5432 dbname=monitor user=monitor password=password sslmode=disable",
		},
		{
			name: "with SSL",
			config: &Config{
				DBHost:     "db.example.com",
				DBPort:     5433,
				DBName:     "monitor_prod",
				DBUser:     "monitor_user",
				DBPassword: "secure_pass",
				DBSSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 dbname=monitor_prod user=monitor_user password=secure_pass sslmode=require",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dsn := tc.config.DSN()
			assert.Equal(t, tc.expected, dsn)
		})
	}
}

// TestConfig_RabbitMQURL тестирует генерацию RabbitMQ URL.
func TestConfig_RabbitMQURL(t *testing.T) {
	testCases := []struct {
		name     string
		config   *Config
		expected string
	}{
		{ //nolint:gosec // G101: тестовые учётные данные RabbitMQ по умолчанию
			name: "default config",
			config: &Config{
				RabbitMQUser:     "guest",
				RabbitMQPassword: "guest",
				RabbitMQHost:     "localhost",
				RabbitMQPort:     5672,
				RabbitMQVHost:    "/",
			},
			expected: "amqp://guest:guest@localhost:5672/",
		},
		{ //nolint:gosec // G101: тестовые учётные данные RabbitMQ
			name: "custom config",
			config: &Config{
				RabbitMQUser:     "rmq_user",
				RabbitMQPassword: "rmq_pass",
				RabbitMQHost:     "rabbitmq.example.com",
				RabbitMQPort:     5671,
				RabbitMQVHost:    "/monitor",
			},
			expected: "amqp://rmq_user:rmq_pass@rabbitmq.example.com:5671/monitor",
		},
		{
			name: "with special characters in password",
			config: &Config{
				RabbitMQUser:     "user",
				RabbitMQPassword: "p@ssw0rd!",
				RabbitMQHost:     "localhost",
				RabbitMQPort:     5672,
				RabbitMQVHost:    "/",
			},
			expected: "amqp://user:p@ssw0rd!@localhost:5672/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := tc.config.RabbitMQURL()
			assert.Equal(t, tc.expected, url)
		})
	}
}

// TestConfig_RabbitMQQueueName тестирует имя очереди.
func TestConfig_RabbitMQQueueName(t *testing.T) {
	cfg := &Config{}

	queueName := cfg.RabbitMQQueueName()

	assert.Equal(t, "monitor.checks", queueName)
}

// TestConfig_RabbitMQExchangeName тестирует имя exchange.
func TestConfig_RabbitMQExchangeName(t *testing.T) {
	cfg := &Config{}

	exchangeName := cfg.RabbitMQExchangeName()

	assert.Equal(t, "monitor.events", exchangeName)
}

// TestConfig_BoolParsing тестирует парсинг булевых значений.
func TestConfig_BoolParsing(t *testing.T) {
	testCases := []struct {
		name          string
		envVarValue   string
		envVarName    string
		expectedValue bool
	}{
		{
			name:          "RABBITMQ_ENABLED true",
			envVarValue:   "true",
			envVarName:    "RABBITMQ_ENABLED",
			expectedValue: true,
		},
		{
			name:          "RABBITMQ_ENABLED True",
			envVarValue:   "True",
			envVarName:    "RABBITMQ_ENABLED",
			expectedValue: true,
		},
		{
			name:          "RABBITMQ_ENABLED 1",
			envVarValue:   "1",
			envVarName:    "RABBITMQ_ENABLED",
			expectedValue: true,
		},
		{
			name:          "RABBITMQ_ENABLED false",
			envVarValue:   "false",
			envVarName:    "RABBITMQ_ENABLED",
			expectedValue: false,
		},
		{
			name:          "RABBITMQ_ENABLED False",
			envVarValue:   "False",
			envVarName:    "RABBITMQ_ENABLED",
			expectedValue: false,
		},
		{
			name:          "RABBITMQ_ENABLED 0",
			envVarValue:   "0",
			envVarName:    "RABBITMQ_ENABLED",
			expectedValue: false,
		},
		{
			name:          "METRICS_ENABLED true",
			envVarValue:   "true",
			envVarName:    "METRICS_ENABLED",
			expectedValue: true,
		},
		{
			name:          "INCIDENT_DETECTION_ENABLED false",
			envVarValue:   "false",
			envVarName:    "INCIDENT_DETECTION_ENABLED",
			expectedValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("DB_PASSWORD", "test_password_123")
			os.Setenv(tc.envVarName, tc.envVarValue)
			defer os.Unsetenv("DB_PASSWORD")
			defer os.Unsetenv(tc.envVarName)

			cfg, err := LoadConfig()

			require.NoError(t, err)
			require.NotNil(t, cfg)

			switch tc.envVarName {
			case "RABBITMQ_ENABLED":
				assert.Equal(t, tc.expectedValue, cfg.RabbitMQEnabled)
			case "METRICS_ENABLED":
				assert.Equal(t, tc.expectedValue, cfg.MetricsEnabled)
			case "INCIDENT_DETECTION_ENABLED":
				assert.Equal(t, tc.expectedValue, cfg.IncidentDetectionEnabled)
			}
		})
	}
}

// TestConfig_DurationParsing тестирует парсинг duration значений.
func TestConfig_DurationParsing(t *testing.T) {
	testCases := []struct {
		name          string
		envVarValue   string
		envVarName    string
		expectedValue time.Duration
	}{
		{
			name:          "DEFAULT_CHECK_INTERVAL seconds",
			envVarValue:   "30s",
			envVarName:    "DEFAULT_CHECK_INTERVAL",
			expectedValue: 30 * time.Second,
		},
		{
			name:          "DEFAULT_CHECK_INTERVAL minutes",
			envVarValue:   "15m",
			envVarName:    "DEFAULT_CHECK_INTERVAL",
			expectedValue: 15 * time.Minute,
		},
		{
			name:          "DEFAULT_CHECK_TIMEOUT",
			envVarValue:   "45s",
			envVarName:    "DEFAULT_CHECK_TIMEOUT",
			expectedValue: 45 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("DB_PASSWORD", "test_password_123")
			os.Setenv(tc.envVarName, tc.envVarValue)
			defer os.Unsetenv("DB_PASSWORD")
			defer os.Unsetenv(tc.envVarName)

			cfg, err := LoadConfig()

			require.NoError(t, err)
			require.NotNil(t, cfg)

			switch tc.envVarName {
			case "DEFAULT_CHECK_INTERVAL":
				assert.Equal(t, tc.expectedValue, cfg.DefaultCheckInterval)
			case "DEFAULT_CHECK_TIMEOUT":
				assert.Equal(t, tc.expectedValue, cfg.DefaultCheckTimeout)
			}
		})
	}
}
