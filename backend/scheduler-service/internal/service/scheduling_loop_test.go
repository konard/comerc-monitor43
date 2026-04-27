package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
	"github.com/raul/monitor/backend/scheduler-service/internal/monitor"
)

// mockMonitorClientForLoop — мок MonitorClient для тестов SchedulingLoop.
type mockMonitorClientForLoop struct {
	mock.Mock
}

func (m *mockMonitorClientForLoop) GetMonitor(ctx context.Context, monitorID string) (*monitor.MonitorInfo, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitor.MonitorInfo), args.Error(1)
}

func (m *mockMonitorClientForLoop) ListActiveMonitors(ctx context.Context, pageSize int) ([]*monitor.MonitorInfo, error) {
	args := m.Called(ctx, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*monitor.MonitorInfo), args.Error(1)
}

func (m *mockMonitorClientForLoop) IsMonitorPaused(ctx context.Context, monitorID string) (bool, error) {
	args := m.Called(ctx, monitorID)
	return args.Bool(0), args.Error(1)
}

// newTestSchedulingLoop создаёт экземпляры SchedulingLoop и всех его зависимостей для тестов.
func newTestSchedulingLoop(t *testing.T, interval time.Duration) (
	*SchedulingLoop,
	*CheckScheduler,
	*WorkerService,
	*mockMonitorClientForLoop,
	*mockWorkerRepo,
	*mockCheckRepo,
	*mockAuditRepo,
) {
	t.Helper()

	workerRepo := &mockWorkerRepo{}
	checkRepo := &mockCheckRepo{}
	auditRepo := &mockAuditRepo{}
	monitorClient := &mockMonitorClientForLoop{}
	logger := slog.New(slog.NewTextHandler(nopWriter{}, nil))

	scheduler := NewCheckScheduler(checkRepo, workerRepo, auditRepo, monitorClient, 2*time.Minute, logger)
	workerSvc := NewWorkerService(workerRepo, checkRepo, auditRepo, 30*time.Second, 10*time.Minute, logger)

	loop := NewSchedulingLoop(scheduler, workerSvc, monitorClient, nil, interval, logger)

	return loop, scheduler, workerSvc, monitorClient, workerRepo, checkRepo, auditRepo
}

// nopWriter поглощает лог-вывод в тестах.
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

// ---------------------------------------------------------------------------
// NewSchedulingLoop / IsRunning
// ---------------------------------------------------------------------------

func TestNewSchedulingLoop_initialState(t *testing.T) {
	t.Parallel()

	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 100*time.Millisecond)

	assert.False(t, loop.IsRunning())
}

// ---------------------------------------------------------------------------
// Start / Stop / IsRunning
// ---------------------------------------------------------------------------

func TestSchedulingLoop_Start_Stop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	loop.Start(ctx)
	assert.True(t, loop.IsRunning())

	loop.Stop()
	assert.False(t, loop.IsRunning())
}

func TestSchedulingLoop_Start_idempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	loop.Start(ctx)
	loop.Start(ctx) // второй вызов не должен паниковать
	assert.True(t, loop.IsRunning())
	loop.Stop()
}

func TestSchedulingLoop_Stop_when_not_running(t *testing.T) {
	t.Parallel()

	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	// Stop на неработающем цикле не должен паниковать
	loop.Stop()
	assert.False(t, loop.IsRunning())
}

func TestSchedulingLoop_Stop_via_context(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	loop.Start(ctx)
	assert.True(t, loop.IsRunning())

	cancel()
	// Даём горутине время завершиться
	time.Sleep(50 * time.Millisecond)
	// После отмены контекста внутренняя горутина завершается;
	// IsRunning всё ещё true (Stop не вызван), но wg.Wait() не блокирует.
	loop.Stop()
}

// ---------------------------------------------------------------------------
// scheduleChecks через withLock(locker==nil)
// ---------------------------------------------------------------------------

