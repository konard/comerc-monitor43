package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	SchedulerAddress   string `env:"CHECK_WORKER_SCHEDULER_ADDRESS" envDefault:"localhost:9094" validate:"required"`
	MonitorAddress     string `env:"CHECK_WORKER_MONITOR_ADDRESS" envDefault:"localhost:9091" validate:"required"`
	MaintenanceAddress string `env:"CHECK_WORKER_MAINTENANCE_ADDRESS" envDefault:"localhost:9095"`

	WorkerName string `env:"CHECK_WORKER_NAME"`
	Zone       string `env:"CHECK_WORKER_ZONE" envDefault:"default" validate:"required"`

	MaxConcurrentChecks int           `env:"CHECK_WORKER_MAX_CONCURRENT_CHECKS" envDefault:"10" validate:"min=1,max=1000"`
	CheckTimeout        time.Duration `env:"CHECK_WORKER_CHECK_TIMEOUT" envDefault:"30s" validate:"required"`
	PollInterval        time.Duration `env:"CHECK_WORKER_POLL_INTERVAL" envDefault:"5s" validate:"required"`
	HeartbeatInterval   time.Duration `env:"CHECK_WORKER_HEARTBEAT_INTERVAL" envDefault:"10s" validate:"required"`

	MetricsEnabled bool `env:"CHECK_WORKER_METRICS_ENABLED" envDefault:"true"`
	MetricsPort    int  `env:"CHECK_WORKER_METRICS_PORT" envDefault:"9105" validate:"min=1,max=65535"`

	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318"`
	OTELServiceName          string `env:"OTEL_SERVICE_NAME" envDefault:"check-worker"`
	OTELLogLevel             string `env:"OTEL_LOG_LEVEL" envDefault:"info"`

	// Retry settings
	RetryMaxAttempts int           `env:"CHECK_WORKER_RETRY_MAX_ATTEMPTS" envDefault:"5" validate:"min=1"`
	RetryBaseDelay   time.Duration `env:"CHECK_WORKER_RETRY_BASE_DELAY" envDefault:"2s" validate:"required"`
	RetryMaxDelay    time.Duration `env:"CHECK_WORKER_RETRY_MAX_DELAY" envDefault:"60s" validate:"required"`

	// Queue settings
	ResultQueueMaxSize int           `env:"CHECK_WORKER_RESULT_QUEUE_MAX_SIZE" envDefault:"1000" validate:"min=1"`
	ResultQueueTTL     time.Duration `env:"CHECK_WORKER_RESULT_QUEUE_TTL" envDefault:"24h" validate:"required"`
	QueueFlushInterval time.Duration `env:"CHECK_WORKER_QUEUE_FLUSH_INTERVAL" envDefault:"30s" validate:"required"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Если имя воркера не задано через env, используем hostname контейнера для уникальности реплик
	if cfg.WorkerName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get hostname for worker name: %w", err)
		}
		cfg.WorkerName = hostname
	}

	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}
