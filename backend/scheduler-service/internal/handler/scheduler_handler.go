package handler

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	api "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

type SchedulerHandler struct {
	api.UnimplementedSchedulerServiceServer
	workerService  WorkerServiceServer
	checkScheduler CheckSchedulerServer
}

type WorkerServiceServer interface {
	RegisterWorker(ctx context.Context, name, zone string, metadata map[string]string) (*model.Worker, error)
	Heartbeat(ctx context.Context, workerID uuid.UUID, status model.WorkerStatus, checksCompleted, checksFailed int, avgDurationMs float64) error //nolint:revive // import-shadowing: status — имя параметра интерфейса
	UnregisterWorker(ctx context.Context, workerID uuid.UUID) error
	GetWorkerStatus(ctx context.Context, workerID uuid.UUID) (*model.Worker, error)
	ListWorkers(ctx context.Context, status, zone string, page, pageSize int) ([]*model.Worker, int, error) //nolint:revive // import-shadowing: status — имя параметра интерфейса
}

type CheckSchedulerServer interface {
	ScheduleCheck(ctx context.Context, monitorID uuid.UUID, priority model.CheckPriority, scheduledAt time.Time) (*model.ScheduledCheck, error)
	GetScheduledChecks(ctx context.Context, from, to time.Time, status string, page, pageSize int) ([]*model.ScheduledCheck, int, error) //nolint:revive // import-shadowing: status — имя параметра интерфейса
	GetNextCheck(ctx context.Context, monitorID uuid.UUID) (*model.ScheduledCheck, error)
	TriggerScheduledCheck(ctx context.Context, monitorID uuid.UUID, priority model.CheckPriority) (*model.ScheduledCheck, error)
	CompleteCheck(ctx context.Context, checkID uuid.UUID, failed bool, errMsg string) error
}

func NewSchedulerHandler(
	workerService WorkerServiceServer,
	checkScheduler CheckSchedulerServer,
) *SchedulerHandler {
	return &SchedulerHandler{
		workerService:  workerService,
		checkScheduler: checkScheduler,
	}
}

func handleError(err error) error {
	var schedErr *model.SchedulerError
	if errors.As(err, &schedErr) {
		switch schedErr.Code {
		case "WORKER_NOT_FOUND", "MONITOR_NOT_FOUND", "CHECK_NOT_FOUND":
			return status.Error(codes.NotFound, schedErr.Message)
		case "WORKER_NAME_EXISTS":
			return status.Error(codes.AlreadyExists, schedErr.Message)
		case "WORKER_NAME_REQUIRED", "WORKER_NAME_TOO_LONG", "INVALID_ZONE",
			"INVALID_WORKER_STATUS", "INVALID_TIME_RANGE", "CHECK_ALREADY_PENDING":
			return status.Error(codes.InvalidArgument, schedErr.Message)
		case "NO_WORKERS_AVAILABLE":
			return status.Error(codes.Unavailable, schedErr.Message)
		}
	}
	return status.Error(codes.Internal, "internal server error")
}

func (h *SchedulerHandler) RegisterWorker(ctx context.Context, req *api.RegisterWorkerRequest) (*api.Worker, error) {
	worker, err := h.workerService.RegisterWorker(ctx, req.Name, req.Zone, req.Metadata)
	if err != nil {
		return nil, handleError(err)
	}
	return toProtoWorker(worker), nil
}

func (h *SchedulerHandler) WorkerHeartbeat(ctx context.Context, req *api.HeartbeatRequest) (*api.Empty, error) {
	workerID, err := uuid.Parse(req.WorkerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid worker_id format")
	}

	s, err := model.ParseWorkerStatus(req.Status)
	if err != nil {
		return nil, handleError(err)
	}

	if err := h.workerService.Heartbeat(ctx, workerID, s, int(req.ChecksCompleted), int(req.ChecksFailed), req.AvgCheckDurationMs); err != nil {
		return nil, handleError(err)
	}

	return &api.Empty{}, nil
}

func (h *SchedulerHandler) UnregisterWorker(ctx context.Context, req *api.UnregisterWorkerRequest) (*api.Empty, error) {
	workerID, err := uuid.Parse(req.WorkerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid worker_id format")
	}

	if err := h.workerService.UnregisterWorker(ctx, workerID); err != nil {
		return nil, handleError(err)
	}

	return &api.Empty{}, nil
}

func (h *SchedulerHandler) GetWorkerStatus(ctx context.Context, req *api.GetWorkerStatusRequest) (*api.Worker, error) {
	workerID, err := uuid.Parse(req.WorkerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid worker_id format")
	}

	worker, err := h.workerService.GetWorkerStatus(ctx, workerID)
	if err != nil {
		return nil, handleError(err)
	}

	return toProtoWorker(worker), nil
}

func (h *SchedulerHandler) ListWorkers(ctx context.Context, req *api.ListWorkersRequest) (*api.ListWorkersResponse, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	workers, total, err := h.workerService.ListWorkers(ctx, req.Status, req.Zone, page, pageSize)
	if err != nil {
		return nil, handleError(err)
	}

	protoWorkers := make([]*api.Worker, 0, len(workers))
	for _, w := range workers {
		protoWorkers = append(protoWorkers, toProtoWorker(w))
	}

	return &api.ListWorkersResponse{
		Workers:  protoWorkers,
		Total:    int32(total),    //nolint:gosec // G115: значения в допустимом диапазоне
		Page:     int32(page),     //nolint:gosec // G115: значения в допустимом диапазоне
		PageSize: int32(pageSize), //nolint:gosec // G115: значения в допустимом диапазоне
	}, nil
}

