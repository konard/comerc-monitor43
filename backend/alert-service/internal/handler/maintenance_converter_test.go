package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"
	api "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/alert-service/internal/model"
)

// TestMaintenanceWindowToProto_ValidInput проверяет конвертацию domain.MaintenanceWindow в proto.MaintenanceWindow
func TestMaintenanceWindowToProto_ValidInput(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()
	activatedAt := now.Add(-30 * time.Minute)
	completedAt := now.Add(-5 * time.Minute)

	mw := &domain.MaintenanceWindow{
		ID:              id,
		UserID:          userID,
		Name:            "Test Maintenance",
		Status:          domain.MaintenanceStatusActive,
		Recurrence:      domain.RecurrenceTypeWeekly,
		IsGlobal:        false,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIDs:      []string{"mon-1", "mon-2"},
		StartsAt:        now.Add(-1 * time.Hour),
		EndsAt:          now.Add(1 * time.Hour),
		ActivatedAt:     &activatedAt,
		CompletedAt:     &completedAt,
		Version:         3,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	proto := MaintenanceWindowToProto(mw)

	require.NotNil(t, proto)
	assert.Equal(t, id.String(), proto.Id)
	assert.Equal(t, userID.String(), proto.UserId)
	assert.Equal(t, "Test Maintenance", proto.Name)
	assert.Equal(t, api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE, proto.Status)
	assert.Equal(t, api.RecurrenceType_RECURRENCE_TYPE_WEEKLY, proto.Recurrence)
	assert.False(t, proto.IsGlobal)
	assert.True(t, proto.PauseMonitoring)
	assert.True(t, proto.SuppressAlerts)
	assert.False(t, proto.SafeMode)
	assert.Equal(t, []string{"mon-1", "mon-2"}, proto.MonitorIds)
	assert.Equal(t, int32(3), proto.Version)
	assert.NotNil(t, proto.ActivatedAt)
	assert.NotNil(t, proto.CompletedAt)
	assert.NotNil(t, proto.CreatedAt)
	assert.NotNil(t, proto.UpdatedAt)
}

// TestMaintenanceWindowToProto_NilInput проверяет обработку nil
func TestMaintenanceWindowToProto_NilInput(t *testing.T) {
	proto := MaintenanceWindowToProto(nil)
	assert.Nil(t, proto)
}

// TestMaintenanceWindowToProto_NoOptionalTimes проверяет конвертацию без ActivatedAt и CompletedAt
func TestMaintenanceWindowToProto_NoOptionalTimes(t *testing.T) {
	now := time.Now().UTC()

	mw := &domain.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Name:       "Scheduled Window",
		Status:     domain.MaintenanceStatusScheduled,
		Recurrence: domain.RecurrenceTypeOnce,
		MonitorIDs: []string{"mon-1"},
		StartsAt:   now.Add(1 * time.Hour),
		EndsAt:     now.Add(2 * time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	proto := MaintenanceWindowToProto(mw)

	require.NotNil(t, proto)
	assert.Nil(t, proto.ActivatedAt)
	assert.Nil(t, proto.CompletedAt)
}

// TestMaintenanceStatusToProto_AllStatuses проверяет конвертацию всех статусов в proto
func TestMaintenanceStatusToProto_AllStatuses(t *testing.T) {
	cases := []struct {
		domain domain.MaintenanceWindowStatus
		proto  api.MaintenanceWindowStatus
	}{
		{domain.MaintenanceStatusScheduled, api.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED},
		{domain.MaintenanceStatusActive, api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE},
		{domain.MaintenanceStatusCompleted, api.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED},
		{domain.MaintenanceStatusCancelled, api.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED},
		{domain.MaintenanceStatusOrphaned, api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED},
		{domain.MaintenanceWindowStatus("unknown"), api.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED},
	}

	for _, tc := range cases {
		t.Run(string(tc.domain), func(t *testing.T) {
			result := maintenanceStatusToProto(tc.domain)
			assert.Equal(t, tc.proto, result)
		})
	}
}

// TestProtoToMaintenanceStatus_AllStatuses проверяет конвертацию всех proto статусов в domain
func TestProtoToMaintenanceStatus_AllStatuses(t *testing.T) {
	cases := []struct {
		proto  api.MaintenanceWindowStatus
		domain domain.MaintenanceWindowStatus
	}{
		{api.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED, domain.MaintenanceStatusScheduled},
		{api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE, domain.MaintenanceStatusActive},
		{api.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED, domain.MaintenanceStatusCompleted},
		{api.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED, domain.MaintenanceStatusCancelled},
		{api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED, domain.MaintenanceStatusOrphaned},
		{api.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED, domain.MaintenanceWindowStatus("")},
	}

	for _, tc := range cases {
		t.Run(tc.proto.String(), func(t *testing.T) {
			result := protoToMaintenanceStatus(tc.proto)
			assert.Equal(t, tc.domain, result)
		})
	}
}

// TestRecurrenceTypeToProto_AllTypes проверяет конвертацию всех типов повторения в proto
func TestRecurrenceTypeToProto_AllTypes(t *testing.T) {
	cases := []struct {
		domain domain.RecurrenceType
		proto  api.RecurrenceType
	}{
		{domain.RecurrenceTypeOnce, api.RecurrenceType_RECURRENCE_TYPE_ONCE},
		{domain.RecurrenceTypeDaily, api.RecurrenceType_RECURRENCE_TYPE_DAILY},
		{domain.RecurrenceTypeWeekly, api.RecurrenceType_RECURRENCE_TYPE_WEEKLY},
		{domain.RecurrenceTypeMonthly, api.RecurrenceType_RECURRENCE_TYPE_MONTHLY},
		{domain.RecurrenceType("unknown"), api.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED},
	}

	for _, tc := range cases {
		t.Run(string(tc.domain), func(t *testing.T) {
			result := recurrenceTypeToProto(tc.domain)
			assert.Equal(t, tc.proto, result)
		})
	}
}

// TestProtoToRecurrenceType_AllTypes проверяет конвертацию всех proto типов повторения в domain
func TestProtoToRecurrenceType_AllTypes(t *testing.T) {
	cases := []struct {
		proto  api.RecurrenceType
		domain domain.RecurrenceType
	}{
		{api.RecurrenceType_RECURRENCE_TYPE_ONCE, domain.RecurrenceTypeOnce},
		{api.RecurrenceType_RECURRENCE_TYPE_DAILY, domain.RecurrenceTypeDaily},
		{api.RecurrenceType_RECURRENCE_TYPE_WEEKLY, domain.RecurrenceTypeWeekly},
		{api.RecurrenceType_RECURRENCE_TYPE_MONTHLY, domain.RecurrenceTypeMonthly},
		{api.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED, domain.RecurrenceTypeOnce},
	}

	for _, tc := range cases {
		t.Run(tc.proto.String(), func(t *testing.T) {
			result := protoToRecurrenceType(tc.proto)
			assert.Equal(t, tc.domain, result)
		})
	}
}
