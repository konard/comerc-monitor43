package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	apperrors "github.com/raul/monitor/backend/monitor-service/pkg/errors"
)

// MockMonitorRepository для тестов
type MockMonitorRepositoryForAuthz struct {
	mock.Mock
}

func (m *MockMonitorRepositoryForAuthz) Create(ctx context.Context, monitor *domain.Monitor) error {
	args := m.Called(ctx, monitor)
	return args.Error(0)
}

func (m *MockMonitorRepositoryForAuthz) GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepositoryForAuthz) GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*domain.Monitor, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepositoryForAuthz) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Monitor, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepositoryForAuthz) ListByUserIDAndStatus(ctx context.Context, userID uuid.UUID, status domain.MonitorStatus) ([]*domain.Monitor, error) {
	args := m.Called(ctx, userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepositoryForAuthz) ListActive(ctx context.Context) ([]*domain.Monitor, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepositoryForAuthz) ListDueForCheck(ctx context.Context, limit int) ([]*domain.Monitor, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Monitor), args.Error(1)
}

func (m *MockMonitorRepositoryForAuthz) Update(ctx context.Context, monitor *domain.Monitor) error {
	args := m.Called(ctx, monitor)
	return args.Error(0)
}

func (m *MockMonitorRepositoryForAuthz) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockMonitorRepositoryForAuthz) UpdateLastCheck(ctx context.Context, id uuid.UUID, lastCheckAt time.Time) error {
	args := m.Called(ctx, id, lastCheckAt)
	return args.Error(0)
}

func (m *MockMonitorRepositoryForAuthz) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMonitorRepositoryForAuthz) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockMonitorRepositoryForAuthz) ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	args := m.Called(ctx, userID, name)
	return args.Bool(0), args.Error(1)
}

// Test CanCreateMonitor

func TestAuthorizationService_CanCreateMonitor_FreeTier_LimitReached(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Мокаем CountByUserID - пользователь уже имеет 5 мониторов (лимит Free)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(5, nil)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Free")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor limit reached")
	mockRepo.AssertExpectations(t)
}

func TestAuthorizationService_CanCreateMonitor_FreeTier_UnderLimit(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Мокаем CountByUserID - пользователь имеет 3 монитора (лимит Free = 5)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(3, nil)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Free")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthorizationService_CanCreateMonitor_BasicTier_LimitReached(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Мокаем CountByUserID - пользователь имеет 25 мониторов (лимит Basic)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(25, nil)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Basic")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor limit reached")
	assert.Contains(t, err.Error(), "Basic")
	mockRepo.AssertExpectations(t)
}

func TestAuthorizationService_CanCreateMonitor_ProTier_UnderLimit(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Мокаем CountByUserID - пользователь имеет 50 мониторов (лимит Pro = 100)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(50, nil)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Pro")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthorizationService_CanCreateMonitor_EnterpriseTier_Unlimited(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Enterprise не вызывает CountByUserID - unlimited
	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Enterprise")

	// Assert
	assert.NoError(t, err)
}

func TestAuthorizationService_CanCreateMonitor_UnknownTier(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "UnknownTier")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrUnknownTier)
}

func TestAuthorizationService_CanCreateMonitor_CountError(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Мокаем ошибку при CountByUserID
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(0, errors.New("database error"))

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Free")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count monitors")
	mockRepo.AssertExpectations(t)
}

// Test CanAccessMonitor

func TestAuthorizationService_CanAccessMonitor_Owner(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Создаём монитор, принадлежащий пользователю
	monitor := mustNewMonitor(t, userID, "Test Monitor", "https://example.com", 60)
	// Override ID to match our test monitorID
	monitor.ID = monitorID

	mockRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

	// Act
	err := authz.CanAccessMonitor(ctx, userID, monitorID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthorizationService_CanAccessMonitor_NotOwner(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	anotherUserID := uuid.New()
	monitorID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Создаём монитор, принадлежащий другому пользователю
	monitor := mustNewMonitor(t, anotherUserID, "Test Monitor", "https://example.com", 60)
	monitor.ID = monitorID

	mockRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

	// Act
	err := authz.CanAccessMonitor(ctx, userID, monitorID)

	// Assert
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "permission denied")
	}
	mockRepo.AssertExpectations(t)
}

func TestAuthorizationService_CanAccessMonitor_NotFound(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	mockRepo.On("GetByID", mock.Anything, monitorID).Return(nil, interfaces.ErrMonitorNotFound)

	// Act
	err := authz.CanAccessMonitor(ctx, userID, monitorID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor not found")
	mockRepo.AssertExpectations(t)
}

func TestAuthorizationService_CanAccessMonitor_RepositoryError(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	mockRepo.On("GetByID", mock.Anything, monitorID).Return(nil, errors.New("database error"))

	// Act
	err := authz.CanAccessMonitor(ctx, userID, monitorID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get monitor")
	mockRepo.AssertExpectations(t)
}

// TestAuthorizationService_CanCreateMonitor_AllowFree тестирует разрешение для Free tier.
func TestAuthorizationService_CanCreateMonitor_AllowFree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Mock: пользователь имеет 3 монитора из 5 возможных
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(3, nil)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Free")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestAuthorizationService_CanCreateMonitor_DenyFree тестирует превышение лимита Free tier.
func TestAuthorizationService_CanCreateMonitor_DenyFree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Mock: пользователь имеет 5 мониторов из 5 возможных
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(5, nil)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Free")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor limit reached")
	assert.Contains(t, err.Error(), "Free")
	mockRepo.AssertExpectations(t)
}

// TestAuthorizationService_CanCreateMonitor_AllowBasic тестирует разрешение для Basic tier.
func TestAuthorizationService_CanCreateMonitor_AllowBasic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Mock: пользователь имеет 20 мониторов из 25 возможных
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(20, nil)

	// Act
	err := authz.CanCreateMonitor(ctx, userID, "Basic")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestAuthorizationService_CanAccessMonitor_DBError тестирует ошибку БД при проверке доступа.
func TestAuthorizationService_CanAccessMonitor_DBError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	mockRepo := new(MockMonitorRepositoryForAuthz)
	authz := NewAuthorizationService(mockRepo)

	// Mock
	mockRepo.On("GetByID", mock.Anything, monitorID).Return(nil, errors.New("database error"))

	// Act
	err := authz.CanAccessMonitor(ctx, userID, monitorID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get monitor")
	mockRepo.AssertExpectations(t)
}
