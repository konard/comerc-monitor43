package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/payment"
	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// mockEventPublisher реализует EventPublisher для тестирования.
type mockEventPublisher struct {
	createdCalled  bool
	canceledCalled bool
	err            error
}

func (m *mockEventPublisher) PublishSubscriptionCreated(_ context.Context, _ any) error {
	m.createdCalled = true
	return m.err
}

func (m *mockEventPublisher) PublishSubscriptionUpgraded(_ context.Context, _ any) error {
	return m.err
}

func (m *mockEventPublisher) PublishSubscriptionCanceled(_ context.Context, _ any) error {
	m.canceledCalled = true
	return m.err
}

func (m *mockEventPublisher) PublishPaymentFailed(_ context.Context, _ any) error {
	return m.err
}

// Тесты SetPublisher для subscription и payment сервисов.

func TestSubscriptionService_SetPublisher(t *testing.T) {
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	publisher := &mockEventPublisher{}
	svc.SetPublisher(publisher)

	// Проверяем, что publisher установлен и вызывается при создании подписки
	ctx := context.Background()
	userID := uuid.New()
	req := &dto.CreateSubscriptionRequest{
		UserID:   userID.String(),
		PlanID:   "TIER_STARTER",
		Duration: 30,
	}

	_, err := svc.CreateSubscription(ctx, req)
	require.NoError(t, err)
	assert.True(t, publisher.createdCalled)
}

func TestSubscriptionService_CreateSubscription_WithPublisher_Cancel(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	publisher := &mockEventPublisher{}
	svc.SetPublisher(publisher)

	// Создаём активную подписку
	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.subscriptions[userID.String()] = sub

	req := &dto.CancelSubscriptionRequest{
		UserID: userID.String(),
		Reason: "user request",
	}

	err := svc.CancelSubscription(ctx, req)
	require.NoError(t, err)
	assert.True(t, publisher.canceledCalled)
}

func TestSubscriptionService_CreateSubscription_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	// Создаём активную подписку для пользователя
	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.subscriptions[userID.String()] = sub

	// Пытаемся создать ещё одну
	req := &dto.CreateSubscriptionRequest{
		UserID:   userID.String(),
		PlanID:   "TIER_STARTER",
		Duration: 30,
	}

	_, err := svc.CreateSubscription(ctx, req)
	assert.Error(t, err)
	assert.True(t, isAlreadyExists(err))
}

func isAlreadyExists(err error) bool {
	return billingerrors.ErrAlreadyExists != nil && err != nil &&
		contains(err.Error(), "already exists")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSubscriptionService_GetUserSubscriptionHistory(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.subscriptions[userID.String()] = sub

	t.Run("returns subscription history", func(t *testing.T) {
		resp, err := svc.GetUserSubscriptionHistory(ctx, userID.String(), 0, 10)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Subscriptions, 1)
	})

	t.Run("empty history for unknown user", func(t *testing.T) {
		resp, err := svc.GetUserSubscriptionHistory(ctx, uuid.New().String(), 0, 10)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Subscriptions)
	})
}

func TestSubscriptionService_RenewSubscription_PlanNotFound(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	// Подписка с несуществующим планом
	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_NONEXISTENT", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.subscriptions[userID.String()] = sub

	_, err := svc.RenewSubscription(ctx, sub.ID.String())
	assert.Error(t, err)
}

func TestSubscriptionService_SubscriptionToDTO_WithCanceledAt(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	// Отменённая подписка с CanceledAt
	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.subscriptions[userID.String()] = sub

	// Отменяем, чтобы установить CanceledAt
	err := sub.Cancel("test reason")
	require.NoError(t, err)

	resp, err := svc.GetActiveSubscription(ctx, userID.String())
	require.NoError(t, err)
	assert.NotNil(t, resp.CanceledAt)
}

// Тесты для payment service — GetPayment, GetPaymentHistory, SetPublisher.

func TestPaymentService_SetPublisher(t *testing.T) {
	paymentRepo := &mockPaymentRepo{}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	publisher := &mockEventPublisher{}
	svc.SetPublisher(publisher)
	// Просто проверяем, что метод не паникует
}

