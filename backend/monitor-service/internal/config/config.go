package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

// Config содержит всю конфигурацию для monitor service.
type Config struct {
	// Server
	ServerPort     int `env:"SERVER_PORT" envDefault:"8081" validate:"required,min=1,max=65535"`
	ServerGRPCPort int `env:"SERVER_GRPC_PORT" envDefault:"5002" validate:"required,min=1,max=65535"`

	// Database
	DBHost     string `env:"DB_HOST" envDefault:"localhost" validate:"required"`
	DBPort     int    `env:"DB_PORT" envDefault:"5432" validate:"required,min=1,max=65535"`
	DBName     string `env:"DB_NAME" envDefault:"monitor" validate:"required"`
	DBUser     string `env:"DB_USER" envDefault:"monitor" validate:"required"`
	DBPassword string `env:"DB_PASSWORD" validate:"required,min=8"`
	DBSSLMode  string `env:"DB_SSL_MODE" envDefault:"disable"`

	// RabbitMQ
	RabbitMQHost     string `env:"RABBITMQ_HOST" envDefault:"localhost"`
	RabbitMQPort     int    `env:"RABBITMQ_PORT" envDefault:"5672" validate:"min=1,max=65535"`
	RabbitMQUser     string `env:"RABBITMQ_USER" envDefault:"guest"`
	RabbitMQPassword string `env:"RABBITMQ_PASSWORD" envDefault:"guest"`
	RabbitMQVHost    string `env:"RABBITMQ_VHOST" envDefault:"/"`
	RabbitMQEnabled  bool   `env:"RABBITMQ_ENABLED" envDefault:"true"`

	// Metrics
	MetricsEnabled bool `env:"METRICS_ENABLED" envDefault:"true"`
	MetricsPort    int  `env:"METRICS_PORT" envDefault:"9091" validate:"min=1,max=65535"`

	// Auth Service Integration
	AuthServiceGRPCAddress string `env:"AUTH_SERVICE_GRPC_ADDRESS" envDefault:"localhost:5001" validate:"required"`

	// JWT секрет для валидации токенов, выдаваемых auth-service
	JWTSecret string `env:"JWT_SECRET" envDefault:"change-me-to-at-least-32-characters-long"`

	// Check Configuration
	DefaultCheckInterval     time.Duration `env:"DEFAULT_CHECK_INTERVAL" envDefault:"5m" validate:"required"`
	DefaultCheckTimeout      time.Duration `env:"DEFAULT_CHECK_TIMEOUT" envDefault:"30s" validate:"required"`
	MaxConcurrentChecks      int           `env:"MAX_CONCURRENT_CHECKS" envDefault:"100" validate:"min=1,max=1000"`
	CheckQueueSize           int           `env:"CHECK_QUEUE_SIZE" envDefault:"1000" validate:"min=1"`
	MaxRecentResultsForStats int           `env:"MAX_RECENT_RESULTS_FOR_STATS" envDefault:"100" validate:"min=1"`

	// Degraded Threshold Defaults
	DefaultDegradedResponseTime int `env:"DEFAULT_DEGRADED_RESPONSE_TIME" envDefault:"1000" validate:"min=1"`      // milliseconds
	DefaultDegradedFailureRate  int `env:"DEFAULT_DEGRADED_FAILURE_RATE" envDefault:"50" validate:"min=1,max=100"` // percentage

	// Incident Detection
	IncidentDetectionEnabled bool `env:"INCIDENT_DETECTION_ENABLED" envDefault:"true"`
	IncidentResolveThreshold int  `env:"INCIDENT_RESOLVE_THRESHOLD" envDefault:"1" validate:"min=1"` // consecutive UPs to resolve

	// Data Retention
	CheckResultRetentionDays int `env:"CHECK_RESULT_RETENTION_DAYS" envDefault:"90" validate:"min=1"`
	IncidentRetentionDays    int `env:"INCIDENT_RETENTION_DAYS" envDefault:"365" validate:"min=1"`
	AuditLogRetentionDays    int `env:"AUDIT_LOG_RETENTION_DAYS" envDefault:"365" validate:"min=1"`

	// Maintenance Window Configuration
	MaintenanceMinDurationMinutes   int `env:"MAINTENANCE_MIN_DURATION_MINUTES" envDefault:"1" validate:"min=1"`
	MaintenanceMaxDurationHours     int `env:"MAINTENANCE_MAX_DURATION_HOURS" envDefault:"24" validate:"min=1,max=168"` // Max 7 days
	MaintenanceMaxWindowsFreeTier   int `env:"MAINTENANCE_MAX_WINDOWS_FREE" envDefault:"5" validate:"min=1"`
	MaintenanceMaxWindowsProTier    int `env:"MAINTENANCE_MAX_WINDOWS_PRO" envDefault:"20" validate:"min=1"`
	MaintenanceMaxWindowsEnterprise int `env:"MAINTENANCE_MAX_WINDOWS_ENTERPRISE" envDefault:"100" validate:"min=1"`
	MaintenanceMaxMonitorsPerWindow int `env:"MAINTENANCE_MAX_MONITORS_PER_WINDOW" envDefault:"50" validate:"min=1,max=500"`

	// Observability
	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318"`
	OTELServiceName          string `env:"OTEL_SERVICE_NAME" envDefault:"monitor-service"`
	OTELLogLevel             string `env:"OTEL_LOG_LEVEL" envDefault:"info"`

	// Rate Limiting
	RateLimitEnabled           bool `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
	RateLimitRequestsPerMinute int  `env:"RATE_LIMIT_REQUESTS_PER_MINUTE" envDefault:"1000" validate:"min=1"`
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

// RabbitMQQueueName возвращает имя очереди для задач проверки.
func (c *Config) RabbitMQQueueName() string {
	return "monitor.checks"
}

// RabbitMQExchangeName возвращает имя exchange для событий мониторинга.
func (c *Config) RabbitMQExchangeName() string {
	return "monitor.events"
}
