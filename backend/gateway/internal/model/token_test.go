package model

import (
	"errors"
	"testing"
)

func TestNewTokenValidationResult(t *testing.T) {
	result := NewTokenValidationResult(true, "user123", "test@example.com", "premium")

	if !result.Valid {
		t.Error("expected valid to be true")
	}

	if result.UserID != "user123" {
		t.Errorf("expected user ID 'user123', got '%s'", result.UserID)
	}

	if result.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", result.Email)
	}

	if result.Tier != "premium" {
		t.Errorf("expected tier 'premium', got '%s'", result.Tier)
	}
}

func TestNewTokenValidationError(t *testing.T) {
	err := errors.New("test error")
	result := NewTokenValidationError(err)

	if result.Valid {
		t.Error("expected valid to be false")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if result.Error != err {
		t.Errorf("expected error '%v', got '%v'", err, result.Error)
	}
}

func TestTokenValidationRequest(t *testing.T) {
	req := TokenValidationRequest{
		AccessToken: "test-token-123",
	}

	if req.AccessToken != "test-token-123" {
		t.Errorf("expected access token 'test-token-123', got '%s'", req.AccessToken)
	}
}

func TestTokenValidationResult(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := &TokenValidationResult{
			Valid:  true,
			UserID: "user123",
			Email:  "test@example.com",
			Tier:   "premium",
		}

		if !result.Valid {
			t.Error("expected valid to be true")
		}

		if result.UserID != "user123" {
			t.Errorf("expected user ID 'user123', got '%s'", result.UserID)
		}
	})

	t.Run("invalid result", func(t *testing.T) {
		result := &TokenValidationResult{
			Valid: false,
		}

		if result.Valid {
			t.Error("expected valid to be false")
		}
	})
}

func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidToken", ErrInvalidToken},
		{"ErrExpiredToken", ErrExpiredToken},
		{"ErrMissingToken", ErrMissingToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("expected error to be non-nil")
			}

			if errors.Is(tt.err, ErrInvalidToken) || errors.Is(tt.err, ErrExpiredToken) || errors.Is(tt.err, ErrMissingToken) {
				// Success - error matches one of the expected errors
			} else {
				t.Errorf("error %v does not match expected types", tt.err)
			}
		})
	}
}

func TestTokenValidationResultError(t *testing.T) {
	t.Run("result with error", func(t *testing.T) {
		err := errors.New("test error")
		result := &TokenValidationResult{
			Valid: false,
			Error: err,
		}

		if result.Error == nil {
			t.Error("expected error to be set")
		}

		if !errors.Is(result.Error, err) {
			t.Error("expected error to match")
		}
	})
}
