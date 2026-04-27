package handler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	billingv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/billing-service/internal/service"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// Mock services для тестирования
type mockPlanService struct {
	plans *dto.PlansResponse
	err   error
}

func (m *mockPlanService) GetAllPlans(ctx context.Context) (*dto.PlansResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.plans, nil
}

func (m *mockPlanService) GetPlan(ctx context.Context, planID string) (*dto.PlanResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, plan := range m.plans.Plans {
		if plan.ID == planID {
			return plan, nil
		}
	}
	return nil, billingerrors.NotFound("plan", planID)
}

func (m *mockPlanService) ValidatePlanLimits(ctx context.Context, planID string, monitorsCount, checkInterval int) error {
	return m.err
}

type mockSubscriptionService struct {
	subscription *dto.SubscriptionResponse
	err          error
}

func (m *mockSubscriptionService) GetActiveSubscription(ctx context.Context, userID string) (*dto.SubscriptionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.subscription, nil
}

func (m *mockSubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &dto.SubscriptionResponse{
		ID:     uuid.New().String(),
		UserID: req.UserID,
		PlanID: req.PlanID,
		Status: "PENDING",
	}, nil
}

func (m *mockSubscriptionService) CancelSubscription(ctx context.Context, req *dto.CancelSubscriptionRequest) error {
	return m.err
}

func (m *mockSubscriptionService) RenewSubscription(ctx context.Context, subscriptionID string) (*dto.SubscriptionResponse, error) {
	return nil, nil
}

func (m *mockSubscriptionService) GetUserSubscriptionHistory(ctx context.Context, userID string, page, pageSize int32) (*dto.SubscriptionHistoryResponse, error) {
	return nil, nil
}

func (m *mockSubscriptionService) SetPublisher(publisher service.EventPublisher) {}

type mockPaymentService struct {
	checkout *dto.CheckoutResponse
	payment  *dto.PaymentResponse
	history  *dto.PaymentHistoryResponse
	err      error
}

func (m *mockPaymentService) CreateCheckout(ctx context.Context, req *dto.CreatePaymentRequest) (*dto.CheckoutResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.checkout, nil
}

func (m *mockPaymentService) GetPayment(ctx context.Context, paymentID string) (*dto.PaymentResponse, error) {
	return nil, nil
}

func (m *mockPaymentService) GetPaymentHistory(ctx context.Context, req *dto.GetPaymentHistoryRequest) (*dto.PaymentHistoryResponse, error) {
	return nil, nil
}

func (m *mockPaymentService) ProcessWebhook(ctx context.Context, req *dto.WebhookRequest) error {
	return m.err
}

func (m *mockPaymentService) RefundPayment(ctx context.Context, paymentID string) error {
	return m.err
}

func (m *mockPaymentService) SetPublisher(publisher service.EventPublisher) {}

