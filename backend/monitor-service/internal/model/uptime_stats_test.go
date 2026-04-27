package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUptimeStats_Calculate(t *testing.T) {
	monitorID := uuid.New()
	now := time.Now()

	tests := []struct {
		name     string
		results  []CheckResult
		expected float64
		note     string
	}{
		{
			name:     "all up",
			results:  createResults(monitorID, 10, StatusUp, now),
			expected: 100.0,
			note:     "",
		},
		{
			name:     "all down",
			results:  createResults(monitorID, 10, StatusDown, now),
			expected: 0.0,
			note:     "",
		},
		{
			name:     "half up half down",
			results:  append(createResults(monitorID, 5, StatusUp, now), createResults(monitorID, 5, StatusDown, now)...),
			expected: 50.0,
			note:     "",
		},
		{
			name:     "mixed with degraded",
			results:  append(append(createResults(monitorID, 4, StatusUp, now), createResults(monitorID, 2, StatusDegraded, now)...), createResults(monitorID, 4, StatusDown, now)...),
			expected: 50.0, // (4 + 2*0.5) / 10 * 100 = 50%
			note:     "",
		},
		{
			name:     "all paused",
			results:  createResults(monitorID, 10, StatusPaused, now),
			expected: 100.0, // No active checks (all paused), default to 100%
			note:     "",
		},
		{
			name:     "empty results",
			results:  []CheckResult{},
			expected: 100.0, // No checks, default to 100%
			note:     "No checks in period",
		},
		{
			name:     "mixed with paused excluded",
			results:  append(append(createResults(monitorID, 5, StatusUp, now), createResults(monitorID, 3, StatusPaused, now)...), createResults(monitorID, 2, StatusDown, now)...),
			expected: 71.42857142857143, // (5 + 0) / 7 * 100 = 71.43% (paused excluded from total)
			note:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewUptimeStats()
			stats.Calculate(tt.results)

			// Count active checks (excluding paused)
			activeCount := 0
			for _, r := range tt.results {
				if r.Status != StatusPaused {
					activeCount++
				}
			}

			if stats.TotalChecks != activeCount {
				t.Errorf("UptimeStats.TotalChecks = %v, want %v (active count)", stats.TotalChecks, activeCount)
			}

			// Check uptime with tolerance for floating point
			if activeCount > 0 {
				if stats.Uptime < tt.expected-0.01 || stats.Uptime > tt.expected+0.01 {
					t.Errorf("UptimeStats.Uptime = %v, want %v", stats.Uptime, tt.expected)
				}
			} else {
				if stats.Note != tt.note && tt.note != "" {
					t.Errorf("UptimeStats.Note = %v, want %v", stats.Note, tt.note)
				}
			}
		})
	}
}

func TestUptimeStats_CalculateWithNote(t *testing.T) {
	monitorID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		results []CheckResult
		note    string
	}{
		{
			name:    "empty results",
			results: []CheckResult{},
			note:    "No checks in period",
		},
		{
			name:    "all paused",
			results: createResults(monitorID, 10, StatusPaused, now),
			note:    "No active checks in period",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewUptimeStats()
			stats.Calculate(tt.results)

			if stats.Note != tt.note {
				t.Errorf("UptimeStats.Note = %v, want %v", stats.Note, tt.note)
			}
		})
	}
}