func (h *SchedulerHandler) GetScheduledChecks(ctx context.Context, req *api.GetScheduledChecksRequest) (*api.ScheduledChecks, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	from := req.FromTime.AsTime()
	to := req.ToTime.AsTime()

	checks, total, err := h.checkScheduler.GetScheduledChecks(ctx, from, to, req.Status, page, pageSize)
	if err != nil {
		return nil, handleError(err)
	}

	protoChecks := make([]*api.ScheduledCheck, 0, len(checks))
	for _, c := range checks {
		protoChecks = append(protoChecks, toProtoCheck(c))
	}

	return &api.ScheduledChecks{
		Checks:   protoChecks,
		Total:    int32(total),    //nolint:gosec // G115: значения в допустимом диапазоне
		Page:     int32(page),     //nolint:gosec // G115: значения в допустимом диапазоне
		PageSize: int32(pageSize), //nolint:gosec // G115: значения в допустимом диапазоне
	}, nil
}

func (h *SchedulerHandler) GetNextCheck(ctx context.Context, req *api.GetNextCheckRequest) (*api.ScheduledCheck, error) {
	monitorID, err := uuid.Parse(req.MonitorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid monitor_id format")
	}

	check, err := h.checkScheduler.GetNextCheck(ctx, monitorID)
	if err != nil {
		return nil, handleError(err)
	}

	return toProtoCheck(check), nil
}

func (h *SchedulerHandler) TriggerScheduledCheck(ctx context.Context, req *api.TriggerScheduledCheckRequest) (*api.Empty, error) {
	monitorID, err := uuid.Parse(req.MonitorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid monitor_id format")
	}

	priority := model.PriorityHigh
	if req.Priority == 0 {
		priority = model.PriorityNormal
	}

	_, err = h.checkScheduler.TriggerScheduledCheck(ctx, monitorID, priority)
	if err != nil {
		return nil, handleError(err)
	}

	return &api.Empty{}, nil
}

func (h *SchedulerHandler) CompleteCheck(ctx context.Context, req *api.CompleteCheckRequest) (*api.Empty, error) {
	checkID, err := uuid.Parse(req.CheckId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid check_id format")
	}

	if err := h.checkScheduler.CompleteCheck(ctx, checkID, req.Failed, req.ErrorMessage); err != nil {
		return nil, handleError(err)
	}

	return &api.Empty{}, nil
}

func toProtoWorker(w *model.Worker) *api.Worker {
	return &api.Worker{
		Id:                 w.ID.String(),
		Name:               w.Name,
		Zone:               w.Zone,
		Status:             modelStatusToProto(w.Status),
		LastHeartbeat:      timestamppb.New(w.LastHeartbeat),
		ChecksCompleted:    int32(w.ChecksCompleted), //nolint:gosec // G115: значения в допустимом диапазоне
		ChecksFailed:       int32(w.ChecksFailed),    //nolint:gosec // G115: значения в допустимом диапазоне
		AvgCheckDurationMs: w.AvgCheckDurationMs,
		Metadata:           w.Metadata,
	}
}

func toProtoCheck(c *model.ScheduledCheck) *api.ScheduledCheck {
	proto := &api.ScheduledCheck{
		Id:           c.ID.String(),
		MonitorId:    c.MonitorID.String(),
		Priority:     modelPriorityToProto(c.Priority),
		Status:       string(c.Status),
		ErrorMessage: c.ErrorMessage,
	}

	if c.WorkerID != uuid.Nil {
		proto.WorkerId = c.WorkerID.String()
	}

	proto.ScheduledAt = timestamppb.New(c.ScheduledAt)
	if c.ExecutedAt != nil {
		proto.ExecutedAt = timestamppb.New(*c.ExecutedAt)
	}
	if c.CompletedAt != nil {
		proto.CompletedAt = timestamppb.New(*c.CompletedAt)
	}

	return proto
}

func modelStatusToProto(s model.WorkerStatus) api.WorkerStatus {
	switch s {
	case model.WorkerStatusIdle:
		return api.WorkerStatus_WORKER_STATUS_IDLE
	case model.WorkerStatusBusy:
		return api.WorkerStatus_WORKER_STATUS_BUSY
	case model.WorkerStatusOffline:
		return api.WorkerStatus_WORKER_STATUS_OFFLINE
	default:
		return api.WorkerStatus_WORKER_STATUS_UNSPECIFIED
	}
}

func modelPriorityToProto(p model.CheckPriority) api.CheckPriority {
	switch p {
	case model.PriorityLow:
		return api.CheckPriority_PRIORITY_LOW
	case model.PriorityNormal:
		return api.CheckPriority_PRIORITY_NORMAL
	case model.PriorityHigh:
		return api.CheckPriority_PRIORITY_HIGH
	case model.PriorityCritical:
		return api.CheckPriority_PRIORITY_CRITICAL
	default:
		return api.CheckPriority_PRIORITY_UNSPECIFIED
	}
}
