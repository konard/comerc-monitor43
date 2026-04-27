package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// DB обёртка над sql.DB для удобства.
type DB struct {
	*sql.DB
}

// NewDB создаёт новое подключение к PostgreSQL.
func NewDB(dsn string) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Close закрывает подключение к БД.
func (db *DB) Close() error {
	return db.DB.Close()
}
