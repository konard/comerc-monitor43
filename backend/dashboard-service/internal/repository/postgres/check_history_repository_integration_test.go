package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/dashboard-service/internal/model"
	testutil "github.com/raul/monitor/backend/dashboard-service/internal/testutil"
)

func TestCheckHistoryRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db, cleanup := testutil.SetupTestDatabase(t)
	defer cleanup()

	dbWrapper := &DB{DB: db}
	repo := NewCheckHistoryRepository(dbWrapper)

	monitorID := uuid.New()

	t.Run("Create", func(t *testing.T) {
		entry := makeCheckEntry(uuid.New(), monitorID, domain.CheckStatusUP, 200, 50.0, nil, time.Now().UTC())
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	})

	t.Run("ListByMonitorID", func(t *testing.T) {
		mid := uuid.New()
		for i := 0; i < 3; i++ {
			entry := makeCheckEntry(uuid.New(), mid, domain.CheckStatusUP, 200, float64(10+i*5), nil, time.Now().UTC())
			err := repo.Create(ctx, entry)
			require.NoError(t, err)
		}

		entries, total, err := repo.ListByMonitorID(ctx, mid.String(), domain.HistoryFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, len(entries))
		assert.Equal(t, 3, total)
	})

	t.Run("ListByMonitorID_with_status_filter", func(t *testing.T) {
		mid := uuid.New()
		entryUP := makeCheckEntry(uuid.New(), mid, domain.CheckStatusUP, 200, 50.0, nil, time.Now().UTC())
		entryDown := makeCheckEntry(uuid.New(), mid, domain.CheckStatusDOWN, 500, 0, nil, time.Now().UTC())
		entryUP2 := makeCheckEntry(uuid.New(), mid, domain.CheckStatusUP, 200, 60.0, nil, time.Now().UTC())
		for _, e := range []*domain.CheckHistoryEntry{entryUP, entryDown, entryUP2} {
			err := repo.Create(ctx, e)
			require.NoError(t, err)
		}

		entries, total, err := repo.ListByMonitorID(ctx, mid.String(), domain.HistoryFilter{
			Status: domain.CheckStatusDOWN,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(entries))
		assert.Equal(t, 1, total)
		assert.Equal(t, domain.CheckStatusDOWN, entries[0].Status)
	})

	t.Run("ListByMonitorID_with_date_range", func(t *testing.T) {
		mid := uuid.New()
		now := time.Now().UTC()
		oldEntry := makeCheckEntry(uuid.New(), mid, domain.CheckStatusUP, 200, 50.0, nil, now.Add(-48*time.Hour))
		newEntry := makeCheckEntry(uuid.New(), mid, domain.CheckStatusDOWN, 500, 0, nil, now.Add(-1*time.Hour))
		for _, e := range []*domain.CheckHistoryEntry{oldEntry, newEntry} {
			err := repo.Create(ctx, e)
			require.NoError(t, err)
		}

		start := now.Add(-24 * time.Hour)
		end := now
		entries, total, err := repo.ListByMonitorID(ctx, mid.String(), domain.HistoryFilter{
			StartDate: &start,
			EndDate:   &end,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(entries))
		assert.Equal(t, 1, total)
		assert.Equal(t, domain.CheckStatusDOWN, entries[0].Status)
	})

	t.Run("ListByMonitorID_with_pagination", func(t *testing.T) {
		mid := uuid.New()
		for i := 0; i < 5; i++ {
			entry := makeCheckEntry(uuid.New(), mid, domain.CheckStatusUP, 200, float64(i*10), nil, time.Now().UTC())
			err := repo.Create(ctx, entry)
			require.NoError(t, err)
		}

		entries, total, err := repo.ListByMonitorID(ctx, mid.String(), domain.HistoryFilter{
			PageSize: 2,
			Page:     1,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(entries), 2)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("ListByMonitorID_with_sort", func(t *testing.T) {
		mid := uuid.New()
		now := time.Now().UTC()
		for i := 0; i < 3; i++ {
			entry := makeCheckEntry(uuid.New(), mid, domain.CheckStatusUP, 200, float64(i*10), nil, now.Add(time.Duration(i)*time.Minute))
			err := repo.Create(ctx, entry)
			require.NoError(t, err)
		}

		entries, _, err := repo.ListByMonitorID(ctx, mid.String(), domain.HistoryFilter{
			SortBy:    "checked_at",
			SortOrder: "asc",
			PageSize:  10,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(entries), 3)
		assert.True(t, !entries[0].CheckedAt.After(entries[1].CheckedAt))
		assert.True(t, !entries[1].CheckedAt.After(entries[2].CheckedAt))
	})

	t.Run("GetPeriodMetrics", func(t *testing.T) {
		mid := uuid.New()
		now := time.Now().UTC()
		responseTimes := []float64{100.0, 200.0, 300.0, 400.0, 500.0, 150.0, 250.0, 350.0, 450.0, 120.0}
		statuses := []domain.CheckStatus{
			domain.CheckStatusUP, domain.CheckStatusUP, domain.CheckStatusUP,
			domain.CheckStatusDOWN, domain.CheckStatusDOWN,
			domain.CheckStatusDEGRADED, domain.CheckStatusDEGRADED,
			domain.CheckStatusUP, domain.CheckStatusUP, domain.CheckStatusUP,
		}
		for i := 0; i < 10; i++ {
			entry := makeCheckEntry(uuid.New(), mid, statuses[i], 200, responseTimes[i], nil, now.Add(time.Duration(i)*time.Minute))
			err := repo.Create(ctx, entry)
			require.NoError(t, err)
		}

		start := now.Add(-1 * time.Hour)
		end := now.Add(1 * time.Hour)
		metrics, err := repo.GetPeriodMetrics(ctx, mid.String(), start, end)
		require.NoError(t, err)
		assert.Equal(t, int64(10), metrics.TotalChecks)
		assert.Equal(t, int64(6), metrics.SuccessCount)
		assert.Equal(t, int64(2), metrics.FailedCount)
		assert.Equal(t, int64(2), metrics.DegradedCount)
		assert.InDelta(t, 60.0, metrics.UptimePercentage, 0.1)
		assert.Greater(t, metrics.P50ResponseMs, 0.0)
		assert.Greater(t, metrics.P95ResponseMs, metrics.P50ResponseMs)
		assert.GreaterOrEqual(t, metrics.P99ResponseMs, metrics.P95ResponseMs)
	})
}

func makeCheckEntry(id, monitorID uuid.UUID, status domain.CheckStatus, statusCode int, rt float64, errMsg *string, checkedAt time.Time) *domain.CheckHistoryEntry {
	return &domain.CheckHistoryEntry{
		ID:             id,
		MonitorID:      monitorID,
		Status:         status,
		StatusCode:     &statusCode,
		ResponseTimeMs: &rt,
		ErrorMessage:   errMsg,
		CheckedAt:      checkedAt,
	}
}
