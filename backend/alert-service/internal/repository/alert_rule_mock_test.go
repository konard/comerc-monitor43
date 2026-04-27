package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// MockAlertRuleRepository implements AlertRuleRepository interface for testing
type MockAlertRuleRepository struct {
	rules      []model.AlertRule
	rulesMutex sync.RWMutex
}

// NewMockAlertRuleRepository creates a new mock alert rule repository
func NewMockAlertRuleRepository() *MockAlertRuleRepository {
	return &MockAlertRuleRepository{
		rules:      make([]model.AlertRule, 0),
		rulesMutex: sync.RWMutex{},
	}
}

// Create creates a new alert rule
func (m *MockAlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	m.rulesMutex.Lock()
	defer m.rulesMutex.Unlock()

	m.rules = append(m.rules, *rule)
	return nil
}

// GetByID retrieves an alert rule by ID
func (m *MockAlertRuleRepository) GetByID(ctx context.Context, id string) (*model.AlertRule, error) {
	m.rulesMutex.RLock()
	defer m.rulesMutex.RUnlock()

	for i, rule := range m.rules {
		if rule.ID.String() == id {
			return &m.rules[i], nil
		}
	}
	return nil, model.ErrAlertRuleNotFound
}

// GetByUserIDAndMonitorID retrieves an alert rule by user and monitor
func (m *MockAlertRuleRepository) GetByUserIDAndMonitorID(ctx context.Context, userID string, monitorID string) (*model.AlertRule, error) {
	m.rulesMutex.RLock()
	defer m.rulesMutex.RUnlock()

	for i, rule := range m.rules {
		if rule.UserID.String() == userID && rule.MonitorID.String() == monitorID {
			return &m.rules[i], nil
		}
	}
	return nil, model.ErrAlertRuleNotFound
}

// List retrieves all alert rules for a user
func (m *MockAlertRuleRepository) List(ctx context.Context, userID string) ([]*model.AlertRule, error) {
	m.rulesMutex.RLock()
	defer m.rulesMutex.RUnlock()

	var results []*model.AlertRule
	for i := range m.rules {
		if m.rules[i].UserID.String() == userID {
			results = append(results, &m.rules[i])
		}
	}

	if len(results) == 0 {
		return nil, model.ErrAlertRuleNotFound
	}
	return results, nil
}

// Update updates an existing alert rule
func (m *MockAlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	m.rulesMutex.Lock()
	defer m.rulesMutex.Unlock()

	for i, r := range m.rules {
		if r.ID == rule.ID {
			m.rules[i] = *rule
			return nil
		}
	}
	return model.ErrAlertRuleNotFound
}

// Delete removes an alert rule
func (m *MockAlertRuleRepository) Delete(ctx context.Context, id string) error {
	m.rulesMutex.Lock()
	defer m.rulesMutex.Unlock()

	for i, r := range m.rules {
		if r.ID.String() == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}
	return model.ErrAlertRuleNotFound
}

// Clear clears all rules for testing
func (m *MockAlertRuleRepository) Clear() {
	m.rulesMutex.Lock()
	defer m.rulesMutex.Unlock()
	m.rules = make([]model.AlertRule, 0)
}

// AddRule adds a predefined rule for testing
func (m *MockAlertRuleRepository) AddRule(rule model.AlertRule) {
	m.rulesMutex.Lock()
	defer m.rulesMutex.Unlock()
	m.rules = append(m.rules, rule)
}

// TestMockAlertRuleRepository_Create_Success проверяет успешное создание правила
func TestMockAlertRuleRepository_Create_Success(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		MonitorID:           uuid.New(),
		Enabled:             true,
		ConsecutiveFailures: 0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	err := repo.Create(ctx, rule)
	assert.NoError(t, err)
}

// TestMockAlertRuleRepository_GetByID_Success проверяет успешное получение правила
func TestMockAlertRuleRepository_GetByID_Success(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	rule := &model.AlertRule{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add rule to repository
	repo.AddRule(*rule)

	result, err := repo.GetByID(ctx, rule.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, rule.ID, result.ID)
}

// TestMockAlertRuleRepository_GetByID_NotFound проверяет случай когда правило не найдено
func TestMockAlertRuleRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	result, err := repo.GetByID(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, model.ErrAlertRuleNotFound))
}

// TestMockAlertRuleRepository_GetByUserIDAndMonitorID_Success проверяет успешное получение правила
func TestMockAlertRuleRepository_GetByUserIDAndMonitorID_Success(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	rule := &model.AlertRule{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add rule to repository
	repo.AddRule(*rule)

	result, err := repo.GetByUserIDAndMonitorID(ctx, rule.UserID.String(), rule.MonitorID.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, rule.ID, result.ID)
}

// TestMockAlertRuleRepository_GetByUserIDAndMonitorID_NotFound проверяет отсутствие правила
func TestMockAlertRuleRepository_GetByUserIDAndMonitorID_NotFound(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	result, err := repo.GetByUserIDAndMonitorID(ctx, "non-existent-user-id", "non-existent-monitor-id")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, model.ErrAlertRuleNotFound))
}

// TestMockAlertRuleRepository_List_Success проверяет успешное получение списка правил
func TestMockAlertRuleRepository_List_Success(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	userID := uuid.New()

	// Add multiple rules for the user
	for i := 0; i < 3; i++ {
		rule := &model.AlertRule{
			ID:        uuid.New(),
			UserID:    userID,
			MonitorID: uuid.New(),
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.AddRule(*rule)
	}

	rules, err := repo.List(ctx, userID.String())
	assert.NoError(t, err)
	assert.Len(t, rules, 3)
}

// TestMockAlertRuleRepository_List_NotFound проверяет отсутствие правил для пользователя
func TestMockAlertRuleRepository_List_NotFound(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	rules, err := repo.List(ctx, "non-existent-user-id")
	assert.Error(t, err)
	assert.Nil(t, rules)
	assert.True(t, errors.Is(err, model.ErrAlertRuleNotFound))
}

// TestMockAlertRuleRepository_Update_Success проверяет успешное обновление правила
func TestMockAlertRuleRepository_Update_Success(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	originalRule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		MonitorID:           uuid.New(),
		Enabled:             true,
		ConsecutiveFailures: 0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Add rule to repository
	repo.AddRule(*originalRule)

	// Update rule
	updatedRule := &model.AlertRule{
		ID:                  originalRule.ID,
		UserID:              originalRule.UserID,
		MonitorID:           originalRule.MonitorID,
		Enabled:             false,
		ConsecutiveFailures: 3,
		CreatedAt:           originalRule.CreatedAt,
		UpdatedAt:           time.Now(),
	}

	err := repo.Update(ctx, updatedRule)
	assert.NoError(t, err)

	// Verify update
	result, err := repo.GetByID(ctx, originalRule.ID.String())
	assert.NoError(t, err)
	assert.False(t, result.Enabled)
	assert.Equal(t, 3, result.ConsecutiveFailures)
}

// TestMockAlertRuleRepository_Delete_Success проверяет успешное удаление правила
func TestMockAlertRuleRepository_Delete_Success(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	rule := &model.AlertRule{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add rule to repository
	repo.AddRule(*rule)

	// Delete rule
	err := repo.Delete(ctx, rule.ID.String())
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, rule.ID.String())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertRuleNotFound))
}

// TestMockAlertRuleRepository_Delete_NotFound проверяет удаление несуществующего правила
func TestMockAlertRuleRepository_Delete_NotFound(t *testing.T) {
	repo := NewMockAlertRuleRepository()

	ctx := context.Background()
	err := repo.Delete(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertRuleNotFound))
}
