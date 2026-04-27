package maintenance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
)

// mockMWRepo реализует repository.MaintenanceWindowRepository для тестов
type mockMWRepo struct {
	mock.Mock
}

func (m *mockMWRepo) Create(ctx context.Context, mw *model.MaintenanceWindow) error {
	args := m.Called(ctx, mw)
	return args.Error(0)
}

func (m *mockMWRepo) GetByID(ctx context.Context, id string) (*model.MaintenanceWindow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MaintenanceWindow), args.Error(1)
}

func (m *mockMWRepo) GetActiveByMonitorID(ctx context.Context, monitorID string) (*model.MaintenanceWindow, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MaintenanceWindow), args.Error(1)
}

func (m *mockMWRepo) IsUnderMaintenance(ctx context.Context, monitorID string) (bool, error) {
	args := m.Called(ctx, monitorID)
	return args.Bool(0), args.Error(1)
}

func (m *mockMWRepo) Update(ctx context.Context, mw *model.MaintenanceWindow) error {
	args := m.Called(ctx, mw)
	return args.Error(0)
}

func (m *mockMWRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockMWRepo) List(ctx context.Context, userID string, filter model.MaintenanceWindowFilter) ([]*model.MaintenanceWindow, int, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*model.MaintenanceWindow), args.Int(1), args.Error(2)
}

func (m *mockMWRepo) CheckOverlapping(ctx context.Context, monitorIDs []string, startsAt, endsAt time.Time, excludeID string) (bool, *model.MaintenanceWindow, error) {
	args := m.Called(ctx, monitorIDs, startsAt, endsAt, excludeID)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*model.MaintenanceWindow), args.Error(2)
}

// compile-time check
var _ repository.MaintenanceWindowRepository = (*mockMWRepo)(nil)

func newTestMaintenanceService(repo repository.MaintenanceWindowRepository) *MaintenanceService {
	tracer := otel.GetTracerProvider().Tracer("test")
	return NewMaintenanceService(repo, tracer, nil)
}

// --- Тесты CreateMaintenanceWindow ---

func TestCreateMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)
	monitorIDs := []string{uuid.New().String()}

	repo.On("CheckOverlapping", mock.Anything, monitorIDs, startsAt, endsAt, "").Return(false, nil, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)

	mw, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:     uuid.New(),
		UserRole:   "USER",
		Name:       "Weekly Maintenance",
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		Recurrence: model.RecurrenceTypeOnce,
		MonitorIDs: monitorIDs,
	})

	assert.NoError(t, err)
	assert.NotNil(t, mw)
	assert.Equal(t, model.MaintenanceStatusScheduled, mw.Status)
	assert.Equal(t, "Weekly Maintenance", mw.Name)
	repo.AssertExpectations(t)
}

func TestCreateMaintenanceWindow_DurationExceeded(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(25 * time.Hour)

	_, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:     uuid.New(),
		UserRole:   "USER",
		Name:       "Too Long",
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		MonitorIDs: []string{uuid.New().String()},
	})

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowDurationExceeded)
}

func TestCreateMaintenanceWindow_DurationTooShort(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(30 * time.Second)

	_, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:     uuid.New(),
		UserRole:   "USER",
		Name:       "Too Short",
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		MonitorIDs: []string{uuid.New().String()},
	})

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowDurationTooShort)
}

func TestCreateMaintenanceWindow_EndBeforeStart(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(2 * time.Hour)
	endsAt := time.Now().Add(1 * time.Hour)

	_, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:     uuid.New(),
		UserRole:   "USER",
		Name:       "Invalid Range",
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		MonitorIDs: []string{uuid.New().String()},
	})

	assert.ErrorIs(t, err, model.ErrEndTimeBeforeStartTime)
}

func TestCreateMaintenanceWindow_GlobalForbiddenForUser(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)

	_, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:   uuid.New(),
		UserRole: "USER",
		Name:     "Global Window",
		StartsAt: startsAt,
		EndsAt:   endsAt,
		IsGlobal: true,
	})

	assert.ErrorIs(t, err, model.ErrInsufficientPermissions)
}

