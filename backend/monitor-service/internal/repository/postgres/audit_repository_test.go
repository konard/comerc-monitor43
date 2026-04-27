// Package postgres предоставляет реализацию репозиториев для работы с PostgreSQL.
// Unit тесты используют sqlmock для мокания SQL-запросов.
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

// TestAuditRepository_Create_Success тестирует успешное создание audit log entry.
func TestAuditRepository_Create_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()
	entryID := uuid.New()

	oldValues := map[string]any{"name": "Old Monitor"}
	newValues := map[string]any{"name": "New Monitor"}

	entry := &interfaces.AuditLogEntry{
		ID:        entryID,
		MonitorID: monitorID,
		UserID:    userID,
		Action:    interfaces.ActionMonitorUpdate,
		OldValues: oldValues,
		NewValues: newValues,
		IPAddress: "192.168.1.1",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
	}

	oldValuesJSON := mustMarshalJSON(t, oldValues)
	newValuesJSON := mustMarshalJSON(t, newValues)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO monitor_audit_log (
			id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)).
		WithArgs(entryID, monitorID, userID, interfaces.ActionMonitorUpdate, oldValuesJSON, newValuesJSON, "192.168.1.1", "test-agent", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), entry)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_Create_Error тестирует ошибку при создании audit log entry.
func TestAuditRepository_Create_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()

	entry := &interfaces.AuditLogEntry{
		ID:        uuid.New(),
		MonitorID: monitorID,
		UserID:    userID,
		Action:    interfaces.ActionMonitorCreate,
		OldValues: map[string]any{},
		NewValues: map[string]any{},
		CreatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO monitor_audit_log`)).
		WillReturnError(sql.ErrConnDone)

	err = repo.Create(context.Background(), entry)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_GetByMonitorID_Success тестирует успешное получение audit log по monitor_id.
func TestAuditRepository_GetByMonitorID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()
	entryID := uuid.New()
	now := time.Now()

	oldValues := map[string]any{"status": "UP"}
	newValues := map[string]any{"status": "DOWN"}
	oldValuesJSON := mustMarshalJSON(t, oldValues)
	newValuesJSON := mustMarshalJSON(t, newValues)

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "user_id", "action", "old_values", "new_values", "ip_address", "user_agent", "created_at",
	}).AddRow(
		entryID, monitorID, userID, interfaces.ActionMonitorUpdate,
		oldValuesJSON, newValuesJSON, "192.168.1.1", "test-agent", now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnRows(rows)

	entries, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entryID, entries[0].ID)
	assert.Equal(t, interfaces.ActionMonitorUpdate, entries[0].Action)
	assert.Equal(t, "UP", entries[0].OldValues["status"])
	assert.Equal(t, "DOWN", entries[0].NewValues["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_GetByMonitorID_Empty тестирует получение пустого audit log.
func TestAuditRepository_GetByMonitorID_Empty(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	monitorID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "user_id", "action", "old_values", "new_values", "ip_address", "user_agent", "created_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnRows(rows)

	entries, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, entries, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_GetByUserID_Success тестирует получение audit log по user_id.
func TestAuditRepository_GetByUserID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	userID := uuid.New()
	monitorID := uuid.New()
	entryID := uuid.New()
	now := time.Now()

	oldValuesJSON := mustMarshalJSON(t, map[string]any{})
	newValuesJSON := mustMarshalJSON(t, map[string]any{"name": "New Monitor"})

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "user_id", "action", "old_values", "new_values", "ip_address", "user_agent", "created_at",
	}).AddRow(
		entryID, monitorID, userID, interfaces.ActionMonitorCreate,
		oldValuesJSON, newValuesJSON, "192.168.1.1", "test-agent", now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(userID, 10, 0).WillReturnRows(rows)

	entries, err := repo.GetByUserID(context.Background(), userID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entryID, entries[0].ID)
	assert.Equal(t, interfaces.ActionMonitorCreate, entries[0].Action)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_DeleteOld_Success тестирует удаление старых audit log entries.
func TestAuditRepository_DeleteOld_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	olderThan := time.Now().Add(-90 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM monitor_audit_log WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnResult(sqlmock.NewResult(0, 500))

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(500), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_DeleteOld_ZeroRows тестирует удаление, когда нет старых записей.
func TestAuditRepository_DeleteOld_ZeroRows(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	olderThan := time.Now().Add(-90 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM monitor_audit_log WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnResult(sqlmock.NewResult(0, 0))

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_GetByMonitorID_Pagination тестирует пагинацию.
func TestAuditRepository_GetByMonitorID_Pagination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
	}{
		{
			name:          "first page",
			limit:         20,
			offset:        0,
			expectedCount: 20,
		},
		{
			name:          "second page",
			limit:         20,
			offset:        20,
			expectedCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			closeTestDB(t, db)

			repo := NewAuditRepository(db)

			monitorID := uuid.New()
			userID := uuid.New()
			now := time.Now()

			oldValuesJSON := mustMarshalJSON(t, map[string]any{})
			newValuesJSON := mustMarshalJSON(t, map[string]any{})

			rows := sqlmock.NewRows([]string{
				"id", "monitor_id", "user_id", "action", "old_values", "new_values", "ip_address", "user_agent", "created_at",
			})

			for i := 0; i < tt.expectedCount; i++ {
				rows.AddRow(
					uuid.New(), monitorID, userID, interfaces.ActionMonitorUpdate,
					oldValuesJSON, newValuesJSON, "192.168.1.1", "test-agent", now,
				)
			}

			mock.ExpectQuery(regexp.QuoteMeta(`
				SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
				FROM monitor_audit_log
				WHERE monitor_id = $1
				ORDER BY created_at DESC
				LIMIT $2 OFFSET $3
			`)).WithArgs(monitorID, tt.limit, tt.offset).WillReturnRows(rows)

			entries, err := repo.GetByMonitorID(context.Background(), monitorID, tt.limit, tt.offset)
			assert.NoError(t, err)
			assert.Len(t, entries, tt.expectedCount)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestAuditRepository_Create_WithEmptyValues тестирует создание с пустыми old/new values.
func TestAuditRepository_Create_WithEmptyValues(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()

	entry := &interfaces.AuditLogEntry{
		ID:        uuid.New(),
		MonitorID: monitorID,
		UserID:    userID,
		Action:    interfaces.ActionMonitorDelete,
		OldValues: map[string]any{},
		NewValues: map[string]any{},
		CreatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO monitor_audit_log`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), entry)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_GetByMonitorID_WithComplexValues тестирует получение со сложными значениями.
