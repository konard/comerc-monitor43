package domain

import (
	"errors"
	"sort"
	"time"
)

// UptimeStats представляет статистику uptime за период.
type UptimeStats struct {
	Uptime              float64
	TotalChecks         int
	UpChecks            int
	DegradedChecks      int
	DownChecks          int
	PausedChecks        int
	TotalDowntime       time.Duration
	AverageResponseTime int
	Incidents           int
	Note                string
	P50                 int
	P95                 int
	P99                 int
}

// NewUptimeStats создаёт новую статистику uptime.
func NewUptimeStats() *UptimeStats {
	return &UptimeStats{
		Uptime: 100.0, // По умолчанию 100%
	}
}

// Calculate вычисляет процент uptime на основе результатов проверок.
//
// Формула: (UP + DEGRADED*0.5) / Total * 100
//
// PAUSED проверки исключаются из расчёта.
func (s *UptimeStats) Calculate(results []CheckResult) {
	if len(results) == 0 {
		s.Note = "No checks in period"
		s.TotalChecks = 0
		return
	}

	// Фильтруем активные проверки (исключаем PAUSED)
	active := make([]CheckResult, 0)
	for _, r := range results {
		if r.Status != StatusPaused {
			active = append(active, r)
		}
	}

	s.PausedChecks = len(results) - len(active)

	if len(active) == 0 {
		s.Note = "No active checks in period"
		s.TotalChecks = 0
		return
	}

	s.TotalChecks = len(active)

	up := 0
	degraded := 0
	down := 0
	totalResponseTime := 0
	responseTimeCount := 0

	for _, r := range active {
		switch r.Status {
		case StatusUp:
			up++
		case StatusDegraded:
			degraded++
		case StatusDown:
			down++
		}

		if r.ResponseTimeMs != nil {
			totalResponseTime += *r.ResponseTimeMs
			responseTimeCount++
		}
	}

	s.UpChecks = up
	s.DegradedChecks = degraded
	s.DownChecks = down

	// Формула: (UP + DEGRADED*0.5) / Total * 100
	uptime := (float64(up) + float64(degraded)*0.5) / float64(len(active)) * 100
	s.Uptime = uptime

	if responseTimeCount > 0 {
		s.AverageResponseTime = totalResponseTime / responseTimeCount
	}
}

// CalculatePercentiles вычисляет P50, P95, P99 для response time.
// Учитывает только результаты с 2xx статус кодами согласно feature uc_01_05_09.
func (s *UptimeStats) CalculatePercentiles(results []CheckResult) {
	var times []int
	for _, r := range results {
		if r.Status != StatusUp && r.Status != StatusDegraded {
			continue
		}
		if r.ResponseTimeMs != nil && r.StatusCode != nil && *r.StatusCode >= 200 && *r.StatusCode < 400 {
			times = append(times, *r.ResponseTimeMs)
		}
	}

	if len(times) == 0 {
		return
	}

	sort.Ints(times)

	s.P50 = percentile(times, 50)
	s.P95 = percentile(times, 95)
	s.P99 = percentile(times, 99)
}

func percentile(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	index := (p * (len(sorted) - 1)) / 100
	lower := index
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[lower]
	}

	weight := float64(p*(len(sorted)-1))/100.0 - float64(lower)
	return int(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// AddIncidents добавляет информацию об инцидентах.
func (s *UptimeStats) AddIncidents(incidents []Incident) {
	s.Incidents = len(incidents)

	totalDowntime := time.Duration(0)
	for _, i := range incidents {
		if i.DurationSeconds != nil {
			totalDowntime += time.Duration(*i.DurationSeconds) * time.Second
		}
	}
	s.TotalDowntime = totalDowntime
}

// GetAvailabilityString возвращает строковое представление доступности.
func (s *UptimeStats) GetAvailabilityString() string {
	if s.Note != "" {
		return s.Note
	}

	switch {
	case s.Uptime >= 99.9:
		return "Excellent"
	case s.Uptime >= 99.0:
		return "Good"
	case s.Uptime >= 95.0:
		return "Fair"
	case s.Uptime >= 90.0:
		return "Poor"
	default:
		return "Critical"
	}
}

// UptimePeriod представляет период для расчёта uptime.
type UptimePeriod string

const (
	PeriodLastHour   UptimePeriod = "1h"
	PeriodLast24h    UptimePeriod = "24h"
	PeriodLast7Days  UptimePeriod = "7d"
	PeriodLast30Days UptimePeriod = "30d"
	PeriodCustom     UptimePeriod = "custom"
)

// ParseUptimePeriod парсит период и возвращает from/to.
func ParseUptimePeriod(period UptimePeriod, customFrom, customTo time.Time) (from, to time.Time, err error) {
	to = time.Now()

	switch period {
	case PeriodLastHour:
		from = to.Add(-1 * time.Hour)
	case PeriodLast24h:
		from = to.Add(-24 * time.Hour)
	case PeriodLast7Days:
		from = to.Add(-7 * 24 * time.Hour)
	case PeriodLast30Days:
		from = to.Add(-30 * 24 * time.Hour)
	case PeriodCustom:
		if customFrom.IsZero() || customTo.IsZero() {
			return time.Time{}, time.Time{}, ErrInvalidCustomPeriod
		}
		from = customFrom
		to = customTo
	default:
		return time.Time{}, time.Time{}, ErrInvalidPeriod
	}

	return from, to, nil
}

var (
	ErrInvalidPeriod       = errors.New("invalid uptime period")
	ErrInvalidCustomPeriod = errors.New("custom period requires from and to")
)
