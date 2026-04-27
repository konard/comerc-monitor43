package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

// DB обёртка над sqlx.DB.
type DB struct {
	DB *sqlx.DB
}

// NewDB создаёт новое подключение к PostgreSQL.
func NewDB(dsn string) (*DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Настройка connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return &DB{DB: db}, nil
}

// Close закрывает подключение к базе данных.
func (d *DB) Close() error {
	if err := d.DB.Close(); err != nil {
		return errors.Wrap(err, "failed to close database")
	}

	return nil
}

// Ping проверяет соединение с базой данных.
func (d *DB) Ping() error {
	return d.DB.Ping()
}

// PingContext проверяет соединение с контекстом.
func (d *DB) PingContext(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return d.DB.PingContext(ctx)
}
