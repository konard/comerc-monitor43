package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// Mock repository implementations

type mockAlertRepository struct {
	alerts     []model.Alert
	createErr  error
	getByIDErr error
	updateErr  error
	listErr    error
}

type mockAlertRuleRepository struct {
	rules []model.AlertRule
}

type mockAlertChannelRepository struct {
	channels []model.AlertChannel
}

// NewMockAlertRepository creates a mock alert repository
func NewMockAlertRepository() *mockAlertRepository {
	return &mockAlertRepository{
		alerts: make([]model.Alert, 0),
	}
}

// NewMockAlertRuleRepository creates a mock alert rule repository
func NewMockAlertRuleRepository() *mockAlertRuleRepository {
	return &mockAlertRuleRepository{
		rules: make([]model.AlertRule, 0),
	}
}

// NewMockAlertChannelRepository creates a mock alert channel repository
func NewMockAlertChannelRepository() *mockAlertChannelRepository {
	return &mockAlertChannelRepository{
		channels: make([]model.AlertChannel, 0),
	}
}

// Implement repository.AlertRepository interface

func (m *mockAlertRepository) Create(ctx context.Context, alert *model.Alert) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.alerts = append(m.alerts, *alert)
	return nil
}

func (m *mockAlertRepository) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	m.alerts = append(m.alerts, *alert)
	return nil
}

func (m *mockAlertRepository) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	for _, alert := range m.alerts {
		if alert.ID.String() == id {
			return &alert, nil
		}
	}
	return nil, model.ErrAlertNotFound
}

func (m *mockAlertRepository) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	for _, alert := range m.alerts {
		if alert.MonitorID.String() == monitorID {
			return &alert, nil
		}
	}
	return &model.Alert{}, model.ErrAlertNotFound
}

func (m *mockAlertRepository) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	var results []*model.Alert
	for i := range m.alerts {
		if m.alerts[i].MonitorID.String() == monitorID && m.alerts[i].Status == model.AlertStatusTriggered {
			results = append(results, &m.alerts[i])
		}
	}
	if len(results) == 0 {
		return nil, model.ErrAlertNotFound
	}
	return results, nil
}

func (m *mockAlertRepository) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var results []*model.Alert
	for i := range m.alerts {
		if m.alerts[i].UserID.String() == userID {
			results = append(results, &m.alerts[i])
		}
	}
	return results, len(results), nil
}

func (m *mockAlertRepository) Update(ctx context.Context, alert *model.Alert) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, a := range m.alerts {
		if a.ID == alert.ID {
			m.alerts[i] = *alert
			return nil
		}
	}
	return model.ErrAlertNotFound
}

func (m *mockAlertRepository) Delete(ctx context.Context, id string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid id format")
	}
	for i, a := range m.alerts {
		if a.ID == idUUID {
			m.alerts = append(m.alerts[:i], m.alerts[i+1:]...)
			return nil
		}
	}
	return model.ErrAlertNotFound
}

func (m *mockAlertRepository) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	return nil
}

func (m *mockAlertRepository) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	return nil, model.ErrAlertNotFound
}

func (m *mockAlertRepository) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	return 0, nil
}

func (m *mockAlertRepository) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	return nil, nil
}

func (m *mockAlertRepository) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	for i, a := range m.alerts {
		if a.ID.String() == alertID {
			if a.Status != model.AlertStatusTriggered {
				return model.ErrAlertNotAcknowledgeable
			}
			m.alerts[i].Status = model.AlertStatusAcknowledged
			return nil
		}
	}
	return model.ErrAlertNotFound
}

// Implement repository.AlertRuleRepository interface

func (m *mockAlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	m.rules = append(m.rules, *rule)
	return nil
}

func (m *mockAlertRuleRepository) GetByID(ctx context.Context, id string) (*model.AlertRule, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid id format")
	}
	for _, rule := range m.rules {
		if rule.ID == idUUID {
			return &rule, nil
		}
	}
	return nil, model.ErrAlertRuleNotFound
}

