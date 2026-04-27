package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// TestMockAlertRepository_Create_Success проверяет успешное создание алерта
func TestMockAlertRepository_Create_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	alertID := mustParseUUID(t, "550e8400-e29b-41d4-a716-446655400000")
	monitorID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")

	alert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, alert)
	assert.NoError(t, err)
}

// TestMockAlertRepository_CreateWithDeliveryAttempt_Success проверяет создание алерта с попыткой доставки
func TestMockAlertRepository_CreateWithDeliveryAttempt_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	alertID := mustParseUUID(t, "550e8400-e29b-41d4-a716-446655400000")
	channelID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")

	alert := &model.Alert{
		ID:        alertID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	attempt := &model.DeliveryAttempt{
		AlertChannelID: channelID,
		Status:         model.DeliveryAttemptStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := repo.CreateWithDeliveryAttempt(ctx, alert, attempt)
	assert.NoError(t, err)
}

// TestMockAlertRepository_GetByID_Success проверяет успешное получение алерта
func TestMockAlertRepository_GetByID_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	alertID := mustParseUUID(t, "550e8400-e29b-41d4-a716-446655400000")
	monitorID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")

	alert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add alert to repository
	repo.AddAlert(*alert)

	result, err := repo.GetByID(ctx, alertID.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, alertID, result.ID)
}

