package handler

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

func TestErrorToStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: codes.OK,
		},
		{
			name:         "webhook not found",
			err:          model.ErrWebhookNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "api key not found",
			err:          model.ErrAPIKeyNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "import not found",
			err:          model.ErrImportNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "invalid uuid",
			err:          model.ErrInvalidUUID,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "unauthorized",
			err:          model.ErrUnauthorized,
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "forbidden",
			err:          model.ErrForbidden,
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "webhook duplicate name",
			err:          model.ErrWebhookDuplicateName,
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "api key duplicate name",
			err:          model.ErrAPIKeyDuplicateName,
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "webhook invalid url",
			err:          model.ErrWebhookInvalidURL,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "webhook injection",
			err:          model.ErrWebhookInjection,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "webhook invalid method",
			err:          model.ErrWebhookInvalidMethod,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "empty webhook name",
			err:          model.ErrEmptyWebhookName,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "webhook name too long",
			err:          model.ErrWebhookNameTooLong,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "webhook payload too large",
			err:          model.ErrWebhookPayloadTooLarge,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "empty api key name",
			err:          model.ErrEmptyAPIKeyName,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "api key name too long",
			err:          model.ErrAPIKeyNameTooLong,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "invalid scope",
			err:          model.ErrInvalidScope,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "insufficient scope",
			err:          model.ErrInsufficientScope,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "ip not allowed",
			err:          model.ErrIPNotAllowed,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "invalid import format",
			err:          model.ErrInvalidImportFormat,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "import validation failed",
			err:          model.ErrImportValidationFailed,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "unsupported import source",
			err:          model.ErrUnsupportedImportSource,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "webhook limit reached",
			err:          model.ErrWebhookLimitReached,
			expectedCode: codes.ResourceExhausted,
		},
		{
			name:         "api key limit reached",
			err:          model.ErrAPIKeyLimitReached,
			expectedCode: codes.ResourceExhausted,
		},
		{
			name:         "monitor limit exceeded",
			err:          model.ErrMonitorLimitExceeded,
			expectedCode: codes.ResourceExhausted,
		},
		{
			name:         "rate limit exceeded",
			err:          model.ErrRateLimitExceeded,
			expectedCode: codes.ResourceExhausted,
		},
		{
			name:         "database write failed",
			err:          model.ErrDatabaseWriteFailed,
			expectedCode: codes.Internal,
		},
		{
			name:         "internal error",
			err:          model.ErrInternal,
			expectedCode: codes.Internal,
		},
		{
			name:         "unknown error",
			err:          errors.New("some unknown error"),
			expectedCode: codes.Internal,
		},
		{
			name:         "wrapped webhook not found",
			err:          errors.Wrap(model.ErrWebhookNotFound, "failed to get webhook"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "wrapped api key not found",
			err:          errors.Wrap(model.ErrAPIKeyNotFound, "context"),
			expectedCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := errorToStatus(tt.err)

			if tt.err == nil {
				assert.NoError(t, result)
				return
			}

			st, ok := status.FromError(result)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedCode, st.Code())
		})
	}
}

func TestInvalidArgument(t *testing.T) {
	t.Parallel()

	err := invalidArgument("user_id", "invalid UUID format")
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "user_id")
	assert.Contains(t, st.Message(), "invalid UUID format")
}
