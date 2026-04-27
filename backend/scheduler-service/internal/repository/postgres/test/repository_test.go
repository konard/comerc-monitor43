package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
	"github.com/raul/monitor/backend/scheduler-service/internal/repository/postgres"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test_scheduler",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
		AutoRemove: true,
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=test_scheduler sslmode=disable",
		host, port.Port())

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return db.PingContext(ctx) == nil
	}, 10*time.Second, 500*time.Millisecond, "database connection failed")

	_, err = db.ExecContext(ctx, schemaSQL)
	require.NoError(t, err)

	cleanup := func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("close test db: %v", closeErr)
		}
		if terminateErr := container.Terminate(ctx); terminateErr != nil &&
			!strings.Contains(terminateErr.Error(), "already in progress") &&
			!strings.Contains(terminateErr.Error(), "No such container") {
			t.Errorf("terminate postgres container: %v", terminateErr)
		}
	}

	return db, cleanup
}

var schemaSQL = `
CREATE TABLE IF NOT EXISTS scheduler_workers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    zone VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'IDLE',
    last_heartbeat TIMESTAMP NOT NULL DEFAULT NOW(),
    checks_completed INTEGER NOT NULL DEFAULT 0,
    checks_failed INTEGER NOT NULL DEFAULT 0,
    avg_check_duration_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduler_workers_status ON scheduler_workers(status);
CREATE INDEX IF NOT EXISTS idx_scheduler_workers_zone ON scheduler_workers(zone);
CREATE INDEX IF NOT EXISTS idx_scheduler_workers_last_heartbeat ON scheduler_workers(last_heartbeat);

CREATE TABLE IF NOT EXISTS scheduled_checks (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL,
    worker_id UUID,
    priority VARCHAR(20) NOT NULL DEFAULT 'NORMAL',
    scheduled_at TIMESTAMP NOT NULL,
    executed_at TIMESTAMP,
    completed_at TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduled_checks_monitor_id ON scheduled_checks(monitor_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_checks_worker_id ON scheduled_checks(worker_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_checks_status ON scheduled_checks(status);
CREATE INDEX IF NOT EXISTS idx_scheduled_checks_scheduled_at ON scheduled_checks(scheduled_at);

CREATE TABLE IF NOT EXISTS scheduler_audit_log (
    id UUID PRIMARY KEY,
    action VARCHAR(100) NOT NULL,
    worker_id UUID,
    check_id UUID,
    monitor_id UUID,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduler_audit_log_action ON scheduler_audit_log(action);
CREATE INDEX IF NOT EXISTS idx_scheduler_audit_log_worker_id ON scheduler_audit_log(worker_id);
CREATE INDEX IF NOT EXISTS idx_scheduler_audit_log_created_at ON scheduler_audit_log(created_at);
`

// --- WorkerRepository ---

func TestWorkerRepository_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)
	worker := model.NewWorker("worker-01", "msk")
	worker.Metadata = map[string]string{"version": "1.0.0", "os": "linux"}

	err := repo.Create(ctx, worker)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, worker.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.Name, found.Name)
	assert.Equal(t, worker.Zone, found.Zone)
	assert.Equal(t, model.WorkerStatusIdle, found.Status)
	assert.Equal(t, "1.0.0", found.Metadata["version"])
	assert.Equal(t, "linux", found.Metadata["os"])
}

func TestWorkerRepository_GetByName(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)
	worker := model.NewWorker("worker-msk-01", "msk")
	err := repo.Create(ctx, worker)
	require.NoError(t, err)

	found, err := repo.GetByName(ctx, "worker-msk-01")
	require.NoError(t, err)
	assert.Equal(t, worker.ID, found.ID)
}

func TestWorkerRepository_GetByName_not_found(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)

	_, err := repo.GetByName(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker not found")
}

func TestWorkerRepository_Create_duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)
	w1 := model.NewWorker("worker-01", "msk")
	err := repo.Create(ctx, w1)
	require.NoError(t, err)

	w2 := model.NewWorker("worker-01", "spb")
	err = repo.Create(ctx, w2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker name already exists")
}

func TestWorkerRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)
	worker := model.NewWorker("worker-01", "msk")
	err := repo.Create(ctx, worker)
	require.NoError(t, err)

	worker.UpdateHeartbeat(model.WorkerStatusBusy, 10, 2, 150.5)
	err = repo.Update(ctx, worker)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, worker.ID)
	require.NoError(t, err)
	assert.Equal(t, model.WorkerStatusBusy, found.Status)
	assert.Equal(t, 10, found.ChecksCompleted)
	assert.Equal(t, 2, found.ChecksFailed)
	assert.InDelta(t, 150.5, found.AvgCheckDurationMs, 0.01)
}

func TestWorkerRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)
	worker := model.NewWorker("worker-01", "msk")
	err := repo.Create(ctx, worker)
	require.NoError(t, err)

	err = repo.Delete(ctx, worker.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, worker.ID)
	require.Error(t, err)
}

func TestWorkerRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)

	for i := 0; i < 3; i++ {
		w := model.NewWorker(fmt.Sprintf("worker-%02d", i), "msk")
		require.NoError(t, repo.Create(ctx, w))
	}
	wOffline := model.NewWorker("worker-offline", "spb")
	wOffline.Status = model.WorkerStatusOffline
	wOffline.UpdatedAt = time.Now()
	require.NoError(t, repo.Create(ctx, wOffline))

	workers, total, err := repo.List(ctx, "", "", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, workers, 4)

	_, filteredTotal, err := repo.List(ctx, "IDLE", "", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 3, filteredTotal)

	_, zoneTotal, err := repo.List(ctx, "", "msk", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 3, zoneTotal)

	pagedWorkers, _, err := repo.List(ctx, "", "", 1, 2)
	require.NoError(t, err)
	assert.Len(t, pagedWorkers, 2)
}

func TestWorkerRepository_ListIdleByZone(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)

	wMsk := model.NewWorker("worker-msk", "msk")
	require.NoError(t, repo.Create(ctx, wMsk))

	wSpb := model.NewWorker("worker-spb", "spb")
	require.NoError(t, repo.Create(ctx, wSpb))

	wBusy := model.NewWorker("worker-busy", "msk")
	wBusy.Status = model.WorkerStatusBusy
	wBusy.UpdatedAt = time.Now()
	require.NoError(t, repo.Create(ctx, wBusy))

	idle, err := repo.ListIdleByZone(ctx, "msk")
	require.NoError(t, err)
	assert.Len(t, idle, 1)
	assert.Equal(t, "worker-msk", idle[0].Name)

	idle, err = repo.ListIdleByZone(ctx, "")
	require.NoError(t, err)
	assert.Len(t, idle, 2)
}

func TestWorkerRepository_ListExpired(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)

	wOld := model.NewWorker("worker-old", "msk")
	wOld.LastHeartbeat = time.Now().Add(-2 * time.Minute)
	wOld.CreatedAt = wOld.LastHeartbeat
	wOld.UpdatedAt = wOld.LastHeartbeat
	require.NoError(t, repo.Create(ctx, wOld))

	wFresh := model.NewWorker("worker-fresh", "msk")
	require.NoError(t, repo.Create(ctx, wFresh))

	expired, err := repo.ListExpired(ctx, 60*time.Second)
	require.NoError(t, err)
	assert.Len(t, expired, 1)
	assert.Equal(t, "worker-old", expired[0].Name)
}

