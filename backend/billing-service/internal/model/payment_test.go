package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestPayment_IsSuccessful(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"successful payment", PaymentStatusSuccess, true},
		{"pending payment", PaymentStatusPending, false},
		{"failed payment", PaymentStatusFailed, false},
		{"refunded payment", PaymentStatusRefunded, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payment{Status: tt.status}
			if got := p.IsSuccessful(); got != tt.want {
				t.Errorf("Payment.IsSuccessful() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPayment_MarkAsSuccess(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		wantError error
	}{
		{
			name:      "mark pending as success",
			status:    PaymentStatusPending,
			wantError: nil,
		},
		{
			name:      "cannot mark failed as success",
			status:    PaymentStatusFailed,
			wantError: ErrInvalidPaymentStatus,
		},
		{
			name:      "cannot mark already success",
			status:    PaymentStatusSuccess,
			wantError: ErrInvalidPaymentStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payment{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Status:   tt.status,
				Metadata: make(map[string]any),
			}

			err := p.MarkAsSuccess()
			if err != tt.wantError {
				t.Errorf("Payment.MarkAsSuccess() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError == nil {
				if p.Status != PaymentStatusSuccess {
					t.Errorf("Payment Status = %v, want %v", p.Status, PaymentStatusSuccess)
				}
			}
		})
	}
}

func TestPayment_MarkAsFailed(t *testing.T) {
	reason := "insufficient funds"

	tests := []struct {
		name      string
		status    string
		wantError error
	}{
		{
			name:      "mark pending as failed",
			status:    PaymentStatusPending,
			wantError: nil,
		},
		{
			name:      "cannot mark success as failed",
			status:    PaymentStatusSuccess,
			wantError: ErrInvalidPaymentStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payment{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Status:   tt.status,
				Metadata: make(map[string]any),
			}

			err := p.MarkAsFailed(reason)
			if err != tt.wantError {
				t.Errorf("Payment.MarkAsFailed() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError == nil {
				if p.Status != PaymentStatusFailed {
					t.Errorf("Payment Status = %v, want %v", p.Status, PaymentStatusFailed)
				}
				if p.Metadata["failure_reason"] != reason {
					t.Errorf("Payment metadata failure_reason = %v, want %v", p.Metadata["failure_reason"], reason)
				}
			}
		})
	}
}

func TestPayment_Refund(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		wantError error
	}{
		{
			name:      "refund successful payment",
			status:    PaymentStatusSuccess,
			wantError: nil,
		},
		{
			name:      "cannot refund pending payment",
			status:    PaymentStatusPending,
			wantError: ErrCannotRefundNonSuccessfulPayment,
		},
		{
			name:      "cannot refund failed payment",
			status:    PaymentStatusFailed,
			wantError: ErrCannotRefundNonSuccessfulPayment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payment{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Status:   tt.status,
				Metadata: make(map[string]any),
			}

			err := p.Refund()
			if err != tt.wantError {
				t.Errorf("Payment.Refund() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError == nil {
				if p.Status != PaymentStatusRefunded {
					t.Errorf("Payment Status = %v, want %v", p.Status, PaymentStatusRefunded)
				}
			}
		})
	}
}

func TestNewPayment(t *testing.T) {
	userID := uuid.New()
	subscriptionID := uuid.New()
	provider := ProviderYookassa
	providerPaymentID := "yp_123456"
	amountKopeks := int64(29900)
	currency := "RUB"

	p := NewPayment(userID, subscriptionID, provider, providerPaymentID, amountKopeks, currency)

	if p.ID == uuid.Nil {
		t.Error("NewPayment() ID = nil, want non-nil")
	}
	if p.UserID != userID {
		t.Errorf("NewPayment() UserID = %v, want %v", p.UserID, userID)
	}
	if p.SubscriptionID == nil {
		t.Error("NewPayment() SubscriptionID = nil, want non-nil")
	}
	if *p.SubscriptionID != subscriptionID {
		t.Errorf("NewPayment() SubscriptionID = %v, want %v", *p.SubscriptionID, subscriptionID)
	}
	if p.Provider != provider {
		t.Errorf("NewPayment() Provider = %v, want %v", p.Provider, provider)
	}
	if p.ProviderPaymentID != providerPaymentID {
		t.Errorf("NewPayment() ProviderPaymentID = %v, want %v", p.ProviderPaymentID, providerPaymentID)
	}
	if p.AmountKopeks != amountKopeks {
		t.Errorf("NewPayment() AmountKopeks = %v, want %v", p.AmountKopeks, amountKopeks)
	}
	if p.Currency != currency {
		t.Errorf("NewPayment() Currency = %v, want %v", p.Currency, currency)
	}
	if p.Status != PaymentStatusPending {
		t.Errorf("NewPayment() Status = %v, want %v", p.Status, PaymentStatusPending)
	}
}
