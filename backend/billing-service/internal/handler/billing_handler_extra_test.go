package handler

import (
	"context"
	"testing"
	"time"

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

// mockPaymentServiceFull полнофункциональный mock для PaymentService.
type mockPaymentServiceFull struct {
	historyResp *dto.PaymentHistoryResponse
	historyErr  error
}

func (m *mockPaymentServiceFull) CreateCheckout(_ context.Context, _ *dto.CreatePaymentRequest) (*dto.CheckoutResponse, error) {
	return nil, nil
}

func (m *mockPaymentServiceFull) GetPayment(_ context.Context, _ string) (*dto.PaymentResponse, error) {
	return nil, nil
}

func (m *mockPaymentServiceFull) GetPaymentHistory(_ context.Context, _ *dto.GetPaymentHistoryRequest) (*dto.PaymentHistoryResponse, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.historyResp, nil
}

func (m *mockPaymentServiceFull) ProcessWebhook(_ context.Context, _ *dto.WebhookRequest) error {
	return nil
}

func (m *mockPaymentServiceFull) RefundPayment(_ context.Context, _ string) error {
	return nil
}

func (m *mockPaymentServiceFull) SetPublisher(_ service.EventPublisher) {}

func TestBillingHandler_GetPaymentHistory(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		// Arrange
		userID := uuid.New()
		ctx := auth.ContextWithUserID(context.Background(), userID.String())

		subID := uuid.New().String()
		paymentService := &mockPaymentServiceFull{
			historyResp: &dto.PaymentHistoryResponse{
				Payments: []*dto.PaymentResponse{
					{
						ID:                uuid.New().String(),
						UserID:            userID.String(),
						SubscriptionID:    &subID,
						Provider:          "YOOKASSA",
						ProviderPaymentID: "yp_hist_1",
						Status:            "SUCCESS",
						AmountKopeks:      29900,
						Currency:          "RUB",
						CreatedAt:         time.Now().Format(time.RFC3339),
						UpdatedAt:         time.Now().Format(time.RFC3339),
					},
				},
				Total:    1,
				Page:     0,
				PageSize: 10,
			},
		}

		handler := NewBillingHandler(nil, nil, paymentService)

		req := &billingv1.GetPaymentHistoryRequest{
			UserId:   userID.String(),
			Page:     0,
			PageSize: 10,
		}

		// Act
		resp, err := handler.GetPaymentHistory(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Payments, 1)
		assert.Equal(t, int32(1), resp.Total)
		assert.Equal(t, "yp_hist_1", resp.Payments[0].ProviderPaymentId)
		assert.Equal(t, billingv1.PaymentStatus_PAYMENT_STATUS_SUCCESS, resp.Payments[0].Status)
		assert.Equal(t, billingv1.PaymentProvider_PROVIDER_YOOKASSA, resp.Payments[0].Provider)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background() // нет auth в контексте
		handler := NewBillingHandler(nil, nil, &mockPaymentServiceFull{})

		req := &billingv1.GetPaymentHistoryRequest{
			UserId:   uuid.New().String(),
			Page:     0,
			PageSize: 10,
		}

		resp, err := handler.GetPaymentHistory(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("permission_denied", func(t *testing.T) {
		t.Parallel()

		// authUserID != req.UserId
		userID := uuid.New()
		ctx := auth.ContextWithUserID(context.Background(), userID.String())
		handler := NewBillingHandler(nil, nil, &mockPaymentServiceFull{})

		req := &billingv1.GetPaymentHistoryRequest{
			UserId:   uuid.New().String(), // другой пользователь
			Page:     0,
			PageSize: 10,
		}

		resp, err := handler.GetPaymentHistory(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("service_error", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		ctx := auth.ContextWithUserID(context.Background(), userID.String())

		// Используем mock, который возвращает ошибку из GetPaymentHistory
		paymentService := &mockPaymentServiceFull{
			historyErr: billingerrors.NotFound("payment", "x"),
		}
		handler := NewBillingHandler(nil, nil, paymentService)

		req := &billingv1.GetPaymentHistoryRequest{
			UserId:   userID.String(),
			Page:     0,
			PageSize: 10,
		}

		resp, err := handler.GetPaymentHistory(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("payment_without_subscription_id", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		ctx := auth.ContextWithUserID(context.Background(), userID.String())

		paymentService := &mockPaymentServiceFull{
			historyResp: &dto.PaymentHistoryResponse{
				Payments: []*dto.PaymentResponse{
					{
						ID:                uuid.New().String(),
						UserID:            userID.String(),
						SubscriptionID:    nil, // нет подписки
						Provider:          "STRIPE",
						ProviderPaymentID: "pi_stripe_1",
						Status:            "PENDING",
						AmountKopeks:      29900,
						Currency:          "RUB",
						CreatedAt:         time.Now().Format(time.RFC3339),
						UpdatedAt:         time.Now().Format(time.RFC3339),
					},
				},
				Total:    1,
				Page:     0,
				PageSize: 10,
			},
		}

		handler := NewBillingHandler(nil, nil, paymentService)

		req := &billingv1.GetPaymentHistoryRequest{
			UserId:   userID.String(),
			Page:     0,
			PageSize: 10,
		}

		resp, err := handler.GetPaymentHistory(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Payments, 1)
		assert.Equal(t, billingv1.PaymentStatus_PAYMENT_STATUS_PENDING, resp.Payments[0].Status)
		assert.Equal(t, billingv1.PaymentProvider_PROVIDER_STRIPE, resp.Payments[0].Provider)
		assert.Empty(t, resp.Payments[0].SubscriptionId)
	})
}

func TestBillingHandler_paymentStatusToProto(t *testing.T) {
	t.Parallel()

	handler := NewBillingHandler(nil, nil, nil)

	tests := []struct {
		name     string
		status   string
		expected billingv1.PaymentStatus
	}{
		{"pending", "PENDING", billingv1.PaymentStatus_PAYMENT_STATUS_PENDING},
		{"success", "SUCCESS", billingv1.PaymentStatus_PAYMENT_STATUS_SUCCESS},
		{"failed", "FAILED", billingv1.PaymentStatus_PAYMENT_STATUS_FAILED},
		{"refunded", "REFUNDED", billingv1.PaymentStatus_PAYMENT_STATUS_REFUNDED},
		{"unknown", "UNKNOWN", billingv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := handler.paymentStatusToProto(tc.status)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBillingHandler_handleError(t *testing.T) {
	t.Parallel()

	handler := NewBillingHandler(nil, nil, nil)

	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{"nil error", nil, codes.OK},
		{"not_found", billingerrors.NotFound("plan", "abc"), codes.NotFound},
		{"already_exists", billingerrors.AlreadyExists("subscription", "xyz"), codes.AlreadyExists},
		{"invalid_argument", billingerrors.InvalidArgument("field", "reason"), codes.InvalidArgument},
		{"permission_denied", billingerrors.ErrPermissionDenied, codes.PermissionDenied},
		{"unauthenticated", billingerrors.ErrUnauthenticated, codes.Unauthenticated},
		{"internal_error", billingerrors.ErrInternal, codes.Internal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := handler.handleError(tc.err)

			if tc.err == nil {
				assert.NoError(t, result)
				return
			}

			require.Error(t, result)
			st, ok := status.FromError(result)
			assert.True(t, ok)
			assert.Equal(t, tc.expectedCode, st.Code())
		})
	}
}

func TestBillingHandler_subscriptionToProto_with_canceled_at(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := NewBillingHandler(nil, nil, nil)
	canceledAt := time.Now().Format(time.RFC3339)
	sub := &dto.SubscriptionResponse{
		ID:         uuid.New().String(),
		UserID:     uuid.New().String(),
		PlanID:     "TIER_STARTER",
		Status:     "CANCELED",
		StartedAt:  time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
		AutoRenew:  false,
		CanceledAt: &canceledAt,
	}

	// Act
	proto := handler.subscriptionToProto(sub)

	// Assert
	assert.NotNil(t, proto.CanceledAt)
	assert.Equal(t, billingv1.SubscriptionStatus_STATUS_CANCELED, proto.Status)
}

func TestBillingHandler_GetSubscription_PermissionDenied(t *testing.T) {
	t.Parallel()

	// Arrange — authUserID != req.UserId
	authUserID := uuid.New()
	ctx := auth.ContextWithUserID(context.Background(), authUserID.String())
	handler := NewBillingHandler(nil, &mockSubscriptionService{}, nil)

	req := &billingv1.GetSubscriptionRequest{
		UserId: uuid.New().String(), // другой пользователь
	}

	// Act
	resp, err := handler.GetSubscription(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestBillingHandler_CreateCheckout_PermissionDenied(t *testing.T) {
	t.Parallel()

	// Arrange — authUserID != req.UserId
	authUserID := uuid.New()
	ctx := auth.ContextWithUserID(context.Background(), authUserID.String())
	handler := NewBillingHandler(nil, nil, &mockPaymentService{})

	req := &billingv1.CreateCheckoutRequest{
		UserId:    uuid.New().String(), // другой пользователь
		PlanId:    "TIER_STARTER",
		ReturnUrl: "https://example.com",
	}

	// Act
	resp, err := handler.CreateCheckout(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}
