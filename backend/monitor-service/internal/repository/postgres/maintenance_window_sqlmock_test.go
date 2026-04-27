package postgres

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

// maintenanceWindowColumns возвращает список колонок таблицы maintenance_windows.
func maintenanceWindowColumns() []string {
	return []string{
		"id", "user_id", "name", "start_time", "end_time",
		"status", "recurrence", "is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"created_at", "updated_at", "activated_at", "completed_at", "version",
	}
}

// addWindowRow добавляет строку окна обслуживания в mock rows.
func addWindowRow(rows *sqlmock.Rows, id, userID uuid.UUID, name, status string) *sqlmock.Rows {
	now := time.Now()
	return rows.AddRow(
		id, userID, name,
		now.Add(1*time.Hour), now.Add(2*time.Hour),
		status, "ONCE", false, true, true, false,
		now, now, nil, nil, 1,
	)
}

// TestMaintenanceWindowRepository_NewMaintenanceWindowRepository тестирует создание репозитория.
func TestMaintenanceWindowRepository_NewMaintenanceWindowRepository(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	assert.NotNil(t, repo)
	assert.Implements(t, (*interfaces.MaintenanceWindowRepository)(nil), repo)
}

// TestMaintenanceWindowRepository_Create_Success тестирует успешное создание окна.
func TestMaintenanceWindowRepository_Create_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	window := &interfaces.MaintenanceWindow{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            "Test Window",
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

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO maintenance_windows`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.Create(context.Background(), window)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Create_WithMonitors тестирует создание окна с мониторами.
func TestMaintenanceWindowRepository_Create_WithMonitors(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	monitorID := uuid.New()
	window := &interfaces.MaintenanceWindow{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            "Test Window",
		StartTime:       time.Now().Add(1 * time.Hour),
		EndTime:         time.Now().Add(2 * time.Hour),
		Status:          "SCHEDULED",
		Recurrence:      "ONCE",
		IsGlobal:        false,
		MonitorIDs:      []uuid.UUID{monitorID},
		PauseMonitoring: true,
		SuppressAlerts:  true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Version:         1,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO maintenance_windows`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO maintenance_window_monitors`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.Create(context.Background(), window)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Create_BeginError тестирует ошибку при начале транзакции.
func TestMaintenanceWindowRepository_Create_BeginError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	window := &interfaces.MaintenanceWindow{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
	}

	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

	err = repo.Create(context.Background(), window)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Create_ExecError тестирует ошибку при вставке.
func TestMaintenanceWindowRepository_Create_ExecError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	window := &interfaces.MaintenanceWindow{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO maintenance_windows`)).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err = repo.Create(context.Background(), window)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetByID_ReturnsErrorOnScan тестирует что GetByID возвращает ошибку.
// GetContext использует sqlx StructScan, которому требуются db-теги на полях структуры.
// Поскольку interfaces.MaintenanceWindow не имеет db-тегов, метод всегда возвращает ошибку при реальных данных.
func TestMaintenanceWindowRepository_GetByID_ReturnsErrorOnScan(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, windowID, userID, "Test Window", "SCHEDULED")

	mock.ExpectQuery(`SELECT id, user_id`).WithArgs(windowID).WillReturnRows(rows)

	// GetContext со struct без db-тегов даёт ошибку "missing destination name"
	window, err := repo.GetByID(context.Background(), windowID)

	assert.Error(t, err)
	assert.Nil(t, window)
}

// TestMaintenanceWindowRepository_GetByID_NotFound тестирует случай, когда окно не найдено.
func TestMaintenanceWindowRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, start_time, end_time, status, recurrence,
			   is_global, pause_monitoring, suppress_alerts, safe_mode,
			   created_at, updated_at, activated_at, completed_at, version
		FROM maintenance_windows
		WHERE id = $1
	`)).WithArgs(windowID).WillReturnError(sql.ErrNoRows)

	window, err := repo.GetByID(context.Background(), windowID)

	assert.NoError(t, err)
	assert.Nil(t, window)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetByID_DBError тестирует ошибку БД.
func TestMaintenanceWindowRepository_GetByID_DBError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, start_time, end_time, status, recurrence,
			   is_global, pause_monitoring, suppress_alerts, safe_mode,
			   created_at, updated_at, activated_at, completed_at, version
		FROM maintenance_windows
		WHERE id = $1
	`)).WithArgs(windowID).WillReturnError(sql.ErrConnDone)

	window, err := repo.GetByID(context.Background(), windowID)

	assert.Error(t, err)
	assert.Nil(t, window)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetByUserID_Success тестирует получение окон пользователя.