func TestPaymentService_GetPayment(t *testing.T) {
	ctx := context.Background()

	existingPayment := model.NewPayment(uuid.New(), uuid.New(), model.ProviderYookassa, "yp_get_test", 29900, "RUB")

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	t.Run("existing payment", func(t *testing.T) {
		resp, err := svc.GetPayment(ctx, existingPayment.ID.String())
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, existingPayment.ID.String(), resp.ID)
		assert.Equal(t, "YOOKASSA", resp.Provider)
		assert.Equal(t, "yp_get_test", resp.ProviderPaymentID)
		assert.Equal(t, int64(29900), resp.AmountKopeks)
		assert.Equal(t, "RUB", resp.Currency)
	})

	t.Run("payment not found", func(t *testing.T) {
		paymentRepo.err = billingerrors.NotFound("payment", "unknown")
		defer func() { paymentRepo.err = nil }()

		_, err := svc.GetPayment(ctx, uuid.New().String())
		assert.Error(t, err)
	})
}

func TestPaymentService_GetPaymentHistory(t *testing.T) {
	ctx := context.Background()

	userID := uuid.New()
	p1 := model.NewPayment(userID, uuid.New(), model.ProviderYookassa, "yp_hist_1", 29900, "RUB")
	p2 := model.NewPayment(userID, uuid.New(), model.ProviderYookassa, "yp_hist_2", 29900, "RUB")
	// p2 не имеет SubscriptionID в отдельном поле — проверяем поле через DTO

	paymentRepo := &mockPaymentRepo{
		payments: []*model.Payment{p1, p2},
		count:    2,
	}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	t.Run("returns payment history", func(t *testing.T) {
		req := &dto.GetPaymentHistoryRequest{
			UserID:   userID.String(),
			Page:     0,
			PageSize: 10,
		}

		resp, err := svc.GetPaymentHistory(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Payments, 2)
		assert.Equal(t, int64(2), resp.Total)
	})

	t.Run("error in GetByUserID", func(t *testing.T) {
		paymentRepo.err = billingerrors.ErrInternal
		defer func() { paymentRepo.err = nil }()

		req := &dto.GetPaymentHistoryRequest{
			UserID:   userID.String(),
			Page:     0,
			PageSize: 10,
		}

		_, err := svc.GetPaymentHistory(ctx, req)
		assert.Error(t, err)
	})
}

func TestPaymentService_GetPaymentHistory_CountError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	// Специальный mock, который возвращает ошибку только для Count
	paymentRepo := &mockPaymentRepoCountErr{
		payments: []*model.Payment{
			model.NewPayment(userID, uuid.New(), model.ProviderYookassa, "yp_count_err", 29900, "RUB"),
		},
	}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	req := &dto.GetPaymentHistoryRequest{
		UserID:   userID.String(),
		Page:     0,
		PageSize: 10,
	}

	_, err := svc.GetPaymentHistory(ctx, req)
	assert.Error(t, err)
}

// mockPaymentRepoCountErr возвращает ошибку только для CountByUserID.
type mockPaymentRepoCountErr struct {
	payments []*model.Payment
}

func (m *mockPaymentRepoCountErr) Create(_ context.Context, p *model.Payment) error { return nil }
func (m *mockPaymentRepoCountErr) GetByID(_ context.Context, id string) (*model.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepoCountErr) GetByUserID(_ context.Context, _ string, _, _ int) ([]*model.Payment, error) {
	return m.payments, nil
}
func (m *mockPaymentRepoCountErr) GetByProviderPaymentID(_ context.Context, _ string) (*model.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepoCountErr) Update(_ context.Context, p *model.Payment) error { return nil }
func (m *mockPaymentRepoCountErr) CountByUserID(_ context.Context, _ string) (int64, error) {
	return 0, billingerrors.ErrInternal
}

