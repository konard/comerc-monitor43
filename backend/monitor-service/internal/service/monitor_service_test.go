package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
	apperrors "github.com/raul/monitor/backend/monitor-service/pkg/errors"
)

// MockMonitorRepository является mock для MonitorRepository.
type MockMonitorRepository struct {
	mock.Mock
}

func (m *MockMonitorRepository) Create(ctx context.Context, monitor *domain.Monitor) error {
	args := m.Called(ctx, monitor)
	return args.Error(0)
}

func (m *MockMonitorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepository) GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*domain.Monitor, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Monitor, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepository) ListByUserIDAndStatus(ctx context.Context, userID uuid.UUID, status domain.MonitorStatus) ([]*domain.Monitor, error) {
	args := m.Called(ctx, userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepository) ListActive(ctx context.Context) ([]*domain.Monitor, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepository) ListDueForCheck(ctx context.Context, limit int) ([]*domain.Monitor, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepository) Update(ctx context.Context, monitor *domain.Monitor) error {
	args := m.Called(ctx, monitor)
	return args.Error(0)
}

func (m *MockMonitorRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockMonitorRepository) UpdateLastCheck(ctx context.Context, id uuid.UUID, lastCheckAt time.Time) error {
	args := m.Called(ctx, id, lastCheckAt)
	return args.Error(0)
}

func (m *MockMonitorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMonitorRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockMonitorRepository) ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	args := m.Called(ctx, userID, name)
	return args.Bool(0), args.Error(1)
}

// MockAuditRepository является mock для AuditRepository.
type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Create(ctx context.Context, entry *interfaces.AuditLogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*interfaces.AuditLogEntry, error) {
	args := m.Called(ctx, monitorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.AuditLogEntry), args.Error(1)
}

func (m *MockAuditRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*interfaces.AuditLogEntry, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.AuditLogEntry), args.Error(1)
}

func (m *MockAuditRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	args := m.Called(ctx, olderThan)
	return int64(args.Int(0)), args.Error(1)
}

