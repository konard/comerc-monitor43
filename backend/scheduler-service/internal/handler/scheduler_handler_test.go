package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	api "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

type mockWorkerService struct {
	mock.Mock
}

func (m *mockWorkerService) RegisterWorker(ctx context.Context, name, zone string, metadata map[string]string) (*model.Worker, error) {
	args := m.Called(ctx, name, zone, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Worker), args.Error(1)
}

func (m *mockWorkerService) Heartbeat(ctx context.Context, workerID uuid.UUID, s model.WorkerStatus, checksCompleted, checksFailed int, avgDurationMs float64) error {
	args := m.Called(ctx, workerID, s, checksCompleted, checksFailed, avgDurationMs)
	return args.Error(0)
}

func (m *mockWorkerService) UnregisterWorker(ctx context.Context, workerID uuid.UUID) error {
	args := m.Called(ctx, workerID)
	return args.Error(0)
}

func (m *mockWorkerService) GetWorkerStatus(ctx context.Context, workerID uuid.UUID) (*model.Worker, error) {
	args := m.Called(ctx, workerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Worker), args.Error(1)
}

func (m *mockWorkerService) ListWorkers(ctx context.Context, status, zone string, page, pageSize int) ([]*model.Worker, int, error) { //nolint:revive // import-shadowing: status — имя параметра интерфейса
	args := m.Called(ctx, status, zone, page, pageSize)
	return args.Get(0).([]*model.Worker), args.Get(1).(int), args.Error(2)
}

type mockCheckScheduler struct {
	mock.Mock
}

func (m *mockCheckScheduler) ScheduleCheck(ctx context.Context, monitorID uuid.UUID, priority model.CheckPriority, scheduledAt time.Time) (*model.ScheduledCheck, error) {
	args := m.Called(ctx, monitorID, priority, scheduledAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ScheduledCheck), args.Error(1)
}

func (m *mockCheckScheduler) GetScheduledChecks(ctx context.Context, from, to time.Time, status string, page, pageSize int) ([]*model.ScheduledCheck, int, error) { //nolint:revive // import-shadowing: status — имя параметра интерфейса
	args := m.Called(ctx, from, to, status, page, pageSize)
	return args.Get(0).([]*model.ScheduledCheck), args.Get(1).(int), args.Error(2)
}

func (m *mockCheckScheduler) GetNextCheck(ctx context.Context, monitorID uuid.UUID) (*model.ScheduledCheck, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ScheduledCheck), args.Error(1)
}

func (m *mockCheckScheduler) TriggerScheduledCheck(ctx context.Context, monitorID uuid.UUID, priority model.CheckPriority) (*model.ScheduledCheck, error) {
	args := m.Called(ctx, monitorID, priority)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ScheduledCheck), args.Error(1)
}

func (m *mockCheckScheduler) CompleteCheck(ctx context.Context, checkID uuid.UUID, failed bool, errMsg string) error {
	args := m.Called(ctx, checkID, failed, errMsg)
	return args.Error(0)
}

func newTestHandler(t *testing.T) (*SchedulerHandler, *mockWorkerService, *mockCheckScheduler) {
	t.Helper()
	ws := &mockWorkerService{}
	cs := &mockCheckScheduler{}
	h := NewSchedulerHandler(ws, cs)
	return h, ws, cs
}

func TestRegisterWorker(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	worker := model.NewWorker("worker-01", "msk")
	ws.On("RegisterWorker", mock.Anything, "worker-01", "msk", map[string]string{"v": "1"}).Return(worker, nil)

	req := &api.RegisterWorkerRequest{Name: "worker-01", Zone: "msk", Metadata: map[string]string{"v": "1"}}
	resp, err := h.RegisterWorker(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, worker.ID.String(), resp.Id)
	assert.Equal(t, "worker-01", resp.Name)
	assert.Equal(t, "msk", resp.Zone)
	assert.Equal(t, api.WorkerStatus_WORKER_STATUS_IDLE, resp.Status)
}

