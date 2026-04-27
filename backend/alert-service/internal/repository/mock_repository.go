package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// MockAlertRepository implements AlertRepository interface for testing
type MockAlertRepository struct {
	alerts      []model.Alert
	alertsMutex sync.RWMutex
}

// NewMockAlertRepository creates a new mock alert repository
func NewMockAlertRepository() *MockAlertRepository {
	return &MockAlertRepository{
		alerts:      make([]model.Alert, 0),
		alertsMutex: sync.RWMutex{},
	}
}

// Create creates a new alert
func (m *MockAlertRepository) Create(ctx context.Context, alert *model.Alert) error {
	m.alertsMutex.Lock()
	defer m.alertsMutex.Unlock()

	m.alerts = append(m.alerts, *alert)
	return nil
}

// CreateWithDeliveryAttempt creates a new alert and delivery attempt
func (m *MockAlertRepository) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	m.alertsMutex.Lock()
	defer m.alertsMutex.Unlock()

	m.alerts = append(m.alerts, *alert)
	return nil
}

// GetByID retrieves an alert by ID
func (m *MockAlertRepository) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	m.alertsMutex.RLock()
	defer m.alertsMutex.RUnlock()

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, model.ErrAlertNotFound
	}

	for i, alert := range m.alerts {
		if alert.ID == parsedID {
			return &m.alerts[i], nil
		}
	}
	return nil, model.ErrAlertNotFound
}

// GetLastAlertTimeAnyStatus retrieves the last alert for a monitor
func (m *MockAlertRepository) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	m.alertsMutex.RLock()
	defer m.alertsMutex.RUnlock()

	// Find the last alert for this monitor (iterating backwards for most recent)
	for i := len(m.alerts) - 1; i >= 0; i-- {
		if m.alerts[i].MonitorID.String() == monitorID {
			return &m.alerts[i], nil
		}
	}

	return nil, model.ErrAlertNotFound
}

// ListActiveByMonitorID retrieves active alerts for a monitor
func (m *MockAlertRepository) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	m.alertsMutex.RLock()
	defer m.alertsMutex.RUnlock()

	var results []*model.Alert
	for i := range m.alerts {
		if m.alerts[i].MonitorID.String() == monitorID && m.alerts[i].Status == model.AlertStatusTriggered && m.alerts[i].Enabled {
			results = append(results, &m.alerts[i])
		}
	}

	if len(results) == 0 {
		return nil, model.ErrAlertNotFound
	}
	return results, nil
}

// List retrieves alerts with filtering and pagination
func (m *MockAlertRepository) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	m.alertsMutex.RLock()
	defer m.alertsMutex.RUnlock()

	var results []*model.Alert
	total := 0

	for i := range m.alerts {
		if m.alerts[i].UserID.String() == userID {
			results = append(results, &m.alerts[i])
			total++
		}
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	limit := filter.PageSize
	switch {
	case offset > len(results):
		results = []*model.Alert{}
	case offset+limit > len(results):
		results = results[offset:]
	default:
		results = results[offset : offset+limit]
	}

	return results, total, nil
}

// Update updates an existing alert
func (m *MockAlertRepository) Update(ctx context.Context, alert *model.Alert) error {
	m.alertsMutex.Lock()
	defer m.alertsMutex.Unlock()

	for i, a := range m.alerts {
		if a.ID == alert.ID {
			m.alerts[i] = *alert
			return nil
		}
	}
	return model.ErrAlertNotFound
}

// Delete removes an alert
func (m *MockAlertRepository) Delete(ctx context.Context, id string) error {
	m.alertsMutex.Lock()
	defer m.alertsMutex.Unlock()

	for i, a := range m.alerts {
		if a.ID.String() == id {
			m.alerts = append(m.alerts[:i], m.alerts[i+1:]...)
			return nil
		}
	}
	return model.ErrAlertNotFound
}

// Clear clears all alerts for testing
func (m *MockAlertRepository) Clear() {
	m.alertsMutex.Lock()
	defer m.alertsMutex.Unlock()
	m.alerts = make([]model.Alert, 0)
}

// AddAlert adds a predefined alert for testing
func (m *MockAlertRepository) AddAlert(alert model.Alert) {
	m.alertsMutex.Lock()
	defer m.alertsMutex.Unlock()
	m.alerts = append(m.alerts, alert)
}

// DeleteResolvedOlderThan deletes resolved alerts older than the specified duration
func (m *MockAlertRepository) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	return nil
}

// GetLastAlertByMonitorIDAndStatus retrieves the last alert for a monitor with the specified status
func (m *MockAlertRepository) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	return nil, model.ErrAlertNotFound
}

func (m *MockAlertRepository) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	return 0, nil
}

func (m *MockAlertRepository) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	return nil, nil
}

func (m *MockAlertRepository) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	for i, a := range m.alerts {
		if a.ID.String() == alertID {
			m.alerts[i].Status = model.AlertStatusAcknowledged
			return nil
		}
	}
	return model.ErrAlertNotAcknowledgeable
}