func (m *mockAlertRuleRepository) GetByUserIDAndMonitorID(ctx context.Context, userID, monitorID string) (*model.AlertRule, error) {
	userIDUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id format")
	}
	monitorIDUUID, err := uuid.Parse(monitorID)
	if err != nil {
		return nil, errors.New("invalid monitor id format")
	}
	for _, rule := range m.rules {
		if rule.UserID == userIDUUID && rule.MonitorID == monitorIDUUID {
			return &rule, nil
		}
	}
	return &model.AlertRule{}, model.ErrAlertRuleNotFound
}

func (m *mockAlertRuleRepository) List(ctx context.Context, userID string) ([]*model.AlertRule, error) {
	userIDUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id format")
	}
	var results []*model.AlertRule
	for i := range m.rules {
		if m.rules[i].UserID == userIDUUID {
			results = append(results, &m.rules[i])
		}
	}
	if len(results) == 0 {
		return nil, model.ErrAlertRuleNotFound
	}
	return results, nil
}

func (m *mockAlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	for i, r := range m.rules {
		if r.ID == rule.ID {
			m.rules[i] = *rule
			return nil
		}
	}
	return model.ErrAlertRuleNotFound
}

func (m *mockAlertRuleRepository) Delete(ctx context.Context, id string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid id format")
	}
	for i, r := range m.rules {
		if r.ID == idUUID {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}
	return model.ErrAlertRuleNotFound
}

// Implement repository.AlertChannelRepository interface

func (m *mockAlertChannelRepository) Create(ctx context.Context, channel *model.AlertChannel) error {
	m.channels = append(m.channels, *channel)
	return nil
}

func (m *mockAlertChannelRepository) GetByID(ctx context.Context, id string) (*model.AlertChannel, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid id format")
	}
	for _, channel := range m.channels {
		if channel.ID == idUUID {
			return &channel, nil
		}
	}
	return nil, model.ErrAlertChannelNotFound
}

func (m *mockAlertChannelRepository) GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error) {
	userIDUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id format")
	}
	for _, channel := range m.channels {
		if channel.UserID == userIDUUID && channel.Type == channelType {
			return &channel, nil
		}
	}
	return &model.AlertChannel{}, model.ErrAlertChannelNotFound
}

func (m *mockAlertChannelRepository) ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error) {
	userIDUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id format")
	}
	var results []*model.AlertChannel
	for i := range m.channels {
		if m.channels[i].UserID == userIDUUID {
			results = append(results, &m.channels[i])
		}
	}
	if len(results) == 0 {
		return nil, model.ErrAlertChannelNotFound
	}
	return results, nil
}

func (m *mockAlertChannelRepository) Update(ctx context.Context, channel *model.AlertChannel) error {
	for i, c := range m.channels {
		if c.ID == channel.ID {
			m.channels[i] = *channel
			return nil
		}
	}
	return model.ErrAlertChannelNotFound
}

func (m *mockAlertChannelRepository) Delete(ctx context.Context, id string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid id format")
	}
	for i, c := range m.channels {
		if c.ID == idUUID {
			m.channels = append(m.channels[:i], m.channels[i+1:]...)
			return nil
		}
	}
	return model.ErrAlertChannelNotFound
}

func (m *mockAlertChannelRepository) ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error) {
	return false, nil
}

func (m *mockAlertChannelRepository) MarkAsFailed(ctx context.Context, id string, failureCount int) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid id format")
	}
	for i, c := range m.channels {
		if c.ID == idUUID {
			m.channels[i].FailureCount = failureCount
			return nil
		}
	}
	return model.ErrAlertChannelNotFound
}

func (m *mockAlertChannelRepository) ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error) {
	return nil, errors.New("use DeliveryAttemptRepository.ListPending instead")
}

