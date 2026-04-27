package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestNewIncidentDetector тестирует создание IncidentDetector.
func TestNewIncidentDetector(t *testing.T) {
	t.Parallel()
	mockMonitorRepo := new(MockMonitorRepository)
	mockIncidentRepo := new(MockIncidentRepository)

	detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

	assert.NotNil(t, detector)
	assert.Equal(t, mockMonitorRepo, detector.monitorRepo)
	assert.Equal(t, mockIncidentRepo, detector.incidentRepo)
	assert.Equal(t, 3, detector.resolveThreshold)
}

// TestIncidentDetector_HandleStatusChange тестирует обработку изменений статуса.
func TestIncidentDetector_HandleStatusChange(t *testing.T) {
	t.Parallel()
	t.Run("creates incident on DOWN status", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusUp
		newStatus := domain.StatusDown

		// Mock: нет активных инцидентов
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{}, nil)

		// Mock: создаем инцидент
		mockIncidentRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.NoError(t, err)
		mockIncidentRepo.AssertCalled(t, "GetActiveByMonitorID", mock.Anything, monitorID)
		mockIncidentRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("does not create duplicate incident when one already exists", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusUp
		newStatus := domain.StatusDown

		// Mock: есть активный инцидент
		existingIncident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{existingIncident}, nil)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.NoError(t, err)
		mockIncidentRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("resolves incidents when status changes from DOWN to UP", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusDown
		newStatus := domain.StatusUp

		// Mock: есть активный инцидент
		incident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{incident}, nil)

		// Mock: закрываем инцидент
		mockIncidentRepo.On("Resolve", mock.Anything, incident.ID, mock.Anything).Return(nil)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.NoError(t, err)
		mockIncidentRepo.AssertCalled(t, "GetActiveByMonitorID", mock.Anything, monitorID)
		mockIncidentRepo.AssertCalled(t, "Resolve", mock.Anything, incident.ID, mock.Anything)
	})

	t.Run("resolves incidents when status changes from DOWN to DEGRADED", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusDown
		newStatus := domain.StatusDegraded

		// Mock: есть активный инцидент
		incident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{incident}, nil)

		// Mock: закрываем инцидент
		mockIncidentRepo.On("Resolve", mock.Anything, incident.ID, mock.Anything).Return(nil)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.NoError(t, err)
		mockIncidentRepo.AssertCalled(t, "Resolve", mock.Anything, incident.ID, mock.Anything)
	})

	t.Run("resolves multiple active incidents", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusDown
		newStatus := domain.StatusUp

		// Mock: несколько активных инцидентов
		incident1 := domain.NewIncident(monitorID)
		incident2 := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{incident1, incident2}, nil)

		// Mock: закрываем оба инцидента
		mockIncidentRepo.On("Resolve", mock.Anything, incident1.ID, mock.Anything).Return(nil)
		mockIncidentRepo.On("Resolve", mock.Anything, incident2.ID, mock.Anything).Return(nil)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.NoError(t, err)
		mockIncidentRepo.AssertNumberOfCalls(t, "Resolve", 2)
	})

	t.Run("does nothing when status stays DOWN", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusDown
		newStatus := domain.StatusDown

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем - ничего не должно быть вызвано
		assert.NoError(t, err)
		mockIncidentRepo.AssertNotCalled(t, "GetActiveByMonitorID", mock.Anything, mock.Anything)
		mockIncidentRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		mockIncidentRepo.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("does nothing when status changes between non-DOWN statuses", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusUp
		newStatus := domain.StatusDegraded

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем - ничего не должно быть вызвано
		assert.NoError(t, err)
		mockIncidentRepo.AssertNotCalled(t, "GetActiveByMonitorID", mock.Anything, mock.Anything)
	})

	t.Run("returns error when failing to get active incidents", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusUp
		newStatus := domain.StatusDown

		// Mock: ошибка при получении активных инцидентов
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return(nil, assert.AnError)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get active incidents")
	})

	t.Run("returns error when failing to create incident", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusUp
		newStatus := domain.StatusDown

		// Mock: нет активных инцидентов
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{}, nil)

		// Mock: ошибка при создании инцидента
		mockIncidentRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create incident")
	})

	t.Run("returns error when failing to resolve incident", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		oldStatus := domain.StatusDown
		newStatus := domain.StatusUp

		// Mock: есть активный инцидент
		incident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{incident}, nil)

		// Mock: ошибка при закрытии инцидента
		mockIncidentRepo.On("Resolve", mock.Anything, incident.ID, mock.Anything).Return(assert.AnError)

		// Выполняем
		err := detector.HandleStatusChange(context.Background(), monitorID, oldStatus, newStatus)

		// Проверяем
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve incident")
	})
}

