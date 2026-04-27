package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

func setupMaintenanceTestDB(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()

	// Запускаем PostgreSQL контейнер
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	terminateTestContainer(t, container)

	// Получаем хост и порт
	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Подключаемся к БД
	dsn := "host=" + host + " port=" + port.Port() + " user=test password=test dbname=test sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	closeTestDB(t, db)

	// Ждём готовности БД
	require.Eventually(t, func() bool {
		err := db.Ping()
		return err == nil
	}, 30*time.Second, 1*time.Second)

	// Создаём таблицы
	_, err = db.ExecContext(ctx, `
		CREATE TABLE maintenance_windows (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED',
			recurrence VARCHAR(20) NOT NULL DEFAULT 'ONCE',
			is_global BOOLEAN NOT NULL DEFAULT false,
			pause_monitoring BOOLEAN NOT NULL DEFAULT true,
			suppress_alerts BOOLEAN NOT NULL DEFAULT true,
			safe_mode BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			activated_at TIMESTAMP,
			completed_at TIMESTAMP,
			version INTEGER NOT NULL DEFAULT 1
		);

		CREATE TABLE maintenance_window_monitors (
			maintenance_window_id UUID NOT NULL,
			monitor_id UUID NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (maintenance_window_id, monitor_id)
		);
	`)
	require.NoError(t, err)

	return db
}

func TestMaintenanceWindowRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	window := &interfaces.MaintenanceWindow{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            "Test Maintenance",
		StartTime:       time.Now().Add(1 * time.Hour),
		EndTime:         time.Now().Add(2 * time.Hour),
		Status:          "SCHEDULED",
		Recurrence:      "ONCE",
		IsGlobal:        false,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Version:         1,
	}

	err := repo.Create(ctx, window)
	require.NoError(t, err)

	// Проверяем, что окно создано
	retrieved, err := repo.GetByID(ctx, window.ID)
	require.NoError(t, err)
	assert.Equal(t, window.ID, retrieved.ID)
	assert.Equal(t, window.Name, retrieved.Name)
	assert.Equal(t, window.UserID, retrieved.UserID)
}

func TestMaintenanceWindowRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	window := &interfaces.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "Test Window",
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(2 * time.Hour),
		Status:     "SCHEDULED",
		Recurrence: "ONCE",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Version:    1,
	}

	err := repo.Create(ctx, window)
	require.NoError(t, err)

	// Получаем созданное окно
	retrieved, err := repo.GetByID(ctx, window.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, window.Name, retrieved.Name)

	// Проверяем несуществующее окно
	_, err = repo.GetByID(ctx, uuid.New())
	require.NoError(t, err)
}

func TestMaintenanceWindowRepository_GetByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	// Создаём несколько окон
	for i := 0; i < 3; i++ {
		window := &interfaces.MaintenanceWindow{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       fmt.Sprintf("Window %d", i),
			StartTime:  time.Now().Add(time.Duration(i+1) * time.Hour),
			EndTime:    time.Now().Add(time.Duration(i+2) * time.Hour),
			Status:     "SCHEDULED",
			Recurrence: "ONCE",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Version:    1,
		}
		err := repo.Create(ctx, window)
		require.NoError(t, err)
	}

	// Получаем окна пользователя
	windows, err := repo.GetByUserID(ctx, userID, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, windows, 3)
}

func TestMaintenanceWindowRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	window := &interfaces.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "Original Name",
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(2 * time.Hour),
		Status:     "SCHEDULED",
		Recurrence: "ONCE",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Version:    1,
	}

	err := repo.Create(ctx, window)
	require.NoError(t, err)

	// Обновляем окно
	window.Name = "Updated Name"
	window.StartTime = time.Now().Add(3 * time.Hour)
	window.EndTime = time.Now().Add(4 * time.Hour)
	window.Version = 2
	window.UpdatedAt = time.Now()

	err = repo.Update(ctx, window)
	require.NoError(t, err)

	// Проверяем обновление
	retrieved, err := repo.GetByID(ctx, window.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, 2, retrieved.Version)
}

func TestMaintenanceWindowRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	window := &interfaces.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "Test Window",
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(2 * time.Hour),
		Status:     "SCHEDULED",
		Recurrence: "ONCE",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Version:    1,
	}

	err := repo.Create(ctx, window)
	require.NoError(t, err)

	// Удаляем окно
	err = repo.Delete(ctx, window.ID)
	require.NoError(t, err)

	// Проверяем, что окно удалено
	deletedWindow, err := repo.GetByID(ctx, window.ID)
	require.NoError(t, err)
	assert.Nil(t, deletedWindow)
}

func TestMaintenanceWindowRepository_CheckOverlap(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	monitorID := uuid.New()

	// Создаём первое окно
	window1 := &interfaces.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "Window 1",
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(3 * time.Hour),
		Status:     "SCHEDULED",
		Recurrence: "ONCE",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Version:    1,
	}

	err := repo.Create(ctx, window1)
	require.NoError(t, err)

	// Добавляем монитор к окну
	err = repo.AddMonitorsToWindow(ctx, window1.ID, []uuid.UUID{monitorID})
	require.NoError(t, err)

	// Проверяем пересечение - должно быть true
	startTime := time.Now().Add(2 * time.Hour)
	endTime := time.Now().Add(4 * time.Hour)

	hasOverlap, err := repo.CheckOverlap(ctx, userID, []uuid.UUID{monitorID}, startTime, endTime, nil)
	require.NoError(t, err)
	assert.True(t, hasOverlap)

	// Проверяем отсутствие пересечения - должно быть false
	startTime2 := time.Now().Add(5 * time.Hour)
	endTime2 := time.Now().Add(6 * time.Hour)

	hasOverlap2, err := repo.CheckOverlap(ctx, userID, []uuid.UUID{monitorID}, startTime2, endTime2, nil)
	require.NoError(t, err)
	assert.False(t, hasOverlap2)
}

func TestMaintenanceWindowRepository_CountByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	// Создаём окна
	for i := 0; i < 3; i++ {
		window := &interfaces.MaintenanceWindow{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       fmt.Sprintf("Window %d", i),
			StartTime:  time.Now().Add(time.Duration(i+1) * time.Hour),
			EndTime:    time.Now().Add(time.Duration(i+2) * time.Hour),
			Status:     "SCHEDULED",
			Recurrence: "ONCE",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Version:    1,
		}
		err := repo.Create(ctx, window)
		require.NoError(t, err)
	}

	// Считаем окна
	count, err := repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMaintenanceWindowRepository_AddAndGetMonitors(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	db := setupMaintenanceTestDB(t)

	repo := NewMaintenanceWindowRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	monitorID1 := uuid.New()
	monitorID2 := uuid.New()

	window := &interfaces.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "Test Window",
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(2 * time.Hour),
		Status:     "SCHEDULED",
		Recurrence: "ONCE",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Version:    1,
	}

	err := repo.Create(ctx, window)
	require.NoError(t, err)

	// Добавляем мониторы
	err = repo.AddMonitorsToWindow(ctx, window.ID, []uuid.UUID{monitorID1, monitorID2})
	require.NoError(t, err)

	// Получаем мониторы
	monitors, err := repo.GetWindowMonitors(ctx, window.ID)
	require.NoError(t, err)
	assert.Len(t, monitors, 2)
	assert.Contains(t, monitors, monitorID1)
	assert.Contains(t, monitors, monitorID2)
}