func TestWorkerRepository_ListOfflineForCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewWorkerRepository(db)

	wOld := model.NewWorker("worker-old-offline", "msk")
	wOld.Status = model.WorkerStatusOffline
	wOld.LastHeartbeat = time.Now().Add(-2 * time.Hour)
	wOld.CreatedAt = wOld.LastHeartbeat
	wOld.UpdatedAt = wOld.LastHeartbeat
	require.NoError(t, repo.Create(ctx, wOld))

	wRecent := model.NewWorker("worker-recent-offline", "msk")
	wRecent.Status = model.WorkerStatusOffline
	wRecent.LastHeartbeat = time.Now().Add(-1 * time.Minute)
	wRecent.CreatedAt = wRecent.LastHeartbeat
	wRecent.UpdatedAt = wRecent.LastHeartbeat
	require.NoError(t, repo.Create(ctx, wRecent))

	offlines, err := repo.ListOfflineForCleanup(ctx, time.Now().Add(-60*time.Minute))
	require.NoError(t, err)
	assert.Len(t, offlines, 1)
	assert.Equal(t, "worker-old-offline", offlines[0].Name)
}

// --- ScheduledCheckRepository ---

func TestScheduledCheckRepository_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.PriorityNormal, time.Now())

	err := repo.Create(ctx, check)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, check.ID)
	require.NoError(t, err)
	assert.Equal(t, monitorID, found.MonitorID)
	assert.Equal(t, model.CheckStatusPending, found.Status)
	assert.Equal(t, model.PriorityNormal, found.Priority)
}

func TestScheduledCheckRepository_GetByMonitorID(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.PriorityHigh, time.Now())
	require.NoError(t, repo.Create(ctx, check))

	found, err := repo.GetByMonitorID(ctx, monitorID)
	require.NoError(t, err)
	assert.Equal(t, check.ID, found.ID)
}

func TestScheduledCheckRepository_HasPendingCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	monitorID := uuid.New()

	exists, err := repo.HasPendingCheck(ctx, monitorID)
	require.NoError(t, err)
	assert.False(t, exists)

	check := model.NewScheduledCheck(monitorID, model.PriorityNormal, time.Now())
	require.NoError(t, repo.Create(ctx, check))

	exists, err = repo.HasPendingCheck(ctx, monitorID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestScheduledCheckRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	check := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now())
	require.NoError(t, repo.Create(ctx, check))

	workerID := uuid.New()
	check.Assign(workerID)
	err := repo.Update(ctx, check)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, check.ID)
	require.NoError(t, err)
	assert.Equal(t, model.CheckStatusInProgress, found.Status)
	assert.Equal(t, workerID, found.WorkerID)
	assert.NotNil(t, found.ExecutedAt)
}

func TestScheduledCheckRepository_Complete(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	check := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now())
	workerID := uuid.New()
	check.Assign(workerID)
	require.NoError(t, repo.Create(ctx, check))
	require.NoError(t, repo.Update(ctx, check))

	check.Complete()
	err := repo.Update(ctx, check)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, check.ID)
	require.NoError(t, err)
	assert.Equal(t, model.CheckStatusCompleted, found.Status)
	assert.NotNil(t, found.CompletedAt)
}

func TestScheduledCheckRepository_Fail(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	check := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now())
	workerID := uuid.New()
	check.Assign(workerID)
	require.NoError(t, repo.Create(ctx, check))
	require.NoError(t, repo.Update(ctx, check))

	check.Fail("connection refused")
	err := repo.Update(ctx, check)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, check.ID)
	require.NoError(t, err)
	assert.Equal(t, model.CheckStatusFailed, found.Status)
	assert.Equal(t, "connection refused", found.ErrorMessage)
	assert.NotNil(t, found.CompletedAt)
}

func TestScheduledCheckRepository_ReassignByWorkerID(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	workerID := uuid.New()

	for i := 0; i < 3; i++ {
		check := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now())
		check.Assign(workerID)
		require.NoError(t, repo.Create(ctx, check))
		require.NoError(t, repo.Update(ctx, check))
	}

	count, err := repo.ReassignByWorkerID(ctx, workerID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestScheduledCheckRepository_ListByTimeRange(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	now := time.Now()

	c1 := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, now.Add(-2*time.Hour))
	c1.Complete()
	require.NoError(t, repo.Create(ctx, c1))
	require.NoError(t, repo.Update(ctx, c1))

	c2 := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, now.Add(-1*time.Hour))
	require.NoError(t, repo.Create(ctx, c2))

	checks, total, err := repo.ListByTimeRange(ctx, now.Add(-3*time.Hour), now, "", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, checks, 2)

	_, filteredCheckTotal, err := repo.ListByTimeRange(ctx, now.Add(-3*time.Hour), now, "PENDING", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, filteredCheckTotal)
}

