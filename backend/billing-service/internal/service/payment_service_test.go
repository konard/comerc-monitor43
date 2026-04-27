package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/payment"
	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// Mock payment provider для тестирования
type mockPaymentProvider struct {
	createPaymentResp *payment.PaymentResponse
	createPaymentErr  error
	getPaymentResp    *payment.PaymentDetails
	getPaymentErr     error
	refundResp        *payment.RefundResponse
	refundErr         error
	verifyErr         error
	parseEvent        *payment.WebhookEvent
	parseErr          error
}

func (m *mockPaymentProvider) CreatePayment(ctx context.Context, req *payment.CreatePaymentRequest) (*payment.PaymentResponse, error) {
	if m.createPaymentErr != nil {
		return nil, m.createPaymentErr
	}
	if m.createPaymentResp != nil {
		return m.createPaymentResp, nil
	}
	// Default response
	return &payment.PaymentResponse{
		PaymentID:    "test_payment_" + uuid.New().String(),
		CheckoutURL:  "https://test.yookassa.ru/pay/" + uuid.New().String(),
		Status:       "pending",
		AmountKopeks: req.AmountKopeks,
		Currency:     req.Currency,
	}, nil
}

func (m *mockPaymentProvider) GetPayment(ctx context.Context, paymentID string) (*payment.PaymentDetails, error) {
	if m.getPaymentErr != nil {
		return nil, m.getPaymentErr
	}
	if m.getPaymentResp != nil {
		return m.getPaymentResp, nil
	}
	return &payment.PaymentDetails{
		PaymentID: paymentID,
		Status:    "succeeded",
	}, nil
}

func (m *mockPaymentProvider) RefundPayment(ctx context.Context, paymentID string) (*payment.RefundResponse, error) {
	if m.refundErr != nil {
		return nil, m.refundErr
	}
	if m.refundResp != nil {
		return m.refundResp, nil
	}
	return &payment.RefundResponse{
		RefundID: "refund_" + paymentID,
		Status:   "succeeded",
	}, nil
}

func (m *mockPaymentProvider) VerifyWebhookSignature(ctx context.Context, payload []byte, signature string) error {
	return m.verifyErr
}

func (m *mockPaymentProvider) ParseWebhookEvent(ctx context.Context, payload []byte) (*payment.WebhookEvent, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	if m.parseEvent != nil {
		return m.parseEvent, nil
	}
	return &payment.WebhookEvent{
		EventType: "payment.succeeded",
		PaymentID: "test_payment",
		Status:    "succeeded",
	}, nil
}

// Mock repositories для payment service tests
type mockPaymentRepo struct {
	payment     *model.Payment
	payments    []*model.Payment
	count       int64
	err         error
	providerMap map[string]*model.Payment // provider_payment_id -> payment
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *model.Payment) error {
	m.payment = p
	if m.providerMap != nil {
		m.providerMap[p.ProviderPaymentID] = p
	}
	return m.err
}

func (m *mockPaymentRepo) GetByID(ctx context.Context, id string) (*model.Payment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.payment, nil
}

func (m *mockPaymentRepo) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Payment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.payments, nil
}

func (m *mockPaymentRepo) GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*model.Payment, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.providerMap != nil {
		p, ok := m.providerMap[providerPaymentID]
		if !ok {
			return nil, billingerrors.NotFound("payment", providerPaymentID)
		}
		return p, nil
	}
	return nil, billingerrors.NotFound("payment", providerPaymentID)
}

func (m *mockPaymentRepo) Update(ctx context.Context, p *model.Payment) error {
	m.payment = p
	return m.err
}

func (m *mockPaymentRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.count, nil
}

type mockSubRepo struct {
	subscription *model.Subscription
	err          error
}

func (m *mockSubRepo) Create(ctx context.Context, sub *model.Subscription) error {
	m.subscription = sub
	return m.err
}

func (m *mockSubRepo) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.subscription, nil
}

func (m *mockSubRepo) GetActiveByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.subscription, nil
}

func (m *mockSubRepo) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.subscription != nil {
		return []*model.Subscription{m.subscription}, nil
	}
	return nil, nil
}

func (m *mockSubRepo) Update(ctx context.Context, sub *model.Subscription) error {
	m.subscription = sub
	return m.err
}

func (m *mockSubRepo) Delete(ctx context.Context, id string) error {
	return m.err
}

