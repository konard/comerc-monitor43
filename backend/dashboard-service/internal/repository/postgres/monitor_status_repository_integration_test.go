package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/dashboard-service/internal/model"
	testutil "github.com/raul/monitor/backend/dashboard-service/internal/testutil"
)

func TestMonitorStatusRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db, cleanup := testutil.SetupTestDatabase(t)
	defer cleanup()

	dbWrapper := &DB{DB: db}
	repo := NewMonitorStatusRepository(dbWrapper)

	now := time.Now().UTC()
	rt := 123.45

	t.Run("Upsert", func(t *testing.T) {
		status := &domain.MonitorStatusView{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Name:               "test-monitor",
			URL:                "https://example.com",
			Status:             domain.MonitorStatusUP,
			UptimePercentage:   99.5,
			LastCheckedAt:      &now,
			LastResponseTimeMs: &rt,
			Tags:               "api,prod",
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		err := repo.Upsert(ctx, status)
		require.NoError(t, err)
	})

	t.Run("GetByID", func(t *testing.T) {
		id := uuid.New()
		status := makeMonitorStatus(id, uuid.New(), "getbyid-test", domain.MonitorStatusUP, 99.0, now, &rt)
		err := repo.Upsert(ctx, status)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, id.String())
		require.NoError(t, err)
		assert.Equal(t, id, found.ID)
		assert.Equal(t, "getbyid-test", found.Name)
		assert.Equal(t, domain.MonitorStatusUP, found.Status)
		assert.InDelta(t, 99.0, found.UptimePercentage, 0.01)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New().String())
		assert.Error(t, err)
	})

	t.Run("Upsert_Update", func(t *testing.T) {
		id := uuid.New()
		uid := uuid.New()
		status := makeMonitorStatus(id, uid, "update-test", domain.MonitorStatusUP, 95.0, now, nil)
		err := repo.Upsert(ctx, status)
		require.NoError(t, err)

		newRt := 250.0
		status.Status = domain.MonitorStatusDOWN
		status.UptimePercentage = 85.0
		status.LastResponseTimeMs = &newRt
		status.UpdatedAt = time.Now().UTC()
		err = repo.Upsert(ctx, status)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, id.String())
		require.NoError(t, err)
		assert.Equal(t, domain.MonitorStatusDOWN, found.Status)
		assert.InDelta(t, 85.0, found.UptimePercentage, 0.01)
		assert.InDelta(t, 250.0, *found.LastResponseTimeMs, 0.01)
	})

	t.Run("Delete", func(t *testing.T) {
		id := uuid.New()
		status := makeMonitorStatus(id, uuid.New(), "delete-test", domain.MonitorStatusUP, 100.0, now, nil)
		err := repo.Upsert(ctx, status)
		require.NoError(t, err)

		err = repo.Delete(ctx, id.String())
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, id.String())
		assert.Error(t, err)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New().String())
		assert.Equal(t, domain.ErrMonitorNotFound, err)
	})

	t.Run("ListByUserID", func(t *testing.T) {
		userID := uuid.New()
		statuses := []domain.MonitorStatus{domain.MonitorStatusUP, domain.MonitorStatusDOWN, domain.MonitorStatusDEGRADED, domain.MonitorStatusUP, domain.MonitorStatusPAUSED}
		for i := 0; i < 5; i++ {
			status := makeMonitorStatus(uuid.New(), userID, fmt.Sprintf("list-monitor-%d", i), statuses[i], float64(80+i*4), now, nil)
			err := repo.Upsert(ctx, status)
			require.NoError(t, err)
		}

		list, total, err := repo.ListByUserID(ctx, userID.String(), domain.DashboardFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 5)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("ListByUserID_with_status_filter", func(t *testing.T) {
		userID := uuid.New()
		for _, s := range []domain.MonitorStatus{domain.MonitorStatusUP, domain.MonitorStatusDOWN, domain.MonitorStatusUP} {
			status := makeMonitorStatus(uuid.New(), userID, "filter-status-"+uuid.New().String()[:8], s, 90.0, now, nil)
			err := repo.Upsert(ctx, status)
			require.NoError(t, err)
		}

		list, total, err := repo.ListByUserID(ctx, userID.String(), domain.DashboardFilter{
			Statuses: []domain.MonitorStatus{domain.MonitorStatusUP},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(list))
		assert.Equal(t, 2, total)
		for _, s := range list {
			assert.Equal(t, domain.MonitorStatusUP, s.Status)
		}
	})

	t.Run("ListByUserID_with_search", func(t *testing.T) {
		userID := uuid.New()
		status1 := makeMonitorStatus(uuid.New(), userID, "SearchableMonitorAlpha", domain.MonitorStatusUP, 99.0, now, nil)
		status2 := makeMonitorStatus(uuid.New(), userID, "SearchableMonitorBeta", domain.MonitorStatusUP, 99.0, now, nil)
		status3 := makeMonitorStatus(uuid.New(), userID, "OtherNameGamma", domain.MonitorStatusUP, 99.0, now, nil)
		for _, s := range []*domain.MonitorStatusView{status1, status2, status3} {
			err := repo.Upsert(ctx, s)
			require.NoError(t, err)
		}

		list, total, err := repo.ListByUserID(ctx, userID.String(), domain.DashboardFilter{
			Search: "Searchable",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, list, 2)
	})

	t.Run("ListByUserID_with_pagination", func(t *testing.T) {
		userID := uuid.New()
		for i := 0; i < 5; i++ {
			status := makeMonitorStatus(uuid.New(), userID, fmt.Sprintf("pag-monitor-%d", i), domain.MonitorStatusUP, 99.0, now, nil)
			err := repo.Upsert(ctx, status)
			require.NoError(t, err)
		}

		list, total, err := repo.ListByUserID(ctx, userID.String(), domain.DashboardFilter{
			PageSize: 2,
			Page:     1,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(list), 2)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("ListByUserID_default_sort_priority", func(t *testing.T) {
		userID := uuid.New()
		for _, s := range []domain.MonitorStatus{domain.MonitorStatusPAUSED, domain.MonitorStatusDOWN, domain.MonitorStatusUP, domain.MonitorStatusDEGRADED} {
			status := makeMonitorStatus(uuid.New(), userID, "sort-"+uuid.New().String()[:8], s, 99.0, now, nil)
			err := repo.Upsert(ctx, status)
			require.NoError(t, err)
		}

		list, _, err := repo.ListByUserID(ctx, userID.String(), domain.DashboardFilter{
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 4)
		if len(list) >= 4 {
			assert.Equal(t, domain.MonitorStatusDOWN, list[0].Status)
			assert.Equal(t, domain.MonitorStatusDEGRADED, list[1].Status)
		}
	})

	t.Run("GetOverallUptime", func(t *testing.T) {
		userID := uuid.New()
		for i := 0; i < 3; i++ {
			status := makeMonitorStatus(uuid.New(), userID, "uptime-"+uuid.New().String()[:8], domain.MonitorStatusUP, float64(90+i*3), now, nil)
			err := repo.Upsert(ctx, status)
			require.NoError(t, err)
		}

		uptime, err := repo.GetOverallUptime(ctx, userID.String(), nil)
		require.NoError(t, err)
		assert.Greater(t, uptime, 0.0)
	})
}

func makeMonitorStatus(id, userID uuid.UUID, name string, status domain.MonitorStatus, uptime float64, now time.Time, rt *float64) *domain.MonitorStatusView {
	return &domain.MonitorStatusView{
		ID:                 id,
		UserID:             userID,
		Name:               name,
		URL:                "https://" + name + ".com",
		Status:             status,
		UptimePercentage:   uptime,
		LastCheckedAt:      &now,
		LastResponseTimeMs: rt,
		Tags:               "",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