func TestBillingHandler_GetSubscriptionPlans(t *testing.T) {
	ctx := context.Background()

	planService := &mockPlanService{
		plans: &dto.PlansResponse{
			Plans: []*dto.PlanResponse{
				{
					ID:                "TIER_FREE",
					Name:              "Free",
					PriceKopeks:       0,
					BillingPeriodDays: 30,
					MaxMonitors:       5,
					Features:          []*dto.FeatureResponse{},
				},
			},
		},
	}

	handler := NewBillingHandler(planService, nil, nil)

	resp, err := handler.GetSubscriptionPlans(ctx, &billingv1.Empty{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Plans, 1)
	assert.Equal(t, "TIER_FREE", resp.Plans[0].Id)
	assert.Equal(t, "Free", resp.Plans[0].Name)
}

func TestBillingHandler_GetSubscriptionPlans_Error(t *testing.T) {
	ctx := context.Background()

	planService := &mockPlanService{
		err: billingerrors.ErrInternal,
	}

	handler := NewBillingHandler(planService, nil, nil)

	resp, err := handler.GetSubscriptionPlans(ctx, &billingv1.Empty{})
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBillingHandler_GetSubscription(t *testing.T) {
	userID := uuid.New()
	ctx := auth.ContextWithUserID(context.Background(), userID.String())

	subService := &mockSubscriptionService{
		subscription: &dto.SubscriptionResponse{
			ID:     uuid.New().String(),
			UserID: userID.String(),
			PlanID: "TIER_STARTER",
			Status: "ACTIVE",
		},
	}

	handler := NewBillingHandler(nil, subService, nil)

	req := &billingv1.GetSubscriptionRequest{
		UserId: userID.String(),
	}

	resp, err := handler.GetSubscription(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, userID.String(), resp.UserId)
	assert.Equal(t, billingv1.SubscriptionStatus_STATUS_ACTIVE, resp.Status)
}

func TestBillingHandler_GetSubscription_ValidationError(t *testing.T) {
	ctx := context.Background()
	handler := NewBillingHandler(nil, &mockSubscriptionService{}, nil)

	// Empty user ID
	req := &billingv1.GetSubscriptionRequest{
		UserId: "",
	}

	resp, err := handler.GetSubscription(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBillingHandler_GetSubscription_NotFound(t *testing.T) {
	userID := uuid.New()
	ctx := auth.ContextWithUserID(context.Background(), userID.String())

	subService := &mockSubscriptionService{
		err: billingerrors.NotFound("subscription", "123"),
	}

	handler := NewBillingHandler(nil, subService, nil)

	req := &billingv1.GetSubscriptionRequest{
		UserId: userID.String(),
	}

	resp, err := handler.GetSubscription(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBillingHandler_CreateCheckout(t *testing.T) {
	userID := uuid.New()
	ctx := auth.ContextWithUserID(context.Background(), userID.String())

	paymentService := &mockPaymentService{
		checkout: &dto.CheckoutResponse{
			CheckoutURL: "https://yookassa.ru/pay/123",
			PaymentID:   "yp_123456",
		},
	}

	handler := NewBillingHandler(nil, nil, paymentService)

	req := &billingv1.CreateCheckoutRequest{
		UserId:    userID.String(),
		PlanId:    "TIER_STARTER",
		ReturnUrl: "https://example.com/return",
	}

	resp, err := handler.CreateCheckout(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.CheckoutUrl, "yookassa.ru")
}

func TestBillingHandler_CreateCheckout_ValidationError(t *testing.T) {
	handler := NewBillingHandler(nil, nil, &mockPaymentService{})

	t.Run("unauthenticated request", func(t *testing.T) {
		ctx := context.Background() // no auth context
		req := &billingv1.CreateCheckoutRequest{
			UserId:    uuid.New().String(),
			PlanId:    "TIER_STARTER",
			ReturnUrl: "https://example.com",
		}

		resp, err := handler.CreateCheckout(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("empty plan id", func(t *testing.T) {
		userID := uuid.New()
		ctx := auth.ContextWithUserID(context.Background(), userID.String())
		req := &billingv1.CreateCheckoutRequest{
			UserId:    userID.String(),
			PlanId:    "",
			ReturnUrl: "https://example.com",
		}

		resp, err := handler.CreateCheckout(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestBillingHandler_CancelSubscription(t *testing.T) {
	userID := uuid.New()
	ctx := auth.ContextWithUserID(context.Background(), userID.String())

	subService := &mockSubscriptionService{
		err: nil,
	}

	handler := NewBillingHandler(nil, subService, nil)

	req := &billingv1.CancelSubscriptionRequest{
		UserId:       userID.String(),
		CancelReason: "test cancellation",
	}

	resp, err := handler.CancelSubscription(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestBillingHandler_CancelSubscription_ValidationError(t *testing.T) {
	handler := NewBillingHandler(nil, &mockSubscriptionService{}, nil)

	t.Run("unauthenticated request", func(t *testing.T) {
		ctx := context.Background() // no auth context
		req := &billingv1.CancelSubscriptionRequest{
			UserId: uuid.New().String(),
		}

		resp, err := handler.CancelSubscription(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("empty user id", func(t *testing.T) {
		userID := uuid.New()
		ctx := auth.ContextWithUserID(context.Background(), userID.String())
		req := &billingv1.CancelSubscriptionRequest{
			UserId: "", // empty - will cause PermissionDenied since "" != authUserID
		}

		resp, err := handler.CancelSubscription(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})
}

func TestBillingHandler_HandleWebhook(t *testing.T) {
	ctx := context.Background()

	paymentService := &mockPaymentService{
		err: nil,
	}

	handler := NewBillingHandler(nil, nil, paymentService)

	req := &billingv1.WebhookRequest{
		Provider:  "YOOKASSA",
		Payload:   map[string]string{"event": "payment.succeeded"},
		Signature: "valid_signature",
	}

	resp, err := handler.HandleWebhook(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestBillingHandler_HandleWebhook_ValidationError(t *testing.T) {
	ctx := context.Background()
	handler := NewBillingHandler(nil, nil, &mockPaymentService{})

	req := &billingv1.WebhookRequest{
		Provider:  "", // empty
		Payload:   map[string]string{},
		Signature: "",
	}

	resp, err := handler.HandleWebhook(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBillingHandler_statusToProto(t *testing.T) {
	handler := NewBillingHandler(nil, nil, nil)

	tests := []struct {
		name     string
		status   string
		expected billingv1.SubscriptionStatus
	}{
		{"active", "ACTIVE", billingv1.SubscriptionStatus_STATUS_ACTIVE},
		{"pending", "PENDING", billingv1.SubscriptionStatus_STATUS_PENDING},
		{"canceled", "CANCELED", billingv1.SubscriptionStatus_STATUS_CANCELED},
		{"expired", "EXPIRED", billingv1.SubscriptionStatus_STATUS_EXPIRED},
		{"unknown", "UNKNOWN", billingv1.SubscriptionStatus_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.statusToProto(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBillingHandler_providerToProto(t *testing.T) {
	handler := NewBillingHandler(nil, nil, nil)

	tests := []struct {
		name     string
		provider string
		expected billingv1.PaymentProvider
	}{
		{"yookassa", "YOOKASSA", billingv1.PaymentProvider_PROVIDER_YOOKASSA},
		{"stripe", "STRIPE", billingv1.PaymentProvider_PROVIDER_STRIPE},
		{"unknown", "UNKNOWN", billingv1.PaymentProvider_PROVIDER_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.providerToProto(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}