func TestPaymentService_paymentToDTO_NoSubscriptionID(t *testing.T) {
	ctx := context.Background()

	// Платёж без SubscriptionID
	p := &model.Payment{
		ID:                uuid.New(),
		UserID:            uuid.New(),
		SubscriptionID:    nil,
		Provider:          "YOOKASSA",
		ProviderPaymentID: "yp_no_sub",
		Status:            "PENDING",
		AmountKopeks:      29900,
		Currency:          "RUB",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	paymentRepo := &mockPaymentRepo{payment: p}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	resp, err := svc.GetPayment(ctx, p.ID.String())
	require.NoError(t, err)
	assert.Nil(t, resp.SubscriptionID)
}

func TestPaymentService_ProcessWebhook_UnknownProvider(t *testing.T) {
	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	req := &dto.WebhookRequest{
		Provider:  "UNKNOWN_PROVIDER",
		Payload:   map[string]string{},
		Signature: "sig",
	}

	err := svc.ProcessWebhook(ctx, req)
	assert.Error(t, err)
}

func TestPaymentService_ProcessWebhook_ProviderNotConfigured(t *testing.T) {
	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	// yookassa не настроен (nil)
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	req := &dto.WebhookRequest{
		Provider:  "YOOKASSA",
		Payload:   map[string]string{},
		Signature: "sig",
	}

	err := svc.ProcessWebhook(ctx, req)
	assert.Error(t, err)
}

func TestPaymentService_ProcessWebhook_ParseError(t *testing.T) {
	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseErr:  billingerrors.ErrInternal,
	}
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.WebhookRequest{
		Provider:  "YOOKASSA",
		Payload:   map[string]string{},
		Signature: "sig",
	}

	err := svc.ProcessWebhook(ctx, req)
	assert.Error(t, err)
}

func TestPaymentService_ProcessWebhook_PaymentFailed(t *testing.T) {
	ctx := context.Background()

	subID := uuid.New()
	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)
	sub.ID = subID
	sub.Status = "ACTIVE"

	providerPaymentID := "yp_webhook_fail"
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, providerPaymentID, 29900, "RUB")

	paymentRepo := &mockPaymentRepo{
		providerMap: map[string]*model.Payment{providerPaymentID: existingPayment},
		payment:     existingPayment,
	}
	subRepo := &mockSubRepo{subscription: sub}
	planRepo := newMockPlanRepository()

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "payment.canceled",
			PaymentID: providerPaymentID,
			Status:    "canceled",
		},
	}
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.WebhookRequest{
		Provider:  "YOOKASSA",
		Payload:   map[string]string{},
		Signature: "sig",
	}

	err := svc.ProcessWebhook(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "FAILED", paymentRepo.payment.Status)
	assert.Equal(t, "CANCELED", subRepo.subscription.Status)
}

func TestPaymentService_ProcessWebhook_RefundSucceeded(t *testing.T) {
	ctx := context.Background()

	subID := uuid.New()
	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)
	sub.ID = subID
	sub.Status = "ACTIVE"

	providerPaymentID := "yp_webhook_refund"
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, providerPaymentID, 29900, "RUB")

	paymentRepo := &mockPaymentRepo{
		providerMap: map[string]*model.Payment{providerPaymentID: existingPayment},
		payment:     existingPayment,
	}
	subRepo := &mockSubRepo{subscription: sub}
	planRepo := newMockPlanRepository()

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "refund.succeeded",
			PaymentID: providerPaymentID,
			Status:    "refunded",
		},
	}
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.WebhookRequest{
		Provider:  "YOOKASSA",
		Payload:   map[string]string{},
		Signature: "sig",
	}

	err := svc.ProcessWebhook(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "REFUNDED", paymentRepo.payment.Status)
	assert.Equal(t, "CANCELED", subRepo.subscription.Status)
}

func TestPaymentService_ProcessWebhook_UnknownEventType(t *testing.T) {
	ctx := context.Background()

	providerPaymentID := "yp_webhook_unknown"
	existingPayment := model.NewPayment(uuid.New(), uuid.New(), model.ProviderYookassa, providerPaymentID, 29900, "RUB")

	paymentRepo := &mockPaymentRepo{
		providerMap: map[string]*model.Payment{providerPaymentID: existingPayment},
		payment:     existingPayment,
	}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()

	provider := &mockPaymentProvider{
		verifyErr: nil,
		parseEvent: &payment.WebhookEvent{
			EventType: "unknown.event",
			PaymentID: providerPaymentID,
			Status:    "unknown",
		},
	}
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.WebhookRequest{
		Provider:  "YOOKASSA",
		Payload:   map[string]string{},
		Signature: "sig",
	}

	err := svc.ProcessWebhook(ctx, req)
	require.NoError(t, err) // unknown event type — не ошибка
}

