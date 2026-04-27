package model

import (
	"time"

	"github.com/google/uuid"
)

// Subscription представляет подписку пользователя.
type Subscription struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	PlanID            string // TIER_FREE, TIER_STARTER, etc.
	Status            string // ACTIVE, PENDING, CANCELED, EXPIRED
	StartedAt         time.Time
	ExpiresAt         time.Time
	CanceledAt        *time.Time
	AutoRenew         bool
	ProviderPaymentID *string
	Metadata          map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Subscription статусы.
const (
	StatusActive   = "ACTIVE"
	StatusPending  = "PENDING"
	StatusCanceled = "CANCELED"
	StatusExpired  = "EXPIRED"
)

// IsActive проверяет, активна ли подписка.
func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive && !s.IsExpired()
}

// IsExpired проверяет, истекла ли подписка.
func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// CanBeCanceled проверяет, можно ли отменить подписку.
func (s *Subscription) CanBeCanceled() bool {
	return s.Status == StatusActive || s.Status == StatusPending
}

// Cancel отменяет подписку с указанием причины.
func (s *Subscription) Cancel(reason string) error {
	if !s.CanBeCanceled() {
		return ErrCannotCancelSubscription
	}

	now := time.Now()
	s.Status = StatusCanceled
	s.CanceledAt = &now
	s.AutoRenew = false
	s.UpdatedAt = now

	if s.Metadata == nil {
		s.Metadata = make(map[string]any)
	}
	s.Metadata["cancel_reason"] = reason

	return nil
}

// Renew продлевает подписку на указанный период.
func (s *Subscription) Renew(duration time.Duration) error {
	if s.Status == StatusCanceled {
		return ErrCannotRenewCanceledSubscription
	}

	s.ExpiresAt = s.ExpiresAt.Add(duration)
	s.Status = StatusActive
	s.UpdatedAt = time.Now()

	return nil
}

// NewSubscription создаёт новую подписку.
func NewSubscription(userID uuid.UUID, planID string, duration time.Duration) *Subscription {
	now := time.Now()
	return &Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		PlanID:    planID,
		Status:    StatusPending,
		StartedAt: now,
		ExpiresAt: now.Add(duration),
		AutoRenew: true,
		Metadata:  make(map[string]any),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Errors.
var (
	ErrCannotCancelSubscription        = &ValidationError{Field: "status", Message: "subscription cannot be canceled"}
	ErrCannotRenewCanceledSubscription = &ValidationError{Field: "status", Message: "cannot renew canceled subscription"}
)
