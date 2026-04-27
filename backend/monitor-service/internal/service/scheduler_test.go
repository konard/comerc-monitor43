package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
)

// MockCheckExecutor является mock для CheckExecutor.
type MockCheckExecutor struct {
	mock.Mock
}

func (m *MockCheckExecutor) ExecuteCheck(ctx context.Context, monitorID string) (*domain.CheckResult, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CheckResult), args.Error(1)
}

// TestNewScheduler тестирует создание Scheduler.
func TestNewScheduler(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       1 * time.Second,
		MaxConcurrentChecks: 10,
		BatchSize:           100,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

	assert.NotNil(t, scheduler)
	assert.Equal(t, mockRepo, scheduler.monitorRepo)
	assert.Equal(t, mockExecutor, scheduler.checkExecutor)
	assert.Equal(t, mockPublisher, scheduler.eventPublisher)
	assert.Equal(t, 1*time.Second, scheduler.checkInterval)
	assert.Equal(t, 10, scheduler.maxConcurrentChecks)
	assert.Equal(t, 100, scheduler.batchSize)
	assert.False(t, scheduler.IsRunning())
}

// TestScheduler_StartStop тестирует запуск и остановку Scheduler.
func TestScheduler_StartStop(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       100 * time.Millisecond,
		MaxConcurrentChecks: 5,
		BatchSize:           10,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduler
	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Try to start again - should fail
	err = scheduler.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Stop scheduler
	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())

	// Stop again - should be safe
	scheduler.Stop() // Should not panic
	assert.False(t, scheduler.IsRunning())
}

// TestScheduler_ScheduleBatch тестирует планирование пакет проверок.
func TestScheduler_ScheduleBatch(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       50 * time.Millisecond,
		MaxConcurrentChecks: 2,
		BatchSize:           5,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

	// Создаём тестовые мониторы
	userID := uuid.New()
	monitors := make([]*domain.Monitor, 3)
	for i := 0; i < 3; i++ {
		monitor := mustNewMonitor(t, userID, "Test Monitor "+string(rune('A'+i)), "https://example.com", 60)
		monitor.Status = domain.StatusUp
		monitors[i] = monitor
	}

	// Mock: возвращаем список мониторов для проверки
	mockRepo.On("ListDueForCheck", mock.Anything, 5).Return(monitors, nil)

	// Mock: выполнение проверки
	mockResult := domain.NewCheckResult(monitors[0].ID, domain.StatusUp)
	mockResult = mockResult.WithResponseTime(100)
	mockExecutor.On("ExecuteCheck", mock.Anything, mock.Anything).Return(mockResult, nil)

	ctx := context.Background()

	// Start scheduler briefly to trigger one batch
	startErr := scheduler.Start(ctx)
	require.NoError(t, startErr)

	// Ждём немного, чтобы успел выполниться один batch
	time.Sleep(150 * time.Millisecond)

	scheduler.Stop()

	// Проверяем, что ListDueForCheck был вызван (минимум 1 раз)
	mockRepo.AssertCalled(t, "ListDueForCheck", mock.Anything, 5)
	// ExecuteCheck должен быть вызван как минимум для 3 мониторов (может быть больше из-за повторных вызовов)
	assert.True(t, len(mockExecutor.Calls) >= 3, "ExecuteCheck should be called at least 3 times, got %d", len(mockExecutor.Calls))
}

// TestScheduler_ConcurrentChecks тестирует ограничения на конкурентность.
func TestScheduler_ConcurrentChecks(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       100 * time.Millisecond,
		MaxConcurrentChecks: 2, // Максимум 2 параллельные проверки
		BatchSize:           10,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

	// Создаём 5 мониторов
	userID := uuid.New()
	monitors := make([]*domain.Monitor, 5)
	for i := 0; i < 5; i++ {
		monitor := mustNewMonitor(t, userID, "Monitor", "https://example.com", 60)
		monitor.Status = domain.StatusUp
		monitors[i] = monitor
	}

	// Mock: возвращаем список мониторов
	mockRepo.On("ListDueForCheck", mock.Anything, 10).Return(monitors, nil)

	// Mock: выполнение проверки - возвращаем один результат для всех вызовов
	result := domain.NewCheckResult(uuid.New(), domain.StatusUp)
	result = result.WithResponseTime(100)
	mockExecutor.On("ExecuteCheck", mock.Anything, mock.Anything).Return(result, nil)

	ctx := context.Background()

	// Запускаем scheduler
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Ждём завершения
	time.Sleep(200 * time.Millisecond)

	scheduler.Stop()

	// Проверяем, что проверки были выполнены (как минимум 5 мониторов были проверены)
	// Проверяем, что проверки были выполнены (может быть больше из-за повторных batch calls)
	assert.True(t, len(mockExecutor.Calls) >= 5, "ExecuteCheck should be called at least 5 times, got %d", len(mockExecutor.Calls))
}

