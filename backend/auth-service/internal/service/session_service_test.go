package service

import (
	"context"
	"testing"

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

func TestListSessions(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		sessions := []*model.Session{
			model.NewSession(userID, "browser", "10.0.0.1", 24*60*60*1000*1000*1000),
		}

		sessionLister := mocks.NewSessionLister(t)
		sessionLister.On("GetByUserID", mock.Anything, userID, 10, 0).Return(sessions, 1, nil)

		svc := NewSessionService(sessionLister, otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.ListSessions(context.Background(), &dto.ListSessionsRequest{
			UserID: userID, Limit: 10, Offset: 0,
		})

		require.NoError(t, err)
		assert.Len(t, resp.Sessions, 1)
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		sessionLister := mocks.NewSessionLister(t)
		sessionLister.On("GetByUserID", mock.Anything, userID, 10, 0).Return(nil, 0, assert.AnError)

		svc := NewSessionService(sessionLister, otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.ListSessions(context.Background(), &dto.ListSessionsRequest{
			UserID: userID, Limit: 10, Offset: 0,
		})

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestRevokeSession(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ownerID := uuid.New()
		session := model.NewSession(ownerID, "browser", "10.0.0.1", 24*60*60*1000*1000*1000)
		sessionLister := mocks.NewSessionLister(t)
		sessionLister.On("GetByID", mock.Anything, session.ID).Return(session, nil)
		sessionLister.On("Delete", mock.Anything, session.ID).Return(nil)

		svc := NewSessionService(sessionLister, otel.GetTracerProvider().Tracer("test"))

		err := svc.RevokeSession(context.Background(), &dto.RevokeSessionRequest{
			SessionID: session.ID,
			UserID:    ownerID,
		})

		require.NoError(t, err)
	})

	t.Run("ownership_mismatch", func(t *testing.T) {
		t.Parallel()

		ownerID := uuid.New()
		otherUserID := uuid.New()
		session := model.NewSession(ownerID, "browser", "10.0.0.1", 24*60*60*1000*1000*1000)
		sessionLister := mocks.NewSessionLister(t)
		sessionLister.On("GetByID", mock.Anything, session.ID).Return(session, nil)

		svc := NewSessionService(sessionLister, otel.GetTracerProvider().Tracer("test"))

		err := svc.RevokeSession(context.Background(), &dto.RevokeSessionRequest{
			SessionID: session.ID,
			UserID:    otherUserID,
		})

		assert.Error(t, err)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeSessionNotFound, appErr.Code)
	})

	t.Run("session_not_found", func(t *testing.T) {
		t.Parallel()

		sessionLister := mocks.NewSessionLister(t)
		sessionLister.On("GetByID", mock.Anything, "nonexistent").Return(nil, assert.AnError)

		svc := NewSessionService(sessionLister, otel.GetTracerProvider().Tracer("test"))

		err := svc.RevokeSession(context.Background(), &dto.RevokeSessionRequest{
			SessionID: "nonexistent",
			UserID:    uuid.New(),
		})

		assert.Error(t, err)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeSessionNotFound, appErr.Code)
	})
}
