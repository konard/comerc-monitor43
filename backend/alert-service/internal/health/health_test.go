package health

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// mockWatchServer implements grpc_health_v1.Health_WatchServer for testing
type mockWatchServer struct {
	grpc.ServerStream
	ctx      context.Context
	received []*grpc_health_v1.HealthCheckResponse
}

func (m *mockWatchServer) Send(resp *grpc_health_v1.HealthCheckResponse) error {
	m.received = append(m.received, resp)
	return nil
}

func (m *mockWatchServer) Context() context.Context {
	return m.ctx
}

type MockPinger struct {
	shouldPing    bool
	pingError     error
	shouldTimeout bool
	pingDelay     time.Duration
}

func (m *MockPinger) PingContext(ctx context.Context) error {
	if m.pingDelay > 0 {
		time.Sleep(m.pingDelay)
	}

	if m.shouldTimeout {
		return context.DeadlineExceeded
	}

	if m.pingError != nil {
		return m.pingError
	}

	if m.shouldPing {
		return nil
	}
	return sql.ErrConnDone
}

func TestNewHealthChecker(t *testing.T) {
	t.Run("with_mock_database", func(t *testing.T) {
		mockDB := &sql.DB{}
		checker := NewHealthChecker(mockDB)
		require.NotNil(t, checker)
	})
}

func TestNewHealthCheckerFromInterface(t *testing.T) {
	t.Run("with_mock_pinger", func(t *testing.T) {
		mockPinger := &MockPinger{shouldPing: true}
		checker := NewHealthCheckerFromInterface(mockPinger)
		require.NotNil(t, checker)
	})
}

func TestNewHealthCheckerFromPostgres(t *testing.T) {
	t.Run("creates_checker_from_pinger", func(t *testing.T) {
		mockPinger := &MockPinger{shouldPing: true}
		checker := NewHealthCheckerFromPostgres(mockPinger)
		require.NotNil(t, checker)
	})
}

func TestHealthChecker_Check_Successful(t *testing.T) {
	mockPinger := &MockPinger{shouldPing: true}
	checker := NewHealthCheckerFromInterface(mockPinger)

	err := checker.Check(context.Background())
	require.NoError(t, err)
}

func TestHealthChecker_Check_DatabaseFailure(t *testing.T) {
	mockPinger := &MockPinger{shouldPing: false, pingError: sql.ErrConnDone}
	checker := NewHealthCheckerFromInterface(mockPinger)

	err := checker.Check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database unhealthy")

	grpcStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, grpcStatus.Code())
}

func TestHealthChecker_Check_NilDatabase(t *testing.T) {
	checker := NewHealthCheckerFromInterface(nil)

	err := checker.Check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database not configured")

	grpcStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, grpcStatus.Code())
}

func TestHealthChecker_Check_ContextTimeout(t *testing.T) {
	mockPinger := &MockPinger{shouldTimeout: true}
	checker := NewHealthCheckerFromInterface(mockPinger)

	err := checker.Check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database unhealthy")
}

func TestHealthChecker_Check_MultipleErrors(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
	}{
		{"connection_refused", sql.ErrConnDone},
		{"connection_timeout", errors.New("connection timeout")},
		{"general_error", errors.New("database error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPinger := &MockPinger{shouldPing: false, pingError: tt.pingErr}
			checker := NewHealthCheckerFromInterface(mockPinger)

			err := checker.Check(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "database unhealthy")

			grpcStatus, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.Unavailable, grpcStatus.Code())
		})
	}
}

func TestGRPCHealthServer_Check_Serving(t *testing.T) {
	mockPinger := &MockPinger{shouldPing: true}
	checker := NewHealthCheckerFromInterface(mockPinger)
	server := NewGRPCHealthServer(checker)

	resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestGRPCHealthServer_Check_NotServing(t *testing.T) {
	mockPinger := &MockPinger{pingError: errors.New("db down")}
	checker := NewHealthCheckerFromInterface(mockPinger)
	server := NewGRPCHealthServer(checker)

	resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)
}

func TestGRPCHealthServer_Watch_Serving(t *testing.T) {
	mockPinger := &MockPinger{shouldPing: true}
	checker := NewHealthCheckerFromInterface(mockPinger)
	server := NewGRPCHealthServer(checker)

	stream := &mockWatchServer{ctx: context.Background()}
	err := server.Watch(&grpc_health_v1.HealthCheckRequest{}, stream)
	require.NoError(t, err)
	require.Len(t, stream.received, 1)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, stream.received[0].Status)
}

func TestGRPCHealthServer_Watch_NotServing(t *testing.T) {
	mockPinger := &MockPinger{pingError: errors.New("db down")}
	checker := NewHealthCheckerFromInterface(mockPinger)
	server := NewGRPCHealthServer(checker)

	stream := &mockWatchServer{ctx: context.Background()}
	err := server.Watch(&grpc_health_v1.HealthCheckRequest{}, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	require.Len(t, stream.received, 1)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, stream.received[0].Status)
}

func TestNewGRPCHealthServerWithDB(t *testing.T) {
	mockPinger := &MockPinger{shouldPing: true}
	server := NewGRPCHealthServerWithDB(mockPinger)
	require.NotNil(t, server)

	resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	mockPinger := &MockPinger{shouldPing: true}
	checker := NewHealthCheckerFromInterface(mockPinger)

	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			err := checker.Check(ctx)
			assert.NoError(t, err)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
