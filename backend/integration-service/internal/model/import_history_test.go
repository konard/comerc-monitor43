package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImportHistory(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	h := NewImportHistory(userID, ImportSourceCSV, "data.csv", 1024, true)

	require.NotNil(t, h)
	assert.Equal(t, userID, h.UserID)
	assert.Equal(t, ImportSourceCSV, h.Source)
	assert.Equal(t, ImportStatusPending, h.Status)
	assert.True(t, h.OverwriteExisting)
	require.NotNil(t, h.FileName)
	assert.Equal(t, "data.csv", *h.FileName)
	require.NotNil(t, h.FileSizeBytes)
	assert.Equal(t, 1024, *h.FileSizeBytes)
	assert.NotEqual(t, uuid.Nil, h.ID)
}

func TestImportHistoryMarkAsRunning(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.MarkAsRunning()
	assert.Equal(t, ImportStatusRunning, h.Status)
}

func TestImportHistoryMarkAsCompleted(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	before := time.Now()
	h.MarkAsCompleted()
	after := time.Now()

	assert.Equal(t, ImportStatusCompleted, h.Status)
	require.NotNil(t, h.CompletedAt)
	assert.True(t, h.CompletedAt.After(before) || h.CompletedAt.Equal(before))
	assert.True(t, h.CompletedAt.Before(after) || h.CompletedAt.Equal(after))
	require.NotNil(t, h.DurationMs)
	assert.GreaterOrEqual(t, *h.DurationMs, 0)
}

func TestImportHistoryMarkAsFailed(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.MarkAsFailed("connection timeout")

	assert.Equal(t, ImportStatusFailed, h.Status)
	require.NotNil(t, h.ErrorMessage)
	assert.Equal(t, "connection timeout", *h.ErrorMessage)
	require.NotNil(t, h.CompletedAt)
	require.NotNil(t, h.DurationMs)
}

func TestImportHistoryMarkAsPartial(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.MarkAsPartial()

	assert.Equal(t, ImportStatusPartial, h.Status)
	require.NotNil(t, h.CompletedAt)
	require.NotNil(t, h.DurationMs)
}

func TestImportHistoryAddSuccessfulImport(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	monitorID := uuid.New()
	h.AddSuccessfulImport(monitorID)

	assert.Equal(t, 1, h.SuccessfulImports)
	assert.Equal(t, 1, h.TotalMonitors)
	assert.Contains(t, h.ImportedMonitorIDs, monitorID)
}

func TestImportHistoryAddFailedImport(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.AddFailedImport()

	assert.Equal(t, 1, h.FailedImports)
	assert.Equal(t, 1, h.TotalMonitors)
}

func TestImportHistoryAddSkippedImport(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.AddSkippedImport()

	assert.Equal(t, 1, h.SkippedImports)
	assert.Equal(t, 1, h.TotalMonitors)
}

func TestImportHistoryAddValidationError(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.AddValidationError(5, "url", "invalid URL", "bad-url")

	assert.Equal(t, 1, h.FailedImports)
	assert.Len(t, h.ValidationErrors, 1)
	assert.Equal(t, 5, h.ValidationErrors[0].Row)
	assert.Equal(t, "url", h.ValidationErrors[0].Field)
	assert.Equal(t, "invalid URL", h.ValidationErrors[0].Message)
	assert.Equal(t, "bad-url", h.ValidationErrors[0].Value)
}

func TestImportHistorySetConflictResolution(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	details := map[string]any{"strategy": "overwrite"}
	h.SetConflictResolution(details)

	assert.Equal(t, details, h.ConflictResolution)
}

func TestImportHistoryGetSuccessRate(t *testing.T) {
	t.Parallel()

	t.Run("zero total returns zero", func(t *testing.T) {
		t.Parallel()
		h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
		assert.Equal(t, 0.0, h.GetSuccessRate())
	})

	t.Run("partial success", func(t *testing.T) {
		t.Parallel()
		h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
		h.AddSuccessfulImport(uuid.New())
		h.AddSuccessfulImport(uuid.New())
		h.AddSuccessfulImport(uuid.New())
		h.AddFailedImport()
		// 3/4 = 75%
		assert.Equal(t, 75.0, h.GetSuccessRate())
	})
}

func TestImportHistoryHasErrors(t *testing.T) {
	t.Parallel()

	t.Run("no errors", func(t *testing.T) {
		t.Parallel()
		h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
		assert.False(t, h.HasErrors())
	})

	t.Run("has validation errors", func(t *testing.T) {
		t.Parallel()
		h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
		h.AddValidationError(1, "name", "required", "")
		assert.True(t, h.HasErrors())
	})
}

func TestImportHistoryIsFinished(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   ImportStatus
		expected bool
	}{
		{"pending is not finished", ImportStatusPending, false},
		{"running is not finished", ImportStatusRunning, false},
		{"completed is finished", ImportStatusCompleted, true},
		{"failed is finished", ImportStatusFailed, true},
		{"partial is finished", ImportStatusPartial, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
			h.Status = tt.status
			assert.Equal(t, tt.expected, h.IsFinished())
		})
	}
}

