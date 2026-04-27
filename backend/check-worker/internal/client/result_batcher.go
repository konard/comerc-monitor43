package client

import (
	"context"
	"log/slog"
	"sync"
	"time"

	api "github.com/raul/monitor/api/proto"
)

// BatchingConfig конфигурация пакетной отправки
type BatchingConfig struct {
	MaxBatchSize  int           // Максимальный размер пакета
	FlushInterval time.Duration // Интервал автоматической отправки
	Timeout       time.Duration // Таймаут отправки
}

// DefaultBatchingConfig возвращает конфигурацию по умолчанию
func DefaultBatchingConfig() BatchingConfig {
	return BatchingConfig{
		MaxBatchSize:  10,
		FlushInterval: 5 * time.Second,
		Timeout:       10 * time.Second,
	}
}

// ResultBatcher накапливает результаты для пакетной отправки
type ResultBatcher struct {
	mu             sync.Mutex
	config         BatchingConfig
	logger         *slog.Logger
	client         MonitorServiceSubmitter
	pendingResults map[string]*api.CheckResult // check_id -> result

	// Duplicate detection
	sentCheckIDs map[string]time.Time // check_id -> timestamp

	// Flush control
	flushCtx    context.Context
	flushCancel context.CancelFunc
	flushWg     sync.WaitGroup

	// Metrics
	totalBatches int64
	totalResults int64
	duplicates   int64
}

// MonitorServiceSubmitter предоставляет интерфейс для отправки результатов
type MonitorServiceSubmitter interface {
	SubmitCheckResults(ctx context.Context, results map[string]*api.CheckResult) error
}

// NewResultBatcher создаёт новый batcher
func NewResultBatcher(client MonitorServiceSubmitter, config BatchingConfig, logger *slog.Logger) *ResultBatcher {
	flushCtx, flushCancel := context.WithCancel(context.Background())

	batcher := &ResultBatcher{
		config:         config,
		logger:         logger,
		client:         client,
		pendingResults: make(map[string]*api.CheckResult),
		sentCheckIDs:   make(map[string]time.Time),
		flushCtx:       flushCtx,
		flushCancel:    flushCancel,
	}

	// Запускаем background flush
	batcher.startFlushLoop()

	return batcher
}

// AddResult добавляет результат в пакет
func (b *ResultBatcher) AddResult(ctx context.Context, checkID string, result *api.CheckResult) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Проверяем на дубликаты
	if _, exists := b.sentCheckIDs[checkID]; exists {
		b.duplicates++
		b.logger.DebugContext(ctx, "duplicate result ignored",
			"check_id", checkID,
			"total_duplicates", b.duplicates,
		)
		return nil // Дубликаты игнорируются без ошибки
	}

	// Добавляем в очередь на отправку
	b.pendingResults[checkID] = result

	// Если достигли лимита, отправляем
	if len(b.pendingResults) >= b.config.MaxBatchSize {
		b.logger.DebugContext(ctx, "batch size reached, flushing",
			"size", len(b.pendingResults),
			"max_size", b.config.MaxBatchSize,
		)
		go func() {
			if err := b.flush(ctx); err != nil {
				b.logger.ErrorContext(ctx, "failed to flush batch", "error", err)
			}
		}()
	}

	return nil
}

// Flush принудительно отправляет все накопленные результаты
func (b *ResultBatcher) Flush(ctx context.Context) error {
	return b.flush(ctx)
}

// flush отправляет накопленные результаты
func (b *ResultBatcher) flush(ctx context.Context) error {
	b.mu.Lock()

	if len(b.pendingResults) == 0 {
		b.mu.Unlock()
		return nil
	}

	// Копируем результаты для отправки
	results := make(map[string]*api.CheckResult, len(b.pendingResults))
	for checkID, result := range b.pendingResults {
		results[checkID] = result
	}

	// Очищаем очередь
	b.pendingResults = make(map[string]*api.CheckResult)
	b.mu.Unlock()

	// Отправляем без блокировки мьютекса
	if err := b.client.SubmitCheckResults(ctx, results); err != nil {
		b.logger.ErrorContext(ctx, "failed to submit batch results",
			"count", len(results),
			"error", err,
		)

		// При ошибке возвращаем результаты в очередь
		b.mu.Lock()
		for checkID, result := range results {
			b.pendingResults[checkID] = result
		}
		b.mu.Unlock()

		return err
	}

	// Успешная отправка - отмечаем как отправленные
	b.mu.Lock()
	now := time.Now()
	for checkID := range results {
		b.sentCheckIDs[checkID] = now
		b.totalResults++
	}
	b.totalBatches++
	b.logger.DebugContext(ctx, "batch submitted successfully",
		"count", len(results),
		"total_batches", b.totalBatches,
		"total_results", b.totalResults,
	)
	b.mu.Unlock()

	return nil
}

// startFlushLoop запускает периодический flush
func (b *ResultBatcher) startFlushLoop() {
	b.flushWg.Add(1)
	go func() {
		defer b.flushWg.Done()

		ticker := time.NewTicker(b.config.FlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := b.flush(b.flushCtx); err != nil {
					b.logger.ErrorContext(b.flushCtx, "failed to flush batch", "error", err)
				}
			case <-b.flushCtx.Done():
				// Финальный flush при остановке
				if err := b.flush(context.Background()); err != nil {
					b.logger.Error("failed to flush batch during shutdown", "error", err)
				}
				return
			}
		}
	}()
}

// Close останавливает batcher и отправляет оставшиеся результаты
func (b *ResultBatcher) Close() error {
	// Останавливаем flush loop
	b.flushCancel()
	b.flushWg.Wait()

	// Финальный flush
	if err := b.flush(context.Background()); err != nil {
		b.logger.Error("failed to flush batch during close", "error", err)
	}

	// Очищаем старые записи (старше 24 часов)
	b.cleanupOldRecords()

	return nil
}

// cleanupOldRecords удаляет старые записи из sentCheckIDs
func (b *ResultBatcher) cleanupOldRecords() {
	b.mu.Lock()
	defer b.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for checkID, timestamp := range b.sentCheckIDs {
		if timestamp.Before(cutoff) {
			delete(b.sentCheckIDs, checkID)
		}
	}
}

// GetStats возвращает статистику
func (b *ResultBatcher) GetStats() BatcherStats {
	b.mu.Lock()
	defer b.mu.Unlock()

	return BatcherStats{
		PendingCount: len(b.pendingResults),
		TotalBatches: b.totalBatches,
		TotalResults: b.totalResults,
		Duplicates:   b.duplicates,
		SentCheckIDs: len(b.sentCheckIDs),
	}
}

// BatcherStats статистика batcher
type BatcherStats struct {
	PendingCount int
	TotalBatches int64
	TotalResults int64
	Duplicates   int64
	SentCheckIDs int
}
