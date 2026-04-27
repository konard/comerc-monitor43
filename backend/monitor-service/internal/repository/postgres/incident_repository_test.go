// Package postgres предоставляет реализацию репозиториев для работы с PostgreSQL.
// Unit тесты используют sqlmock для мокания SQL-запросов.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
)

// TestIncidentRepository_Create_Success тестирует успешное создание инцидента.
func TestIncidentRepository_Create_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()
	startTime := time.Now()

	incident := &domain.Incident{
		ID:        uuid.New(),
		MonitorID: monitorID,
		StartTime: startTime,
		Status:    domain.IncidentActive,
		CreatedAt: startTime,
	}

	// sqlx.NamedExec конвертирует :param в $1, $2, etc., поэтому ожидаем простой INSERT
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO incidents`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), incident)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_Create_Error тестирует ошибку при создании инцидента.
func TestIncidentRepository_Create_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()
	incident := &domain.Incident{
		ID:        uuid.New(),
		MonitorID: monitorID,
		StartTime: time.Now(),
		Status:    domain.IncidentActive,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO incidents`)).
		WillReturnError(sql.ErrConnDone)

	err = repo.Create(context.Background(), incident)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetByID_Success тестирует успешное получение инцидента по ID.
func TestIncidentRepository_GetByID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	incidentID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
	}).AddRow(
		incidentID, monitorID, startTime, nil, nil, "ACTIVE", startTime,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE id = $1
	`)).WithArgs(incidentID).WillReturnRows(rows)

	incident, err := repo.GetByID(context.Background(), incidentID)
	assert.NoError(t, err)
	assert.NotNil(t, incident)
	assert.Equal(t, incidentID, incident.ID)
	assert.Equal(t, domain.IncidentActive, incident.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetByID_NotFound тестирует случай, когда инцидент не найден.
func TestIncidentRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	incidentID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE id = $1
	`)).WithArgs(incidentID).WillReturnError(sql.ErrNoRows)

	incident, err := repo.GetByID(context.Background(), incidentID)
	assert.Error(t, err)
	assert.Nil(t, incident)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetByMonitorID_Success тестирует получение инцидентов по monitor_id.