func TestScheduledCheckRepository_DeleteCompletedBefore(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewScheduledCheckRepository(db)
	oldTime := time.Now().Add(-24 * time.Hour)

	c1 := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, oldTime)
	c1.Complete()
	oldCompletedAt := oldTime
	c1.CompletedAt = &oldCompletedAt
	require.NoError(t, repo.Create(ctx, c1))
	require.NoError(t, repo.Update(ctx, c1))

	c2 := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now())
	require.NoError(t, repo.Create(ctx, c2))

	err := repo.DeleteCompletedBefore(ctx, time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, c1.ID)
	assert.Error(t, err)

	_, err = repo.GetByID(ctx, c2.ID)
	assert.NoError(t, err)
}

// --- AuditRepository ---

func TestAuditRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewAuditRepository(db)
	workerID := uuid.New()

	err := repo.Create(ctx, "worker_registered", workerID, uuid.Nil, uuid.Nil, map[string]any{
		"worker_name": "worker-01",
		"zone":        "msk",
	})
	require.NoError(t, err)

	var action string
	var details string
	err = db.QueryRowContext(ctx,
		"SELECT action, details FROM scheduler_audit_log WHERE worker_id = $1", workerID).
		Scan(&action, &details)
	require.NoError(t, err)
	assert.Equal(t, "worker_registered", action)
	assert.Contains(t, details, "worker_name")
	assert.Contains(t, details, "worker-01")
}

func TestAuditRepository_Create_with_nil_worker(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := postgres.NewAuditRepository(db)
	monitorID := uuid.New()

	err := repo.Create(ctx, "check_completed", uuid.Nil, uuid.Nil, monitorID, map[string]any{
		"monitor_id": monitorID.String(),
	})
	require.NoError(t, err)

	var action string
	var workerID *uuid.UUID
	err = db.QueryRowContext(ctx,
		"SELECT action, worker_id FROM scheduler_audit_log WHERE monitor_id = $1", monitorID).
		Scan(&action, &workerID)
	require.NoError(t, err)
	assert.Equal(t, "check_completed", action)
	assert.Nil(t, workerID)
}

// --- DistributedLocker ---

func TestDistributedLocker_TryLock(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locker := postgres.NewDistributedLocker(db)

	acquired, unlock, err := locker.TryLock(ctx, "test-lock-1")
	require.NoError(t, err)
	assert.True(t, acquired)
	assert.NotNil(t, unlock)

	require.NoError(t, unlock())
}

func TestDistributedLocker_TryLock_contention(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locker := postgres.NewDistributedLocker(db)

	acquired1, unlock1, err := locker.TryLock(ctx, "test-lock-contention")
	require.NoError(t, err)
	assert.True(t, acquired1)

	acquired2, _, err := locker.TryLock(ctx, "test-lock-contention")
	require.NoError(t, err)
	assert.False(t, acquired2)

	require.NoError(t, unlock1())

	acquired3, unlock3, err := locker.TryLock(ctx, "test-lock-contention")
	require.NoError(t, err)
	assert.True(t, acquired3)

	require.NoError(t, unlock3())
}

func TestDistributedLocker_different_keys(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locker := postgres.NewDistributedLocker(db)

	acquired1, unlock1, err := locker.TryLock(ctx, "lock-a")
	require.NoError(t, err)
	assert.True(t, acquired1)

	acquired2, unlock2, err := locker.TryLock(ctx, "lock-b")
	require.NoError(t, err)
	assert.True(t, acquired2)

	require.NoError(t, unlock1())
	require.NoError(t, unlock2())
}
