package handler

import (
	"context"
	"errors"

	"github.com/google/uuid"
	api "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	domain "github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/service/maintenance"
)

// epic=08_maintenance, us=08_01_maintenance_windows

// MaintenanceWindowService определяет интерфейс сервиса окон обслуживания
type MaintenanceWindowService interface {
	CreateMaintenanceWindow(ctx context.Context, req maintenance.CreateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error)
	UpdateMaintenanceWindow(ctx context.Context, req maintenance.UpdateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error)
	ListMaintenanceWindows(ctx context.Context, req maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error)
	DeleteMaintenanceWindow(ctx context.Context, id string, userID uuid.UUID) error
	CancelMaintenanceWindow(ctx context.Context, id string, userID uuid.UUID, reason string) (*domain.MaintenanceWindow, error)
	GetMaintenanceWindowHistory(ctx context.Context, req maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error)
}

// MaintenanceWindowServiceServer реализует gRPC MaintenanceWindowService
type MaintenanceWindowServiceServer struct {
	maintenanceService MaintenanceWindowService
	authMiddleware     *auth.AuthMiddleware
	api.UnimplementedMaintenanceWindowServiceServer
}

// NewMaintenanceWindowServiceServer создаёт новый экземпляр MaintenanceWindowServiceServer
func NewMaintenanceWindowServiceServer(
	maintenanceService MaintenanceWindowService,
	authMiddleware *auth.AuthMiddleware,
) *MaintenanceWindowServiceServer {
	return &MaintenanceWindowServiceServer{
		maintenanceService: maintenanceService,
		authMiddleware:     authMiddleware,
	}
}

// uc_08_01_01: Create a maintenance window
func (s *MaintenanceWindowServiceServer) CreateMaintenanceWindow(ctx context.Context, req *api.CreateMaintenanceWindowRequest) (*api.MaintenanceWindow, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.StartTime == nil {
		return nil, status.Error(codes.InvalidArgument, "start_time is required")
	}
	if req.EndTime == nil {
		return nil, status.Error(codes.InvalidArgument, "end_time is required")
	}
	if !req.IsGlobal && len(req.MonitorIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "monitor_ids is required for non-global windows")
	}

	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	userRole := extractUserRole(ctx)

	svcReq := maintenance.CreateMaintenanceWindowRequest{
		UserID:          userID,
		UserRole:        userRole,
		Name:            req.Name,
		StartsAt:        req.StartTime.AsTime(),
		EndsAt:          req.EndTime.AsTime(),
		Recurrence:      protoToRecurrenceType(req.Recurrence),
		IsGlobal:        req.IsGlobal,
		PauseMonitoring: req.PauseMonitoring,
		SuppressAlerts:  req.SuppressAlerts,
		SafeMode:        req.SafeMode,
		MonitorIDs:      req.MonitorIds,
	}

	mw, err := s.maintenanceService.CreateMaintenanceWindow(ctx, svcReq)
	if err != nil {
		return nil, mapMaintenanceError(err)
	}

	return MaintenanceWindowToProto(mw), nil
}

// uc_08_01_19: Update a maintenance window
func (s *MaintenanceWindowServiceServer) UpdateMaintenanceWindow(ctx context.Context, req *api.UpdateMaintenanceWindowRequest) (*api.MaintenanceWindow, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if _, err := uuid.Parse(req.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	svcReq := maintenance.UpdateMaintenanceWindowRequest{
		ID:      req.Id,
		UserID:  userID,
		Name:    req.Name,
		Version: req.Version,
	}

	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		svcReq.StartsAt = &t
	}
	if req.EndTime != nil {
		t := req.EndTime.AsTime()
		svcReq.EndsAt = &t
	}

	mw, err := s.maintenanceService.UpdateMaintenanceWindow(ctx, svcReq)
	if err != nil {
		return nil, mapMaintenanceError(err)
	}

	return MaintenanceWindowToProto(mw), nil
}

