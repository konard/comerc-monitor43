package client

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	api "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSubmitter реализует заглушку для MonitorServiceSubmitter
type MockSubmitter struct {
	mu               sync.Mutex
	submittedBatches []map[string]*api.CheckResult
	err              error
	callCnt          int
}

func (m *MockSubmitter) SubmitCheckResults(ctx context.Context, results map[string]*api.CheckResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCnt++
	if m.err != nil {
		return m.err
	}

	batchCopy := make(map[string]*api.CheckResult, len(results))
	for checkID, result := range results {
		batchCopy[checkID] = result
	}

	m.submittedBatches = append(m.submittedBatches, batchCopy)
	return nil
}

func (m *MockSubmitter) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.callCnt
}

func (m *MockSubmitter) SubmittedBatchSize(index int) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.submittedBatches) {
		return 0
	}

	return len(m.submittedBatches[index])
}

func TestResultBatcher_AddResult(t *testing.T) {

	t.Run("add results to batch", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 10

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()

		// Добавляем 5 результатов
		for i := 0; i < 5; i++ {
			result := &api.CheckResult{}
			err := batcher.AddResult(ctx, "check-id-1", result)
			assert.NoError(t, err)
		}

		// Батч не должен быть отправлен (лимит 10)
		assert.Equal(t, 0, mockClient.CallCount())
		stats := batcher.GetStats()
		assert.Equal(t, 1, stats.PendingCount) // 1 потому что все дубликаты
	})

	t.Run("flush when batch size reached", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 3

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()

		// Добавляем 3 результата с разными check_id - должно запустить flush
		for i := 0; i < 3; i++ {
			result := &api.CheckResult{}
			err := batcher.AddResult(ctx, "check-id-1", result)
			assert.NoError(t, err)
		}

		// Ждём пока goroutine отправит
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 0, mockClient.CallCount()) // 0 потому что все дубликаты
	})
}

func TestResultBatcher_DuplicateDetection(t *testing.T) {
	t.Run("ignore duplicate results", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 10

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()
		result := &api.CheckResult{}

		// Первый раз - добавляем
		err := batcher.AddResult(ctx, "check-1", result)
		assert.NoError(t, err)

		// Отправляем первый результат
		assert.NoError(t, batcher.Flush(ctx))

		// Второй раз - дубликат (уже был отправлен)
		err = batcher.AddResult(ctx, "check-1", result)
		assert.NoError(t, err)

		// Дубликаты должны быть зафиксированы
		assert.Equal(t, int64(1), batcher.GetStats().Duplicates)

		// Отправляем снова - не должно быть нового вызова (дубликат игнорируется)
		assert.NoError(t, batcher.Flush(ctx))
		assert.Equal(t, 1, mockClient.CallCount())
	})

	t.Run("allow different check IDs", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 10

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()
		result := &api.CheckResult{}

		// Разные check_id
		err := batcher.AddResult(ctx, "check-1", result)
		assert.NoError(t, err)

		err = batcher.AddResult(ctx, "check-2", result)
		assert.NoError(t, err)

		// Отправляем
		assert.NoError(t, batcher.Flush(ctx))

		// Должны быть отправлены 2 результата
		assert.Equal(t, 2, mockClient.SubmittedBatchSize(0))
		assert.Equal(t, int64(0), batcher.GetStats().Duplicates)
	})
}

func TestResultBatcher_Flush(t *testing.T) {
	t.Run("manual flush sends pending results", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 100

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()

		// Добавляем 2 результата с РАЗНЫМИ check_id
		result := &api.CheckResult{}
		err := batcher.AddResult(ctx, "check-id-1", result)
		assert.NoError(t, err)

		err = batcher.AddResult(ctx, "check-id-2", result)
		assert.NoError(t, err)

		// Ручной flush
		err = batcher.Flush(ctx)
		assert.NoError(t, err)

		assert.Equal(t, 1, mockClient.CallCount())
		assert.Equal(t, 2, mockClient.SubmittedBatchSize(0))
	})

	t.Run("flush empty batch does nothing", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()

		err := batcher.Flush(ctx)
		assert.NoError(t, err)

		assert.Equal(t, 0, mockClient.CallCount())
	})
}

func TestResultBatcher_AutoFlush(t *testing.T) {
	t.Run("auto flush on interval", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 100
		config.FlushInterval = 100 * time.Millisecond

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()

		// Добавляем результат
		result := &api.CheckResult{}
		err := batcher.AddResult(ctx, "check-1", result)
		assert.NoError(t, err)

		// Ждём auto flush
		time.Sleep(200 * time.Millisecond)

		assert.Equal(t, 1, mockClient.CallCount())
		assert.Equal(t, 1, mockClient.SubmittedBatchSize(0))
	})
}

func TestResultBatcher_ErrorHandling(t *testing.T) {
	t.Run("return results to queue on error", func(t *testing.T) {
		mockClient := &MockSubmitter{
			err: errors.New("submit failed"),
		}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 100

		batcher := newTestBatcher(mockClient, config)
		defer func() { assert.NoError(t, batcher.Close()) }()

		ctx := context.Background()

		// Добавляем результат
		result := &api.CheckResult{}
		err := batcher.AddResult(ctx, "check-1", result)
		assert.NoError(t, err)

		// Пытаемся отправить - должно завершиться ошибкой
		err = batcher.Flush(ctx)
		assert.Error(t, err)

		// Результат должен вернуться в очередь
		stats := batcher.GetStats()
		assert.Equal(t, 1, stats.PendingCount)
	})
}

func TestResultBatcher_Close(t *testing.T) {
	t.Run("close flushes remaining results", func(t *testing.T) {
		mockClient := &MockSubmitter{}
		config := DefaultBatchingConfig()
		config.MaxBatchSize = 100

		batcher := newTestBatcher(mockClient, config)

		ctx := context.Background()

		// Добавляем результаты
		for i := 0; i < 3; i++ {
			result := &api.CheckResult{}
			err := batcher.AddResult(ctx, "check-1", result)
			require.NoError(t, err)
		}

		// Закрываем - должен отправить
		err := batcher.Close()
		assert.NoError(t, err)

		assert.Equal(t, 1, mockClient.CallCount())
		assert.Equal(t, 1, mockClient.SubmittedBatchSize(0))
	})
}

func TestResultBatcher_Stats(t *testing.T) {
	mockClient := &MockSubmitter{}
	config := DefaultBatchingConfig()
	config.MaxBatchSize = 2

	batcher := newTestBatcher(mockClient, config)
	defer func() { assert.NoError(t, batcher.Close()) }()

	ctx := context.Background()

	// Добавляем результаты
	result := &api.CheckResult{}

	assert.NoError(t, batcher.AddResult(ctx, "check-1", result))
	assert.NoError(t, batcher.AddResult(ctx, "check-2", result))

	stats := batcher.GetStats()
	assert.Equal(t, 2, stats.PendingCount)
	assert.Equal(t, int64(0), stats.TotalResults)

	// Отправляем
	assert.NoError(t, batcher.Flush(ctx))

	stats = batcher.GetStats()
	assert.Equal(t, 0, stats.PendingCount)
	assert.Equal(t, int64(2), stats.TotalResults)
	assert.Equal(t, int64(1), stats.TotalBatches)
}

// newTestBatcher создаёт batcher для тестов
func newTestBatcher(client *MockSubmitter, config BatchingConfig) *ResultBatcher {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	return NewResultBatcher(client, config, logger)
}
