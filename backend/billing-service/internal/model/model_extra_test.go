package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayment_IsPending(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"pending", PaymentStatusPending, true},
		{"success", PaymentStatusSuccess, false},
		{"failed", PaymentStatusFailed, false},
		{"refunded", PaymentStatusRefunded, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := &Payment{Status: tc.status}
			assert.Equal(t, tc.want, p.IsPending())
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	err := &ValidationError{Field: "status", Message: "invalid value"}

	msg := err.Error()

	assert.Contains(t, msg, "status")
	assert.Contains(t, msg, "invalid value")
}
