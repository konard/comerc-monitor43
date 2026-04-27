package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
)

func mustNewMonitor(tb testing.TB, userID uuid.UUID, name, url string, intervalSeconds int) *domain.Monitor {
	tb.Helper()

	monitor, err := domain.NewMonitor(userID, name, url, intervalSeconds)
	require.NoError(tb, err)

	return monitor
}