func TestSchedulingLoop_scheduleChecks_success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, monitorClient, workerRepo, checkRepo, _ := newTestSchedulingLoop(t, 10*time.Second)

	monID := uuid.New()
	// монитор, у которого lastCheckedAt = zero => нужна проверка
	mon := &monitor.MonitorInfo{
		ID:              monID.String(),
		IntervalSeconds: 60,
	}
	monitorClient.On("ListActiveMonitors", mock.Anything, 100).Return([]*monitor.MonitorInfo{mon}, nil)
	checkRepo.On("HasPendingCheck", mock.Anything, monID).Return(false, nil)
	workerRepo.On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{}, nil)
	checkRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	loop.scheduleChecks(ctx)

	checkRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestSchedulingLoop_scheduleChecks_monitor_not_due(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, monitorClient, _, checkRepo, _ := newTestSchedulingLoop(t, 10*time.Second)

	// монитор был проверен только что => следующая проверка через 60 секунд
	mon := &monitor.MonitorInfo{
		ID:              uuid.New().String(),
		IntervalSeconds: 60,
		LastCheckedAt:   time.Now(),
	}
	monitorClient.On("ListActiveMonitors", mock.Anything, 100).Return([]*monitor.MonitorInfo{mon}, nil)

	loop.scheduleChecks(ctx)

	checkRepo.AssertNotCalled(t, "Create")
}

func TestSchedulingLoop_scheduleChecks_list_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, monitorClient, _, checkRepo, _ := newTestSchedulingLoop(t, 10*time.Second)

	monitorClient.On("ListActiveMonitors", mock.Anything, 100).Return(nil, errors.New("db error"))

	// не должен паниковать
	loop.scheduleChecks(ctx)

	checkRepo.AssertNotCalled(t, "Create")
}

func TestSchedulingLoop_scheduleChecks_invalid_monitor_id(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, monitorClient, _, checkRepo, _ := newTestSchedulingLoop(t, 10*time.Second)

	mon := &monitor.MonitorInfo{
		ID:              "not-a-uuid",
		IntervalSeconds: 60,
	}
	monitorClient.On("ListActiveMonitors", mock.Anything, 100).Return([]*monitor.MonitorInfo{mon}, nil)

	loop.scheduleChecks(ctx)

	checkRepo.AssertNotCalled(t, "Create")
}

func TestSchedulingLoop_scheduleChecks_already_pending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, monitorClient, _, checkRepo, _ := newTestSchedulingLoop(t, 10*time.Second)

	monID := uuid.New()
	mon := &monitor.MonitorInfo{
		ID:              monID.String(),
		IntervalSeconds: 60,
	}
	monitorClient.On("ListActiveMonitors", mock.Anything, 100).Return([]*monitor.MonitorInfo{mon}, nil)
	checkRepo.On("HasPendingCheck", mock.Anything, monID).Return(true, nil)

	loop.scheduleChecks(ctx)

	// Create не вызывается — check already pending
	checkRepo.AssertNotCalled(t, "Create")
}

func TestSchedulingLoop_scheduleChecks_schedule_error_non_scheduler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, monitorClient, _, checkRepo, _ := newTestSchedulingLoop(t, 10*time.Second)

	monID := uuid.New()
	mon := &monitor.MonitorInfo{
		ID:              monID.String(),
		IntervalSeconds: 60,
	}
	monitorClient.On("ListActiveMonitors", mock.Anything, 100).Return([]*monitor.MonitorInfo{mon}, nil)
	checkRepo.On("HasPendingCheck", mock.Anything, monID).Return(false, nil)
	// не scheduler-ошибка — обычная ошибка БД
	checkRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db write error"))
	// ListIdleByZone нужен для ScheduleCheck
	loop.scheduler.workerRepo.(*mockWorkerRepo).On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{}, nil)

	loop.scheduleChecks(ctx)
	// проверяем что не паниковало — Create был вызван
	checkRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// detectOfflineAndReassign
// ---------------------------------------------------------------------------

func TestSchedulingLoop_detectOfflineAndReassign_success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, workerRepo, checkRepo, auditRepo := newTestSchedulingLoop(t, 10*time.Second)

	w := model.NewWorker("worker-01", "msk")
	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{w}, nil)
	checkRepo.On("ReassignByWorkerID", mock.Anything, w.ID).Return(1, nil)
	workerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// ReassignOverdueChecks
	checkRepo.On("ListOverdue", mock.Anything, mock.Anything).Return([]*model.ScheduledCheck{}, nil)

	loop.detectOfflineAndReassign(ctx)

	workerRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestSchedulingLoop_detectOfflineAndReassign_offline_detect_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, workerRepo, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{}, errors.New("db error"))

	// не должен паниковать
	loop.detectOfflineAndReassign(ctx)
}

