package middleware

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/auth"
)

func TestExtractUserID(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)

		extractedID, err := ExtractUserID(ctx)

		require.NoError(t, err)
		assert.Equal(t, userID.String(), extractedID)
	})

	t.Run("error_not_authenticated", func(t *testing.T) {
		t.Parallel()

		extractedID, err := ExtractUserID(context.Background())

		require.Error(t, err)
		assert.Equal(t, "", extractedID)
	})
}
