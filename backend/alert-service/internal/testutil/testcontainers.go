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

// PostgreSQLContainer represents a PostgreSQL container
type PostgreSQLContainer struct {
	*postgres.PostgresContainer
}

// StartPostgreSQLContainer создаёт и запускает PostgreSQL контейнер для тестов
func StartPostgreSQLContainer(ctx context.Context) (*PostgreSQLContainer, *sqlx.DB, error) {
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
		return nil, nil, fmt.Errorf("failed to create postgres container: %v", err)
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection string: %v", err)
	}

	// Add a small delay to ensure database is fully ready
	time.Sleep(2 * time.Second)

	// Connect to database with retries
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
		return nil, nil, fmt.Errorf("failed to connect to postgres: %v", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to ping postgres: %v", err)
	}

	return &PostgreSQLContainer{PostgresContainer: postgresContainer}, db, nil
}

// Shutdown stops and removes the container
func (c *PostgreSQLContainer) Shutdown(ctx context.Context) error {
	if c.PostgresContainer == nil {
		return nil
	}
	if err := c.Terminate(ctx); err != nil {
		return fmt.Errorf("failed to terminate postgres container: %v", err)
	}
	return nil
}
