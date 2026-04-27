package monitor

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/scheduler-service/internal/circuitbreaker"
)

// mockMonitorServiceClient — мок gRPC-клиента для тестов.
type mockMonitorServiceClient struct {
	mock.Mock
}

func (m *mockMonitorServiceClient) CreateMonitor(ctx context.Context, in *monitov1.CreateMonitorRequest, opts ...grpc.CallOption) (*monitov1.CreateMonitorResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.CreateMonitorResponse), args.Error(1)
}

func (m *mockMonitorServiceClient) GetMonitor(ctx context.Context, in *monitov1.GetMonitorRequest, opts ...grpc.CallOption) (*monitov1.GetMonitorResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.GetMonitorResponse), args.Error(1)
}

func (m *mockMonitorServiceClient) ListMonitors(ctx context.Context, in *monitov1.ListMonitorsRequest, opts ...grpc.CallOption) (*monitov1.ListMonitorsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.ListMonitorsResponse), args.Error(1)
}

func (m *mockMonitorServiceClient) UpdateMonitor(ctx context.Context, in *monitov1.UpdateMonitorRequest, opts ...grpc.CallOption) (*monitov1.UpdateMonitorResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.UpdateMonitorResponse), args.Error(1)
}

func (m *mockMonitorServiceClient) DeleteMonitor(ctx context.Context, in *monitov1.DeleteMonitorRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *mockMonitorServiceClient) PauseMonitor(ctx context.Context, in *monitov1.PauseMonitorRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *mockMonitorServiceClient) ResumeMonitor(ctx context.Context, in *monitov1.ResumeMonitorRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *mockMonitorServiceClient) GetMonitorHistory(ctx context.Context, in *monitov1.GetMonitorHistoryRequest, opts ...grpc.CallOption) (*monitov1.GetMonitorHistoryResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.GetMonitorHistoryResponse), args.Error(1)
}

func (m *mockMonitorServiceClient) GetUptimeStats(ctx context.Context, in *monitov1.GetUptimeStatsRequest, opts ...grpc.CallOption) (*monitov1.GetUptimeStatsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.GetUptimeStatsResponse), args.Error(1)
}

func (m *mockMonitorServiceClient) GetIncidents(ctx context.Context, in *monitov1.GetIncidentsRequest, opts ...grpc.CallOption) (*monitov1.GetIncidentsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.GetIncidentsResponse), args.Error(1)
}

func (m *mockMonitorServiceClient) GetCheckResults(ctx context.Context, in *monitov1.GetCheckResultsRequest, opts ...grpc.CallOption) (*monitov1.GetCheckResultsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitov1.GetCheckResultsResponse), args.Error(1)
}

// newTestClient создаёт Client с инжектированным моком gRPC-клиента.
func newTestClient(t *testing.T) (*Client, *mockMonitorServiceClient) {
	t.Helper()
	grpcMock := &mockMonitorServiceClient{}
	c := &Client{
		address:   "localhost:50051",
		client:    grpcMock,
		logger:    slog.Default(),
		connected: true,
	}
	return c, grpcMock
}

// newTestAdapter создаёт Adapter с инжектированным моком.
func newTestAdapter(t *testing.T) (*Adapter, *mockMonitorServiceClient) {
	t.Helper()
	c, grpcMock := newTestClient(t)
	a := NewAdapter(c, slog.Default())
	return a, grpcMock
}

// ---------------------------------------------------------------------------
// NewClient / IsConnected / Close
// ---------------------------------------------------------------------------

func TestNewClient(t *testing.T) {
	t.Parallel()

	c := NewClient("localhost:50051", slog.Default(), "test-secret")

	assert.Equal(t, "localhost:50051", c.address)
	assert.False(t, c.IsConnected())
}

func TestClient_Close_not_connected(t *testing.T) {
	t.Parallel()

	c := NewClient("localhost:50051", slog.Default(), "test-secret")

	err := c.Close()

	require.NoError(t, err)
}