func TestCreateMaintenanceWindow_GlobalAllowedForAdmin(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)

	mw, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:   uuid.New(),
		UserRole: "ADMIN",
		Name:     "Global Window",
		StartsAt: startsAt,
		EndsAt:   endsAt,
		IsGlobal: true,
	})

	assert.NoError(t, err)
	assert.True(t, mw.IsGlobal)
	repo.AssertExpectations(t)
}

func TestCreateMaintenanceWindow_Overlapping(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)
	monitorIDs := []string{uuid.New().String()}

	existing := &model.MaintenanceWindow{ID: uuid.New()}
	repo.On("CheckOverlapping", mock.Anything, monitorIDs, startsAt, endsAt, "").Return(true, existing, nil)

	_, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:     uuid.New(),
		UserRole:   "USER",
		Name:       "Overlapping",
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		MonitorIDs: monitorIDs,
	})

	assert.ErrorIs(t, err, model.ErrOverlappingMaintenanceWindows)
	repo.AssertExpectations(t)
}

// --- Тесты UpdateMaintenanceWindow ---

func TestUpdateMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()
	startsAt := time.Now().Add(2 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)

	existing := &model.MaintenanceWindow{
		ID:       windowID,
		Name:     "Old Name",
		Status:   model.MaintenanceStatusScheduled,
		StartsAt: startsAt,
		EndsAt:   endsAt,
		Version:  0,
	}
	updated := &model.MaintenanceWindow{
		ID:       windowID,
		Name:     "New Name",
		Status:   model.MaintenanceStatusScheduled,
		StartsAt: startsAt,
		EndsAt:   endsAt,
		Version:  1,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil).Once()
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)
	repo.On("GetByID", mock.Anything, windowID.String()).Return(updated, nil).Once()

	mw, err := svc.UpdateMaintenanceWindow(ctx, UpdateMaintenanceWindowRequest{
		ID:      windowID.String(),
		UserID:  uuid.New(),
		Name:    "New Name",
		Version: 0,
	})

	assert.NoError(t, err)
	assert.Equal(t, "New Name", mw.Name)
	repo.AssertExpectations(t)
}

func TestUpdateMaintenanceWindow_NotScheduled(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Status: model.MaintenanceStatusActive,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil)

	_, err := svc.UpdateMaintenanceWindow(ctx, UpdateMaintenanceWindowRequest{
		ID:      windowID.String(),
		UserID:  uuid.New(),
		Version: 0,
	})

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowNotScheduled)
}

func TestUpdateMaintenanceWindow_Conflict(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()
	startsAt := time.Now().Add(2 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)

	existing := &model.MaintenanceWindow{
		ID:       windowID,
		Status:   model.MaintenanceStatusScheduled,
		StartsAt: startsAt,
		EndsAt:   endsAt,
		Version:  0,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(model.ErrMaintenanceWindowConflict)

	_, err := svc.UpdateMaintenanceWindow(ctx, UpdateMaintenanceWindowRequest{
		ID:      windowID.String(),
		UserID:  uuid.New(),
		Version: 0,
	})

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowConflict)
}

// --- Тесты DeleteMaintenanceWindow ---

func TestDeleteMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Name:   "Planned",
		Status: model.MaintenanceStatusScheduled,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil)
	repo.On("Delete", mock.Anything, windowID.String()).Return(nil)

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), uuid.New())

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteMaintenanceWindow_NotScheduled(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Status: model.MaintenanceStatusCompleted,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil)

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), uuid.New())

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowNotScheduled)
}

// --- Тесты CancelMaintenanceWindow ---

func TestCancelMaintenanceWindow_FromActive(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Name:   "Active Window",
		Status: model.MaintenanceStatusActive,
	}
	cancelled := &model.MaintenanceWindow{
		ID:     windowID,
		Name:   "Active Window",
		Status: model.MaintenanceStatusCancelled,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil).Once()
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)
	repo.On("GetByID", mock.Anything, windowID.String()).Return(cancelled, nil).Once()

	mw, err := svc.CancelMaintenanceWindow(ctx, windowID.String(), uuid.New(), "emergency fix")

	assert.NoError(t, err)
	assert.Equal(t, model.MaintenanceStatusCancelled, mw.Status)
	repo.AssertExpectations(t)
}

// --- Тесты ListMaintenanceWindows ---

