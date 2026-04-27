package model

import (
	"time"

	"github.com/google/uuid"
)

// Payment представляет платеж.
type Payment struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	SubscriptionID    *uuid.UUID
	Provider          string // YOOKASSA, STRIPE
	ProviderPaymentID string
	Status            string // PENDING, SUCCESS, FAILED, REFUNDED
	AmountKopeks      int64
	Currency          string
	Metadata          map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Payment статусы.
const (
	PaymentStatusPending  = "PENDING"
	PaymentStatusSuccess  = "SUCCESS"
	PaymentStatusFailed   = "FAILED"
	PaymentStatusRefunded = "REFUNDED"
)

// Payment провайдеры.
const (
	ProviderYookassa = "YOOKASSA"
	ProviderStripe   = "STRIPE"
)

// IsSuccessful проверяет, успешен ли платеж.
func (p *Payment) IsSuccessful() bool {
	return p.Status == PaymentStatusSuccess
}

// IsPending проверяет, в ожидании ли платеж.
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// MarkAsSuccess отмечает платеж как успешный.
func (p *Payment) MarkAsSuccess() error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidPaymentStatus
	}

	p.Status = PaymentStatusSuccess
	p.UpdatedAt = time.Now()

	return nil
}

// MarkAsFailed отмечает платеж как неудачный.
func (p *Payment) MarkAsFailed(reason string) error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidPaymentStatus
	}

	p.Status = PaymentStatusFailed
	p.UpdatedAt = time.Now()

	if p.Metadata == nil {
		p.Metadata = make(map[string]any)
	}
	p.Metadata["failure_reason"] = reason

	return nil
}

// Refund выполняет возврат платежа.
func (p *Payment) Refund() error {
	if !p.IsSuccessful() {
		return ErrCannotRefundNonSuccessfulPayment
	}

	p.Status = PaymentStatusRefunded
	p.UpdatedAt = time.Now()

	return nil
}

// NewPayment создаёт новый платеж.
func NewPayment(userID uuid.UUID, subscriptionID uuid.UUID, provider, providerPaymentID string, amountKopeks int64, currency string) *Payment {
	now := time.Now()
	return &Payment{
		ID:                uuid.New(),
		UserID:            userID,
		SubscriptionID:    &subscriptionID,
		Provider:          provider,
		ProviderPaymentID: providerPaymentID,
		Status:            PaymentStatusPending,
		AmountKopeks:      amountKopeks,
		Currency:          currency,
		Metadata:          make(map[string]any),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// Errors.
var (
	ErrInvalidPaymentStatus             = &ValidationError{Field: "status", Message: "invalid payment status for this operation"}
	ErrCannotRefundNonSuccessfulPayment = &ValidationError{Field: "status", Message: "cannot refund non-successful payment"}
)
