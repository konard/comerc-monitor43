// Package postgres предоставляет реализацию репозиториев для работы с PostgreSQL.
// Unit тесты проверяют бизнес-логику и конвертацию данных без использования реальной БД.
package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
)

// TestMonitorRepository_monitorToDB тестирует конвертацию domain.Monitor в dbMonitor.
func TestMonitorRepository_monitorToDB(t *testing.T) {
	t.Parallel()
	repo := &monitorRepository{}

	t.Run("full monitor with all fields", func(t *testing.T) {
		now := time.Now()
		startTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 1, 18, 0, 0, 0, time.UTC)
		degradedThreshold := 500
		failureThreshold := 50

		monitor := &domain.Monitor{
			ID:                            uuid.New(),
			UserID:                        uuid.New(),
			Name:                          "test-monitor",
			URL:                           "https://example.com",
			CheckType:                     "HTTPS",
			IntervalSeconds:               60,
			TimeoutSeconds:                30,
			Status:                        domain.StatusUp,
			WorkingHoursStart:             &startTime,
			WorkingHoursEnd:               &endTime,
			WorkingDays:                   []time.Weekday{time.Monday, time.Wednesday, time.Friday},
			DegradedResponseTimeThreshold: &degradedThreshold,
			DegradedFailureRateThreshold:  &failureThreshold,
			LastCheckAt:                   &now,
			CreatedAt:                     now,
			UpdatedAt:                     now,
		}

		dbMon := repo.monitorToDB(monitor)

		assert.Equal(t, monitor.ID, dbMon.ID)
		assert.Equal(t, monitor.UserID, dbMon.UserID)
		assert.Equal(t, monitor.Name, dbMon.Name)
		assert.Equal(t, monitor.URL, dbMon.URL)
		assert.Equal(t, monitor.CheckType, dbMon.CheckType)
		assert.Equal(t, monitor.IntervalSeconds, dbMon.IntervalSeconds)
		assert.Equal(t, monitor.TimeoutSeconds, dbMon.TimeoutSeconds)
		assert.Equal(t, string(domain.StatusUp), dbMon.Status)
		assert.True(t, dbMon.WorkingHoursStart.Valid)
		assert.Equal(t, startTime, dbMon.WorkingHoursStart.Time)
		assert.True(t, dbMon.WorkingHoursEnd.Valid)
		assert.Equal(t, endTime, dbMon.WorkingHoursEnd.Time)
		assert.Equal(t, []string{"Monday", "Wednesday", "Friday"}, []string(dbMon.WorkingDays))
		assert.True(t, dbMon.DegradedResponseTimeThreshold.Valid)
		assert.Equal(t, int64(degradedThreshold), dbMon.DegradedResponseTimeThreshold.Int64)
		assert.True(t, dbMon.DegradedFailureRateThreshold.Valid)
		assert.Equal(t, int64(failureThreshold), dbMon.DegradedFailureRateThreshold.Int64)
		assert.True(t, dbMon.LastCheckAt.Valid)
		assert.Equal(t, now, dbMon.LastCheckAt.Time)
		assert.Equal(t, now, dbMon.CreatedAt)
		assert.Equal(t, now, dbMon.UpdatedAt)
	})

	t.Run("minimal monitor without optional fields", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			Name:            "minimal-monitor",
			URL:             "http://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 30,
			TimeoutSeconds:  10,
			Status:          domain.StatusPending,
			WorkingDays:     nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		dbMon := repo.monitorToDB(monitor)

		assert.False(t, dbMon.WorkingHoursStart.Valid)
		assert.False(t, dbMon.WorkingHoursEnd.Valid)
		assert.Nil(t, []string(dbMon.WorkingDays))
		assert.False(t, dbMon.DegradedResponseTimeThreshold.Valid)
		assert.False(t, dbMon.DegradedFailureRateThreshold.Valid)
		assert.False(t, dbMon.LastCheckAt.Valid)
	})

	t.Run("empty working days", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			Name:            "test-monitor",
			URL:             "http://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 30,
			TimeoutSeconds:  10,
			Status:          domain.StatusPending,
			WorkingDays:     []time.Weekday{},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		dbMon := repo.monitorToDB(monitor)

		assert.Nil(t, []string(dbMon.WorkingDays))
	})
}

