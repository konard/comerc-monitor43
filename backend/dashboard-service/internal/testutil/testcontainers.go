package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgreSQLContainer struct {
	*postgres.PostgresContainer
}

func NewPostgreSQLContainer(ctx context.Context) (*PostgreSQLContainer, *sqlx.DB, error) {
	postgresContainer, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create postgres container: %w", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	time.Sleep(2 * time.Second)

	var db *sqlx.DB
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("postgres", connStr)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(1 * time.Second)
		}
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgreSQLContainer{PostgresContainer: postgresContainer}, db, nil
}

func (c *PostgreSQLContainer) Shutdown(ctx context.Context) error {
	if c.PostgresContainer != nil {
		if err := c.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate postgres container: %w", err)
		}
	}
	return nil
}