func TestCheckResultSlice_CalculateFailureRate(t *testing.T) {
	monitorID := uuid.New()
	now := time.Now()

	tests := []struct {
		name     string
		results  CheckResultSlice
		expected float64
	}{
		{
			name:     "no failures",
			results:  CheckResultSlice(createResults(monitorID, 10, StatusUp, now)),
			expected: 0.0,
		},
		{
			name:     "all failures",
			results:  CheckResultSlice(createResults(monitorID, 10, StatusDown, now)),
			expected: 100.0,
		},
		{
			name:     "50% failures",
			results:  CheckResultSlice(append(createResults(monitorID, 5, StatusUp, now), createResults(monitorID, 5, StatusDown, now)...)),
			expected: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.results.CalculateFailureRate()
			if got != tt.expected {
				t.Errorf("CheckResultSlice.CalculateFailureRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to create test results
func createResults(monitorID uuid.UUID, count int, status MonitorStatus, timestamp time.Time) []CheckResult {
	results := make([]CheckResult, count)
	for i := 0; i < count; i++ {
		results[i] = *NewCheckResult(monitorID, status)
		results[i].CheckedAt = timestamp.Add(time.Duration(i) * time.Second)
	}
	return results
}

func TestUptimeStats_CalculatePercentiles(t *testing.T) {
	t.Parallel()

	t.Run("empty results", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.CalculatePercentiles(nil)
		assert.Equal(t, 0, stats.P50)
		assert.Equal(t, 0, stats.P95)
		assert.Equal(t, 0, stats.P99)
	})

	t.Run("single 2xx value", func(t *testing.T) {
		monitorID := uuid.New()
		results := []CheckResult{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100).WithStatusCode(200),
		}

		stats := NewUptimeStats()
		stats.CalculatePercentiles(results)

		assert.Equal(t, 100, stats.P50)
		assert.Equal(t, 100, stats.P95)
		assert.Equal(t, 100, stats.P99)
	})

	t.Run("only 2xx results", func(t *testing.T) {
		monitorID := uuid.New()
		results := []CheckResult{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(10).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(20).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(30).WithStatusCode(201),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(40).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(50).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(60).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(70).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(80).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(90).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100).WithStatusCode(200),
		}

		stats := NewUptimeStats()
		stats.CalculatePercentiles(results)

		assert.Equal(t, 55, stats.P50)
		assert.Equal(t, 95, stats.P95)
		assert.Equal(t, 99, stats.P99)
	})

	t.Run("excludes 5xx results", func(t *testing.T) {
		monitorID := uuid.New()
		results := []CheckResult{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusDown).WithResponseTime(50).WithStatusCode(500),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(200).WithStatusCode(200),
		}

		stats := NewUptimeStats()
		stats.CalculatePercentiles(results)

		assert.Equal(t, 150, stats.P50)
		assert.Equal(t, 195, stats.P95)
		assert.Equal(t, 199, stats.P99)
	})

	t.Run("excludes 4xx results", func(t *testing.T) {
		monitorID := uuid.New()
		results := []CheckResult{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusDown).WithResponseTime(50).WithStatusCode(404),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(200).WithStatusCode(200),
		}

		stats := NewUptimeStats()
		stats.CalculatePercentiles(results)

		assert.Equal(t, 150, stats.P50)
		assert.Equal(t, 195, stats.P95)
		assert.Equal(t, 199, stats.P99)
	})

	t.Run("excludes results without response time", func(t *testing.T) {
		monitorID := uuid.New()
		results := []CheckResult{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(300).WithStatusCode(200),
		}

		stats := NewUptimeStats()
		stats.CalculatePercentiles(results)

		assert.Equal(t, 200, stats.P50)
		assert.Equal(t, 290, stats.P95)
		assert.Equal(t, 298, stats.P99)
	})

	t.Run("includes degraded with 2xx", func(t *testing.T) {
		monitorID := uuid.New()
		results := []CheckResult{
			*NewCheckResult(monitorID, StatusDegraded).WithResponseTime(100).WithStatusCode(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(200).WithStatusCode(200),
		}

		stats := NewUptimeStats()
		stats.CalculatePercentiles(results)

		assert.Equal(t, 150, stats.P50)
	})

	t.Run("excludes results without status code", func(t *testing.T) {
		monitorID := uuid.New()
		results := []CheckResult{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(200).WithStatusCode(200),
		}

		stats := NewUptimeStats()
		stats.CalculatePercentiles(results)

		assert.Equal(t, 200, stats.P50)
		assert.Equal(t, 200, stats.P95)
		assert.Equal(t, 200, stats.P99)
	})
}

func TestPercentile(t *testing.T) {
	t.Parallel()

	t.Run("empty slice", func(t *testing.T) {
		assert.Equal(t, 0, percentile(nil, 50))
	})

	t.Run("single element", func(t *testing.T) {
		assert.Equal(t, 42, percentile([]int{42}, 50))
		assert.Equal(t, 42, percentile([]int{42}, 99))
	})
}
