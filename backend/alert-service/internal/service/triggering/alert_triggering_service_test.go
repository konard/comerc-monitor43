package triggering

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

// TestNewAlertTriggeringService проверяет создание сервиса
func TestNewAlertTriggeringService(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockAlertRepo, service.alertRepo)
	assert.Equal(t, mockRuleRepo, service.ruleRepo)
}

// TestAlertTriggeringService_ProcessMonitorCheck проверяет базовую логику
func TestAlertTriggeringService_ProcessMonitorCheck(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule with threshold 3
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3, // Threshold is 3 failures
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return nil (no previous alert)
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(nil, nil)

	// Mock Create to succeed
	mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// Call with consecutive failures = 5 (above threshold of 3)
	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, model.AlertStatusTriggered, alert.Status)
	assert.Equal(t, monitorID, alert.MonitorID)
	assert.Equal(t, userID, alert.UserID)

	// Verify the alert was created with the rule ID
	assert.NotNil(t, alert.AlertRuleID)
	assert.Equal(t, rule.ID, *alert.AlertRuleID)

	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_WithMetrics проверяет запись метрик
func TestAlertTriggeringService_ProcessMonitorCheck_WithMetrics(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	// Create a real metrics instance
	metrics := apptelemetry.NewMetrics("alert-service-test")

	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, metrics)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(nil, nil)
	mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "UP", 5)

	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, model.AlertStatusTriggered, alert.Status)

	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_DifferentStatuses проверяет различные статусы
func TestAlertTriggeringService_ProcessMonitorCheck_DifferentStatuses(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	statuses := []string{"DOWN", "UP", "UNKNOWN", "MAINTENANCE"}

	for _, status := range statuses {
		// Setup mocks for each iteration
		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           monitorID,
			Enabled:             true,
			ConsecutiveFailures: 3,
		}
		mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)
		mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(nil, nil)
		mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, status, 5)

		assert.NoError(t, err, "status %s should not error", status)
		assert.NotNil(t, alert, "status %s should return alert", status)
		assert.Equal(t, model.AlertStatusTriggered, alert.Status)

		mockAlertRepo.AssertExpectations(t)
		mockRuleRepo.AssertExpectations(t)

		// Reset mocks for next iteration
		mockAlertRepo = &MockAlertRepository{}
		mockRuleRepo = &MockAlertRuleRepository{}
		service = NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)
	}
}

// TestAlertTriggeringService_ProcessMonitorCheck_DifferentConsecutiveFailures проверяет различные значения consecutive_failures
func TestAlertTriggeringService_ProcessMonitorCheck_DifferentConsecutiveFailures(t *testing.T) {
	testCases := []struct {
		name              string
		failures          int
		threshold         int
		shouldCreateAlert bool
	}{
		{
			name:              "below threshold",
			failures:          1,
			threshold:         3,
			shouldCreateAlert: false,
		},
		{
			name:              "at threshold",
			failures:          3,
			threshold:         3,
			shouldCreateAlert: true,
		},
		{
			name:              "above threshold",
			failures:          5,
			threshold:         3,
			shouldCreateAlert: true,
		},
		{
			name:              "way above threshold",
			failures:          10,
			threshold:         3,
			shouldCreateAlert: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAlertRepo := &MockAlertRepository{}
			mockRuleRepo := &MockAlertRuleRepository{}
			tracer := otel.GetTracerProvider().Tracer("test")
			service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

			ctx := context.Background()
			userID := uuid.New()
			monitorID := uuid.New()

			// Setup mocks
			rule := &model.AlertRule{
				ID:                  uuid.New(),
				UserID:              userID,
				MonitorID:           monitorID,
				Enabled:             true,
				ConsecutiveFailures: tc.threshold,
			}
			mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

			if tc.shouldCreateAlert {
				// Only when threshold is met, the service calls GetLastAlertTimeAnyStatus and Create
				mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(nil, nil)
				mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
			}

			alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", tc.failures)

			assert.NoError(t, err)

			if tc.shouldCreateAlert {
				// Alert should be created when threshold is met
				assert.NotNil(t, alert, "alert should be created when threshold is met")
				assert.Equal(t, model.AlertStatusTriggered, alert.Status)
			} else {
				// No alert should be created when threshold is not met
				assert.Nil(t, alert, "alert should not be created when threshold is not met")
			}

			mockAlertRepo.AssertExpectations(t)
			mockRuleRepo.AssertExpectations(t)
		})
	}
}