func (m *mockAlertChannelRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (m *mockAlertChannelRepository) IncrementFailureCount(ctx context.Context, id string) (int, error) {
	return 0, nil
}

func (m *mockAlertChannelRepository) DisableChannel(ctx context.Context, id string, reason string) error {
	return nil
}

func (m *mockAlertChannelRepository) SetChannelPriorities(ctx context.Context, ruleID string, priorities []model.AlertChannelPriority) error {
	return nil
}

func (m *mockAlertChannelRepository) GetChannelPriorities(ctx context.Context, ruleID string) ([]*model.AlertChannelPriority, error) {
	return nil, nil
}

// TestAlertServiceAdapter_Create_Success проверяет успешное создание алерта
func TestAlertServiceAdapter_Create_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()

	alert := &model.Alert{
		ID:        alertID,
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := adapter.CreateAlert(ctx, alert)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, alertID, result.ID)
}

// TestAlertServiceAdapter_GetAlert_Success проверяет успешное получение алерта
func TestAlertServiceAdapter_GetAlert_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()

	// Create alert first
	alert := &model.Alert{
		ID:        alertID,
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := adapter.CreateAlert(ctx, alert)
	assert.NoError(t, err)

	// Get alert
	result, err := adapter.GetAlert(ctx, alertID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, alertID, result.ID)
}

// TestAlertServiceAdapter_GetAlert_NotFound проверяет случай, когда алерт не найден
func TestAlertServiceAdapter_GetAlert_NotFound(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()

	result, err := adapter.GetAlert(ctx, alertID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestAlertServiceAdapter_ListAlerts_Success проверяет успешное получение списка алертов
func TestAlertServiceAdapter_ListAlerts_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	// Create alerts
	alertID1 := uuid.New()
	alertID2 := uuid.New()
	_, err := adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID1,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)
	_, err = adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID2,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeResponseTime,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)

	// List alerts
	filter := model.AlertFilter{
		Page:     1,
		PageSize: 10,
	}

	alerts, total, err := adapter.ListAlerts(ctx, userID, filter)
	assert.NoError(t, err)
	assert.Len(t, alerts, 2)
	assert.Equal(t, 2, total)
}

// TestAlertServiceAdapter_UpdateAlert_Success проверяет успешное обновление алерта
func TestAlertServiceAdapter_UpdateAlert_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()
	userID := uuid.New()

	// Create alert
	_, err := adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)

	// Update alert
	updatedAlert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusResolved,
		Enabled:   false,
		UpdatedAt: time.Now(),
	}

	result, err := adapter.UpdateAlert(ctx, updatedAlert)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, model.AlertStatusResolved, result.Status)
	assert.False(t, result.Enabled)
}

