package domain

import (
	"testing"
)

func TestMonitorStatus_IsValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status MonitorStatus
		want   bool
	}{
		{
			name:   "valid pending",
			status: StatusPending,
			want:   true,
		},
		{
			name:   "valid up",
			status: StatusUp,
			want:   true,
		},
		{
			name:   "valid down",
			status: StatusDown,
			want:   true,
		},
		{
			name:   "valid degraded",
			status: StatusDegraded,
			want:   true,
		},
		{
			name:   "valid paused",
			status: StatusPaused,
			want:   true,
		},
		{
			name:   "invalid status",
			status: MonitorStatus("INVALID"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("MonitorStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMonitorStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		from     MonitorStatus
		to       MonitorStatus
		expected bool
	}{
		{
			name:     "pending to up",
			from:     StatusPending,
			to:       StatusUp,
			expected: true,
		},
		{
			name:     "pending to down",
			from:     StatusPending,
			to:       StatusDown,
			expected: true,
		},
		{
			name:     "up to down",
			from:     StatusUp,
			to:       StatusDown,
			expected: true,
		},
		{
			name:     "up to degraded",
			from:     StatusUp,
			to:       StatusDegraded,
			expected: true,
		},
		{
			name:     "up to paused",
			from:     StatusUp,
			to:       StatusPaused,
			expected: true,
		},
		{
			name:     "paused to up",
			from:     StatusPaused,
			to:       StatusUp,
			expected: true,
		},
		{
			name:     "paused to down",
			from:     StatusPaused,
			to:       StatusDown,
			expected: false,
		},
		{
			name:     "down to paused",
			from:     StatusDown,
			to:       StatusPaused,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.expected {
				t.Errorf("MonitorStatus.CanTransitionTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMonitorStatus_IsActive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status MonitorStatus
		want   bool
	}{
		{
			name:   "paused is not active",
			status: StatusPaused,
			want:   false,
		},
		{
			name:   "up is active",
			status: StatusUp,
			want:   true,
		},
		{
			name:   "down is active",
			status: StatusDown,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.want {
				t.Errorf("MonitorStatus.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMonitorStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		s       string
		want    MonitorStatus
		wantErr bool
	}{
		{
			name:    "valid pending",
			s:       "PENDING",
			want:    StatusPending,
			wantErr: false,
		},
		{
			name:    "valid up",
			s:       "UP",
			want:    StatusUp,
			wantErr: false,
		},
		{
			name:    "invalid status",
			s:       "INVALID",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMonitorStatus(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMonitorStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseMonitorStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
