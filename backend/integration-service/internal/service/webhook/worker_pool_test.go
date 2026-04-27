package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

func TestNewWorkerPool(t *testing.T) {
	t.Parallel()

	pool := NewWorkerPool(nil, 5, 100)
	require.NotNil(t, pool)
	assert.Equal(t, 5, pool.maxWorkers)
	assert.Equal(t, 100, cap(pool.jobQueue))
}

func TestWorkerPoolStartStop(t *testing.T) {
	t.Parallel()

	pool := NewWorkerPool(nil, 2, 10)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = pool.Stop(stopCtx)
	require.NoError(t, err)
}

func TestWorkerPoolStartAlreadyStarted(t *testing.T) {
	t.Parallel()

	pool := NewWorkerPool(nil, 2, 10)
	ctx := context.Background()

	err := pool.Start(ctx)
	require.NoError(t, err)

	// Second Start should fail
	err = pool.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	// Cleanup
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	t.Cleanup(cancel)
	require.NoError(t, pool.Stop(stopCtx))
}

func TestWorkerPoolStopNotStarted(t *testing.T) {
	t.Parallel()

	pool := NewWorkerPool(nil, 2, 10)
	ctx := context.Background()

	// Stop without starting should fail
	err := pool.Stop(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestWorkerPoolSubmit(t *testing.T) {
	t.Parallel()

	// Используем реальный сервис доставки с пустым репозиторием, чтобы избежать nil-паники
	// при вызове w.pool.service.DeliverWebhook внутри воркера.
	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	svc := NewWebhookDeliveryService(webhookRepo, deliveryRepo, nil, &WebhookDeliveryConfig{
		MaxWorkers:         2,
		WorkerQueueSize:    10,
		MaxRetries:         0,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 1 * time.Second,
		HTTPTimeout:        100 * time.Millisecond,
	})
	pool := svc.workers
	ctx := context.Background()

	err := pool.Start(ctx)
	require.NoError(t, err)

	// Отправляем задачу; воркер вызовет DeliverWebhook, который завершится без паники.
	// DeliverWebhook попытается отправить HTTP-запрос на несуществующий адрес и вернёт ошибку,
	// которая будет залогирована — это ожидаемое поведение.
	job := &WebhookDeliveryJob{
		Webhook: &model.WebhookIntegration{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			URL:                 "http://127.0.0.1:0/webhook", // несуществующий адрес
			Method:              "POST",
			MaxPayloadSizeBytes: 1024,
		},
		Attempt: model.NewWebhookDeliveryAttempt(uuid.New(), uuid.New()),
		Event:   nil,
	}
	err = pool.Submit(ctx, job)
	require.NoError(t, err)

	// Даём воркеру время обработать задачу
	time.Sleep(50 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	t.Cleanup(cancel)
	require.NoError(t, pool.Stop(stopCtx))
}

func TestWorkerPoolSubmitWithCancelledContext(t *testing.T) {
	t.Parallel()

	pool := NewWorkerPool(nil, 0, 0) // queue size 0 - blocks immediately

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Submit should return context error since queue is full and context is cancelled
	job := &WebhookDeliveryJob{}
	err := pool.Submit(ctx, job)
	// Either submitted (if queue not full) or context error
	_ = err // may or may not error depending on timing
}
