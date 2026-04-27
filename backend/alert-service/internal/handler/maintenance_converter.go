package handler

import (
	api "github.com/raul/monitor/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "github.com/raul/monitor/backend/alert-service/internal/model"
)

// MaintenanceWindowToProto конвертирует domain.MaintenanceWindow в proto.MaintenanceWindow
func MaintenanceWindowToProto(mw *domain.MaintenanceWindow) *api.MaintenanceWindow {
	if mw == nil {
		return nil
	}

	proto := &api.MaintenanceWindow{
		Id:              mw.ID.String(),
		UserId:          mw.UserID.String(),
		Name:            mw.Name,
		StartTime:       timestamppb.New(mw.StartsAt),
		EndTime:         timestamppb.New(mw.EndsAt),
		Status:          maintenanceStatusToProto(mw.Status),
		Recurrence:      recurrenceTypeToProto(mw.Recurrence),
		IsGlobal:        mw.IsGlobal,
		PauseMonitoring: mw.PauseMonitoring,
		SuppressAlerts:  mw.SuppressAlerts,
		SafeMode:        mw.SafeMode,
		MonitorIds:      mw.MonitorIDs,
		CreatedAt:       timestamppb.New(mw.CreatedAt),
		UpdatedAt:       timestamppb.New(mw.UpdatedAt),
		Version:         int32(mw.Version), // #nosec G115 -- version always within int32 range
	}

	if mw.ActivatedAt != nil {
		proto.ActivatedAt = timestamppb.New(*mw.ActivatedAt)
	}
	if mw.CompletedAt != nil {
		proto.CompletedAt = timestamppb.New(*mw.CompletedAt)
	}

	return proto
}

// maintenanceStatusToProto конвертирует domain.MaintenanceWindowStatus в proto.MaintenanceWindowStatus
func maintenanceStatusToProto(s domain.MaintenanceWindowStatus) api.MaintenanceWindowStatus {
	switch s {
	case domain.MaintenanceStatusScheduled:
		return api.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED
	case domain.MaintenanceStatusActive:
		return api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE
	case domain.MaintenanceStatusCompleted:
		return api.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED
	case domain.MaintenanceStatusCancelled:
		return api.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED
	case domain.MaintenanceStatusOrphaned:
		return api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED
	default:
		return api.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED
	}
}

// protoToMaintenanceStatus конвертирует proto.MaintenanceWindowStatus в domain.MaintenanceWindowStatus
func protoToMaintenanceStatus(s api.MaintenanceWindowStatus) domain.MaintenanceWindowStatus {
	switch s {
	case api.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED:
		return domain.MaintenanceStatusScheduled
	case api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE:
		return domain.MaintenanceStatusActive
	case api.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED:
		return domain.MaintenanceStatusCompleted
	case api.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED:
		return domain.MaintenanceStatusCancelled
	case api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED:
		return domain.MaintenanceStatusOrphaned
	default:
		return ""
	}
}

// recurrenceTypeToProto конвертирует domain.RecurrenceType в proto.RecurrenceType
func recurrenceTypeToProto(r domain.RecurrenceType) api.RecurrenceType {
	switch r {
	case domain.RecurrenceTypeOnce:
		return api.RecurrenceType_RECURRENCE_TYPE_ONCE
	case domain.RecurrenceTypeDaily:
		return api.RecurrenceType_RECURRENCE_TYPE_DAILY
	case domain.RecurrenceTypeWeekly:
		return api.RecurrenceType_RECURRENCE_TYPE_WEEKLY
	case domain.RecurrenceTypeMonthly:
		return api.RecurrenceType_RECURRENCE_TYPE_MONTHLY
	default:
		return api.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED
	}
}

// protoToRecurrenceType конвертирует proto.RecurrenceType в domain.RecurrenceType
func protoToRecurrenceType(r api.RecurrenceType) domain.RecurrenceType {
	switch r {
	case api.RecurrenceType_RECURRENCE_TYPE_ONCE:
		return domain.RecurrenceTypeOnce
	case api.RecurrenceType_RECURRENCE_TYPE_DAILY:
		return domain.RecurrenceTypeDaily
	case api.RecurrenceType_RECURRENCE_TYPE_WEEKLY:
		return domain.RecurrenceTypeWeekly
	case api.RecurrenceType_RECURRENCE_TYPE_MONTHLY:
		return domain.RecurrenceTypeMonthly
	default:
		return domain.RecurrenceTypeOnce
	}
}
