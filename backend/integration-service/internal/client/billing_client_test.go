package client

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	apiv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// mockBillingServer реализует BillingServiceServer для тестов.
type mockBillingServer struct {
	apiv1.UnimplementedBillingServiceServer
	subscription    *apiv1.Subscription
	plans           *apiv1.SubscriptionPlans
	err             error
	subscriptionErr error
	plansErr        error
}

func (m *mockBillingServer) GetSubscription(_ context.Context, req *apiv1.GetSubscriptionRequest) (*apiv1.Subscription, error) {
	if m.subscriptionErr != nil {
		return nil, m.subscriptionErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.subscription, nil
}

func (m *mockBillingServer) GetSubscriptionPlans(_ context.Context, _ *apiv1.Empty) (*apiv1.SubscriptionPlans, error) {
	if m.plansErr != nil {
		return nil, m.plansErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.plans, nil
}

func newTestBillingServer(t *testing.T, srv *mockBillingServer) *BillingClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	apiv1.RegisterBillingServiceServer(s, srv)

	go func() {
		if err := s.Serve(lis); err != nil {
			return
		}
	}()

	t.Cleanup(func() { s.Stop() })

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close gRPC connection: %v", err)
		}
	})

	return NewBillingClient(conn)
}

func TestNewBillingClient(t *testing.T) {
	t.Parallel()

	srv := &mockBillingServer{}
	client := newTestBillingServer(t, srv)
	require.NotNil(t, client)
}

func TestBillingClientGetSubscription(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("returns subscription", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:     uuid.New().String(),
				UserId: userID.String(),
				Status: apiv1.SubscriptionStatus_STATUS_ACTIVE,
				Tier:   apiv1.SubscriptionTier_TIER_FREE,
			},
		}
		client := newTestBillingServer(t, srv)
		sub, err := client.GetSubscription(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, sub)
		assert.Equal(t, apiv1.SubscriptionStatus_STATUS_ACTIVE, sub.Status)
	})

	t.Run("propagates error", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{err: assert.AnError}
		client := newTestBillingServer(t, srv)
		_, err := client.GetSubscription(context.Background(), userID)
		require.Error(t, err)
	})
}

func TestBillingClientGetSubscriptionPlans(t *testing.T) {
	t.Parallel()

	t.Run("returns plans", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			plans: &apiv1.SubscriptionPlans{
				Plans: []*apiv1.SubscriptionPlan{
					{Id: "free", Name: "Free", MaxMonitors: 5},
				},
			},
		}
		client := newTestBillingServer(t, srv)
		plans, err := client.GetSubscriptionPlans(context.Background())
		require.NoError(t, err)
		require.NotNil(t, plans)
		assert.Len(t, plans.Plans, 1)
	})
}

func TestBillingClientCheckMonitorLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("active subscription with matching plan allows creation", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:     uuid.New().String(),
				UserId: userID.String(),
				Status: apiv1.SubscriptionStatus_STATUS_ACTIVE,
				PlanId: "free",
			},
			plans: &apiv1.SubscriptionPlans{
				Plans: []*apiv1.SubscriptionPlan{
					{Id: "free", Name: "Free", MaxMonitors: 10},
				},
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckMonitorLimit(context.Background(), userID, 5)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("inactive subscription blocks creation", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:     uuid.New().String(),
				UserId: userID.String(),
				Status: apiv1.SubscriptionStatus_STATUS_CANCELED,
				PlanId: "free",
			},
			plans: &apiv1.SubscriptionPlans{
				Plans: []*apiv1.SubscriptionPlan{
					{Id: "free", MaxMonitors: 10},
				},
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckMonitorLimit(context.Background(), userID, 0)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("plan not found uses default limit of 10", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:     uuid.New().String(),
				UserId: userID.String(),
				Status: apiv1.SubscriptionStatus_STATUS_ACTIVE,
				PlanId: "unknown-plan",
			},
			plans: &apiv1.SubscriptionPlans{Plans: []*apiv1.SubscriptionPlan{}},
		}
		client := newTestBillingServer(t, srv)
		// With 5 current monitors and default limit 10, should be allowed
		allowed, err := client.CheckMonitorLimit(context.Background(), userID, 5)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("limit exceeded blocks creation", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:     uuid.New().String(),
				UserId: userID.String(),
				Status: apiv1.SubscriptionStatus_STATUS_ACTIVE,
				PlanId: "free",
			},
			plans: &apiv1.SubscriptionPlans{
				Plans: []*apiv1.SubscriptionPlan{
					{Id: "free", MaxMonitors: 3},
				},
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckMonitorLimit(context.Background(), userID, 3)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("subscription error propagates", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{err: assert.AnError}
		client := newTestBillingServer(t, srv)
		_, err := client.CheckMonitorLimit(context.Background(), userID, 0)
		require.Error(t, err)
	})

	t.Run("plans fetch error propagates", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:     uuid.New().String(),
				UserId: userID.String(),
				Status: apiv1.SubscriptionStatus_STATUS_ACTIVE,
				PlanId: "free",
			},
			plansErr: assert.AnError,
		}
		client := newTestBillingServer(t, srv)
		_, err := client.CheckMonitorLimit(context.Background(), userID, 0)
		require.Error(t, err)
	})
}

func TestBillingClientCheckWebhookLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("free tier allows up to 5 webhooks", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:     uuid.New().String(),
				UserId: userID.String(),
				Status: apiv1.SubscriptionStatus_STATUS_ACTIVE,
				Tier:   apiv1.SubscriptionTier_TIER_FREE,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckWebhookLimit(context.Background(), userID, 4)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("free tier blocks at 5 webhooks", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_FREE,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckWebhookLimit(context.Background(), userID, 5)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("starter tier allows 20 webhooks", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_STARTER,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckWebhookLimit(context.Background(), userID, 15)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("professional tier allows 100 webhooks", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_PROFESSIONAL,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckWebhookLimit(context.Background(), userID, 50)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("business tier allows 1000 webhooks", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_BUSINESS,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckWebhookLimit(context.Background(), userID, 500)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("unknown tier uses conservative default of 5", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_UNSPECIFIED,
			},
		}
		client := newTestBillingServer(t, srv)
		// 4 < 5, should be allowed
		allowed, err := client.CheckWebhookLimit(context.Background(), userID, 4)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("subscription error propagates", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{err: assert.AnError}
		client := newTestBillingServer(t, srv)
		_, err := client.CheckWebhookLimit(context.Background(), userID, 0)
		require.Error(t, err)
	})
}

func TestBillingClientCheckAPIKeyLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("free tier allows up to 2 api keys", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_FREE,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckAPIKeyLimit(context.Background(), userID, 1)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("free tier blocks at 2 api keys", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_FREE,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckAPIKeyLimit(context.Background(), userID, 2)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("starter tier allows 10 api keys", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_STARTER,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckAPIKeyLimit(context.Background(), userID, 5)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("professional tier allows 100 api keys", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_PROFESSIONAL,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckAPIKeyLimit(context.Background(), userID, 50)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("business tier allows 1000 api keys", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_BUSINESS,
			},
		}
		client := newTestBillingServer(t, srv)
		allowed, err := client.CheckAPIKeyLimit(context.Background(), userID, 500)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("unknown tier uses conservative default of 2", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{
			subscription: &apiv1.Subscription{
				Id:   uuid.New().String(),
				Tier: apiv1.SubscriptionTier_TIER_UNSPECIFIED,
			},
		}
		client := newTestBillingServer(t, srv)
		// 1 < 2, should be allowed
		allowed, err := client.CheckAPIKeyLimit(context.Background(), userID, 1)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("subscription error propagates", func(t *testing.T) {
		t.Parallel()
		srv := &mockBillingServer{err: assert.AnError}
		client := newTestBillingServer(t, srv)
		_, err := client.CheckAPIKeyLimit(context.Background(), userID, 0)
		require.Error(t, err)
	})
}