func TestClient_Close_marks_disconnected(t *testing.T) {
	t.Parallel()

	c, _ := newTestClient(t)
	require.True(t, c.IsConnected())

	err := c.Close()

	require.NoError(t, err)
	assert.False(t, c.IsConnected())
}

// ---------------------------------------------------------------------------
// Client.IsRetryableError
// ---------------------------------------------------------------------------

func TestClient_IsRetryableError(t *testing.T) {
	t.Parallel()

	c := NewClient("localhost:50051", slog.Default(), "test-secret")

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil_error", nil, false},
		{"deadline_exceeded", status.Error(codes.DeadlineExceeded, "deadline"), true},
		{"unavailable", status.Error(codes.Unavailable, "unavailable"), true},
		{"canceled", status.Error(codes.Canceled, "canceled"), true},
		{"resource_exhausted", status.Error(codes.ResourceExhausted, "exhausted"), true},
		{"not_found", status.Error(codes.NotFound, "not found"), false},
		{"invalid_argument", status.Error(codes.InvalidArgument, "invalid"), false},
		{"failed_precondition", status.Error(codes.FailedPrecondition, "precond"), false},
		{"ok", status.Error(codes.OK, ""), false},
		{"non_grpc_error", errors.New("plain error"), true},
		{"internal_error", status.Error(codes.Internal, "internal"), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, c.IsRetryableError(tc.err))
		})
	}
}

// ---------------------------------------------------------------------------
// Client.GetMonitor
// ---------------------------------------------------------------------------

func TestClient_GetMonitor(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	resp := &monitov1.GetMonitorResponse{
		Monitor: &monitov1.Monitor{
			Id:     "monitor-01",
			Name:   "Test Monitor",
			Status: "ACTIVE",
		},
	}
	grpcMock.On("GetMonitor", mock.Anything, &monitov1.GetMonitorRequest{Id: "monitor-01"}).Return(resp, nil)

	result, err := c.GetMonitor(context.Background(), "monitor-01")

	require.NoError(t, err)
	assert.Equal(t, "monitor-01", result.Id)
}