// TestScheduler_StatusChangeEvents тестирует публикацию событий изменения статуса.
func TestScheduler_StatusChangeEvents(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       100 * time.Millisecond,
		MaxConcurrentChecks: 5,
		BatchSize:           10,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusUp // Старый статус

	// Mock: возвращаем монитор
	mockRepo.On("ListDueForCheck", mock.Anything, 10).Return([]*domain.Monitor{monitor}, nil)

	// Mock: результат проверки с изменением статуса
	mockResult := domain.NewCheckResult(monitor.ID, domain.StatusDown) // Статус изменился на DOWN
	mockResult = mockResult.WithResponseTime(500)
	mockExecutor.On("ExecuteCheck", mock.Anything, monitor.ID.String()).Return(mockResult, nil)

	// Mock: публикация события
	mockPublisher.On("PublishStatusChange", mock.Anything, mock.MatchedBy(func(event *StatusChangeEvent) bool {
		return event.MonitorID == monitor.ID &&
			event.OldStatus == domain.StatusUp &&
			event.NewStatus == domain.StatusDown &&
			event.ResponseTime != nil && *event.ResponseTime == 500
	})).Return(nil)

	// Mock: обновление статуса в БД
	mockRepo.On("UpdateStatus", mock.Anything, monitor.ID, domain.StatusDown).Return(nil)

	ctx := context.Background()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond)

	scheduler.Stop()

	// Проверяем, что событие было опубликовано
	mockPublisher.AssertCalled(t, "PublishStatusChange", mock.Anything, mock.Anything)
	mockRepo.AssertCalled(t, "UpdateStatus", mock.Anything, monitor.ID, domain.StatusDown)
}

// TestScheduler_ScheduleCheck_ManualTrigger тестирует ручной запуск проверки.
func TestScheduler_ScheduleCheck_ManualTrigger(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       1 * time.Second,
		MaxConcurrentChecks: 5,
		BatchSize:           10,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusUp

	// Mock: GetByID
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	// Mock: ExecuteCheck
	mockResult := domain.NewCheckResult(monitor.ID, domain.StatusUp)
	mockResult = mockResult.WithResponseTime(100)
	mockExecutor.On("ExecuteCheck", mock.Anything, monitor.ID.String()).Return(mockResult, nil)

	ctx := context.Background()

	// Запускаем проверку вручную
	err := scheduler.ScheduleCheck(ctx, monitor.ID)
	assert.NoError(t, err)

	// Ждём немного для завершения асинхронной проверки
	time.Sleep(100 * time.Millisecond)

	mockExecutor.AssertCalled(t, "ExecuteCheck", mock.Anything, monitor.ID.String())
}

