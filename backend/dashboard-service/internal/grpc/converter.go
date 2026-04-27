package grpc

import (
	"fmt"

	dashboardproto "github.com/raul/monitor/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "github.com/raul/monitor/backend/dashboard-service/internal/model"
)

func safeInt32(v int, field string) (int32, error) {
	if v < -2147483648 || v > 2147483647 {
		return 0, fmt.Errorf("%s out of int32 range: %d", field, v)
	}
	return int32(v), nil
}

func monitorStatusToProto(view *domain.MonitorStatusView) (*dashboardproto.MonitorStatusView, error) {
	if view == nil {
		return nil, nil
	}

	pv := &dashboardproto.MonitorStatusView{
		Id:               view.ID.String(),
		UserId:           view.UserID.String(),
		Name:             view.Name,
		Url:              view.URL,
		Status:           healthStatusToProto(view.Status),
		UptimePercentage: view.UptimePercentage,
		CreatedAt:        timestamppb.New(view.CreatedAt),
		UpdatedAt:        timestamppb.New(view.UpdatedAt),
	}

	if view.LastCheckedAt != nil {
		pv.LastCheckedAt = timestamppb.New(*view.LastCheckedAt)
	}
	if view.LastResponseTimeMs != nil {
		pv.LastResponseTimeMs = *view.LastResponseTimeMs
	}

	return pv, nil
}

func checkHistoryToProto(entry *domain.CheckHistoryEntry) (*dashboardproto.CheckHistoryEntry, error) {
	if entry == nil {
		return nil, nil
	}

	pe := &dashboardproto.CheckHistoryEntry{
		Id:        entry.ID.String(),
		MonitorId: entry.MonitorID.String(),
		Status:    checkStatusToProto(entry.Status),
		CheckedAt: timestamppb.New(entry.CheckedAt),
	}

	if entry.StatusCode != nil {
		statusCode, err := safeInt32(*entry.StatusCode, "status_code")
		if err != nil {
			return nil, err
		}
		pe.StatusCode = statusCode
	}
	if entry.ResponseTimeMs != nil {
		pe.ResponseTimeMs = *entry.ResponseTimeMs
	}
	if entry.ErrorMessage != nil {
		pe.ErrorMessage = *entry.ErrorMessage
	}

	return pe, nil
}

func incidentToProto(incident *domain.Incident) (*dashboardproto.Incident, error) {
	if incident == nil {
		return nil, nil
	}

	checkCount, err := safeInt32(incident.CheckCount, "check_count")
	if err != nil {
		return nil, err
	}

	pi := &dashboardproto.Incident{
		Id:         incident.ID.String(),
		MonitorId:  incident.MonitorID.String(),
		StartedAt:  timestamppb.New(incident.StartedAt),
		CheckCount: checkCount,
		Status:     incidentStatusToProto(incident.Status),
		CreatedAt:  timestamppb.New(incident.CreatedAt),
		UpdatedAt:  timestamppb.New(incident.UpdatedAt),
	}

	if incident.EndedAt != nil {
		pi.EndedAt = timestamppb.New(*incident.EndedAt)
	}
	if incident.DurationSeconds != nil {
		pi.DurationSeconds = *incident.DurationSeconds
	}

	return pi, nil
}

func periodMetricsToProto(metrics *domain.PeriodMetrics) *dashboardproto.PeriodMetrics {
	if metrics == nil {
		return nil
	}

	return &dashboardproto.PeriodMetrics{
		TotalChecks:       metrics.TotalChecks,
		SuccessCount:      metrics.SuccessCount,
		FailedCount:       metrics.FailedCount,
		DegradedCount:     metrics.DegradedCount,
		UptimePercentage:  metrics.UptimePercentage,
		P50ResponseTimeMs: metrics.P50ResponseMs,
		P95ResponseTimeMs: metrics.P95ResponseMs,
		P99ResponseTimeMs: metrics.P99ResponseMs,
	}
}

func healthStatusToProto(s domain.MonitorStatus) dashboardproto.HealthStatus {
	switch s {
	case domain.MonitorStatusUP:
		return dashboardproto.HealthStatus_HEALTH_STATUS_UP
	case domain.MonitorStatusDOWN:
		return dashboardproto.HealthStatus_HEALTH_STATUS_DOWN
	case domain.MonitorStatusDEGRADED:
		return dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED
	case domain.MonitorStatusPAUSED:
		return dashboardproto.HealthStatus_HEALTH_STATUS_PAUSED
	default:
		return dashboardproto.HealthStatus_HEALTH_STATUS_UNSPECIFIED
	}
}

