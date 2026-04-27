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

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// mockMonitorServer реализует MonitorServiceServer для тестов.
type mockMonitorServer struct {
	apiv1.UnimplementedMonitorServiceServer
	createdMonitor *apiv1.Monitor
	monitors       []*apiv1.Monitor
	err            error
}

func (m *mockMonitorServer) CreateMonitor(_ context.Context, _ *apiv1.CreateMonitorRequest) (*apiv1.Monitor, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.createdMonitor, nil
}

func (m *mockMonitorServer) ListMonitors(_ context.Context, _ *apiv1.ListMonitorsRequest) (*apiv1.ListMonitorsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &apiv1.ListMonitorsResponse{Monitors: m.monitors}, nil
}

func newTestMonitorServer(t *testing.T, srv *mockMonitorServer) *MonitorClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	apiv1.RegisterMonitorServiceServer(s, srv)

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

	return NewMonitorClient(conn)
}

func TestNewMonitorClient(t *testing.T) {
	t.Parallel()

	srv := &mockMonitorServer{}
	client := newTestMonitorServer(t, srv)
	require.NotNil(t, client)
}

func TestMonitorClientCreateMonitor(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	monitorID := uuid.New()

	t.Run("creates monitor successfully", func(t *testing.T) {
		t.Parallel()
		srv := &mockMonitorServer{
			createdMonitor: &apiv1.Monitor{
				Id:   monitorID.String(),
				Name: "Test Monitor",
			},
		}
		client := newTestMonitorServer(t, srv)

		interval := 30
		data := &model.MonitorImportData{
			Name:          "Test Monitor",
			URL:           "https://example.com",
			CheckInterval: &interval,
		}

		id, err := client.CreateMonitor(context.Background(), userID, data)
		require.NoError(t, err)
		require.NotNil(t, id)
		assert.Equal(t, monitorID, *id)
	})

	t.Run("with expected status and timeout", func(t *testing.T) {
		t.Parallel()
		srv := &mockMonitorServer{
			createdMonitor: &apiv1.Monitor{
				Id:   monitorID.String(),
				Name: "Status Monitor",
			},
		}
		client := newTestMonitorServer(t, srv)

		status := 201
		timeout := 10
		pattern := "OK"
		data := &model.MonitorImportData{
			Name:            "Status Monitor",
			URL:             "https://example.com",
			ExpectedStatus:  &status,
			Timeout:         &timeout,
			ExpectedPattern: &pattern,
			Headers:         map[string]string{"Authorization": "Bearer token"},
		}

		id, err := client.CreateMonitor(context.Background(), userID, data)
		require.NoError(t, err)
		require.NotNil(t, id)
	})

	t.Run("grpc error propagates", func(t *testing.T) {
		t.Parallel()
		srv := &mockMonitorServer{err: assert.AnError}
		client := newTestMonitorServer(t, srv)

		data := &model.MonitorImportData{
			Name: "Fail Monitor",
			URL:  "https://example.com",
		}

		_, err := client.CreateMonitor(context.Background(), userID, data)
		require.Error(t, err)
	})

	t.Run("invalid monitor id in response returns error", func(t *testing.T) {
		t.Parallel()
		srv := &mockMonitorServer{
			createdMonitor: &apiv1.Monitor{
				Id:   "not-a-uuid",
				Name: "Bad ID",
			},
		}
		client := newTestMonitorServer(t, srv)

		data := &model.MonitorImportData{
			Name: "Bad ID Monitor",
			URL:  "https://example.com",
		}

		_, err := client.CreateMonitor(context.Background(), userID, data)
		require.Error(t, err)
	})
}

func TestMonitorClientGetMonitorByName(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	monitorID := uuid.New()

	t.Run("finds monitor by name", func(t *testing.T) {
		t.Parallel()
		srv := &mockMonitorServer{
			monitors: []*apiv1.Monitor{
				{Id: monitorID.String(), Name: "Target Monitor"},
				{Id: uuid.New().String(), Name: "Other Monitor"},
			},
		}
		client := newTestMonitorServer(t, srv)

		id, err := client.GetMonitorByName(context.Background(), userID, "Target Monitor")
		require.NoError(t, err)
		require.NotNil(t, id)
		assert.Equal(t, monitorID, *id)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		t.Parallel()
		srv := &mockMonitorServer{
			monitors: []*apiv1.Monitor{
				{Id: uuid.New().String(), Name: "Other Monitor"},
			},
		}
		client := newTestMonitorServer(t, srv)

		id, err := client.GetMonitorByName(context.Background(), userID, "NonExistent")
		require.NoError(t, err)
		assert.Nil(t, id)
	})

	t.Run("grpc error propagates", func(t *testing.T) {
		t.Parallel()
		srv := &mockMonitorServer{err: assert.AnError}
		client := newTestMonitorServer(t, srv)

		_, err := client.GetMonitorByName(context.Background(), userID, "Any")
		require.Error(t, err)
	})
}