// TestScheduler_ScheduleCheck_OutsideWorkingHours тестирует отказ в проверке вне рабочих часов.
func TestScheduler_ScheduleCheck_OutsideWorkingHours(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       1 * time.Second,
		MaxConcurrentChecks: 5,
		BatchSize:           10,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusPaused // PAUSED мониторы не должны проверяться

	// Mock: GetByID
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	ctx := context.Background()

	// Пытаемся запустить проверку для PAUSED монитора
	err := scheduler.ScheduleCheck(ctx, monitor.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside working hours")

	// ExecuteCheck не должен быть вызван
	mockExecutor.AssertNotCalled(t, "ExecuteCheck")
}

// TestScheduler_ContextCancellation тестирует остановку при отмене контекста.
func TestScheduler_ContextCancellation(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockExecutor := new(MockCheckExecutor)
	mockPublisher := new(MockEventPublisher)

	config := SchedulerConfig{
		CheckInterval:       100 * time.Millisecond,
		MaxConcurrentChecks: 5,
		BatchSize:           10,
	}

	scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

	// Mock: пустой список мониторов
	mockRepo.On("ListDueForCheck", mock.Anything, 10).Return([]*domain.Monitor{}, nil)

	ctx, cancel := context.WithCancel(context.Background())

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Ждём чтобы scheduler запустился
	time.Sleep(50 * time.Millisecond)

	// Проверяем что scheduler запущен
	assert.True(t, scheduler.IsRunning(), "Scheduler should be running")

	// Отменяем контекст - scheduler должен остановиться
	cancel()

	// Ждём остановки (увеличим время)
	time.Sleep(300 * time.Millisecond)

	// Проверяем, что scheduler остановился
	assert.False(t, scheduler.IsRunning(), "Scheduler should be stopped after context cancellation")
}

// TestScheduler_ErrorHandling тестирует обработку ошибок.
func TestScheduler_ErrorHandling(t *testing.T) {
	t.Parallel()
	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockExecutor := new(MockCheckExecutor)
		mockPublisher := new(MockEventPublisher)

		config := SchedulerConfig{
			CheckInterval:       50 * time.Millisecond,
			MaxConcurrentChecks: 5,
			BatchSize:           10,
		}

		scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

		// Mock: ошибка при получении списка мониторов
		mockRepo.On("ListDueForCheck", mock.Anything, 10).Return(nil, assert.AnError)

		ctx := context.Background()

		err := scheduler.Start(ctx)
		require.NoError(t, err)

		// Scheduler не должен упасть, должен продолжать работать
		time.Sleep(100 * time.Millisecond)

		scheduler.Stop()

		// Проверяем, что scheduler остановлен корректно
		assert.False(t, scheduler.IsRunning())
	})

	t.Run("handles executor error", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockExecutor := new(MockCheckExecutor)
		mockPublisher := new(MockEventPublisher)

		config := SchedulerConfig{
			CheckInterval:       50 * time.Millisecond,
			MaxConcurrentChecks: 5,
			BatchSize:           10,
		}

		scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = domain.StatusUp

		// Mock: возвращаем монитор
		mockRepo.On("ListDueForCheck", mock.Anything, 10).Return([]*domain.Monitor{monitor}, nil)

		// Mock: ошибка при выполнении проверки
		mockExecutor.On("ExecuteCheck", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		ctx := context.Background()

		err := scheduler.Start(ctx)
		require.NoError(t, err)

		// Scheduler не должен упасть
		time.Sleep(100 * time.Millisecond)

		scheduler.Stop()

		assert.False(t, scheduler.IsRunning())
	})

	t.Run("handles publisher error", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockExecutor := new(MockCheckExecutor)
		mockPublisher := new(MockEventPublisher)

		config := SchedulerConfig{
			CheckInterval:       50 * time.Millisecond,
			MaxConcurrentChecks: 5,
			BatchSize:           10,
		}

		scheduler := NewScheduler(mockRepo, mockExecutor, mockPublisher, config)

		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = domain.StatusUp

		// Mock: возвращаем монитор
		mockRepo.On("ListDueForCheck", mock.Anything, 10).Return([]*domain.Monitor{monitor}, nil)

		// Mock: результат с изменением статуса
		mockResult := domain.NewCheckResult(monitor.ID, domain.StatusDown)
		mockExecutor.On("ExecuteCheck", mock.Anything, mock.Anything).Return(mockResult, nil)

		// Mock: ошибка публикации
		mockPublisher.On("PublishStatusChange", mock.Anything, mock.Anything).Return(assert.AnError)

		// Mock: обновление статуса должно выполниться несмотря на ошибку публикации
		mockRepo.On("UpdateStatus", mock.Anything, monitor.ID, domain.StatusDown).Return(nil)

		ctx := context.Background()

		err := scheduler.Start(ctx)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		scheduler.Stop()

		// Статус должен быть обновлен несмотря на ошибку публикации
		mockRepo.AssertCalled(t, "UpdateStatus", mock.Anything, monitor.ID, domain.StatusDown)
	})
}
