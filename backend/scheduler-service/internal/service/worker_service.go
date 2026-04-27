package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

var workerServiceTracer = otel.Tracer("scheduler-service/worker_service")

type WorkerService struct {
	workerRepo         WorkerRepository
	checkRepo          CheckRepository
	auditRepo          AuditRepository
	heartbeatTimeout   time.Duration
	offlineCleanupTime time.Duration
	logger             *slog.Logger
}

func NewWorkerService(
	workerRepo WorkerRepository,
	checkRepo CheckRepository,
	auditRepo AuditRepository,
	heartbeatTimeout, offlineCleanup time.Duration,
	logger *slog.Logger,
) *WorkerService {
	return &WorkerService{
		workerRepo:         workerRepo,
		checkRepo:          checkRepo,
		auditRepo:          auditRepo,
		heartbeatTimeout:   heartbeatTimeout,
		offlineCleanupTime: offlineCleanup,
		logger:             logger,
	}
}

func (s *WorkerService) RegisterWorker(ctx context.Context, name, zone string, metadata map[string]string) (*model.Worker, error) {
	ctx, span := workerServiceTracer.Start(ctx, "WorkerService.RegisterWorker")
	defer span.End()

	span.SetAttributes(
		attribute.String("worker_name", name),
		attribute.String("zone", zone),
	)

	if strings.TrimSpace(name) == "" {
		return nil, model.ErrWorkerNameRequired
	}
	if len(name) > 255 {
		return nil, model.ErrWorkerNameTooLong
	}
	if strings.TrimSpace(zone) == "" {
		return nil, model.ErrInvalidZone
	}

	existing, err := s.workerRepo.GetByName(ctx, name)
	if err == nil && existing != nil {
		return nil, model.ErrWorkerNameExists
	}

	worker := model.NewWorker(name, zone)
	if metadata != nil {
		worker.Metadata = metadata
	}

	if err := s.workerRepo.Create(ctx, worker); err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}

	s.logger.InfoContext(ctx, "worker registered",
		"worker_id", worker.ID,
		"worker_name", worker.Name,
		"zone", worker.Zone,
	)

	s.createAuditLog(ctx, "worker_registered", worker.ID, uuid.Nil, uuid.Nil, map[string]any{
		"worker_name": worker.Name,
		"zone":        worker.Zone,
	})

	return worker, nil
}

func (s *WorkerService) Heartbeat(ctx context.Context, workerID uuid.UUID, status model.WorkerStatus, checksCompleted, checksFailed int, avgDurationMs float64) error {
	ctx, span := workerServiceTracer.Start(ctx, "WorkerService.Heartbeat")
	defer span.End()

	span.SetAttributes(
		attribute.String("worker_id", workerID.String()),
		attribute.String("status", string(status)),
	)

	worker, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return model.ErrWorkerNotFound
	}

	worker.UpdateHeartbeat(status, checksCompleted, checksFailed, avgDurationMs)

	if err := s.workerRepo.Update(ctx, worker); err != nil {
		return fmt.Errorf("failed to update worker heartbeat: %w", err)
	}

	return nil
}

func (s *WorkerService) UnregisterWorker(ctx context.Context, workerID uuid.UUID) error {
	ctx, span := workerServiceTracer.Start(ctx, "WorkerService.UnregisterWorker")
	defer span.End()

	span.SetAttributes(attribute.String("worker_id", workerID.String()))

	worker, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return model.ErrWorkerNotFound
	}

	reassigned, err := s.checkRepo.ReassignByWorkerID(ctx, workerID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to reassign checks during unregister",
			"worker_id", workerID,
			"error", err,
		)
	}

	if err := s.workerRepo.Delete(ctx, workerID); err != nil {
		return fmt.Errorf("failed to unregister worker: %w", err)
	}

	s.logger.InfoContext(ctx, "worker unregistered",
		"worker_id", workerID,
		"worker_name", worker.Name,
		"reassigned_checks", reassigned,
	)

	s.createAuditLog(ctx, "worker_unregistered", workerID, uuid.Nil, uuid.Nil, map[string]any{
		"worker_name":   worker.Name,
		"pending_tasks": reassigned,
		"reason":        "graceful",
	})

	return nil
}

