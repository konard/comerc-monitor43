package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type Config struct {
	ServerPort int    `env:"SERVER_PORT" envDefault:"9095"`
	ServerHost string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	HTTPPort   int    `env:"HTTP_PORT" envDefault:"8095"`

	DBHost         string        `env:"DB_HOST" envDefault:"localhost"`
	DBPort         int           `env:"DB_PORT" envDefault:"5432"`
	DBUser         string        `env:"DB_USER" envDefault:"postgres"`
	DBPassword     string        `env:"DB_PASSWORD" envDefault:"postgres"`
	DBName         string        `env:"DB_NAME" envDefault:"dashboard"`
	DBSSLMode      string        `env:"DB_SSL_MODE" envDefault:"disable"`
	DBMaxOpenConns int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	DBMaxIdleConns int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBMaxLifetime  time.Duration `env:"DB_MAX_LIFETIME" envDefault:"5m"`

	RabbitMQURL string `env:"RABBITMQ_URL" envDefault:"amqp://guest:guest@localhost:5672/"`

	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318"`
	OTELServiceName          string `env:"OTEL_SERVICE_NAME" envDefault:"dashboard-service"`
	OTELLogLevel             string `env:"OTEL_LOG_LEVEL" envDefault:"info"`

	JWTSecret string `env:"JWT_SECRET" envDefault:"test-secret-key-change-in-production"`

	WSMaxConnections int           `env:"WS_MAX_CONNECTIONS" envDefault:"1000"`
	WSWriteTimeout   time.Duration `env:"WS_WRITE_TIMEOUT" envDefault:"10s"`
	WSPongTimeout    time.Duration `env:"WS_PONG_TIMEOUT" envDefault:"60s"`
	WSPingInterval   time.Duration `env:"WS_PING_INTERVAL" envDefault:"54s"`
	WSMaxMessageSize int64         `env:"WS_MAX_MESSAGE_SIZE" envDefault:"1024"`

	ThrottleBatchSize     int           `env:"THROTTLE_BATCH_SIZE" envDefault:"50"`
	ThrottleFlushInterval time.Duration `env:"THROTTLE_FLUSH_INTERVAL" envDefault:"50ms"`

	ExportDir string `env:"EXPORT_DIR" envDefault:"/tmp/dashboard-exports"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, errors.Wrap(err, "failed to parse config from environment")
	}
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", c.ServerPort)
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("http port must be between 1 and 65535, got %d", c.HTTPPort)
	}
	if c.DBPort < 1 || c.DBPort > 65535 {
		return fmt.Errorf("db port must be between 1 and 65535, got %d", c.DBPort)
	}
	if c.DBHost == "" {
		return fmt.Errorf("db host must not be empty")
	}
	if c.DBName == "" {
		return fmt.Errorf("db name must not be empty")
	}
	if c.DBUser == "" {
		return fmt.Errorf("db user must not be empty")
	}
	if c.DBPassword == "" {
		return fmt.Errorf("db password must not be empty")
	}
	if c.DBMaxOpenConns <= 0 {
		return fmt.Errorf("db max open connections must be > 0, got %d", c.DBMaxOpenConns)
	}
	if c.DBMaxIdleConns <= 0 {
		return fmt.Errorf("db max idle connections must be > 0, got %d", c.DBMaxIdleConns)
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns {
		return fmt.Errorf("db max idle connections (%d) must not exceed max open connections (%d)", c.DBMaxIdleConns, c.DBMaxOpenConns)
	}
	if c.DBMaxLifetime <= 0 {
		return fmt.Errorf("db max lifetime must be > 0, got %v", c.DBMaxLifetime)
	}
	if c.RabbitMQURL == "" {
		return fmt.Errorf("rabbitmq url must not be empty")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("jwt secret must not be empty")
	}
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("jwt secret must be at least 16 characters")
	}
	switch c.OTELLogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("otel log level must be one of: debug, info, warn, error, got %q", c.OTELLogLevel)
	}
	if c.ExportDir == "" {
		return fmt.Errorf("export dir must not be empty")
	}
	if c.WSWriteTimeout <= 0 {
		return fmt.Errorf("ws write timeout must be > 0, got %v", c.WSWriteTimeout)
	}
	if c.WSPongTimeout <= 0 {
		return fmt.Errorf("ws pong timeout must be > 0, got %v", c.WSPongTimeout)
	}
	if c.WSPingInterval <= 0 {
		return fmt.Errorf("ws ping interval must be > 0, got %v", c.WSPingInterval)
	}
	if c.WSMaxMessageSize <= 0 {
		return fmt.Errorf("ws max message size must be > 0, got %d", c.WSMaxMessageSize)
	}
	if c.ThrottleBatchSize <= 0 {
		return fmt.Errorf("throttle batch size must be > 0, got %d", c.ThrottleBatchSize)
	}
	if c.ThrottleFlushInterval <= 0 {
		return fmt.Errorf("throttle flush interval must be > 0, got %v", c.ThrottleFlushInterval)
	}
	return nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

func (c *Config) HTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.HTTPPort)
}