func checkStatusToProto(s domain.CheckStatus) dashboardproto.HealthStatus {
	switch s {
	case domain.CheckStatusUP:
		return dashboardproto.HealthStatus_HEALTH_STATUS_UP
	case domain.CheckStatusDOWN:
		return dashboardproto.HealthStatus_HEALTH_STATUS_DOWN
	case domain.CheckStatusDEGRADED:
		return dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED
	default:
		return dashboardproto.HealthStatus_HEALTH_STATUS_UNSPECIFIED
	}
}

func incidentStatusToProto(s domain.IncidentStatus) dashboardproto.IncidentStatus {
	switch s {
	case domain.IncidentStatusActive:
		return dashboardproto.IncidentStatus_INCIDENT_STATUS_ACTIVE
	case domain.IncidentStatusResolved:
		return dashboardproto.IncidentStatus_INCIDENT_STATUS_RESOLVED
	default:
		return dashboardproto.IncidentStatus_INCIDENT_STATUS_UNSPECIFIED
	}
}

func protoToHealthStatus(s dashboardproto.HealthStatus) domain.MonitorStatus {
	switch s {
	case dashboardproto.HealthStatus_HEALTH_STATUS_UP:
		return domain.MonitorStatusUP
	case dashboardproto.HealthStatus_HEALTH_STATUS_DOWN:
		return domain.MonitorStatusDOWN
	case dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED:
		return domain.MonitorStatusDEGRADED
	case dashboardproto.HealthStatus_HEALTH_STATUS_PAUSED:
		return domain.MonitorStatusPAUSED
	default:
		return domain.MonitorStatusUP
	}
}

func protoToDashboardFilter(req *dashboardproto.GetDashboardRequest) domain.DashboardFilter {
	statuses := make([]domain.MonitorStatus, 0, len(req.Statuses))
	for _, s := range req.Statuses {
		statuses = append(statuses, protoToHealthStatus(s))
	}

	sortOrder := "asc"
	if req.SortOrder == dashboardproto.SortOrder_SORT_ORDER_DESC {
		sortOrder = "desc"
	}

	return domain.DashboardFilter{
		Statuses:  statuses,
		Tags:      req.Tags,
		Search:    req.Search,
		SortBy:    req.SortBy,
		SortOrder: sortOrder,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
	}
}

func protoToExportDashboardFilter(req *dashboardproto.ExportDashboardRequest) domain.DashboardFilter {
	statuses := make([]domain.MonitorStatus, 0, len(req.Statuses))
	for _, s := range req.Statuses {
		statuses = append(statuses, protoToHealthStatus(s))
	}

	return domain.DashboardFilter{
		Statuses: statuses,
		Tags:     req.Tags,
	}
}

func protoToHistoryFilter(req *dashboardproto.GetCheckHistoryRequest) domain.HistoryFilter {
	sortOrder := "asc"
	if req.SortOrder == dashboardproto.SortOrder_SORT_ORDER_DESC {
		sortOrder = "desc"
	}

	filter := domain.HistoryFilter{
		Status:    protoToCheckStatus(req.Status),
		SortBy:    req.SortBy,
		SortOrder: sortOrder,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		filter.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		filter.EndDate = &t
	}

	return filter
}

func protoToCheckStatus(s dashboardproto.HealthStatus) domain.CheckStatus {
	switch s {
	case dashboardproto.HealthStatus_HEALTH_STATUS_UP:
		return domain.CheckStatusUP
	case dashboardproto.HealthStatus_HEALTH_STATUS_DOWN:
		return domain.CheckStatusDOWN
	case dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED:
		return domain.CheckStatusDEGRADED
	default:
		return domain.CheckStatusUP
	}
}

func protoToIncidentFilter(req *dashboardproto.GetIncidentsRequest) domain.IncidentFilter {
	filter := domain.IncidentFilter{
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

	return filter
}

func protoToExportHistoryFilter(req *dashboardproto.ExportHistoryRequest) domain.HistoryFilter {
	filter := domain.HistoryFilter{
		Status: protoToCheckStatus(req.Status),
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		filter.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		filter.EndDate = &t
	}

	return filter
}