func TestImportHistoryGetValidationErrorsJSON(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.AddValidationError(1, "url", "invalid URL", "bad")

	data, err := h.GetValidationErrorsJSON()
	require.NoError(t, err)
	require.NotNil(t, data)

	var errors []ValidationError
	require.NoError(t, json.Unmarshal(data, &errors))
	assert.Len(t, errors, 1)
	assert.Equal(t, 1, errors[0].Row)
}

func TestImportHistoryGetImportedMonitorIDsJSON(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	id1 := uuid.New()
	id2 := uuid.New()
	h.AddSuccessfulImport(id1)
	h.AddSuccessfulImport(id2)

	data, err := h.GetImportedMonitorIDsJSON()
	require.NoError(t, err)
	require.NotNil(t, data)

	var ids []string
	require.NoError(t, json.Unmarshal(data, &ids))
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, id1.String())
	assert.Contains(t, ids, id2.String())
}

func TestImportHistoryGetConflictResolutionJSON(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.SetConflictResolution(map[string]any{"action": "skip"})

	data, err := h.GetConflictResolutionJSON()
	require.NoError(t, err)
	require.NotNil(t, data)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "skip", result["action"])
}

func TestParseValidationErrorsFromJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid json", func(t *testing.T) {
		t.Parallel()
		data := `[{"row":1,"field":"name","message":"required"}]`
		errors, err := ParseValidationErrorsFromJSON([]byte(data))
		require.NoError(t, err)
		assert.Len(t, errors, 1)
		assert.Equal(t, 1, errors[0].Row)
		assert.Equal(t, "name", errors[0].Field)
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseValidationErrorsFromJSON([]byte("not-json"))
		require.Error(t, err)
	})
}

func TestParseImportedMonitorIDsFromJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid json with uuids", func(t *testing.T) {
		t.Parallel()
		id := uuid.New()
		data, err := json.Marshal([]string{id.String()})
		require.NoError(t, err)
		ids, err := ParseImportedMonitorIDsFromJSON(data)
		require.NoError(t, err)
		assert.Len(t, ids, 1)
		assert.Equal(t, id, ids[0])
	})

	t.Run("invalid uuid returns error", func(t *testing.T) {
		t.Parallel()
		data, err := json.Marshal([]string{"not-a-uuid"})
		require.NoError(t, err)
		_, err = ParseImportedMonitorIDsFromJSON(data)
		require.Error(t, err)
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseImportedMonitorIDsFromJSON([]byte("not-json"))
		require.Error(t, err)
	})
}

func TestParseConflictResolutionFromJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid json", func(t *testing.T) {
		t.Parallel()
		data := `{"action":"overwrite"}`
		result, err := ParseConflictResolutionFromJSON([]byte(data))
		require.NoError(t, err)
		assert.Equal(t, "overwrite", result["action"])
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseConflictResolutionFromJSON([]byte("not-json"))
		require.Error(t, err)
	})
}

func TestValidateImportSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		wantErr  bool
		expected ImportSource
	}{
		{"csv", "csv", false, ImportSourceCSV},
		{"json", "json", false, ImportSourceJSON},
		{"pingdom", "pingdom", false, ImportSourcePingdom},
		{"uptimerobot", "uptimerobot", false, ImportSourceUptimeRobot},
		{"invalid source", "datadog", true, ""},
		{"empty source", "", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ValidateImportSource(tt.source)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMonitorImportDataValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid data passes", func(t *testing.T) {
		t.Parallel()
		m := &MonitorImportData{Name: "My Monitor", URL: "https://example.com"}
		assert.NoError(t, m.Validate())
	})

	t.Run("empty name returns error", func(t *testing.T) {
		t.Parallel()
		m := &MonitorImportData{URL: "https://example.com"}
		require.Error(t, m.Validate())
	})

	t.Run("empty url returns error", func(t *testing.T) {
		t.Parallel()
		m := &MonitorImportData{Name: "Test"}
		require.Error(t, m.Validate())
	})

	t.Run("invalid http method returns error", func(t *testing.T) {
		t.Parallel()
		m := &MonitorImportData{Name: "Test", URL: "https://example.com", Method: "CONNECT"}
		require.Error(t, m.Validate())
	})

	t.Run("valid http method passes", func(t *testing.T) {
		t.Parallel()
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			m := &MonitorImportData{Name: "Test", URL: "https://example.com", Method: method}
			assert.NoError(t, m.Validate(), "method %s should pass", method)
		}
	})
}

func TestNewImportResult(t *testing.T) {
	t.Parallel()

	h := NewImportHistory(uuid.New(), ImportSourceCSV, "f.csv", 100, false)
	h.AddSuccessfulImport(uuid.New())
	h.AddFailedImport()
	h.AddSkippedImport()
	h.MarkAsPartial()

	result := NewImportResult(h)
	require.NotNil(t, result)
	assert.Equal(t, h.ID, result.ImportID)
	assert.Equal(t, h.Status, result.Status)
	assert.Equal(t, 3, result.TotalMonitors)
	assert.Equal(t, 1, result.SuccessfulCount)
	assert.Equal(t, 1, result.FailedCount)
	assert.Equal(t, 1, result.SkippedCount)
}