func TestIncidentRepository_GetByMonitorID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()
	startTime := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
	}).
		AddRow(
			uuid.New(), monitorID, startTime, nil, nil, "ACTIVE", startTime,
		).
		AddRow(
			uuid.New(), monitorID, startTime.Add(-time.Hour), startTime.Add(-30*time.Minute),
			int64(1800), "RESOLVED", startTime.Add(-time.Hour),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1
		ORDER BY start_time DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnRows(rows)

	incidents, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, incidents, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetByMonitorID_Empty тестирует получение пустого списка инцидентов.
func TestIncidentRepository_GetByMonitorID_Empty(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1
		ORDER BY start_time DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnRows(rows)

	incidents, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, incidents, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetByMonitorIDAndPeriod_Success тестирует получение инцидентов за период.
func TestIncidentRepository_GetByMonitorIDAndPeriod_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
	}).AddRow(
		uuid.New(), monitorID, time.Now().Add(-12*time.Hour), time.Now().Add(-6*time.Hour),
		int64(21600), "RESOLVED", time.Now().Add(-12*time.Hour),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1 AND start_time >= $2 AND start_time <= $3
		ORDER BY start_time DESC
	`)).WithArgs(monitorID, from, to).WillReturnRows(rows)

	incidents, err := repo.GetByMonitorIDAndPeriod(context.Background(), monitorID, from, to)
	assert.NoError(t, err)
	assert.Len(t, incidents, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetActiveByMonitorID_Success тестирует получение активных инцидентов.
func TestIncidentRepository_GetActiveByMonitorID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()
	startTime := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
	}).AddRow(
		uuid.New(), monitorID, startTime, nil, nil, "ACTIVE", startTime,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1 AND status = 'ACTIVE'
		ORDER BY start_time DESC
	`)).WithArgs(monitorID).WillReturnRows(rows)

	incidents, err := repo.GetActiveByMonitorID(context.Background(), monitorID)
	assert.NoError(t, err)
	assert.Len(t, incidents, 1)
	assert.Equal(t, domain.IncidentActive, incidents[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetActiveByMonitorID_NoneActive тестирует отсутствие активных инцидентов.
func TestIncidentRepository_GetActiveByMonitorID_NoneActive(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1 AND status = 'ACTIVE'
		ORDER BY start_time DESC
	`)).WithArgs(monitorID).WillReturnRows(rows)

	incidents, err := repo.GetActiveByMonitorID(context.Background(), monitorID)
	assert.NoError(t, err)
	assert.Len(t, incidents, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_Update_Success тестирует успешное обновление инцидента.
func TestIncidentRepository_Update_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	incidentID := uuid.New()
	monitorID := uuid.New()
	endTime := time.Now()
	duration := 3600

	incident := &domain.Incident{
		ID:              incidentID,
		MonitorID:       monitorID,
		StartTime:       time.Now().Add(-time.Hour),
		EndTime:         &endTime,
		DurationSeconds: &duration,
		Status:          domain.IncidentResolved,
		CreatedAt:       time.Now().Add(-time.Hour),
	}

	// sqlx.NamedExec конвертирует :param в $1, $2, etc.
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE incidents SET`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Update(context.Background(), incident)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_Update_NotFound тестирует обновление несуществующего инцидента.
func TestIncidentRepository_Update_NotFound(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	incidentID := uuid.New()
	monitorID := uuid.New()
	endTime := time.Now()
	duration := 3600

	incident := &domain.Incident{
		ID:              incidentID,
		MonitorID:       monitorID,
		StartTime:       time.Now().Add(-time.Hour),
		EndTime:         &endTime,
		DurationSeconds: &duration,
		Status:          domain.IncidentResolved,
		CreatedAt:       time.Now().Add(-time.Hour),
	}

	// sqlx.NamedExec конвертирует :param в $1, $2, etc.
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE incidents SET`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Update(context.Background(), incident)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_Resolve_Success тестирует успешное разрешение инцидента.
func TestIncidentRepository_Resolve_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	incidentID := uuid.New()
	endTime := time.Now()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE incidents SET
			end_time = $1,
			duration_seconds = EXTRACT(EPOCH FROM ($1 - start_time))::INTEGER,
			status = 'RESOLVED'
		WHERE id = $2 AND status = 'ACTIVE'
	`)).WithArgs(endTime, incidentID).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Resolve(context.Background(), incidentID, endTime)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_Resolve_NotFound тестирует разрешение несуществующего инцидента.
func TestIncidentRepository_Resolve_NotFound(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	incidentID := uuid.New()
	endTime := time.Now()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE incidents SET
			end_time = $1,
			duration_seconds = EXTRACT(EPOCH FROM ($1 - start_time))::INTEGER,
			status = 'RESOLVED'
		WHERE id = $2 AND status = 'ACTIVE'
	`)).WithArgs(endTime, incidentID).WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Resolve(context.Background(), incidentID, endTime)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_DeleteOld_Success тестирует удаление старых инцидентов.
func TestIncidentRepository_DeleteOld_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	olderThan := time.Now().Add(-90 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM incidents WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnResult(sqlmock.NewResult(0, 25))

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(25), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_DeleteOld_ZeroRows тестирует удаление, когда нет старых записей.
func TestIncidentRepository_DeleteOld_ZeroRows(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	olderThan := time.Now().Add(-90 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM incidents WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnResult(sqlmock.NewResult(0, 0))

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_CountByMonitorID_Success тестирует подсчёт инцидентов монитора.
func TestIncidentRepository_CountByMonitorID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM incidents WHERE monitor_id = $1`)).
		WithArgs(monitorID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

	count, err := repo.CountByMonitorID(context.Background(), monitorID)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_GetByMonitorID_Pagination тестирует пагинацию.
func TestIncidentRepository_GetByMonitorID_Pagination(t *testing.T) {
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

			repo := NewIncidentRepository(db)

			monitorID := uuid.New()
			startTime := time.Now()

			rows := sqlmock.NewRows([]string{
				"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
			})

			for i := 0; i < tt.expectedCount; i++ {
				rows.AddRow(
					uuid.New(), monitorID, startTime.Add(-time.Duration(i)*time.Hour), nil, nil, "ACTIVE", startTime,
				)
			}

			mock.ExpectQuery(regexp.QuoteMeta(`
				SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
				FROM incidents
				WHERE monitor_id = $1
				ORDER BY start_time DESC
				LIMIT $2 OFFSET $3
			`)).WithArgs(monitorID, tt.limit, tt.offset).WillReturnRows(rows)

			incidents, err := repo.GetByMonitorID(context.Background(), monitorID, tt.limit, tt.offset)
			assert.NoError(t, err)
			assert.Len(t, incidents, tt.expectedCount)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestIncidentRepository_Create_Resolved тестирует создание сразу разрешённого инцидента.
func TestIncidentRepository_Create_Resolved(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	monitorID := uuid.New()
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)
	duration := 300

	incident := &domain.Incident{
		ID:              uuid.New(),
		MonitorID:       monitorID,
		StartTime:       startTime,
		EndTime:         &endTime,
		DurationSeconds: &duration,
		Status:          domain.IncidentResolved,
		CreatedAt:       startTime,
	}

	// sqlx.NamedExec конвертирует :param в $1, $2, etc., поэтому ожидаем простой INSERT
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO incidents`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), incident)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestIncidentRepository_dbListToIncidents_InvalidStatus тестирует конвертацию списка с невалидным статусом.
func TestIncidentRepository_dbListToIncidents_InvalidStatus(t *testing.T) {
	t.Parallel()
	repo := &incidentRepository{}

	now := time.Now()
	dbIncidents := []dbIncident{
		{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			StartTime: now,
			Status:    string(domain.IncidentActive),
			CreatedAt: now,
		},
		{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			StartTime: now,
			Status:    "INVALID_STATUS",
			CreatedAt: now,
		},
	}

	_, err := repo.dbListToIncidents(dbIncidents)
	assert.Error(t, err)
}

// TestIncidentRepository_DeleteOld_RowsAffectedError тестирует ошибку при получении количества удалённых строк.
func TestIncidentRepository_DeleteOld_RowsAffectedError(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	olderThan := time.Now().Add(-30 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM incidents WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnResult(&rowsAffectedErrorMock{})

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.Error(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// rowsAffectedErrorMock реализует sql.Result с ошибкой при RowsAffected.
type rowsAffectedErrorMock struct{}

func (m *rowsAffectedErrorMock) LastInsertId() (int64, error) {
	return 0, nil
}

func (m *rowsAffectedErrorMock) RowsAffected() (int64, error) {
	return 0, errors.New("rows affected error")
}

// TestIncidentRepository_GetByID_ScanError тестирует ошибку при сканировании.
func TestIncidentRepository_GetByID_ScanError(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewIncidentRepository(db)

	incidentID := uuid.New()

	// Возвращаем неверный тип для duration_seconds
	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "start_time", "end_time", "duration_seconds", "status", "created_at",
	}).AddRow(
		incidentID, uuid.New(), time.Now(), nil, "not_a_number", "ACTIVE", time.Now(),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE id = $1
	`)).WithArgs(incidentID).WillReturnRows(rows)

	incident, err := repo.GetByID(context.Background(), incidentID)
	assert.Error(t, err)
	assert.Nil(t, incident)
}
