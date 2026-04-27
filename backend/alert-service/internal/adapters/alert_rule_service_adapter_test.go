package adapters

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// Mock implementations for testing

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

func (m *MockAlertRuleRepository) GetByUserIDAndMonitorID(ctx context.Context, userID string, monitorID string) (*model.AlertRule, error) {
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

// Test functions

func TestNewAlertRuleServiceAdapter(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	assert.NotNil(t, adapter)
	assert.Equal(t, mockRepo, adapter.ruleRepo)
}

func TestAlertRuleServiceAdapter_GetRuleByMonitorID_Success(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	expectedRule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
	}

	mockRepo.On("GetByUserIDAndMonitorID", ctx, userID.String(), monitorID.String()).Return(expectedRule, nil)

	rule, err := adapter.GetRuleByMonitorID(ctx, userID, monitorID)

	assert.NoError(t, err)
	assert.Equal(t, expectedRule.ID, rule.ID)
	assert.Equal(t, expectedRule.UserID, rule.UserID)
	assert.Equal(t, expectedRule.MonitorID, rule.MonitorID)
	mockRepo.AssertExpectations(t)
}

func TestAlertRuleServiceAdapter_GetRuleByMonitorID_NotFound(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	mockRepo.On("GetByUserIDAndMonitorID", ctx, userID.String(), monitorID.String()).Return(nil, sql.ErrNoRows)

	rule, err := adapter.GetRuleByMonitorID(ctx, userID, monitorID)

	assert.Error(t, err)
	assert.Nil(t, rule)
	mockRepo.AssertExpectations(t)
}

func TestAlertRuleServiceAdapter_GetRuleByMonitorID_Error(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	mockRepo.On("GetByUserIDAndMonitorID", ctx, userID.String(), monitorID.String()).Return(nil, assert.AnError)

	rule, err := adapter.GetRuleByMonitorID(ctx, userID, monitorID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alert rule by monitor")
	assert.Nil(t, rule)
	mockRepo.AssertExpectations(t)
}

func TestAlertRuleServiceAdapter_GetRuleByMonitorID_SQLNoRows(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	mockRepo.On("GetByUserIDAndMonitorID", ctx, userID.String(), monitorID.String()).Return(nil, sql.ErrNoRows)

	rule, err := adapter.GetRuleByMonitorID(ctx, userID, monitorID)

	assert.Error(t, err)
	assert.Equal(t, model.ErrAlertRuleNotFound, err)
	assert.Nil(t, rule)
	mockRepo.AssertExpectations(t)
}

func TestAlertRuleServiceAdapter_GetRuleByID_Success(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	id := uuid.New()
	expected := &model.AlertRule{ID: id}

	mockRepo.On("GetByID", mock.Anything, id.String()).Return(expected, nil)

	rule, err := adapter.GetRuleByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, expected, rule)
}

func TestAlertRuleServiceAdapter_GetRuleByID_NotFound(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	id := uuid.New()

	mockRepo.On("GetByID", mock.Anything, id.String()).Return(nil, sql.ErrNoRows)

	rule, err := adapter.GetRuleByID(ctx, id)

	assert.Error(t, err)
	assert.Equal(t, model.ErrAlertRuleNotFound, err)
	assert.Nil(t, rule)
}

func TestAlertRuleServiceAdapter_GetRuleByID_Error(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	id := uuid.New()

	mockRepo.On("GetByID", mock.Anything, id.String()).Return(nil, assert.AnError)

	_, err := adapter.GetRuleByID(ctx, id)

	assert.Error(t, err)
}

func TestAlertRuleServiceAdapter_CreateRule_Success(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	rule := &model.AlertRule{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
	}

	mockRepo.On("Create", mock.Anything, rule).Return(nil)

	result, err := adapter.CreateRule(ctx, rule)

	assert.NoError(t, err)
	assert.Equal(t, rule, result)
}

func TestAlertRuleServiceAdapter_CreateRule_Error(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	rule := &model.AlertRule{ID: uuid.New()}

	mockRepo.On("Create", mock.Anything, rule).Return(assert.AnError)

	_, err := adapter.CreateRule(ctx, rule)

	assert.Error(t, err)
}

func TestAlertRuleServiceAdapter_DeleteRule_Success(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	ruleID := uuid.New()
	rule := &model.AlertRule{ID: ruleID, UserID: userID, MonitorID: monitorID}

	mockRepo.On("GetByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(rule, nil)
	mockRepo.On("Delete", mock.Anything, ruleID.String()).Return(nil)

	err := adapter.DeleteRule(ctx, userID, monitorID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAlertRuleServiceAdapter_DeleteRule_NotFound(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	mockRepo.On("GetByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(nil, sql.ErrNoRows)

	err := adapter.DeleteRule(ctx, userID, monitorID)

	assert.ErrorIs(t, err, model.ErrAlertRuleNotFound)
	mockRepo.AssertExpectations(t)
}

func TestAlertRuleServiceAdapter_DeleteRule_GetError(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	mockRepo.On("GetByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(nil, assert.AnError)

	err := adapter.DeleteRule(ctx, userID, monitorID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alert rule for deletion")
	mockRepo.AssertExpectations(t)
}

func TestAlertRuleServiceAdapter_DeleteRule_DeleteError(t *testing.T) {
	mockRepo := &MockAlertRuleRepository{}
	adapter := NewAlertRuleServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	ruleID := uuid.New()
	rule := &model.AlertRule{ID: ruleID, UserID: userID, MonitorID: monitorID}

	mockRepo.On("GetByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(rule, nil)
	mockRepo.On("Delete", mock.Anything, ruleID.String()).Return(assert.AnError)

	err := adapter.DeleteRule(ctx, userID, monitorID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete alert rule")
	mockRepo.AssertExpectations(t)
}

func TestAlertServiceAdapter_WithAuditService(t *testing.T) {
	mockAlertRepo := NewMockAlertRepository()
	mockRuleRepo := &MockAlertRuleRepository{}
	mockChannelRepo := NewMockAlertChannelRepository()

	adapter := NewAlertServiceAdapter(mockAlertRepo, mockRuleRepo, mockChannelRepo, nil)
	result := adapter.WithAuditService(nil)

	assert.Equal(t, adapter, result)
}
