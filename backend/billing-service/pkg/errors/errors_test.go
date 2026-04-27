package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotFound(t *testing.T) {
	t.Parallel()

	err := NotFound("subscription", "abc-123")

	assert.ErrorIs(t, err, ErrNotFound)
	assert.Contains(t, err.Error(), "subscription")
	assert.Contains(t, err.Error(), "abc-123")
}

func TestAlreadyExists(t *testing.T) {
	t.Parallel()

	err := AlreadyExists("plan", "TIER_FREE")

	assert.ErrorIs(t, err, ErrAlreadyExists)
	assert.Contains(t, err.Error(), "plan")
	assert.Contains(t, err.Error(), "TIER_FREE")
}

func TestInvalidArgument(t *testing.T) {
	t.Parallel()

	err := InvalidArgument("user_id", "required")

	assert.ErrorIs(t, err, ErrInvalidArgument)
	assert.Contains(t, err.Error(), "user_id")
	assert.Contains(t, err.Error(), "required")
}

func TestPaymentFailed(t *testing.T) {
	t.Parallel()

	err := PaymentFailed("insufficient funds")

	assert.ErrorIs(t, err, ErrPaymentFailed)
	assert.Contains(t, err.Error(), "insufficient funds")
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	// Проверяем, что все sentinel-ошибки не nil
	assert.NotNil(t, ErrNotFound)
	assert.NotNil(t, ErrAlreadyExists)
	assert.NotNil(t, ErrInvalidArgument)
	assert.NotNil(t, ErrPermissionDenied)
	assert.NotNil(t, ErrUnauthenticated)
	assert.NotNil(t, ErrInternal)
	assert.NotNil(t, ErrPaymentFailed)
	assert.NotNil(t, ErrSubscriptionExpired)
	assert.NotNil(t, ErrSubscriptionCanceled)
	assert.NotNil(t, ErrPlanLimitExceeded)
}

func TestNotFound_differentTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		entity string
		id     any
	}{
		{"string id", "payment", "pay-123"},
		{"int id", "subscription", 42},
		{"uuid id", "plan", "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NotFound(tc.entity, tc.id)

			assert.ErrorIs(t, err, ErrNotFound)
		})
	}
}