// TestIncidentDetector_ResolveIncidents_EmptyList тестирует закрытие инцидентов когда список пуст.
func TestIncidentDetector_ResolveIncidents_EmptyList(t *testing.T) {
	t.Parallel()
	mockMonitorRepo := new(MockMonitorRepository)
	mockIncidentRepo := new(MockIncidentRepository)

	detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

	monitorID := uuid.New()

	// Mock: нет активных инцидентов
	mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{}, nil)

	// Act
	err := detector.resolveIncidents(context.Background(), monitorID)

	// Assert
	assert.NoError(t, err)
	mockIncidentRepo.AssertExpectations(t)
	// Resolve не должен быть вызван
	mockIncidentRepo.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
}

// TestIncidentDetector_GetIncidents тестирует получение инцидентов.
func TestIncidentDetector_GetIncidents(t *testing.T) {
	t.Parallel()
	t.Run("returns incidents by period", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetIncidentsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
			Limit:     10,
			Offset:    0,
		}

		// Mock: получаем монитор
		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		// Mock: получаем инциденты за период
		incident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{incident}, nil)

		// Mock: общее количество
		mockIncidentRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(1), nil)

		// Выполняем
		resp, err := detector.GetIncidents(context.Background(), req)

		// Проверяем
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Incidents, 1)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, 10, resp.Limit)
		assert.Equal(t, 0, resp.Offset)

		// Проверяем инцидент
		incidentResp := resp.Incidents[0]
		assert.Equal(t, monitorID.String(), incidentResp.MonitorID)
		assert.NotEmpty(t, incidentResp.ID)
		assert.False(t, incidentResp.StartTime.IsZero())

		mockMonitorRepo.AssertExpectations(t)
		mockIncidentRepo.AssertExpectations(t)
	})

	t.Run("returns incidents without period filter", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		userID := uuid.New()
		monitorID := uuid.New()

		req := &dto.GetIncidentsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			Limit:     10,
			Offset:    0,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		incidents := make([]*domain.Incident, 3)
		for i := 0; i < 3; i++ {
			incidents[i] = domain.NewIncident(monitorID)
		}
		mockIncidentRepo.On("GetByMonitorID", mock.Anything, monitorID, 10, 0).Return(incidents, nil)
		mockIncidentRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(3), nil)

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Incidents, 3)
		assert.Equal(t, 3, resp.Total)
	})

	t.Run("returns error for invalid monitor_id", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		req := &dto.GetIncidentsRequest{
			MonitorID: "invalid-uuid",
			UserID:    uuid.New().String(),
		}

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("returns error for invalid user_id", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		req := &dto.GetIncidentsRequest{
			MonitorID: uuid.New().String(),
			UserID:    "invalid-uuid",
		}

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("returns error when monitor not found", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		userID := uuid.New()

		req := &dto.GetIncidentsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
		}

		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(nil, interfaces.ErrMonitorNotFound)

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to get monitor")
	})

	t.Run("returns error when user does not own monitor", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()
		ownerID := uuid.New()
		requesterID := uuid.New()

		req := &dto.GetIncidentsRequest{
			MonitorID: monitorID.String(),
			UserID:    requesterID.String(),
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: ownerID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "monitor not found")
	})

	t.Run("returns empty list when no incidents", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetIncidentsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{}, nil)
		mockIncidentRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(0), nil)

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Incidents)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("returns error when failing to get incidents", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetIncidentsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(nil, assert.AnError)
		mockIncidentRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(0), nil)

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to get incidents")
	})

	t.Run("returns error when failing to count incidents", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetIncidentsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		incident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{incident}, nil)
		mockIncidentRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(0), assert.AnError)

		resp, err := detector.GetIncidents(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to count incidents")
	})
}

