package dto

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// AlertEvent событие алерта от Alert Service.
type AlertEvent struct {
	UserID              uuid.UUID
	AlertID             uuid.UUID
	MonitorID           uuid.UUID
	MonitorName         string
	MonitorStatus       string
	Severity            string
	ConsecutiveFailures int
	IsFlapping          bool
	FlapCount           int
	TriggeredAt         time.Time
}

// ParseAlertEventFromJSON парсит событие алерта из JSON.
func ParseAlertEventFromJSON(data []byte) (*AlertEvent, error) {
	var event AlertEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	// Валидация обязательных полей
	if event.AlertID == uuid.Nil {
		return nil, errors.New("alert_id is required")
	}

	if event.MonitorID == uuid.Nil {
		return nil, errors.New("monitor_id is required")
	}

	if event.TriggeredAt.IsZero() {
		event.TriggeredAt = time.Now()
	}

	return &event, nil
}
