package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSubscription_IsActive(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		expiresAt  time.Time
		wantActive bool
	}{
		{
			name:       "active and not expired",
			status:     StatusActive,
			expiresAt:  time.Now().Add(24 * time.Hour),
			wantActive: true,
		},
		{
			name:       "active but expired",
			status:     StatusActive,
			expiresAt:  time.Now().Add(-24 * time.Hour),
			wantActive: false,
		},
		{
			name:       "pending status",
			status:     StatusPending,
			expiresAt:  time.Now().Add(24 * time.Hour),
			wantActive: false,
		},
		{
			name:       "canceled status",
			status:     StatusCanceled,
			expiresAt:  time.Now().Add(24 * time.Hour),
			wantActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Subscription{
				Status:    tt.status,
				ExpiresAt: tt.expiresAt,
			}
			if got := s.IsActive(); got != tt.wantActive {
				t.Errorf("Subscription.IsActive() = %v, want %v", got, tt.wantActive)
			}
		})
	}
}

func TestSubscription_Cancel(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		wantError error
	}{
		{
			name:      "cancel active subscription",
			status:    StatusActive,
			wantError: nil,
		},
		{
			name:      "cancel pending subscription",
			status:    StatusPending,
			wantError: nil,
		},
		{
			name:      "cannot cancel canceled subscription",
			status:    StatusCanceled,
			wantError: ErrCannotCancelSubscription,
		},
		{
			name:      "cannot cancel expired subscription",
			status:    StatusExpired,
			wantError: ErrCannotCancelSubscription,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Subscription{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				PlanID:    PlanTierFree,
				Status:    tt.status,
				ExpiresAt: time.Now().Add(24 * time.Hour),
				AutoRenew: true,
				Metadata:  make(map[string]any),
			}

			err := s.Cancel("test reason")
			if err != tt.wantError {
				t.Errorf("Subscription.Cancel() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError == nil {
				if s.Status != StatusCanceled {
					t.Errorf("Subscription status = %v, want %v", s.Status, StatusCanceled)
				}
				if s.AutoRenew {
					t.Errorf("Subscription AutoRenew = true, want false")
				}
				if s.CanceledAt == nil {
					t.Errorf("Subscription CanceledAt = nil, want non-nil")
				}
			}
		})
	}
}

func TestSubscription_Renew(t *testing.T) {
	duration := 30 * 24 * time.Hour

	tests := []struct {
		name      string
		status    string
		expiresAt time.Time
		wantError error
	}{
		{
			name:      "renew active subscription",
			status:    StatusActive,
			expiresAt: time.Now().Add(24 * time.Hour),
			wantError: nil,
		},
		{
			name:      "renew expired subscription",
			status:    StatusExpired,
			expiresAt: time.Now().Add(-24 * time.Hour),
			wantError: nil,
		},
		{
			name:      "cannot renew canceled subscription",
			status:    StatusCanceled,
			expiresAt: time.Now().Add(24 * time.Hour),
			wantError: ErrCannotRenewCanceledSubscription,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Subscription{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				PlanID:    PlanTierFree,
				Status:    tt.status,
				ExpiresAt: tt.expiresAt,
			}

			oldExpiresAt := s.ExpiresAt
			err := s.Renew(duration)
			if err != tt.wantError {
				t.Errorf("Subscription.Renew() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError == nil {
				if s.Status != StatusActive {
					t.Errorf("Subscription status = %v, want %v", s.Status, StatusActive)
				}
				expectedExpiry := oldExpiresAt.Add(duration)
				if s.ExpiresAt.Before(expectedExpiry) || s.ExpiresAt.After(expectedExpiry.Add(time.Second)) {
					t.Errorf("Subscription ExpiresAt = %v, want approximately %v", s.ExpiresAt, expectedExpiry)
				}
			}
		})
	}
}

func TestNewSubscription(t *testing.T) {
	userID := uuid.New()
	planID := PlanTierStarter
	duration := 30 * 24 * time.Hour

	sub := NewSubscription(userID, planID, duration)

	if sub.ID == uuid.Nil {
		t.Error("NewSubscription() ID = nil, want non-nil")
	}
	if sub.UserID != userID {
		t.Errorf("NewSubscription() UserID = %v, want %v", sub.UserID, userID)
	}
	if sub.PlanID != planID {
		t.Errorf("NewSubscription() PlanID = %v, want %v", sub.PlanID, planID)
	}
	if sub.Status != StatusPending {
		t.Errorf("NewSubscription() Status = %v, want %v", sub.Status, StatusPending)
	}
	if !sub.AutoRenew {
		t.Error("NewSubscription() AutoRenew = false, want true")
	}
	if sub.Metadata == nil {
		t.Error("NewSubscription() Metadata = nil, want initialized map")
	}
}
