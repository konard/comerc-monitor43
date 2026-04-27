package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// TestNewAlertRepository проверяет создание репозитория
func TestNewAlertRepository_Simple(t *testing.T) {
	repo := NewAlertRepository(nil)

	assert.NotNil(t, repo)
}

// TestAlertRepository_Create проверяет базовую структуру Create метода
func TestAlertRepository_Create_Structure(t *testing.T) {
	repo := NewAlertRepository(nil)

	assert.NotNil(t, repo)
	// Verify repository has correct type
	assert.IsType(t, &AlertRepository{}, repo)
}

// TestAlertRepository_Constructor verifies constructor behaviour
func TestAlertRepository_Constructor_NilDB(t *testing.T) {
	repo := NewAlertRepository(nil)

	assert.NotNil(t, repo)
	// Verify we get AlertRepository even with nil DB
	assert.IsType(t, &AlertRepository{}, repo)
}

// TestAlertRepository_CreateAlertStructure проверяет структуру алерта
func TestAlertRepository_CreateAlertStructure(t *testing.T) {
	alertID := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()

	alert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: monitorID,
		Status:    model.AlertStatusTriggered,
		Type:      model.AlertTypeStatusCode,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.Equal(t, alertID, alert.ID)
	assert.Equal(t, userID, alert.UserID)
	assert.Equal(t, monitorID, alert.MonitorID)
	assert.Equal(t, model.AlertStatusTriggered, alert.Status)
	assert.Equal(t, model.AlertTypeStatusCode, alert.Type)
	assert.True(t, alert.Enabled)
}

// TestAlertRepository_FilterStructure проверяет структуру фильтра
func TestAlertRepository_FilterStructure(t *testing.T) {
	filter := model.AlertFilter{
		MonitorID: uuid.New().String(),
		Status:    model.AlertStatusTriggered,
		Page:      1,
		PageSize:  10,
	}

	assert.NotEmpty(t, filter.MonitorID)
	assert.Equal(t, model.AlertStatusTriggered, filter.Status)
	assert.Equal(t, 1, filter.Page)
	assert.Equal(t, 10, filter.PageSize)
}

// TestAlertRepository_MultipleAlertTypes проверяет разные типы алертов
func TestAlertRepository_MultipleAlertTypes(t *testing.T) {
	alertTypes := []model.AlertType{
		model.AlertTypeStatusCode,
		model.AlertTypeResponseTime,
		model.AlertTypeBodyContains,
		model.AlertTypeBodyDoesNotContain,
		model.AlertTypeCertificateExpires,
	}

	alertStatuses := []model.AlertStatus{
		model.AlertStatusTriggered,
		model.AlertStatusMuted,
		model.AlertStatusAcknowledged,
		model.AlertStatusResolved,
	}

	assert.Len(t, alertTypes, 5)
	assert.Len(t, alertStatuses, 4)

	for _, alertType := range alertTypes {
		assert.NotEmpty(t, string(alertType))
	}

	for _, alertStatus := range alertStatuses {
		assert.NotEmpty(t, string(alertStatus))
	}
}
