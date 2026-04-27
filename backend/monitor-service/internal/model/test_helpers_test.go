package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func mustNewWorkingHours(tb testing.TB, start, end string, days []time.Weekday) *WorkingHours {
	tb.Helper()

	wh, err := NewWorkingHours(start, end, days)
	if err != nil {
		tb.Fatalf("new working hours: %v", err)
	}

	return wh
}

func mustNewMonitor(tb testing.TB, userID uuid.UUID, name, url string, interval int) *Monitor {
	tb.Helper()

	monitor, err := NewMonitor(userID, name, url, interval)
	if err != nil {
		tb.Fatalf("new monitor: %v", err)
	}

	return monitor
}