func TestClient_GetMonitor_error(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	_, err := c.GetMonitor(context.Background(), "monitor-01")

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Client.ListActiveMonitors
// ---------------------------------------------------------------------------

func TestClient_ListActiveMonitors_single_page(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	monitors := []*monitov1.Monitor{
		{Id: "mon-01", Status: "ACTIVE"},
		{Id: "mon-02", Status: "ACTIVE"},
	}
	resp := &monitov1.ListMonitorsResponse{Monitors: monitors, Total: 2}
	grpcMock.On("ListMonitors", mock.Anything, mock.Anything).Return(resp, nil)

	result, err := c.ListActiveMonitors(context.Background(), 100)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestClient_ListActiveMonitors_multi_page(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	page1 := &monitov1.ListMonitorsResponse{
		Monitors: []*monitov1.Monitor{{Id: "mon-01"}},
		Total:    2,
	}
	page2 := &monitov1.ListMonitorsResponse{
		Monitors: []*monitov1.Monitor{{Id: "mon-02"}},
		Total:    2,
	}
	grpcMock.On("ListMonitors", mock.Anything, mock.MatchedBy(func(req *monitov1.ListMonitorsRequest) bool {
		return req.Offset == 0
	})).Return(page1, nil)
	grpcMock.On("ListMonitors", mock.Anything, mock.MatchedBy(func(req *monitov1.ListMonitorsRequest) bool {
		return req.Offset == 1
	})).Return(page2, nil)

	result, err := c.ListActiveMonitors(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestClient_ListActiveMonitors_error(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	grpcMock.On("ListMonitors", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	_, err := c.ListActiveMonitors(context.Background(), 100)

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Client.IsMonitorPaused
// ---------------------------------------------------------------------------

func TestClient_IsMonitorPaused_true(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	resp := &monitov1.GetMonitorResponse{Monitor: &monitov1.Monitor{Id: "mon-01", Status: "PAUSED"}}
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(resp, nil)

	paused, err := c.IsMonitorPaused(context.Background(), "mon-01")

	require.NoError(t, err)
	assert.True(t, paused)
}

func TestClient_IsMonitorPaused_false(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	resp := &monitov1.GetMonitorResponse{Monitor: &monitov1.Monitor{Id: "mon-01", Status: "ACTIVE"}}
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(resp, nil)

	paused, err := c.IsMonitorPaused(context.Background(), "mon-01")

	require.NoError(t, err)
	assert.False(t, paused)
}

func TestClient_IsMonitorPaused_not_found(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(nil, status.Error(codes.NotFound, "not found"))

	_, err := c.IsMonitorPaused(context.Background(), "mon-01")

	assert.Error(t, err)
}

func TestClient_IsMonitorPaused_grpc_error(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "internal"))

	_, err := c.IsMonitorPaused(context.Background(), "mon-01")

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Adapter.GetMonitor
// ---------------------------------------------------------------------------

func TestAdapter_GetMonitor(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestAdapter(t)
	ts := time.Now()
	resp := &monitov1.GetMonitorResponse{
		Monitor: &monitov1.Monitor{
			Id:              "mon-01",
			Name:            "Test",
			Status:          "ACTIVE",
			IntervalSeconds: 60,
			LastCheckAt:     timestamppb.New(ts),
		},
	}
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(resp, nil)

	result, err := a.GetMonitor(context.Background(), "mon-01")

	require.NoError(t, err)
	assert.Equal(t, "mon-01", result.ID)
	assert.Equal(t, "Test", result.Name)
	assert.Equal(t, int32(60), result.IntervalSeconds)
	assert.False(t, result.LastCheckedAt.IsZero())
}

func TestAdapter_GetMonitor_error(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestAdapter(t)
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	_, err := a.GetMonitor(context.Background(), "mon-01")

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Adapter.ListActiveMonitors
// ---------------------------------------------------------------------------

func TestAdapter_ListActiveMonitors(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestAdapter(t)
	monitors := []*monitov1.Monitor{
		{Id: "mon-01", Status: "ACTIVE"},
	}
	grpcMock.On("ListMonitors", mock.Anything, mock.Anything).Return(&monitov1.ListMonitorsResponse{Monitors: monitors, Total: 1}, nil)

	result, err := a.ListActiveMonitors(context.Background(), 100)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "mon-01", result[0].ID)
}

func TestAdapter_ListActiveMonitors_error(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestAdapter(t)
	grpcMock.On("ListMonitors", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	_, err := a.ListActiveMonitors(context.Background(), 100)

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Adapter.IsMonitorPaused
// ---------------------------------------------------------------------------

func TestAdapter_IsMonitorPaused(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestAdapter(t)
	resp := &monitov1.GetMonitorResponse{Monitor: &monitov1.Monitor{Id: "mon-01", Status: "PAUSED"}}
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(resp, nil)

	paused, err := a.IsMonitorPaused(context.Background(), "mon-01")

	require.NoError(t, err)
	assert.True(t, paused)
}

// ---------------------------------------------------------------------------
// toMonitorInfo — nil LastCheckAt
// ---------------------------------------------------------------------------

func TestToMonitorInfo_no_last_checked(t *testing.T) {
	t.Parallel()

	mon := &monitov1.Monitor{
		Id:              "mon-01",
		Name:            "Test",
		Status:          "ACTIVE",
		IntervalSeconds: 30,
		LastCheckAt:     nil,
	}

	info := toMonitorInfo(mon)

	assert.Equal(t, "mon-01", info.ID)
	assert.True(t, info.LastCheckedAt.IsZero())
}

// ---------------------------------------------------------------------------
// CircuitBreakerAdapter
// ---------------------------------------------------------------------------

func newTestCBAdapter(t *testing.T) (*CircuitBreakerAdapter, *mockMonitorServiceClient) {
	t.Helper()
	c, grpcMock := newTestClient(t)
	inner := NewAdapter(c, slog.Default())
	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
		SuccessThreshold: 1,
	})
	a := NewCircuitBreakerAdapter(inner, cb, slog.Default())
	return a, grpcMock
}

func TestCircuitBreakerAdapter_GetMonitor_success(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestCBAdapter(t)
	resp := &monitov1.GetMonitorResponse{Monitor: &monitov1.Monitor{Id: "mon-01", Status: "ACTIVE"}}
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(resp, nil)

	result, err := a.GetMonitor(context.Background(), "mon-01")

	require.NoError(t, err)
	assert.Equal(t, "mon-01", result.ID)
}

func TestCircuitBreakerAdapter_GetMonitor_error(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestCBAdapter(t)
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	_, err := a.GetMonitor(context.Background(), "mon-01")

	assert.Error(t, err)
}

func TestCircuitBreakerAdapter_GetMonitor_circuit_open(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	inner := NewAdapter(c, slog.Default())
	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 2,
		Timeout:          10 * time.Second,
		SuccessThreshold: 1,
	})
	a := NewCircuitBreakerAdapter(inner, cb, slog.Default())

	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	// открываем circuit breaker двумя ошибками
	for i := 0; i < 2; i++ {
		_, err := a.GetMonitor(context.Background(), "mon-01")
		assert.Error(t, err)
	}

	// теперь circuit открыт — следующий вызов должен вернуть ошибку о CB
	_, err := a.GetMonitor(context.Background(), "mon-01")

	assert.Error(t, err)
	assert.True(t, circuitbreaker.IsErrCircuitOpen(err))
}

func TestCircuitBreakerAdapter_ListActiveMonitors_success(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestCBAdapter(t)
	monitors := []*monitov1.Monitor{{Id: "mon-01"}}
	grpcMock.On("ListMonitors", mock.Anything, mock.Anything).Return(&monitov1.ListMonitorsResponse{Monitors: monitors, Total: 1}, nil)

	result, err := a.ListActiveMonitors(context.Background(), 100)

	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCircuitBreakerAdapter_ListActiveMonitors_circuit_open(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	inner := NewAdapter(c, slog.Default())
	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 2,
		Timeout:          10 * time.Second,
		SuccessThreshold: 1,
	})
	a := NewCircuitBreakerAdapter(inner, cb, slog.Default())

	grpcMock.On("ListMonitors", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	for i := 0; i < 2; i++ {
		_, err := a.ListActiveMonitors(context.Background(), 100)
		assert.Error(t, err)
	}

	_, err := a.ListActiveMonitors(context.Background(), 100)

	assert.Error(t, err)
	assert.True(t, circuitbreaker.IsErrCircuitOpen(err))
}

func TestCircuitBreakerAdapter_IsMonitorPaused_success(t *testing.T) {
	t.Parallel()

	a, grpcMock := newTestCBAdapter(t)
	resp := &monitov1.GetMonitorResponse{Monitor: &monitov1.Monitor{Id: "mon-01", Status: "PAUSED"}}
	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(resp, nil)

	paused, err := a.IsMonitorPaused(context.Background(), "mon-01")

	require.NoError(t, err)
	assert.True(t, paused)
}

func TestCircuitBreakerAdapter_IsMonitorPaused_circuit_open(t *testing.T) {
	t.Parallel()

	c, grpcMock := newTestClient(t)
	inner := NewAdapter(c, slog.Default())
	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 2,
		Timeout:          10 * time.Second,
		SuccessThreshold: 1,
	})
	a := NewCircuitBreakerAdapter(inner, cb, slog.Default())

	grpcMock.On("GetMonitor", mock.Anything, mock.Anything).Return(nil, errors.New("grpc error"))

	for i := 0; i < 2; i++ {
		_, err := a.IsMonitorPaused(context.Background(), "mon-01")
		assert.Error(t, err)
	}

	_, err := a.IsMonitorPaused(context.Background(), "mon-01")

	assert.Error(t, err)
	assert.True(t, circuitbreaker.IsErrCircuitOpen(err))
}
