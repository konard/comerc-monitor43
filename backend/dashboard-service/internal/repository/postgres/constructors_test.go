package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB_returns_instance(t *testing.T) {
	t.Parallel()

	db := NewDB("postgres://localhost/test")

	require.NotNil(t, db)
	assert.Equal(t, "postgres://localhost/test", db.dsn)
	assert.Nil(t, db.DB)
}

func TestNewMonitorStatusRepository_returns_instance(t *testing.T) {
	t.Parallel()

	db := NewDB("postgres://localhost/test")

	repo := NewMonitorStatusRepository(db)

	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestNewCheckHistoryRepository_returns_instance(t *testing.T) {
	t.Parallel()

	db := NewDB("postgres://localhost/test")

	repo := NewCheckHistoryRepository(db)

	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestNewIncidentRepository_returns_instance(t *testing.T) {
	t.Parallel()

	db := NewDB("postgres://localhost/test")

	repo := NewIncidentRepository(db)

	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestPercentileIndex_single_element(t *testing.T) {
	t.Parallel()

	idx := percentileIndex(1, 50)

	assert.Equal(t, 0, idx)
}

func TestPercentileIndex_multiple_elements(t *testing.T) {
	t.Parallel()

	cases := []struct {
		count      int
		percentile int
		wantIdx    int
	}{
		{100, 50, 50},
		{100, 95, 95},
		{100, 99, 99},
		{10, 50, 5},
		{10, 99, 9},
		{1, 99, 0},
		{2, 99, 1},
	}

	for _, tc := range cases {
		t.Run("percentile", func(t *testing.T) {
			t.Parallel()

			idx := percentileIndex(tc.count, tc.percentile)

			assert.Equal(t, tc.wantIdx, idx)
		})
	}
}

func TestPercentileIndex_clamps_to_last_element(t *testing.T) {
	t.Parallel()

	// count * percentile / 100 = 5 * 100 / 100 = 5 которое >= count(5), должно вернуть 4
	idx := percentileIndex(5, 100)

	assert.Equal(t, 4, idx)
}

func TestDB_SetObservability_does_not_panic(t *testing.T) {
	t.Parallel()

	db := NewDB("postgres://localhost/test")

	assert.NotPanics(t, func() {
		db.SetObservability(nil, nil)
	})
}
