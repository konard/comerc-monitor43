package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVParser_Parse(t *testing.T) {
	parser := NewCSVParser()

	tests := []struct {
		name      string
		data      []byte
		wantCount int
		wantError bool
	}{
		{
			name: "valid CSV with minimal columns",
			data: []byte(`name,url
Monitor 1,https://example.com
Monitor 2,https://test.com`),
			wantCount: 2,
			wantError: false,
		},
		{
			name: "valid CSV with all columns",
			data: []byte(`name,url,method,interval,timeout,expected_status,enabled
Monitor 1,https://example.com,GET,60,30,200,true
Monitor 2,https://test.com,POST,120,60,201,false`),
			wantCount: 2,
			wantError: false,
		},
		{
			name: "CSV with alternative column names",
			data: []byte(`monitor_name,monitor_url,http_method,check_interval
Monitor 1,https://example.com,GET,60s
Monitor 2,https://test.com,POST,2m`),
			wantCount: 2,
			wantError: false,
		},
		{
			name:      "empty CSV",
			data:      []byte(`name,url`),
			wantCount: 0,
			wantError: false,
		},
		{
			name: "missing required column",
			data: []byte(`name,method
Monitor 1,GET`),
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

func TestCSVParser_Validate(t *testing.T) {
	parser := NewCSVParser()

	tests := []struct {
		name      string
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid CSV",
			data: []byte(`name,url
Monitor 1,https://example.com`),
			wantError: false,
		},
		{
			name:      "empty CSV",
			data:      []byte(``),
			wantError: true,
			errorMsg:  "empty",
		},
		{
			name: "missing name column",
			data: []byte(`url,method
https://example.com,GET`),
			wantError: true,
			errorMsg:  "name",
		},
		{
			name: "missing url column",
			data: []byte(`name,method
Monitor 1,GET`),
			wantError: true,
			errorMsg:  "url",
		},
		{
			name: "less than 2 columns",
			data: []byte(`name
Monitor 1`),
			wantError: true,
			errorMsg:  "at least 2 columns",
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

func TestCSVParser_GetSource(t *testing.T) {
	parser := NewCSVParser()
	assert.Equal(t, "csv", string(parser.GetSource()))
}

func TestCSVParser_ParseInterval(t *testing.T) {
	tests := []struct {
		input     string
		expected  int
		wantError bool
	}{
		{"60", 60, false},
		{"60s", 60, false},
		{"1m", 60, false},
		{"2m", 120, false},
		{"1h", 3600, false},
		{"90", 90, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseInterval(tt.input)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCSVParser_EmptyRows(t *testing.T) {
	parser := NewCSVParser()

	data := []byte(`name,url
Monitor 1,https://example.com

Monitor 2,https://test.com
`)

	monitors, err := parser.Parse(data)
	require.NoError(t, err)

	// Должны пропустить пустую строку
	assert.Len(t, monitors, 2)
	assert.Equal(t, "Monitor 1", monitors[0].Name)
	assert.Equal(t, "Monitor 2", monitors[1].Name)
}

func TestCSVParser_DefaultValues(t *testing.T) {
	parser := NewCSVParser()

	data := []byte(`name,url
Monitor 1,https://example.com`)

	monitors, err := parser.Parse(data)
	require.NoError(t, err)
	require.Len(t, monitors, 1)

	// Проверяем дефолтные значения
	assert.Equal(t, "GET", monitors[0].Method)
	assert.NotNil(t, monitors[0].Enabled)
	assert.True(t, *monitors[0].Enabled)
}

func TestCSVParser_BooleanParsing(t *testing.T) {
	parser := NewCSVParser()

	tests := []struct {
		enabled  string
		wantBool bool
	}{
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"FALSE", false},
		{"1", true},
		{"0", false},
	}

	for _, tt := range tests {
		t.Run(tt.enabled, func(t *testing.T) {
			data := []byte("name,url,enabled\nMonitor 1,https://example.com," + tt.enabled)

			monitors, err := parser.Parse(data)
			require.NoError(t, err)
			require.Len(t, monitors, 1)

			assert.NotNil(t, monitors[0].Enabled)
			assert.Equal(t, tt.wantBool, *monitors[0].Enabled)
		})
	}
}

func TestCSVParser_CaseInsensitiveColumns(t *testing.T) {
	parser := NewCSVParser()

	data := []byte(`NAME,URL,METHOD
Monitor 1,https://example.com,GET`)

	monitors, err := parser.Parse(data)
	require.NoError(t, err)
	require.Len(t, monitors, 1)

	assert.Equal(t, "Monitor 1", monitors[0].Name)
	assert.Equal(t, "https://example.com", monitors[0].URL)
	assert.Equal(t, "GET", monitors[0].Method)
}
