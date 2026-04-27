package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/pkg/logger"
)

// CheckExecutorInterface определяет интерфейс для выполнения проверок.
type CheckExecutorInterface interface {
	ExecuteCheck(ctx context.Context, monitorID string) (*domain.CheckResult, error)
}

// Scheduler планирует и выполняет проверки мониторов.
type Scheduler struct {
	monitorRepo    interfaces.MonitorRepository
	checkExecutor  CheckExecutorInterface
	eventPublisher EventPublisher

	// Configuration
	checkInterval       time.Duration
	maxConcurrentChecks int
	batchSize           int

	// State
	mu       sync.RWMutex
	running  atomic.Bool
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// EventPublisher публикует события о статусах мониторов.
type EventPublisher interface {
	PublishStatusChange(ctx context.Context, event *StatusChangeEvent) error
}

// StatusChangeEvent событие изменения статуса монитора.
type StatusChangeEvent struct {
	MonitorID    uuid.UUID
	OldStatus    domain.MonitorStatus
	NewStatus    domain.MonitorStatus
	Timestamp    time.Time
	ResponseTime *int
	ErrorMessage *string
}

// SchedulerConfig конфигурация Scheduler.
type SchedulerConfig struct {
	CheckInterval       time.Duration
	MaxConcurrentChecks int
	BatchSize           int
}

// NewScheduler создаёт новый Scheduler.
func NewScheduler(
	monitorRepo interfaces.MonitorRepository,
	checkExecutor CheckExecutorInterface,
	eventPublisher EventPublisher,
	config SchedulerConfig,
) *Scheduler {
	return &Scheduler{
		monitorRepo:         monitorRepo,
		checkExecutor:       checkExecutor,
		eventPublisher:      eventPublisher,
		checkInterval:       config.CheckInterval,
		maxConcurrentChecks: config.MaxConcurrentChecks,
		batchSize:           config.BatchSize,
		stopChan:            make(chan struct{}),
	}
}

// Start запускает планировщик.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return errors.New("scheduler is already running")
	}

	s.running.Store(true)

	// Запускаем горутину для планирования проверок
	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

// Stop останавливает планировщик.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running.Load() {
		return
	}

	s.running.Store(false)
	close(s.stopChan)
	s.wg.Wait()

	// Восстанавливаем stopChan для следующего запуска
	s.stopChan = make(chan struct{})
}

// run основной цикл планировщика.
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scheduleBatch(ctx)
		case <-s.stopChan:
			s.running.Store(false)
			return
		case <-ctx.Done():
			s.running.Store(false)
			return
		}
	}
}

// scheduleBatch планирует пакет проверок.
func (s *Scheduler) scheduleBatch(ctx context.Context) {
	ctx, span := tracing.StartSpan(ctx, "Scheduler.scheduleBatch")
	defer span.End()

	// Получаем мониторы, которые нужно проверить
	monitors, err := s.monitorRepo.ListDueForCheck(ctx, s.batchSize)
	if err != nil {
		tracing.RecordError(span, err)
		logger.DefaultLogger.ErrorContext(ctx, "failed to list monitors due for check",
			"error", err,
		)
		return
	}

	if len(monitors) == 0 {
		tracing.SetSuccess(span)
		return
	}

	// Создаём semaphore для ограничения concurrently
	sem := make(chan struct{}, s.maxConcurrentChecks)
	var wg sync.WaitGroup

	for _, monitor := range monitors {
		wg.Add(1)

		go func(m *domain.Monitor) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Выполняем проверку
			s.executeCheck(ctx, m)
		}(monitor)
	}

	// Ждём завершения всех проверок в пакете
	wg.Wait()
	tracing.SetSuccess(span)
}

// executeCheck выполняет проверку монитора.
func (s *Scheduler) executeCheck(ctx context.Context, monitor *domain.Monitor) {
	// Сохраняем старый статус для детекции изменений
	oldStatus := monitor.Status

	// Выполняем проверку
	result, err := s.checkExecutor.ExecuteCheck(ctx, monitor.ID.String())
	if err != nil {
		logger.DefaultLogger.ErrorContext(ctx, "failed to execute check",
			"monitor_id", monitor.ID,
			"error", err,
		)
		return
	}

	// Обновляем статус монитора
	if result.Status != oldStatus {
		tracing.AddEvent(ctx, "status_changed", map[string]any{
			"monitor_id": monitor.ID,
			"old_status": oldStatus,
			"new_status": result.Status,
		})

		// Публикуем событие изменения статуса
		if s.eventPublisher != nil {
			event := &StatusChangeEvent{
				MonitorID:    monitor.ID,
				OldStatus:    oldStatus,
				NewStatus:    result.Status,
				Timestamp:    time.Now(),
				ResponseTime: result.ResponseTimeMs,
				ErrorMessage: result.ErrorMessage,
			}

			if err := s.eventPublisher.PublishStatusChange(ctx, event); err != nil {
				logger.DefaultLogger.ErrorContext(ctx, "failed to publish status change event",
					"monitor_id", monitor.ID,
					"old_status", oldStatus,
					"new_status", result.Status,
					"error", err,
				)
			}
		}

		// Обновляем статус в БД
		if err := s.monitorRepo.UpdateStatus(ctx, monitor.ID, result.Status); err != nil {
			logger.DefaultLogger.ErrorContext(ctx, "failed to update monitor status",
				"monitor_id", monitor.ID,
				"status", result.Status,
				"error", err,
			)
		}
	}
}

// ScheduleCheck планирует проверку конкретного монитора (для manual trigger).
func (s *Scheduler) ScheduleCheck(ctx context.Context, monitorID uuid.UUID) error {
	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return errors.Wrap(err, "failed to get monitor")
	}

	if !monitor.ShouldCheckNow() {
		return errors.New("monitor is outside working hours")
	}

	// Выполняем проверку асинхронно
	go s.executeCheck(ctx, monitor)

	return nil
}

// IsRunning возвращает true, если планировщик запущен.
func (s *Scheduler) IsRunning() bool {
	return s.running.Load()
}