// TestMockAlertRepository_GetByID_NotFound проверяет случай когда алерт не найден
func TestMockAlertRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	alertID := "non-existent-id"

	result, err := repo.GetByID(ctx, alertID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestMockAlertRepository_ListActiveByMonitorID_Success проверяет успешное получение активных алертов
func TestMockAlertRepository_ListActiveByMonitorID_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	monitorID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")

	// Add multiple alerts for different monitors
	for i := 0; i < 3; i++ {
		otherMonitorID := generateTestUUID(80 + i)
		alert := &model.Alert{
			ID:        generateTestUUID(100 + i),
			UserID:    userID,
			MonitorID: otherMonitorID,
			Type:      model.AlertTypeStatusCode,
			Status:    model.AlertStatusTriggered,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.AddAlert(*alert)
	}

	// Add one triggered alert for the target monitor
	targetAlert := &model.Alert{
		ID:        mustParseUUID(t, "550e8400-e29b-41d4-a716-446655440003"),
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.AddAlert(*targetAlert)

	// Add a resolved alert (should not be included)
	resolvedAlert := &model.Alert{
		ID:        mustParseUUID(t, "550e8400-e29b-41d4-a716-446655440004"),
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusResolved,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.AddAlert(*resolvedAlert)

	// Add a disabled alert (should not be included)
	disabledAlert := &model.Alert{
		ID:        mustParseUUID(t, "550e8400-e29b-41d4-a716-446655440005"),
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.AddAlert(*disabledAlert)

	alerts, err := repo.ListActiveByMonitorID(ctx, monitorID.String())
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, targetAlert.ID, alerts[0].ID)
	assert.True(t, alerts[0].Enabled)
}

// TestMockAlertRepository_ListActiveByMonitorID_NotFound проверяет отсутствие активных алертов
func TestMockAlertRepository_ListActiveByMonitorID_NotFound(t *testing.T) {
	repo := NewMockAlertRepository()
	ctx := context.Background()

	monitorID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")

	// Add a resolved alert for this monitor
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")
	resolvedAlert := &model.Alert{
		ID:        mustParseUUID(t, "550e8400-e29b-41d4-a716-446655440006"),
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusResolved,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.AddAlert(*resolvedAlert)

	alerts, err := repo.ListActiveByMonitorID(ctx, monitorID.String())
	assert.Error(t, err)
	assert.Nil(t, alerts)
}

// TestMockAlertRepository_List_Success проверяет успешное получение списка с фильтрацией
func TestMockAlertRepository_List_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")

	// Add multiple alerts for the user
	for i := 0; i < 10; i++ {
		alert := &model.Alert{
			ID:        mustParseUUID(t, fmt.Sprintf("088e8400-e29b-41d4-a716-446655400%03d", i)),
			UserID:    userID,
			Type:      model.AlertTypeStatusCode,
			Status:    model.AlertStatusTriggered,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.AddAlert(*alert)
	}

	filter := model.AlertFilter{
		Page:     1,
		PageSize: 5,
	}

	alerts, total, err := repo.List(ctx, userID.String(), filter)
	assert.NoError(t, err)
	assert.Len(t, alerts, 5)
	assert.Equal(t, 10, total)
}

// TestMockAlertRepository_List_EmptyResults проверяет пустые результаты
func TestMockAlertRepository_List_EmptyResults(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")

	// No alerts for this user
	filter := model.AlertFilter{
		Page:     1,
		PageSize: 10,
	}

	alerts, total, err := repo.List(ctx, userID.String(), filter)
	assert.NoError(t, err)
	assert.Len(t, alerts, 0)
	assert.Equal(t, 0, total)
}

// TestMockAlertRepository_List_Pagination проверяет пагинацию
func TestMockAlertRepository_List_Pagination(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")

	// Add 15 alerts for the user
	for i := 0; i < 15; i++ {
		alert := &model.Alert{
			ID:        generateTestUUID(200 + i),
			UserID:    userID,
			Type:      model.AlertTypeStatusCode,
			Status:    model.AlertStatusTriggered,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.AddAlert(*alert)
	}

	// Test first page with 5 items
	filter := model.AlertFilter{
		Page:     1,
		PageSize: 5,
	}

	alerts, total, err := repo.List(ctx, userID.String(), filter)
	assert.NoError(t, err)
	assert.Len(t, alerts, 5)
	assert.Equal(t, 15, total)

	// Test second page with 5 items
	filter.Page = 2
	alerts, _, err = repo.List(ctx, userID.String(), filter)
	assert.NoError(t, err)
	assert.Len(t, alerts, 5)

	// Test third page with 5 items
	filter.Page = 3
	alerts, _, err = repo.List(ctx, userID.String(), filter)
	assert.NoError(t, err)
	assert.Len(t, alerts, 5)
}

// TestMockAlertRepository_Update_Success проверяет успешное обновление алерта
func TestMockAlertRepository_Update_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	alertID := mustParseUUID(t, "550e8400-e29b-41d4-a716-446655400000")
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655400100")

	alert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add alert to repository
	repo.AddAlert(*alert)

	// Update alert status
	updatedAlert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusResolved,
		Enabled:   false,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(ctx, updatedAlert)
	assert.NoError(t, err)
}

// TestMockAlertRepository_Delete_Success проверяет успешное удаление алерта
func TestMockAlertRepository_Delete_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	alertID := mustParseUUID(t, "550e8400-e29b-41d4-a716-446655400000")
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655400100")

	alert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add alert to repository
	repo.AddAlert(*alert)

	// Delete alert
	err := repo.Delete(ctx, alertID.String())
	assert.NoError(t, err)

	// Verify alert is deleted
	_, err = repo.GetByID(ctx, alertID.String())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestMockAlertRepository_Delete_NotFound проверяет удаление несуществующего алерта
func TestMockAlertRepository_Delete_NotFound(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	alertID := "non-existent-id"

	err := repo.Delete(ctx, alertID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertNotFound))
}

// TestMockAlertRepository_GetLastAlertTimeAnyStatus_Success проверяет получение последнего алерта
func TestMockAlertRepository_GetLastAlertTimeAnyStatus_Success(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	monitorID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")

	// Add multiple alerts for the monitor with different timestamps
	baseTime := time.Now().Add(-24 * time.Hour)
	for i := 0; i < 3; i++ {
		alert := &model.Alert{
			ID:        mustParseUUID(t, fmt.Sprintf("099e8400-e29b-41d4-a716-446655440%03d", i)),
			UserID:    userID,
			MonitorID: monitorID,
			Type:      model.AlertTypeStatusCode,
			Status:    model.AlertStatusTriggered,
			Enabled:   true,
			CreatedAt: baseTime.Add(time.Duration(i) * time.Hour),
			UpdatedAt: baseTime.Add(time.Duration(i) * time.Hour),
		}
		repo.AddAlert(*alert)
	}

	// Get last alert
	alert, err := repo.GetLastAlertTimeAnyStatus(ctx, monitorID.String())
	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, mustParseUUID(t, "099e8400-e29b-41d4-a716-446655440002"), alert.ID)
}

// TestMockAlertRepository_GetLastAlertTimeAnyStatus_NotFound проверяет отсутствие алертов для монитора
func TestMockAlertRepository_GetLastAlertTimeAnyStatus_NotFound(t *testing.T) {
	repo := NewMockAlertRepository()
	ctx := context.Background()

	monitorID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")

	alert, err := repo.GetLastAlertTimeAnyStatus(ctx, monitorID.String())
	assert.Error(t, err)
	assert.Nil(t, alert)
}

// TestMockAlertRepository_Concurrency проверяет конкурентный доступ
func TestMockAlertRepository_Concurrency(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")
	monitorID := mustParseUUID(t, "660e8400-e29b-41d4-a716-446655440001")

	// Test concurrent creates
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			alertID := mustParseUUID(t, fmt.Sprintf("88%de8400-e29b-41d4-a716-446655440%03d", id, id))
			alert := &model.Alert{
				ID:        alertID,
				UserID:    userID,
				MonitorID: monitorID,
				Type:      model.AlertTypeStatusCode,
				Status:    model.AlertStatusTriggered,
				Enabled:   true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			require.NoError(t, repo.Create(ctx, alert))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all alerts were created (should have 10 alerts now)
	filter := model.AlertFilter{
		Page:     1,
		PageSize: 100,
	}

	_, total, err := repo.List(ctx, userID.String(), filter)
	assert.NoError(t, err)
	assert.Equal(t, 10, total)
}

// TestMockAlertRepository_EdgeCases проверяет граничные случаи
func TestMockAlertRepository_EdgeCases(t *testing.T) {
	repo := NewMockAlertRepository()

	ctx := context.Background()

	// Test 1: Empty repository
	emptyFilter := model.AlertFilter{
		Page:     1,
		PageSize: 10,
	}
	alerts, total, err := repo.List(ctx, "test-user-id", emptyFilter)
	assert.NoError(t, err)
	assert.Len(t, alerts, 0)
	assert.Equal(t, 0, total)

	// Test 2: Pagination beyond available items
	userID := mustParseUUID(t, "770e8400-e29b-41d4-a716-446655440002")
	for i := 0; i < 3; i++ {
		alert := &model.Alert{
			ID:        mustParseUUID(t, fmt.Sprintf("088e8400-e29b-41d4-a716-446655400%03d", i)),
			UserID:    userID,
			Type:      model.AlertTypeStatusCode,
			Status:    model.AlertStatusTriggered,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.AddAlert(*alert)
	}

	edgeCaseFilter := model.AlertFilter{
		Page:     2,
		PageSize: 5,
	}

	alerts, total, err = repo.List(ctx, userID.String(), edgeCaseFilter)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(alerts), 5) // Should return at most 5 items
	assert.Equal(t, 3, total)
}
