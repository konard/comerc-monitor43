// Package grpc предоставляет gRPC handlers для monitor service.
//
// Пакет реализует gRPC API согласно протоколу, определённому в api/proto/monitor/v1.
package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/monitor-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// MonitorHandler реализует gRPC handlers для monitor service.
type MonitorHandler struct {
	monitov1.UnimplementedMonitorServiceServer

	monitorService   service.MonitorServiceInterface
	uptimeCalculator service.UptimeCalculatorInterface
	incidentDetector service.IncidentDetectorInterface
	validator        *validator.Validate
}

// NewMonitorHandler создаёт новый MonitorHandler.
func NewMonitorHandler(
	monitorService service.MonitorServiceInterface,
	uptimeCalculator service.UptimeCalculatorInterface,
	incidentDetector service.IncidentDetectorInterface,
) *MonitorHandler {
	return &MonitorHandler{
		monitorService:   monitorService,
		uptimeCalculator: uptimeCalculator,
		incidentDetector: incidentDetector,
		validator:        validator.New(),
	}
}

// CreateMonitor создаёт новый монитор.
func (h *MonitorHandler) CreateMonitor(ctx context.Context, req *monitov1.CreateMonitorRequest) (*monitov1.CreateMonitorResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.CreateMonitor")
	defer span.End()

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}
	userTier, err := h.extractUserTier(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	if err := h.validateStruct(req); err != nil {
		tracing.RecordError(span, err)
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	createReq := &dto.CreateMonitorRequest{
		UserID:                        userID,
		Tier:                          userTier,
		Name:                          req.Name,
		URL:                           req.Url,
		CheckType:                     req.CheckType,
		IntervalSeconds:               int(req.IntervalSeconds),
		TimeoutSeconds:                int(req.TimeoutSeconds),
		WorkingHoursStart:             req.WorkingHoursStart,
		WorkingHoursEnd:               req.WorkingHoursEnd,
		WorkingDays:                   req.WorkingDays,
		DegradedResponseTimeThreshold: intPtr(req.DegradedResponseTimeThreshold),
		DegradedFailureRateThreshold:  intPtr(req.DegradedFailureRateThreshold),
	}

	resp, err := h.monitorService.CreateMonitor(ctx, createReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &monitov1.CreateMonitorResponse{
		Monitor: h.dtoToProto(resp),
	}, nil
}

// GetMonitor возвращает информацию о мониторе.
func (h *MonitorHandler) GetMonitor(ctx context.Context, req *monitov1.GetMonitorRequest) (*monitov1.GetMonitorResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.GetMonitor")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Получаем монитор
	resp, err := h.monitorService.GetMonitor(ctx, req.Id, userID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &monitov1.GetMonitorResponse{
		Monitor: h.dtoToProto(resp),
	}, nil
}

// ListMonitors возвращает список мониторов пользователя.
func (h *MonitorHandler) ListMonitors(ctx context.Context, req *monitov1.ListMonitorsRequest) (*monitov1.ListMonitorsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.ListMonitors")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Сервисные вызовы (Role=SERVICE, например scheduler-service) получают все
	// активные мониторы без фильтра по user_id. UserID таких токенов не является UUID.
	role, _ := middleware.ExtractUserRole(ctx) //nolint:errcheck // отсутствие роли в контексте допустимо, трактуем как пользовательский вызов

	// Конвертируем в DTO
	listReq := &dto.ListMonitorsRequest{
		UserID:         userID,
		Status:         req.Status,
		Limit:          int(req.Limit),
		Offset:         int(req.Offset),
		SkipUserFilter: role == "SERVICE",
	}

	if listReq.Limit <= 0 {
		listReq.Limit = 100
	}

	// Получаем список мониторов
	resp, err := h.monitorService.ListMonitors(ctx, listReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	// Конвертируем в protobuf
	monitors := make([]*monitov1.Monitor, len(resp.Monitors))
	for i, m := range resp.Monitors {
		monitors[i] = h.dtoToProto(m)
	}

	tracing.SetSuccess(span)
	return &monitov1.ListMonitorsResponse{
		Monitors: monitors,
		Total:    int32(resp.Total),  //nolint:gosec // G115: Total — счётчик, переполнение невозможно
		Limit:    int32(resp.Limit),  //nolint:gosec // G115: Limit — параметр страницы, переполнение невозможно
		Offset:   int32(resp.Offset), //nolint:gosec // G115: Offset — смещение страницы, переполнение невозможно
	}, nil
}

// UpdateMonitor обновляет параметры монитора.
func (h *MonitorHandler) UpdateMonitor(ctx context.Context, req *monitov1.UpdateMonitorRequest) (*monitov1.UpdateMonitorResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.UpdateMonitor")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Валидация запроса
	if err := h.validateStruct(req); err != nil {
		tracing.RecordError(span, err)
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Конвертируем в DTO
	updateReq := &dto.UpdateMonitorRequest{
		ID:                            req.Id,
		UserID:                        userID,
		Name:                          stringPtr(req.Name),
		URL:                           stringPtr(req.Url),
		IntervalSeconds:               intPtr(req.IntervalSeconds),
		TimeoutSeconds:                intPtr(req.TimeoutSeconds),
		WorkingHoursStart:             stringPtr(req.WorkingHoursStart),
		WorkingHoursEnd:               stringPtr(req.WorkingHoursEnd),
		WorkingDays:                   req.WorkingDays,
		DegradedResponseTimeThreshold: intPtr(req.DegradedResponseTimeThreshold),
		DegradedFailureRateThreshold:  intPtr(req.DegradedFailureRateThreshold),
	}

	// Обновляем монитор
	resp, err := h.monitorService.UpdateMonitor(ctx, updateReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &monitov1.UpdateMonitorResponse{
		Monitor: h.dtoToProto(resp),
	}, nil
}

// DeleteMonitor удаляет монитор.
func (h *MonitorHandler) DeleteMonitor(ctx context.Context, req *monitov1.DeleteMonitorRequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.DeleteMonitor")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Удаляем монитор
	if err := h.monitorService.DeleteMonitor(ctx, req.Id, userID); err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &emptypb.Empty{}, nil
}

// PauseMonitor приостанавливает монитор.
func (h *MonitorHandler) PauseMonitor(ctx context.Context, req *monitov1.PauseMonitorRequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.PauseMonitor")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Приостанавливаем монитор
	pauseReq := &dto.PauseMonitorRequest{
		ID:     req.Id,
		UserID: userID,
	}

	if err := h.monitorService.PauseMonitor(ctx, pauseReq); err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &emptypb.Empty{}, nil
}

// ResumeMonitor возобновляет работу монитора.
func (h *MonitorHandler) ResumeMonitor(ctx context.Context, req *monitov1.ResumeMonitorRequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.ResumeMonitor")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Возобновляем монитор
	resumeReq := &dto.ResumeMonitorRequest{
		ID:     req.Id,
		UserID: userID,
	}

	if err := h.monitorService.ResumeMonitor(ctx, resumeReq); err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &emptypb.Empty{}, nil
}

// GetMonitorHistory возвращает историю проверок монитора.
func (h *MonitorHandler) GetMonitorHistory(ctx context.Context, req *monitov1.GetMonitorHistoryRequest) (*monitov1.GetMonitorHistoryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.GetMonitorHistory")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Конвертируем в DTO
	historyReq := &dto.GetMonitorHistoryRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime(),
		To:        req.To.AsTime(),
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	}

	if historyReq.Limit <= 0 {
		historyReq.Limit = 100
	}

	// Получаем историю
	resp, err := h.uptimeCalculator.GetMonitorHistory(ctx, historyReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	// Конвертируем в protobuf
	results := make([]*monitov1.CheckResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = h.checkResultDtoToProto(r)
	}

	tracing.SetSuccess(span)
	return &monitov1.GetMonitorHistoryResponse{
		Results: results,
		Total:   int32(resp.Total),  //nolint:gosec // G115: Total — счётчик, переполнение невозможно
		Limit:   int32(resp.Limit),  //nolint:gosec // G115: Limit — параметр страницы, переполнение невозможно
		Offset:  int32(resp.Offset), //nolint:gosec // G115: Offset — смещение страницы, переполнение невозможно
	}, nil
}

// GetUptimeStats возвращает статистику uptime за период.
func (h *MonitorHandler) GetUptimeStats(ctx context.Context, req *monitov1.GetUptimeStatsRequest) (*monitov1.GetUptimeStatsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.GetUptimeStats")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Валидация monitor_id
	if req.MonitorId == "" {
		err := status.Error(codes.InvalidArgument, "monitor_id is required")
		tracing.RecordError(span, err)
		return nil, err
	}

	// Дефолты периода: если from/to не заданы — последние 24 часа.
	// timestamppb.AsTime() возвращает Unix epoch (1970-01-01) для nil/zero, а не Go zero time
	from := req.From.AsTime()
	to := req.To.AsTime()
	now := time.Now().UTC()
	if req.From == nil || from.IsZero() || from.Unix() <= 0 {
		from = now.Add(-24 * time.Hour)
	}
	if req.To == nil || to.IsZero() || to.Unix() <= 0 {
		to = now
	}

	// Конвертируем в DTO
	statsReq := &dto.GetUptimeStatsRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      from,
		To:        to,
	}

	// Получаем статистику
	resp, err := h.uptimeCalculator.CalculateUptime(ctx, statsReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	slog.Default().InfoContext(ctx, "computing uptime stats",
		"monitor_id", req.MonitorId,
		"from", from,
		"to", to,
		"results_in_period", resp.TotalChecks,
	)

	tracing.SetSuccess(span)
	return &monitov1.GetUptimeStatsResponse{
		Uptime:              resp.Uptime,
		TotalChecks:         int32(resp.TotalChecks),    //nolint:gosec // G115: счётчики проверок, переполнение невозможно
		UpChecks:            int32(resp.UpChecks),       //nolint:gosec // G115
		DegradedChecks:      int32(resp.DegradedChecks), //nolint:gosec // G115
		DownChecks:          int32(resp.DownChecks),     //nolint:gosec // G115
		PausedChecks:        int32(resp.PausedChecks),   //nolint:gosec // G115
		TotalDowntime:       resp.TotalDowntime,
		AverageResponseTime: int32(resp.AverageResponseTime), //nolint:gosec // G115: среднее время ответа в мс, переполнение невозможно
		Incidents:           int32(resp.Incidents),           //nolint:gosec // G115: счётчик инцидентов, переполнение невозможно
		Note:                resp.Note,
		P50:                 int32(resp.P50), //nolint:gosec // G115: перцентиль в мс, переполнение невозможно
		P95:                 int32(resp.P95), //nolint:gosec // G115
		P99:                 int32(resp.P99), //nolint:gosec // G115
	}, nil
}

// GetIncidents возвращает инциденты монитора.
func (h *MonitorHandler) GetIncidents(ctx context.Context, req *monitov1.GetIncidentsRequest) (*monitov1.GetIncidentsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.GetIncidents")
	defer span.End()

	// Извлекаем user_id из контекста (JWT)
	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	// Конвертируем в DTO
	incidentsReq := &dto.GetIncidentsRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime(),
		To:        req.To.AsTime(),
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	}

	if incidentsReq.Limit <= 0 {
		incidentsReq.Limit = 100
	}

	// Получаем инциденты
	resp, err := h.incidentDetector.GetIncidents(ctx, incidentsReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	// Конвертируем в protobuf
	incidents := make([]*monitov1.Incident, len(resp.Incidents))
	for i, inc := range resp.Incidents {
		incidents[i] = h.incidentDtoToProto(inc)
	}

	tracing.SetSuccess(span)
	return &monitov1.GetIncidentsResponse{
		Incidents: incidents,
		Total:     int32(resp.Total),  //nolint:gosec // G115: счётчик инцидентов, переполнение невозможно
		Limit:     int32(resp.Limit),  //nolint:gosec // G115: параметр страницы, переполнение невозможно
		Offset:    int32(resp.Offset), //nolint:gosec // G115: смещение страницы, переполнение невозможно
	}, nil
}

// GetCheckResults возвращает raw check results за период для аналитики.
func (h *MonitorHandler) GetCheckResults(ctx context.Context, req *monitov1.GetCheckResultsRequest) (*monitov1.GetCheckResultsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MonitorHandler.GetCheckResults")
	defer span.End()

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	checkResultsReq := &dto.GetCheckResultsRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime(),
		To:        req.To.AsTime(),
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	}

	if checkResultsReq.Limit <= 0 {
		checkResultsReq.Limit = 100
	}

	resp, err := h.uptimeCalculator.GetCheckResults(ctx, checkResultsReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	results := make([]*monitov1.CheckResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = h.checkResultDtoToProto(r)
	}

	tracing.SetSuccess(span)
	return &monitov1.GetCheckResultsResponse{
		Results: results,
		Total:   int32(resp.Total),  //nolint:gosec // G115: счётчик, переполнение невозможно
		Limit:   int32(resp.Limit),  //nolint:gosec // G115: параметр страницы, переполнение невозможно
		Offset:  int32(resp.Offset), //nolint:gosec // G115: смещение страницы, переполнение невозможно
	}, nil
}

// Helper методы

// extractUserID извлекает user_id из контекста (JWT).
//
// NOTE: JWT токен валидируется в middleware.AuthenticateJWT(),
// который извлекает user_id/role/tier и добавляет их в контекст.
// Этот метод просто извлекает уже валидированные данные из контекста.
func (h *MonitorHandler) extractUserID(ctx context.Context) (string, error) {
	return middleware.ExtractUserID(ctx)
}

// extractUserTier извлекает user_tier из контекста (JWT).
func (h *MonitorHandler) extractUserTier(ctx context.Context) (string, error) {
	return middleware.ExtractUserTier(ctx)
}

// validateStruct валидирует структуру.
func (h *MonitorHandler) validateStruct(v any) error {
	return h.validator.Struct(v)
}

// handleError конвертирует ошибку в gRPC status.
func (h *MonitorHandler) handleError(err error) error {
	if errors.Is(err, interfaces.ErrMonitorNotFound) {
		return status.Error(codes.NotFound, "monitor not found")
	}
	return status.Error(codes.Internal, "internal server error")
}

// dtoToProto конвертирует DTO в protobuf сообщение.
func (h *MonitorHandler) dtoToProto(m *dto.MonitorResponse) *monitov1.Monitor {
	proto := &monitov1.Monitor{
		Id:              m.ID,
		UserId:          m.UserID,
		Name:            m.Name,
		Url:             m.URL,
		CheckType:       m.CheckType,
		IntervalSeconds: int32(m.IntervalSeconds), //nolint:gosec // G115: значения из БД в допустимом диапазоне
		TimeoutSeconds:  int32(m.TimeoutSeconds),  //nolint:gosec // G115: значения из БД в допустимом диапазоне
		Status:          m.Status,
		WorkingDays:     m.WorkingDays,
		CreatedAt:       timestamppb.New(m.CreatedAt),
		UpdatedAt:       timestamppb.New(m.UpdatedAt),
	}

	if m.WorkingHoursStart != nil {
		proto.WorkingHoursStart = *m.WorkingHoursStart
	}
	if m.WorkingHoursEnd != nil {
		proto.WorkingHoursEnd = *m.WorkingHoursEnd
	}
	if m.DegradedResponseTimeThreshold != nil {
		proto.DegradedResponseTimeThreshold = int32(*m.DegradedResponseTimeThreshold) //nolint:gosec // G115: порог деградации в мс, переполнение невозможно
	}
	if m.DegradedFailureRateThreshold != nil {
		proto.DegradedFailureRateThreshold = int32(*m.DegradedFailureRateThreshold) //nolint:gosec // G115: порог в процентах, переполнение невозможно
	}
	if m.LastCheckAt != nil {
		proto.LastCheckAt = timestamppb.New(*m.LastCheckAt)
	}

	return proto
}

// checkResultDtoToProto конвертирует CheckResult DTO в protobuf.
func (h *MonitorHandler) checkResultDtoToProto(r *dto.CheckResultResponse) *monitov1.CheckResult {
	proto := &monitov1.CheckResult{
		Id:        r.ID,
		MonitorId: r.MonitorID,
		Status:    r.Status,
		CheckedAt: timestamppb.New(r.CheckedAt),
		CreatedAt: timestamppb.New(r.CreatedAt),
	}

	if r.ResponseTimeMs != nil {
		proto.ResponseTimeMs = int32(*r.ResponseTimeMs) //nolint:gosec // G115: время ответа в мс, переполнение невозможно
	}
	if r.StatusCode != nil {
		proto.StatusCode = int32(*r.StatusCode) //nolint:gosec // G115: HTTP status code ≤ 599, переполнение невозможно
	}
	if r.ErrorMessage != nil {
		proto.ErrorMessage = *r.ErrorMessage
	}

	return proto
}

// incidentDtoToProto конвертирует Incident DTO в protobuf.
func (h *MonitorHandler) incidentDtoToProto(inc *dto.IncidentResponse) *monitov1.Incident {
	proto := &monitov1.Incident{
		Id:        inc.ID,
		MonitorId: inc.MonitorID,
		StartTime: timestamppb.New(inc.StartTime),
		Status:    inc.Status,
		CreatedAt: timestamppb.New(inc.CreatedAt),
	}

	if inc.EndTime != nil {
		proto.EndTime = timestamppb.New(*inc.EndTime)
	}
	if inc.DurationSeconds != nil {
		proto.DurationSeconds = int32(*inc.DurationSeconds) //nolint:gosec // G115: длительность в секундах, переполнение невозможно
	}

	return proto
}

// Helper функции для указателей

func intPtr(i int32) *int {
	if i == 0 {
		return nil
	}
	result := int(i)
	return &result
}

func int32Ptr(i int) *int32 {
	if i == 0 {
		return nil
	}
	result := int32(i) //nolint:gosec // G115: конвертация безопасных значений
	return &result
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
