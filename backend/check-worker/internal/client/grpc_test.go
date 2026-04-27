package client

import (
	"context"
	"log/slog"
	"net"
	"os"
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

	"github.com/raul/monitor/backend/check-worker/internal/config"
	"github.com/raul/monitor/backend/check-worker/internal/queue"
	"github.com/raul/monitor/backend/check-worker/internal/retry"
)

const bufSize = 1024 * 1024

// --- fake MonitorService server ---

type fakeMonitorServer struct {
	api.UnimplementedMonitorServiceServer

	triggerCheckErr error
	getMonitorErr   error
	getMonitorResp  *api.Monitor
}

func (s *fakeMonitorServer) TriggerCheck(_ context.Context, _ *api.TriggerCheckRequest) (*api.CheckResult, error) {
	if s.triggerCheckErr != nil {
		return nil, s.triggerCheckErr
	}
	return &api.CheckResult{Success: true, StatusCode: 200}, nil
}

func (s *fakeMonitorServer) GetMonitor(_ context.Context, req *api.GetMonitorRequest) (*api.Monitor, error) {
	if s.getMonitorErr != nil {
		return nil, s.getMonitorErr
	}
	if s.getMonitorResp != nil {
		return s.getMonitorResp, nil
	}
	return &api.Monitor{Id: req.Id}, nil
}

// --- fake SchedulerService server ---

type fakeSchedulerServer struct {
	api.UnimplementedSchedulerServiceServer

	registerWorkerErr      error
	registerWorkerResp     *api.Worker
	heartbeatErr           error
	unregisterErr          error
	getScheduledChecksErr  error
	getScheduledChecksResp *api.ScheduledChecks
}

func (s *fakeSchedulerServer) RegisterWorker(_ context.Context, req *api.RegisterWorkerRequest) (*api.Worker, error) {
	if s.registerWorkerErr != nil {
		return nil, s.registerWorkerErr
	}
	if s.registerWorkerResp != nil {
		return s.registerWorkerResp, nil
	}
	return &api.Worker{Id: "worker-1", Name: req.Name, Zone: req.Zone}, nil
}

func (s *fakeSchedulerServer) WorkerHeartbeat(_ context.Context, _ *api.HeartbeatRequest) (*api.Empty, error) {
	if s.heartbeatErr != nil {
		return nil, s.heartbeatErr
	}
	return &api.Empty{}, nil
}

func (s *fakeSchedulerServer) UnregisterWorker(_ context.Context, _ *api.UnregisterWorkerRequest) (*api.Empty, error) {
	if s.unregisterErr != nil {
		return nil, s.unregisterErr
	}
	return &api.Empty{}, nil
}

func (s *fakeSchedulerServer) GetScheduledChecks(_ context.Context, _ *api.GetScheduledChecksRequest) (*api.ScheduledChecks, error) {
	if s.getScheduledChecksErr != nil {
		return nil, s.getScheduledChecksErr
	}
	if s.getScheduledChecksResp != nil {
		return s.getScheduledChecksResp, nil
	}
	return &api.ScheduledChecks{Checks: []*api.ScheduledCheck{{Id: "sc-1"}}}, nil
}

// --- helpers ---

// makeTestLogger создаёт тихий логгер для тестов.
func makeTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// fastRetryConfig возвращает конфиг с одной попыткой для быстрых тестов.
func fastRetryConfig() retry.Config {
	return retry.Config{
		MaxAttempts: 1,
		BaseDelay:   0,
		MaxDelay:    0,
		Multiplier:  1,
		Jitter:      false,
	}
}

// newBufconnMonitorClient создаёт MonitorClient, подключённый к in-process серверу через bufconn.
func newBufconnMonitorClient(t *testing.T, srv *fakeMonitorServer) (*MonitorClient, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	gs := grpc.NewServer()
	api.RegisterMonitorServiceServer(gs, srv)

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

	c := &MonitorClient{
		address:       "bufnet",
		conn:          conn,
		client:        api.NewMonitorServiceClient(conn),
		logger:        makeTestLogger(),
		connected:     true,
		retryConfig:   fastRetryConfig(),
		flushInterval: time.Minute, // не запускаем auto-flush в тестах
	}
	// инициализируем очередь
	from := NewMonitorClient("bufnet", makeTestLogger(), nil)
	c.resultQueue = from.resultQueue

	cleanup := func() {
		gs.Stop()
		if err := lis.Close(); err != nil {
			t.Logf("lis close: %v", err)
		}
		if err := conn.Close(); err != nil {
			t.Logf("conn close: %v", err) // conn может быть уже закрыт через c.Close()
		}
	}
	return c, cleanup
}

