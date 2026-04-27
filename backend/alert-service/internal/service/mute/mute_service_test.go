package mute

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
)

type mockMuteRepo struct {
	mock.Mock
}

func (m *mockMuteRepo) Create(ctx context.Context, mute *model.AlertMute) error {
	args := m.Called(ctx, mock.Anything)
	return args.Error(0)
}

func (m *mockMuteRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockMuteRepo) GetActiveByMonitorID(ctx context.Context, monitorID string) (*model.AlertMute, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertMute), args.Error(1)
}

func (m *mockMuteRepo) GetActiveByUserID(ctx context.Context, userID string) ([]*model.AlertMute, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return []*model.AlertMute{}, args.Error(1)
	}
	return args.Get(0).([]*model.AlertMute), args.Error(1)
}

func (m *mockMuteRepo) IsMuted(ctx context.Context, userID, monitorID string) (bool, error) {
	args := m.Called(ctx, userID, monitorID)
	return args.Bool(0), args.Error(1)
}

func (m *mockMuteRepo) DeleteByUserIDAndMonitorID(ctx context.Context, userID, monitorID string) error {
	args := m.Called(ctx, userID, monitorID)
	return args.Error(0)
}

func (m *mockMuteRepo) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func newTestMuteService(repo repository.AlertMuteRepository) *MuteService {
	tracer := otel.GetTracerProvider().Tracer("test")
	return NewMuteService(repo, nil, tracer, nil)
}

func TestMuteForUser_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.MuteForUser(ctx, userID, monitorID, nil)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestIsMuted_ActiveMute(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	repo.On("IsMuted", mock.Anything, userID.String(), monitorID.String()).Return(true, nil)

	muted, err := service.IsMuted(ctx, userID.String(), monitorID.String())

	assert.NoError(t, err)
	assert.True(t, muted)
}

func TestIsMuted_NoMute(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	repo.On("IsMuted", mock.Anything, userID.String(), monitorID.String()).Return(false, nil)

	muted, err := service.IsMuted(ctx, userID.String(), monitorID.String())

	assert.NoError(t, err)
	assert.False(t, muted)
}

func TestUnmute_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	muteID := uuid.New().String()

	repo.On("Delete", mock.Anything, muteID).Return(nil)

	err := service.Unmute(ctx, muteID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUnmuteForUser_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	repo.On("DeleteByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(nil)

	err := service.UnmuteForUser(ctx, userID, monitorID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMuteGlobally_AdminSuccess(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.WithValue(context.Background(), auth.RoleKey, auth.RoleAdmin)
	adminID := uuid.New()
	monitorID := uuid.New()

	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.MuteGlobally(ctx, adminID, monitorID, nil)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMuteGlobally_NonAdminForbidden(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.WithValue(context.Background(), auth.RoleKey, auth.RoleUser)
	userID := uuid.New()
	monitorID := uuid.New()

	err := service.MuteGlobally(ctx, userID, monitorID, nil)

	assert.ErrorIs(t, err, model.ErrForbidden)
	repo.AssertNotCalled(t, "Create")
}

func TestMuteForUser_TimedMute(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()
	mutedUntil := time.Now().Add(1 * time.Hour)

	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.MuteForUser(ctx, userID, monitorID, &mutedUntil)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMuteForUser_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	repo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	err := service.MuteForUser(ctx, userID, monitorID, nil)

	assert.Error(t, err)
}

func TestMuteGlobally_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.WithValue(context.Background(), auth.RoleKey, auth.RoleAdmin)
	adminID := uuid.New()
	monitorID := uuid.New()

	repo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	err := service.MuteGlobally(ctx, adminID, monitorID, nil)

	assert.Error(t, err)
}

func TestUnmute_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	muteID := uuid.New().String()

	repo.On("Delete", mock.Anything, muteID).Return(assert.AnError)

	err := service.Unmute(ctx, muteID)

	assert.Error(t, err)
}

func TestUnmuteForUser_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	repo.On("DeleteByUserIDAndMonitorID", mock.Anything, userID.String(), monitorID.String()).Return(assert.AnError)

	err := service.UnmuteForUser(ctx, userID, monitorID)

	assert.Error(t, err)
}

func TestIsMuted_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	repo.On("IsMuted", mock.Anything, userID.String(), monitorID.String()).Return(false, assert.AnError)

	_, err := service.IsMuted(ctx, userID.String(), monitorID.String())

	assert.Error(t, err)
}

func TestProcessExpiredMutes(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	repo.On("DeleteExpired", mock.Anything).Return(int64(0), nil)

	err := service.ProcessExpiredMutes(context.Background())

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetActiveMutes_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New().String()
	expected := []*model.AlertMute{{ID: uuid.New()}}

	repo.On("GetActiveByUserID", mock.Anything, userID).Return(expected, nil)

	mutes, err := service.GetActiveMutes(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expected, mutes)
}

func TestGetActiveMutes_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	userID := uuid.New().String()

	repo.On("GetActiveByUserID", mock.Anything, userID).Return(nil, assert.AnError)

	_, err := service.GetActiveMutes(ctx, userID)

	assert.Error(t, err)
}

func TestProcessExpiredMutes_Success(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	repo.On("DeleteExpired", mock.Anything).Return(int64(3), nil)

	err := service.ProcessExpiredMutes(ctx)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestProcessExpiredMutes_Error(t *testing.T) {
	t.Parallel()

	repo := &mockMuteRepo{}
	service := newTestMuteService(repo)

	ctx := context.Background()
	repo.On("DeleteExpired", mock.Anything).Return(int64(0), assert.AnError)

	err := service.ProcessExpiredMutes(ctx)
	assert.Error(t, err)
}
