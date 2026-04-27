package webhook

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

// WorkerPool пул воркеров для асинхронной доставки webhook.
type WorkerPool struct {
	service    *WebhookDeliveryService
	maxWorkers int
	jobQueue   chan *WebhookDeliveryJob
	workers    []*Worker
	wg         sync.WaitGroup
	started    atomic.Bool
	stopped    atomic.Bool
	log        *slog.Logger
}

// Worker воркер для доставки webhook.
type Worker struct {
	id   int
	pool *WorkerPool
}

// NewWorkerPool создаёт новый WorkerPool.
func NewWorkerPool(service *WebhookDeliveryService, maxWorkers, queueSize int) *WorkerPool {
	return &WorkerPool{
		service:    service,
		maxWorkers: maxWorkers,
		jobQueue:   make(chan *WebhookDeliveryJob, queueSize),
		log:        slog.Default(),
	}
}

// Start запускает worker pool.
func (p *WorkerPool) Start(ctx context.Context) error {
	if !p.started.CompareAndSwap(false, true) {
		return errors.New("worker pool already started")
	}

	p.log.InfoContext(ctx, "starting worker pool",
		"max_workers", p.maxWorkers,
		"queue_size", cap(p.jobQueue),
	)

	// Создаём воркеров
	p.workers = make([]*Worker, p.maxWorkers)
	for i := 0; i < p.maxWorkers; i++ {
		worker := &Worker{
			id:   i + 1,
			pool: p,
		}
		p.workers[i] = worker

		// Запускаем воркер в горутине
		p.wg.Add(1)
		go worker.run(ctx)
	}

	p.log.InfoContext(ctx, "worker pool started",
		"workers", p.maxWorkers,
	)

	return nil
}

// Stop останавливает worker pool.
func (p *WorkerPool) Stop(ctx context.Context) error {
	if !p.started.CompareAndSwap(true, false) {
		return errors.New("worker pool not started")
	}

	if !p.stopped.CompareAndSwap(false, true) {
		return nil // Already stopping
	}

	p.log.InfoContext(ctx, "stopping worker pool")

	// Закрываем очередь
	close(p.jobQueue)

	// Ждём завершения воркеров
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.InfoContext(ctx, "worker pool stopped")
		return nil
	case <-ctx.Done():
		p.log.WarnContext(ctx, "worker pool stop cancelled")
		return ctx.Err()
	}
}

// Submit отправляет job в очередь на выполнение.
func (p *WorkerPool) Submit(ctx context.Context, job *WebhookDeliveryJob) error {
	select {
	case p.jobQueue <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// run основной цикл воркера.
func (w *Worker) run(ctx context.Context) {
	defer w.pool.wg.Done()

	w.pool.log.DebugContext(ctx, "worker started",
		"worker_id", w.id,
	)

	for {
		select {
		case <-ctx.Done():
			w.pool.log.DebugContext(ctx, "worker stopping",
				"worker_id", w.id,
			)
			return

		case job, ok := <-w.pool.jobQueue:
			if !ok {
				// Очередь закрыта
				w.pool.log.DebugContext(ctx, "worker queue closed",
					"worker_id", w.id,
				)
				return
			}

			// Выполняем job
			w.processJob(ctx, job)
		}
	}
}

// processJob обрабатывает job доставки webhook.
func (w *Worker) processJob(ctx context.Context, job *WebhookDeliveryJob) {
	w.pool.log.DebugContext(ctx, "processing webhook job",
		"worker_id", w.id,
		"webhook_id", job.Webhook.ID,
		"alert_id", job.Attempt.AlertID,
		"retry_count", job.Attempt.RetryCount,
	)

	// Вызываем delivery service
	err := w.pool.service.DeliverWebhook(ctx, job)

	if err != nil {
		w.pool.log.ErrorContext(ctx, "webhook delivery job failed",
			"worker_id", w.id,
			"webhook_id", job.Webhook.ID,
			"error", err,
		)
	}
}
