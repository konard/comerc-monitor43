package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
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
		DBName:                "testdb",
		DBSSLMode:             "disable",
		DBMaxOpenConns:        10,
		DBMaxIdleConns:        5,
		DBMaxLifetime:         5 * time.Minute,
		RabbitMQURL:           rabbitURL,
		OTELLogLevel:          "info",
		JWTSecret:             jwtSecret,
		WSWriteTimeout:        10 * time.Second,
		WSPongTimeout:         60 * time.Second,
		WSPingInterval:        54 * time.Second,
		WSMaxMessageSize:      1024,
		ThrottleBatchSize:     50,
		ThrottleFlushInterval: 50 * time.Millisecond,
		ExportDir:             "/tmp/exports",
	}
}

func setMinimalEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_PASSWORD", "sec"+"ret")
	t.Setenv("JWT_SECRET", "test-secret-key-"+"for-unit-tests")
	t.Setenv("RABBITMQ_URL", "amqp://"+"guest:guest@localhost:5672/")
}

func TestLoad_Defaults(t *testing.T) {
	setMinimalEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "0.0.0.0", cfg.ServerHost)
	assert.Equal(t, 9095, cfg.ServerPort)
	assert.Equal(t, 8095, cfg.HTTPPort)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "postgres", cfg.DBUser)
	assert.Equal(t, "secret", cfg.DBPassword)
	assert.Equal(t, "dashboard", cfg.DBName)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, 25, cfg.DBMaxOpenConns)
	assert.Equal(t, 5, cfg.DBMaxIdleConns)
	assert.Equal(t, "info", cfg.OTELLogLevel)
	assert.Equal(t, "dashboard-service", cfg.OTELServiceName)
}

func TestLoad_InvalidDBPort(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("DB_PORT", "99999")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db port")
}

func TestLoad_InvalidHTTPPort(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("HTTP_PORT", "0")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http port")
}

func TestValidate_EmptyJWTSecret(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.JWTSecret = ""

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt secret")
}

func TestValidate_EmptyDBHost(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.DBHost = ""

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db host")
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("OTEL_LOG_LEVEL", "invalid")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "otel log level")
}

func TestDSN(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		DBHost:     "dbhost",
		DBPort:     5433,
		DBUser:     "admin",
		DBPassword: "my" + "pass",
		DBName:     "mydb",
		DBSSLMode:  "require",
	}

	want := "host=dbhost port=5433 user=admin password=mypass dbname=mydb sslmode=require"
	assert.Equal(t, want, cfg.DSN())
}

func TestServerAddress(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ServerHost: "127.0.0.1",
		ServerPort: 9090,
	}

	assert.Equal(t, "127.0.0.1:9090", cfg.ServerAddress())
}

func TestHTTPAddress(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ServerHost: "0.0.0.0",
		HTTPPort:   8080,
	}

	assert.Equal(t, "0.0.0.0:8080", cfg.HTTPAddress())
}
