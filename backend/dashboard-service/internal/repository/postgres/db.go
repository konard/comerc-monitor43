package postgres

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	apptelemetry "github.com/raul/monitor/backend/dashboard-service/pkg/telemetry"
)

type DB struct {
	dsn string
	*sqlx.DB
	tracer  trace.Tracer
	metrics *apptelemetry.Metrics
}

func NewDB(dsn string) *DB {
	return &DB{dsn: dsn}
}

// WrapDB оборачивает существующее sqlx.DB соединение в DB для тестов.
func WrapDB(db *sqlx.DB) *DB {
	return &DB{DB: db}
}

func (db *DB) Connect(ctx context.Context) error {
	inner, err := sqlx.Connect("postgres", db.dsn)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	db.DB = inner

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return errors.Wrap(err, "failed to ping database")
	}

	return nil
}

func (db *DB) SetObservability(tracer trace.Tracer, metrics *apptelemetry.Metrics) {
	db.tracer = tracer
	db.metrics = metrics
}

func (db *DB) Close() error {
	return errors.Wrap(db.DB.Close(), "failed to close database")
}