func TestPaymentService_CreateCheckout(t *testing.T) {
	ctx := context.Background()

	planRepo := newMockPlanRepository()
	paymentRepo := &mockPaymentRepo{providerMap: make(map[string]*model.Payment)}
	subRepo := &mockSubRepo{}
	provider := &mockPaymentProvider{
		createPaymentResp: &payment.PaymentResponse{
			PaymentID:   "yp_test_123",
			CheckoutURL: "https://yookassa.ru/pay/123",
			Status:      "pending",
		},
	}

	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	t.Run("successful checkout", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.CreatePaymentRequest{
			UserID:    userID.String(),
			PlanID:    "TIER_STARTER",
			ReturnURL: "https://example.com/return",
		}

		resp, err := svc.CreateCheckout(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.CheckoutURL, "yookassa.ru")
		assert.Equal(t, "yp_test_123", resp.PaymentID)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		req := &dto.CreatePaymentRequest{
			UserID:    "",
			PlanID:    "TIER_STARTER",
			ReturnURL: "https://example.com",
		}

		_, err := svc.CreateCheckout(ctx, req)
		assert.Error(t, err)
	})

	t.Run("plan not found", func(t *testing.T) {
		req := &dto.CreatePaymentRequest{
			UserID:    uuid.New().String(),
			PlanID:    "TIER_NONEXISTENT",
			ReturnURL: "https://example.com",
		}

		_, err := svc.CreateCheckout(ctx, req)
		assert.Error(t, err)
	})
}

func TestPaymentService_ProcessWebhook(t *testing.T) {
	ctx := context.Background()

	planRepo := newMockPlanRepository()
	subID := uuid.New()
	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*1000000000) // 30 days in nanoseconds
	sub.ID = subID
	sub.Status = "PENDING"

	subRepo := &mockSubRepo{
		subscription: sub,
	}

	providerPaymentID := "yp_webhook_123"
	userID := uuid.New()
	existingPayment := model.NewPayment(userID, subID, model.ProviderYookassa, providerPaymentID, 29900, "RUB")

	paymentRepo := &mockPaymentRepo{
		providerMap: map[string]*model.Payment{
			providerPaymentID: existingPayment,
		},
		payment: existingPayment,
	}

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.succeeded",
			PaymentID: providerPaymentID,
			Status:    "succeeded",
		},
	}

	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	t.Run("successful webhook processing", func(t *testing.T) {
		req := &dto.WebhookRequest{
			Provider:  "YOOKASSA",
			Payload:   map[string]string{},
			Signature: "valid_signature",
		}

		err := svc.ProcessWebhook(ctx, req)
		require.NoError(t, err)

		// Verify payment status updated
		assert.Equal(t, "SUCCESS", paymentRepo.payment.Status)
		// Verify subscription activated
		assert.Equal(t, "ACTIVE", subRepo.subscription.Status)
	})

	t.Run("invalid signature", func(t *testing.T) {
		provider.verifyErr = billingerrors.ErrPermissionDenied

		req := &dto.WebhookRequest{
			Provider:  "YOOKASSA",
			Payload:   map[string]string{},
			Signature: "invalid",
		}

		err := svc.ProcessWebhook(ctx, req)
		assert.Error(t, err)
	})

	t.Run("payment not found", func(t *testing.T) {
		provider.verifyErr = nil
		provider.parseEvent = &payment.WebhookEvent{
			EventType: "payment.succeeded",
			PaymentID: "nonexistent_payment",
			Status:    "succeeded",
		}

		req := &dto.WebhookRequest{
			Provider:  "YOOKASSA",
			Payload:   map[string]string{},
			Signature: "valid",
		}

		err := svc.ProcessWebhook(ctx, req)
		assert.Error(t, err)
	})
}