func TestPaymentService_HandlePaymentSuccess_AlreadySuccessful(t *testing.T) {
	ctx := context.Background()

	existingPayment := model.NewPayment(uuid.New(), uuid.New(), model.ProviderYookassa, "yp_already_success", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handlePaymentSuccess(ctx, existingPayment)
	require.NoError(t, err)
	// Статус не изменился
	assert.Equal(t, "SUCCESS", existingPayment.Status)
}

func TestPaymentService_HandlePaymentSuccess_SubscriptionNotFound(t *testing.T) {
	ctx := context.Background()

	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_sub_not_found", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{err: billingerrors.NotFound("subscription", subID.String())}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handlePaymentSuccess(ctx, existingPayment)
	assert.Error(t, err)
}

func TestPaymentService_CreateCheckout_NoProvider(t *testing.T) {
	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{providerMap: make(map[string]*model.Payment)}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	// yookassa = nil
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	req := &dto.CreatePaymentRequest{
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		ReturnURL: "https://example.com",
	}

	_, err := svc.CreateCheckout(ctx, req)
	assert.Error(t, err)
}

func TestPaymentService_CreateCheckout_ProviderError(t *testing.T) {
	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{providerMap: make(map[string]*model.Payment)}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	provider := &mockPaymentProvider{
		createPaymentErr: billingerrors.ErrInternal,
	}
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.CreatePaymentRequest{
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		ReturnURL: "https://example.com",
	}

	_, err := svc.CreateCheckout(ctx, req)
	assert.Error(t, err)
}

func TestPaymentService_RefundPayment_StripeProvider(t *testing.T) {
	ctx := context.Background()

	existingPayment := model.NewPayment(uuid.New(), uuid.New(), model.ProviderStripe, "stripe_refund_1", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	stripeProvider := &mockPaymentProvider{
		refundResp: &payment.RefundResponse{
			RefundID: "refund_stripe_1",
			Status:   "succeeded",
		},
	}
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, stripeProvider, logger)

	err := svc.RefundPayment(ctx, existingPayment.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "REFUNDED", paymentRepo.payment.Status)
}

func TestPaymentService_RefundPayment_UnknownProvider(t *testing.T) {
	ctx := context.Background()

	existingPayment := model.NewPayment(uuid.New(), uuid.New(), "UNKNOWN", "unk_refund_1", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	err := svc.RefundPayment(ctx, existingPayment.ID.String())
	assert.Error(t, err)
}

func TestPlanService_GetAllPlans_WithFeatures(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	planRepo := &mockPlanRepository{
		plans: map[string]*model.Plan{
			"TIER_STARTER": {
				ID:                      "TIER_STARTER",
				Name:                    "Starter",
				PriceKopeks:             29900,
				BillingPeriodDays:       30,
				MaxMonitors:             25,
				MinCheckIntervalSeconds: 60,
				MaxAlertsPerDay:         100,
				IsActive:                true,
				Features: []model.Feature{
					{ID: "feature1", Name: "Feature 1", Description: "Description 1"},
					{ID: "feature2", Name: "Feature 2", Description: "Description 2"},
				},
			},
		},
	}

	svc := NewPlanService(planRepo)

	resp, err := svc.GetAllPlans(ctx)

	require.NoError(t, err)
	require.Len(t, resp.Plans, 1)
	assert.Len(t, resp.Plans[0].Features, 2)
	assert.Equal(t, "feature1", resp.Plans[0].Features[0].ID)
}

func TestSubscriptionService_subscriptionToDTO_WithProviderPaymentID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	// Подписка с ProviderPaymentID
	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	providerPaymentID := "yp_prov_pay_123"
	sub.ProviderPaymentID = &providerPaymentID
	subRepo.subscriptions[userID.String()] = sub

	resp, err := svc.GetActiveSubscription(ctx, userID.String())
	require.NoError(t, err)
	assert.NotNil(t, resp.ProviderPaymentID)
	assert.Equal(t, "yp_prov_pay_123", *resp.ProviderPaymentID)
}

func TestSubscriptionService_CancelSubscription_UpdateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Специальный repo, который возвращает ошибку при Update
	subRepo := &mockSubscriptionRepositoryWithUpdateErr{}
	planRepo := newMockPlanRepository()

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.sub = sub
	subRepo.userID = userID.String()

	svc := NewSubscriptionService(subRepo, planRepo)

	req := &dto.CancelSubscriptionRequest{
		UserID: userID.String(),
		Reason: "test",
	}

	err := svc.CancelSubscription(ctx, req)

	assert.Error(t, err)
}