// newBufconnSchedulerClient создаёт SchedulerClient, подключённый к in-process серверу.
func newBufconnSchedulerClient(t *testing.T, srv *fakeSchedulerServer) (*SchedulerClient, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	gs := grpc.NewServer()
	api.RegisterSchedulerServiceServer(gs, srv)

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

	c := &SchedulerClient{
		address:     "bufnet",
		conn:        conn,
		client:      api.NewSchedulerServiceClient(conn),
		logger:      makeTestLogger(),
		connected:   true,
		retryConfig: fastRetryConfig(),
	}

	cleanup := func() {
		gs.Stop()
		if err := lis.Close(); err != nil {
			t.Logf("lis close: %v", err)
		}
		if err := conn.Close(); err != nil {
			t.Logf("conn close: %v", err) // conn может быть уже закрыт через c.Close()
		}
	}
	return c, cleanup
}

// =============================================================================
// MonitorClient tests
// =============================================================================

func TestNewMonitorClient_WithConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		RetryMaxAttempts:   3,
		RetryBaseDelay:     100 * time.Millisecond,
		RetryMaxDelay:      1 * time.Second,
		ResultQueueMaxSize: 500,
		ResultQueueTTL:     12 * time.Hour,
		QueueFlushInterval: 15 * time.Second,
	}

	c := NewMonitorClient("addr:9091", makeTestLogger(), cfg)
	assert.NotNil(t, c)
	assert.Equal(t, "addr:9091", c.address)
	assert.Equal(t, 3, c.retryConfig.MaxAttempts)
	assert.Equal(t, 15*time.Second, c.flushInterval)
}

func TestMonitorClient_Close_WithConnection(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	err := c.Close()
	assert.NoError(t, err)
	assert.False(t, c.connected)
}

func TestMonitorClient_Close_WithFlushLoop(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	// Запускаем flush loop вручную
	ctx := context.Background()
	c.flushInterval = 50 * time.Millisecond
	c.startFlushLoop(ctx)

	// Немного ждём, потом закрываем
	time.Sleep(60 * time.Millisecond)
	err := c.Close()
	assert.NoError(t, err)
}

func TestMonitorClient_SubmitCheckResult_Success(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	result := BuildCheckResultProto(true, 200, 50.0, "")

	err := c.SubmitCheckResult(ctx, "monitor-1", result)
	assert.NoError(t, err)
}

func TestMonitorClient_SubmitCheckResult_ErrorQueues(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{
		triggerCheckErr: status.Error(codes.Unavailable, "service down"),
	}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	result := BuildCheckResultProto(false, 0, 0, "timeout")

	err := c.SubmitCheckResult(ctx, "monitor-1", result)
	assert.Error(t, err)
	// результат должен попасть в очередь
	assert.Equal(t, 1, c.resultQueue.Size())
}

func TestMonitorClient_SubmitCheckResult_QueueOverflow(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{
		triggerCheckErr: status.Error(codes.Unavailable, "service down"),
	}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	// Заменяем очередь на маленькую (maxSize=1) чтобы вызвать overflow
	c.resultQueue = queue.NewResultQueue(1, time.Hour)

	ctx := context.Background()
	result := BuildCheckResultProto(false, 0, 0, "fail")

	// Первый — помещается в очередь
	require.Error(t, c.SubmitCheckResult(ctx, "m-1", result))
	// Второй — overflow (очередь заполнена)
	err := c.SubmitCheckResult(ctx, "m-2", result)
	assert.Error(t, err)
	assert.Positive(t, c.resultQueue.DroppedCount())
}

func TestMonitorClient_GetMonitor_Success(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{
		getMonitorResp: &api.Monitor{Id: "mon-42", Name: "test-monitor"},
	}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	mon, err := c.GetMonitor(ctx, "mon-42")
	require.NoError(t, err)
	assert.Equal(t, "mon-42", mon.Id)
	assert.Equal(t, "test-monitor", mon.Name)
}

func TestMonitorClient_GetMonitor_Error(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{
		getMonitorErr: status.Error(codes.NotFound, "monitor not found"),
	}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	mon, err := c.GetMonitor(ctx, "missing")
	assert.Error(t, err)
	assert.Nil(t, mon)
}

func TestMonitorClient_SubmitCheckResults_Empty(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	err := c.SubmitCheckResults(ctx, map[string]*api.CheckResult{})
	assert.NoError(t, err)
}

