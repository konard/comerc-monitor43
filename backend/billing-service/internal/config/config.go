package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// Config конфигурация billing-service.
// Передаётся по значению (напомним, это намеренное копирование).
type Config struct {
	LogLevel     string `env:"LOG_LEVEL" envDefault:"info"`
	Server       ServerConfig
	Database     DatabaseConfig
	Yookassa     YookassaConfig
	Stripe       StripeConfig
	Tracing      TracingConfig
	Metrics      MetricsConfig
	Subscription SubscriptionConfig
	Auth         AuthConfig
	RabbitMQ     RabbitMQConfig
}

// ServerConfig конфигурация HTTP и gRPC серверов.
type ServerConfig struct {
	Port     int           `env:"SERVER_PORT" envDefault:"8082" validate:"required,min=1,max=65535"`
	GRPCPort int           `env:"SERVER_GRPC_PORT" envDefault:"5003" validate:"required,min=1,max=65535"`
	Timeout  time.Duration `env:"SERVER_TIMEOUT" envDefault:"30s" validate:"required"`
}

// DatabaseConfig конфигурация подключения к PostgreSQL.
type DatabaseConfig struct {
	Host            string        `env:"DB_HOST" envDefault:"localhost" validate:"required"`
	Port            int           `env:"DB_PORT" envDefault:"5432" validate:"required,min=1,max=65535"`
	Name            string        `env:"DB_NAME" envDefault:"billing" validate:"required"`
	User            string        `env:"DB_USER" envDefault:"postgres" validate:"required"`
	Password        string        `env:"DB_PASSWORD" envDefault:"postgres" validate:"required,min=8"`
	SSLMode         string        `env:"DB_SSL_MODE" envDefault:"disable" validate:"required,oneof=disable require allow prefer verify-ca verify-full"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25" validate:"min=1"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5" validate:"min=1"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m" validate:"required"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"30s" validate:"required"`
}

// YookassaConfig конфигурация платежного провайдера Yookassa.
type YookassaConfig struct {
	ShopID        string `env:"YOOKASSA_SHOP_ID" validate:"required_if=YookassaEnabled true"`
	SecretKey     string `env:"YOOKASSA_SECRET_KEY" validate:"required_if=YookassaEnabled true,min=32"`
	WebhookSecret string `env:"YOOKASSA_WEBHOOK_SECRET"`
	ReturnURL     string `env:"YOOKASSA_RETURN_URL" validate:"url"`
	BaseURL       string `env:"YOOKASSA_BASE_URL" envDefault:"https://api.yookassa.ru/v3" validate:"url"`
}

// StripeConfig конфигурация платежного провайдера Stripe.
type StripeConfig struct {
	APIKey         string `env:"STRIPE_API_KEY" validate:"required_if=StripeEnabled true,min=32"`
	WebhookSecret  string `env:"STRIPE_WEBHOOK_SECRET"`
	PublishableKey string `env:"STRIPE_PUBLISHABLE_KEY"`
	BaseURL        string `env:"STRIPE_BASE_URL" envDefault:"https://api.stripe.com" validate:"url"`
}

// TracingConfig конфигурация OpenTelemetry трассировки.
type TracingConfig struct {
	Enabled     bool    `env:"OTEL_ENABLED" envDefault:"true"`
	Endpoint    string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318" validate:"url"`
	ServiceName string  `env:"OTEL_SERVICE_NAME" envDefault:"billing-service" validate:"required"`
	Environment string  `env:"OTEL_ENVIRONMENT" envDefault:"development" validate:"required,oneof=development staging production"`
	SampleRate  float64 `env:"OTEL_SAMPLING_RATE" envDefault:"1.0" validate:"min=0,max=1"`
}

// MetricsConfig конфигурация Prometheus метрик.
type MetricsConfig struct {
	Enabled bool   `env:"METRICS_ENABLED" envDefault:"true"`
	Port    int    `env:"METRICS_PORT" envDefault:"9092" validate:"min=1,max=65535"`
	Path    string `env:"METRICS_PATH" envDefault:"/metrics" validate:"required"`
}

// SubscriptionConfig конфигурация подписок.
type SubscriptionConfig struct {
	TrialPeriodDays     int  `env:"DEFAULT_TRIAL_PERIOD_DAYS" envDefault:"7"`
	GracePeriodDays     int  `env:"SUBSCRIPTION_GRACE_PERIOD_DAYS" envDefault:"3"`
	AutoRenewEnabled    bool `env:"AUTO_RENEW_ENABLED" envDefault:"true"`
	WebhookVerification bool `env:"FEATURE_WEBHOOK_VERIFICATION" envDefault:"true"`
}

// AuthConfig конфигурация интеграции с Auth Service.
type AuthConfig struct {
	GRPCAddress string `env:"AUTH_SERVICE_GRPC_ADDRESS" envDefault:"localhost:5001" validate:"required,hostname_port"`
	JWTSecret   string `env:"JWT_SECRET" envDefault:"change-me-to-at-least-32-characters-long"`
}

// RabbitMQConfig конфигурация RabbitMQ.
type RabbitMQConfig struct {
	URL      string `env:"RABBITMQ_URL" envDefault:"amqp://guest:guest@localhost:5672/" validate:"required,url"`
	Exchange string `env:"RABBITMQ_EXCHANGE" envDefault:"billing.events" validate:"required"`
}

// Load загружает конфигурацию из переменных окружения и валидирует её.
func Load() (Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "failed to parse config from environment")
	}

	// Валидация конфигурации
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config: %v", err)
	}

	return cfg, nil
}
