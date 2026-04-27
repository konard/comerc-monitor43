package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

// Config holds all configuration for the auth service
type Config struct {
	ServerPort     int `env:"SERVER_PORT" envDefault:"8080" validate:"required,min=1,max=65535"`
	ServerGRPCPort int `env:"SERVER_GRPC_PORT" envDefault:"5001" validate:"required,min=1,max=65535"`

	// Database
	DBHost     string `env:"DB_HOST" envDefault:"localhost" validate:"required"`
	DBPort     int    `env:"DB_PORT" envDefault:"5432" validate:"required,min=1,max=65535"`
	DBName     string `env:"DB_NAME" envDefault:"monitor" validate:"required"`
	DBUser     string `env:"DB_USER" envDefault:"monitor" validate:"required"`
	DBPassword string `env:"DB_PASSWORD" validate:"required,min=8"`
	DBSSLMode  string `env:"DB_SSL_MODE" envDefault:"disable"`

	// Redis
	RedisHost     string `env:"REDIS_HOST" envDefault:"localhost" validate:"required"`
	RedisPort     int    `env:"REDIS_PORT" envDefault:"6379" validate:"required,min=1,max=65535"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0" validate:"min=0"`

	// RabbitMQ — подключение
	RabbitMQHost     string `env:"RABBITMQ_HOST" envDefault:"localhost"`
	RabbitMQPort     int    `env:"RABBITMQ_PORT" envDefault:"5672" validate:"min=1,max=65535"`
	RabbitMQUser     string `env:"RABBITMQ_USER" envDefault:"guest"`
	RabbitMQPassword string `env:"RABBITMQ_PASSWORD" envDefault:"guest"`
	RabbitMQVHost    string `env:"RABBITMQ_VHOST" envDefault:"/"`
	RabbitMQEnabled  bool   `env:"RABBITMQ_ENABLED" envDefault:"false"`

	// RabbitMQ — топология (exchange и очередь событий)
	RabbitMQExchange   string `env:"RABBITMQ_EXCHANGE"    envDefault:"auth.events"`
	RabbitMQQueue      string `env:"RABBITMQ_QUEUE"       envDefault:"auth.events"`
	RabbitMQRoutingKey string `env:"RABBITMQ_ROUTING_KEY" envDefault:"auth.events"`

	// Metrics
	MetricsEnabled bool `env:"METRICS_ENABLED" envDefault:"false"`
	MetricsPort    int  `env:"METRICS_PORT" envDefault:"9090" validate:"min=1,max=65535"`

	// JWT
	JWTSecret          string        `env:"JWT_SECRET" validate:"required,min=32"`
	JWTAccessDuration  time.Duration `env:"JWT_ACCESS_DURATION" envDefault:"15m" validate:"required"`
	JWTRefreshDuration time.Duration `env:"JWT_REFRESH_DURATION" envDefault:"168h" validate:"required"`

	// Google OAuth
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID" validate:"required"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET" validate:"required"`
	GoogleRedirectURI  string `env:"GOOGLE_REDIRECT_URI" validate:"required,url"`
	// Google OAuth endpoint overrides (for testing)
	GoogleTokenURL    string `env:"GOOGLE_TOKEN_URL"    envDefault:"https://oauth2.googleapis.com/token"`
	GoogleUserInfoURL string `env:"GOOGLE_USERINFO_URL" envDefault:"https://www.googleapis.com/oauth2/v2/userinfo"`

	// Security
	MaxLoginAttempts    int           `env:"MAX_LOGIN_ATTEMPTS" envDefault:"10" validate:"min=1"`
	AccountLockDuration time.Duration `env:"ACCOUNT_LOCK_DURATION" envDefault:"1h" validate:"required"`
	SessionExpiry       time.Duration `env:"SESSION_EXPIRY" envDefault:"24h" validate:"required"`

	// Observability
	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4318"`
	OTELServiceName          string `env:"OTEL_SERVICE_NAME" envDefault:"auth-service"`
	OTELLogLevel             string `env:"OTEL_LOG_LEVEL" envDefault:"info"`
}

// LoadConfig loads and validates configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return cfg, nil
}

// DSN returns the PostgreSQL data source name
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBName, c.DBUser, c.DBPassword, c.DBSSLMode)
}

// RedisAddr returns the Redis address
func (c *Config) RedisAddr() string {
	if c.RedisPassword != "" {
		return fmt.Sprintf(":%s@%s:%d", c.RedisPassword, c.RedisHost, c.RedisPort)
	}
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

// RabbitMQURL returns RabbitMQ connection URL
func (c *Config) RabbitMQURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		c.RabbitMQUser, c.RabbitMQPassword, c.RabbitMQHost, c.RabbitMQPort, c.RabbitMQVHost)
}