func (s *WorkerService) GetWorkerStatus(ctx context.Context, workerID uuid.UUID) (*model.Worker, error) {
	ctx, span := workerServiceTracer.Start(ctx, "WorkerService.GetWorkerStatus")
	defer span.End()

	span.SetAttributes(attribute.String("worker_id", workerID.String()))

	worker, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return nil, model.ErrWorkerNotFound
	}
	return worker, nil
}

func (s *WorkerService) ListWorkers(ctx context.Context, status, zone string, page, pageSize int) ([]*model.Worker, int, error) {
	return s.workerRepo.List(ctx, status, zone, page, pageSize)
}

func (s *WorkerService) DetectOfflineWorkers(ctx context.Context) (int, error) {
	ctx, span := workerServiceTracer.Start(ctx, "WorkerService.DetectOfflineWorkers")
	defer span.End()

	expired, err := s.workerRepo.ListExpired(ctx, s.heartbeatTimeout)
	if err != nil {
		return 0, fmt.Errorf("failed to detect offline workers: %w", err)
	}

	count := 0
	for _, w := range expired {
		reassigned, reassignErr := s.checkRepo.ReassignByWorkerID(ctx, w.ID)
		if reassignErr != nil {
			s.logger.ErrorContext(ctx, "failed to reassign checks for offline worker",
				"worker_id", w.ID,
				"error", reassignErr,
			)
		}

		w.MarkOffline()
		if err := s.workerRepo.Update(ctx, w); err != nil {
			s.logger.ErrorContext(ctx, "failed to mark worker offline",
				"worker_id", w.ID,
				"error", err,
			)
			continue
		}

		count++
		s.logger.InfoContext(ctx, "worker marked offline",
			"worker_id", w.ID,
			"worker_name", w.Name,
			"reassigned_checks", reassigned,
		)

		s.createAuditLog(ctx, "worker_marked_offline", w.ID, uuid.Nil, uuid.Nil, map[string]any{
			"worker_name":  w.Name,
			"last_seen":    w.LastHeartbeat,
			"pending_task": reassigned,
			"reason":       "heartbeat_timeout",
		})
	}

	return count, nil
}

func (s *WorkerService) CleanupOfflineWorkers(ctx context.Context) (int, error) {
	ctx, span := workerServiceTracer.Start(ctx, "WorkerService.CleanupOfflineWorkers")
	defer span.End()

	workers, err := s.workerRepo.ListOfflineForCleanup(ctx, time.Now().Add(-s.offlineCleanupTime))
	if err != nil {
		return 0, fmt.Errorf("failed to list workers for cleanup: %w", err)
	}

	count := 0
	for _, w := range workers {
		if err := s.workerRepo.Delete(ctx, w.ID); err != nil {
			s.logger.ErrorContext(ctx, "failed to cleanup offline worker",
				"worker_id", w.ID,
				"error", err,
			)
			continue
		}
		count++
		s.logger.InfoContext(ctx, "offline worker cleaned up",
			"worker_id", w.ID,
			"worker_name", w.Name,
		)

		s.createAuditLog(ctx, "worker_removed", w.ID, uuid.Nil, uuid.Nil, map[string]any{
			"worker_name": w.Name,
			"reason":      "extended_timeout",
		})
	}

	return count, nil
}

func (s *WorkerService) createAuditLog(ctx context.Context, action string, workerID, checkID, monitorID uuid.UUID, details map[string]any) {
	if err := s.auditRepo.Create(ctx, action, workerID, checkID, monitorID, details); err != nil {
		s.logger.ErrorContext(ctx, "failed to create audit log",
			"error", err,
			"action", action,
		)
	}
}

func (s *WorkerService) SelectWorker(ctx context.Context, zone string) (*model.Worker, error) {
	ctx, span := workerServiceTracer.Start(ctx, "WorkerService.SelectWorker")
	defer span.End()

	span.SetAttributes(attribute.String("zone", zone))

	workers, err := s.workerRepo.ListIdleByZone(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to select worker: %w", err)
	}

	if len(workers) == 0 {
		workers, err = s.workerRepo.ListIdleByZone(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("failed to select worker from any zone: %w", err)
		}
	}

	if len(workers) == 0 {
		return nil, model.ErrNoWorkersAvailable
	}

	return workers[0], nil
}