func TestListMaintenanceWindows_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	userID := uuid.New()

	windows := []*model.MaintenanceWindow{
		{ID: uuid.New(), Name: "Window 1", Status: model.MaintenanceStatusScheduled},
		{ID: uuid.New(), Name: "Window 2", Status: model.MaintenanceStatusScheduled},
	}
	filter := model.MaintenanceWindowFilter{Page: 1, PageSize: 20}

	repo.On("List", mock.Anything, userID.String(), filter).Return(windows, 2, nil)

	result, total, err := svc.ListMaintenanceWindows(ctx, ListMaintenanceWindowsRequest{
		UserID: userID,
		Filter: filter,
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

// --- Backward-compat тесты ---

func TestIsUnderMaintenance_ActiveWindow(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	repo.On("IsUnderMaintenance", mock.Anything, "mon-123").Return(true, nil)

	underMaintenance, err := svc.IsUnderMaintenance(context.Background(), "mon-123")

	assert.NoError(t, err)
	assert.True(t, underMaintenance)
}

func TestIsUnderMaintenance_NoWindow(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	repo.On("IsUnderMaintenance", mock.Anything, "mon-123").Return(false, nil)

	underMaintenance, err := svc.IsUnderMaintenance(context.Background(), "mon-123")

	assert.NoError(t, err)
	assert.False(t, underMaintenance)
}

func TestCreateWindow_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := time.Now().Add(2 * time.Hour)
	reason := "planned upgrade"

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)

	err := svc.CreateWindow(ctx, userID, monitorID, startsAt, endsAt, &reason)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetActiveWindow_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	monitorID := "mon-123"
	expected := &model.MaintenanceWindow{ID: uuid.New()}

	repo.On("GetActiveByMonitorID", mock.Anything, monitorID).Return(expected, nil)

	mw, err := svc.GetActiveWindow(context.Background(), monitorID)

	assert.NoError(t, err)
	assert.Equal(t, expected, mw)
}

// --- Тесты GetMaintenanceWindowHistory ---

func TestGetMaintenanceWindowHistory_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	userID := uuid.New()

	windows := []*model.MaintenanceWindow{
		{ID: uuid.New(), Status: model.MaintenanceStatusCompleted},
	}
	filter := model.MaintenanceWindowFilter{
		Status:   model.MaintenanceStatusCompleted,
		Page:     1,
		PageSize: 20,
	}

	repo.On("List", mock.Anything, userID.String(), filter).Return(windows, 1, nil)

	result, total, err := svc.GetMaintenanceWindowHistory(ctx, ListMaintenanceWindowsRequest{
		UserID: userID,
		Filter: filter,
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

func TestGetMaintenanceWindowHistory_DefaultStatus(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	userID := uuid.New()

	// Если статус не задан, должен использоваться MaintenanceStatusCompleted
	expectedFilter := model.MaintenanceWindowFilter{
		Status: model.MaintenanceStatusCompleted,
	}

	repo.On("List", mock.Anything, userID.String(), expectedFilter).Return([]*model.MaintenanceWindow{}, 0, nil)

	_, _, err := svc.GetMaintenanceWindowHistory(ctx, ListMaintenanceWindowsRequest{
		UserID: userID,
		Filter: model.MaintenanceWindowFilter{},
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetMaintenanceWindowHistory_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	userID := uuid.New()

	filter := model.MaintenanceWindowFilter{Status: model.MaintenanceStatusCompleted}
	repo.On("List", mock.Anything, userID.String(), filter).Return(nil, 0, assert.AnError)

	_, _, err := svc.GetMaintenanceWindowHistory(ctx, ListMaintenanceWindowsRequest{
		UserID: userID,
		Filter: filter,
	})

	assert.Error(t, err)
}

// --- Дополнительные тесты покрытия ошибочных путей ---

func TestListMaintenanceWindows_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	userID := uuid.New()
	filter := model.MaintenanceWindowFilter{}

	repo.On("List", mock.Anything, userID.String(), filter).Return(nil, 0, assert.AnError)

	_, _, err := svc.ListMaintenanceWindows(ctx, ListMaintenanceWindowsRequest{
		UserID: userID,
		Filter: filter,
	})

	assert.Error(t, err)
}

func TestDeleteMaintenanceWindow_GetByIDError(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	repo.On("GetByID", mock.Anything, windowID.String()).Return(nil, model.ErrMaintenanceWindowNotFound)

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), uuid.New())

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowNotFound)
}

