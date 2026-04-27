package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config содержит конфигурацию alert-service
type Config struct {
	// Server
	ServerPort  int    `env:"SERVER_PORT" envDefault:"50051"`
	ServerHost  string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	MetricsPort int    `env:"METRICS_PORT" envDefault:"9090"`

	// Database
	DBHost         string        `env:"DB_HOST" envDefault:"localhost"`
	DBPort         int           `env:"DB_PORT" envDefault:"5432"`
	DBUser         string        `env:"DB_USER" envDefault:"postgres"`
	DBPassword     string        `env:"DB_PASSWORD" envDefault:"postgres"`
	DBName         string        `env:"DB_NAME" envDefault:"milan"`
	DBSSLMode      string        `env:"DB_SSL_MODE" envDefault:"disable"`
	DBMaxOpenConns int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	DBMaxIdleConns int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBMaxLifetime  time.Duration `env:"DB_MAX_LIFETIME" envDefault:"5m"`

	// Observability
	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318"`
	OTELServiceName          string `env:"OTEL_SERVICE_NAME" envDefault:"alert-service"`
	OTELLogLevel             string `env:"OTEL_LOG_LEVEL" envDefault:"info"`

	// Authentication
	JWTSecret string `env:"JWT_SECRET"`

	// Telegram (для каналов уведомлений)
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`

	// Email SMTP
	SMTPHost     string `env:"SMTP_HOST" envDefault:"smtp.gmail.com"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser     string `env:"SMTP_USER"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM" envDefault:"noreply@monitor.example.com"`

	// Webhook
	WebhookTimeout time.Duration `env:"WEBHOOK_TIMEOUT" envDefault:"30s"`

	// Retry settings
	DeliveryMaxRetries    int           `env:"DELIVERY_MAX_RETRIES" envDefault:"3"`
	DeliveryRetryInterval time.Duration `env:"DELIVERY_RETRY_INTERVAL" envDefault:"5m"`
	// AlertRetryBackoffBase задаёт базовую задержку для exponential backoff retry.
	AlertRetryBackoffBase time.Duration `env:"ALERT_RETRY_BACKOFF_BASE" envDefault:"1m"`
	// AlertRetryMaxDelay ограничивает максимальную задержку retry.
	AlertRetryMaxDelay time.Duration `env:"ALERT_RETRY_MAX_DELAY" envDefault:"10m"`
	// AlertRateLimitRetryDelay задаёт фиксированную задержку retry для rate-limit ошибок.
	AlertRateLimitRetryDelay time.Duration `env:"ALERT_RATE_LIMIT_RETRY_DELAY" envDefault:"5m"`
	// AlertRateLimitWindow задаёт окно подсчёта доставок per-monitor для rate limiting.
	AlertRateLimitWindow time.Duration `env:"ALERT_RATE_LIMIT_WINDOW" envDefault:"5m"`

	// Alert throttling
	AlertCooldownPeriod  time.Duration `env:"ALERT_COOLDOWN_PERIOD" envDefault:"15m"`
	AlertRateLimitPeriod time.Duration `env:"ALERT_RATE_LIMIT_PERIOD" envDefault:"5m"`
	AlertStormWindow     time.Duration `env:"ALERT_STORM_WINDOW" envDefault:"5m"`
	AlertStormThreshold  int           `env:"ALERT_STORM_THRESHOLD" envDefault:"5"`
	AlertStormDuration   time.Duration `env:"ALERT_STORM_DURATION" envDefault:"10m"`

	// Flapping detection
	FlappingWindow     time.Duration `env:"FLAPPING_WINDOW" envDefault:"10m"`
	FlappingThreshold  int           `env:"FLAPPING_THRESHOLD" envDefault:"5"`
	FlappingExitPeriod time.Duration `env:"FLAPPING_EXIT_PERIOD" envDefault:"15m"`

	// Escalation
	EscalationTimeout time.Duration `env:"ESCALATION_TIMEOUT" envDefault:"30m"`

	// Retention
	AlertRetentionDays int `env:"ALERT_RETENTION_DAYS" envDefault:"90"`

	// Channel limits per user tier
	ChannelLimitFree         int `env:"CHANNEL_LIMIT_FREE" envDefault:"10"`
	ChannelLimitStarter      int `env:"CHANNEL_LIMIT_STARTER" envDefault:"25"`
	ChannelLimitProfessional int `env:"CHANNEL_LIMIT_PROFESSIONAL" envDefault:"100"`
	ChannelLimitBusiness     int `env:"CHANNEL_LIMIT_BUSINESS" envDefault:"500"`

	// Delivery queue
	DeliveryQueueLimit int `env:"DELIVERY_QUEUE_LIMIT" envDefault:"1000"`

	// gRPC reflection (включать только в development/staging, не в production)
	GRPCReflection bool `env:"GRPC_REFLECTION" envDefault:"false"`
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from environment: %v", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &cfg, nil
}

// validate проверяет корректность конфигурации
func (c *Config) validate() error {
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid SERVER_PORT: must be between 1 and 65535, got %d", c.ServerPort)
	}

	if c.MetricsPort < 1 || c.MetricsPort > 65535 {
		return fmt.Errorf("invalid METRICS_PORT: must be between 1 and 65535, got %d", c.MetricsPort)
	}

	if c.DBHost == "" {
		return errors.New("missing required env: DB_HOST")
	}

	if c.DBPort < 1 || c.DBPort > 65535 {
		return fmt.Errorf("invalid DB_PORT: must be between 1 and 65535, got %d", c.DBPort)
	}

	if c.DBName == "" {
		return errors.New("missing required env: DB_NAME")
	}

	if c.DBUser == "" {
		return errors.New("missing required env: DB_USER")
	}

	if c.JWTSecret == "" {
		return errors.New("missing required env: JWT_SECRET")
	}

	if len(c.JWTSecret) < 32 {
		return errors.New("invalid JWT_SECRET: must be at least 32 characters")
	}

	if c.DeliveryMaxRetries < 0 {
		return fmt.Errorf("invalid DELIVERY_MAX_RETRIES: must be non-negative, got %d", c.DeliveryMaxRetries)
	}

	if c.DeliveryRetryInterval < 0 {
		return errors.New("invalid DELIVERY_RETRY_INTERVAL: must be non-negative")
	}

	if c.WebhookTimeout < 0 {
		return errors.New("invalid WEBHOOK_TIMEOUT: must be non-negative")
	}

	return nil
}

// AlertThrottlingConfig содержит конфигурацию throttling алертов
type AlertThrottlingConfig struct {
	CooldownPeriod  time.Duration
	RateLimitPeriod time.Duration
	StormWindow     time.Duration
	StormThreshold  int
	StormDuration   time.Duration
}

// AlertThrottlingConfigFromEnv создаёт AlertThrottlingConfig из переменных окружения
func AlertThrottlingConfigFromEnv(cfg Config) AlertThrottlingConfig {
	return AlertThrottlingConfig{
		CooldownPeriod:  cfg.AlertCooldownPeriod,
		RateLimitPeriod: cfg.AlertRateLimitPeriod,
		StormWindow:     cfg.AlertStormWindow,
		StormThreshold:  cfg.AlertStormThreshold,
		StormDuration:   cfg.AlertStormDuration,
	}
}

// DSN возвращает строку подключения к PostgreSQL
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBSSLMode,
	)
}

// ServerAddress возвращает адрес gRPC сервера
func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}