// TestIncidentDetector_Integration тестирует интеграционные сценарии.
func TestIncidentDetector_Integration(t *testing.T) {
	t.Parallel()
	t.Run("full incident lifecycle: create -> monitor DOWN -> monitor UP -> resolve", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()

		// Step 1: Monitor goes UP -> DOWN
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{}, nil).Once()
		mockIncidentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

		err := detector.HandleStatusChange(context.Background(), monitorID, domain.StatusUp, domain.StatusDown)
		assert.NoError(t, err)

		// Step 2: Monitor goes DOWN -> UP
		// Нам нужно настроить mock так, чтобы он возвращал инцидент из шага 1
		incident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{incident}, nil).Once()
		mockIncidentRepo.On("Resolve", mock.Anything, incident.ID, mock.Anything).Return(nil).Once()

		err = detector.HandleStatusChange(context.Background(), monitorID, domain.StatusDown, domain.StatusUp)
		assert.NoError(t, err)

		// Проверяем, что инцидент был создан и затем закрыт
		mockIncidentRepo.AssertNumberOfCalls(t, "Create", 1)
		mockIncidentRepo.AssertNumberOfCalls(t, "Resolve", 1)
	})

	t.Run("status changes without DOWN do not trigger incidents", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()

		// UP -> DEGRADED: ничего не должно произойти
		err := detector.HandleStatusChange(context.Background(), monitorID, domain.StatusUp, domain.StatusDegraded)
		assert.NoError(t, err)
		mockIncidentRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

		// DEGRADED -> UP: ничего не должно произойти
		err = detector.HandleStatusChange(context.Background(), monitorID, domain.StatusDegraded, domain.StatusUp)
		assert.NoError(t, err)
		mockIncidentRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

		// UP -> PAUSED: ничего не должно произойти
		err = detector.HandleStatusChange(context.Background(), monitorID, domain.StatusUp, domain.StatusPaused)
		assert.NoError(t, err)
		mockIncidentRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("handles rapid status changes correctly", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		detector := NewIncidentDetector(mockMonitorRepo, mockIncidentRepo, 3)

		monitorID := uuid.New()

		// Scenario: UP -> DOWN -> UP -> DOWN -> UP (rapid flapping)

		// First DOWN: create incident
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{}, nil).Once()
		mockIncidentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

		err := detector.HandleStatusChange(context.Background(), monitorID, domain.StatusUp, domain.StatusDown)
		assert.NoError(t, err)

		// First UP: resolve incident
		incident := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{incident}, nil).Once()
		mockIncidentRepo.On("Resolve", mock.Anything, incident.ID, mock.Anything).Return(nil).Once()

		err = detector.HandleStatusChange(context.Background(), monitorID, domain.StatusDown, domain.StatusUp)
		assert.NoError(t, err)

		// Second DOWN: create new incident
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{}, nil).Once()
		mockIncidentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

		err = detector.HandleStatusChange(context.Background(), monitorID, domain.StatusUp, domain.StatusDown)
		assert.NoError(t, err)

		// Second UP: resolve second incident
		incident2 := domain.NewIncident(monitorID)
		mockIncidentRepo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return([]*domain.Incident{incident2}, nil).Once()
		mockIncidentRepo.On("Resolve", mock.Anything, incident2.ID, mock.Anything).Return(nil).Once()

		err = detector.HandleStatusChange(context.Background(), monitorID, domain.StatusDown, domain.StatusUp)
		assert.NoError(t, err)

		// Проверяем: 2 инцидента создано, 2 закрыто
		mockIncidentRepo.AssertNumberOfCalls(t, "Create", 2)
		mockIncidentRepo.AssertNumberOfCalls(t, "Resolve", 2)
	})
}
