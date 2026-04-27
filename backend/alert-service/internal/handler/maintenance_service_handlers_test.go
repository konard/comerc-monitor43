package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	api "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	domain "github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/service/maintenance"
)

// mockMaintenanceWindowService реализует MaintenanceWindowService для тестов
type mockMaintenanceWindowService struct {
	createFn  func(ctx context.Context, req maintenance.CreateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error)
	updateFn  func(ctx context.Context, req maintenance.UpdateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error)
	listFn    func(ctx context.Context, req maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error)
	deleteFn  func(ctx context.Context, id string, userID uuid.UUID) error
	cancelFn  func(ctx context.Context, id string, userID uuid.UUID, reason string) (*domain.MaintenanceWindow, error)
	historyFn func(ctx context.Context, req maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error)
}

func (m *mockMaintenanceWindowService) CreateMaintenanceWindow(ctx context.Context, req maintenance.CreateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return buildTestMW(), nil
}

func (m *mockMaintenanceWindowService) UpdateMaintenanceWindow(ctx context.Context, req maintenance.UpdateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return buildTestMW(), nil
}

func (m *mockMaintenanceWindowService) ListMaintenanceWindows(ctx context.Context, req maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return []*domain.MaintenanceWindow{buildTestMW()}, 1, nil
}

func (m *mockMaintenanceWindowService) DeleteMaintenanceWindow(ctx context.Context, id string, userID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, userID)
	}
	return nil
}

func (m *mockMaintenanceWindowService) CancelMaintenanceWindow(ctx context.Context, id string, userID uuid.UUID, reason string) (*domain.MaintenanceWindow, error) {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, id, userID, reason)
	}
	return buildTestMW(), nil
}

func (m *mockMaintenanceWindowService) GetMaintenanceWindowHistory(ctx context.Context, req maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error) {
	if m.historyFn != nil {
		return m.historyFn(ctx, req)
	}
	return []*domain.MaintenanceWindow{buildTestMW()}, 1, nil
}

// buildTestMW создаёт тестовое окно обслуживания
func buildTestMW() *domain.MaintenanceWindow {
	now := time.Now().UTC()
	return &domain.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Name:       "Test Window",
		Status:     domain.MaintenanceStatusScheduled,
		Recurrence: domain.RecurrenceTypeOnce,
		MonitorIDs: []string{"mon-1"},
		StartsAt:   now.Add(1 * time.Hour),
		EndsAt:     now.Add(2 * time.Hour),
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// ctxWithUser добавляет userID в контекст для тестов
func ctxWithUser(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), auth.UserIDKey, userID)
}

// newMaintSrv создаёт сервер с заданным моком
func newMaintSrv(svc MaintenanceWindowService) *MaintenanceWindowServiceServer {
	return NewMaintenanceWindowServiceServer(svc, auth.NewAuthMiddleware("test-secret"))
}

// --- CreateMaintenanceWindow ---