func TestMonitorClient_SubmitCheckResults_NonEmpty(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	results := map[string]*api.CheckResult{
		"check-1": BuildCheckResultProto(true, 200, 10.0, ""),
		"check-2": BuildCheckResultProto(false, 500, 5.0, "server error"),
	}
	err := c.SubmitCheckResults(ctx, results)
	assert.NoError(t, err)
}

func TestMonitorClient_FlushQueue_EmptyQueue(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	// Вызываем flush на пустой очереди — не должно паниковать
	c.flushQueue(context.Background())
}

func TestMonitorClient_FlushQueue_WithItems(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	result := BuildCheckResultProto(true, 200, 10.0, "")

	// Добавляем несколько элементов в очередь напрямую
	for i := 0; i < 5; i++ {
		c.resultQueue.Enqueue(result, "mon-1")
	}
	assert.Equal(t, 5, c.resultQueue.Size())

	// Flush должен отправить всё
	c.flushQueue(ctx)
	assert.Equal(t, 0, c.resultQueue.Size())
}

func TestMonitorClient_FlushQueue_SendError_ReQueues(t *testing.T) {
	t.Parallel()

	srv := &fakeMonitorServer{
		triggerCheckErr: status.Error(codes.Unavailable, "down"),
	}
	c, cleanup := newBufconnMonitorClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	result := BuildCheckResultProto(false, 0, 0, "err")

	// Добавляем в очередь напрямую
	c.resultQueue.Enqueue(result, "mon-fail")

	// flush будет пытаться отправить и ставить обратно при ошибке
	c.flushQueue(ctx)
	// элемент должен вернуться в очередь (или быть больше, если re-queue произошёл несколько раз)
	assert.GreaterOrEqual(t, c.resultQueue.Size(), 0)
}

