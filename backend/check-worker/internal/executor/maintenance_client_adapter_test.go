package executor

import (
	"context"
	"net"
	"testing"
	"time"

	api "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maintenanceBufSize = 1024 * 1024

// fakeMaintenanceServer реализует MaintenanceWindowServiceServer для тестов.
type fakeMaintenanceServer struct {
	api.UnimplementedMaintenanceWindowServiceServer

	listResp *api.ListMaintenanceWindowsResponse
	listErr  error
}

func (s *fakeMaintenanceServer) ListMaintenanceWindows(_ context.Context, _ *api.ListMaintenanceWindowsRequest) (*api.ListMaintenanceWindowsResponse, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp != nil {
		return s.listResp, nil
	}
	return &api.ListMaintenanceWindowsResponse{}, nil
}

// newBufconnMaintenanceClient создаёт GRPCMaintenanceClient, подключённый к in-process серверу.
func newBufconnMaintenanceClient(t *testing.T, srv *fakeMaintenanceServer) (*GRPCMaintenanceClient, func()) {
	t.Helper()

	lis := bufconn.Listen(maintenanceBufSize)
	gs := grpc.NewServer()
	api.RegisterMaintenanceWindowServiceServer(gs, srv)

	go func() {
		if err := gs.Serve(lis); err != nil {
			return
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
	)
	require.NoError(t, err)

	c := NewGRPCMaintenanceClient(GRPCMaintenanceConfig{
		Address: "bufnet",
		Timeout: 5 * time.Second,
	})
	c.conn = conn
	c.client = api.NewMaintenanceWindowServiceClient(conn)

	cleanup := func() {
		gs.Stop()
		assert.NoError(t, lis.Close())
		assert.NoError(t, conn.Close())
	}
	return c, cleanup
}

// --- тесты GRPCMaintenanceClient ---

func TestGRPCMaintenanceClient_GetActiveMaintenanceWindows_EmptyResponse(t *testing.T) {
	t.Parallel()

	srv := &fakeMaintenanceServer{
		listResp: &api.ListMaintenanceWindowsResponse{},
	}
	c, cleanup := newBufconnMaintenanceClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	windows, err := c.GetActiveMaintenanceWindows(ctx)
	require.NoError(t, err)
	assert.Empty(t, windows)
}

func TestGRPCMaintenanceClient_GetActiveMaintenanceWindows_GlobalWindow(t *testing.T) {
	t.Parallel()

	now := time.Now()
	srv := &fakeMaintenanceServer{
		listResp: &api.ListMaintenanceWindowsResponse{
			Windows: []*api.MaintenanceWindow{
				{
					Id:              "global-1",
					IsGlobal:        true,
					PauseMonitoring: true,
					StartTime:       timestamppb.New(now.Add(-1 * time.Hour)),
					EndTime:         timestamppb.New(now.Add(1 * time.Hour)),
					Status:          api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE,
				},
			},
		},
	}
	c, cleanup := newBufconnMaintenanceClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	windows, err := c.GetActiveMaintenanceWindows(ctx)
	require.NoError(t, err)
	require.Len(t, windows, 1)

	w := windows[0]
	assert.Equal(t, "global-1", w.ID)
	assert.True(t, w.IsGlobal)
	assert.True(t, w.PauseMonitoring)
	assert.True(t, w.IsActive())
}

func TestGRPCMaintenanceClient_GetActiveMaintenanceWindows_MultipleMonitorIDs(t *testing.T) {
	t.Parallel()

	now := time.Now()
	srv := &fakeMaintenanceServer{
		listResp: &api.ListMaintenanceWindowsResponse{
			Windows: []*api.MaintenanceWindow{
				{
					Id:              "window-multi",
					IsGlobal:        false,
					PauseMonitoring: true,
					MonitorIds:      []string{"mon-1", "mon-2", "mon-3"},
					StartTime:       timestamppb.New(now.Add(-30 * time.Minute)),
					EndTime:         timestamppb.New(now.Add(30 * time.Minute)),
					Status:          api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE,
				},
			},
		},
	}
	c, cleanup := newBufconnMaintenanceClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	windows, err := c.GetActiveMaintenanceWindows(ctx)
	require.NoError(t, err)

	// Одно proto-окно с тремя monitor_ids → три внутренних записи
	assert.Len(t, windows, 3)
	monitorIDs := make([]string, 0, len(windows))
	for _, w := range windows {
		assert.Equal(t, "window-multi", w.ID)
		assert.False(t, w.IsGlobal)
		assert.True(t, w.IsActive())
		monitorIDs = append(monitorIDs, w.MonitorID)
	}
	assert.ElementsMatch(t, []string{"mon-1", "mon-2", "mon-3"}, monitorIDs)
}

func TestGRPCMaintenanceClient_GetActiveMaintenanceWindows_UnavailableError(t *testing.T) {
	t.Parallel()

	srv := &fakeMaintenanceServer{
		listErr: status.Error(codes.Unavailable, "service unavailable"),
	}
	c, cleanup := newBufconnMaintenanceClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	// При Unavailable клиент должен вернуть пустой список, а не ошибку
	windows, err := c.GetActiveMaintenanceWindows(ctx)
	require.NoError(t, err)
	assert.Empty(t, windows)
}

func TestGRPCMaintenanceClient_GetActiveMaintenanceWindows_InternalError(t *testing.T) {
	t.Parallel()

	srv := &fakeMaintenanceServer{
		listErr: status.Error(codes.Internal, "internal error"),
	}
	c, cleanup := newBufconnMaintenanceClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	windows, err := c.GetActiveMaintenanceWindows(ctx)
	assert.Error(t, err)
	assert.Nil(t, windows)
}

func TestGRPCMaintenanceClient_Close_WithoutConnect(t *testing.T) {
	t.Parallel()

	c := NewGRPCMaintenanceClient(GRPCMaintenanceConfig{Address: "localhost:9999"})
	err := c.Close()
	assert.NoError(t, err)
}

func TestNewGRPCMaintenanceClient_DefaultTimeout(t *testing.T) {
	t.Parallel()

	c := NewGRPCMaintenanceClient(GRPCMaintenanceConfig{Address: "localhost:9999"})
	assert.Equal(t, 10*time.Second, c.cfg.Timeout)
}

// --- тесты convertProtoWindows ---

func TestConvertProtoWindows_NilEntry(t *testing.T) {
	t.Parallel()

	result := convertProtoWindows([]*api.MaintenanceWindow{nil})
	assert.Empty(t, result)
}

func TestConvertProtoWindows_EmptyMonitorIDs_TreatedAsGlobal(t *testing.T) {
	t.Parallel()

	now := time.Now()
	windows := []*api.MaintenanceWindow{
		{
			Id:              "no-monitors",
			IsGlobal:        false,
			MonitorIds:      []string{}, // пустой список → глобальное
			PauseMonitoring: false,
			StartTime:       timestamppb.New(now.Add(-1 * time.Hour)),
			EndTime:         timestamppb.New(now.Add(1 * time.Hour)),
		},
	}

	result := convertProtoWindows(windows)
	require.Len(t, result, 1)
	assert.True(t, result[0].IsGlobal)
}

// --- тесты ConfigMaintenanceClient ---

func TestConfigMaintenanceClient_GetActiveMaintenanceWindows(t *testing.T) {
	t.Parallel()

	now := time.Now()
	windows := []MaintenanceWindow{
		{
			ID:        "active",
			IsGlobal:  true,
			StartTime: now.Add(-1 * time.Hour),
			EndTime:   now.Add(1 * time.Hour),
		},
		{
			ID:        "future",
			IsGlobal:  true,
			StartTime: now.Add(1 * time.Hour),
			EndTime:   now.Add(2 * time.Hour),
		},
	}

	c := NewConfigMaintenanceClient(windows)
	ctx := context.Background()

	active, err := c.GetActiveMaintenanceWindows(ctx)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "active", active[0].ID)
}

// --- тесты ErrorMaintenanceClient ---

func TestErrorMaintenanceClient_ReturnsError(t *testing.T) {
	t.Parallel()

	c := NewErrorMaintenanceClient(assert.AnError)
	ctx := context.Background()

	windows, err := c.GetActiveMaintenanceWindows(ctx)
	assert.Error(t, err)
	assert.Nil(t, windows)
}