// TestAlertTriggeringService_CheckAlertRules проверяет проверку правил
func TestAlertTriggeringService_CheckAlertRules(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Create test rules - some for this monitor, some for other monitors
	otherMonitorID := uuid.New()
	rules := []*model.AlertRule{
		{
			ID:        uuid.New(),
			UserID:    userID,
			MonitorID: monitorID, // Rule for this monitor
			Enabled:   true,
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			MonitorID: otherMonitorID, // Rule for other monitor
			Enabled:   true,
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			MonitorID: monitorID, // Another rule for this monitor
			Enabled:   true,
		},
	}

	// Mock List to return all rules
	mockRuleRepo.On("List", mock.Anything, mock.Anything).Return(rules, nil)

	// Check rules for the specific monitor
	result, err := service.CheckAlertRules(ctx, userID, monitorID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Should return only 2 rules (the ones for this monitor)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, monitorID, result[0].MonitorID)
	assert.Equal(t, monitorID, result[1].MonitorID)

	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_CreateAlert проверяет создание алерта
func TestAlertTriggeringService_CreateAlert(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    userID,
		MonitorID: monitorID,
		Status:    model.AlertStatusTriggered,
	}

	// Mock Create to succeed
	mockAlertRepo.On("Create", mock.Anything, alert).Return(nil)

	createdAlert, err := service.CreateAlert(ctx, alert)

	assert.NoError(t, err)
	assert.NotNil(t, createdAlert)
	assert.Equal(t, alert.ID, createdAlert.ID)
	assert.Equal(t, alert.Status, createdAlert.Status)
	mockAlertRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_CreateAlert_WithError проверяет случай ошибки при создании
func TestAlertTriggeringService_CreateAlert_WithError(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    userID,
		MonitorID: monitorID,
		Status:    model.AlertStatusTriggered,
	}

	// Mock Create to return error
	mockAlertRepo.On("Create", mock.Anything, alert).Return(assert.AnError)

	createdAlert, err := service.CreateAlert(ctx, alert)

	assert.Error(t, err)
	assert.Nil(t, createdAlert)
	mockAlertRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_DisabledRule проверяет случай отключенного правила
func TestAlertTriggeringService_ProcessMonitorCheck_DisabledRule(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return disabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             false, // Rule is disabled
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.Nil(t, alert) // No alert should be created for disabled rule
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_RecentAlert проверяет cooldown период для алертов
func TestAlertTriggeringService_ProcessMonitorCheck_RecentAlert(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return recent alert (within 30 min cooldown)
	recentAlert := &model.Alert{
		ID:        uuid.New(),
		CreatedAt: time.Now().Add(-10 * time.Minute), // 10 minutes ago
	}
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(recentAlert, nil)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.Nil(t, alert) // No alert should be created within cooldown period
	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_OldAlert проверяет случай с устаревшим алертом
func TestAlertTriggeringService_ProcessMonitorCheck_OldAlert(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return old alert (older than 30 min cooldown)
	oldAlert := &model.Alert{
		ID:        uuid.New(),
		CreatedAt: time.Now().Add(-45 * time.Minute), // 45 minutes ago (outside cooldown)
	}
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(oldAlert, nil)
	mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.NotNil(t, alert) // Alert should be created outside cooldown period
	assert.Equal(t, model.AlertStatusTriggered, alert.Status)
	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_CreateAlert_WithMetrics проверяет создание алерта с метриками
func TestAlertTriggeringService_CreateAlert_WithMetrics(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	// Create a real metrics instance
	metrics := apptelemetry.NewMetrics("alert-service-test")

	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, metrics)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    userID,
		MonitorID: monitorID,
		Status:    model.AlertStatusTriggered,
		Type:      model.AlertTypeStatusCode,
	}

	mockAlertRepo.On("Create", mock.Anything, alert).Return(nil)

	createdAlert, err := service.CreateAlert(ctx, alert)

	assert.NoError(t, err)
	assert.NotNil(t, createdAlert)
	assert.Equal(t, alert.ID, createdAlert.ID)
	assert.Equal(t, alert.Status, createdAlert.Status)
	mockAlertRepo.AssertExpectations(t)
}

// MockThrottleChecker - mock implementation for ThrottleChecker
type MockThrottleChecker struct {
	mock.Mock
}

func (m *MockThrottleChecker) CheckThrottle(ctx context.Context, userID, monitorID uuid.UUID, status string) (*ThrottleResult, error) {
	args := m.Called(ctx, userID, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ThrottleResult), args.Error(1)
}

// MockAlertRepository - mock implementation for testing
type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) Create(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return []*model.Alert{}, 0, args.Error(2)
	}
	return args.Get(0).([]*model.Alert), args.Int(1), args.Error(2)
}

