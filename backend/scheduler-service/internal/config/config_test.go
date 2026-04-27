package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "default values",
			env: map[string]string{
				"DB_PASSWORD": "test",
			},
			want: &Config{
				ServerPort:                8084,
				ServerGRPCPort:            9094,
				DBHost:                    "localhost",
				DBPort:                    5432,
				DBName:                    "monitor",
				DBUser:                    "scheduler",
				DBPassword:                "test",
				DBSSLMode:                 "disable",
				RabbitMQHost:              "localhost",
				RabbitMQPort:              5672,
				RabbitMQUser:              "guest",
				RabbitMQPassword:          "guest",
				RabbitMQVHost:             "/",
				RabbitMQEnabled:           true,
				MonitorServiceGRPCAddress: "localhost:9091",
				MetricsEnabled:            true,
				MetricsPort:               9104,
				HeartbeatTimeout:          30 * time.Second,
				OfflineCleanupInterval:    10 * time.Minute,
				OverdueCheckThreshold:     2 * time.Minute,
				SchedulingInterval:        10 * time.Second,
				CBFailureThreshold:        5,
				CBTimeout:                 30 * time.Second,
				CBSuccessThreshold:        1,
				OTELExporterOTLPEndpoint:  "http://localhost:4318",
				OTELServiceName:           "scheduler-service",
				OTELLogLevel:              "info",
			},
			wantErr: false,
		},
		{
			name: "custom values",
			env: map[string]string{
				"SERVER_PORT":       "8080",
				"SERVER_GRPC_PORT":  "9090",
				"DB_HOST":           "db.example.com",
				"DB_PORT":           "5433",
				"DB_NAME":           "production",
				"DB_USER":           "admin",
				"DB_PASSWORD":       "secret123",
				"HEARTBEAT_TIMEOUT": "60s",
				"METRICS_ENABLED":   "false",
			},
			want: &Config{
				ServerPort:                8080,
				ServerGRPCPort:            9090,
				DBHost:                    "db.example.com",
				DBPort:                    5433,
				DBName:                    "production",
				DBUser:                    "admin",
				DBPassword:                "secret123",
				DBSSLMode:                 "disable",
				RabbitMQHost:              "localhost",
				RabbitMQPort:              5672,
				RabbitMQUser:              "guest",
				RabbitMQPassword:          "guest",
				RabbitMQVHost:             "/",
				RabbitMQEnabled:           true,
				MonitorServiceGRPCAddress: "localhost:9091",
				MetricsEnabled:            false,
				MetricsPort:               9104,
				HeartbeatTimeout:          60 * time.Second,
				OfflineCleanupInterval:    10 * time.Minute,
				OverdueCheckThreshold:     2 * time.Minute,
				SchedulingInterval:        10 * time.Second,
				CBFailureThreshold:        5,
				CBTimeout:                 30 * time.Second,
				CBSuccessThreshold:        1,
				OTELExporterOTLPEndpoint:  "http://localhost:4318",
				OTELServiceName:           "scheduler-service",
				OTELLogLevel:              "info",
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			env: map[string]string{
				"DB_PASSWORD": "test",
				"SERVER_PORT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "missing required field",
			env: map[string]string{
				"DB_PASSWORD": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.env {
					os.Unsetenv(k)
				}
			}()

			got, err := LoadConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.ServerPort, got.ServerPort)
				assert.Equal(t, tt.want.ServerGRPCPort, got.ServerGRPCPort)
				assert.Equal(t, tt.want.DBHost, got.DBHost)
				assert.Equal(t, tt.want.DBPort, got.DBPort)
				assert.Equal(t, tt.want.DBName, got.DBName)
				assert.Equal(t, tt.want.DBUser, got.DBUser)
				assert.Equal(t, tt.want.DBPassword, got.DBPassword)
				assert.Equal(t, tt.want.MetricsEnabled, got.MetricsEnabled)
			}
		})
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg := &Config{
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "testdb",
		DBUser:     "testuser",
		DBPassword: "testpass",
		DBSSLMode:  "disable",
	}

	want := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"
	got := cfg.DSN()

	assert.Equal(t, want, got)
}

func TestConfig_RabbitMQURL(t *testing.T) {
	cfg := &Config{
		RabbitMQHost:     "localhost",
		RabbitMQPort:     5672,
		RabbitMQUser:     "guest",
		RabbitMQPassword: "guest",
		RabbitMQVHost:    "/",
	}

	want := "amqp://guest:guest@localhost:5672/" //nolint:gosec // G101: тестовые учётные данные RabbitMQ
	got := cfg.RabbitMQURL()

	assert.Equal(t, want, got)
}