// uc_08_01_20: List maintenance windows
func (s *MaintenanceWindowServiceServer) ListMaintenanceWindows(ctx context.Context, req *api.ListMaintenanceWindowsRequest) (*api.ListMaintenanceWindowsResponse, error) {
	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	filter := domain.MaintenanceWindowFilter{
		Status:   protoToMaintenanceStatus(req.Status),
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		filter.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		filter.EndDate = &t
	}

	svcReq := maintenance.ListMaintenanceWindowsRequest{
		UserID: userID,
		Filter: filter,
	}

	windows, total, err := s.maintenanceService.ListMaintenanceWindows(ctx, svcReq)
	if err != nil {
		return nil, mapMaintenanceError(err)
	}

	protoWindows := make([]*api.MaintenanceWindow, 0, len(windows))
	for _, mw := range windows {
		protoWindows = append(protoWindows, MaintenanceWindowToProto(mw))
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	return &api.ListMaintenanceWindowsResponse{
		Windows:  protoWindows,
		Total:    int32(total), // #nosec G115 -- bounded by DB result count
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// uc_08_01_24: Delete a maintenance window
func (s *MaintenanceWindowServiceServer) DeleteMaintenanceWindow(ctx context.Context, req *api.DeleteMaintenanceWindowRequest) (*api.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if _, err := uuid.Parse(req.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	if err := s.maintenanceService.DeleteMaintenanceWindow(ctx, req.Id, userID); err != nil {
		return nil, mapMaintenanceError(err)
	}

	return &api.Empty{}, nil
}

// uc_08_01_25a: Cancel an active maintenance window
func (s *MaintenanceWindowServiceServer) CancelMaintenanceWindow(ctx context.Context, req *api.CancelMaintenanceWindowRequest) (*api.MaintenanceWindow, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if _, err := uuid.Parse(req.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	mw, err := s.maintenanceService.CancelMaintenanceWindow(ctx, req.Id, userID, req.CancellationReason)
	if err != nil {
		return nil, mapMaintenanceError(err)
	}

	return MaintenanceWindowToProto(mw), nil
}

// uc_08_01_21: Get maintenance window history
func (s *MaintenanceWindowServiceServer) GetMaintenanceWindowHistory(ctx context.Context, req *api.GetMaintenanceWindowHistoryRequest) (*api.GetMaintenanceWindowHistoryResponse, error) {
	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	filter := domain.MaintenanceWindowFilter{
		Status:   domain.MaintenanceStatusCompleted,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		filter.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		filter.EndDate = &t
	}

	svcReq := maintenance.ListMaintenanceWindowsRequest{
		UserID: userID,
		Filter: filter,
	}

	windows, total, err := s.maintenanceService.GetMaintenanceWindowHistory(ctx, svcReq)
	if err != nil {
		return nil, mapMaintenanceError(err)
	}

	protoWindows := make([]*api.MaintenanceWindow, 0, len(windows))
	for _, mw := range windows {
		protoWindows = append(protoWindows, MaintenanceWindowToProto(mw))
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	return &api.GetMaintenanceWindowHistoryResponse{
		Windows:  protoWindows,
		Total:    int32(total), // #nosec G115 -- bounded by DB result count
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// mapMaintenanceError маппит доменные ошибки в gRPC status codes
func mapMaintenanceError(err error) error {
	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}

	switch domainErr.Code() {
	case domain.ErrMaintenanceWindowNotFound.Code():
		return status.Error(codes.NotFound, domainErr.Error())
	case domain.ErrMaintenanceWindowDurationExceeded.Code(),
		domain.ErrMaintenanceWindowDurationTooShort.Code(),
		domain.ErrMaintenanceWindowInPast.Code(),
		domain.ErrEndTimeBeforeStartTime.Code():
		return status.Error(codes.InvalidArgument, domainErr.Error())
	case domain.ErrOverlappingMaintenanceWindows.Code():
		return status.Error(codes.AlreadyExists, domainErr.Error())
	case domain.ErrMaintenanceWindowNotScheduled.Code():
		return status.Error(codes.FailedPrecondition, domainErr.Error())
	case domain.ErrMaintenanceWindowConflict.Code():
		return status.Error(codes.Aborted, domainErr.Error())
	case domain.ErrInsufficientPermissions.Code():
		return status.Error(codes.PermissionDenied, domainErr.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

// extractUserRole извлекает роль пользователя из контекста JWT
func extractUserRole(ctx context.Context) string {
	return string(auth.ExtractRole(ctx))
}
