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

func TestIncidentRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db, cleanup := testutil.SetupTestDatabase(t)
	defer cleanup()

	dbWrapper := &DB{DB: db}
	repo := NewIncidentRepository(dbWrapper)

	now := time.Now().UTC()

	t.Run("Upsert", func(t *testing.T) {
		incident := makeIncident(uuid.New(), uuid.New(), now, domain.IncidentStatusActive, 0, 0)
		err := repo.Upsert(ctx, incident)
		require.NoError(t, err)
	})

	t.Run("Upsert_Update", func(t *testing.T) {
		id := uuid.New()
		mid := uuid.New()
		startedAt := now.Add(-1 * time.Hour)
		incident := makeIncident(id, mid, startedAt, domain.IncidentStatusActive, 0, 0)
		err := repo.Upsert(ctx, incident)
		require.NoError(t, err)

		endedAt := now
		duration := int64(3600)
		incident.EndedAt = &endedAt
		incident.DurationSeconds = &duration
		incident.CheckCount = 10
		incident.Status = domain.IncidentStatusResolved
		incident.UpdatedAt = now
		err = repo.Upsert(ctx, incident)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, id.String())
		require.NoError(t, err)
		assert.Equal(t, domain.IncidentStatusResolved, found.Status)
		assert.Equal(t, 10, found.CheckCount)
		assert.NotNil(t, found.EndedAt)
		assert.NotNil(t, found.DurationSeconds)
		assert.Equal(t, int64(3600), *found.DurationSeconds)
	})

	t.Run("GetByID", func(t *testing.T) {
		id := uuid.New()
		incident := makeIncident(id, uuid.New(), now, domain.IncidentStatusActive, 5, 0)
		err := repo.Upsert(ctx, incident)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, id.String())
		require.NoError(t, err)
		assert.Equal(t, id, found.ID)
		assert.Equal(t, domain.IncidentStatusActive, found.Status)
		assert.Equal(t, 5, found.CheckCount)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New().String())
		assert.Error(t, err)
	})

	t.Run("ListByMonitorID", func(t *testing.T) {
		mid := uuid.New()
		for i := 0; i < 3; i++ {
			incident := makeIncident(uuid.New(), mid, now.Add(-time.Duration(i+1)*time.Hour), domain.IncidentStatusActive, i+1, 0)
			err := repo.Upsert(ctx, incident)
			require.NoError(t, err)
		}

		incidents, total, err := repo.ListByMonitorID(ctx, mid.String(), domain.IncidentFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, len(incidents))
		assert.Equal(t, 3, total)
	})

	t.Run("ListByMonitorID_with_date_range", func(t *testing.T) {
		mid := uuid.New()
		oldIncident := makeIncident(uuid.New(), mid, now.Add(-72*time.Hour), domain.IncidentStatusResolved, 5, 7200)
		recentIncident := makeIncident(uuid.New(), mid, now.Add(-2*time.Hour), domain.IncidentStatusActive, 3, 0)
		for _, inc := range []*domain.Incident{oldIncident, recentIncident} {
			err := repo.Upsert(ctx, inc)
			require.NoError(t, err)
		}

		start := now.Add(-48 * time.Hour)
		end := now
		incidents, total, err := repo.ListByMonitorID(ctx, mid.String(), domain.IncidentFilter{
			StartDate: &start,
			EndDate:   &end,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(incidents))
		assert.Equal(t, 1, total)
		assert.Equal(t, domain.IncidentStatusActive, incidents[0].Status)
	})

	t.Run("ListByMonitorID_with_pagination", func(t *testing.T) {
		mid := uuid.New()
		for i := 0; i < 5; i++ {
			incident := makeIncident(uuid.New(), mid, now.Add(-time.Duration(i+1)*time.Hour), domain.IncidentStatusActive, i+1, 0)
			err := repo.Upsert(ctx, incident)
			require.NoError(t, err)
		}

		incidents, total, err := repo.ListByMonitorID(ctx, mid.String(), domain.IncidentFilter{
			PageSize: 2,
			Page:     1,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(incidents), 2)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("Update", func(t *testing.T) {
		id := uuid.New()
		mid := uuid.New()
		incident := makeIncident(id, mid, now.Add(-1*time.Hour), domain.IncidentStatusActive, 3, 0)
		err := repo.Upsert(ctx, incident)
		require.NoError(t, err)

		endedAt := now
		duration := int64(3600)
		incident.EndedAt = &endedAt
		incident.DurationSeconds = &duration
		incident.CheckCount = 15
		incident.Status = domain.IncidentStatusResolved
		incident.UpdatedAt = now
		err = repo.Update(ctx, incident)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, id.String())
		require.NoError(t, err)
		assert.Equal(t, domain.IncidentStatusResolved, found.Status)
		assert.Equal(t, 15, found.CheckCount)
		assert.Equal(t, int64(3600), *found.DurationSeconds)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		incident := makeIncident(uuid.New(), uuid.New(), now, domain.IncidentStatusResolved, 0, 3600)
		err := repo.Update(ctx, incident)
		assert.Equal(t, domain.ErrIncidentNotFound, err)
	})
}

func makeIncident(id, monitorID uuid.UUID, startedAt time.Time, status domain.IncidentStatus, checkCount int, duration int64) *domain.Incident {
	inc := &domain.Incident{
		ID:         id,
		MonitorID:  monitorID,
		StartedAt:  startedAt,
		CheckCount: checkCount,
		Status:     status,
		CreatedAt:  startedAt,
		UpdatedAt:  startedAt,
	}
	if duration > 0 {
		endedAt := startedAt.Add(time.Duration(duration) * time.Second)
		inc.EndedAt = &endedAt
		inc.DurationSeconds = &duration
	}
	return inc
}
