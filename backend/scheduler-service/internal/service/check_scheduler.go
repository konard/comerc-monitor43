package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

var checkSchedulerTracer = otel.Tracer("scheduler-service/check_scheduler")

type CheckScheduler struct {
	checkRepo     CheckRepository
	workerRepo    WorkerRepository
	auditRepo     AuditRepository
	monitorClient MonitorClient
	overdueThresh time.Duration
	logger        *slog.Logger
}

func NewCheckScheduler(
	checkRepo CheckRepository,
	workerRepo WorkerRepository,
	auditRepo AuditRepository,
	monitorClient MonitorClient,
	overdueThreshold time.Duration,
	logger *slog.Logger,
) *CheckScheduler {
	return &CheckScheduler{
		checkRepo:     checkRepo,
		workerRepo:    workerRepo,
		auditRepo:     auditRepo,
		monitorClient: monitorClient,
		overdueThresh: overdueThreshold,
		logger:        logger,
	}
}

func (s *CheckScheduler) ScheduleCheck(ctx context.Context, monitorID uuid.UUID, priority model.CheckPriority, scheduledAt time.Time) (*model.ScheduledCheck, error) {
	ctx, span := checkSchedulerTracer.Start(ctx, "CheckScheduler.ScheduleCheck")
	defer span.End()

	span.SetAttributes(
		attribute.String("monitor_id", monitorID.String()),
		attribute.String("priority", string(priority)),
	)

	hasPending, err := s.checkRepo.HasPendingCheck(ctx, monitorID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending: %w", err)
	}
	if hasPending {
		s.logger.WarnContext(ctx, "check already pending for monitor, skipping",
			"monitor_id", monitorID,
		)
		return nil, model.ErrCheckAlreadyPending
	}

	check := model.NewScheduledCheck(monitorID, priority, scheduledAt)

	worker, err := s.workerRepo.ListIdleByZone(ctx, "")
	if err == nil && len(worker) > 0 {
		check.Assign(worker[0].ID)
	}

	if err := s.checkRepo.Create(ctx, check); err != nil {
		return nil, fmt.Errorf("failed to schedule check: %w", err)
	}

	s.logger.InfoContext(ctx, "check scheduled",
		"check_id", check.ID,
		"monitor_id", monitorID,
		"priority", priority,
		"worker_id", check.WorkerID,
	)

	return check, nil
}

func (s *CheckScheduler) GetScheduledChecks(ctx context.Context, from, to time.Time, status string, page, pageSize int) ([]*model.ScheduledCheck, int, error) {
	ctx, span := checkSchedulerTracer.Start(ctx, "CheckScheduler.GetScheduledChecks")
	defer span.End()

	if !from.Before(to) {
		return nil, 0, model.ErrInvalidTimeRange
	}
	return s.checkRepo.ListByTimeRange(ctx, from, to, status, page, pageSize)
}

func (s *CheckScheduler) GetNextCheck(ctx context.Context, monitorID uuid.UUID) (*model.ScheduledCheck, error) {
	ctx, span := checkSchedulerTracer.Start(ctx, "CheckScheduler.GetNextCheck")
	defer span.End()

	span.SetAttributes(attribute.String("monitor_id", monitorID.String()))

	check, err := s.checkRepo.GetByMonitorID(ctx, monitorID)
	if err != nil {
		return nil, model.ErrMonitorNotFound
	}
	return check, nil
}

func (s *CheckScheduler) TriggerScheduledCheck(ctx context.Context, monitorID uuid.UUID, priority model.CheckPriority) (*model.ScheduledCheck, error) {
	ctx, span := checkSchedulerTracer.Start(ctx, "CheckScheduler.TriggerScheduledCheck")
	defer span.End()

	span.SetAttributes(
		attribute.String("monitor_id", monitorID.String()),
		attribute.String("priority", string(priority)),
	)

	if s.monitorClient != nil {
		paused, err := s.monitorClient.IsMonitorPaused(ctx, monitorID.String())
		if err != nil {
			return nil, model.ErrMonitorNotFound
		}
		if paused {
			return nil, model.ErrCheckAlreadyPending
		}
	}

	return s.ScheduleCheck(ctx, monitorID, priority, time.Now())
}

// CompleteCheck помечает проверку как завершённую (успешно или с ошибкой).
func (s *CheckScheduler) CompleteCheck(ctx context.Context, checkID uuid.UUID, failed bool, errMsg string) error {
	ctx, span := checkSchedulerTracer.Start(ctx, "CheckScheduler.CompleteCheck")
	defer span.End()

	span.SetAttributes(
		attribute.String("check_id", checkID.String()),
		attribute.Bool("failed", failed),
	)

	check, err := s.checkRepo.GetByID(ctx, checkID)
	if err != nil {
		return model.ErrCheckNotFound
	}

	if failed {
		check.MarkFailed(errMsg)
	} else {
		check.MarkCompleted()
	}

	if err := s.checkRepo.Update(ctx, check); err != nil {
		return fmt.Errorf("failed to complete check: %w", err)
	}

	s.logger.InfoContext(ctx, "check completed",
		"check_id", checkID,
		"failed", failed,
	)

	return nil
}

func (s *CheckScheduler) ReassignOverdueChecks(ctx context.Context) (int, error) {
	ctx, span := checkSchedulerTracer.Start(ctx, "CheckScheduler.ReassignOverdueChecks")
	defer span.End()

	overdue, err := s.checkRepo.ListOverdue(ctx, s.overdueThresh)
	if err != nil {
		return 0, fmt.Errorf("failed to list overdue checks: %w", err)
	}

	count := 0
	for _, check := range overdue {
		workers, wErr := s.workerRepo.ListIdleByZone(ctx, "")
		if wErr != nil || len(workers) == 0 {
			s.logger.WarnContext(ctx, "no workers available for overdue check",
				"check_id", check.ID,
			)
			continue
		}

		check.Priority = model.PriorityHigh
		check.Assign(workers[0].ID)

		if uErr := s.checkRepo.Update(ctx, check); uErr != nil {
			s.logger.ErrorContext(ctx, "failed to reassign overdue check",
				"check_id", check.ID,
				"error", uErr,
			)
			continue
		}
		count++
	}

	if count > 0 {
		s.logger.InfoContext(ctx, "overdue checks reassigned", "count", count)
	}

	return count, nil
}