func (m *MockAlertRepository) Update(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, alert, attempt)
	if args.Get(0) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockAlertRepository) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	args := m.Called(ctx, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	args := m.Called(ctx, since)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockAlertRepository) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	args := m.Called(ctx, alertID, userID)
	return args.Error(0)
}

// MockAlertRuleRepository - mock implementation for testing
type MockAlertRuleRepository struct {
	mock.Mock
}

func (m *MockAlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	args := m.Called(ctx, rule)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRuleRepository) GetByID(ctx context.Context, id string) (*model.AlertRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertRule), args.Error(1)
}

func (m *MockAlertRuleRepository) GetByUserIDAndMonitorID(ctx context.Context, userID, monitorID string) (*model.AlertRule, error) {
	args := m.Called(ctx, userID, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertRule), args.Error(1)
}

func (m *MockAlertRuleRepository) List(ctx context.Context, userID string) ([]*model.AlertRule, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return []*model.AlertRule{}, args.Error(1)
	}
	return args.Get(0).([]*model.AlertRule), args.Error(1)
}

func (m *MockAlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	args := m.Called(ctx, rule)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRuleRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

// TestAlertTriggeringService_ProcessMonitorCheck_CooldownBoundary tests the exact cooldown boundary
func TestAlertTriggeringService_ProcessMonitorCheck_CooldownBoundary(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return alert exactly at cooldown boundary (30 minutes ago)
	oldAlertTime := time.Now().Add(-30 * time.Minute)
	oldAlert := &model.Alert{
		ID:        uuid.New(),
		MonitorID: monitorID,
		CreatedAt: oldAlertTime,
	}
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(oldAlert, nil)

	// Mock Create to succeed
	mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// Call with consecutive failures above threshold
	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	// Should create alert (exactly at cooldown boundary)
	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, userID, alert.UserID)
	assert.Equal(t, monitorID, alert.MonitorID)

	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_JustBeforeCooldown tests alert just before cooldown period
func TestAlertTriggeringService_ProcessMonitorCheck_JustBeforeCooldown(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return alert just before cooldown (14 minutes)
	justBeforeCooldown := time.Now().Add(-14 * time.Minute)
	oldAlert := &model.Alert{
		ID:        uuid.New(),
		MonitorID: monitorID,
		CreatedAt: justBeforeCooldown,
	}
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(oldAlert, nil)

	// Don't mock Create since it shouldn't be called
	// If Create is called, the test will fail due to unmet expectation

	// Call with consecutive failures above threshold
	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	// Should NOT create alert (within cooldown period)
	assert.NoError(t, err)
	assert.Nil(t, alert)

	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_JustAfterCooldown tests alert just after cooldown period
func TestAlertTriggeringService_ProcessMonitorCheck_JustAfterCooldown(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return alert just after cooldown (31 minutes)
	justAfterCooldown := time.Now().Add(-31 * time.Minute)
	oldAlert := &model.Alert{
		ID:        uuid.New(),
		MonitorID: monitorID,
		CreatedAt: justAfterCooldown,
	}
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(oldAlert, nil)

	// Mock Create to succeed
	mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// Call with consecutive failures above threshold
	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	// Should create alert (outside cooldown period)
	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, userID, alert.UserID)
	assert.Equal(t, monitorID, alert.MonitorID)

	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_RuleNotFound проверяет случай когда правило не найдено
func TestAlertTriggeringService_ProcessMonitorCheck_RuleNotFound(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return rule not found error
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(nil, model.ErrAlertRuleNotFound)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.Nil(t, alert) // No alert should be created when rule not found
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_RuleRepoError проверяет ошибку при получении правила
func TestAlertTriggeringService_ProcessMonitorCheck_RuleRepoError(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return error (not "not found")
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.Error(t, err)
	assert.Nil(t, alert)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_LastAlertError проверяет ошибку при получении последнего алерта
func TestAlertTriggeringService_ProcessMonitorCheck_LastAlertError(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return non-not-found error
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.Error(t, err)
	assert.Nil(t, alert)
	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_CreateError проверяет ошибку при создании алерта
func TestAlertTriggeringService_ProcessMonitorCheck_CreateError(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	// Mock GetLastAlertTimeAnyStatus to return nil (no previous alert)
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, mock.Anything).Return(nil, nil)

	// Mock Create to return error
	mockAlertRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.Error(t, err)
	assert.Nil(t, alert)
	mockAlertRepo.AssertExpectations(t)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_CheckAlertRules_RepoError проверяет ошибку при получении списка правил
func TestAlertTriggeringService_CheckAlertRules_RepoError(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock List to return error
	mockRuleRepo.On("List", mock.Anything, mock.Anything).Return([]*model.AlertRule{}, assert.AnError)

	result, err := service.CheckAlertRules(ctx, userID, monitorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_ZeroConsecutiveFailures проверяет нулевое количество отказов
func TestAlertTriggeringService_ProcessMonitorCheck_ZeroConsecutiveFailures(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return enabled rule with threshold 1
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 1,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(rule, nil)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 0)

	assert.NoError(t, err)
	assert.Nil(t, alert) // No alert should be created with 0 failures (threshold is 1)
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_CheckAlertRules_EmptyRules проверяет случай когда нет правил для монитора
func TestAlertTriggeringService_CheckAlertRules_EmptyRules(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock List to return empty list
	mockRuleRepo.On("List", mock.Anything, mock.Anything).Return([]*model.AlertRule{}, nil)

	result, err := service.CheckAlertRules(ctx, userID, monitorID)

	assert.NoError(t, err)
	// Result can be nil or empty slice - both are valid for "no rules found"
	if result == nil {
		assert.Nil(t, result)
	} else {
		assert.Equal(t, 0, len(result))
	}
	mockRuleRepo.AssertExpectations(t)
}

// TestAlertTriggeringService_ProcessMonitorCheck_WithMetrics_Error проверяет что метрики не записываются при ошибке
func TestAlertTriggeringService_ProcessMonitorCheck_WithMetrics_Error(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	// Create a real metrics instance
	metrics := apptelemetry.NewMetrics("alert-service-test")

	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, metrics)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	// Mock GetByUserIDAndMonitorID to return rule not found error
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, mock.Anything, mock.Anything).Return(nil, model.ErrAlertRuleNotFound)

	alert, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.Nil(t, alert) // No alert created, no metrics should be recorded
	mockRuleRepo.AssertExpectations(t)
}

// TestWithThrottlingService проверяет настройку сервиса throttling
func TestWithThrottlingService(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	mockThrottle := &MockThrottleChecker{}

	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil, WithThrottlingService(mockThrottle))

	assert.NotNil(t, service)
	assert.Equal(t, mockThrottle, service.throttlingService)
}

// TestCleanupOldResolvedAlerts_Success проверяет очистку старых алертов
func TestCleanupOldResolvedAlerts_Success(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	mockAlertRepo.On("DeleteResolvedOlderThan", mock.Anything, mock.Anything).Return(nil)

	err := service.CleanupOldResolvedAlerts(ctx, 30)

	assert.NoError(t, err)
	mockAlertRepo.AssertExpectations(t)
}

// TestCleanupOldResolvedAlerts_Error проверяет обработку ошибки при очистке
func TestCleanupOldResolvedAlerts_Error(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	mockAlertRepo.On("DeleteResolvedOlderThan", mock.Anything, mock.Anything).Return(assert.AnError)

	err := service.CleanupOldResolvedAlerts(ctx, 30)

	assert.Error(t, err)
}

// TestProcessMonitorCheck_SuccessStatus_AutoResolves проверяет авто-разрешение при успешном статусе
func TestProcessMonitorCheck_SuccessStatus_AutoResolves(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	activeAlert := &model.Alert{
		ID:        uuid.New(),
		UserID:    userID,
		MonitorID: monitorID,
		Status:    model.AlertStatusTriggered,
	}
	mockAlertRepo.On("GetActiveAlertsForMonitor", mock.Anything, monitorID.String()).Return([]*model.Alert{activeAlert}, nil)
	mockAlertRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Alert) bool {
		return a.Status == model.AlertStatusResolved
	})).Return(nil)

	result, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "up", 0)

	assert.NoError(t, err)
	assert.Nil(t, result)
	mockAlertRepo.AssertExpectations(t)
}

