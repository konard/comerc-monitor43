package service

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
	"github.com/raul/monitor/backend/scheduler-service/internal/monitor"
)

type MonitorClient interface {
	GetMonitor(ctx context.Context, monitorID string) (*monitor.MonitorInfo, error)
	ListActiveMonitors(ctx context.Context, pageSize int) ([]*monitor.MonitorInfo, error)
	IsMonitorPaused(ctx context.Context, monitorID string) (bool, error)
}

type SchedulingLoop struct {
	scheduler     *CheckScheduler
	workerService *WorkerService
	monitorClient MonitorClient
	locker        DistributedLocker

	schedulingInterval time.Duration
	batchSize          int

	running  atomic.Bool
	stopChan chan struct{}
	wg       sync.WaitGroup
	logger   *slog.Logger
}

func NewSchedulingLoop(
	scheduler *CheckScheduler,
	workerService *WorkerService,
	monitorClient MonitorClient,
	locker DistributedLocker,
	schedulingInterval time.Duration,
	logger *slog.Logger,
) *SchedulingLoop {
	return &SchedulingLoop{
		scheduler:          scheduler,
		workerService:      workerService,
		monitorClient:      monitorClient,
		locker:             locker,
		schedulingInterval: schedulingInterval,
		batchSize:          100,
		logger:             logger,
	}
}

func (l *SchedulingLoop) Start(ctx context.Context) {
	if l.running.Load() {
		return
	}

	l.running.Store(true)
	l.stopChan = make(chan struct{})

	l.wg.Add(1)
	go l.run(ctx)

	l.logger.InfoContext(ctx, "scheduling loop started",
		"interval", l.schedulingInterval,
	)
}

func (l *SchedulingLoop) Stop() {
	if !l.running.Load() {
		return
	}

	close(l.stopChan)
	l.wg.Wait()
	l.running.Store(false)

	l.logger.InfoContext(context.Background(), "scheduling loop stopped")
}

func (l *SchedulingLoop) IsRunning() bool {
	return l.running.Load()
}

func (l *SchedulingLoop) run(ctx context.Context) {
	defer l.wg.Done()

	schedulingTicker := time.NewTicker(l.schedulingInterval)
	defer schedulingTicker.Stop()

	offlineTicker := time.NewTicker(30 * time.Second)
	defer offlineTicker.Stop()

	cleanupTicker := time.NewTicker(10 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-l.stopChan:
			return
		case <-ctx.Done():
			return
		case <-schedulingTicker.C:
			l.withLock(ctx, "scheduling", l.scheduleChecks)
		case <-offlineTicker.C:
			l.withLock(ctx, "offline-detection", l.detectOfflineAndReassign)
		case <-cleanupTicker.C:
			l.withLock(ctx, "cleanup", l.cleanupOfflineWorkers)
		}
	}
}

func (l *SchedulingLoop) withLock(ctx context.Context, key string, fn func(context.Context)) {
	if l.locker == nil {
		fn(ctx)
		return
	}

	acquired, unlock, err := l.locker.TryLock(ctx, "scheduler:"+key)
	if err != nil {
		l.logger.ErrorContext(ctx, "failed to acquire lock",
			"lock_key", key,
			"error", err,
		)
		return
	}

	if !acquired {
		l.logger.DebugContext(ctx, "lock not acquired, skipping",
			"lock_key", key,
		)
		return
	}

	defer func() {
		if uErr := unlock(); uErr != nil {
			l.logger.ErrorContext(ctx, "failed to release lock",
				"lock_key", key,
				"error", uErr,
			)
		}
	}()

	fn(ctx)
}

func (l *SchedulingLoop) scheduleChecks(ctx context.Context) {
	monitors, err := l.monitorClient.ListActiveMonitors(ctx, l.batchSize)
	if err != nil {
		l.logger.ErrorContext(ctx, "failed to list active monitors for scheduling",
			"error", err,
		)
		return
	}

	now := time.Now()
	scheduled := 0

	for _, mon := range monitors {
		if ctx.Err() != nil {
			break
		}

		lastChecked := mon.LastCheckedAt
		interval := time.Duration(mon.IntervalSeconds) * time.Second

		if !lastChecked.IsZero() && lastChecked.Add(interval).After(now) {
			continue
		}

		monitorID, parseErr := uuid.Parse(mon.ID)
		if parseErr != nil {
			l.logger.WarnContext(ctx, "invalid monitor ID, skipping",
				"monitor_id", mon.ID,
			)
			continue
		}

		_, schedErr := l.scheduler.ScheduleCheck(ctx, monitorID, model.PriorityNormal, now)
		if schedErr != nil {
			if model.IsSchedulerError(schedErr) {
				l.logger.DebugContext(ctx, "check not scheduled",
					"monitor_id", mon.ID,
					"error", schedErr,
				)
			} else {
				l.logger.ErrorContext(ctx, "failed to schedule check",
					"monitor_id", mon.ID,
					"error", schedErr,
				)
			}
			continue
		}

		scheduled++
	}

	if scheduled > 0 {
		l.logger.InfoContext(ctx, "checks scheduled", "count", scheduled)
	}
}

func (l *SchedulingLoop) detectOfflineAndReassign(ctx context.Context) {
	offlineCount, err := l.workerService.DetectOfflineWorkers(ctx)
	if err != nil {
		l.logger.ErrorContext(ctx, "failed to detect offline workers",
			"error", err,
		)
		return
	}

	if offlineCount > 0 {
		l.logger.InfoContext(ctx, "offline workers detected", "count", offlineCount)
	}

	reassigned, err := l.scheduler.ReassignOverdueChecks(ctx)
	if err != nil {
		l.logger.ErrorContext(ctx, "failed to reassign overdue checks",
			"error", err,
		)
		return
	}

	if reassigned > 0 {
		l.logger.InfoContext(ctx, "overdue checks reassigned", "count", reassigned)
	}
}

func (l *SchedulingLoop) cleanupOfflineWorkers(ctx context.Context) {
	cleaned, err := l.workerService.CleanupOfflineWorkers(ctx)
	if err != nil {
		l.logger.ErrorContext(ctx, "failed to cleanup offline workers",
			"error", err,
		)
		return
	}

	if cleaned > 0 {
		l.logger.InfoContext(ctx, "offline workers cleaned up", "count", cleaned)
	}
}
