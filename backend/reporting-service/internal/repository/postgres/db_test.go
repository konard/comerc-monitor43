package postgres

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/reporting-service/internal/config"
)

func TestNewDB(t *testing.T) {
	t.Parallel()

	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Name:     "db",
		SSLMode:  "disable",
	}

	db := NewDB(cfg)

	require.NotNil(t, db)
	assert.Equal(t, cfg, db.cfg)
	assert.Nil(t, db.DB)
}

func TestDB_Close_NilDB(t *testing.T) {
	t.Parallel()

	cfg := config.DatabaseConfig{}
	db := NewDB(cfg)

	// закрытие незаинициализированной БД должно возвращать nil
	err := db.Close()
	assert.NoError(t, err)
}

func TestDB_Close_WithDB(t *testing.T) {
	t.Parallel()

	stdDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectClose()

	db := NewDB(config.DatabaseConfig{})
	db.DB = sqlx.NewDb(stdDB, "postgres")

	err = db.Close()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Close_Error(t *testing.T) {
	t.Parallel()

	stdDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectClose().WillReturnError(fmt.Errorf("close failed"))

	db := NewDB(config.DatabaseConfig{})
	db.DB = sqlx.NewDb(stdDB, "postgres")

	err = db.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close database connection")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Ping_Success(t *testing.T) {
	t.Parallel()

	stdDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	mock.ExpectPing()

	db := NewDB(config.DatabaseConfig{})
	db.DB = sqlx.NewDb(stdDB, "postgres")

	err = db.Ping()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Ping_Error(t *testing.T) {
	t.Parallel()

	stdDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	mock.ExpectPing().WillReturnError(fmt.Errorf("ping failed"))

	db := NewDB(config.DatabaseConfig{})
	db.DB = sqlx.NewDb(stdDB, "postgres")

	err = db.Ping()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database ping failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}