// TestAlertServiceAdapter_DeleteAlert_Success проверяет успешное удаление алерта
func TestAlertServiceAdapter_DeleteAlert_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()
	var err error

	// Create alert
	_, err = adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID,
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)

	// Delete alert
	err = adapter.DeleteAlert(ctx, alertID)
	assert.NoError(t, err)

	// Verify alert is deleted
	_, err = adapter.GetAlert(ctx, alertID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestAlertServiceAdapter_DeleteAlert_NotFound проверяет удаление несуществующего алерта
func TestAlertServiceAdapter_DeleteAlert_NotFound(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()

	err := adapter.DeleteAlert(ctx, alertID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestAlertServiceAdapter_EnableAlert_Success проверяет успешное включение алерта
func TestAlertServiceAdapter_EnableAlert_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()
	var err error

	// Create disabled alert
	_, err = adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusMuted,
		Enabled:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)

	// Enable alert
	err = adapter.EnableAlert(ctx, alertID)
	assert.NoError(t, err)

	// Verify alert is enabled
	alert, err := adapter.GetAlert(ctx, alertID)
	assert.NoError(t, err)
	assert.True(t, alert.Enabled)
}

// TestAlertServiceAdapter_EnableAlert_NotFound проверяет включение несуществующего алерта
func TestAlertServiceAdapter_EnableAlert_NotFound(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()

	err := adapter.EnableAlert(ctx, alertID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestAlertServiceAdapter_EnableAlert_AlreadyEnabled проверяет попытку включить уже включенный алерт
func TestAlertServiceAdapter_EnableAlert_AlreadyEnabled(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()
	var err error

	// Create already enabled alert
	_, err = adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)

	// Try to enable again (should work as it's idempotent)
	err = adapter.EnableAlert(ctx, alertID)
	assert.NoError(t, err)

	// Verify alert is still enabled
	alert, err := adapter.GetAlert(ctx, alertID)
	assert.NoError(t, err)
	assert.True(t, alert.Enabled)
}

// TestAlertServiceAdapter_DisableAlert_Success проверяет успешное отключение алерта
func TestAlertServiceAdapter_DisableAlert_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()
	var err error

	// Create enabled alert
	_, err = adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)

	// Disable alert
	err = adapter.DisableAlert(ctx, alertID)
	assert.NoError(t, err)

	// Verify alert is disabled
	alert, err := adapter.GetAlert(ctx, alertID)
	assert.NoError(t, err)
	assert.False(t, alert.Enabled)
}

