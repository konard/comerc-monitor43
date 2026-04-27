package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all configuration for the API Gateway
type Config struct {
	Server        ServerConfig        `validate:"required"`
	Auth          AuthConfig          `validate:"required"`
	Downstream    DownstreamConfig    `validate:"required"`
	Observability ObservabilityConfig `validate:"required"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port           string        `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout    time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout   time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout    time.Duration `env:"SERVER_IDLE_TIMEOUT" envDefault:"60s"`
	AllowedOrigins []string      // заполняется из ALLOWED_ORIGINS (через запятую)
}

// AuthConfig holds authentication service configuration
type AuthConfig struct {
	Address string        `env:"AUTH_SERVICE_ADDR" envDefault:"localhost:5001"`
	Timeout time.Duration `env:"AUTH_SERVICE_TIMEOUT" envDefault:"5s"`
}

// DownstreamConfig holds downstream service configuration
type DownstreamConfig struct {
	ServiceTimeout  time.Duration `env:"SERVICE_TIMEOUT" envDefault:"30s"`
	Services        map[string]string
	GrpcAddresses   map[string]string
	DashboardWSAddr string // адрес HTTP-сервера dashboard-service для WebSocket
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	LogLevel      string        `env:"LOG_LEVEL" envDefault:"info"`
	EnableTracing bool          `env:"ENABLE_TRACING" envDefault:"false"`
	Metrics       MetricsConfig `env_prefix:"METRICS_"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool   `env:"ENABLED" envDefault:"false"`
	Port    string `env:"PORT" envDefault:"9090"`
}

// NewConfig creates a new configuration from environment variables
func NewConfig() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			ReadTimeout:    getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:   getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:    getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
			AllowedOrigins: getSliceEnv("ALLOWED_ORIGINS"),
		},
		Auth: AuthConfig{
			Address: getEnv("AUTH_SERVICE_ADDR", "localhost:5001"),
			Timeout: getDurationEnv("AUTH_SERVICE_TIMEOUT", 5*time.Second),
		},
		Downstream: DownstreamConfig{
			ServiceTimeout: getDurationEnv("SERVICE_TIMEOUT", 30*time.Second),
			Services: map[string]string{
				"monitor":     getEnv("MONITOR_SERVICE_ADDR", "http://localhost:5002"),
				"alert":       getEnv("ALERT_SERVICE_ADDR", "http://localhost:5003"),
				"billing":     getEnv("BILLING_SERVICE_ADDR", "http://localhost:5004"),
				"scheduler":   getEnv("SCHEDULER_SERVICE_ADDR", "http://localhost:5005"),
				"dashboard":   getEnv("DASHBOARD_SERVICE_ADDR", "http://localhost:5006"),
				"reporting":   getEnv("REPORTING_SERVICE_ADDR", "http://localhost:5007"),
				"maintenance": getEnv("MAINTENANCE_SERVICE_ADDR", "http://localhost:5008"),
			},
			DashboardWSAddr: getEnv("DASHBOARD_WS_ADDR", "http://localhost:8095"),
			GrpcAddresses: map[string]string{
				"auth":         getEnv("AUTH_GRPC_ADDR", "localhost:5001"),
				"monitor":      getEnv("MONITOR_GRPC_ADDR", "localhost:5002"),
				"alert":        getEnv("ALERT_GRPC_ADDR", "localhost:5003"),
				"billing":      getEnv("BILLING_GRPC_ADDR", "localhost:5004"),
				"scheduler":    getEnv("SCHEDULER_GRPC_ADDR", "localhost:5005"),
				"dashboard":    getEnv("DASHBOARD_GRPC_ADDR", "localhost:5006"),
				"reporting":    getEnv("REPORTING_GRPC_ADDR", "localhost:5007"),
				"maintenance":  getEnv("MAINTENANCE_GRPC_ADDR", "localhost:5008"),
				"templates":    getEnv("TEMPLATES_GRPC_ADDR", "localhost:50051"),
				"integrations": getEnv("INTEGRATIONS_GRPC_ADDR", "localhost:5014"),
			},
		},
		Observability: ObservabilityConfig{
			LogLevel:      getEnv("LOG_LEVEL", "info"),
			EnableTracing: getBoolEnv("ENABLE_TRACING", false),
			Metrics: MetricsConfig{
				Enabled: getBoolEnv("METRICS_ENABLED", false),
				Port:    getEnv("METRICS_PORT", "9090"),
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Auth.Address == "" {
		return fmt.Errorf("auth service address is required")
	}
	return nil
}

// getEnv gets an environment variable or returns the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable or returns the default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable or returns the default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// getSliceEnv gets a comma-separated environment variable as a string slice
func getSliceEnv(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
