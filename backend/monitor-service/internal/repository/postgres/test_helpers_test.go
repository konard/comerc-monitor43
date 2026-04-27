package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

func closeTestDB(tb testing.TB, db *sql.DB) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := db.Close(); err != nil && !strings.Contains(err.Error(), "call to database Close was not expected") {
			tb.Errorf("close test db: %v", err)
		}
	})
}

func mustMarshalJSON(tb testing.TB, value any) []byte {
	tb.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		tb.Fatalf("marshal json: %v", err)
	}

	return data
}

func terminateTestContainer(tb testing.TB, container testcontainers.Container) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			tb.Errorf("terminate test container: %v", err)
		}
	})
}