// TestAlertServiceAdapter_DisableAlert_NotFound проверяет отключение несуществующего алерта
func TestAlertServiceAdapter_DisableAlert_NotFound(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	alertID := uuid.New()

	err := adapter.DisableAlert(ctx, alertID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestAlertServiceAdapter_DisableAlert_AlreadyDisabled проверяет попытку отключить уже отключенный алерт
func TestAlertServiceAdapter_DisableAlert_AlreadyDisabled(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()
	var err error

	// Create already disabled alert
	_, err = adapter.CreateAlert(ctx, &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: uuid.New(),
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusMuted,
		Enabled:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	assert.NoError(t, err)

	// Try to disable again (should work as it's idempotent)
	err = adapter.DisableAlert(ctx, alertID)
	assert.NoError(t, err)

	// Verify alert is still disabled
	alert, err := adapter.GetAlert(ctx, alertID)
	assert.NoError(t, err)
	assert.False(t, alert.Enabled)
}

func TestAlertServiceAdapter_AcknowledgeAlert_Success(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()
	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()

	mockAlertRepo.alerts = append(mockAlertRepo.alerts, model.Alert{
		ID:     alertID,
		UserID: userID,
		Status: model.AlertStatusTriggered,
	})

	err := adapter.AcknowledgeAlert(ctx, alertID, userID)
	assert.NoError(t, err)
}

func TestAlertServiceAdapter_AcknowledgeAlert_Forbidden(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()
	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	alertID := uuid.New()

	mockAlertRepo.alerts = append(mockAlertRepo.alerts, model.Alert{
		ID:     alertID,
		UserID: ownerID,
		Status: model.AlertStatusTriggered,
	})

	err := adapter.AcknowledgeAlert(ctx, alertID, otherUserID)
	assert.ErrorIs(t, err, model.ErrForbidden)
}

func TestAlertServiceAdapter_AcknowledgeAlert_NotAcknowledgeable(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()
	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()

	mockAlertRepo.alerts = append(mockAlertRepo.alerts, model.Alert{
		ID:     alertID,
		UserID: userID,
		Status: model.AlertStatusResolved,
	})

	err := adapter.AcknowledgeAlert(ctx, alertID, userID)
	assert.ErrorIs(t, err, model.ErrAlertNotAcknowledgeable)
}

// TestAlertServiceAdapter_AcknowledgeAlert_AlreadyAcknowledged проверяет concurrent ACK (uc_02_02_33)
func TestAlertServiceAdapter_AcknowledgeAlert_AlreadyAcknowledged(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := NewMockAlertRuleRepository()
	mockChannelRepo := NewMockAlertChannelRepository()
	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()

	mockAlertRepo.alerts = append(mockAlertRepo.alerts, model.Alert{
		ID:     alertID,
		UserID: userID,
		Status: model.AlertStatusAcknowledged,
	})

	err := adapter.AcknowledgeAlert(ctx, alertID, userID)
	assert.ErrorIs(t, err, model.ErrAlertAlreadyAcknowledged)
}

func TestAlertServiceAdapter_CreateAlert_Error(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockAlertRepo.createErr = errors.New("db error")
	adapter := NewAlertServiceAdapter(mockAlertRepo, NewMockAlertRuleRepository(), NewMockAlertChannelRepository(), nil)

	ctx := context.Background()
	_, err := adapter.CreateAlert(ctx, &model.Alert{ID: uuid.New()})
	assert.Error(t, err)
}

func TestAlertServiceAdapter_ListAlerts_Error(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockAlertRepo.listErr = errors.New("db error")
	adapter := NewAlertServiceAdapter(mockAlertRepo, NewMockAlertRuleRepository(), NewMockAlertChannelRepository(), nil)

	ctx := context.Background()
	_, _, err := adapter.ListAlerts(ctx, uuid.New(), model.AlertFilter{})
	assert.Error(t, err)
}

func TestAlertServiceAdapter_UpdateAlert_GetError(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockAlertRepo.getByIDErr = errors.New("db error")
	adapter := NewAlertServiceAdapter(mockAlertRepo, NewMockAlertRuleRepository(), NewMockAlertChannelRepository(), nil)

	ctx := context.Background()
	_, err := adapter.UpdateAlert(ctx, &model.Alert{ID: uuid.New()})
	assert.Error(t, err)
}

func TestAlertServiceAdapter_UpdateAlert_UpdateError(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	alertID := uuid.New()
	mockAlertRepo.alerts = append(mockAlertRepo.alerts, model.Alert{ID: alertID})
	mockAlertRepo.updateErr = errors.New("db error")
	adapter := NewAlertServiceAdapter(mockAlertRepo, NewMockAlertRuleRepository(), NewMockAlertChannelRepository(), nil)

	ctx := context.Background()
	_, err := adapter.UpdateAlert(ctx, &model.Alert{ID: alertID})
	assert.Error(t, err)
}

func TestAlertServiceAdapter_EnableAlert_UpdateError(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	alertID := uuid.New()
	mockAlertRepo.alerts = append(mockAlertRepo.alerts, model.Alert{ID: alertID})
	mockAlertRepo.updateErr = errors.New("db error")
	adapter := NewAlertServiceAdapter(mockAlertRepo, NewMockAlertRuleRepository(), NewMockAlertChannelRepository(), nil)

	ctx := context.Background()
	err := adapter.EnableAlert(ctx, alertID)
	assert.Error(t, err)
}

func TestAlertServiceAdapter_DisableAlert_UpdateError(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	alertID := uuid.New()
	mockAlertRepo.alerts = append(mockAlertRepo.alerts, model.Alert{ID: alertID})
	mockAlertRepo.updateErr = errors.New("db error")
	adapter := NewAlertServiceAdapter(mockAlertRepo, NewMockAlertRuleRepository(), NewMockAlertChannelRepository(), nil)

	ctx := context.Background()
	err := adapter.DisableAlert(ctx, alertID)
	assert.Error(t, err)
}

func TestAlertServiceAdapter_AcknowledgeAlert_GetError(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockAlertRepo.getByIDErr = errors.New("db error")
	adapter := NewAlertServiceAdapter(mockAlertRepo, NewMockAlertRuleRepository(), NewMockAlertChannelRepository(), nil)

	ctx := context.Background()
	err := adapter.AcknowledgeAlert(ctx, uuid.New(), uuid.New())
	assert.Error(t, err)
}