func TestRegisterWorker_duplicate(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	ws.On("RegisterWorker", mock.Anything, "worker-01", "msk", mock.Anything).Return(nil, model.ErrWorkerNameExists)

	req := &api.RegisterWorkerRequest{Name: "worker-01", Zone: "msk"}
	_, err := h.RegisterWorker(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

func TestRegisterWorker_validation_error(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	ws.On("RegisterWorker", mock.Anything, "", "msk", mock.Anything).Return(nil, model.ErrWorkerNameRequired)

	req := &api.RegisterWorkerRequest{Name: "", Zone: "msk"}
	_, err := h.RegisterWorker(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestWorkerHeartbeat(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	worker := model.NewWorker("worker-01", "msk")
	ws.On("Heartbeat", mock.Anything, worker.ID, model.WorkerStatusBusy, 10, 2, 150.5).Return(nil)

	req := &api.HeartbeatRequest{
		WorkerId:           worker.ID.String(),
		Status:             "BUSY",
		ChecksCompleted:    10,
		ChecksFailed:       2,
		AvgCheckDurationMs: 150.5,
	}
	_, err := h.WorkerHeartbeat(context.Background(), req)

	require.NoError(t, err)
}

func TestWorkerHeartbeat_invalid_uuid(t *testing.T) {
	t.Parallel()

	h, _, _ := newTestHandler(t)

	req := &api.HeartbeatRequest{WorkerId: "not-a-uuid"}
	_, err := h.WorkerHeartbeat(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid worker_id format")
}

func TestWorkerHeartbeat_invalid_status(t *testing.T) {
	t.Parallel()

	h, _, _ := newTestHandler(t)
	workerID := uuid.New()

	req := &api.HeartbeatRequest{WorkerId: workerID.String(), Status: "INVALID"}
	_, err := h.WorkerHeartbeat(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestWorkerHeartbeat_worker_not_found(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	workerID := uuid.New()
	ws.On("Heartbeat", mock.Anything, workerID, model.WorkerStatusIdle, 0, 0, float64(0)).Return(model.ErrWorkerNotFound)

	req := &api.HeartbeatRequest{WorkerId: workerID.String(), Status: "IDLE"}
	_, err := h.WorkerHeartbeat(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUnregisterWorker(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	workerID := uuid.New()
	ws.On("UnregisterWorker", mock.Anything, workerID).Return(nil)

	req := &api.UnregisterWorkerRequest{WorkerId: workerID.String()}
	_, err := h.UnregisterWorker(context.Background(), req)

	require.NoError(t, err)
}

func TestUnregisterWorker_invalid_uuid(t *testing.T) {
	t.Parallel()

	h, _, _ := newTestHandler(t)

	req := &api.UnregisterWorkerRequest{WorkerId: "bad"}
	_, err := h.UnregisterWorker(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestUnregisterWorker_not_found(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	workerID := uuid.New()
	ws.On("UnregisterWorker", mock.Anything, workerID).Return(model.ErrWorkerNotFound)

	req := &api.UnregisterWorkerRequest{WorkerId: workerID.String()}
	_, err := h.UnregisterWorker(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestGetWorkerStatus(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	worker := model.NewWorker("worker-01", "msk")
	ws.On("GetWorkerStatus", mock.Anything, worker.ID).Return(worker, nil)

	req := &api.GetWorkerStatusRequest{WorkerId: worker.ID.String()}
	resp, err := h.GetWorkerStatus(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "worker-01", resp.Name)
	assert.Equal(t, api.WorkerStatus_WORKER_STATUS_IDLE, resp.Status)
}

func TestGetWorkerStatus_invalid_uuid(t *testing.T) {
	t.Parallel()

	h, _, _ := newTestHandler(t)

	req := &api.GetWorkerStatusRequest{WorkerId: "bad"}
	_, err := h.GetWorkerStatus(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetWorkerStatus_not_found(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	workerID := uuid.New()
	ws.On("GetWorkerStatus", mock.Anything, workerID).Return(nil, model.ErrWorkerNotFound)

	req := &api.GetWorkerStatusRequest{WorkerId: workerID.String()}
	_, err := h.GetWorkerStatus(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestListWorkers(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	w1 := model.NewWorker("w1", "msk")
	w2 := model.NewWorker("w2", "spb")
	ws.On("ListWorkers", mock.Anything, "IDLE", "msk", 1, 20).Return([]*model.Worker{w1, w2}, 2, nil)

	req := &api.ListWorkersRequest{Status: "IDLE", Zone: "msk", Page: 1, PageSize: 20}
	resp, err := h.ListWorkers(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Workers, 2)
	assert.Equal(t, int32(2), resp.Total)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(20), resp.PageSize)
}

func TestListWorkers_default_pagination(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	ws.On("ListWorkers", mock.Anything, "", "", 1, 20).Return([]*model.Worker{}, 0, nil)

	req := &api.ListWorkersRequest{}
	resp, err := h.ListWorkers(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Workers, 0)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(20), resp.PageSize)
}

func TestGetScheduledChecks(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.PriorityNormal, time.Now())
	cs.On("GetScheduledChecks", mock.Anything, mock.Anything, mock.Anything, "", 1, 20).Return([]*model.ScheduledCheck{check}, 1, nil)

	req := &api.GetScheduledChecksRequest{
		FromTime: timestamppb.New(time.Now().Add(-1 * time.Hour)),
		ToTime:   timestamppb.New(time.Now()),
		Page:     1,
		PageSize: 20,
	}
	resp, err := h.GetScheduledChecks(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Checks, 1)
	assert.Equal(t, monitorID.String(), resp.Checks[0].MonitorId)
	assert.Equal(t, int32(1), resp.Total)
}

func TestGetNextCheck(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.PriorityNormal, time.Now())
	cs.On("GetNextCheck", mock.Anything, monitorID).Return(check, nil)

	req := &api.GetNextCheckRequest{MonitorId: monitorID.String()}
	resp, err := h.GetNextCheck(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, check.ID.String(), resp.Id)
	assert.Equal(t, monitorID.String(), resp.MonitorId)
}

func TestGetNextCheck_invalid_uuid(t *testing.T) {
	t.Parallel()

	h, _, _ := newTestHandler(t)

	req := &api.GetNextCheckRequest{MonitorId: "bad"}
	_, err := h.GetNextCheck(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetNextCheck_not_found(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	monitorID := uuid.New()
	cs.On("GetNextCheck", mock.Anything, monitorID).Return(nil, model.ErrMonitorNotFound)

	req := &api.GetNextCheckRequest{MonitorId: monitorID.String()}
	_, err := h.GetNextCheck(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestTriggerScheduledCheck(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.PriorityHigh, time.Now())
	cs.On("TriggerScheduledCheck", mock.Anything, monitorID, model.PriorityHigh).Return(check, nil)

	req := &api.TriggerScheduledCheckRequest{MonitorId: monitorID.String(), Priority: 1}
	_, err := h.TriggerScheduledCheck(context.Background(), req)

	require.NoError(t, err)
}

func TestTriggerScheduledCheck_default_priority(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	monitorID := uuid.New()
	cs.On("TriggerScheduledCheck", mock.Anything, monitorID, model.PriorityNormal).Return(nil, nil)

	req := &api.TriggerScheduledCheckRequest{MonitorId: monitorID.String()}
	_, err := h.TriggerScheduledCheck(context.Background(), req)

	require.NoError(t, err)
}

func TestTriggerScheduledCheck_invalid_uuid(t *testing.T) {
	t.Parallel()

	h, _, _ := newTestHandler(t)

	req := &api.TriggerScheduledCheckRequest{MonitorId: "bad"}
	_, err := h.TriggerScheduledCheck(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestCompleteCheck(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	checkID := uuid.New()
	cs.On("CompleteCheck", mock.Anything, checkID, false, "").Return(nil)

	req := &api.CompleteCheckRequest{CheckId: checkID.String(), Failed: false}
	_, err := h.CompleteCheck(context.Background(), req)

	require.NoError(t, err)
}

func TestCompleteCheck_failed(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	checkID := uuid.New()
	cs.On("CompleteCheck", mock.Anything, checkID, true, "connection refused").Return(nil)

	req := &api.CompleteCheckRequest{CheckId: checkID.String(), Failed: true, ErrorMessage: "connection refused"}
	_, err := h.CompleteCheck(context.Background(), req)

	require.NoError(t, err)
}

func TestCompleteCheck_invalid_uuid(t *testing.T) {
	t.Parallel()

	h, _, _ := newTestHandler(t)

	req := &api.CompleteCheckRequest{CheckId: "bad"}
	_, err := h.CompleteCheck(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestHandleError_unknown_error(t *testing.T) {
	t.Parallel()

	err := handleError(errors.New("something unexpected"))
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal server error", st.Message())
}

func TestHandleError_no_workers_available(t *testing.T) {
	t.Parallel()

	err := handleError(model.ErrNoWorkersAvailable)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
}

func TestGetScheduledChecks_error(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	cs.On("GetScheduledChecks", mock.Anything, mock.Anything, mock.Anything, "", 1, 20).Return(
		[]*model.ScheduledCheck{}, 0, model.ErrInvalidTimeRange,
	)

	req := &api.GetScheduledChecksRequest{
		FromTime: timestamppb.New(time.Now().Add(-1 * time.Hour)),
		ToTime:   timestamppb.New(time.Now()),
		Page:     1,
		PageSize: 20,
	}
	_, err := h.GetScheduledChecks(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetScheduledChecks_default_pagination(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	cs.On("GetScheduledChecks", mock.Anything, mock.Anything, mock.Anything, "", 1, 20).Return(
		[]*model.ScheduledCheck{}, 0, nil,
	)

	req := &api.GetScheduledChecksRequest{
		FromTime: timestamppb.New(time.Now().Add(-1 * time.Hour)),
		ToTime:   timestamppb.New(time.Now()),
	}
	resp, err := h.GetScheduledChecks(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(20), resp.PageSize)
}

func TestTriggerScheduledCheck_error(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	monitorID := uuid.New()
	cs.On("TriggerScheduledCheck", mock.Anything, monitorID, model.PriorityNormal).Return(nil, model.ErrMonitorNotFound)

	req := &api.TriggerScheduledCheckRequest{MonitorId: monitorID.String()}
	_, err := h.TriggerScheduledCheck(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCompleteCheck_not_found(t *testing.T) {
	t.Parallel()

	h, _, cs := newTestHandler(t)
	checkID := uuid.New()
	cs.On("CompleteCheck", mock.Anything, checkID, false, "").Return(model.ErrCheckNotFound)

	req := &api.CompleteCheckRequest{CheckId: checkID.String()}
	_, err := h.CompleteCheck(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestListWorkers_error(t *testing.T) {
	t.Parallel()

	h, ws, _ := newTestHandler(t)
	ws.On("ListWorkers", mock.Anything, "", "", 1, 20).Return(
		[]*model.Worker{}, 0, model.ErrWorkerNotFound,
	)

	req := &api.ListWorkersRequest{}
	_, err := h.ListWorkers(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestModelStatusToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status model.WorkerStatus
		want   api.WorkerStatus
	}{
		{model.WorkerStatusIdle, api.WorkerStatus_WORKER_STATUS_IDLE},
		{model.WorkerStatusBusy, api.WorkerStatus_WORKER_STATUS_BUSY},
		{model.WorkerStatusOffline, api.WorkerStatus_WORKER_STATUS_OFFLINE},
		{model.WorkerStatus("UNKNOWN"), api.WorkerStatus_WORKER_STATUS_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, modelStatusToProto(tc.status))
		})
	}
}

func TestModelPriorityToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		priority model.CheckPriority
		want     api.CheckPriority
	}{
		{model.PriorityLow, api.CheckPriority_PRIORITY_LOW},
		{model.PriorityNormal, api.CheckPriority_PRIORITY_NORMAL},
		{model.PriorityHigh, api.CheckPriority_PRIORITY_HIGH},
		{model.PriorityCritical, api.CheckPriority_PRIORITY_CRITICAL},
		{model.CheckPriority("UNKNOWN"), api.CheckPriority_PRIORITY_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(string(tc.priority), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, modelPriorityToProto(tc.priority))
		})
	}
}

func TestToProtoCheck_with_worker_and_times(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()
	workerID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.PriorityHigh, time.Now())
	check.Assign(workerID)
	check.Complete()

	proto := toProtoCheck(check)

	assert.Equal(t, workerID.String(), proto.WorkerId)
	assert.NotNil(t, proto.ExecutedAt)
	assert.NotNil(t, proto.CompletedAt)
	assert.NotNil(t, proto.ScheduledAt)
}

func TestToProtoCheck_no_worker(t *testing.T) {
	t.Parallel()

	check := model.NewScheduledCheck(uuid.New(), model.PriorityLow, time.Now())

	proto := toProtoCheck(check)

	assert.Empty(t, proto.WorkerId)
	assert.Nil(t, proto.ExecutedAt)
	assert.Nil(t, proto.CompletedAt)
}
