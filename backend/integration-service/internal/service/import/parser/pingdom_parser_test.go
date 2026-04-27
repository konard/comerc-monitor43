package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPingdomParser_Parse(t *testing.T) {
	parser := NewPingdomParser()

	tests := []struct {
		name      string
		data      []byte
		wantCount int
		wantError bool
	}{
		{
			name: "valid Pingdom export",
			data: []byte(`{
				"checks": [
					{
						"id": 12345,
						"name": "My Website",
						"type": "http",
						"hostname": "https://example.com",
						"resolution": 1,
						"interval": 60,
						"timeout": 30,
						"status": "up",
						"tags": ["production", "web"],
						"alert_threshold": 5
					},
					{
						"id": 67890,
						"name": "API Endpoint",
						"type": "http",
						"hostname": "https://api.example.com",
						"resolution": 1,
						"interval": 120,
						"timeout": 30,
						"status": "paused",
						"tags": ["api"]
					}
				]
			}`),
			wantCount: 2,
			wantError: false,
		},
		{
			name:      "empty checks array",
			data:      []byte(`{"checks": []}`),
			wantCount: 0,
			wantError: false,
		},
		{
			name:      "invalid JSON",
			data:      []byte(`{invalid}`),
			wantCount: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitors, err := parser.Parse(tt.data)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Len(t, monitors, tt.wantCount)

			// Проверяем что валидные мониторы имеют правильные данные
			if len(monitors) > 0 {
				assert.NotEmpty(t, monitors[0].Name)
				assert.NotEmpty(t, monitors[0].URL)
				assert.Contains(t, monitors[0].Tags, "pingdom")
			}
		})
	}
}

func TestPingdomParser_Validate(t *testing.T) {
	parser := NewPingdomParser()

	tests := []struct {
		name      string
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid Pingdom export",
			data: []byte(`{
				"checks": [
					{
						"id": 12345,
						"name": "My Website",
						"hostname": "https://example.com"
					}
				]
			}`),
			wantError: false,
		},
		{
			name:      "empty checks array",
			data:      []byte(`{"checks": []}`),
			wantError: true,
			errorMsg:  "no checks found",
		},
		{
			name:      "invalid JSON",
			data:      []byte(`{invalid}`),
			wantError: true,
			errorMsg:  "invalid Pingdom JSON format",
		},
		{
			name:      "missing checks field",
			data:      []byte(`{"other": "data"}`),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Validate(tt.data)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPingdomParser_GetSource(t *testing.T) {
	parser := NewPingdomParser()
	assert.Equal(t, "pingdom", string(parser.GetSource()))
}

func TestPingdomParser_ConvertCheck(t *testing.T) {
	parser := NewPingdomParser()

	data := []byte(`{
		"checks": [
			{
				"id": 12345,
				"name": "My Website",
				"type": "http",
				"hostname": "https://example.com",
				"interval": 60,
				"status": "up",
				"tags": ["production", "web"],
				"alert_threshold": 5
			}
		]
	}`)

	monitors, err := parser.Parse(data)
	require.NoError(t, err)
	require.Len(t, monitors, 1)

	monitor := monitors[0]

	// Проверяем базовые поля
	assert.Equal(t, "My Website", monitor.Name)
	assert.Equal(t, "https://example.com", monitor.URL)
	assert.Equal(t, "GET", monitor.Method)

	// Проверяем интервал
	assert.NotNil(t, monitor.CheckInterval)
	assert.Equal(t, 60, *monitor.CheckInterval)

	// Проверяем статус
	assert.NotNil(t, monitor.Enabled)
	assert.True(t, *monitor.Enabled)

	// Проверяем теги
	assert.Contains(t, monitor.Tags, "pingdom")
	assert.Contains(t, monitor.Tags, "production")
	assert.Contains(t, monitor.Tags, "web")
	assert.Contains(t, monitor.Tags, "alert_threshold:5")
}

func TestPingdomParser_StatusMapping(t *testing.T) {
	parser := NewPingdomParser()

	tests := []struct {
		status      string
		wantEnabled bool
	}{
		{"up", true},
		{"paused", true},
		{"down", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			data := []byte(`{
				"checks": [
					{
						"id": 12345,
						"name": "Test",
						"hostname": "https://example.com",
						"status": "` + tt.status + `"
					}
				]
			}`)

			monitors, err := parser.Parse(data)
			require.NoError(t, err)
			require.Len(t, monitors, 1)

			assert.NotNil(t, monitors[0].Enabled)
			assert.Equal(t, tt.wantEnabled, *monitors[0].Enabled)
		})
	}
}