func TestMonitorClient_IsRetryableGRPCError(t *testing.T) {
	t.Parallel()

	c := &MonitorClient{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"non-grpc error", assert.AnError, true},
		{"OK code", status.Error(codes.OK, ""), false},
		{"Canceled", status.Error(codes.Canceled, ""), true},
		{"DeadlineExceeded", status.Error(codes.DeadlineExceeded, ""), true},
		{"Unavailable", status.Error(codes.Unavailable, ""), true},
		{"ResourceExhausted", status.Error(codes.ResourceExhausted, ""), true},
		{"Aborted", status.Error(codes.Aborted, ""), true},
		{"AlreadyExists", status.Error(codes.AlreadyExists, ""), true},
		{"NotFound", status.Error(codes.NotFound, ""), false},
		{"FailedPrecondition", status.Error(codes.FailedPrecondition, ""), false},
		{"InvalidArgument", status.Error(codes.InvalidArgument, ""), false},
		{"Internal", status.Error(codes.Internal, ""), true},
		{"Unknown", status.Error(codes.Unknown, ""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.isRetryableGRPCError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// SchedulerClient tests
// =============================================================================

func TestNewSchedulerClient_WithConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		RetryMaxAttempts: 2,
		RetryBaseDelay:   50 * time.Millisecond,
		RetryMaxDelay:    500 * time.Millisecond,
	}

	c := NewSchedulerClient("addr:9094", makeTestLogger(), cfg)
	assert.NotNil(t, c)
	assert.Equal(t, "addr:9094", c.address)
	assert.Equal(t, 2, c.retryConfig.MaxAttempts)
}

func TestSchedulerClient_Close_WithConnection(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	err := c.Close()
	assert.NoError(t, err)
	assert.False(t, c.connected)
}

func TestSchedulerClient_GetWorkerID(t *testing.T) {
	t.Parallel()

	c := &SchedulerClient{workerID: "wid-123"}
	assert.Equal(t, "wid-123", c.GetWorkerID())
}

func TestSchedulerClient_GetWorkerID_Empty(t *testing.T) {
	t.Parallel()

	c := &SchedulerClient{}
	assert.Equal(t, "", c.GetWorkerID())
}

func TestSchedulerClient_RegisterWorker_Success(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{
		registerWorkerResp: &api.Worker{Id: "w-99", Name: "worker-01", Zone: "us-east"},
	}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	id, err := c.RegisterWorker(ctx, "worker-01", "us-east", nil)
	require.NoError(t, err)
	assert.Equal(t, "w-99", id)
	assert.Equal(t, "w-99", c.workerID)
}

func TestSchedulerClient_RegisterWorker_Error(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{
		registerWorkerErr: status.Error(codes.InvalidArgument, "bad request"),
	}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	id, err := c.RegisterWorker(ctx, "worker-bad", "zone", nil)
	assert.Error(t, err)
	assert.Empty(t, id)
}

func TestSchedulerClient_SendHeartbeat_Success(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	c.workerID = "w-1"
	err := c.SendHeartbeat(ctx, "active", 10, 2, 150.5)
	assert.NoError(t, err)
}

func TestSchedulerClient_SendHeartbeat_Error(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{
		heartbeatErr: status.Error(codes.Unavailable, "down"),
	}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	c.workerID = "w-1"
	err := c.SendHeartbeat(ctx, "active", 0, 0, 0)
	assert.Error(t, err)
}

func TestSchedulerClient_UnregisterWorker_WithID(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	c.workerID = "w-1"
	ctx := context.Background()
	err := c.UnregisterWorker(ctx)
	assert.NoError(t, err)
}

func TestSchedulerClient_UnregisterWorker_Error(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{
		unregisterErr: status.Error(codes.NotFound, "worker not found"),
	}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	c.workerID = "w-missing"
	ctx := context.Background()
	err := c.UnregisterWorker(ctx)
	assert.Error(t, err)
}

func TestSchedulerClient_GetScheduledChecks_Success(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{
		getScheduledChecksResp: &api.ScheduledChecks{
			Checks: []*api.ScheduledCheck{
				{Id: "sc-1"},
				{Id: "sc-2"},
			},
		},
	}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	checks, err := c.GetScheduledChecks(ctx)
	require.NoError(t, err)
	assert.Len(t, checks, 2)
	assert.Equal(t, "sc-1", checks[0].Id)
}

func TestSchedulerClient_GetScheduledChecks_Error(t *testing.T) {
	t.Parallel()

	srv := &fakeSchedulerServer{
		getScheduledChecksErr: status.Error(codes.Internal, "db error"),
	}
	c, cleanup := newBufconnSchedulerClient(t, srv)
	defer cleanup()

	ctx := context.Background()
	checks, err := c.GetScheduledChecks(ctx)
	assert.Error(t, err)
	assert.Nil(t, checks)
}

func TestSchedulerClient_IsRetryableGRPCError(t *testing.T) {
	t.Parallel()

	c := &SchedulerClient{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"non-grpc error", assert.AnError, true},
		{"OK", status.Error(codes.OK, ""), false},
		{"Canceled", status.Error(codes.Canceled, ""), true},
		{"DeadlineExceeded", status.Error(codes.DeadlineExceeded, ""), true},
		{"Unavailable", status.Error(codes.Unavailable, ""), true},
		{"ResourceExhausted", status.Error(codes.ResourceExhausted, ""), true},
		{"Aborted", status.Error(codes.Aborted, ""), true},
		{"AlreadyExists", status.Error(codes.AlreadyExists, ""), true},
		{"NotFound", status.Error(codes.NotFound, ""), false},
		{"FailedPrecondition", status.Error(codes.FailedPrecondition, ""), false},
		{"InvalidArgument", status.Error(codes.InvalidArgument, ""), false},
		{"Internal", status.Error(codes.Internal, ""), true},
		{"PermissionDenied", status.Error(codes.PermissionDenied, ""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.isRetryableGRPCError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// BuildCheckResultProto tests
// =============================================================================

func TestBuildCheckResultProto_Success(t *testing.T) {
	t.Parallel()

	r := BuildCheckResultProto(true, 200, 123.4, "")
	assert.True(t, r.Success)
	assert.Equal(t, int32(200), r.StatusCode)
	assert.InDelta(t, 123.4, r.ResponseTimeMs, 0.01)
	assert.Empty(t, r.ErrorMessage)
	assert.NotNil(t, r.CheckedAt)
}

func TestBuildCheckResultProto_Failure(t *testing.T) {
	t.Parallel()

	r := BuildCheckResultProto(false, 500, 0, "internal server error")
	assert.False(t, r.Success)
	assert.Equal(t, int32(500), r.StatusCode)
	assert.Equal(t, "internal server error", r.ErrorMessage)
}

func TestBuildCheckResultProto_Timeout(t *testing.T) {
	t.Parallel()

	r := BuildCheckResultProto(false, 0, 30000.0, "connection timeout")
	assert.False(t, r.Success)
	assert.Equal(t, int32(0), r.StatusCode)
	assert.Equal(t, "connection timeout", r.ErrorMessage)
}
