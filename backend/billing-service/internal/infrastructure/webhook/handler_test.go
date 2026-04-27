package webhook

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/payment"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Mock repositories
type mockSubscriptionRepository struct {
	subscriptions map[string]*Subscription
}

func newMockSubscriptionRepository() *mockSubscriptionRepository {
	userID := uuid.New().String()
	subID := uuid.New().String()

	return &mockSubscriptionRepository{
		subscriptions: map[string]*Subscription{
			subID: {
				ID:        subID,
				UserID:    userID,
				PlanID:    "TIER_STARTER",
				Status:    "PENDING",
				ExpiresAt: "2026-04-27T00:00:00Z",
			},
		},
	}
}

func (m *mockSubscriptionRepository) GetByID(_ context.Context, id string) (*Subscription, error) {
	sub, ok := m.subscriptions[id]
	if !ok {
		return nil, &NotFoundError{ID: id}
	}
	return sub, nil
}

func (m *mockSubscriptionRepository) Update(_ context.Context, sub *Subscription) error {
	m.subscriptions[sub.ID] = sub
	return nil
}

type mockPaymentRepository struct {
	payments map[string]*Payment
}

func newMockPaymentRepository() *mockPaymentRepository {
	subIDStr := uuid.New().String()
	paymentID := uuid.New()

	return &mockPaymentRepository{
		payments: map[string]*Payment{
			"yp_test_123": {
				ID:                paymentID.String(),
				UserID:            uuid.New().String(),
				SubscriptionID:    &subIDStr,
				Status:            "PENDING",
				Provider:          "YOOKASSA",
				ProviderPaymentID: "yp_test_123",
			},
		},
	}
}

func (m *mockPaymentRepository) GetByProviderPaymentID(_ context.Context, providerPaymentID string) (*Payment, error) {
	pmt, ok := m.payments[providerPaymentID]
	if !ok {
		return nil, &NotFoundError{ID: providerPaymentID}
	}
	return pmt, nil
}

func (m *mockPaymentRepository) Update(_ context.Context, pmt *Payment) error {
	m.payments[pmt.ProviderPaymentID] = pmt
	return nil
}

// NotFoundError represents not found error
type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return "not found: " + e.ID
}

func TestHandler_Handle(t *testing.T) {
	t.Skip("requires payment provider mock")

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	paymentRepo := newMockPaymentRepository()

	handler := NewHandler(
		nil, // yookassa provider
		nil, // stripe provider
		subRepo,
		paymentRepo,
		newTestLogger(),
	)

	t.Run("payment succeeded", func(t *testing.T) {
		// This test requires a payment provider mock
		// For now, we'll skip the actual execution
		_ = handler
		_ = ctx
	})

	t.Run("payment failed", func(t *testing.T) {
		// TODO: implement
	})

	t.Run("refund succeeded", func(t *testing.T) {
		// TODO: implement
	})
}

func TestHandler_activateSubscription(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	paymentRepo := newMockPaymentRepository()

	handler := NewHandler(nil, nil, subRepo, paymentRepo, newTestLogger())

	subID := uuid.New().String()
	sub := &Subscription{
		ID:        subID,
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		Status:    "PENDING",
		ExpiresAt: "2026-04-27T00:00:00Z",
	}
	subRepo.subscriptions[subID] = sub

	err := handler.activateSubscription(ctx, subID)
	require.NoError(t, err)

	updated := subRepo.subscriptions[subID]
	assert.Equal(t, "ACTIVE", updated.Status)
}

func TestHandler_cancelSubscription(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	paymentRepo := newMockPaymentRepository()

	handler := NewHandler(nil, nil, subRepo, paymentRepo, newTestLogger())

	subID := uuid.New().String()
	sub := &Subscription{
		ID:        subID,
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		Status:    "ACTIVE",
		ExpiresAt: "2026-04-27T00:00:00Z",
	}
	subRepo.subscriptions[subID] = sub

	err := handler.cancelSubscription(ctx, subID, "test reason")
	require.NoError(t, err)

	updated := subRepo.subscriptions[subID]
	assert.Equal(t, "CANCELED", updated.Status)
}

func TestHandler_activateSubscription_not_found(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	paymentRepo := newMockPaymentRepository()
	handler := NewHandler(nil, nil, subRepo, paymentRepo, newTestLogger())

	// ID не существует
	err := handler.activateSubscription(ctx, "nonexistent-id")
	assert.Error(t, err)
}

func TestHandler_cancelSubscription_not_found(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	paymentRepo := newMockPaymentRepository()
	handler := NewHandler(nil, nil, subRepo, paymentRepo, newTestLogger())

	// ID не существует
	err := handler.cancelSubscription(ctx, "nonexistent-id", "test reason")
	assert.Error(t, err)
}

func TestHandle_payment_succeeded_update_error(t *testing.T) {
	t.Parallel()

	// Arrange — paymentRepo.Update возвращает ошибку
	providerPaymentID := "yp_succ_upd_err"
	paymentRepo := &mockPaymentRepositoryWithErr{
		payments: map[string]*Payment{
			providerPaymentID: {
				ID:                "pay-succ-err-1",
				UserID:            "user-1",
				Status:            "PENDING",
				Provider:          "YOOKASSA",
				ProviderPaymentID: providerPaymentID,
			},
		},
		updateErr: assert.AnError,
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.succeeded",
			PaymentID: providerPaymentID,
		},
	}

	h := NewHandler(provider, nil, newMockSubscriptionRepository(), paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

func TestHandle_payment_failed_sub_update_error(t *testing.T) {
	t.Parallel()

	// Arrange — paymentRepo.Update успешен, но subRepo.Update возвращает ошибку
	providerPaymentID := "yp_fail_sub_upd"
	subIDStr := "sub-upd-err-id"
	paymentRepo := newMockPaymentRepository()
	paymentRepo.payments[providerPaymentID] = &Payment{
		ID:                "pay-fail-1",
		UserID:            "user-1",
		SubscriptionID:    &subIDStr,
		Status:            "PENDING",
		Provider:          "YOOKASSA",
		ProviderPaymentID: providerPaymentID,
	}

	// subRepo с ошибкой Update
	subRepo := &mockSubscriptionRepositoryWithUpdateErr{
		subscriptions: map[string]*Subscription{
			subIDStr: {
				ID:        subIDStr,
				UserID:    "user-1",
				PlanID:    "TIER_STARTER",
				Status:    "PENDING",
				ExpiresAt: "2026-04-27T00:00:00Z",
			},
		},
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.canceled",
			PaymentID: providerPaymentID,
		},
	}

	h := NewHandler(provider, nil, subRepo, paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

// mockSubscriptionRepositoryWithUpdateErr возвращает ошибку только при Update.
type mockSubscriptionRepositoryWithUpdateErr struct {
	subscriptions map[string]*Subscription
}

func (m *mockSubscriptionRepositoryWithUpdateErr) GetByID(_ context.Context, id string) (*Subscription, error) {
	sub, ok := m.subscriptions[id]
	if !ok {
		return nil, &NotFoundError{ID: id}
	}
	return sub, nil
}

func (m *mockSubscriptionRepositoryWithUpdateErr) Update(_ context.Context, _ *Subscription) error {
	return assert.AnError
}