func TestSchedulingLoop_detectOfflineAndReassign_reassign_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, workerRepo, checkRepo, _ := newTestSchedulingLoop(t, 10*time.Second)

	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{}, nil)
	checkRepo.On("ListOverdue", mock.Anything, mock.Anything).Return([]*model.ScheduledCheck{}, errors.New("overdue error"))

	// не должен паниковать
	loop.detectOfflineAndReassign(ctx)
}

// ---------------------------------------------------------------------------
// cleanupOfflineWorkers
// ---------------------------------------------------------------------------

func TestSchedulingLoop_cleanupOfflineWorkers_success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, workerRepo, _, auditRepo := newTestSchedulingLoop(t, 10*time.Second)

	w := model.NewWorker("worker-stale-01", "msk")
	workerRepo.On("ListOfflineForCleanup", mock.Anything, mock.Anything).Return([]*model.Worker{w}, nil)
	workerRepo.On("Delete", mock.Anything, w.ID).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	loop.cleanupOfflineWorkers(ctx)

	workerRepo.AssertCalled(t, "Delete", mock.Anything, w.ID)
}

func TestSchedulingLoop_cleanupOfflineWorkers_list_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, workerRepo, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	workerRepo.On("ListOfflineForCleanup", mock.Anything, mock.Anything).Return([]*model.Worker{}, errors.New("db error"))

	// не должен паниковать
	loop.cleanupOfflineWorkers(ctx)
}

func TestSchedulingLoop_cleanupOfflineWorkers_delete_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, workerRepo, _, auditRepo := newTestSchedulingLoop(t, 10*time.Second)

	w := model.NewWorker("worker-stale-01", "msk")
	workerRepo.On("ListOfflineForCleanup", mock.Anything, mock.Anything).Return([]*model.Worker{w}, nil)
	workerRepo.On("Delete", mock.Anything, w.ID).Return(errors.New("delete error"))
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	loop.cleanupOfflineWorkers(ctx)

	// Delete был вызван, ошибка поглощена
	workerRepo.AssertCalled(t, "Delete", mock.Anything, w.ID)
}

// ---------------------------------------------------------------------------
// withLock
// ---------------------------------------------------------------------------

func TestSchedulingLoop_withLock_no_locker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)
	// locker == nil — fn должна быть вызвана напрямую

	called := false
	loop.withLock(ctx, "test-key", func(_ context.Context) {
		called = true
	})

	assert.True(t, called)
}

func TestSchedulingLoop_withLock_acquired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	locker := &mockDistributedLocker{}
	loop.locker = locker
	locker.On("TryLock", mock.Anything, "scheduler:test").Return(true, nil, nil)

	called := false
	loop.withLock(ctx, "test", func(_ context.Context) {
		called = true
	})

	assert.True(t, called)
}

func TestSchedulingLoop_withLock_not_acquired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	locker := &mockDistributedLocker{}
	loop.locker = locker
	locker.On("TryLock", mock.Anything, "scheduler:test").Return(false, nil, nil)

	called := false
	loop.withLock(ctx, "test", func(_ context.Context) {
		called = true
	})

	assert.False(t, called)
}

func TestSchedulingLoop_withLock_lock_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	locker := &mockDistributedLocker{}
	loop.locker = locker
	locker.On("TryLock", mock.Anything, "scheduler:test").Return(false, nil, errors.New("redis error"))

	called := false
	loop.withLock(ctx, "test", func(_ context.Context) {
		called = true
	})

	assert.False(t, called)
}

func TestSchedulingLoop_withLock_unlock_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	loop, _, _, _, _, _, _ := newTestSchedulingLoop(t, 10*time.Second)

	locker := &mockDistributedLocker{}
	loop.locker = locker
	unlockFn := func() error { return errors.New("unlock failed") }
	locker.On("TryLock", mock.Anything, "scheduler:test").Return(true, unlockFn, nil)

	called := false
	loop.withLock(ctx, "test", func(_ context.Context) {
		called = true
	})

	// fn всё равно вызывается, ошибка unlock логируется, не паникует
	assert.True(t, called)
}