func TestAuditRepository_GetByMonitorID_WithComplexValues(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()
	entryID := uuid.New()
	now := time.Now()

	oldValues := map[string]any{
		"name":             "Old Name",
		"interval_seconds": 60,
		"timeout_seconds":  30,
		"tags":             []string{"tag1", "tag2"},
	}
	newValues := map[string]any{
		"name":             "New Name",
		"interval_seconds": 120,
		"timeout_seconds":  60,
		"tags":             []string{"tag1", "tag2", "tag3"},
	}
	oldValuesJSON := mustMarshalJSON(t, oldValues)
	newValuesJSON := mustMarshalJSON(t, newValues)

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "user_id", "action", "old_values", "new_values", "ip_address", "user_agent", "created_at",
	}).AddRow(
		entryID, monitorID, userID, interfaces.ActionMonitorUpdate,
		oldValuesJSON, newValuesJSON, "192.168.1.1", "test-agent", now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnRows(rows)

	entries, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entryID, entries[0].ID)
	assert.Equal(t, "Old Name", entries[0].OldValues["name"])
	assert.Equal(t, 60, int(entries[0].OldValues["interval_seconds"].(float64)))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_GetByMonitorID_Error тестирует ошибку при получении audit log entries.
func TestAuditRepository_GetByMonitorID_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnError(sql.ErrConnDone)

	entries, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.Error(t, err)
	assert.Nil(t, entries)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_GetByUserID_Error тестирует ошибку при получении audit log entries по user_id.
func TestAuditRepository_GetByUserID_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(userID, 10, 0).WillReturnError(sql.ErrConnDone)

	entries, err := repo.GetByUserID(context.Background(), userID, 10, 0)
	assert.Error(t, err)
	assert.Nil(t, entries)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_DeleteOld_Error тестирует ошибку при удалении старых записей.
func TestAuditRepository_DeleteOld_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	olderThan := time.Now().Add(-90 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM monitor_audit_log WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnError(sql.ErrConnDone)

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.Error(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditRepository_scanAuditLogRows_UnmarshalError тестирует ошибку при unmarshal JSON.

// TestAuditRepository_Create_MarshalError тестирует ошибку при маршалинге.
func TestAuditRepository_Create_MarshalError(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewAuditRepository(db)

	// Создаём entry с невалидными данными для JSON
	entry := &interfaces.AuditLogEntry{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		Action:    "UPDATE",
		OldValues: map[string]any{"func": func() {}}, // функция не маршалится в JSON
		NewValues: map[string]any{},
		CreatedAt: time.Now(),
	}

	err = repo.Create(context.Background(), entry)
	assert.Error(t, err)
}
