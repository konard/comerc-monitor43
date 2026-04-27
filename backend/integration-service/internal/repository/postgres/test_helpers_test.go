package postgres

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func isUnexpectedSQLMockCloseError(err error) bool {
	return err != nil &&
		strings.Contains(err.Error(), "call to database Close") &&
		strings.Contains(err.Error(), "was not expected")
}

func closeTestDB(tb testing.TB, db *DB) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := db.Close(); err != nil && !isUnexpectedSQLMockCloseError(err) {
			tb.Errorf("close test db: %v", err)
		}
	})
}

func closeSQLMockDB(tb testing.TB, db *sql.DB) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := db.Close(); err != nil && !isUnexpectedSQLMockCloseError(err) {
			tb.Errorf("close sqlmock db: %v", err)
		}
	})
}

func cleanupExecContext(ctx context.Context, tb testing.TB, db *DB, query string, args ...any) {
	tb.Helper()

	tb.Cleanup(func() {
		if _, err := db.DB.ExecContext(ctx, query, args...); err != nil {
			tb.Errorf("cleanup query: %v", err)
		}
	})
}
