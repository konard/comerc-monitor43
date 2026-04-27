package model

import (
	"testing"
	"time"
)

func TestPlan_IsFree(t *testing.T) {
	tests := []struct {
		name        string
		planID      string
		priceKopeks int64
		wantFree    bool
	}{
		{"free tier by ID", PlanTierFree, 0, true},
		{"free tier by price", "CUSTOM", 0, true},
		{"paid plan", PlanTierStarter, 29900, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plan{
				ID:          tt.planID,
				PriceKopeks: tt.priceKopeks,
			}
			if got := p.IsFree(); got != tt.wantFree {
				t.Errorf("Plan.IsFree() = %v, want %v", got, tt.wantFree)
			}
		})
	}
}

func TestPlan_IsValidForMonitors(t *testing.T) {
	tests := []struct {
		name        string
		maxMonitors int32
		count       int
		wantValid   bool
	}{
		{"within limit", 5, 3, true},
		{"at limit", 5, 5, true},
		{"over limit", 5, 6, false},
		{"unlimited", -1, 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plan{MaxMonitors: tt.maxMonitors}
			if got := p.IsValidForMonitors(tt.count); got != tt.wantValid {
				t.Errorf("Plan.IsValidForMonitors() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestPlan_IsValidForCheckInterval(t *testing.T) {
	tests := []struct {
		name        string
		minInterval int32
		seconds     int
		wantValid   bool
	}{
		{"valid interval", 60, 120, true},
		{"exact minimum", 60, 60, true},
		{"too short", 60, 30, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plan{MinCheckIntervalSeconds: tt.minInterval}
			if got := p.IsValidForCheckInterval(tt.seconds); got != tt.wantValid {
				t.Errorf("Plan.IsValidForCheckInterval() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestPlan_IsValidForAlerts(t *testing.T) {
	tests := []struct {
		name      string
		maxAlerts int32
		count     int
		wantValid bool
	}{
		{"within limit", 100, 50, true},
		{"at limit", 100, 100, true},
		{"over limit", 100, 101, false},
		{"unlimited", -1, 10000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plan{MaxAlertsPerDay: tt.maxAlerts}
			if got := p.IsValidForAlerts(tt.count); got != tt.wantValid {
				t.Errorf("Plan.IsValidForAlerts() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestPlan_CalculateExpiryDate(t *testing.T) {
	tests := []struct {
		name        string
		billingDays int32
		wantRange   time.Duration
	}{
		{"30 days", 30, 29 * 24 * time.Hour}, // approximately
		{"90 days", 90, 89 * 24 * time.Hour},
		{"365 days", 365, 364 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plan{BillingPeriodDays: tt.billingDays}
			expiresAt := p.CalculateExpiryDate()
			duration := time.Until(expiresAt)

			if duration < tt.wantRange {
				t.Errorf("Plan.CalculateExpiryDate() duration = %v, want >= %v", duration, tt.wantRange)
			}
		})
	}
}

func TestGetDefaultPlan(t *testing.T) {
	plan := GetDefaultPlan()

	if plan.ID != PlanTierFree {
		t.Errorf("GetDefaultPlan() ID = %v, want %v", plan.ID, PlanTierFree)
	}
	if !plan.IsFree() {
		t.Error("GetDefaultPlan() IsFree = false, want true")
	}
	if plan.MaxMonitors != 5 {
		t.Errorf("GetDefaultPlan() MaxMonitors = %v, want %v", plan.MaxMonitors, 5)
	}
	if plan.MinCheckIntervalSeconds != 300 {
		t.Errorf("GetDefaultPlan() MinCheckIntervalSeconds = %v, want %v", plan.MinCheckIntervalSeconds, 300)
	}
	if len(plan.Features) != 2 {
		t.Errorf("GetDefaultPlan() Features length = %v, want %v", len(plan.Features), 2)
	}
}
