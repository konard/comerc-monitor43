package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	api "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func createTestResult(success bool) *api.CheckResult {
	return &api.CheckResult{
		Success:        success,
		StatusCode:     200,
		ResponseTimeMs: 100.0,
		CheckedAt:      timestamppb.Now(),
	}
}

func TestNewResultQueue(t *testing.T) {
	q := NewResultQueue(100, time.Hour)

	assert.NotNil(t, q)
	assert.Equal(t, 0, q.Size())
	assert.Equal(t, 100, q.maxSize)
	assert.Equal(t, int64(0), q.DroppedCount())
}

func TestNewResultQueue_Defaults(t *testing.T) {
	q := NewResultQueue(0, 0)

	assert.NotNil(t, q)
	assert.Equal(t, 1000, q.maxSize)     // default
	assert.Equal(t, 24*time.Hour, q.ttl) // default
}

func TestEnqueue(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	result := createTestResult(true)
	added := q.Enqueue(result, "monitor-1")

	assert.True(t, added)
	assert.Equal(t, 1, q.Size())
	assert.Equal(t, int64(0), q.DroppedCount())
}

func TestEnqueue_Overflow(t *testing.T) {
	// Отключаем TTL для чистого теста overflow
	q := NewResultQueue(3, 0) // maxSize = 3, TTL = 0 (disabled)

	// Добавляем 5 элементов (должно уместиться только 3)
	for i := 0; i < 5; i++ {
		result := createTestResult(true)
		q.Enqueue(result, "monitor-1")
	}

	assert.Equal(t, 3, q.Size())
	assert.Equal(t, int64(2), q.DroppedCount()) // 2 элемента были удалены
}

func TestEnqueue_OverflowFIFOEviction(t *testing.T) {
	// Отключаем TTL для чистого теста FIFO eviction
	q := NewResultQueue(3, 0)

	// Добавляем элементы с уникальными метками времени
	now := time.Now()
	result1 := &api.CheckResult{CheckedAt: timestamppb.New(now.Add(0 * time.Second))}
	result2 := &api.CheckResult{CheckedAt: timestamppb.New(now.Add(1 * time.Second))}
	result3 := &api.CheckResult{CheckedAt: timestamppb.New(now.Add(2 * time.Second))}
	result4 := &api.CheckResult{CheckedAt: timestamppb.New(now.Add(3 * time.Second))}

	q.Enqueue(result1, "monitor-1")
	q.Enqueue(result2, "monitor-2")
	q.Enqueue(result3, "monitor-3")
	q.Enqueue(result4, "monitor-4") // должен вытеснить result1

	assert.Equal(t, 3, q.Size())

	// Проверяем FIFO eviction - первый элемент должен быть удален
	item := q.Dequeue()
	assert.NotNil(t, item)
	assert.Equal(t, "monitor-2", item.MonitorID) // result2 теперь первый
}

func TestDequeue(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	result := createTestResult(true)
	q.Enqueue(result, "monitor-1")

	item := q.Dequeue()

	require.NotNil(t, item)
	assert.Equal(t, "monitor-1", item.MonitorID)
	assert.Equal(t, 0, q.Size()) // очередь пуста
}

func TestDequeue_Empty(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	item := q.Dequeue()

	assert.Nil(t, item)
}

func TestDequeueBatch(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	// Добавляем 5 элементов
	for i := 0; i < 5; i++ {
		result := createTestResult(true)
		q.Enqueue(result, "monitor-1")
	}

	// Забираем батч из 3 элементов
	batch := q.DequeueBatch(3)

	require.Len(t, batch, 3)
	assert.Equal(t, 2, q.Size()) // осталось 2 элемента
}

func TestDequeueBatch_MoreThanAvailable(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	// Добавляем только 2 элемента
	for i := 0; i < 2; i++ {
		result := createTestResult(true)
		q.Enqueue(result, "monitor-1")
	}

	// Пытаемся взять батч из 5 элементов (должно вернуть только 2)
	batch := q.DequeueBatch(5)

	require.Len(t, batch, 2)
	assert.Equal(t, 0, q.Size()) // очередь пуста
}

func TestDequeueBatch_Empty(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	batch := q.DequeueBatch(5)

	assert.Nil(t, batch)
}

func TestIsFull(t *testing.T) {
	q := NewResultQueue(3, time.Hour)

	assert.False(t, q.IsFull())

	q.Enqueue(createTestResult(true), "m1")
	assert.False(t, q.IsFull())

	q.Enqueue(createTestResult(true), "m2")
	assert.False(t, q.IsFull())

	q.Enqueue(createTestResult(true), "m3")
	assert.True(t, q.IsFull())
}

