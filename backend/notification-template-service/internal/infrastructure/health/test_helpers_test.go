package health

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

func closeTestDB(tb testing.TB, db *sqlx.DB) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := db.Close(); err != nil {
			tb.Errorf("close test db: %v", err)
		}
	})
}