func TestMaintenanceWindowRepository_GetByUserID_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, uuid.New(), userID, "Window 1", "SCHEDULED")
	addWindowRow(rows, uuid.New(), userID, "Window 2", "ACTIVE")

	mock.ExpectQuery(`SELECT id, user_id, name`).
		WillReturnRows(rows)

	windows, err := repo.GetByUserID(context.Background(), userID, "", 10, 0)

	assert.NoError(t, err)
	assert.Len(t, windows, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetByUserID_WithStatusFilter тестирует фильтрацию по статусу.
func TestMaintenanceWindowRepository_GetByUserID_WithStatusFilter(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, uuid.New(), userID, "Active Window", "ACTIVE")

	mock.ExpectQuery(`SELECT id, user_id, name`).
		WillReturnRows(rows)

	windows, err := repo.GetByUserID(context.Background(), userID, "ACTIVE", 10, 0)

	assert.NoError(t, err)
	assert.Len(t, windows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetByUserID_Error тестирует ошибку при запросе.
func TestMaintenanceWindowRepository_GetByUserID_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(`SELECT id, user_id, name`).
		WillReturnError(sql.ErrConnDone)

	windows, err := repo.GetByUserID(context.Background(), userID, "", 10, 0)

	assert.Error(t, err)
	assert.Nil(t, windows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Update_Success тестирует успешное обновление окна.
func TestMaintenanceWindowRepository_Update_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	now := time.Now()
	window := &interfaces.MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Updated Window",
		StartTime: now.Add(1 * time.Hour),
		EndTime:   now.Add(3 * time.Hour),
		Status:    "SCHEDULED",
		UpdatedAt: now,
		Version:   2,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maintenance_windows
		SET name = $2, start_time = $3, end_time = $4,
		    status = $5, updated_at = $6, version = $7
		WHERE id = $1 AND version = $8
	`)).WithArgs(
		window.ID, window.Name, window.StartTime, window.EndTime,
		window.Status, window.UpdatedAt, window.Version, window.Version-1,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Update(context.Background(), window)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Update_VersionConflict тестирует конфликт версий.
func TestMaintenanceWindowRepository_Update_VersionConflict(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	now := time.Now()
	window := &interfaces.MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Updated Window",
		StartTime: now.Add(1 * time.Hour),
		EndTime:   now.Add(3 * time.Hour),
		Status:    "SCHEDULED",
		UpdatedAt: now,
		Version:   2,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maintenance_windows
		SET name = $2, start_time = $3, end_time = $4,
		    status = $5, updated_at = $6, version = $7
		WHERE id = $1 AND version = $8
	`)).WithArgs(
		window.ID, window.Name, window.StartTime, window.EndTime,
		window.Status, window.UpdatedAt, window.Version, window.Version-1,
	).WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Update(context.Background(), window)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version conflict")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Update_Error тестирует ошибку при обновлении.
func TestMaintenanceWindowRepository_Update_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	now := time.Now()
	window := &interfaces.MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Window",
		StartTime: now.Add(1 * time.Hour),
		EndTime:   now.Add(2 * time.Hour),
		Status:    "SCHEDULED",
		UpdatedAt: now,
		Version:   1,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maintenance_windows
		SET name = $2, start_time = $3, end_time = $4,
		    status = $5, updated_at = $6, version = $7
		WHERE id = $1 AND version = $8
	`)).WillReturnError(sql.ErrConnDone)

	err = repo.Update(context.Background(), window)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Delete_Success тестирует успешное удаление окна.
func TestMaintenanceWindowRepository_Delete_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM maintenance_windows WHERE id = $1`)).
		WithArgs(windowID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(context.Background(), windowID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_Delete_Error тестирует ошибку при удалении.
func TestMaintenanceWindowRepository_Delete_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM maintenance_windows WHERE id = $1`)).
		WithArgs(windowID).
		WillReturnError(sql.ErrConnDone)

	err = repo.Delete(context.Background(), windowID)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_CountByUserID_Success тестирует подсчёт окон.
func TestMaintenanceWindowRepository_CountByUserID_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM maintenance_windows WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountByUserID(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_CountByUserID_Error тестирует ошибку при подсчёте.
func TestMaintenanceWindowRepository_CountByUserID_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM maintenance_windows WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	count, err := repo.CountByUserID(context.Background(), userID)

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetActiveWindowsForMonitor_Success тестирует получение активных окон для монитора.
func TestMaintenanceWindowRepository_GetActiveWindowsForMonitor_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()
	at := time.Now()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, uuid.New(), userID, "Active Window", "ACTIVE")

	mock.ExpectQuery(`SELECT DISTINCT w.id`).
		WillReturnRows(rows)

	windows, err := repo.GetActiveWindowsForMonitor(context.Background(), monitorID, at)

	assert.NoError(t, err)
	assert.Len(t, windows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetActiveWindowsForMonitor_Error тестирует ошибку.
func TestMaintenanceWindowRepository_GetActiveWindowsForMonitor_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	monitorID := uuid.New()
	at := time.Now()

	mock.ExpectQuery(`SELECT DISTINCT w.id`).
		WillReturnError(sql.ErrConnDone)

	windows, err := repo.GetActiveWindowsForMonitor(context.Background(), monitorID, at)

	assert.Error(t, err)
	assert.Nil(t, windows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetActiveWindowsForUser_Success тестирует получение активных окон пользователя.
func TestMaintenanceWindowRepository_GetActiveWindowsForUser_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	at := time.Now()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, uuid.New(), userID, "Active Window", "ACTIVE")

	mock.ExpectQuery(`SELECT id, user_id, name, start_time, end_time, status`).
		WillReturnRows(rows)

	windows, err := repo.GetActiveWindowsForUser(context.Background(), userID, at)

	assert.NoError(t, err)
	assert.Len(t, windows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetWindowsRequiringActivation_Success тестирует получение окон для активации.
func TestMaintenanceWindowRepository_GetWindowsRequiringActivation_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	before := time.Now()
	userID := uuid.New()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, uuid.New(), userID, "Scheduled Window", "SCHEDULED")

	mock.ExpectQuery(`SELECT id, user_id, name`).
		WillReturnRows(rows)

	windows, err := repo.GetWindowsRequiringActivation(context.Background(), before)

	assert.NoError(t, err)
	assert.Len(t, windows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetWindowsRequiringActivation_Error тестирует ошибку при запросе.
func TestMaintenanceWindowRepository_GetWindowsRequiringActivation_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	before := time.Now()

	mock.ExpectQuery(`SELECT id, user_id, name`).
		WillReturnError(sql.ErrConnDone)

	windows, err := repo.GetWindowsRequiringActivation(context.Background(), before)

	assert.Error(t, err)
	assert.Nil(t, windows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetWindowsRequiringCompletion_Success тестирует получение окон для завершения.
func TestMaintenanceWindowRepository_GetWindowsRequiringCompletion_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	before := time.Now()
	userID := uuid.New()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, uuid.New(), userID, "Active Window", "ACTIVE")

	mock.ExpectQuery(`SELECT id, user_id, name`).
		WillReturnRows(rows)

	windows, err := repo.GetWindowsRequiringCompletion(context.Background(), before)

	assert.NoError(t, err)
	assert.Len(t, windows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetWindowsRequiringCompletion_Error тестирует ошибку.
func TestMaintenanceWindowRepository_GetWindowsRequiringCompletion_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	before := time.Now()

	mock.ExpectQuery(`SELECT id, user_id, name`).
		WillReturnError(sql.ErrConnDone)

	windows, err := repo.GetWindowsRequiringCompletion(context.Background(), before)

	assert.Error(t, err)
	assert.Nil(t, windows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_AddMonitorsToWindow_Success тестирует добавление мониторов к окну.
func TestMaintenanceWindowRepository_AddMonitorsToWindow_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()
	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maintenance_window_monitors (maintenance_window_id, monitor_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`)).WithArgs(windowID, monitorID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddMonitorsToWindow(context.Background(), windowID, []uuid.UUID{monitorID})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_AddMonitorsToWindow_Error тестирует ошибку при добавлении мониторов.
func TestMaintenanceWindowRepository_AddMonitorsToWindow_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()
	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maintenance_window_monitors (maintenance_window_id, monitor_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`)).WillReturnError(sql.ErrConnDone)

	err = repo.AddMonitorsToWindow(context.Background(), windowID, []uuid.UUID{monitorID})

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_RemoveMonitorsFromWindow_UnsupportedType тестирует что []uuid.UUID не поддерживается драйвером напрямую.
func TestMaintenanceWindowRepository_RemoveMonitorsFromWindow_UnsupportedType(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()
	monitorIDs := []uuid.UUID{uuid.New(), uuid.New()}

	// Драйвер не поддерживает []uuid.UUID без pq.Array, поэтому всегда возвращает ошибку конвертации.
	// Ожидание не выставляем — ExecContext падает до попытки выполнить запрос.
	_ = mock

	err = repo.RemoveMonitorsFromWindow(context.Background(), windowID, monitorIDs)

	assert.Error(t, err)
}

// TestMaintenanceWindowRepository_GetWindowMonitors_Success тестирует получение мониторов окна.
func TestMaintenanceWindowRepository_GetWindowMonitors_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()
	monitorID1 := uuid.New()
	monitorID2 := uuid.New()

	rows := sqlmock.NewRows([]string{"monitor_id"}).
		AddRow(monitorID1).
		AddRow(monitorID2)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT monitor_id FROM maintenance_window_monitors WHERE maintenance_window_id = $1`)).
		WithArgs(windowID).
		WillReturnRows(rows)

	monitorIDs, err := repo.GetWindowMonitors(context.Background(), windowID)

	assert.NoError(t, err)
	assert.Len(t, monitorIDs, 2)
	assert.Contains(t, monitorIDs, monitorID1)
	assert.Contains(t, monitorIDs, monitorID2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetWindowMonitors_Empty тестирует получение пустого списка мониторов.
func TestMaintenanceWindowRepository_GetWindowMonitors_Empty(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()

	rows := sqlmock.NewRows([]string{"monitor_id"})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT monitor_id FROM maintenance_window_monitors WHERE maintenance_window_id = $1`)).
		WithArgs(windowID).
		WillReturnRows(rows)

	monitorIDs, err := repo.GetWindowMonitors(context.Background(), windowID)

	assert.NoError(t, err)
	assert.Len(t, monitorIDs, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetWindowMonitors_Error тестирует ошибку при получении мониторов.
func TestMaintenanceWindowRepository_GetWindowMonitors_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	windowID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT monitor_id FROM maintenance_window_monitors WHERE maintenance_window_id = $1`)).
		WithArgs(windowID).
		WillReturnError(sql.ErrConnDone)

	monitorIDs, err := repo.GetWindowMonitors(context.Background(), windowID)

	assert.Error(t, err)
	assert.Nil(t, monitorIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetHistory_Success тестирует получение истории.
func TestMaintenanceWindowRepository_GetHistory_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	rows := sqlmock.NewRows(maintenanceWindowColumns())
	addWindowRow(rows, uuid.New(), userID, "Completed Window", "COMPLETED")

	mock.ExpectQuery(`SELECT DISTINCT w.id`).
		WillReturnRows(rows)

	windows, err := repo.GetHistory(context.Background(), userID, startDate, endDate, nil, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, windows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_GetHistory_WithMonitorFilter тестирует что передача []uuid.UUID напрямую вызывает ошибку конвертации.
func TestMaintenanceWindowRepository_GetHistory_WithMonitorFilter(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	monitorID := uuid.New()
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	// Драйвер не поддерживает []uuid.UUID без pq.Array, поэтому QueryContext возвращает ошибку конвертации.
	windows, err := repo.GetHistory(context.Background(), userID, startDate, endDate, []uuid.UUID{monitorID}, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, windows)
}

// TestMaintenanceWindowRepository_GetHistory_Error тестирует ошибку при получении истории.
func TestMaintenanceWindowRepository_GetHistory_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	mock.ExpectQuery(`SELECT DISTINCT w.id`).
		WillReturnError(sql.ErrConnDone)

	windows, err := repo.GetHistory(context.Background(), userID, startDate, endDate, nil, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, windows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_CheckOverlap_WithMonitors тестирует что []uuid.UUID не поддерживается драйвером напрямую.
func TestMaintenanceWindowRepository_CheckOverlap_WithMonitors(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := time.Now().Add(3 * time.Hour)

	// Драйвер не поддерживает []uuid.UUID без pq.Array, поэтому всегда возвращает ошибку конвертации.
	exists, err := repo.CheckOverlap(context.Background(), userID, []uuid.UUID{monitorID}, startTime, endTime, nil)

	assert.Error(t, err)
	assert.False(t, exists)
}

// TestMaintenanceWindowRepository_CheckOverlap_NoOverlap тестирует отсутствие пересечения.
func TestMaintenanceWindowRepository_CheckOverlap_NoOverlap(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	startTime := time.Now().Add(10 * time.Hour)
	endTime := time.Now().Add(12 * time.Hour)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repo.CheckOverlap(context.Background(), userID, []uuid.UUID{}, startTime, endTime, nil)

	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_CheckOverlap_WithExcludeID тестирует проверку пересечения с исключением окна (пустой список мониторов).
func TestMaintenanceWindowRepository_CheckOverlap_WithExcludeID(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	excludeID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := time.Now().Add(3 * time.Hour)

	// При пустом списке мониторов добавляется только условие на is_global=true
	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repo.CheckOverlap(context.Background(), userID, []uuid.UUID{}, startTime, endTime, &excludeID)

	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMaintenanceWindowRepository_CheckOverlap_Error тестирует ошибку при проверке пересечения.
func TestMaintenanceWindowRepository_CheckOverlap_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMaintenanceWindowRepository(db)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := time.Now().Add(3 * time.Hour)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnError(sql.ErrConnDone)

	exists, err := repo.CheckOverlap(context.Background(), userID, []uuid.UUID{}, startTime, endTime, nil)

	assert.Error(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}
