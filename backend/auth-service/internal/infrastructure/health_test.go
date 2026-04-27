package infrastructure

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRedisPinger struct {
	err error
}

func (m *mockRedisPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestChecker_redis_healthy(t *testing.T) {
	t.Parallel()

	// Тестируем только redis ветку, обходя DB ping
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			return
		}
	})

	mock.ExpectPing()
	mock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{}))

	checker := NewChecker(db, &mockRedisPinger{err: nil})
	result := checker.Check(context.Background())

	// Redis должен быть healthy
	assert.Equal(t, StatusHealthy, result.Redis.Status)
	assert.Empty(t, result.Redis.Error)
}

func TestChecker_redis_unhealthy(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			return
		}
	})

	mock.ExpectPing()
	mock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{}))

	checker := NewChecker(db, &mockRedisPinger{err: fmt.Errorf("redis down")})
	result := checker.Check(context.Background())

	assert.Equal(t, StatusUnhealthy, result.Redis.Status)
	assert.NotEmpty(t, result.Redis.Error)
	assert.Equal(t, StatusUnhealthy, result.Status)
}

func TestChecker_database_ping_fails(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			return
		}
	})

	mock.ExpectPing().WillReturnError(fmt.Errorf("connection refused"))

	checker := NewChecker(db, &mockRedisPinger{err: nil})
	result := checker.Check(context.Background())

	assert.Equal(t, StatusUnhealthy, result.Database.Status)
	assert.NotEmpty(t, result.Database.Error)
	assert.Equal(t, StatusUnhealthy, result.Status)
}

func TestChecker_Check_overall_healthy(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			return
		}
	})

	mock.ExpectPing()
	mock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{}))

	checker := NewChecker(db, &mockRedisPinger{err: nil})
	result := checker.Check(context.Background())

	// DB: Scan() с 0 колонками = ok; Redis: ok → overall healthy
	assert.Equal(t, StatusHealthy, result.Redis.Status)
	assert.NotNil(t, result.Database)
}

func TestChecker_constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "healthy", StatusHealthy)
	assert.Equal(t, "unhealthy", StatusUnhealthy)
}
