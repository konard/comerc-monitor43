// Package test содержит roundtrip-тест миграций для всех сервисов монорепозитория.
package test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// serviceConfig описывает конфигурацию сервиса для проверки миграций.
//
// Несколько сервисов могут делить одну БД (dbName) — в этом случае миграции
// сервиса могут содержать FK на таблицы, созданные ранее другим сервисом в той
// же БД. Порядок в слайсе services фиксирует зависимости: в рамках одной dbName
// более ранние записи являются предусловием для более поздних.
type serviceConfig struct {
	name      string
	dbName    string
	tableName string
	migrDir   string
}

// repoRoot возвращает абсолютный путь к корню монорепозитория.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// Путь: test/migration_roundtrip_test.go → 1 уровень вверх = корень репо
	return filepath.Join(filepath.Dir(file), "..")
}

// TestMigrationRoundtrip проверяет, что миграции всех сервисов корректно
// применяются (фаза Up) в реальной конфигурации БД.
//
// Сервисы, делящие одну БД, накатываются в порядке зависимостей: монитор-
// сервис создаёт базовые таблицы (monitors, maintenance_windows), auth-service
// добавляет users, alert-service и дальше опираются на эти таблицы через FK.
//
// Фаза Down намеренно не проверяется: в shared DB владение таблицами
// размывается между сервисами (например, maintenance_windows создаётся в
// monitor-service, но на неё завязаны миграции alert-service), и универсальный
// Reset по одному сервису в такой модели невозможен без более глубокой чистки
// в самих миграциях.
func TestMigrationRoundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	root := repoRoot()

	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(pg); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	adminDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	adminDB, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	t.Cleanup(func() {
		//nolint:errcheck // close в Cleanup, ошибка не влияет на результат теста
		adminDB.Close()
	})

	// Порядок внутри группы dbName фиксирует зависимости миграций.
	services := []serviceConfig{
		{name: "monitor-service", dbName: "monitor", tableName: "goose_db_version", migrDir: filepath.Join(root, "backend", "monitor-service", "migrations")},
		{name: "auth-service", dbName: "monitor", tableName: "goose_auth_version", migrDir: filepath.Join(root, "backend", "auth-service", "migrations")},
		{name: "alert-service", dbName: "monitor", tableName: "goose_alert_version", migrDir: filepath.Join(root, "backend", "alert-service", "migrations")},
		{name: "scheduler-service", dbName: "monitor", tableName: "goose_scheduler_version", migrDir: filepath.Join(root, "backend", "scheduler-service", "migrations")},
		{name: "reporting-service", dbName: "monitor", tableName: "goose_reporting_version", migrDir: filepath.Join(root, "backend", "reporting-service", "migrations")},
		{name: "billing-service", dbName: "billing", tableName: "goose_db_version", migrDir: filepath.Join(root, "backend", "billing-service", "migrations")},
		{name: "dashboard-service", dbName: "dashboard", tableName: "goose_db_version", migrDir: filepath.Join(root, "backend", "dashboard-service", "migrations")},
		{name: "integration-service", dbName: "integration_service", tableName: "goose_db_version", migrDir: filepath.Join(root, "backend", "integration-service", "migrations")},
		{name: "notification-template-service", dbName: "notification_templates", tableName: "goose_db_version", migrDir: filepath.Join(root, "backend", "notification-template-service", "migrations")},
	}

	// Группируем сервисы по dbName, сохраняя исходный порядок появления групп.
	var groupOrder []string
	groups := map[string][]serviceConfig{}
	for _, svc := range services {
		if _, seen := groups[svc.dbName]; !seen {
			groupOrder = append(groupOrder, svc.dbName)
		}
		groups[svc.dbName] = append(groups[svc.dbName], svc)
	}

	require.NoError(t, goose.SetDialect("postgres"))

	for _, dbName := range groupOrder {
		group := groups[dbName]
		t.Run(dbName, func(t *testing.T) {
			// Отфильтровываем сервисы, чьи директории миграций отсутствуют.
			present := make([]serviceConfig, 0, len(group))
			for _, svc := range group {
				if _, err := os.Stat(svc.migrDir); os.IsNotExist(err) {
					t.Logf("migrations directory not found, skipping %s: %s", svc.name, svc.migrDir)
					continue
				}
				present = append(present, svc)
			}
			if len(present) == 0 {
				t.Skipf("no services with migrations for db %q", dbName)
			}

			testDBName := fmt.Sprintf("test_%s", dbName)
			_, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %q", testDBName))
			require.NoError(t, err)
			t.Cleanup(func() {
				//nolint:errcheck // best-effort cleanup тестовой БД
				_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %q", testDBName))
			})

			svcDSN, err := pg.ConnectionString(ctx, "sslmode=disable", fmt.Sprintf("dbname=%s", testDBName))
			require.NoError(t, err)

			db, err := sql.Open("postgres", svcDSN)
			require.NoError(t, err)
			t.Cleanup(func() {
				//nolint:errcheck // close в Cleanup, ошибка не влияет на результат теста
				db.Close()
			})

			// Накатываем миграции по порядку зависимостей.
			for _, svc := range present {
				goose.SetTableName(svc.tableName)
				require.NoErrorf(t, goose.Up(db, svc.migrDir), "goose Up failed for %s", svc.name)

				version, err := goose.GetDBVersion(db)
				require.NoErrorf(t, err, "get db version after Up %s", svc.name)
				require.Greaterf(t, version, int64(0), "expected migrations applied for %s", svc.name)
				t.Logf("%s: applied migrations up to version %d", svc.name, version)
			}
		})
	}
}
