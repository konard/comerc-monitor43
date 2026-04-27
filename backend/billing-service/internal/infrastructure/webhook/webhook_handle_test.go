package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/payment"
)

// mockPaymentProvider реализует payment.PaymentProvider для тестирования.
type mockPaymentProvider struct {
	verifyErr  error
	parseEvent *payment.WebhookEvent
	parseErr   error
}

func (m *mockPaymentProvider) CreatePayment(_ context.Context, _ *payment.CreatePaymentRequest) (*payment.PaymentResponse, error) {
	return nil, nil
}

func (m *mockPaymentProvider) GetPayment(_ context.Context, _ string) (*payment.PaymentDetails, error) {
	return nil, nil
}

func (m *mockPaymentProvider) RefundPayment(_ context.Context, _ string) (*payment.RefundResponse, error) {
	return nil, nil
}

func (m *mockPaymentProvider) VerifyWebhookSignature(_ context.Context, _ []byte, _ string) error {
	return m.verifyErr
}

func (m *mockPaymentProvider) ParseWebhookEvent(_ context.Context, payload []byte) (*payment.WebhookEvent, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	if m.parseEvent != nil {
		return m.parseEvent, nil
	}
	// Разбираем payload напрямую
	var event payment.WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func TestHandle_unknown_provider(t *testing.T) {
	t.Parallel()

	// Arrange
	h := NewHandler(nil, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Act
	err := h.Handle(context.Background(), "UNKNOWN_PROVIDER", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestHandle_signature_verification_failed(t *testing.T) {
	t.Parallel()

	// Arrange
	provider := &mockPaymentProvider{
		verifyErr: assert.AnError,
	}
	h := NewHandler(provider, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "bad_sig")

	// Assert
	assert.Error(t, err)
}

func TestHandle_parse_event_error(t *testing.T) {
	t.Parallel()

	// Arrange
	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseErr:  assert.AnError,
	}
	h := NewHandler(provider, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

func TestHandle_payment_succeeded(t *testing.T) {
	t.Parallel()

	// Arrange
	providerPaymentID := "yp_handle_success"
	subIDStr := uuid.New().String()
	paymentRepo := newMockPaymentRepository()
	paymentRepo.payments[providerPaymentID] = &Payment{
		ID:                uuid.New().String(),
		UserID:            uuid.New().String(),
		SubscriptionID:    &subIDStr,
		Status:            "PENDING",
		Provider:          "YOOKASSA",
		ProviderPaymentID: providerPaymentID,
	}

	subRepo := newMockSubscriptionRepository()
	subRepo.subscriptions[subIDStr] = &Subscription{
		ID:        subIDStr,
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		Status:    "PENDING",
		ExpiresAt: "2026-04-27T00:00:00Z",
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.succeeded",
			PaymentID: providerPaymentID,
			Status:    "succeeded",
		},
	}

	h := NewHandler(provider, nil, subRepo, paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", paymentRepo.payments[providerPaymentID].Status)
	assert.Equal(t, "ACTIVE", subRepo.subscriptions[subIDStr].Status)
}

func TestHandle_payment_canceled(t *testing.T) {
	t.Parallel()

	// Arrange
	providerPaymentID := "yp_handle_cancel"
	subIDStr := uuid.New().String()
	paymentRepo := newMockPaymentRepository()
	paymentRepo.payments[providerPaymentID] = &Payment{
		ID:                uuid.New().String(),
		UserID:            uuid.New().String(),
		SubscriptionID:    &subIDStr,
		Status:            "PENDING",
		Provider:          "YOOKASSA",
		ProviderPaymentID: providerPaymentID,
	}

	subRepo := newMockSubscriptionRepository()
	subRepo.subscriptions[subIDStr] = &Subscription{
		ID:        subIDStr,
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		Status:    "PENDING",
		ExpiresAt: "2026-04-27T00:00:00Z",
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.canceled",
			PaymentID: providerPaymentID,
			Status:    "canceled",
		},
	}

	h := NewHandler(provider, nil, subRepo, paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "FAILED", paymentRepo.payments[providerPaymentID].Status)
	assert.Equal(t, "CANCELED", subRepo.subscriptions[subIDStr].Status)
}

func TestHandle_refund_succeeded(t *testing.T) {
	t.Parallel()

	// Arrange
	providerPaymentID := "yp_handle_refund"
	subIDStr := uuid.New().String()
	paymentRepo := newMockPaymentRepository()
	paymentRepo.payments[providerPaymentID] = &Payment{
		ID:                uuid.New().String(),
		UserID:            uuid.New().String(),
		SubscriptionID:    &subIDStr,
		Status:            "SUCCESS",
		Provider:          "YOOKASSA",
		ProviderPaymentID: providerPaymentID,
	}

	subRepo := newMockSubscriptionRepository()
	subRepo.subscriptions[subIDStr] = &Subscription{
		ID:        subIDStr,
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		Status:    "ACTIVE",
		ExpiresAt: "2026-04-27T00:00:00Z",
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "refund.succeeded",
			PaymentID: providerPaymentID,
			Status:    "refunded",
		},
	}

	h := NewHandler(provider, nil, subRepo, paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "REFUNDED", paymentRepo.payments[providerPaymentID].Status)
	assert.Equal(t, "CANCELED", subRepo.subscriptions[subIDStr].Status)
}

func TestHandle_unknown_event_type(t *testing.T) {
	t.Parallel()

	// Arrange
	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "some.unknown.event",
			PaymentID: "pay_123",
		},
	}

	h := NewHandler(provider, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert — неизвестный тип не является ошибкой
	assert.NoError(t, err)
}

func TestHandle_payment_not_found(t *testing.T) {
	t.Parallel()

	// Arrange
	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.succeeded",
			PaymentID: "nonexistent_payment",
		},
	}

	// Пустой репозиторий — платёж не найден
	h := NewHandler(provider, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

func TestHandle_payment_already_successful(t *testing.T) {
	t.Parallel()

	// Arrange
	providerPaymentID := "yp_already_ok"
	paymentRepo := newMockPaymentRepository()
	paymentRepo.payments[providerPaymentID] = &Payment{
		ID:                uuid.New().String(),
		UserID:            uuid.New().String(),
		Status:            "SUCCESS", // уже успешен
		Provider:          "YOOKASSA",
		ProviderPaymentID: providerPaymentID,
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.succeeded",
			PaymentID: providerPaymentID,
			Status:    "succeeded",
		},
	}

	h := NewHandler(provider, nil, newMockSubscriptionRepository(), paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert — идемпотентная операция
	assert.NoError(t, err)
}

func TestHandle_stripe_provider(t *testing.T) {
	t.Parallel()

	// Arrange
	providerPaymentID := "pi_stripe_test"
	paymentRepo := newMockPaymentRepository()
	paymentRepo.payments[providerPaymentID] = &Payment{
		ID:                uuid.New().String(),
		UserID:            uuid.New().String(),
		Status:            "PENDING",
		Provider:          "STRIPE",
		ProviderPaymentID: providerPaymentID,
	}

	stripeProvider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment_intent.succeeded",
			PaymentID: providerPaymentID,
			Status:    "succeeded",
		},
	}

	h := NewHandler(nil, stripeProvider, newMockSubscriptionRepository(), paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "STRIPE", []byte(`{}`), "sig")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", paymentRepo.payments[providerPaymentID].Status)
}