// TestProcessMonitorCheck_SuccessStatus_GetActiveAlertsError
func TestProcessMonitorCheck_SuccessStatus_GetActiveAlertsError(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	mockAlertRepo.On("GetActiveAlertsForMonitor", mock.Anything, monitorID.String()).Return(nil, assert.AnError)

	_, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "up", 0)

	assert.Error(t, err)
}

// TestProcessMonitorCheck_RuleRepoError проверяет обработку общей ошибки репозитория
func TestProcessMonitorCheck_RuleRepoError(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(nil, assert.AnError)

	_, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "down", 5)

	assert.Error(t, err)
}

// TestProcessMonitorCheck_ThrottlingSuppressed проверяет подавление при throttling suppress
func TestProcessMonitorCheck_ThrottlingSuppressed(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	mockThrottle := &MockThrottleChecker{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil, WithThrottlingService(mockThrottle))

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(rule, nil)
	mockThrottle.On("CheckThrottle", mock.Anything, userID, monitorID, "DOWN").Return(&ThrottleResult{Action: "suppress", Reason: "cooldown"}, nil)

	result, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestProcessMonitorCheck_CooldownRecentAlert проверяет cooldown без throttling-сервиса
func TestProcessMonitorCheck_CooldownRecentAlert(t *testing.T) {
	mockAlertRepo := &MockAlertRepository{}
	mockRuleRepo := &MockAlertRuleRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAlertTriggeringService(mockAlertRepo, mockRuleRepo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
	}
	recentAlert := &model.Alert{
		ID:        uuid.New(),
		CreatedAt: time.Now().Add(-5 * time.Minute), // within 15-min cooldown
	}

	mockRuleRepo.On("GetByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(rule, nil)
	mockAlertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, monitorID.String()).Return(recentAlert, nil)

	result, err := service.ProcessMonitorCheck(ctx, userID, monitorID, "DOWN", 5)

	assert.NoError(t, err)
	assert.Nil(t, result)
}
