package grpc

import (
	"context"
	"testing"

	maintenancepb "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
)

// TestMaintenanceHandler_DeleteMaintenanceWindow_NoAuth тестирует DeleteMaintenanceWindow без auth.
func TestMaintenanceHandler_DeleteMaintenanceWindow_NoAuth(t *testing.T) {
	t.Parallel()

	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.DeleteMaintenanceWindowRequest{
		Id: "window-id",
	}

	resp, err := handler.DeleteMaintenanceWindow(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceHandler_CancelMaintenanceWindow_NoAuth тестирует CancelMaintenanceWindow без auth.
func TestMaintenanceHandler_CancelMaintenanceWindow_NoAuth(t *testing.T) {
	t.Parallel()

	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.CancelMaintenanceWindowRequest{
		Id: "window-id",
	}

	resp, err := handler.CancelMaintenanceWindow(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceHandler_UpdateMaintenanceWindow_NoAuth тестирует UpdateMaintenanceWindow без auth.
func TestMaintenanceHandler_UpdateMaintenanceWindow_NoAuth(t *testing.T) {
	t.Parallel()

	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.UpdateMaintenanceWindowRequest{
		Id: "window-id",
	}

	resp, err := handler.UpdateMaintenanceWindow(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceHandler_GetMaintenanceWindowHistory_NoAuth тестирует GetMaintenanceWindowHistory без auth.
func TestMaintenanceHandler_GetMaintenanceWindowHistory_NoAuth(t *testing.T) {
	t.Parallel()

	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.GetMaintenanceWindowHistoryRequest{
		UserId: "user-id",
	}

	resp, err := handler.GetMaintenanceWindowHistory(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceHandler_ListMaintenanceWindows_NoAuth тестирует ListMaintenanceWindows без auth.
func TestMaintenanceHandler_ListMaintenanceWindows_NoAuth(t *testing.T) {
	t.Parallel()

	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.ListMaintenanceWindowsRequest{}

	resp, err := handler.ListMaintenanceWindows(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}
