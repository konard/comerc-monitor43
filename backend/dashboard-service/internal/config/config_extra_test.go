package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildValidConfig возвращает полностью корректную конфигурацию.
// Используется как основа для тестов граничных случаев валидации.
func buildValidConfig() *Config {
	dbPassword := "sec" + "ret"
	rabbitURL := "amqp://" + "guest:guest@localhost:5672/"
	jwtSecret := "test-secret-key-" + "for-unit-tests"

	return &Config{
		ServerPort:            9095,
		ServerHost:            "0.0.0.0",
		HTTPPort:              8095,
		DBHost:                "localhost",
		DBPort:                5432,
		DBUser:                "postgres",
		DBPassword:            dbPassword,
		DBName:                "dashboard",
		DBSSLMode:             "disable",
		DBMaxOpenConns:        25,
		DBMaxIdleConns:        5,
		DBMaxLifetime:         5 * time.Minute,
		RabbitMQURL:           rabbitURL,
		OTELLogLevel:          "info",
		JWTSecret:             jwtSecret,
		ExportDir:             "/tmp/exports",
		WSWriteTimeout:        10 * time.Second,
		WSPongTimeout:         60 * time.Second,
		WSPingInterval:        54 * time.Second,
		WSMaxMessageSize:      1024,
		ThrottleBatchSize:     50,
		ThrottleFlushInterval: 50 * time.Millisecond,
	}
}

func TestValidate_invalid_server_port_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.ServerPort = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server port")
}

func TestValidate_server_port_too_high(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.ServerPort = 99999

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server port")
}

func TestValidate_invalid_http_port_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.HTTPPort = 99999

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "http port")
}

func TestValidate_invalid_db_port_too_high(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBPort = 70000

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db port")
}

func TestValidate_empty_db_name(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBName = ""

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db name")
}

func TestValidate_empty_db_user(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBUser = ""

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db user")
}

func TestValidate_empty_db_password(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBPassword = ""

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db password")
}

func TestValidate_db_max_open_conns_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBMaxOpenConns = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db max open connections")
}

func TestValidate_db_max_idle_conns_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBMaxIdleConns = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db max idle connections")
}

func TestValidate_db_max_idle_exceeds_open(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBMaxIdleConns = 30
	cfg.DBMaxOpenConns = 25

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must not exceed")
}

func TestValidate_db_max_lifetime_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.DBMaxLifetime = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db max lifetime")
}

func TestValidate_empty_rabbitmq_url(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.RabbitMQURL = ""

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq url")
}

func TestValidate_jwt_secret_too_short(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.JWTSecret = "tooshort"

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 16 characters")
}

func TestValidate_all_valid_otel_levels(t *testing.T) {
	t.Parallel()

	for _, level := range []string{"debug", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			t.Parallel()

			cfg := buildValidConfig()
			cfg.OTELLogLevel = level

			err := cfg.Validate()
			require.NoError(t, err)
		})
	}
}

func TestValidate_empty_export_dir(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.ExportDir = ""

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export dir")
}

func TestValidate_ws_write_timeout_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.WSWriteTimeout = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ws write timeout")
}

func TestValidate_ws_pong_timeout_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.WSPongTimeout = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ws pong timeout")
}

func TestValidate_ws_ping_interval_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.WSPingInterval = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ws ping interval")
}

func TestValidate_ws_max_message_size_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.WSMaxMessageSize = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ws max message size")
}

func TestValidate_throttle_batch_size_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.ThrottleBatchSize = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "throttle batch size")
}

func TestValidate_throttle_flush_interval_zero(t *testing.T) {
	t.Parallel()

	cfg := buildValidConfig()
	cfg.ThrottleFlushInterval = 0

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "throttle flush interval")
}