// TestMonitorService_CreateMonitor тестирует создание монитора.
func TestMonitorService_CreateMonitor(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New().String()
	req := &dto.CreateMonitorRequest{
		UserID:          userID,
		Tier:            "Free", // Добавляем subscription tier
		Name:            "Test Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
	}

	// Mock: проверяем, что монитор с таким именем не существует
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Test Monitor").Return(false, nil)
	mockRepo.On("CountByUserID", mock.Anything, mock.Anything).Return(0, nil) // Mock для CanCreateMonitor
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("CountByUserID", mock.Anything, mock.Anything).Return(0, nil) // Mock для CanCreateMonitor

	// Выполняем
	resp, err := service.CreateMonitor(context.Background(), req)

	// Проверяем
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Test Monitor", resp.Name)
	assert.Equal(t, "https://example.com", resp.URL)

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_CreateMonitor_DuplicateName тестирует создание монитора с дубликатным именем.
func TestMonitorService_CreateMonitor_DuplicateName(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New().String()
	req := &dto.CreateMonitorRequest{
		UserID:          userID,
		Tier:            "Free",
		Name:            "Existing Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
	}

	// Mock: монитор с таким именем уже существует
	mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Existing Monitor").Return(true, nil)

	// Выполняем
	resp, err := service.CreateMonitor(context.Background(), req)

	// Проверяем
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "already exists")

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_CreateMonitor_ExistsByNameError тестирует ошибку при проверке существования имени.
func TestMonitorService_CreateMonitor_ExistsByNameError(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New().String()
	req := &dto.CreateMonitorRequest{
		UserID:          userID,
		Tier:            "Free", // Добавляем subscription tier
		Name:            "Test Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
	}

	// Mock: ошибка при проверке существования
	mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Test Monitor").Return(false, errors.New("database error"))

	// Выполняем
	resp, err := service.CreateMonitor(context.Background(), req)

	// Проверяем
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to check monitor existence")

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_CreateMonitor_InvalidUserData тестирует создание с невалидными данными пользователя.
func TestMonitorService_CreateMonitor_InvalidUserData(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	req := &dto.CreateMonitorRequest{
		UserID:          "invalid-uuid", // Невалидный UUID
		Tier:            "Free",
		Name:            "Test Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
	}

	// Выполняем
	resp, err := service.CreateMonitor(context.Background(), req)

	// Проверяем
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid user_id")
}

// TestMonitorService_GetMonitor тестирует получение монитора.
func TestMonitorService_GetMonitor(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	// Mock
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	// Выполняем
	resp, err := service.GetMonitor(context.Background(), monitor.ID.String(), userID.String())

	// Проверяем
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, monitor.ID.String(), resp.ID)
	assert.Equal(t, "Test", resp.Name)

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_GetMonitor_NotFound тестирует получение несуществующего монитора.
func TestMonitorService_GetMonitor_NotFound(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	randomID := uuid.New()

	// Mock: монитор не найден
	mockRepo.On("GetByID", mock.Anything, randomID).Return(nil, interfaces.ErrMonitorNotFound)

	// Выполняем
	resp, err := service.GetMonitor(context.Background(), randomID.String(), userID.String())

	// Проверяем
	assert.Error(t, err)
	assert.Nil(t, resp)

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_GetMonitor_ErrorCases тестирует error cases для GetMonitor.
func TestMonitorService_GetMonitor_ErrorCases(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("invalid monitor ID", func(t *testing.T) {
		resp, err := service.GetMonitor(context.Background(), "invalid-uuid", uuid.New().String())

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		resp, err := service.GetMonitor(context.Background(), uuid.New().String(), "invalid-uuid")

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("repository get error", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		mockRepo.On("GetByID", mock.Anything, monitorID).
			Return(nil, errors.New("database error"))

		resp, err := service.GetMonitor(context.Background(), monitorID.String(), userID.String())

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to get monitor")
		assert.Contains(t, err.Error(), "database error")
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_ListMonitors тестирует получение списка мониторов.
func TestMonitorService_ListMonitors(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor1 := mustNewMonitor(t, userID, "Monitor 1", "https://example1.com", 60)
	monitor2 := mustNewMonitor(t, userID, "Monitor 2", "https://example2.com", 60)

	req := &dto.ListMonitorsRequest{
		UserID: userID.String(),
		Limit:  10,
		Offset: 0,
	}

	// Mock
	monitors := []*domain.Monitor{monitor1, monitor2}
	mockRepo.On("ListByUserID", mock.Anything, userID, 10, 0).Return(monitors, nil)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(2, nil)

	// Выполняем
	resp, err := service.ListMonitors(context.Background(), req)

	// Проверяем
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Monitors, 2)
	assert.Equal(t, 2, resp.Total)

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_ListMonitors_ServiceRole проверяет, что при SkipUserFilter=true
// (сервисный вызов, например scheduler-service) возвращаются все активные мониторы
// без попытки распарсить UserID как UUID.
func TestMonitorService_ListMonitors_ServiceRole(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userA := uuid.New()
	userB := uuid.New()
	monitor1 := mustNewMonitor(t, userA, "Monitor A", "https://example-a.com", 60)
	monitor2 := mustNewMonitor(t, userB, "Monitor B", "https://example-b.com", 60)

	req := &dto.ListMonitorsRequest{
		UserID:         "scheduler-service", // не UUID
		Limit:          10,
		Offset:         0,
		SkipUserFilter: true,
	}

	monitors := []*domain.Monitor{monitor1, monitor2}
	mockRepo.On("ListActive", mock.Anything).Return(monitors, nil)

	resp, err := service.ListMonitors(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Monitors, 2)
	assert.Equal(t, 2, resp.Total)

	// Убеждаемся, что user-scoped методы не вызывались.
	mockRepo.AssertNotCalled(t, "ListByUserID", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "CountByUserID", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

// TestMonitorService_DeleteMonitor тестирует удаление монитора.
func TestMonitorService_DeleteMonitor(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	// Mock
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Delete", mock.Anything, monitor.ID).Return(nil)

	// Выполняем
	err := service.DeleteMonitor(context.Background(), monitor.ID.String(), userID.String())

	// Проверяем
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_PauseMonitor тестирует приостановку монитора.
func TestMonitorService_PauseMonitor(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusUp

	// Mock
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	// Выполняем
	req := &dto.PauseMonitorRequest{
		ID:     monitor.ID.String(),
		UserID: userID.String(),
	}

	err := service.PauseMonitor(context.Background(), req)

	// Проверяем
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// TestMonitorService_ResumeMonitor тестирует возобновление монитора.
func TestMonitorService_ResumeMonitor(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusPaused

	// Mock
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	// Выполняем
	req := &dto.ResumeMonitorRequest{
		ID:     monitor.ID.String(),
		UserID: userID.String(),
	}

	err := service.ResumeMonitor(context.Background(), req)

	// Проверяем
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// TestParseWorkingDays тестирует парсинг рабочих дней.
func TestParseWorkingDays(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		days     []string
		expected []time.Weekday
	}{
		{
			name:     "empty slice",
			days:     []string{},
			expected: nil,
		},
		{
			name:     "nil slice",
			days:     nil,
			expected: nil,
		},
		{
			name:     "single day",
			days:     []string{"Monday"},
			expected: []time.Weekday{time.Monday},
		},
		{
			name:     "all weekdays",
			days:     []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
			expected: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		},
		{
			name:     "including weekend",
			days:     []string{"Saturday", "Sunday"},
			expected: []time.Weekday{time.Saturday, time.Sunday},
		},
		{
			name:     "mixed order",
			days:     []string{"Friday", "Monday", "Wednesday"},
			expected: []time.Weekday{time.Friday, time.Monday, time.Wednesday},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWorkingDays(tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWorkingDaysToStrings тестирует конвертацию рабочих дней в строки.
func TestWorkingDaysToStrings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		days     []time.Weekday
		expected []string
	}{
		{
			name:     "empty slice",
			days:     []time.Weekday{},
			expected: nil,
		},
		{
			name:     "nil slice",
			days:     nil,
			expected: nil,
		},
		{
			name:     "single day",
			days:     []time.Weekday{time.Monday},
			expected: []string{"Monday"},
		},
		{
			name:     "all weekdays",
			days:     []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			expected: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
		},
		{
			name:     "including weekend",
			days:     []time.Weekday{time.Saturday, time.Sunday},
			expected: []string{"Saturday", "Sunday"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := workingDaysToStrings(tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMonitorService_MonitorToResponse тестирует конвертацию монитора в response.
func TestMonitorService_MonitorToResponse(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("all fields populated", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		// Set all optional fields
		monitor.CheckType = "HTTP"
		monitor.TimeoutSeconds = 30

		// Working hours
		startTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC)
		monitor.WorkingHoursStart = &startTime
		monitor.WorkingHoursEnd = &endTime
		monitor.WorkingDays = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}

		// Thresholds
		responseTimeThreshold := 500
		monitor.DegradedResponseTimeThreshold = &responseTimeThreshold
		failureRateThreshold := 30
		monitor.DegradedFailureRateThreshold = &failureRateThreshold

		// Last check
		lastCheck := time.Now()
		monitor.LastCheckAt = &lastCheck

		resp := service.monitorToResponse(monitor)

		assert.NotNil(t, resp)
		assert.Equal(t, monitor.ID.String(), resp.ID)
		assert.Equal(t, monitor.UserID.String(), resp.UserID)
		assert.Equal(t, "Test", resp.Name)
		assert.Equal(t, "https://example.com", resp.URL)
		assert.Equal(t, "HTTP", resp.CheckType)
		assert.Equal(t, 60, resp.IntervalSeconds)
		assert.Equal(t, 30, resp.TimeoutSeconds)
		assert.Equal(t, string(monitor.Status), resp.Status)

		// Optional fields
		assert.NotNil(t, resp.WorkingHoursStart)
		assert.Equal(t, "09:00", *resp.WorkingHoursStart)
		assert.NotNil(t, resp.WorkingHoursEnd)
		assert.Equal(t, "17:00", *resp.WorkingHoursEnd)
		assert.Len(t, resp.WorkingDays, 5)
		assert.Equal(t, []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}, resp.WorkingDays)
		assert.NotNil(t, resp.DegradedResponseTimeThreshold)
		assert.Equal(t, 500, *resp.DegradedResponseTimeThreshold)
		assert.NotNil(t, resp.DegradedFailureRateThreshold)
		assert.Equal(t, 30, *resp.DegradedFailureRateThreshold)
		assert.NotNil(t, resp.LastCheckAt)
		assert.Equal(t, lastCheck, *resp.LastCheckAt)
	})

	t.Run("minimal fields", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		resp := service.monitorToResponse(monitor)

		assert.NotNil(t, resp)
		assert.Equal(t, monitor.ID.String(), resp.ID)
		assert.Equal(t, monitor.UserID.String(), resp.UserID)
		assert.Equal(t, "Test", resp.Name)
		assert.Equal(t, "https://example.com", resp.URL)
		assert.Equal(t, 60, resp.IntervalSeconds)
		assert.Equal(t, string(monitor.Status), resp.Status)

		// Optional fields should be nil/zero (except WorkingDays which defaults to all days)
		assert.Nil(t, resp.WorkingHoursStart)
		assert.Nil(t, resp.WorkingHoursEnd)
		// WorkingDays defaults to all days when creating a monitor
		assert.NotNil(t, resp.WorkingDays) // Has default value
		assert.Nil(t, resp.DegradedResponseTimeThreshold)
		assert.Nil(t, resp.DegradedFailureRateThreshold)
		assert.Nil(t, resp.LastCheckAt)
	})

	t.Run("partial optional fields", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		// Set only some optional fields
		responseTimeThreshold := 1000
		monitor.DegradedResponseTimeThreshold = &responseTimeThreshold

		startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		monitor.WorkingHoursStart = &startTime

		resp := service.monitorToResponse(monitor)

		assert.NotNil(t, resp)
		assert.NotNil(t, resp.DegradedResponseTimeThreshold)
		assert.Equal(t, 1000, *resp.DegradedResponseTimeThreshold)
		assert.Nil(t, resp.DegradedFailureRateThreshold)
		assert.NotNil(t, resp.WorkingHoursStart)
		assert.Equal(t, "10:00", *resp.WorkingHoursStart)
		assert.Nil(t, resp.WorkingHoursEnd)
	})
}

// TestMonitorService_UpdateMonitor_PartialUpdates тестирует частичные обновления.
func TestMonitorService_UpdateMonitor_PartialUpdates(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("update only name", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Old Name", "https://example.com", 60)

		newName := "New Name"
		req := &dto.UpdateMonitorRequest{
			ID:     monitor.ID.String(),
			UserID: userID.String(),
			Name:   &newName,
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "New Name", resp.Name)
		assert.Equal(t, "https://example.com", resp.URL) // unchanged
		mockRepo.AssertExpectations(t)
	})

	t.Run("update only URL", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://old.com", 60)

		newURL := "https://new.com"
		req := &dto.UpdateMonitorRequest{
			ID:     monitor.ID.String(),
			UserID: userID.String(),
			URL:    &newURL,
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Test", resp.Name) // unchanged
		assert.Equal(t, "https://new.com", resp.URL)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update only interval", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		newInterval := 120
		req := &dto.UpdateMonitorRequest{
			ID:              monitor.ID.String(),
			UserID:          userID.String(),
			IntervalSeconds: &newInterval,
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 120, resp.IntervalSeconds)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update only thresholds", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		newResponseTime := 800
		newFailureRate := 40
		req := &dto.UpdateMonitorRequest{
			ID:                            monitor.ID.String(),
			UserID:                        userID.String(),
			DegradedResponseTimeThreshold: &newResponseTime,
			DegradedFailureRateThreshold:  &newFailureRate,
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.DegradedResponseTimeThreshold)
		assert.Equal(t, 800, *resp.DegradedResponseTimeThreshold)
		assert.NotNil(t, resp.DegradedFailureRateThreshold)
		assert.Equal(t, 40, *resp.DegradedFailureRateThreshold)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update working hours only", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		startTime := "09:00"
		endTime := "18:00"
		days := []string{"Monday", "Wednesday", "Friday"}
		req := &dto.UpdateMonitorRequest{
			ID:                monitor.ID.String(),
			UserID:            userID.String(),
			WorkingHoursStart: &startTime,
			WorkingHoursEnd:   &endTime,
			WorkingDays:       days,
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.WorkingHoursStart)
		assert.Equal(t, "09:00", *resp.WorkingHoursStart)
		assert.NotNil(t, resp.WorkingHoursEnd)
		assert.Equal(t, "18:00", *resp.WorkingHoursEnd)
		assert.Len(t, resp.WorkingDays, 3)
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_UpdateMonitor_ErrorCases тестирует error cases для UpdateMonitor.
func TestMonitorService_UpdateMonitor_ErrorCases(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("invalid monitor ID", func(t *testing.T) {
		newName := "New Name"
		req := &dto.UpdateMonitorRequest{
			ID:     "invalid-uuid",
			UserID: uuid.New().String(),
			Name:   &newName,
		}

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		newName := "New Name"
		req := &dto.UpdateMonitorRequest{
			ID:     uuid.New().String(),
			UserID: "invalid-uuid",
			Name:   &newName,
		}

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("monitor not found", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()
		newName := "New Name"

		req := &dto.UpdateMonitorRequest{
			ID:     monitorID.String(),
			UserID: userID.String(),
			Name:   &newName,
		}

		mockRepo.On("GetByID", mock.Anything, monitorID).
			Return(nil, apperrors.ErrMonitorNotFound)

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, apperrors.ErrMonitorNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository update error", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		newName := "New Name"
		req := &dto.UpdateMonitorRequest{
			ID:     monitor.ID.String(),
			UserID: userID.String(),
			Name:   &newName,
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).
			Return(errors.New("database error"))

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to update monitor")
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid working hours", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		startTime := "invalid-time"
		endTime := "17:00"

		req := &dto.UpdateMonitorRequest{
			ID:                monitor.ID.String(),
			UserID:            userID.String(),
			WorkingHoursStart: &startTime,
			WorkingHoursEnd:   &endTime,
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

		resp, err := service.UpdateMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid working hours")
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_CreateMonitor_ValidationEdgeCases тестирует edge cases валидации при создании.
func TestMonitorService_CreateMonitor_ValidationEdgeCases(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("timeout greater than interval", func(t *testing.T) {
		userID := uuid.New().String()
		req := &dto.CreateMonitorRequest{
			UserID:          userID,
			Tier:            "Free", // Добавляем subscription tier для валидации
			Name:            "Test Monitor",
			URL:             "https://example.com",
			IntervalSeconds: 60,
			TimeoutSeconds:  120, // timeout > interval
		}

		mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Test Monitor").Return(false, nil)
		mockRepo.On("CountByUserID", mock.Anything, mock.Anything).Return(0, nil) // Mock для CanCreateMonitor

		resp, err := service.CreateMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "timeout must be less than interval")
		mockRepo.AssertExpectations(t)
	})

	t.Run("timeout equals interval", func(t *testing.T) {
		userID := uuid.New().String()
		req := &dto.CreateMonitorRequest{
			UserID:          userID,
			Tier:            "Free", // Добавляем subscription tier для валидации
			Name:            "Test Monitor",
			URL:             "https://example.com",
			IntervalSeconds: 60,
			TimeoutSeconds:  60, // timeout == interval
		}

		mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Test Monitor").Return(false, nil)
		mockRepo.On("CountByUserID", mock.Anything, mock.Anything).Return(0, nil) // Mock для CanCreateMonitor

		resp, err := service.CreateMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "timeout must be less than interval")
		mockRepo.AssertExpectations(t)
	})

	t.Run("zero timeout is allowed", func(t *testing.T) {
		userID := uuid.New().String()
		req := &dto.CreateMonitorRequest{
			UserID:          userID,
			Tier:            "Free", // Добавляем subscription tier для валидации
			Name:            "Test Monitor",
			URL:             "https://example.com",
			IntervalSeconds: 60,
			TimeoutSeconds:  0, // zero means use default
		}

		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Test Monitor").Return(false, nil)
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		resp, err := service.CreateMonitor(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRepo.AssertExpectations(t)
	})

	t.Run("with all optional fields", func(t *testing.T) {
		userID := uuid.New().String()
		startTime := "09:00"
		endTime := "17:00"
		days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
		responseTimeThreshold := 500
		failureRateThreshold := 30

		req := &dto.CreateMonitorRequest{
			UserID:                        userID,
			Tier:                          "Free",
			Name:                          "Test Monitor",
			URL:                           "https://example.com",
			CheckType:                     "HTTP",
			IntervalSeconds:               60,
			TimeoutSeconds:                30,
			WorkingHoursStart:             startTime,
			WorkingHoursEnd:               endTime,
			WorkingDays:                   days,
			DegradedResponseTimeThreshold: &responseTimeThreshold,
			DegradedFailureRateThreshold:  &failureRateThreshold,
		}

		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Test Monitor").Return(false, nil)
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("CountByUserID", mock.Anything, mock.Anything).Return(0, nil) // Mock для CanCreateMonitor

		resp, err := service.CreateMonitor(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "HTTP", resp.CheckType)
		assert.Equal(t, 30, resp.TimeoutSeconds)
		assert.NotNil(t, resp.WorkingHoursStart)
		assert.Equal(t, "09:00", *resp.WorkingHoursStart)
		assert.NotNil(t, resp.WorkingHoursEnd)
		assert.Equal(t, "17:00", *resp.WorkingHoursEnd)
		assert.Len(t, resp.WorkingDays, 5)
		assert.NotNil(t, resp.DegradedResponseTimeThreshold)
		assert.Equal(t, 500, *resp.DegradedResponseTimeThreshold)
		assert.NotNil(t, resp.DegradedFailureRateThreshold)
		assert.Equal(t, 30, *resp.DegradedFailureRateThreshold)
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_ListMonitors_ErrorCases тестирует error cases для ListMonitors.
func TestMonitorService_ListMonitors_ErrorCases(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("invalid user ID", func(t *testing.T) {
		req := &dto.ListMonitorsRequest{
			UserID: "invalid-uuid",
			Limit:  10,
			Offset: 0,
		}

		resp, err := service.ListMonitors(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("invalid status", func(t *testing.T) {
		req := &dto.ListMonitorsRequest{
			UserID: uuid.New().String(),
			Status: "INVALID_STATUS",
			Limit:  10,
			Offset: 0,
		}

		resp, err := service.ListMonitors(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("repository list error", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.ListMonitorsRequest{
			UserID: userID.String(),
			Limit:  10,
			Offset: 0,
		}

		mockRepo.On("ListByUserID", mock.Anything, userID, 10, 0).
			Return(nil, errors.New("database error"))

		resp, err := service.ListMonitors(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to list monitors")
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository count error", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.ListMonitorsRequest{
			UserID: userID.String(),
			Limit:  10,
			Offset: 0,
		}

		mockRepo.On("ListByUserID", mock.Anything, userID, 10, 0).
			Return([]*domain.Monitor{}, nil)
		mockRepo.On("CountByUserID", mock.Anything, userID).
			Return(0, errors.New("database error"))

		resp, err := service.ListMonitors(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to count monitors")
		mockRepo.AssertExpectations(t)
	})

	t.Run("filters by status", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.ListMonitorsRequest{
			UserID: userID.String(),
			Status: "UP",
			Limit:  10,
			Offset: 0,
		}

		mockRepo.On("ListByUserIDAndStatus", mock.Anything, userID, domain.StatusUp).
			Return([]*domain.Monitor{}, nil)
		mockRepo.On("CountByUserID", mock.Anything, userID).
			Return(0, nil)

		resp, err := service.ListMonitors(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRepo.AssertExpectations(t)
	})

	t.Run("uses default limit when limit <= 0", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.ListMonitorsRequest{
			UserID: userID.String(),
			Limit:  0,
			Offset: 0,
		}

		mockRepo.On("ListByUserID", mock.Anything, userID, 100, 0).
			Return([]*domain.Monitor{}, nil)
		mockRepo.On("CountByUserID", mock.Anything, userID).
			Return(0, nil)

		resp, err := service.ListMonitors(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 100, resp.Limit)
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_DeleteMonitor_ErrorCases тестирует error cases для DeleteMonitor.
func TestMonitorService_DeleteMonitor_ErrorCases(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("invalid monitor ID", func(t *testing.T) {
		err := service.DeleteMonitor(context.Background(), "invalid-uuid", uuid.New().String())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		err := service.DeleteMonitor(context.Background(), uuid.New().String(), "invalid-uuid")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("monitor not found", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		mockRepo.On("GetByID", mock.Anything, monitorID).
			Return(nil, apperrors.ErrMonitorNotFound)

		err := service.DeleteMonitor(context.Background(), monitorID.String(), userID.String())

		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrMonitorNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository delete error", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Delete", mock.Anything, monitor.ID).
			Return(errors.New("database error"))

		err := service.DeleteMonitor(context.Background(), monitor.ID.String(), userID.String())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete monitor")
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_PauseMonitor_ErrorCases тестирует error cases для PauseMonitor.
func TestMonitorService_PauseMonitor_ErrorCases(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("invalid monitor ID", func(t *testing.T) {
		req := &dto.PauseMonitorRequest{
			ID:     "invalid-uuid",
			UserID: uuid.New().String(),
		}

		err := service.PauseMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		req := &dto.PauseMonitorRequest{
			ID:     uuid.New().String(),
			UserID: "invalid-uuid",
		}

		err := service.PauseMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("monitor not found", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		req := &dto.PauseMonitorRequest{
			ID:     monitorID.String(),
			UserID: userID.String(),
		}

		mockRepo.On("GetByID", mock.Anything, monitorID).
			Return(nil, apperrors.ErrMonitorNotFound)

		err := service.PauseMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrMonitorNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update error", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = domain.StatusUp

		req := &dto.PauseMonitorRequest{
			ID:     monitor.ID.String(),
			UserID: userID.String(),
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).
			Return(errors.New("database error"))

		err := service.PauseMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update monitor")
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_ResumeMonitor_ErrorCases тестирует error cases для ResumeMonitor.
func TestMonitorService_ResumeMonitor_ErrorCases(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	t.Run("invalid monitor ID", func(t *testing.T) {
		req := &dto.ResumeMonitorRequest{
			ID:     "invalid-uuid",
			UserID: uuid.New().String(),
		}

		err := service.ResumeMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		req := &dto.ResumeMonitorRequest{
			ID:     uuid.New().String(),
			UserID: "invalid-uuid",
		}

		err := service.ResumeMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("monitor not found", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		req := &dto.ResumeMonitorRequest{
			ID:     monitorID.String(),
			UserID: userID.String(),
		}

		mockRepo.On("GetByID", mock.Anything, monitorID).
			Return(nil, apperrors.ErrMonitorNotFound)

		err := service.ResumeMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrMonitorNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update error", func(t *testing.T) {
		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = domain.StatusPaused

		req := &dto.ResumeMonitorRequest{
			ID:     monitor.ID.String(),
			UserID: userID.String(),
		}

		mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
		mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("Update", mock.Anything, mock.Anything).
			Return(errors.New("database error"))

		err := service.ResumeMonitor(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update monitor")
		mockRepo.AssertExpectations(t)
	})
}

// TestMonitorService_CreateAuditLog_Error тестирует что операция не падает при ошибке audit log.
func TestMonitorService_CreateAuditLog_Error(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	req := &dto.CreateMonitorRequest{
		UserID:          userID.String(),
		Name:            "Test Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
		Tier:            "Free",
	}

	mockRepo.On("ExistsByName", mock.Anything, userID, "Test Monitor").Return(false, nil)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil) // For authorization check
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	// Audit log fails, but CreateMonitor should still succeed
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(errors.New("audit log error"))

	resp, err := service.CreateMonitor(context.Background(), req)

	// Assert - operation should succeed despite audit log failure
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Test Monitor", resp.Name)
	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}
