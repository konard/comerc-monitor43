package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// Config содержит полную конфигурацию reporting-service.
type Config struct {
	LogLevel       string `env:"LOG_LEVEL" envDefault:"info" validate:"required,oneof=debug info warn error"`
	Server         ServerConfig
	Database       DatabaseConfig
	Tracing        TracingConfig
	Metrics        MetricsConfig
	MonitorService MonitorServiceConfig
	Auth           AuthConfig
	Reporting      ReportingConfig
}

// ServerConfig содержит настройки HTTP/gRPC сервера.
type ServerConfig struct {
	Port     int           `env:"SERVER_PORT" envDefault:"8084" validate:"required,min=1,max=65535"`
	GRPCPort int           `env:"SERVER_GRPC_PORT" envDefault:"5007" validate:"required,min=1,max=65535"`
	Timeout  time.Duration `env:"SERVER_TIMEOUT" envDefault:"30s" validate:"required"`
}

// DatabaseConfig содержит параметры подключения к PostgreSQL.
type DatabaseConfig struct {
	Host            string        `env:"DB_HOST" envDefault:"localhost" validate:"required"`
	Port            int           `env:"DB_PORT" envDefault:"5432" validate:"required,min=1,max=65535"`
	Name            string        `env:"DB_NAME" envDefault:"monitor" validate:"required"`
	User            string        `env:"DB_USER" envDefault:"monitor" validate:"required"`
	Password        string        `env:"DB_PASSWORD" envDefault:"monitor" validate:"required"`
	SSLMode         string        `env:"DB_SSL_MODE" envDefault:"disable" validate:"required,oneof=disable require allow prefer verify-ca verify-full"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25" validate:"min=1"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5" validate:"min=1"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m" validate:"required"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"30s" validate:"required"`
}

// TracingConfig содержит настройки распределённой трассировки.
type TracingConfig struct {
	Enabled     bool    `env:"OTEL_ENABLED" envDefault:"true"`
	Endpoint    string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318" validate:"url"`
	ServiceName string  `env:"OTEL_SERVICE_NAME" envDefault:"reporting-service" validate:"required"`
	Environment string  `env:"OTEL_ENVIRONMENT" envDefault:"development" validate:"required,oneof=development staging production"`
	SampleRate  float64 `env:"OTEL_SAMPLING_RATE" envDefault:"1.0" validate:"min=0,max=1"`
}

// MetricsConfig содержит настройки сбора метрик.
type MetricsConfig struct {
	Enabled bool   `env:"METRICS_ENABLED" envDefault:"true"`
	Port    int    `env:"METRICS_PORT" envDefault:"9097" validate:"min=1,max=65535"`
	Path    string `env:"METRICS_PATH" envDefault:"/metrics" validate:"required"`
}

// MonitorServiceConfig содержит адрес gRPC monitor-service.
type MonitorServiceConfig struct {
	GRPCAddress string `env:"MONITOR_SERVICE_GRPC_ADDRESS" envDefault:"localhost:5002" validate:"required,hostname_port"`
}

// AuthConfig содержит параметры аутентификации.
type AuthConfig struct {
	SecretKey string `env:"JWT_SECRET_KEY" envDefault:"secret-key" validate:"required,min=8"`
}

// ReportingConfig содержит настройки генерации отчётов.
type ReportingConfig struct {
	RetentionDays int           `env:"REPORTING_RETENTION_DAYS" envDefault:"90" validate:"min=1,max=365"`
	ExportTimeout time.Duration `env:"REPORTING_EXPORT_TIMEOUT" envDefault:"60s" validate:"required"`
	MaxExportRows int           `env:"REPORTING_MAX_EXPORT_ROWS" envDefault:"50000" validate:"min=1"`
}

// Load загружает и валидирует конфигурацию из переменных окружения.
func Load() (Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "failed to parse config from environment")
	}

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "invalid config")
	}

	return cfg, nil
}