// ---------------------------------------------------------------------------
// run — интеграционный тест с synctest (виртуальные часы)
// ---------------------------------------------------------------------------

func TestSchedulingLoop_run_schedulingTick(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		workerRepo := &mockWorkerRepo{}
		checkRepo := &mockCheckRepo{}
		auditRepo := &mockAuditRepo{}
		monitorClient := &mockMonitorClientForLoop{}
		logger := slog.New(slog.NewTextHandler(nopWriter{}, nil))

		scheduler := NewCheckScheduler(checkRepo, workerRepo, auditRepo, monitorClient, 2*time.Minute, logger)
		workerSvc := NewWorkerService(workerRepo, checkRepo, auditRepo, 30*time.Second, 10*time.Minute, logger)
		interval := 1 * time.Second

		loop := NewSchedulingLoop(scheduler, workerSvc, monitorClient, nil, interval, logger)

		monID := uuid.New()
		mon := &monitor.MonitorInfo{
			ID:              monID.String(),
			IntervalSeconds: 60,
		}
		monitorClient.On("ListActiveMonitors", mock.Anything, 100).Return([]*monitor.MonitorInfo{mon}, nil)
		checkRepo.On("HasPendingCheck", mock.Anything, monID).Return(false, nil)
		workerRepo.On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{}, nil)
		checkRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		ctx, cancel := context.WithCancel(t.Context())

		loop.Start(ctx)
		synctest.Wait()

		// продвигаем виртуальное время на 1 тик
		time.Sleep(1 * time.Second)
		synctest.Wait()

		cancel()
		loop.Stop()

		checkRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestSchedulingLoop_run_cancelContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		workerRepo := &mockWorkerRepo{}
		checkRepo := &mockCheckRepo{}
		auditRepo := &mockAuditRepo{}
		monitorClient := &mockMonitorClientForLoop{}
		logger := slog.New(slog.NewTextHandler(nopWriter{}, nil))

		scheduler := NewCheckScheduler(checkRepo, workerRepo, auditRepo, monitorClient, 2*time.Minute, logger)
		workerSvc := NewWorkerService(workerRepo, checkRepo, auditRepo, 30*time.Second, 10*time.Minute, logger)

		loop := NewSchedulingLoop(scheduler, workerSvc, monitorClient, nil, 10*time.Minute, logger)

		ctx, cancel := context.WithCancel(t.Context())

		loop.Start(ctx)
		synctest.Wait()

		// отмена контекста завершает внутреннюю горутину
		cancel()
		synctest.Wait()

		loop.Stop()

		assert.False(t, loop.IsRunning())
	})
}

// ---------------------------------------------------------------------------
// WorkerService — непокрытые методы
// ---------------------------------------------------------------------------

func TestListWorkers(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workers := []*model.Worker{model.NewWorker("w1", "msk"), model.NewWorker("w2", "spb")}
	workerRepo.On("List", mock.Anything, "IDLE", "msk", 1, 20).Return(workers, 2, nil)

	result, total, err := svc.ListWorkers(context.Background(), "IDLE", "msk", 1, 20)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, result, 2)
}

func TestListWorkers_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workerRepo.On("List", mock.Anything, "", "", 1, 10).Return([]*model.Worker{}, 0, errors.New("db error"))

	_, _, err := svc.ListWorkers(context.Background(), "", "", 1, 10)

	assert.Error(t, err)
}

func TestDetectOfflineWorkers_no_expired(t *testing.T) {
	t.Parallel()

	svc, workerRepo, checkRepo, _ := newTestWorkerService(t)
	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{}, nil)
	// ReassignOverdueChecks не вызывается в DetectOfflineWorkers напрямую

	count, err := svc.DetectOfflineWorkers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	checkRepo.AssertNotCalled(t, "ReassignByWorkerID")
}

