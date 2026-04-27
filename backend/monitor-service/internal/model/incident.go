package domain

import (
	"time"

	"github.com/google/uuid"
)

// IncidentStatus представляет статус инцидента.
type IncidentStatus string

const (
	// IncidentActive активный инцидент (ещё не закрыт)
	IncidentActive IncidentStatus = "ACTIVE"
	// IncidentResolved закрытый инцидент
	IncidentResolved IncidentStatus = "RESOLVED"
)

// Incident представляет период недоступности монитора (DOWN).
type Incident struct {
	// ID уникальный идентификатор инцидента
	ID uuid.UUID
	// MonitorID идентификатор монитора
	MonitorID uuid.UUID
	// StartTime время начала инцидента (первый DOWN)
	StartTime time.Time
	// EndTime время окончания инцидента (первый UP после DOWN, опционально)
	EndTime *time.Time
	// DurationSeconds длительность инцидента в секундах (вычисляется при закрытии)
	DurationSeconds *int
	// Status статус инцидента
	Status IncidentStatus
	// CreatedAt время записи инцидента
	CreatedAt time.Time
}

// NewIncident создаёт новый активный инцидент.
func NewIncident(monitorID uuid.UUID) *Incident {
	now := time.Now()
	return &Incident{
		ID:        uuid.New(),
		MonitorID: monitorID,
		StartTime: now,
		Status:    IncidentActive,
		CreatedAt: now,
	}
}

// Resolve закрывает инцидент и вычисляет длительность.
func (i *Incident) Resolve(endTime time.Time) {
	i.EndTime = &endTime
	duration := int(endTime.Sub(i.StartTime).Seconds())
	i.DurationSeconds = &duration
	i.Status = IncidentResolved
}

// IsActive возвращает true, если инцидент активен.
func (i *Incident) IsActive() bool {
	return i.Status == IncidentActive
}

// IsResolved возвращает true, если инцидент закрыт.
func (i *Incident) IsResolved() bool {
	return i.Status == IncidentResolved
}

// GetDuration возвращает длительность инцидента.
// Для активных инцидентов возвращает длительность от начала до сейчас.
func (i *Incident) GetDuration() time.Duration {
	if i.DurationSeconds != nil {
		return time.Duration(*i.DurationSeconds) * time.Second
	}
	if i.EndTime != nil {
		return i.EndTime.Sub(i.StartTime)
	}
	return time.Since(i.StartTime)
}

// IncidentSlice represents a slice of Incident for helper methods.
type IncidentSlice []Incident

// FilterActive возвращает только активные инциденты.
func (s IncidentSlice) FilterActive() IncidentSlice {
	filtered := make(IncidentSlice, 0)
	for _, i := range s {
		if i.IsActive() {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

// FilterByPeriod возвращает инциденты, пересекающиеся с периодом.
func (s IncidentSlice) FilterByPeriod(from, to time.Time) IncidentSlice {
	filtered := make(IncidentSlice, 0)
	for _, i := range s {
		// Инцидент пересекается с периодом, если:
		// - Начался до окончания периода И
		// - Закончился после начала периода (или ещё активен)
		if (i.StartTime.Before(to) || i.StartTime.Equal(to)) &&
			(i.EndTime == nil || i.EndTime.After(from) || i.EndTime.Equal(from)) {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

// CalculateTotalDowntime вычисляет общее время простоя в периоде.
func (s IncidentSlice) CalculateTotalDowntime(from, to time.Time) time.Duration {
	var total time.Duration
	for _, i := range s.FilterByPeriod(from, to) {
		// Вычисляем пересечение инцидента с периодом
		start := i.StartTime
		if start.Before(from) {
			start = from
		}

		end := to
		if i.EndTime != nil {
			end = *i.EndTime
			if end.After(to) {
				end = to
			}
		}

		total += end.Sub(start)
	}
	return total
}

// Count возвращает количество инцидентов.
func (s IncidentSlice) Count() int {
	return len(s)
}

// CountActive возвращает количество активных инцидентов.
func (s IncidentSlice) CountActive() int {
	count := 0
	for _, i := range s {
		if i.IsActive() {
			count++
		}
	}
	return count
}