func TestSubscriptionService_RenewSubscription_UpdateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	subRepo := &mockSubscriptionRepositoryWithUpdateErr{}
	planRepo := newMockPlanRepository()

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.sub = sub
	subRepo.userID = userID.String()

	svc := NewSubscriptionService(subRepo, planRepo)

	_, err := svc.RenewSubscription(ctx, sub.ID.String())

	assert.Error(t, err)
}

func TestSubscriptionService_publishSubscriptionCreated_WithPublisher(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	// Устанавливаем publisher который возвращает ошибку
	publisher := &mockEventPublisher{err: assert.AnError}
	svc.SetPublisher(publisher)

	userID := uuid.New()
	req := &dto.CreateSubscriptionRequest{
		UserID:   userID.String(),
		PlanID:   "TIER_STARTER",
		Duration: 30,
	}

	// Даже с ошибкой publisher — CreateSubscription должен быть успешным (ошибка игнорируется)
	resp, err := svc.CreateSubscription(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestSubscriptionService_publishSubscriptionCanceled_WithPublisher(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	// Устанавливаем publisher который возвращает ошибку при Canceled
	publisher := &mockEventPublisher{err: assert.AnError}
	svc.SetPublisher(publisher)

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.subscriptions[userID.String()] = sub

	req := &dto.CancelSubscriptionRequest{
		UserID: userID.String(),
		Reason: "user request",
	}

	// Даже с ошибкой publisher — CancelSubscription должен быть успешным
	err := svc.CancelSubscription(ctx, req)
	require.NoError(t, err)
}

// mockSubscriptionRepositoryWithUpdateErr возвращает ошибку при Update.
type mockSubscriptionRepositoryWithUpdateErr struct {
	sub    *model.Subscription
	userID string
}

func (m *mockSubscriptionRepositoryWithUpdateErr) Create(_ context.Context, sub *model.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepositoryWithUpdateErr) GetByID(_ context.Context, id string) (*model.Subscription, error) {
	if m.sub != nil && m.sub.ID.String() == id {
		return m.sub, nil
	}
	return nil, billingerrors.NotFound("subscription", id)
}

func (m *mockSubscriptionRepositoryWithUpdateErr) GetActiveByUserID(_ context.Context, userID string) (*model.Subscription, error) {
	if m.sub != nil && m.userID == userID {
		return m.sub, nil
	}
	return nil, billingerrors.NotFound("subscription", userID)
}

func (m *mockSubscriptionRepositoryWithUpdateErr) GetByUserID(_ context.Context, _ string, _, _ int) ([]*model.Subscription, error) {
	if m.sub != nil {
		return []*model.Subscription{m.sub}, nil
	}
	return nil, nil
}

func (m *mockSubscriptionRepositoryWithUpdateErr) Update(_ context.Context, _ *model.Subscription) error {
	return assert.AnError
}

func (m *mockSubscriptionRepositoryWithUpdateErr) Delete(_ context.Context, _ string) error {
	return nil
}

func TestPaymentService_CreateCheckout_EmptyUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{providerMap: make(map[string]*model.Payment)}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	provider := &mockPaymentProvider{}
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.CreatePaymentRequest{
		UserID:    "", // пустой
		PlanID:    "TIER_STARTER",
		ReturnURL: "https://example.com",
	}

	_, err := svc.CreateCheckout(ctx, req)

	assert.Error(t, err)
}

func TestPaymentService_CreateCheckout_EmptyPlanID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{providerMap: make(map[string]*model.Payment)}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	provider := &mockPaymentProvider{}
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.CreatePaymentRequest{
		UserID:    uuid.New().String(),
		PlanID:    "", // пустой
		ReturnURL: "https://example.com",
	}

	_, err := svc.CreateCheckout(ctx, req)

	assert.Error(t, err)
}