// TestMonitorRepository_dbToMonitor тестирует конвертацию dbMonitor в domain.Monitor.
func TestMonitorRepository_dbToMonitor(t *testing.T) {
	t.Parallel()
	repo := &monitorRepository{}

	t.Run("full dbMonitor with all fields", func(t *testing.T) {
		now := time.Now()
		startTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 1, 18, 0, 0, 0, time.UTC)
		degradedThreshold := int64(500)
		failureThreshold := int64(50)

		dbMon := dbMonitor{
			ID:                            uuid.New(),
			UserID:                        uuid.New(),
			Name:                          "test-monitor",
			URL:                           "https://example.com",
			CheckType:                     "HTTPS",
			IntervalSeconds:               60,
			TimeoutSeconds:                30,
			Status:                        string(domain.StatusUp),
			WorkingHoursStart:             sql.NullTime{Time: startTime, Valid: true},
			WorkingHoursEnd:               sql.NullTime{Time: endTime, Valid: true},
			WorkingDays:                   []string{"Monday", "Wednesday", "Friday"},
			DegradedResponseTimeThreshold: sql.NullInt64{Int64: degradedThreshold, Valid: true},
			DegradedFailureRateThreshold:  sql.NullInt64{Int64: failureThreshold, Valid: true},
			LastCheckAt:                   sql.NullTime{Time: now, Valid: true},
			CreatedAt:                     now,
			UpdatedAt:                     now,
		}

		monitor, err := repo.dbToMonitor(&dbMon)

		require.NoError(t, err)
		assert.Equal(t, dbMon.ID, monitor.ID)
		assert.Equal(t, dbMon.UserID, monitor.UserID)
		assert.Equal(t, dbMon.Name, monitor.Name)
		assert.Equal(t, dbMon.URL, monitor.URL)
		assert.Equal(t, dbMon.CheckType, monitor.CheckType)
		assert.Equal(t, dbMon.IntervalSeconds, monitor.IntervalSeconds)
		assert.Equal(t, dbMon.TimeoutSeconds, monitor.TimeoutSeconds)
		assert.Equal(t, domain.StatusUp, monitor.Status)
		assert.NotNil(t, monitor.WorkingHoursStart)
		assert.Equal(t, startTime, *monitor.WorkingHoursStart)
		assert.NotNil(t, monitor.WorkingHoursEnd)
		assert.Equal(t, endTime, *monitor.WorkingHoursEnd)
		assert.Equal(t, []time.Weekday{time.Monday, time.Wednesday, time.Friday}, monitor.WorkingDays)
		assert.NotNil(t, monitor.DegradedResponseTimeThreshold)
		assert.Equal(t, int(degradedThreshold), *monitor.DegradedResponseTimeThreshold)
		assert.NotNil(t, monitor.DegradedFailureRateThreshold)
		assert.Equal(t, int(failureThreshold), *monitor.DegradedFailureRateThreshold)
		assert.NotNil(t, monitor.LastCheckAt)
		assert.Equal(t, now, *monitor.LastCheckAt)
		assert.Equal(t, now, monitor.CreatedAt)
		assert.Equal(t, now, monitor.UpdatedAt)
	})

	t.Run("minimal dbMonitor without optional fields", func(t *testing.T) {
		now := time.Now()
		dbMon := dbMonitor{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			Name:            "minimal-monitor",
			URL:             "http://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 30,
			TimeoutSeconds:  10,
			Status:          string(domain.StatusPending),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		monitor, err := repo.dbToMonitor(&dbMon)

		require.NoError(t, err)
		assert.Nil(t, monitor.WorkingHoursStart)
		assert.Nil(t, monitor.WorkingHoursEnd)
		assert.Nil(t, monitor.WorkingDays)
		assert.Nil(t, monitor.DegradedResponseTimeThreshold)
		assert.Nil(t, monitor.DegradedFailureRateThreshold)
		assert.Nil(t, monitor.LastCheckAt)
	})

	t.Run("invalid status", func(t *testing.T) {
		dbMon := dbMonitor{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			Name:            "test-monitor",
			URL:             "http://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 30,
			TimeoutSeconds:  10,
			Status:          "INVALID_STATUS",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		_, err := repo.dbToMonitor(&dbMon)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid monitor status")
	})

	t.Run("all valid statuses", func(t *testing.T) {
		validStatuses := []domain.MonitorStatus{
			domain.StatusUp,
			domain.StatusDown,
			domain.StatusDegraded,
			domain.StatusPending,
			domain.StatusPaused,
		}

		for _, status := range validStatuses {
			dbMon := dbMonitor{
				ID:              uuid.New(),
				UserID:          uuid.New(),
				Name:            "test-monitor",
				URL:             "http://example.com",
				CheckType:       "HTTP",
				IntervalSeconds: 30,
				TimeoutSeconds:  10,
				Status:          string(status),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			monitor, err := repo.dbToMonitor(&dbMon)

			require.NoError(t, err, "status %s should be valid", status)
			assert.Equal(t, status, monitor.Status)
		}
	})
}

// TestMonitorRepository_dbListToMonitors тестирует конвертацию списка dbMonitor.
func TestMonitorRepository_dbListToMonitors(t *testing.T) {
	t.Parallel()
	repo := &monitorRepository{}

	t.Run("successful conversion", func(t *testing.T) {
		now := time.Now()
		dbMonitors := []dbMonitor{
			{
				ID:              uuid.New(),
				UserID:          uuid.New(),
				Name:            "monitor-1",
				URL:             "http://example1.com",
				CheckType:       "HTTP",
				IntervalSeconds: 30,
				TimeoutSeconds:  10,
				Status:          string(domain.StatusUp),
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			{
				ID:              uuid.New(),
				UserID:          uuid.New(),
				Name:            "monitor-2",
				URL:             "http://example2.com",
				CheckType:       "HTTPS",
				IntervalSeconds: 60,
				TimeoutSeconds:  20,
				Status:          string(domain.StatusDown),
				CreatedAt:       now,
				UpdatedAt:       now,
			},
		}

		monitors, err := repo.dbListToMonitors(dbMonitors)

		require.NoError(t, err)
		assert.Len(t, monitors, 2)
		assert.Equal(t, "monitor-1", monitors[0].Name)
		assert.Equal(t, "monitor-2", monitors[1].Name)
	})

	t.Run("empty list", func(t *testing.T) {
		monitors, err := repo.dbListToMonitors([]dbMonitor{})

		require.NoError(t, err)
		assert.Len(t, monitors, 0)
	})

	t.Run("invalid status in list", func(t *testing.T) {
		dbMonitors := []dbMonitor{
			{
				ID:              uuid.New(),
				UserID:          uuid.New(),
				Name:            "valid-monitor",
				URL:             "http://example.com",
				CheckType:       "HTTP",
				IntervalSeconds: 30,
				TimeoutSeconds:  10,
				Status:          string(domain.StatusUp),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              uuid.New(),
				UserID:          uuid.New(),
				Name:            "invalid-monitor",
				URL:             "http://example.com",
				CheckType:       "HTTP",
				IntervalSeconds: 30,
				TimeoutSeconds:  10,
				Status:          "INVALID_STATUS",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
		}

		_, err := repo.dbListToMonitors(dbMonitors)

		assert.Error(t, err)
	})
}

// TestMonitorRepository_workingDaysToString тестирует конвертацию рабочих дней.
func TestMonitorRepository_workingDaysToString(t *testing.T) {
	t.Parallel()
	t.Run("all weekdays", func(t *testing.T) {
		days := []time.Weekday{
			time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
			time.Friday, time.Saturday, time.Sunday,
		}

		result := workingDaysToString(days)

		assert.Equal(t, []string{
			"Monday", "Tuesday", "Wednesday", "Thursday",
			"Friday", "Saturday", "Sunday",
		}, result)
	})

	t.Run("working days only", func(t *testing.T) {
		days := []time.Weekday{time.Monday, time.Wednesday, time.Friday}

		result := workingDaysToString(days)

		assert.Equal(t, []string{"Monday", "Wednesday", "Friday"}, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		result := workingDaysToString([]time.Weekday{})

		assert.Nil(t, result)
	})

	t.Run("nil slice", func(t *testing.T) {
		result := workingDaysToString(nil)

		assert.Nil(t, result)
	})

	t.Run("single day", func(t *testing.T) {
		days := []time.Weekday{time.Friday}

		result := workingDaysToString(days)

		assert.Equal(t, []string{"Friday"}, result)
	})
}

// TestMonitorRepository_stringToWorkingDays тестирует конвертацию строк в рабочие дни.
func TestMonitorRepository_stringToWorkingDays(t *testing.T) {
	t.Parallel()
	t.Run("all weekdays", func(t *testing.T) {
		strings := []string{
			"Monday", "Tuesday", "Wednesday", "Thursday",
			"Friday", "Saturday", "Sunday",
		}

		result := stringToWorkingDays(strings)

		assert.Equal(t, []time.Weekday{
			time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
			time.Friday, time.Saturday, time.Sunday,
		}, result)
	})

	t.Run("working days only", func(t *testing.T) {
		strings := []string{"Monday", "Wednesday", "Friday"}

		result := stringToWorkingDays(strings)

		assert.Equal(t, []time.Weekday{time.Monday, time.Wednesday, time.Friday}, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		result := stringToWorkingDays([]string{})

		assert.Nil(t, result)
	})

	t.Run("nil slice", func(t *testing.T) {
		result := stringToWorkingDays(nil)

		assert.Nil(t, result)
	})

	t.Run("single day", func(t *testing.T) {
		strings := []string{"Friday"}

		result := stringToWorkingDays(strings)

		assert.Equal(t, []time.Weekday{time.Friday}, result)
	})
}

// TestMonitorRepository_bidirectionalConversion тестирует двунаправленную конвертацию.
func TestMonitorRepository_bidirectionalConversion(t *testing.T) {
	t.Parallel()
	repo := &monitorRepository{}

	t.Run("full monitor", func(t *testing.T) {
		now := time.Now()
		startTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 1, 18, 0, 0, 0, time.UTC)
		degradedThreshold := 500
		failureThreshold := 50

		original := &domain.Monitor{
			ID:                            uuid.New(),
			UserID:                        uuid.New(),
			Name:                          "test-monitor",
			URL:                           "https://example.com",
			CheckType:                     "HTTPS",
			IntervalSeconds:               60,
			TimeoutSeconds:                30,
			Status:                        domain.StatusUp,
			WorkingHoursStart:             &startTime,
			WorkingHoursEnd:               &endTime,
			WorkingDays:                   []time.Weekday{time.Monday, time.Wednesday, time.Friday},
			DegradedResponseTimeThreshold: &degradedThreshold,
			DegradedFailureRateThreshold:  &failureThreshold,
			LastCheckAt:                   &now,
			CreatedAt:                     now,
			UpdatedAt:                     now,
		}

		dbMon := repo.monitorToDB(original)
		restored, err := repo.dbToMonitor(&dbMon)

		require.NoError(t, err)
		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.UserID, restored.UserID)
		assert.Equal(t, original.Name, restored.Name)
		assert.Equal(t, original.URL, restored.URL)
		assert.Equal(t, original.CheckType, restored.CheckType)
		assert.Equal(t, original.IntervalSeconds, restored.IntervalSeconds)
		assert.Equal(t, original.TimeoutSeconds, restored.TimeoutSeconds)
		assert.Equal(t, original.Status, restored.Status)
		assert.Equal(t, original.WorkingDays, restored.WorkingDays)
		assert.Equal(t, *original.WorkingHoursStart, *restored.WorkingHoursStart)
		assert.Equal(t, *original.WorkingHoursEnd, *restored.WorkingHoursEnd)
		assert.Equal(t, *original.DegradedResponseTimeThreshold, *restored.DegradedResponseTimeThreshold)
		assert.Equal(t, *original.DegradedFailureRateThreshold, *restored.DegradedFailureRateThreshold)
		assert.Equal(t, *original.LastCheckAt, *restored.LastCheckAt)
		assert.Equal(t, original.CreatedAt, restored.CreatedAt)
		assert.Equal(t, original.UpdatedAt, restored.UpdatedAt)
	})

	t.Run("minimal monitor", func(t *testing.T) {
		original := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			Name:            "minimal-monitor",
			URL:             "http://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 30,
			TimeoutSeconds:  10,
			Status:          domain.StatusPending,
			WorkingDays:     nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		dbMon := repo.monitorToDB(original)
		restored, err := repo.dbToMonitor(&dbMon)

		require.NoError(t, err)
		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.UserID, restored.UserID)
		assert.Equal(t, original.Name, restored.Name)
		assert.Equal(t, original.URL, restored.URL)
		assert.Equal(t, original.CheckType, restored.CheckType)
		assert.Equal(t, original.IntervalSeconds, restored.IntervalSeconds)
		assert.Equal(t, original.TimeoutSeconds, restored.TimeoutSeconds)
		assert.Equal(t, original.Status, restored.Status)
		assert.Nil(t, restored.WorkingHoursStart)
		assert.Nil(t, restored.WorkingHoursEnd)
		assert.Nil(t, restored.WorkingDays)
		assert.Nil(t, restored.DegradedResponseTimeThreshold)
		assert.Nil(t, restored.DegradedFailureRateThreshold)
		assert.Nil(t, restored.LastCheckAt)
	})
}

// Note: SQL-based tests (Create, GetByID, etc.) are skipped because sqlmock doesn't support
// PostgreSQL array types (working_days). These are covered by integration tests instead.
// The conversion logic tests above provide good coverage of the repository's business logic.