func TestClear(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	// Добавляем элементы
	for i := 0; i < 5; i++ {
		q.Enqueue(createTestResult(true), "monitor-1")
	}

	count := q.Clear()

	assert.Equal(t, 5, count)
	assert.Equal(t, 0, q.Size())
}

func TestStats(t *testing.T) {
	// Отключаем TTL
	q := NewResultQueue(3, 0)

	// Добавляем элементы
	now := time.Now()
	for i := 0; i < 3; i++ {
		result := &api.CheckResult{CheckedAt: timestamppb.New(now.Add(time.Duration(i) * time.Second))}
		q.Enqueue(result, "monitor-1")
	}

	// Вызываем overflow (добавляем еще 2 элемента)
	q.Enqueue(createTestResult(true), "monitor-1")
	q.Enqueue(createTestResult(true), "monitor-1")

	stats := q.Stats()

	assert.Equal(t, 3, stats.Size)
	assert.Equal(t, 3, stats.MaxSize)
	assert.Equal(t, int64(2), stats.Dropped)
	assert.NotNil(t, stats.OldestItem)
	assert.NotNil(t, stats.NewestItem)
	assert.True(t, stats.NewestItem.After(*stats.OldestItem))
}

func TestStats_Empty(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	stats := q.Stats()

	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, 10, stats.MaxSize)
	assert.Equal(t, int64(0), stats.Dropped)
	assert.Nil(t, stats.OldestItem)
	assert.Nil(t, stats.NewestItem)
}

func TestCleanupTTL(t *testing.T) {
	q := NewResultQueue(10, 100*time.Millisecond)

	// Добавляем результат
	result1 := createTestResult(true)
	q.Enqueue(result1, "monitor-1")

	assert.Equal(t, 1, q.Size())

	// Ждем пока результат устареет по TTL
	time.Sleep(150 * time.Millisecond)

	// Триггерим cleanup через enqueue нового результата
	q.Enqueue(createTestResult(true), "monitor-2")

	// Старый результат должен быть удален по TTL
	assert.Equal(t, 1, q.Size())

	item := q.Dequeue()
	assert.Equal(t, "monitor-2", item.MonitorID) // только новый результат
}

func TestStartCleanup(t *testing.T) {
	q := NewResultQueue(10, 50*time.Millisecond)

	now := time.Now()
	oldResult := &api.CheckResult{CheckedAt: timestamppb.New(now.Add(-100 * time.Millisecond))}
	q.Enqueue(oldResult, "monitor-old")

	assert.Equal(t, 1, q.Size())

	cancel := q.StartCleanup(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Ждем cleanup cycle
	time.Sleep(100 * time.Millisecond)

	// Старый результат должен быть удален
	assert.Equal(t, 0, q.Size())
}

func TestStartCleanup_ContextCancel(t *testing.T) {
	q := NewResultQueue(10, time.Hour)

	stopCleanup := q.StartCleanup(context.Background(), 10*time.Millisecond)

	// Проверяем что cleanup можно остановить
	time.Sleep(50 * time.Millisecond)
	stopCleanup()

	// Очередь должна работать нормально после остановки cleanup
	q.Enqueue(createTestResult(true), "monitor-1")
	assert.Equal(t, 1, q.Size())
}

func TestConcurrentEnqueue(t *testing.T) {
	q := NewResultQueue(1000, time.Hour)

	// Запускаем несколько goroutine для конкурентной записи
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				result := createTestResult(true)
				q.Enqueue(result, "monitor-1")
			}
			done <- true
		}(i)
	}

	// Ждем завершения всех goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 1000, q.Size())
}

func TestConcurrentDequeue(t *testing.T) {
	q := NewResultQueue(1000, 0)

	// Сначала заполняем очередь
	for i := 0; i < 1000; i++ {
		q.Enqueue(createTestResult(true), "monitor-1")
	}

	// Запускаем несколько goroutines для конкурентного чтения
	done := make(chan bool)
	totalDequeued := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		go func() {
			localCount := 0
			for {
				item := q.Dequeue()
				if item == nil {
					break
				}
				localCount++
			}
			mu.Lock()
			totalDequeued += localCount
			mu.Unlock()
			done <- true
		}()
	}

	// Ждем завершения
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 0, q.Size())
	assert.Equal(t, 1000, totalDequeued)
}