func TestMaintenanceServer_Create_Success(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	resp, err := srv.CreateMaintenanceWindow(ctx, &api.CreateMaintenanceWindowRequest{
		Name:       "Deploy",
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:    timestamppb.New(now.Add(2 * time.Hour)),
		MonitorIds: []string{"mon-1"},
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Id)
}

func TestMaintenanceServer_Create_MissingName(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	_, err := srv.CreateMaintenanceWindow(ctx, &api.CreateMaintenanceWindowRequest{
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:    timestamppb.New(now.Add(2 * time.Hour)),
		MonitorIds: []string{"mon-1"},
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Create_MissingStartTime(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	_, err := srv.CreateMaintenanceWindow(ctx, &api.CreateMaintenanceWindowRequest{
		Name:       "Deploy",
		EndTime:    timestamppb.New(now.Add(2 * time.Hour)),
		MonitorIds: []string{"mon-1"},
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Create_MissingEndTime(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	_, err := srv.CreateMaintenanceWindow(ctx, &api.CreateMaintenanceWindowRequest{
		Name:       "Deploy",
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		MonitorIds: []string{"mon-1"},
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Create_NonGlobalEmptyMonitors(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	_, err := srv.CreateMaintenanceWindow(ctx, &api.CreateMaintenanceWindowRequest{
		Name:      "Deploy",
		StartTime: timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:   timestamppb.New(now.Add(2 * time.Hour)),
		IsGlobal:  false,
		// MonitorIds is empty
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Create_GlobalNoMonitors(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	// Global windows don't require monitor IDs
	resp, err := srv.CreateMaintenanceWindow(ctx, &api.CreateMaintenanceWindowRequest{
		Name:      "Global Maintenance",
		StartTime: timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:   timestamppb.New(now.Add(2 * time.Hour)),
		IsGlobal:  true,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMaintenanceServer_Create_Unauthenticated(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	now := time.Now().UTC()

	_, err := srv.CreateMaintenanceWindow(context.Background(), &api.CreateMaintenanceWindowRequest{
		Name:       "Deploy",
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:    timestamppb.New(now.Add(2 * time.Hour)),
		MonitorIds: []string{"mon-1"},
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestMaintenanceServer_Create_ServiceError(t *testing.T) {
	svc := &mockMaintenanceWindowService{
		createFn: func(_ context.Context, _ maintenance.CreateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error) {
			return nil, domain.ErrOverlappingMaintenanceWindows
		},
	}
	srv := newMaintSrv(svc)
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	_, err := srv.CreateMaintenanceWindow(ctx, &api.CreateMaintenanceWindowRequest{
		Name:       "Deploy",
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:    timestamppb.New(now.Add(2 * time.Hour)),
		MonitorIds: []string{"mon-1"},
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

// --- UpdateMaintenanceWindow ---

func TestMaintenanceServer_Update_Success(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	id := uuid.New()

	resp, err := srv.UpdateMaintenanceWindow(ctx, &api.UpdateMaintenanceWindowRequest{
		Id:   id.String(),
		Name: "Updated Name",
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMaintenanceServer_Update_EmptyID(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	_, err := srv.UpdateMaintenanceWindow(ctx, &api.UpdateMaintenanceWindowRequest{Id: ""})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Update_InvalidID(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	_, err := srv.UpdateMaintenanceWindow(ctx, &api.UpdateMaintenanceWindowRequest{Id: "bad-uuid"})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Update_Unauthenticated(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})

	_, err := srv.UpdateMaintenanceWindow(context.Background(), &api.UpdateMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestMaintenanceServer_Update_WithTimes(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	resp, err := srv.UpdateMaintenanceWindow(ctx, &api.UpdateMaintenanceWindowRequest{
		Id:        uuid.New().String(),
		StartTime: timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:   timestamppb.New(now.Add(2 * time.Hour)),
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMaintenanceServer_Update_NotScheduled(t *testing.T) {
	svc := &mockMaintenanceWindowService{
		updateFn: func(_ context.Context, _ maintenance.UpdateMaintenanceWindowRequest) (*domain.MaintenanceWindow, error) {
			return nil, domain.ErrMaintenanceWindowNotScheduled
		},
	}
	srv := newMaintSrv(svc)
	ctx := ctxWithUser(uuid.New())

	_, err := srv.UpdateMaintenanceWindow(ctx, &api.UpdateMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

// --- ListMaintenanceWindows ---

func TestMaintenanceServer_List_Success(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	resp, err := srv.ListMaintenanceWindows(ctx, &api.ListMaintenanceWindowsRequest{
		Page:     1,
		PageSize: 10,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Total)
	assert.Len(t, resp.Windows, 1)
}

func TestMaintenanceServer_List_Unauthenticated(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})

	_, err := srv.ListMaintenanceWindows(context.Background(), &api.ListMaintenanceWindowsRequest{})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestMaintenanceServer_List_WithFilters(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	resp, err := srv.ListMaintenanceWindows(ctx, &api.ListMaintenanceWindowsRequest{
		Status:    api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE,
		StartDate: timestamppb.New(now.Add(-24 * time.Hour)),
		EndDate:   timestamppb.New(now.Add(24 * time.Hour)),
		Page:      2,
		PageSize:  5,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(2), resp.Page)
	assert.Equal(t, int32(5), resp.PageSize)
}

func TestMaintenanceServer_List_DefaultPagination(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	resp, err := srv.ListMaintenanceWindows(ctx, &api.ListMaintenanceWindowsRequest{})

	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(20), resp.PageSize)
}

func TestMaintenanceServer_List_ServiceError(t *testing.T) {
	svc := &mockMaintenanceWindowService{
		listFn: func(_ context.Context, _ maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	srv := newMaintSrv(svc)
	ctx := ctxWithUser(uuid.New())

	_, err := srv.ListMaintenanceWindows(ctx, &api.ListMaintenanceWindowsRequest{})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- DeleteMaintenanceWindow ---

func TestMaintenanceServer_Delete_Success(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	resp, err := srv.DeleteMaintenanceWindow(ctx, &api.DeleteMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMaintenanceServer_Delete_EmptyID(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	_, err := srv.DeleteMaintenanceWindow(ctx, &api.DeleteMaintenanceWindowRequest{Id: ""})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Delete_InvalidID(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	_, err := srv.DeleteMaintenanceWindow(ctx, &api.DeleteMaintenanceWindowRequest{Id: "not-uuid"})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Delete_Unauthenticated(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})

	_, err := srv.DeleteMaintenanceWindow(context.Background(), &api.DeleteMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestMaintenanceServer_Delete_NotFound(t *testing.T) {
	svc := &mockMaintenanceWindowService{
		deleteFn: func(_ context.Context, _ string, _ uuid.UUID) error {
			return domain.ErrMaintenanceWindowNotFound
		},
	}
	srv := newMaintSrv(svc)
	ctx := ctxWithUser(uuid.New())

	_, err := srv.DeleteMaintenanceWindow(ctx, &api.DeleteMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestMaintenanceServer_Delete_NotScheduled(t *testing.T) {
	svc := &mockMaintenanceWindowService{
		deleteFn: func(_ context.Context, _ string, _ uuid.UUID) error {
			return domain.ErrMaintenanceWindowNotScheduled
		},
	}
	srv := newMaintSrv(svc)
	ctx := ctxWithUser(uuid.New())

	_, err := srv.DeleteMaintenanceWindow(ctx, &api.DeleteMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

// --- CancelMaintenanceWindow ---

func TestMaintenanceServer_Cancel_Success(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	resp, err := srv.CancelMaintenanceWindow(ctx, &api.CancelMaintenanceWindowRequest{
		Id:                 uuid.New().String(),
		CancellationReason: "emergency deploy",
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMaintenanceServer_Cancel_EmptyID(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	_, err := srv.CancelMaintenanceWindow(ctx, &api.CancelMaintenanceWindowRequest{Id: ""})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Cancel_InvalidID(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	_, err := srv.CancelMaintenanceWindow(ctx, &api.CancelMaintenanceWindowRequest{Id: "bad"})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMaintenanceServer_Cancel_Unauthenticated(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})

	_, err := srv.CancelMaintenanceWindow(context.Background(), &api.CancelMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestMaintenanceServer_Cancel_Conflict(t *testing.T) {
	svc := &mockMaintenanceWindowService{
		cancelFn: func(_ context.Context, _ string, _ uuid.UUID, _ string) (*domain.MaintenanceWindow, error) {
			return nil, domain.ErrMaintenanceWindowConflict
		},
	}
	srv := newMaintSrv(svc)
	ctx := ctxWithUser(uuid.New())

	_, err := srv.CancelMaintenanceWindow(ctx, &api.CancelMaintenanceWindowRequest{
		Id: uuid.New().String(),
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Aborted, st.Code())
}

// --- GetMaintenanceWindowHistory ---

func TestMaintenanceServer_History_Success(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	resp, err := srv.GetMaintenanceWindowHistory(ctx, &api.GetMaintenanceWindowHistoryRequest{
		Page:     1,
		PageSize: 10,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Total)
	assert.Len(t, resp.Windows, 1)
}

func TestMaintenanceServer_History_Unauthenticated(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})

	_, err := srv.GetMaintenanceWindowHistory(context.Background(), &api.GetMaintenanceWindowHistoryRequest{})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestMaintenanceServer_History_WithDateFilter(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())
	now := time.Now().UTC()

	resp, err := srv.GetMaintenanceWindowHistory(ctx, &api.GetMaintenanceWindowHistoryRequest{
		StartDate: timestamppb.New(now.Add(-7 * 24 * time.Hour)),
		EndDate:   timestamppb.New(now),
		Page:      1,
		PageSize:  50,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMaintenanceServer_History_DefaultPagination(t *testing.T) {
	srv := newMaintSrv(&mockMaintenanceWindowService{})
	ctx := ctxWithUser(uuid.New())

	resp, err := srv.GetMaintenanceWindowHistory(ctx, &api.GetMaintenanceWindowHistoryRequest{})

	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(20), resp.PageSize)
}

func TestMaintenanceServer_History_ServiceError(t *testing.T) {
	svc := &mockMaintenanceWindowService{
		historyFn: func(_ context.Context, _ maintenance.ListMaintenanceWindowsRequest) ([]*domain.MaintenanceWindow, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	srv := newMaintSrv(svc)
	ctx := ctxWithUser(uuid.New())

	_, err := srv.GetMaintenanceWindowHistory(ctx, &api.GetMaintenanceWindowHistoryRequest{})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- mapMaintenanceError ---

func TestMapMaintenanceError_AllCodes(t *testing.T) {
	cases := []struct {
		err      error
		expected codes.Code
	}{
		{domain.ErrMaintenanceWindowNotFound, codes.NotFound},
		{domain.ErrMaintenanceWindowDurationExceeded, codes.InvalidArgument},
		{domain.ErrMaintenanceWindowDurationTooShort, codes.InvalidArgument},
		{domain.ErrMaintenanceWindowInPast, codes.InvalidArgument},
		{domain.ErrEndTimeBeforeStartTime, codes.InvalidArgument},
		{domain.ErrOverlappingMaintenanceWindows, codes.AlreadyExists},
		{domain.ErrMaintenanceWindowNotScheduled, codes.FailedPrecondition},
		{domain.ErrMaintenanceWindowConflict, codes.Aborted},
		{domain.ErrInsufficientPermissions, codes.PermissionDenied},
		{errors.New("generic error"), codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			mapped := mapMaintenanceError(tc.err)
			st, ok := status.FromError(mapped)
			require.True(t, ok)
			assert.Equal(t, tc.expected, st.Code())
		})
	}
}

// TestNewMaintenanceWindowServiceServer проверяет конструктор
func TestNewMaintenanceWindowServiceServer(t *testing.T) {
	srv := NewMaintenanceWindowServiceServer(&mockMaintenanceWindowService{}, auth.NewAuthMiddleware("secret"))
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.maintenanceService)
	assert.NotNil(t, srv.authMiddleware)
}
