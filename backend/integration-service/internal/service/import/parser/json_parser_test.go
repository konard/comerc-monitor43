package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

func TestJSONParser_Parse(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name      string
		data      []byte
		wantCount int
		wantError bool
		validate  bool
	}{
		{
			name: "valid monitors",
			data: []byte(`[
				{"name": "Monitor 1", "url": "https://example.com", "method": "GET"},
				{"name": "Monitor 2", "url": "https://test.com", "method": "POST"}
			]`),
			wantCount: 2,
			wantError: false,
		},
		{
			name:      "empty array",
			data:      []byte(`[]`),
			wantCount: 0,
			wantError: false,
		},
		{
			name:      "invalid JSON",
			data:      []byte(`{invalid json}`),
			wantCount: 0,
			wantError: true,
		},
		{
			name: "monitor with validation error",
			data: []byte(`[
				{"name": "Valid Monitor", "url": "https://example.com"},
				{"name": "", "url": ""}
			]`),
			wantCount: 1, // Only valid monitor
			wantError: false,
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

func TestJSONParser_Validate(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name      string
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid monitors",
			data: []byte(`[
				{"name": "Monitor 1", "url": "https://example.com"},
				{"name": "Monitor 2", "url": "https://test.com"}
			]`),
			wantError: false,
		},
		{
			name:      "empty array",
			data:      []byte(`[]`),
			wantError: true,
			errorMsg:  "no monitors found",
		},
		{
			name:      "invalid JSON",
			data:      []byte(`{invalid}`),
			wantError: true,
		},
		{
			name: "monitor with missing name",
			data: []byte(`[
				{"name": "", "url": "https://example.com"}
			]`),
			wantError: true,
			errorMsg:  "monitor 0",
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

func TestJSONParser_GetSource(t *testing.T) {
	parser := NewJSONParser()
	assert.Equal(t, model.ImportSourceJSON, parser.GetSource())
}

func TestJSONParser_ParseWithInvalidMonitor(t *testing.T) {
	parser := NewJSONParser()

	data := []byte(`[
		{"name": "Valid Monitor", "url": "https://example.com", "method": "GET"},
		{"name": "Invalid Monitor", "url": "https://test.com", "method": "INVALID"},
		{"name": "Another Valid", "url": "https://valid.com"}
	]`)

	monitors, err := parser.Parse(data)
	require.NoError(t, err)

	// Должны получить 2 валидных монитора (Valid Monitor и Another Valid)
	assert.Len(t, monitors, 2)

	// Проверяем что невалидный монитор не включен
	for _, m := range monitors {
		assert.NotEqual(t, "Invalid Monitor", m.Name)
	}
}

func TestJSONParser_RoundTrip(t *testing.T) {
	parser := NewJSONParser()

	// Создаём тестовые данные
	original := []*model.MonitorImportData{
		{
			Name:          "Test Monitor",
			URL:           "https://example.com",
			Method:        "GET",
			CheckInterval: intPtr(60),
			Timeout:       intPtr(30),
			Enabled:       boolPtr(true),
			Tags:          []string{"test", "json"},
		},
	}

	// Сериализуем в JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Парсим обратно
	parsed, err := parser.Parse(data)
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	// Проверяем что данные совпадают
	assert.Equal(t, original[0].Name, parsed[0].Name)
	assert.Equal(t, original[0].URL, parsed[0].URL)
	assert.Equal(t, original[0].Method, parsed[0].Method)
}

func intPtr(i int) *int {
	return &i
}
