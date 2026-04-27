package config

import (
	"os"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		wantErr  bool
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name:    "default values",
			env:     map[string]string{},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != "8080" {
					t.Errorf("expected port 8080, got %s", cfg.Server.Port)
				}
				if cfg.Auth.Address != "localhost:5001" {
					t.Errorf("expected auth address localhost:5001, got %s", cfg.Auth.Address)
				}
			},
		},
		{
			name: "custom values",
			env: map[string]string{
				"SERVER_PORT":       "9090",
				"AUTH_SERVICE_ADDR": "auth-service:5001",
				"LOG_LEVEL":         "debug",
				"ENABLE_TRACING":    "true",
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != "9090" {
					t.Errorf("expected port 9090, got %s", cfg.Server.Port)
				}
				if cfg.Auth.Address != "auth-service:5001" {
					t.Errorf("expected auth address auth-service:5001, got %s", cfg.Auth.Address)
				}
				if cfg.Observability.LogLevel != "debug" {
					t.Errorf("expected log level debug, got %s", cfg.Observability.LogLevel)
				}
				if !cfg.Observability.EnableTracing {
					t.Error("expected tracing enabled")
				}
			},
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

			cfg, err := NewConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestGetDurationEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		defaults time.Duration
		want     time.Duration
	}{
		{
			name:     "default value",
			key:      "MISSING_KEY",
			value:    "",
			defaults: 10 * time.Second,
			want:     10 * time.Second,
		},
		{
			name:     "custom value",
			key:      "TIMEOUT_KEY",
			value:    "30s",
			defaults: 10 * time.Second,
			want:     30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			got := getDurationEnv(tt.key, tt.defaults)
			if got != tt.want {
				t.Errorf("getDurationEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		defaults bool
		want     bool
	}{
		{
			name:     "default value",
			key:      "MISSING_KEY",
			value:    "",
			defaults: false,
			want:     false,
		},
		{
			name:     "true string",
			key:      "BOOL_KEY",
			value:    "true",
			defaults: false,
			want:     true,
		},
		{
			name:     "one string",
			key:      "BOOL_KEY",
			value:    "1",
			defaults: false,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			got := getBoolEnv(tt.key, tt.defaults)
			if got != tt.want {
				t.Errorf("getBoolEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
