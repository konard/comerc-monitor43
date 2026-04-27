package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUptimeRobotParser_Parse(t *testing.T) {
	parser := NewUptimeRobotParser()

	tests := []struct {
		name      string
		data      []byte
		wantCount int
		wantError bool
	}{
		{
			name: "valid UptimeRobot export",
			data: []byte(`{
				"monitors": [
					{
						"id": 12345678,
						"user_id": 987,
						"friendly_name": "My Website",
						"url": "https://example.com",
						"type": 1,
						"interval": 300,
						"timeout": 30,
						"status": 1,
						"contact_groups": [123, 456],
						"tags": ["production", "web"]
					},
					{
						"id": 87654321,
						"user_id": 987,
						"friendly_name": "API Endpoint",
						"url": "https://api.example.com",
						"type": 1,
						"interval": 60,
						"timeout": 30,
						"status": 2,
						"tags": ["api"]
					}
				]
			}`),
			wantCount: 2,
			wantError: false,
		},
		{
			name:      "empty monitors array",
			data:      []byte(`{"monitors": []}`),
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
			}
		})
	}
}

func TestUptimeRobotParser_Validate(t *testing.T) {
	parser := NewUptimeRobotParser()

	tests := []struct {
		name      string
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid UptimeRobot export",
			data: []byte(`{
				"monitors": [
					{
						"id": 12345678,
						"friendly_name": "My Website",
						"url": "https://example.com"
					}
				]
			}`),
			wantError: false,
		},
		{
			name:      "empty monitors array",
			data:      []byte(`{"monitors": []}`),
			wantError: true,
			errorMsg:  "no monitors found",
		},
		{
			name:      "invalid JSON",
			data:      []byte(`{invalid}`),
			wantError: true,
			errorMsg:  "invalid UptimeRobot JSON format",
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

func TestUptimeRobotParser_GetSource(t *testing.T) {
	parser := NewUptimeRobotParser()
	assert.Equal(t, "uptimerobot", string(parser.GetSource()))
}

func TestUptimeRobotParser_ConvertMonitor(t *testing.T) {
	parser := NewUptimeRobotParser()

	data := []byte(`{
		"monitors": [
			{
				"id": 12345678,
				"user_id": 987,
				"friendly_name": "My Website",
				"url": "https://example.com",
				"type": 1,
				"interval": 300,
				"timeout": 30,
				"status": 1,
				"contact_groups": [123, 456],
				"tags": ["production", "web"]
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
	assert.Equal(t, 300, *monitor.CheckInterval)

	// Проверяем статус (status=1 means up, so enabled)
	assert.NotNil(t, monitor.Enabled)
	assert.True(t, *monitor.Enabled)

	// Проверяем теги
	assert.Contains(t, monitor.Tags, "production")
	assert.Contains(t, monitor.Tags, "web")
}

func TestUptimeRobotParser_StatusMapping(t *testing.T) {
	parser := NewUptimeRobotParser()

	tests := []struct {
		status      int
		wantEnabled bool
	}{
		{0, false}, // not checked
		{1, true},  // up
		{2, true},  // paused (still enabled)
		{9, false}, // down
	}

	for _, tt := range tests {
		statusName := string(rune(tt.status))      //nolint:gosec // G115: status ∈ {0,1,2,9}
		statusVal := string(rune(tt.status + '0')) //nolint:gosec // G115
		t.Run(statusName, func(t *testing.T) {
			data := []byte(`{
				"monitors": [
					{
						"id": 12345678,
						"friendly_name": "Test",
						"url": "https://example.com",
						"status": ` + statusVal + `
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

func TestUptimeRobotParser_TypeMapping(t *testing.T) {
	parser := NewUptimeRobotParser()

	tests := []struct {
		monitorType int
		wantMethod  string
	}{
		{1, "GET"}, // HTTP
		{2, "GET"}, // Keyword
	}

	for _, tt := range tests {
		typeName := string(rune(tt.monitorType))      //nolint:gosec // G115: type ∈ {1,2}
		typeVal := string(rune(tt.monitorType + '0')) //nolint:gosec // G115
		t.Run(typeName, func(t *testing.T) {
			data := []byte(`{
				"monitors": [
					{
						"id": 12345678,
						"friendly_name": "Test",
						"url": "https://example.com",
						"type": ` + typeVal + `
					}
				]
			}`)

			monitors, err := parser.Parse(data)
			require.NoError(t, err)
			require.Len(t, monitors, 1)

			assert.Equal(t, tt.wantMethod, monitors[0].Method)
		})
	}
}
