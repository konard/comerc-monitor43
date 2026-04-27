package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name:    "default_values",
			envVars: map[string]string{},
			wantErr: false,
		},
		{
			name: "custom_values",
			envVars: map[string]string{
				"CHECK_WORKER_SCHEDULER_ADDRESS":     "scheduler:9094",
				"CHECK_WORKER_MONITOR_ADDRESS":       "monitor:9090",
				"CHECK_WORKER_NAME":                  "worker-prod-01",
				"CHECK_WORKER_ZONE":                  "eu-west-1",
				"CHECK_WORKER_MAX_CONCURRENT_CHECKS": "50",
				"CHECK_WORKER_CHECK_TIMEOUT":         "60s",
				"CHECK_WORKER_POLL_INTERVAL":         "2s",
				"CHECK_WORKER_HEARTBEAT_INTERVAL":    "5s",
				"CHECK_WORKER_METRICS_PORT":          "9110",
				"OTEL_SERVICE_NAME":                  "check-worker-prod",
			},
			wantErr: false,
		},
		{
			name: "invalid_max_concurrent_too_low",
			envVars: map[string]string{
				"CHECK_WORKER_MAX_CONCURRENT_CHECKS": "0",
			},
			wantErr: true,
		},
		{
			name: "invalid_max_concurrent_too_high",
			envVars: map[string]string{
				"CHECK_WORKER_MAX_CONCURRENT_CHECKS": "1001",
			},
			wantErr: true,
		},
		{
			name: "invalid_retry_max_attempts",
			envVars: map[string]string{
				"CHECK_WORKER_RETRY_MAX_ATTEMPTS": "0",
			},
			wantErr: true,
		},
		{
			name: "invalid_metrics_port",
			envVars: map[string]string{
				"CHECK_WORKER_METRICS_PORT": "0",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			cfg, err := LoadConfig()

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				assert.NotEmpty(t, cfg.SchedulerAddress)
				assert.NotEmpty(t, cfg.MonitorAddress)
				assert.NotEmpty(t, cfg.WorkerName)
				assert.NotEmpty(t, cfg.Zone)
				assert.Greater(t, cfg.MaxConcurrentChecks, 0)
				assert.Greater(t, int(cfg.CheckTimeout), 0)
				assert.Greater(t, int(cfg.PollInterval), 0)
				assert.Greater(t, int(cfg.HeartbeatInterval), 0)
			}
		})
	}
}

func TestLoadConfig_custom_env_overrides_defaults(t *testing.T) {
	t.Setenv("CHECK_WORKER_NAME", "custom-worker")
	t.Setenv("CHECK_WORKER_ZONE", "custom-zone")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "custom-worker", cfg.WorkerName)
	assert.Equal(t, "custom-zone", cfg.Zone)
}
