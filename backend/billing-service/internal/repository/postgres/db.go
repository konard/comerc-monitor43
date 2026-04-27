package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

// DB является wrapper для *sqlx.DB с дополнительными методами.
type DB struct {
	*sqlx.DB
}

// NewDB создаёт новое подключение к PostgreSQL.
func NewDB(host string, port int, user, password, dbname, sslMode string) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	// Настройка connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(30 * time.Second)

	return &DB{DB: db}, nil
}

// Close закрывает подключение к базе данных.
func (db *DB) Close() error {
	return errors.Wrap(db.DB.Close(), "failed to close database connection")
}

// Ping проверяет соединение с базой данных.
func (db *DB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return errors.Wrap(db.PingContext(ctx), "database ping failed")
}