func TestDetectOfflineWorkers_marks_offline(t *testing.T) {
	t.Parallel()

	svc, workerRepo, checkRepo, auditRepo := newTestWorkerService(t)
	w := model.NewWorker("worker-dead", "msk")

	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{w}, nil)
	checkRepo.On("ReassignByWorkerID", mock.Anything, w.ID).Return(3, nil)
	workerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	count, err := svc.DetectOfflineWorkers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, model.WorkerStatusOffline, w.Status)
}

func TestDetectOfflineWorkers_list_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{}, errors.New("db error"))

	_, err := svc.DetectOfflineWorkers(context.Background())

	assert.Error(t, err)
}

func TestDetectOfflineWorkers_reassign_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, checkRepo, auditRepo := newTestWorkerService(t)
	w := model.NewWorker("worker-dead", "msk")

	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{w}, nil)
	checkRepo.On("ReassignByWorkerID", mock.Anything, w.ID).Return(0, errors.New("reassign error"))
	workerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// ошибка ReassignByWorkerID логируется, но не прерывает обработку
	count, err := svc.DetectOfflineWorkers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDetectOfflineWorkers_update_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, checkRepo, _ := newTestWorkerService(t)
	w := model.NewWorker("worker-dead", "msk")

	workerRepo.On("ListExpired", mock.Anything, 30*time.Second).Return([]*model.Worker{w}, nil)
	checkRepo.On("ReassignByWorkerID", mock.Anything, w.ID).Return(0, nil)
	workerRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update error"))

	// Update-ошибка логируется через continue — count остаётся 0
	count, err := svc.DetectOfflineWorkers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCleanupOfflineWorkers_success(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, auditRepo := newTestWorkerService(t)
	w := model.NewWorker("worker-stale", "msk")

	workerRepo.On("ListOfflineForCleanup", mock.Anything, mock.Anything).Return([]*model.Worker{w}, nil)
	workerRepo.On("Delete", mock.Anything, w.ID).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	count, err := svc.CleanupOfflineWorkers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCleanupOfflineWorkers_list_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workerRepo.On("ListOfflineForCleanup", mock.Anything, mock.Anything).Return([]*model.Worker{}, errors.New("db error"))

	_, err := svc.CleanupOfflineWorkers(context.Background())

	assert.Error(t, err)
}

func TestCleanupOfflineWorkers_delete_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	w := model.NewWorker("worker-stale", "msk")

	workerRepo.On("ListOfflineForCleanup", mock.Anything, mock.Anything).Return([]*model.Worker{w}, nil)
	workerRepo.On("Delete", mock.Anything, w.ID).Return(errors.New("delete error"))

	// ошибка поглощается через continue
	count, err := svc.CleanupOfflineWorkers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCleanupOfflineWorkers_empty(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workerRepo.On("ListOfflineForCleanup", mock.Anything, mock.Anything).Return([]*model.Worker{}, nil)

	count, err := svc.CleanupOfflineWorkers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRegisterWorker_create_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workerRepo.On("GetByName", mock.Anything, "worker-01").Return(nil, errors.New("not found"))
	workerRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db write error"))

	_, err := svc.RegisterWorker(context.Background(), "worker-01", "msk", nil)

	assert.Error(t, err)
}

func TestUnregisterWorker_reassign_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, checkRepo, auditRepo := newTestWorkerService(t)
	w := model.NewWorker("worker-01", "msk")

	workerRepo.On("GetByID", mock.Anything, w.ID).Return(w, nil)
	checkRepo.On("ReassignByWorkerID", mock.Anything, w.ID).Return(0, errors.New("reassign error"))
	workerRepo.On("Delete", mock.Anything, w.ID).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// ошибка reassign логируется, но Delete продолжается
	err := svc.UnregisterWorker(context.Background(), w.ID)

	require.NoError(t, err)
}

func TestHeartbeat_update_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	w := model.NewWorker("worker-01", "msk")

	workerRepo.On("GetByID", mock.Anything, w.ID).Return(w, nil)
	workerRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update error"))

	err := svc.Heartbeat(context.Background(), w.ID, model.WorkerStatusBusy, 10, 0, 100.0)

	assert.Error(t, err)
}