func TestDeleteMaintenanceWindow_DeleteError(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Name:   "Planned",
		Status: model.MaintenanceStatusScheduled,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil)
	repo.On("Delete", mock.Anything, windowID.String()).Return(assert.AnError)

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), uuid.New())

	assert.Error(t, err)
}

func TestCancelMaintenanceWindow_GetByIDError(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	repo.On("GetByID", mock.Anything, windowID.String()).Return(nil, model.ErrMaintenanceWindowNotFound)

	_, err := svc.CancelMaintenanceWindow(ctx, windowID.String(), uuid.New(), "")

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowNotFound)
}

func TestCancelMaintenanceWindow_InvalidStatus(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Status: model.MaintenanceStatusCompleted,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil)

	_, err := svc.CancelMaintenanceWindow(ctx, windowID.String(), uuid.New(), "")

	assert.ErrorIs(t, err, model.ErrMaintenanceWindowNotScheduled)
}

func TestCancelMaintenanceWindow_UpdateError(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Status: model.MaintenanceStatusScheduled,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(assert.AnError)

	_, err := svc.CancelMaintenanceWindow(ctx, windowID.String(), uuid.New(), "")

	assert.Error(t, err)
}

func TestCancelMaintenanceWindow_ReloadError(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()

	existing := &model.MaintenanceWindow{
		ID:     windowID,
		Status: model.MaintenanceStatusScheduled,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil).Once()
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)
	repo.On("GetByID", mock.Anything, windowID.String()).Return(nil, assert.AnError).Once()

	_, err := svc.CancelMaintenanceWindow(ctx, windowID.String(), uuid.New(), "")

	assert.Error(t, err)
}

func TestIsUnderMaintenance_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	repo.On("IsUnderMaintenance", mock.Anything, "mon-err").Return(false, assert.AnError)

	_, err := svc.IsUnderMaintenance(context.Background(), "mon-err")

	assert.Error(t, err)
}

func TestGetActiveWindow_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	repo.On("GetActiveByMonitorID", mock.Anything, "mon-err").Return(nil, assert.AnError)

	_, err := svc.GetActiveWindow(context.Background(), "mon-err")

	assert.Error(t, err)
}

func TestWithAuditService(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	assert.Nil(t, svc.auditService)

	result := svc.WithAuditService(nil)

	assert.Same(t, svc, result)
}

func TestUpdateMaintenanceWindow_ReloadError(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	windowID := uuid.New()
	startsAt := time.Now().Add(2 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)

	existing := &model.MaintenanceWindow{
		ID:       windowID,
		Status:   model.MaintenanceStatusScheduled,
		StartsAt: startsAt,
		EndsAt:   endsAt,
		Version:  0,
	}

	repo.On("GetByID", mock.Anything, windowID.String()).Return(existing, nil).Once()
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)
	repo.On("GetByID", mock.Anything, windowID.String()).Return(nil, assert.AnError).Once()

	_, err := svc.UpdateMaintenanceWindow(ctx, UpdateMaintenanceWindowRequest{
		ID:      windowID.String(),
		UserID:  uuid.New(),
		Version: 0,
	})

	assert.Error(t, err)
}

func TestCreateMaintenanceWindow_OverlapCheckError(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)
	monitorIDs := []string{uuid.New().String()}

	repo.On("CheckOverlapping", mock.Anything, monitorIDs, startsAt, endsAt, "").Return(false, nil, assert.AnError)

	_, err := svc.CreateMaintenanceWindow(ctx, CreateMaintenanceWindowRequest{
		UserID:     uuid.New(),
		UserRole:   "USER",
		Name:       "Window",
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		MonitorIDs: monitorIDs,
	})

	assert.Error(t, err)
}

func TestCreateWindow_NilReason(t *testing.T) {
	t.Parallel()

	repo := &mockMWRepo{}
	svc := newTestMaintenanceService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := time.Now().Add(2 * time.Hour)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MaintenanceWindow")).Return(nil)

	err := svc.CreateWindow(ctx, userID, monitorID, startsAt, endsAt, nil)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