func TestNewHandler_nil_providers(t *testing.T) {
	t.Parallel()

	// Arrange + Act
	h := NewHandler(nil, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Assert
	assert.NotNil(t, h)
	assert.Empty(t, h.providers)
}

func TestNewHandler_with_providers(t *testing.T) {
	t.Parallel()

	// Arrange
	yookassa := &mockPaymentProvider{}
	stripe := &mockPaymentProvider{}

	// Act
	h := NewHandler(yookassa, stripe, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Assert
	assert.NotNil(t, h)
	assert.Len(t, h.providers, 2)
	assert.Contains(t, h.providers, "YOOKASSA")
	assert.Contains(t, h.providers, "STRIPE")
}

func TestHandle_payment_failed_update_error(t *testing.T) {
	t.Parallel()

	// Arrange — paymentRepo.Update возвращает ошибку
	providerPaymentID := "yp_failed_update_err"
	paymentRepo := &mockPaymentRepositoryWithErr{
		payments: map[string]*Payment{
			providerPaymentID: {
				ID:                "pay-err-1",
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
			EventType: "payment.canceled",
			PaymentID: providerPaymentID,
		},
	}

	h := NewHandler(provider, nil, newMockSubscriptionRepository(), paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

func TestHandle_refund_succeeded_update_error(t *testing.T) {
	t.Parallel()

	// Arrange — paymentRepo.Update возвращает ошибку
	providerPaymentID := "yp_refund_update_err"
	paymentRepo := &mockPaymentRepositoryWithErr{
		payments: map[string]*Payment{
			providerPaymentID: {
				ID:                "pay-err-2",
				UserID:            "user-1",
				Status:            "SUCCESS",
				Provider:          "YOOKASSA",
				ProviderPaymentID: providerPaymentID,
			},
		},
		updateErr: assert.AnError,
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "refund.succeeded",
			PaymentID: providerPaymentID,
		},
	}

	h := NewHandler(provider, nil, newMockSubscriptionRepository(), paymentRepo, newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

func TestHandle_payment_failed_not_found(t *testing.T) {
	t.Parallel()

	// Arrange — платёж не найден
	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.canceled",
			PaymentID: "nonexistent",
		},
	}

	h := NewHandler(provider, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

func TestHandle_refund_succeeded_not_found(t *testing.T) {
	t.Parallel()

	// Arrange — платёж не найден
	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "refund.succeeded",
			PaymentID: "nonexistent",
		},
	}

	h := NewHandler(provider, nil, newMockSubscriptionRepository(), newMockPaymentRepository(), newTestLogger())

	// Act
	err := h.Handle(context.Background(), "YOOKASSA", []byte(`{}`), "sig")

	// Assert
	assert.Error(t, err)
}

// mockPaymentRepositoryWithErr позволяет задавать ошибку для Update.
type mockPaymentRepositoryWithErr struct {
	payments  map[string]*Payment
	updateErr error
}

func (m *mockPaymentRepositoryWithErr) GetByProviderPaymentID(_ context.Context, providerPaymentID string) (*Payment, error) {
	pmt, ok := m.payments[providerPaymentID]
	if !ok {
		return nil, &NotFoundError{ID: providerPaymentID}
	}
	return pmt, nil
}

func (m *mockPaymentRepositoryWithErr) Update(_ context.Context, pmt *Payment) error {
	_ = pmt
	return m.updateErr
}
