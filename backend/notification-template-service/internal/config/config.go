package config

import (
	"fmt"
	"time"
)

// Config представляет конфигурацию notification-template-service.
type Config struct {
	// Server настройки
	ServerAddress string        `envconfig:"SERVER_ADDRESS" default:":50051"`
	ServerTimeout time.Duration `envconfig:"SERVER_TIMEOUT" default:"30s"`

	// Database настройки
	DatabaseHost     string `envconfig:"DB_HOST" default:"localhost"`
	DatabasePort     int    `envconfig:"DB_PORT" default:"5432"`
	DatabaseUser     string `envconfig:"DB_USER" default:"monitor"`
	DatabasePassword string `envconfig:"DB_PASSWORD" default:"monitor"`
	DatabaseName     string `envconfig:"DB_NAME" default:"notification_templates"`
	DatabaseSSLMode  string `envconfig:"DB_SSLMODE" default:"disable"`

	// Migration настройки
	MigrationsPath string `envconfig:"MIGRATIONS_PATH" default:"./migrations"`

	// Observability
	JaegerEndpoint string `envconfig:"JAEGER_ENDPOINT" default:"localhost:4318"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"info"`
	Environment    string `envconfig:"ENVIRONMENT" default:"development"`

	// Template настройки
	MaxTemplateSize int  `envconfig:"MAX_TEMPLATE_SIZE" default:"100000"` // 100KB
	EnablePreview   bool `envconfig:"ENABLE_PREVIEW" default:"true"`
}

// DatabaseDSN возвращает DSN для подключения к PostgreSQL.
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseName,
		c.DatabaseSSLMode,
	)
}

// Validate проверяет конфигурацию.
func (c *Config) Validate() error {
	if c.ServerAddress == "" {
		return fmt.Errorf("server address is required")
	}
	if c.DatabaseHost == "" {
		return fmt.Errorf("database host is required")
	}
	if c.DatabaseName == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}
