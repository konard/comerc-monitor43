package handler

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
)

func closeTestDB(tb testing.TB, db *sqlx.DB) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := db.Close(); err != nil {
			tb.Errorf("close test db: %v", err)
		}
	})
}

func terminateTestContainer(tb testing.TB, container testcontainers.Container) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			tb.Errorf("terminate test container: %v", err)
		}
	})
}
