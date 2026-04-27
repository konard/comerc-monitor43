package model

import (
	"testing"
)

func TestNewUserContext(t *testing.T) {
	userCtx := NewUserContext("user123", "test@example.com", "premium")

	if userCtx.UserID != "user123" {
		t.Errorf("expected user ID 'user123', got '%s'", userCtx.UserID)
	}

	if userCtx.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", userCtx.Email)
	}

	if userCtx.Tier != "premium" {
		t.Errorf("expected tier 'premium', got '%s'", userCtx.Tier)
	}

	if !userCtx.Valid {
		t.Error("expected valid to be true")
	}

	if userCtx.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestUserContext_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		userCtx  *UserContext
		expected bool
	}{
		{
			name: "valid user context",
			userCtx: &UserContext{
				UserID: "user123",
				Valid:  true,
			},
			expected: true,
		},
		{
			name: "invalid user context",
			userCtx: &UserContext{
				UserID: "user123",
				Valid:  false,
			},
			expected: false,
		},
		{
			name: "empty user ID",
			userCtx: &UserContext{
				UserID: "",
				Valid:  true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.userCtx.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUserContext_HasTier(t *testing.T) {
	tests := []struct {
		name     string
		userCtx  *UserContext
		tier     string
		expected bool
	}{
		{
			name: "matching tier",
			userCtx: &UserContext{
				UserID: "user123",
				Tier:   "premium",
			},
			tier:     "premium",
			expected: true,
		},
		{
			name: "non-matching tier",
			userCtx: &UserContext{
				UserID: "user123",
				Tier:   "premium",
			},
			tier:     "free",
			expected: false,
		},
		{
			name: "case sensitive",
			userCtx: &UserContext{
				UserID: "user123",
				Tier:   "premium",
			},
			tier:     "Premium",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.userCtx.HasTier(tt.tier)
			if got != tt.expected {
				t.Errorf("HasTier(%q) = %v, want %v", tt.tier, got, tt.expected)
			}
		})
	}
}

func TestUserContext_IsPremium(t *testing.T) {
	tests := []struct {
		name     string
		userCtx  *UserContext
		expected bool
	}{
		{
			name: "premium tier",
			userCtx: &UserContext{
				UserID: "user123",
				Tier:   "premium",
			},
			expected: true,
		},
		{
			name: "free tier",
			userCtx: &UserContext{
				UserID: "user123",
				Tier:   "free",
			},
			expected: false,
		},
		{
			name: "empty tier",
			userCtx: &UserContext{
				UserID: "user123",
				Tier:   "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.userCtx.IsPremium()
			if got != tt.expected {
				t.Errorf("IsPremium() = %v, want %v", got, tt.expected)
			}
		})
	}
}
