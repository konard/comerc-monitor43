package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel/trace"

	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

// Config представляет конфигурацию подключения к PostgreSQL
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "milan",
		SSLMode:      "disable",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		MaxLifetime:  5 * time.Minute,
	}
}

// DB представляет обёртку над sqlx.DB
type DB struct {
	*sqlx.DB
	config  *Config
	dsn     string
	tracer  trace.Tracer
	metrics *apptelemetry.Metrics
}

// NewDB создаёт DB из конфигурации без установки соединения
func NewDB(config Config) *DB {
	return &DB{config: &config}
}

// NewDBFromDSN создаёт DB из DSN строки без установки соединения
func NewDBFromDSN(dsn string) *DB {
	return &DB{dsn: dsn}
}

// Connect устанавливает соединение с базой данных
func (db *DB) Connect() error {
	dsn := db.dsn
	if db.config != nil {
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			db.config.Host,
			db.config.Port,
			db.config.User,
			db.config.Password,
			db.config.Database,
			db.config.SSLMode,
		)
	}

	conn, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	if db.config != nil {
		conn.SetMaxOpenConns(db.config.MaxOpenConns)
		conn.SetMaxIdleConns(db.config.MaxIdleConns)
		conn.SetConnMaxLifetime(db.config.MaxLifetime)
	}

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	db.DB = conn
	return nil
}

// SetObservability устанавливает tracer и metrics для наблюдаемости
func (db *DB) SetObservability(tracer trace.Tracer, metrics *apptelemetry.Metrics) {
	db.tracer = tracer
	db.metrics = metrics
}

// GetTracer возвращает tracer для создания спанов
func (db *DB) GetTracer() trace.Tracer {
	return db.tracer
}

// GetMetrics возвращает metrics для записи метрик
func (db *DB) GetMetrics() *apptelemetry.Metrics {
	return db.metrics
}

// BeginTxx начинает транзакцию (для совместимости с sqlx)
func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (any, error) {
	tx, err := db.DB.BeginTxx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	return tx, nil
}

// BeginTx начинает транзакцию (возвращает конкретный тип)
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	tx, err := db.DB.BeginTxx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	return tx, nil
}

// Rollback откатывает транзакцию
func (db *DB) Rollback(tx any) error {
	if sqlxTx, ok := tx.(*sqlx.Tx); ok {
		if err := sqlxTx.Rollback(); err != nil {
			return fmt.Errorf("failed to rollback transaction: %v", err)
		}
		return nil
	}
	return errors.New("invalid transaction type")
}

// Commit фиксирует транзакцию
func (db *DB) Commit(tx any) error {
	if sqlxTx, ok := tx.(*sqlx.Tx); ok {
		if err := sqlxTx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}
		return nil
	}
	return errors.New("invalid transaction type")
}
