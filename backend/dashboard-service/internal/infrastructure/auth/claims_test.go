package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetUserID_returns_user_id(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, id)

	result, err := GetUserID(ctx)

	require.NoError(t, err)
	assert.Equal(t, id, result)
}

func TestGetUserID_missing_returns_error(t *testing.T) {
	t.Parallel()

	result, err := GetUserID(context.Background())

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestExtractUserID_present(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, id)

	result := ExtractUserID(ctx)

	assert.Equal(t, id, result)
}

func TestExtractUserID_missing_returns_nil(t *testing.T) {
	t.Parallel()

	result := ExtractUserID(context.Background())

	assert.Equal(t, uuid.Nil, result)
}

func TestWrappedStream_Context(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, id)
	ws := &wrappedStream{ctx: ctx}

	result := ws.Context()

	assert.Equal(t, ctx, result)
	assert.Equal(t, id, result.Value(UserIDKey))
}
