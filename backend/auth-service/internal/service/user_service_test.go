package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/internal/service/mocks"
	apperrors "github.com/raul/monitor/backend/auth-service/pkg/errors"
)

func TestGetUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		user := &model.User{ID: userID, Email: "user@example.com", FullName: "Test User", Tier: "Free", CreatedAt: time.Now()}

		userManager := mocks.NewUserManager(t)
		userManager.On("GetByID", mock.Anything, userID).Return(user, nil)

		svc := NewUserService(userManager, mocks.NewAuditWriter(t), otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.GetUser(context.Background(), &dto.GetUserRequest{UserID: userID})

		require.NoError(t, err)
		assert.Equal(t, userID, resp.ID)
		assert.Equal(t, "user@example.com", resp.Email)
		assert.Equal(t, "Test User", resp.FullName)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		userManager := mocks.NewUserManager(t)
		userManager.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError)

		svc := NewUserService(userManager, mocks.NewAuditWriter(t), otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.GetUser(context.Background(), &dto.GetUserRequest{UserID: userID})

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeUserNotFound, appErr.Code)
	})
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		user := &model.User{ID: userID, Email: "old@example.com", FullName: "Old Name", Tier: "Free", CreatedAt: time.Now()}

		userManager := mocks.NewUserManager(t)
		userManager.On("GetByID", mock.Anything, userID).Return(user, nil)
		userManager.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		svc := NewUserService(userManager, mocks.NewAuditWriter(t), otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.UpdateUser(context.Background(), &dto.UpdateUserRequest{
			UserID: userID, Email: "new@example.com", FullName: "New Name",
		})

		require.NoError(t, err)
		assert.Equal(t, "new@example.com", resp.Email)
		assert.Equal(t, "New Name", resp.FullName)
	})

	t.Run("user_not_found", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		userManager := mocks.NewUserManager(t)
		userManager.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError)

		svc := NewUserService(userManager, mocks.NewAuditWriter(t), otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.UpdateUser(context.Background(), &dto.UpdateUserRequest{
			UserID: userID, Email: "new@example.com",
		})

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeUserNotFound, appErr.Code)
	})
}

func TestLockAccount(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		userManager := mocks.NewUserManager(t)
		userManager.On("LockAccount", mock.Anything, userID, mock.Anything).Return(nil)
		auditWriter := mocks.NewAuditWriter(t)
		auditWriter.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewUserService(userManager, auditWriter, otel.GetTracerProvider().Tracer("test"))

		err := svc.LockAccount(context.Background(), &dto.LockAccountRequest{
			UserID: userID, DurationHours: 24,
		})

		require.NoError(t, err)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		userManager := mocks.NewUserManager(t)
		userManager.On("LockAccount", mock.Anything, userID, mock.Anything).Return(assert.AnError)

		svc := NewUserService(userManager, mocks.NewAuditWriter(t), otel.GetTracerProvider().Tracer("test"))

		err := svc.LockAccount(context.Background(), &dto.LockAccountRequest{
			UserID: userID, DurationHours: 24,
		})

		assert.Error(t, err)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})
}

func TestUnlockAccount(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		userManager := mocks.NewUserManager(t)
		userManager.On("UnlockAccount", mock.Anything, userID).Return(nil)
		auditWriter := mocks.NewAuditWriter(t)
		auditWriter.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewUserService(userManager, auditWriter, otel.GetTracerProvider().Tracer("test"))

		err := svc.UnlockAccount(context.Background(), &dto.UnlockAccountRequest{UserID: userID})

		require.NoError(t, err)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		userManager := mocks.NewUserManager(t)
		userManager.On("UnlockAccount", mock.Anything, userID).Return(assert.AnError)

		svc := NewUserService(userManager, mocks.NewAuditWriter(t), otel.GetTracerProvider().Tracer("test"))

		err := svc.UnlockAccount(context.Background(), &dto.UnlockAccountRequest{UserID: userID})

		assert.Error(t, err)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})
}
