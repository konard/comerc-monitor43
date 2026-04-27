package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractUserID_WithValue(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	got := ExtractUserID(ctx)
	assert.Equal(t, userID, got)
}

func TestExtractUserID_Nil(t *testing.T) {
	t.Parallel()

	got := ExtractUserID(context.Background())
	assert.Equal(t, uuid.Nil, got)
}

func TestExtractUserID_WrongType(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), UserIDKey, "not-a-uuid")
	got := ExtractUserID(ctx)
	assert.Equal(t, uuid.Nil, got)
}

func TestGetUserID_WithValue(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	got, err := GetUserID(ctx)
	require.NoError(t, err)
	assert.Equal(t, userID, got)
}

func TestGetUserID_Nil(t *testing.T) {
	t.Parallel()

	_, err := GetUserID(context.Background())
	require.Error(t, err)
}

func TestGetUserID_WrongType(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), UserIDKey, 42)
	_, err := GetUserID(ctx)
	require.Error(t, err)
}