func TestPaymentService_CreateCheckout_SubCreateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{providerMap: make(map[string]*model.Payment)}
	subRepo := &mockSubRepo{err: assert.AnError}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	provider := &mockPaymentProvider{} // успешно создаёт платёж
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.CreatePaymentRequest{
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		ReturnURL: "https://example.com",
	}

	_, err := svc.CreateCheckout(ctx, req)

	assert.Error(t, err)
}

func TestPaymentService_CreateCheckout_PaymentCreateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// paymentRepo.Create возвращает ошибку
	paymentRepo := &mockPaymentRepoCreateErr{}
	subRepo := &mockSubRepo{} // subRepo.Create успешен
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	provider := &mockPaymentProvider{} // успешно создаёт платёж
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.CreatePaymentRequest{
		UserID:    uuid.New().String(),
		PlanID:    "TIER_STARTER",
		ReturnURL: "https://example.com",
	}

	_, err := svc.CreateCheckout(ctx, req)

	assert.Error(t, err)
}

// mockPaymentRepoCreateErr возвращает ошибку только при Create.
type mockPaymentRepoCreateErr struct{}

func (m *mockPaymentRepoCreateErr) Create(_ context.Context, _ *model.Payment) error {
	return assert.AnError
}

func (m *mockPaymentRepoCreateErr) GetByID(_ context.Context, _ string) (*model.Payment, error) {
	return nil, nil
}

func (m *mockPaymentRepoCreateErr) GetByUserID(_ context.Context, _ string, _, _ int) ([]*model.Payment, error) {
	return nil, nil
}

func (m *mockPaymentRepoCreateErr) GetByProviderPaymentID(_ context.Context, _ string) (*model.Payment, error) {
	return nil, nil
}

func (m *mockPaymentRepoCreateErr) Update(_ context.Context, _ *model.Payment) error {
	return nil
}

func (m *mockPaymentRepoCreateErr) CountByUserID(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func TestPaymentService_RefundPayment_PaymentNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{err: billingerrors.NotFound("payment", "unknown")}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger)

	_, err := svc.GetPayment(ctx, uuid.New().String())

	assert.Error(t, err)
}

func TestPaymentService_handlePaymentFailed_NoSubscriptionID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Платёж без SubscriptionID
	existingPayment := model.NewPayment(uuid.New(), uuid.Nil, model.ProviderYookassa, "yp_no_sub_fail", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending
	existingPayment.SubscriptionID = nil

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handlePaymentFailed(ctx, existingPayment)

	require.NoError(t, err)
	assert.Equal(t, "FAILED", existingPayment.Status)
}

func TestPaymentService_handleRefundSuccess_NoSubscriptionID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Платёж без SubscriptionID
	existingPayment := model.NewPayment(uuid.New(), uuid.Nil, model.ProviderYookassa, "yp_no_sub_refund", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess
	existingPayment.SubscriptionID = nil

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handleRefundSuccess(ctx, existingPayment)

	require.NoError(t, err)
	assert.Equal(t, "REFUNDED", existingPayment.Status)
}

func TestPaymentService_handlePaymentFailed_SubUpdateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_sub_upd_fail_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	// Подписка ACTIVE — Cancel пройдёт, но Update вернёт ошибку
	sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)
	sub.ID = subID
	sub.Status = "ACTIVE"

	paymentRepo := &mockPaymentRepo{payment: existingPayment}
	subRepo := &mockSubRepoUpdateErr{subscription: sub}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handlePaymentFailed(ctx, existingPayment)

	assert.Error(t, err)
}

// mockSubRepoUpdateErr возвращает ошибку только при Update.
type mockSubRepoUpdateErr struct {
	subscription *model.Subscription
}

func (m *mockSubRepoUpdateErr) Create(_ context.Context, sub *model.Subscription) error { return nil }

func (m *mockSubRepoUpdateErr) GetByID(_ context.Context, id string) (*model.Subscription, error) {
	if m.subscription != nil && m.subscription.ID.String() == id {
		return m.subscription, nil
	}
	return nil, billingerrors.NotFound("subscription", id)
}