func TestSelectWorker_zone_list_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workerRepo.On("ListIdleByZone", mock.Anything, "msk").Return([]*model.Worker{}, errors.New("db error"))

	_, err := svc.SelectWorker(context.Background(), "msk")

	assert.Error(t, err)
}

func TestSelectWorker_any_zone_list_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, _ := newTestWorkerService(t)
	workerRepo.On("ListIdleByZone", mock.Anything, "msk").Return([]*model.Worker{}, nil)
	workerRepo.On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{}, errors.New("db error"))

	_, err := svc.SelectWorker(context.Background(), "msk")

	assert.Error(t, err)
}

func TestCreateAuditLog_error_is_logged(t *testing.T) {
	t.Parallel()

	svc, workerRepo, _, auditRepo := newTestWorkerService(t)
	workerRepo.On("GetByName", mock.Anything, "worker-01").Return(nil, errors.New("not found"))
	workerRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	// auditRepo.Create возвращает ошибку — она логируется, но не прерывает вызов
	auditRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("audit error"))

	worker, err := svc.RegisterWorker(context.Background(), "worker-01", "msk", nil)

	// ошибка audit не влияет на результат
	require.NoError(t, err)
	assert.Equal(t, "worker-01", worker.Name)
}

func TestUnregisterWorker_delete_error(t *testing.T) {
	t.Parallel()

	svc, workerRepo, checkRepo, _ := newTestWorkerService(t)
	w := model.NewWorker("worker-01", "msk")

	workerRepo.On("GetByID", mock.Anything, w.ID).Return(w, nil)
	checkRepo.On("ReassignByWorkerID", mock.Anything, w.ID).Return(0, nil)
	workerRepo.On("Delete", mock.Anything, w.ID).Return(errors.New("delete error"))

	err := svc.UnregisterWorker(context.Background(), w.ID)

	assert.Error(t, err)
}

func TestScheduleCheck_check_pending_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, nil, nil, nil, 2*time.Minute, logger)

	checkRepo.On("HasPendingCheck", mock.Anything, mock.Anything).Return(false, errors.New("db error"))

	_, err := svc.ScheduleCheck(ctx, uuid.New(), model.PriorityNormal, time.Now())

	assert.Error(t, err)
}

func TestCompleteCheck_not_found(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, nil, nil, nil, 2*time.Minute, logger)

	checkRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	err := svc.CompleteCheck(ctx, uuid.New(), false, "")

	assert.ErrorIs(t, err, model.ErrCheckNotFound)
}

func TestCompleteCheck_update_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	check := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now())
	checkRepo := &mockCheckRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, nil, nil, nil, 2*time.Minute, logger)

	checkRepo.On("GetByID", mock.Anything, check.ID).Return(check, nil)
	checkRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := svc.CompleteCheck(ctx, check.ID, false, "")

	assert.Error(t, err)
}

func TestReassignOverdueChecks_list_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, nil, nil, nil, 2*time.Minute, logger)

	checkRepo.On("ListOverdue", mock.Anything, mock.Anything).Return([]*model.ScheduledCheck{}, errors.New("db error"))

	_, err := svc.ReassignOverdueChecks(ctx)

	assert.Error(t, err)
}

func TestReassignOverdueChecks_update_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	check := model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now().Add(-5*time.Minute))
	checkRepo := &mockCheckRepo{}
	worker := model.NewWorker("worker-01", "msk")
	workerRepo := &mockWorkerRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, nil, nil, 2*time.Minute, logger)

	checkRepo.On("ListOverdue", mock.Anything, mock.Anything).Return([]*model.ScheduledCheck{check}, nil)
	workerRepo.On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{worker}, nil)
	checkRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	count, err := svc.ReassignOverdueChecks(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTriggerScheduledCheck_no_monitor_client(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	logger := slog.Default()
	// monitorClient == nil
	svc := NewCheckScheduler(checkRepo, workerRepo, nil, nil, 2*time.Minute, logger)

	monitorID := uuid.New()
	checkRepo.On("HasPendingCheck", mock.Anything, monitorID).Return(false, nil)
	workerRepo.On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{}, nil)
	checkRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	result, err := svc.TriggerScheduledCheck(ctx, monitorID, model.PriorityNormal)

	require.NoError(t, err)
	assert.NotNil(t, result)
}
