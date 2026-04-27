package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/reporting-service/internal/config"
)

type DB struct {
	*sqlx.DB
	cfg config.DatabaseConfig
}

func NewDB(cfg config.DatabaseConfig) *DB {
	return &DB{cfg: cfg}
}

func (db *DB) Connect(ctx context.Context) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		db.cfg.Host, db.cfg.Port, db.cfg.User, db.cfg.Password, db.cfg.Name, db.cfg.SSLMode,
	)

	sqlDB, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}

	sqlDB.SetMaxOpenConns(db.cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(db.cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(db.cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(db.cfg.ConnMaxIdleTime)

	db.DB = sqlDB
	return nil
}

func (db *DB) Close() error {
	if db.DB == nil {
		return nil
	}
	if err := db.DB.Close(); err != nil {
		return errors.Wrap(err, "failed to close database connection")
	}
	return nil
}

func (db *DB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return errors.Wrap(db.PingContext(ctx), "database ping failed")
}
