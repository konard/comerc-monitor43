// Package grpc предоставляет gRPC handlers для monitor service.
//
// Пакет реализует gRPC API согласно протоколу, определённому в api/proto/maintenance.proto.
package grpc

import (
	"context"

	"github.com/pkg/errors"
	maintenancepb "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/monitor-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/monitor-service/internal/service"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// MaintenanceServiceAlias - псевдоним для интерфейса, чтобы избежать конфликта имён
type MaintenanceServiceAlias = service.MaintenanceServiceInterface

// MaintenanceHandler реализует gRPC handlers для maintenance window service.
type MaintenanceHandler struct {
	maintenancepb.MaintenanceWindowServiceServer
	maintenanceService MaintenanceServiceAlias
}

// NewMaintenanceHandler создаёт новый MaintenanceHandler.
func NewMaintenanceHandler(maintenanceService MaintenanceServiceAlias) *MaintenanceHandler {
	return &MaintenanceHandler{
		maintenanceService: maintenanceService,
	}
}

// CreateMaintenanceWindow создаёт новое окно обслуживания (uc_08_01_01).
func (h *MaintenanceHandler) CreateMaintenanceWindow(ctx context.Context, req *maintenancepb.CreateMaintenanceWindowRequest) (*maintenancepb.MaintenanceWindow, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceHandler.CreateMaintenanceWindow")
	defer span.End()

	// Извлекаем user_id и role из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	userRole, err := h.extractUserRole(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	userTier, err := h.extractUserTier(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Конвертируем monitor IDs
	monitorIDs := make([]string, len(req.MonitorIds))
	copy(monitorIDs, req.MonitorIds)

	// Конвертируем в DTO
	createReq := &dto.CreateMaintenanceWindowRequest{
		UserID:          userID,
		Name:            req.Name,
		StartTime:       req.StartTime.AsTime(),
		EndTime:         req.EndTime.AsTime(),
		Recurrence:      mapRecurrenceFromProto(req.Recurrence),
		IsGlobal:        req.IsGlobal,
		PauseMonitoring: req.PauseMonitoring,
		SuppressAlerts:  req.SuppressAlerts,
		SafeMode:        req.SafeMode,
		MonitorIDs:      monitorIDs,
		UserRole:        userRole,
		UserTier:        userTier,
	}

	// Создаём окно
	resp, err := h.maintenanceService.CreateMaintenanceWindow(ctx, createReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	var activatedAt, completedAt *timestamppb.Timestamp
	if resp.ActivatedAt != nil {
		activatedAt = timestamppb.New(*resp.ActivatedAt)
	}
	if resp.CompletedAt != nil {
		completedAt = timestamppb.New(*resp.CompletedAt)
	}

	tracing.SetSuccess(span)
	return &maintenancepb.MaintenanceWindow{
		Id:              resp.ID,
		UserId:          resp.UserID,
		Name:            resp.Name,
		StartTime:       timestamppb.New(resp.StartTime),
		EndTime:         timestamppb.New(resp.EndTime),
		Status:          mapStatusToProto(resp.Status),
		Recurrence:      mapRecurrenceToProto(resp.Recurrence),
		IsGlobal:        resp.IsGlobal,
		PauseMonitoring: resp.PauseMonitoring,
		SuppressAlerts:  resp.SuppressAlerts,
		SafeMode:        resp.SafeMode,
		MonitorIds:      resp.MonitorIDs,
		CreatedAt:       timestamppb.New(resp.CreatedAt),
		UpdatedAt:       timestamppb.New(resp.UpdatedAt),
		ActivatedAt:     activatedAt,
		CompletedAt:     completedAt,
		Version:         int32(resp.Version), //nolint:gosec // G115: Version — небольшое число, переполнение невозможно
	}, nil
}

// UpdateMaintenanceWindow обновляет окно обслуживания (uc_08_01_19).
func (h *MaintenanceHandler) UpdateMaintenanceWindow(ctx context.Context, req *maintenancepb.UpdateMaintenanceWindowRequest) (*maintenancepb.MaintenanceWindow, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceHandler.UpdateMaintenanceWindow")
	defer span.End()

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	updateReq := &dto.UpdateMaintenanceWindowRequest{
		ID:        req.Id,
		Name:      req.Name,
		StartTime: req.StartTime.AsTime(),
		EndTime:   req.EndTime.AsTime(),
		Version:   int(req.Version),
		UserID:    userID,
	}

	resp, err := h.maintenanceService.UpdateMaintenanceWindow(ctx, updateReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return h.dtoToProto(resp), nil
}

// ListMaintenanceWindows возвращает список окон (uc_08_01_20).
func (h *MaintenanceHandler) ListMaintenanceWindows(ctx context.Context, req *maintenancepb.ListMaintenanceWindowsRequest) (*maintenancepb.ListMaintenanceWindowsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceHandler.ListMaintenanceWindows")
	defer span.End()

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	listReq := &dto.ListMaintenanceWindowsRequest{
		UserID:   userID,
		Status:   mapStatusFromProto(req.Status),
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		listReq.StartDate = &t
	}

	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		listReq.EndDate = &t
	}

	resp, err := h.maintenanceService.ListMaintenanceWindows(ctx, listReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	windows := make([]*maintenancepb.MaintenanceWindow, len(resp.Windows))
	for i, w := range resp.Windows {
		windows[i] = h.dtoToProto(w)
	}

	tracing.SetSuccess(span)
	return &maintenancepb.ListMaintenanceWindowsResponse{
		Windows:  windows,
		Total:    int32(resp.Total),    //nolint:gosec // G115: Total — счётчик страниц, переполнение невозможно
		Page:     int32(resp.Page),     //nolint:gosec // G115: Page — номер страницы, переполнение невозможно
		PageSize: int32(resp.PageSize), //nolint:gosec // G115: PageSize — размер страницы, переполнение невозможно
	}, nil
}

// DeleteMaintenanceWindow удаляет окно (uc_08_01_24).
func (h *MaintenanceHandler) DeleteMaintenanceWindow(ctx context.Context, req *maintenancepb.DeleteMaintenanceWindowRequest) (*maintenancepb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceHandler.DeleteMaintenanceWindow")
	defer span.End()

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	if err := h.maintenanceService.DeleteMaintenanceWindow(ctx, req.Id, userID); err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &maintenancepb.Empty{}, nil
}

// CancelMaintenanceWindow отменяет окно (uc_08_01_25a, uc_08_01_25b).
func (h *MaintenanceHandler) CancelMaintenanceWindow(ctx context.Context, req *maintenancepb.CancelMaintenanceWindowRequest) (*maintenancepb.MaintenanceWindow, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceHandler.CancelMaintenanceWindow")
	defer span.End()

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	cancelReq := &dto.CancelMaintenanceWindowRequest{
		ID:                 req.Id,
		CancellationReason: req.CancellationReason,
		UserID:             userID,
	}

	resp, err := h.maintenanceService.CancelMaintenanceWindow(ctx, cancelReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return h.dtoToProto(resp), nil
}

// GetMaintenanceWindowHistory возвращает историю окон (uc_08_01_21).
func (h *MaintenanceHandler) GetMaintenanceWindowHistory(ctx context.Context, req *maintenancepb.GetMaintenanceWindowHistoryRequest) (*maintenancepb.GetMaintenanceWindowHistoryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MaintenanceHandler.GetMaintenanceWindowHistory")
	defer span.End()

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	historyReq := &dto.GetMaintenanceWindowHistoryRequest{
		UserID:     userID,
		StartDate:  req.StartDate.AsTime(),
		EndDate:    req.EndDate.AsTime(),
		MonitorIDs: req.MonitorIds,
		Page:       int(req.Page),
		PageSize:   int(req.PageSize),
	}

	resp, err := h.maintenanceService.GetMaintenanceWindowHistory(ctx, historyReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	windows := make([]*maintenancepb.MaintenanceWindow, len(resp.Windows))
	for i, w := range resp.Windows {
		windows[i] = h.dtoToProto(w)
	}

	tracing.SetSuccess(span)
	return &maintenancepb.GetMaintenanceWindowHistoryResponse{
		Windows:  windows,
		Total:    int32(resp.Total),    //nolint:gosec // G115: Total — счётчик страниц, переполнение невозможно
		Page:     int32(resp.Page),     //nolint:gosec // G115: Page — номер страницы, переполнение невозможно
		PageSize: int32(resp.PageSize), //nolint:gosec // G115: PageSize — размер страницы, переполнение невозможно
	}, nil
}

// Helper methods

// mapStatusToProto конвертирует строку статуса в proto enum.
func mapStatusToProto(status string) maintenancepb.MaintenanceWindowStatus { //nolint:revive // import-shadowing: параметр status перекрывает пакет, имя семантически правильное
	// Маппинг коротких имен service слоя на полные имена proto enum
	switch status {
	case "SCHEDULED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED
	case "ACTIVE":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE
	case "COMPLETED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED
	case "CANCELLED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED
	case "ORPHANED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED
	default:
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED
	}
}

// mapStatusFromProto конвертирует proto enum статуса в строку service слоя.
func mapStatusFromProto(status maintenancepb.MaintenanceWindowStatus) string { //nolint:revive // import-shadowing: параметр status перекрывает пакет, имя семантически правильное
	switch status {
	case maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED:
		return "SCHEDULED"
	case maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE:
		return "ACTIVE"
	case maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED:
		return "COMPLETED"
	case maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED:
		return "CANCELLED"
	case maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED:
		return "ORPHANED"
	default:
		return ""
	}
}

// mapRecurrenceToProto конвертирует строку recurrence в proto enum.
func mapRecurrenceToProto(recurrence string) maintenancepb.RecurrenceType {
	switch recurrence {
	case "ONCE":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE
	case "DAILY":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_DAILY
	case "WEEKLY":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_WEEKLY
	case "MONTHLY":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_MONTHLY
	default:
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED
	}
}

// mapRecurrenceFromProto конвертирует proto enum recurrence в строку service слоя.
func mapRecurrenceFromProto(recurrence maintenancepb.RecurrenceType) string {
	switch recurrence {
	case maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE:
		return "ONCE"
	case maintenancepb.RecurrenceType_RECURRENCE_TYPE_DAILY:
		return "DAILY"
	case maintenancepb.RecurrenceType_RECURRENCE_TYPE_WEEKLY:
		return "WEEKLY"
	case maintenancepb.RecurrenceType_RECURRENCE_TYPE_MONTHLY:
		return "MONTHLY"
	default:
		return ""
	}
}

// extractUserID извлекает user_id из контекста запроса.
// Должен быть вызван authentication middleware до вызова handler.
func (h *MaintenanceHandler) extractUserID(ctx context.Context) (string, error) {
	return middleware.ExtractUserID(ctx)
}

// extractUserRole извлекает user_role из контекста запроса.
func (h *MaintenanceHandler) extractUserRole(ctx context.Context) (string, error) {
	return middleware.ExtractUserRole(ctx)
}

// extractUserTier извлекает user_tier из контекста запроса.
func (h *MaintenanceHandler) extractUserTier(ctx context.Context) (string, error) {
	return middleware.ExtractUserTier(ctx)
}

func (h *MaintenanceHandler) dtoToProto(dto *dto.MaintenanceWindowResponse) *maintenancepb.MaintenanceWindow { //nolint:revive // import-shadowing: параметр dto перекрывает пакет, имя семантически правильное
	var activatedAt, completedAt *timestamppb.Timestamp
	if dto.ActivatedAt != nil {
		activatedAt = timestamppb.New(*dto.ActivatedAt)
	}
	if dto.CompletedAt != nil {
		completedAt = timestamppb.New(*dto.CompletedAt)
	}

	return &maintenancepb.MaintenanceWindow{
		Id:              dto.ID,
		UserId:          dto.UserID,
		Name:            dto.Name,
		StartTime:       timestamppb.New(dto.StartTime),
		EndTime:         timestamppb.New(dto.EndTime),
		Status:          mapStatusToProto(dto.Status),
		Recurrence:      mapRecurrenceToProto(dto.Recurrence),
		IsGlobal:        dto.IsGlobal,
		PauseMonitoring: dto.PauseMonitoring,
		SuppressAlerts:  dto.SuppressAlerts,
		SafeMode:        dto.SafeMode,
		MonitorIds:      dto.MonitorIDs,
		CreatedAt:       timestamppb.New(dto.CreatedAt),
		UpdatedAt:       timestamppb.New(dto.UpdatedAt),
		ActivatedAt:     activatedAt,
		CompletedAt:     completedAt,
		Version:         int32(dto.Version), //nolint:gosec // G115: Version — небольшое число, переполнение невозможно
	}
}

func (h *MaintenanceHandler) handleError(err error) error {
	if err == nil {
		return nil
	}

	// Преобразуем ошибки в правильные gRPC статусы
	errMsg := err.Error()

	switch {
	case contains(errMsg, "INSUFFICIENT_PERMISSIONS"):
		return status.Error(codes.PermissionDenied, errMsg)
	case contains(errMsg, "DURATION_EXCEEDS_MAXIMUM"):
		return status.Error(codes.InvalidArgument, errMsg)
	case contains(errMsg, "DURATION_BELOW_MINIMUM"):
		return status.Error(codes.InvalidArgument, errMsg)
	case contains(errMsg, "END_TIME_BEFORE_START_TIME"):
		return status.Error(codes.InvalidArgument, errMsg)
	case contains(errMsg, "CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST"):
		return status.Error(codes.InvalidArgument, errMsg)
	case contains(errMsg, "OVERLAPPING_MAINTENANCE_WINDOWS"):
		return status.Error(codes.AlreadyExists, errMsg)
	case contains(errMsg, "MAINTENANCE_WINDOW_LIMIT_REACHED"):
		return status.Error(codes.ResourceExhausted, errMsg)
	case contains(errMsg, "MONITOR_LIMIT_PER_WINDOW_REACHED"):
		return status.Error(codes.ResourceExhausted, errMsg)
	case contains(errMsg, "not found"):
		return status.Error(codes.NotFound, errMsg)
	case contains(errMsg, "CONFLICT"):
		return status.Error(codes.Aborted, errMsg)
	default:
		return status.Error(codes.Internal, errors.Wrap(err, "internal server error").Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
