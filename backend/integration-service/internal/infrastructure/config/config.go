package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config конфигурация Integration Service.
type Config struct {
	// Server
	ServerPort     int
	ServerGRPCPort int

	// Database
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	// RabbitMQ
	RabbitMQEnabled  bool
	RabbitMQHost     string
	RabbitMQPort     int
	RabbitMQUser     string
	RabbitMQPassword string
	RabbitMQVHost    string
	RabbitMQExchange string

	// Observability
	OTELExporterOTLPEndpoint string
	MetricsEnabled           bool
	MetricsPort              int
	LogLevel                 string

	// External Services
	AuthServiceGRPCAddress    string
	BillingServiceGRPCAddress string
	MonitorServiceGRPCAddress string

	// Security
	EncryptionKey string
	JWTSecret     string
}

// LoadConfig загружает конфигурацию из environment variables.
func LoadConfig() (*Config, error) {
	// Load .env file if exists (ignore error if file doesn't exist)
	//nolint:errcheck // .env file is optional, failure is acceptable
	_ = godotenv.Load()

	cfg := &Config{}

	// Server
	cfg.ServerPort = getEnvInt("SERVER_PORT", 8084)
	cfg.ServerGRPCPort = getEnvInt("SERVER_GRPC_PORT", 5014)

	// Database
	cfg.DBHost = getEnv("DB_HOST", "localhost")
	cfg.DBPort = getEnvInt("DB_PORT", 5432)
	cfg.DBName = getEnv("DB_NAME", "integration_service")
	cfg.DBUser = getEnv("DB_USER", "postgres")
	cfg.DBPassword = getEnv("DB_PASSWORD", "postgres")
	cfg.DBSSLMode = getEnv("DB_SSLMODE", "disable")

	// RabbitMQ
	cfg.RabbitMQEnabled = getEnvBool("RABBITMQ_ENABLED", true)
	cfg.RabbitMQHost = getEnv("RABBITMQ_HOST", "localhost")
	cfg.RabbitMQPort = getEnvInt("RABBITMQ_PORT", 5672)
	cfg.RabbitMQUser = getEnv("RABBITMQ_USER", "guest")
	cfg.RabbitMQPassword = getEnv("RABBITMQ_PASSWORD", "guest")
	cfg.RabbitMQVHost = getEnv("RABBITMQ_VHOST", "/")
	cfg.RabbitMQExchange = getEnv("RABBITMQ_EXCHANGE", "monitor")

	// Observability
	cfg.OTELExporterOTLPEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	cfg.MetricsEnabled = getEnvBool("METRICS_ENABLED", true)
	cfg.MetricsPort = getEnvInt("METRICS_PORT", 9094)
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	// External Services
	cfg.AuthServiceGRPCAddress = getEnv("AUTH_SERVICE_GRPC_ADDRESS", "localhost:5001")
	cfg.BillingServiceGRPCAddress = getEnv("BILLING_SERVICE_GRPC_ADDRESS", "localhost:5003")
	cfg.MonitorServiceGRPCAddress = getEnv("MONITOR_SERVICE_GRPC_ADDRESS", "localhost:5000")

	// Security
	cfg.EncryptionKey = getEnv("ENCRYPTION_KEY", "")
	cfg.JWTSecret = getEnv("JWT_SECRET", "")

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Validate проверяет валидность конфигурации.
func (c *Config) Validate() error {
	// Server ports
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", c.ServerPort)
	}

	if c.ServerGRPCPort < 1 || c.ServerGRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.ServerGRPCPort)
	}

	// Database
	if c.DBHost == "" {
		return fmt.Errorf("db host is required")
	}

	if c.DBPort < 1 || c.DBPort > 65535 {
		return fmt.Errorf("invalid db port: %d", c.DBPort)
	}

	if c.DBName == "" {
		return fmt.Errorf("db name is required")
	}

	if c.DBUser == "" {
		return fmt.Errorf("db user is required")
	}

	if c.DBPassword == "" {
		return fmt.Errorf("db password is required")
	}

	// Security
	if c.EncryptionKey == "" {
		return fmt.Errorf("encryption key is required (set ENCRYPTION_KEY env var)")
	}

	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("encryption key must be exactly 32 bytes (characters)")
	}

	// JWT secret обязателен для аутентификации gRPC запросов
	if c.JWTSecret == "" {
		return fmt.Errorf("jwt secret is required (set JWT_SECRET env var)")
	}

	return nil
}

// DSN возвращает connection string для PostgreSQL.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBUser,
		c.DBPassword,
		c.DBSSLMode,
	)
}

// RabbitMQURL возвращает URL для подключения к RabbitMQ.
func (c *Config) RabbitMQURL() string {
	if !c.RabbitMQEnabled {
		return ""
	}

	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d%s",
		c.RabbitMQUser,
		c.RabbitMQPassword,
		c.RabbitMQHost,
		c.RabbitMQPort,
		c.RabbitMQVHost,
	)
}

// RabbitMQExchangeName возвращает имя exchange для RabbitMQ.
func (c *Config) RabbitMQExchangeName() string {
	return c.RabbitMQExchange
}

// getEnv получает значение environment variable или возвращает значение по умолчанию.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// getEnvInt получает int значение environment variable или возвращает значение по умолчанию.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	return defaultValue
}

// getEnvBool получает bool значение environment variable или возвращает значение по умолчанию.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}

	return defaultValue
}
