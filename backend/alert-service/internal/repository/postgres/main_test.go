package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/raul/monitor/backend/alert-service/internal/testutil"
)

// sharedDB хранит единственное подключение к БД для всех интеграционных тестов пакета.
// Один контейнер на весь пакет предотвращает истощение ресурсов Docker при параллельном запуске.
var sharedDB *sqlx.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, db, err := testutil.StartPostgreSQLContainer(ctx)
	if err != nil {
		// Если контейнер не удалось запустить — пропускаем интеграционные тесты
		os.Exit(m.Run())
	}
	sharedDB = db

	code := m.Run()

	if shutdownErr := container.Shutdown(ctx); shutdownErr != nil {
		// Не фатально — логируем и продолжаем
		_ = shutdownErr
	}
	os.Exit(code)
}

// setupSharedTestDB возвращает подключение к общей тестовой БД и создаёт таблицы.
// Используется в тестах вместо testutil.SetupTestDatabase.
func setupSharedTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	if sharedDB == nil {
		t.Skip("shared test database not available")
	}
	testutil.CreateTablesForTest(t, sharedDB)
	return sharedDB
}