func TestPaymentService_RefundPayment(t *testing.T) {
	ctx := context.Background()

	existingPayment := model.NewPayment(uuid.New(), uuid.New(), model.ProviderYookassa, "yp_refund_123", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	paymentRepo := &mockPaymentRepo{
		payment: existingPayment,
	}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()

	provider := &mockPaymentProvider{
		refundResp: &payment.RefundResponse{
			RefundID: "refund_123",
			Status:   "succeeded",
		},
	}

	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	t.Run("successful refund", func(t *testing.T) {
		err := svc.RefundPayment(ctx, existingPayment.ID.String())
		require.NoError(t, err)

		// Verify payment marked as refunded
		assert.Equal(t, "REFUNDED", paymentRepo.payment.Status)
	})

	t.Run("payment not found", func(t *testing.T) {
		paymentRepo.err = billingerrors.NotFound("payment", "unknown")
		err := svc.RefundPayment(ctx, uuid.New().String())
		assert.Error(t, err)
		paymentRepo.err = nil
	})

	t.Run("invalid payment ID", func(t *testing.T) {
		err := svc.RefundPayment(ctx, "invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("refund failed", func(t *testing.T) {
		provider.refundErr = billingerrors.ErrInternal
		existingPayment.Status = model.PaymentStatusSuccess

		err := svc.RefundPayment(ctx, existingPayment.ID.String())
		assert.Error(t, err)
	})
}

func TestPaymentService_handlePaymentSuccess(t *testing.T) {
	ctx := context.Background()

	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_success_test", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*1000000000)
	sub.ID = subID
	sub.Status = "PENDING"

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{
		subscription: sub,
	}
	planRepo := newMockPlanRepository()

	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handlePaymentSuccess(ctx, existingPayment)
	require.NoError(t, err)

	assert.Equal(t, "SUCCESS", existingPayment.Status)
	assert.Equal(t, "ACTIVE", subRepo.subscription.Status)
}

func TestPaymentService_handlePaymentFailed(t *testing.T) {
	ctx := context.Background()

	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_failed_test", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*1000000000)
	sub.ID = subID
	sub.Status = "ACTIVE"

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{
		subscription: sub,
	}
	planRepo := newMockPlanRepository()

	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handlePaymentFailed(ctx, existingPayment)
	require.NoError(t, err)

	assert.Equal(t, "FAILED", existingPayment.Status)
	assert.Equal(t, "CANCELED", subRepo.subscription.Status)
}

func TestPaymentService_handleRefundSuccess(t *testing.T) {
	ctx := context.Background()

	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_refund_test", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*1000000000)
	sub.ID = subID
	sub.Status = "ACTIVE"

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{
		subscription: sub,
	}
	planRepo := newMockPlanRepository()

	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handleRefundSuccess(ctx, existingPayment)
	require.NoError(t, err)

	assert.Equal(t, "REFUNDED", existingPayment.Status)
	assert.Equal(t, "CANCELED", subRepo.subscription.Status)
}

// Проверка, что errors используется.
var _ = errors.New

func TestPaymentService_handlePaymentFailed_update_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_failed_upd_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	paymentRepo := &mockPaymentRepo{payment: existingPayment, err: assert.AnError}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	// Act
	err := svc.handlePaymentFailed(ctx, existingPayment)

	// Assert
	assert.Error(t, err)
}

func TestPaymentService_handlePaymentFailed_sub_get_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_sub_get_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	// paymentRepo.Update успешен, но subRepo.GetByID возвращает ошибку
	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{err: assert.AnError}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	// Act
	err := svc.handlePaymentFailed(ctx, existingPayment)

	// Assert
	assert.Error(t, err)
}

func TestPaymentService_handlePaymentFailed_cancel_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_cancel_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	// Подписка уже CANCELED — Cancel() вернёт ошибку
	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)
	sub.ID = subID
	sub.Status = "CANCELED"

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{subscription: sub}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	// Act
	err := svc.handlePaymentFailed(ctx, existingPayment)

	// Assert
	assert.Error(t, err)
}

func TestPaymentService_handleRefundSuccess_update_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_refund_upd_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	paymentRepo := &mockPaymentRepo{payment: existingPayment, err: assert.AnError}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	// Act
	err := svc.handleRefundSuccess(ctx, existingPayment)

	// Assert
	assert.Error(t, err)
}

func TestPaymentService_handleRefundSuccess_sub_get_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_refund_sub_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{err: assert.AnError}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	// Act
	err := svc.handleRefundSuccess(ctx, existingPayment)

	// Assert
	assert.Error(t, err)
}

func TestPaymentService_handleRefundSuccess_cancel_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_refund_cancel_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	// Подписка уже CANCELED — Cancel() вернёт ошибку
	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)
	sub.ID = subID
	sub.Status = "CANCELED"

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{subscription: sub}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	// Act
	err := svc.handleRefundSuccess(ctx, existingPayment)

	// Assert
	assert.Error(t, err)
}
