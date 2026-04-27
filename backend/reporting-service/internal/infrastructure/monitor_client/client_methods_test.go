package monitor_client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

// mockMonitorServiceClient реализует monitov1.MonitorServiceClient для тестов.
type mockMonitorServiceClient struct {
	mock.Mock
}

func (m *mockMonitorServiceClient) CreateMonitor(ctx context.Context, in *monitov1.CreateMonitorRequest, opts ...grpc.CallOption) (*monitov1.CreateMonitorResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.CreateMonitorResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) GetMonitor(ctx context.Context, in *monitov1.GetMonitorRequest, opts ...grpc.CallOption) (*monitov1.GetMonitorResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetMonitorResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) ListMonitors(ctx context.Context, in *monitov1.ListMonitorsRequest, opts ...grpc.CallOption) (*monitov1.ListMonitorsResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.ListMonitorsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) UpdateMonitor(ctx context.Context, in *monitov1.UpdateMonitorRequest, opts ...grpc.CallOption) (*monitov1.UpdateMonitorResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.UpdateMonitorResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) DeleteMonitor(ctx context.Context, in *monitov1.DeleteMonitorRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*emptypb.Empty), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) PauseMonitor(ctx context.Context, in *monitov1.PauseMonitorRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*emptypb.Empty), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) ResumeMonitor(ctx context.Context, in *monitov1.ResumeMonitorRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*emptypb.Empty), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) GetMonitorHistory(ctx context.Context, in *monitov1.GetMonitorHistoryRequest, opts ...grpc.CallOption) (*monitov1.GetMonitorHistoryResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetMonitorHistoryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) GetUptimeStats(ctx context.Context, in *monitov1.GetUptimeStatsRequest, opts ...grpc.CallOption) (*monitov1.GetUptimeStatsResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetUptimeStatsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) GetIncidents(ctx context.Context, in *monitov1.GetIncidentsRequest, opts ...grpc.CallOption) (*monitov1.GetIncidentsResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetIncidentsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMonitorServiceClient) GetCheckResults(ctx context.Context, in *monitov1.GetCheckResultsRequest, opts ...grpc.CallOption) (*monitov1.GetCheckResultsResponse, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetCheckResultsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

// newTestClient создаёт grpcMonitorClient с подменённым внутренним клиентом.
func newTestClient(inner monitov1.MonitorServiceClient) *grpcMonitorClient {
	return &grpcMonitorClient{
		conn:   nil,
		client: inner,
	}
}

// --- GetUptimeStats ---

func TestGrpcMonitorClient_GetUptimeStats_Success(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	monitorID := "mon-1"
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	want := &monitov1.GetUptimeStatsResponse{Uptime: 99.9, TotalChecks: 100}

	inner.On("GetUptimeStats", mock.Anything, mock.Anything).
		Return(want, nil)

	got, err := c.GetUptimeStats(context.Background(), monitorID, from, to)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetUptimeStats_Error(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetUptimeStats", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("rpc error"))

	got, err := c.GetUptimeStats(context.Background(), "mon-1", time.Now().Add(-time.Hour), time.Now())
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "failed to get uptime stats from monitor-service")
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetUptimeStats_GrpcError(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetUptimeStats", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.Unavailable, "service unavailable"))

	_, err := c.GetUptimeStats(context.Background(), "mon-1", time.Now().Add(-time.Hour), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get uptime stats")
	inner.AssertExpectations(t)
}

// --- GetCheckResults ---

func TestGrpcMonitorClient_GetCheckResults_Success(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	monitorID := "mon-2"
	from := time.Now().Add(-48 * time.Hour)
	to := time.Now()
	want := &monitov1.GetCheckResultsResponse{Total: 5, Limit: 10, Offset: 0}

	inner.On("GetCheckResults", mock.Anything, mock.Anything).
		Return(want, nil)

	got, err := c.GetCheckResults(context.Background(), monitorID, from, to, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetCheckResults_Error(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetCheckResults", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("timeout"))

	got, err := c.GetCheckResults(context.Background(), "mon-2", time.Now().Add(-time.Hour), time.Now(), 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "failed to get check results from monitor-service")
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetCheckResults_GrpcDeadlineExceeded(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetCheckResults", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.DeadlineExceeded, "deadline exceeded"))

	_, err := c.GetCheckResults(context.Background(), "mon-2", time.Now().Add(-time.Hour), time.Now(), 20, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get check results")
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetCheckResults_LimitOffset(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	want := &monitov1.GetCheckResultsResponse{Total: 100}
	inner.On("GetCheckResults", mock.Anything, mock.MatchedBy(func(req *monitov1.GetCheckResultsRequest) bool {
		return req.Limit == 25 && req.Offset == 50
	})).Return(want, nil)

	got, err := c.GetCheckResults(context.Background(), "mon-2", time.Now().Add(-time.Hour), time.Now(), 25, 50)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	inner.AssertExpectations(t)
}

// --- GetIncidents ---

func TestGrpcMonitorClient_GetIncidents_Success(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	monitorID := "mon-3"
	from := time.Now().Add(-72 * time.Hour)
	to := time.Now()
	want := &monitov1.GetIncidentsResponse{Total: 2}

	inner.On("GetIncidents", mock.Anything, mock.Anything).
		Return(want, nil)

	got, err := c.GetIncidents(context.Background(), monitorID, from, to, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetIncidents_Error(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetIncidents", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("connection refused"))

	got, err := c.GetIncidents(context.Background(), "mon-3", time.Now().Add(-time.Hour), time.Now(), 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "failed to get incidents from monitor-service")
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetIncidents_GrpcNotFound(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetIncidents", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "monitor not found"))

	_, err := c.GetIncidents(context.Background(), "mon-3", time.Now().Add(-time.Hour), time.Now(), 5, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get incidents")
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetIncidents_LimitOffset(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	want := &monitov1.GetIncidentsResponse{Total: 50}
	inner.On("GetIncidents", mock.Anything, mock.MatchedBy(func(req *monitov1.GetIncidentsRequest) bool {
		return req.Limit == 15 && req.Offset == 30
	})).Return(want, nil)

	got, err := c.GetIncidents(context.Background(), "mon-3", time.Now().Add(-time.Hour), time.Now(), 15, 30)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	inner.AssertExpectations(t)
}

// --- GetMonitor ---

func TestGrpcMonitorClient_GetMonitor_Success(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	monitorID := "mon-4"
	want := &monitov1.GetMonitorResponse{
		Monitor: &monitov1.Monitor{Id: monitorID, Name: "my-monitor"},
	}

	inner.On("GetMonitor", mock.Anything, mock.Anything).
		Return(want, nil)

	got, err := c.GetMonitor(context.Background(), monitorID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, monitorID, got.Monitor.Id)
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetMonitor_Error(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetMonitor", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("internal error"))

	got, err := c.GetMonitor(context.Background(), "mon-4")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "failed to get monitor from monitor-service")
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetMonitor_GrpcPermissionDenied(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	inner.On("GetMonitor", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

	_, err := c.GetMonitor(context.Background(), "mon-4")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get monitor")
	inner.AssertExpectations(t)
}

func TestGrpcMonitorClient_GetMonitor_MonitorIDPassedThrough(t *testing.T) {
	t.Parallel()

	inner := &mockMonitorServiceClient{}
	c := newTestClient(inner)

	const wantID = "specific-monitor-id-123"
	inner.On("GetMonitor", mock.Anything, mock.MatchedBy(func(req *monitov1.GetMonitorRequest) bool {
		return req.Id == wantID
	})).Return(&monitov1.GetMonitorResponse{}, nil)

	_, err := c.GetMonitor(context.Background(), wantID)
	require.NoError(t, err)
	inner.AssertExpectations(t)
}

// --- Context cancellation via bufconn ---

// fakeMonitorServer — минимальный gRPC сервер для тестов через bufconn.
type fakeMonitorServer struct {
	monitov1.UnimplementedMonitorServiceServer
}

func startFakeServer(t *testing.T) (*grpc.Server, *bufconn.Listener) {
	t.Helper()

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	monitov1.RegisterMonitorServiceServer(srv, &fakeMonitorServer{})

	go func() {
		if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Logf("fake server error: %v", err)
		}
	}()

	t.Cleanup(func() { srv.GracefulStop() })
	return srv, lis
}

func dialBufconn(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := conn.Close(); err != nil && !strings.Contains(err.Error(), "client connection is closing") {
			t.Errorf("close gRPC connection: %v", err)
		}
	})
	return conn
}

func TestGrpcMonitorClient_ContextCancellation_GetUptimeStats(t *testing.T) {
	t.Parallel()

	_, lis := startFakeServer(t)
	conn := dialBufconn(t, lis)

	c := &grpcMonitorClient{
		conn:   conn,
		client: monitov1.NewMonitorServiceClient(conn),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем немедленно

	_, err := c.GetUptimeStats(ctx, "mon-1", time.Now().Add(-time.Hour), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get uptime stats")
}

func TestGrpcMonitorClient_ContextCancellation_GetCheckResults(t *testing.T) {
	t.Parallel()

	_, lis := startFakeServer(t)
	conn := dialBufconn(t, lis)

	c := &grpcMonitorClient{
		conn:   conn,
		client: monitov1.NewMonitorServiceClient(conn),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.GetCheckResults(ctx, "mon-1", time.Now().Add(-time.Hour), time.Now(), 10, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get check results")
}

func TestGrpcMonitorClient_ContextCancellation_GetIncidents(t *testing.T) {
	t.Parallel()

	_, lis := startFakeServer(t)
	conn := dialBufconn(t, lis)

	c := &grpcMonitorClient{
		conn:   conn,
		client: monitov1.NewMonitorServiceClient(conn),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.GetIncidents(ctx, "mon-1", time.Now().Add(-time.Hour), time.Now(), 10, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get incidents")
}

func TestGrpcMonitorClient_ContextCancellation_GetMonitor(t *testing.T) {
	t.Parallel()

	_, lis := startFakeServer(t)
	conn := dialBufconn(t, lis)

	c := &grpcMonitorClient{
		conn:   conn,
		client: monitov1.NewMonitorServiceClient(conn),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.GetMonitor(ctx, "mon-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get monitor")
}

// --- Timeout ---

func TestGrpcMonitorClient_Timeout_GetUptimeStats(t *testing.T) {
	t.Parallel()

	_, lis := startFakeServer(t)
	conn := dialBufconn(t, lis)

	c := &grpcMonitorClient{
		conn:   conn,
		client: monitov1.NewMonitorServiceClient(conn),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // гарантируем истечение таймаута

	_, err := c.GetUptimeStats(ctx, "mon-1", time.Now().Add(-time.Hour), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get uptime stats")
}

// --- Close with real connection ---

func TestGrpcMonitorClient_Close_WithConn(t *testing.T) {
	t.Parallel()

	_, lis := startFakeServer(t)
	conn := dialBufconn(t, lis)

	// создаём клиент вручную, используя соединение от bufconn
	c := &grpcMonitorClient{
		conn:   conn,
		client: monitov1.NewMonitorServiceClient(conn),
	}

	err := c.Close()
	assert.NoError(t, err)
}
