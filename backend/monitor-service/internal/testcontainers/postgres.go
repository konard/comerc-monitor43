package testcontainers

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgreSQLContainer содержит контейнер и его connection string.
type PostgreSQLContainer struct {
	Container        testcontainers.Container
	ConnectionString string
}

// SetupPostgreSQLContainer создаёт PostgreSQL контейнер для тестов.
// Возвращает структуру с контейнером и connection string для правильного cleanup.
func SetupPostgreSQLContainer(ctx context.Context) (*PostgreSQLContainer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "testdb",
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second).
				WithPollInterval(1 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Получаем порт хоста
	host, err := container.Host(ctx)
	if err != nil {
		if terminateErr := container.Terminate(ctx); terminateErr != nil {
			return nil, fmt.Errorf("failed to get host: %w (cleanup failed: %v)", err, terminateErr)
		}
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		if terminateErr := container.Terminate(ctx); terminateErr != nil {
			return nil, fmt.Errorf("failed to get port: %w (cleanup failed: %v)", err, terminateErr)
		}
		return nil, fmt.Errorf("failed to get port: %w", err)
	}

	// Формируем connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, int(port.Num()), "testuser", "testpass", "testdb")

	return &PostgreSQLContainer{
		Container:        container,
		ConnectionString: connStr,
	}, nil
}

// TeardownPostgreSQLContainer останавливает и удаляет контейнер.
func TeardownPostgreSQLContainer(ctx context.Context, pgContainer *PostgreSQLContainer) error {
	if pgContainer != nil && pgContainer.Container != nil {
		return pgContainer.Container.Terminate(ctx)
	}
	return nil
}
