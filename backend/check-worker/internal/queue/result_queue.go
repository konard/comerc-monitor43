package queue

import (
	"context"
	"sync"
	"time"

	api "github.com/raul/monitor/api/proto"
)

// QueuedResult представляет результат проверки в очереди.
type QueuedResult struct {
	Result       *api.CheckResult
	MonitorID    string
	QueuedAt     time.Time
	AttemptCount int
	RetryAfter   time.Time
}

// ResultQueue представляет потокобезопасную очередь результатов с overflow protection.
type ResultQueue struct {
	mu      sync.RWMutex
	items   []*QueuedResult
	maxSize int
	dropped int64 // счетчик удаленных результатов при overflow

	// TTL для результатов - старые результаты будут удалены
	ttl time.Duration
}

// NewResultQueue создает новую очередь результатов.
func NewResultQueue(maxSize int, ttl time.Duration) *ResultQueue {
	if maxSize <= 0 {
		maxSize = 1000 // default
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour // default 24 hours
	}

	return &ResultQueue{
		items:   make([]*QueuedResult, 0, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Enqueue добавляет результат в очередь с overflow protection.
// Если очередь полна, удаляет самый старый результат (FIFO eviction).
// Возвращает true, если результат был добавлен без замены, false если произошло замещение.
func (q *ResultQueue) Enqueue(result *api.CheckResult, monitorID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	queueItem := &QueuedResult{
		Result:       result,
		MonitorID:    monitorID,
		QueuedAt:     now,
		AttemptCount: 0,
	}

	// Очищаем устаревшие результаты
	q.cleanupLocked(now)

	// Проверяем overflow после cleanup
	if len(q.items) >= q.maxSize {
		// Overflow: удаляем самый старый результат и добавляем новый
		q.items = q.items[1:] // FIFO eviction
		q.dropped++
		q.items = append(q.items, queueItem) // добавляем новый элемент
		return false                         // указывает что произошло замещение
	}

	q.items = append(q.items, queueItem)
	return true // указывает что добавление без замещения
}

// Dequeue возвращает и удаляет первый элемент из очереди.
// Если очередь пуста, возвращает nil.
func (q *ResultQueue) Dequeue() *QueuedResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item
}

// DequeueBatch возвращает и удаляет до batchSize элементов из очереди.
// Всегда возвращает непустой слайс (или nil если очередь пуста).
func (q *ResultQueue) DequeueBatch(batchSize int) []*QueuedResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	// Ограничиваем batchSize размером очереди
	if batchSize > len(q.items) {
		batchSize = len(q.items)
	}

	batch := q.items[:batchSize]
	q.items = q.items[batchSize:]

	return batch
}

// Size возвращает текущее количество элементов в очереди.
func (q *ResultQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// IsFull возвращает true, если очередь достигла maxSize.
func (q *ResultQueue) IsFull() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items) >= q.maxSize
}

// DroppedCount возвращает количество удаленных результатов при overflow.
func (q *ResultQueue) DroppedCount() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.dropped
}

// Clear очищает очередь и возвращает количество удаленных элементов.
func (q *ResultQueue) Clear() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	count := len(q.items)
	q.items = q.items[:0]
	return count
}

// cleanupLocked удаляет устаревшие результаты (TTL).
// Должен вызываться с удержанным mu (write lock).
func (q *ResultQueue) cleanupLocked(now time.Time) {
	// Если TTL не задан или <= 0, cleanup отключен
	if q.ttl <= 0 {
		return
	}

	cutoff := now.Add(-q.ttl)

	// Находим первый элемент, который НЕ устарел
	// Элемент устарел если: QueuedAt <= cutoff (т.е. старше или равен cutoff времени)
	i := 0
	for i < len(q.items) {
		// Если элемент после cutoff - он еще свежий, останавливаемся
		if q.items[i].QueuedAt.After(cutoff) {
			break
		}
		// Иначе элемент устарел (QueuedAt <= cutoff), продолжаем
		i++
	}

	if i > 0 {
		// Удаляем все устаревшие элементы (первые i элементов)
		q.items = q.items[i:]
	}
}

// StartCleanup запускает background goroutine для периодической очистки.
// Возвращает функцию cancel для остановки cleanup.
func (q *ResultQueue) StartCleanup(ctx context.Context, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				q.mu.Lock()
				q.cleanupLocked(time.Now())
				q.mu.Unlock()
			}
		}
	}()

	return cancel
}

// GetStats возвращает статистику очереди.
type QueueStats struct {
	Size       int
	MaxSize    int
	Dropped    int64
	OldestItem *time.Time
	NewestItem *time.Time
}

// Stats возвращает статистику очереди.
func (q *ResultQueue) Stats() QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := QueueStats{
		Size:    len(q.items),
		MaxSize: q.maxSize,
		Dropped: q.dropped,
	}

	if len(q.items) > 0 {
		oldest := q.items[0].QueuedAt
		newest := q.items[len(q.items)-1].QueuedAt
		stats.OldestItem = &oldest
		stats.NewestItem = &newest
	}

	return stats
}