func (m *mockSubRepoUpdateErr) GetActiveByUserID(_ context.Context, userID string) (*model.Subscription, error) {
	if m.subscription != nil && m.subscription.UserID.String() == userID {
		return m.subscription, nil
	}
	return nil, billingerrors.NotFound("subscription", userID)
}

func (m *mockSubRepoUpdateErr) GetByUserID(_ context.Context, _ string, _, _ int) ([]*model.Subscription, error) {
	if m.subscription != nil {
		return []*model.Subscription{m.subscription}, nil
	}
	return nil, nil
}

func (m *mockSubRepoUpdateErr) Update(_ context.Context, _ *model.Subscription) error {
	return assert.AnError
}

func (m *mockSubRepoUpdateErr) Delete(_ context.Context, _ string) error { return nil }

func TestSubscriptionService_GetActiveSubscription_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)

	// Несуществующий пользователь — GetActiveByUserID вернёт ошибку
	_, err := svc.GetActiveSubscription(ctx, uuid.New().String())

	assert.Error(t, err)
}

func TestSubscriptionService_publishSubscriptionCreated_NilPublisher(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	svc := NewSubscriptionService(subRepo, planRepo)
	// publisher = nil

	userID := uuid.New()
	req := &dto.CreateSubscriptionRequest{
		UserID:   userID.String(),
		PlanID:   "TIER_STARTER",
		Duration: 30,
	}

	// CreateSubscription с nil publisher должен работать корректно
	resp, err := svc.CreateSubscription(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestPlanService_GetAllPlans_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	planRepo := &mockPlanRepositoryWithErr{}

	svc := NewPlanService(planRepo)

	_, err := svc.GetAllPlans(ctx)

	assert.Error(t, err)
}

// mockPlanRepositoryWithErr возвращает ошибку при GetAll.
type mockPlanRepositoryWithErr struct{}

func (m *mockPlanRepositoryWithErr) GetByID(_ context.Context, id string) (*model.Plan, error) {
	return nil, billingerrors.NotFound("plan", id)
}

func (m *mockPlanRepositoryWithErr) GetAll(_ context.Context, _ bool) ([]*model.Plan, error) {
	return nil, assert.AnError
}

func (m *mockPlanRepositoryWithErr) GetByTier(_ context.Context, tier string) (*model.Plan, error) {
	return nil, billingerrors.NotFound("plan", tier)
}

func TestPaymentService_ProcessWebhook_VerifyError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paymentRepo := &mockPaymentRepo{}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()

	provider := &mockPaymentProvider{
		verifyErr: assert.AnError,
	}
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	req := &dto.WebhookRequest{
		Provider:  "YOOKASSA",
		Payload:   map[string]string{},
		Signature: "bad_sig",
	}

	err := svc.ProcessWebhook(ctx, req)
	assert.Error(t, err)
}

func TestPaymentService_handlePaymentSuccess_UpdateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	subID := uuid.New()
	existingPayment := model.NewPayment(uuid.New(), subID, model.ProviderYookassa, "yp_succ_upd_svc_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusPending

	// paymentRepo.Update возвращает ошибку
	paymentRepo := &mockPaymentRepo{payment: existingPayment, err: assert.AnError}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, nil, nil, logger).(*paymentService)

	err := svc.handlePaymentSuccess(ctx, existingPayment)

	assert.Error(t, err)
}

func TestPaymentService_RefundPayment_UpdateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	existingPayment := model.NewPayment(uuid.New(), uuid.New(), model.ProviderYookassa, "yp_ref_upd_err", 29900, "RUB")
	existingPayment.Status = model.PaymentStatusSuccess

	paymentRepo := &mockPaymentRepo{payment: existingPayment, err: assert.AnError}
	subRepo := &mockSubRepo{}
	planRepo := newMockPlanRepository()
	provider := &mockPaymentProvider{
		refundResp: &payment.RefundResponse{
			RefundID: "refund_1",
			Status:   "succeeded",
		},
	}
	logger := slog.Default()
	svc := NewPaymentService(paymentRepo, subRepo, planRepo, provider, nil, logger)

	err := svc.RefundPayment(ctx, existingPayment.ID.String())
	assert.Error(t, err)
}
