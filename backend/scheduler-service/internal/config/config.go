package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

// Config содержит конфигурацию scheduler-service.
type Config struct {
	ServerPort     int `env:"SERVER_PORT" envDefault:"8084" validate:"required,min=1,max=65535"`
	ServerGRPCPort int `env:"SERVER_GRPC_PORT" envDefault:"9094" validate:"required,min=1,max=65535"`

	DBHost     string `env:"DB_HOST" envDefault:"localhost" validate:"required"`
	DBPort     int    `env:"DB_PORT" envDefault:"5432" validate:"required,min=1,max=65535"`
	DBName     string `env:"DB_NAME" envDefault:"monitor" validate:"required"`
	DBUser     string `env:"DB_USER" envDefault:"scheduler" validate:"required"`
	DBPassword string `env:"DB_PASSWORD" validate:"required"`
	DBSSLMode  string `env:"DB_SSL_MODE" envDefault:"disable"`

	RabbitMQHost     string `env:"RABBITMQ_HOST" envDefault:"localhost"`
	RabbitMQPort     int    `env:"RABBITMQ_PORT" envDefault:"5672" validate:"min=1,max=65535"`
	RabbitMQUser     string `env:"RABBITMQ_USER" envDefault:"guest"`
	RabbitMQPassword string `env:"RABBITMQ_PASSWORD" envDefault:"guest"`
	RabbitMQVHost    string `env:"RABBITMQ_VHOST" envDefault:"/"`
	RabbitMQEnabled  bool   `env:"RABBITMQ_ENABLED" envDefault:"true"`

	MonitorServiceGRPCAddress string `env:"MONITOR_SERVICE_GRPC_ADDRESS" envDefault:"localhost:9091" validate:"required"`

	JWTSecret string `env:"JWT_SECRET" envDefault:"change-me-to-at-least-32-characters-long"`

	MetricsEnabled bool `env:"METRICS_ENABLED" envDefault:"true"`
	MetricsPort    int  `env:"METRICS_PORT" envDefault:"9104" validate:"min=1,max=65535"`

	HeartbeatTimeout       time.Duration `env:"HEARTBEAT_TIMEOUT" envDefault:"30s" validate:"required"`
	OfflineCleanupInterval time.Duration `env:"OFFLINE_CLEANUP_INTERVAL" envDefault:"10m" validate:"required"`
	OverdueCheckThreshold  time.Duration `env:"OVERDUE_CHECK_THRESHOLD" envDefault:"2m" validate:"required"`

	SchedulingInterval time.Duration `env:"SCHEDULING_INTERVAL" envDefault:"10s" validate:"required"`

	CBFailureThreshold int           `env:"CB_FAILURE_THRESHOLD" envDefault:"5" validate:"min=1,max=100"`
	CBTimeout          time.Duration `env:"CB_TIMEOUT" envDefault:"30s" validate:"required"`
	CBSuccessThreshold int           `env:"CB_SUCCESS_THRESHOLD" envDefault:"1" validate:"min=1,max=10"`

	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318"`
	OTELServiceName          string `env:"OTEL_SERVICE_NAME" envDefault:"scheduler-service"`
	OTELLogLevel             string `env:"OTEL_LOG_LEVEL" envDefault:"info"`
}

// LoadConfig загружает и валидирует конфигурацию из переменных окружения.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// DSN возвращает PostgreSQL data source name.
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBName, c.DBUser, c.DBPassword, c.DBSSLMode)
}

// RabbitMQURL возвращает RabbitMQ connection URL.
func (c *Config) RabbitMQURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		c.RabbitMQUser, c.RabbitMQPassword, c.RabbitMQHost, c.RabbitMQPort, c.RabbitMQVHost)
}
