package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

type maintenanceWindowRepository struct {
	db *sqlx.DB
}

// NewMaintenanceWindowRepository создаёт новый репозиторий окон обслуживания.
func NewMaintenanceWindowRepository(db *sql.DB) interfaces.MaintenanceWindowRepository {
	return &maintenanceWindowRepository{
		db: sqlx.NewDb(db, "postgres"),
	}
}

func (r *maintenanceWindowRepository) Create(ctx context.Context, window *interfaces.MaintenanceWindow) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer rollbackOnError(tx, &err, "failed to rollback maintenance window transaction")

	query := `
		INSERT INTO maintenance_windows (
			id, user_id, name, start_time, end_time, status, recurrence,
			is_global, pause_monitoring, suppress_alerts, safe_mode,
			created_at, updated_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = tx.ExecContext(ctx, query,
		window.ID, window.UserID, window.Name, window.StartTime, window.EndTime,
		window.Status, window.Recurrence, window.IsGlobal, window.PauseMonitoring,
		window.SuppressAlerts, window.SafeMode, window.CreatedAt, window.UpdatedAt, window.Version,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create maintenance window")
	}

	// Добавляем мониторы к окну
	if len(window.MonitorIDs) > 0 && !window.IsGlobal {
		if err := r.addMonitorsToTx(ctx, tx, window.ID, window.MonitorIDs); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

func (r *maintenanceWindowRepository) GetByID(ctx context.Context, id uuid.UUID) (*interfaces.MaintenanceWindow, error) {
	query := `
		SELECT id, user_id, name, start_time, end_time, status, recurrence,
			   is_global, pause_monitoring, suppress_alerts, safe_mode,
			   created_at, updated_at, activated_at, completed_at, version
		FROM maintenance_windows
		WHERE id = $1
	`

	window := &interfaces.MaintenanceWindow{}
	err := r.db.GetContext(ctx, window, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get maintenance window by id")
	}

	// Загружаем мониторы
	monitorIDs, err := r.GetWindowMonitors(ctx, id)
	if err != nil {
		return nil, err
	}
	window.MonitorIDs = monitorIDs

	return window, nil
}

func (r *maintenanceWindowRepository) GetByUserID(ctx context.Context, userID uuid.UUID, status string, limit, offset int) (_ []*interfaces.MaintenanceWindow, err error) {
	query := `
		SELECT id, user_id, name, start_time, end_time, status, recurrence,
			   is_global, pause_monitoring, suppress_alerts, safe_mode,
			   created_at, updated_at, activated_at, completed_at, version
		FROM maintenance_windows
		WHERE user_id = $1
	`

	args := []any{userID}
	argPos := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get maintenance windows by user_id")
	}
	defer closeResource(rows, &err, "failed to close maintenance window rows")

	return r.scanRows(ctx, rows)
}

func (r *maintenanceWindowRepository) GetActiveWindowsForMonitor(ctx context.Context, monitorID uuid.UUID, at time.Time) (_ []*interfaces.MaintenanceWindow, err error) {
	query := `
		SELECT DISTINCT w.id, w.user_id, w.name, w.start_time, w.end_time, w.status,
		       w.recurrence, w.is_global, w.pause_monitoring, w.suppress_alerts, w.safe_mode,
		       w.created_at, w.updated_at, w.activated_at, w.completed_at, w.version
		FROM maintenance_windows w
		LEFT JOIN maintenance_window_monitors mwm ON w.id = mwm.maintenance_window_id
		WHERE w.status = 'ACTIVE'
		  AND w.start_time <= $1
		  AND w.end_time > $1
		  AND (w.is_global = true OR mwm.monitor_id = $2)
		ORDER BY w.start_time
	`

	rows, err := r.db.QueryContext(ctx, query, at, monitorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active windows for monitor")
	}
	defer closeResource(rows, &err, "failed to close active maintenance window rows")

	return r.scanRows(ctx, rows)
}

func (r *maintenanceWindowRepository) GetActiveWindowsForUser(ctx context.Context, userID uuid.UUID, at time.Time) (_ []*interfaces.MaintenanceWindow, err error) {
	query := `
		SELECT id, user_id, name, start_time, end_time, status, recurrence,
		       is_global, pause_monitoring, suppress_alerts, safe_mode,
		       created_at, updated_at, activated_at, completed_at, version
		FROM maintenance_windows
		WHERE user_id = $1
		  AND status = 'ACTIVE'
		  AND start_time <= $2
		  AND end_time > $2
		ORDER BY start_time
	`

	rows, err := r.db.QueryContext(ctx, query, userID, at)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active windows for user")
	}
	defer closeResource(rows, &err, "failed to close user maintenance window rows")

	return r.scanRows(ctx, rows)
}

func (r *maintenanceWindowRepository) Update(ctx context.Context, window *interfaces.MaintenanceWindow) error {
	query := `
		UPDATE maintenance_windows
		SET name = $2, start_time = $3, end_time = $4,
		    status = $5, updated_at = $6, version = $7
		WHERE id = $1 AND version = $8
	`

	result, err := r.db.ExecContext(ctx, query,
		window.ID, window.Name, window.StartTime, window.EndTime,
		window.Status, window.UpdatedAt, window.Version, window.Version-1,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update maintenance window")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.New("version conflict or window not found")
	}

	return nil
}

func (r *maintenanceWindowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM maintenance_windows WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete maintenance window")
	}

	return nil
}

func (r *maintenanceWindowRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM maintenance_windows WHERE user_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count maintenance windows")
	}

	return count, nil
}

func (r *maintenanceWindowRepository) CheckOverlap(ctx context.Context, userID uuid.UUID, monitorIDs []uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM maintenance_windows w
			WHERE w.user_id = $1
			  AND w.status IN ('SCHEDULED', 'ACTIVE')
			  AND w.start_time < $3
			  AND w.end_time > $2
	`

	args := []any{userID, startTime, endTime}
	argPos := 4

	if excludeID != nil {
		query += fmt.Sprintf(" AND w.id != $%d", argPos)
		args = append(args, *excludeID)
		argPos++
	}

	if len(monitorIDs) > 0 {
		query += fmt.Sprintf(`
			  AND (w.is_global = true OR EXISTS(
				  SELECT 1 FROM maintenance_window_monitors mwm
				  WHERE mwm.maintenance_window_id = w.id
				  AND mwm.monitor_id = ANY($%d)
			  ))
		`, argPos)
		args = append(args, monitorIDs)
	} else {
		// Для глобальных окон или окон без мониторов
		query += " AND w.is_global = true"
	}

	query += ")"

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, args...)
	if err != nil {
		return false, errors.Wrap(err, "failed to check maintenance window overlap")
	}

	return exists, nil
}

func (r *maintenanceWindowRepository) GetWindowsRequiringActivation(ctx context.Context, before time.Time) (_ []*interfaces.MaintenanceWindow, err error) {
	query := `
		SELECT id, user_id, name, start_time, end_time, status, recurrence,
		       is_global, pause_monitoring, suppress_alerts, safe_mode,
		       created_at, updated_at, activated_at, completed_at, version
		FROM maintenance_windows
		WHERE status = 'SCHEDULED'
		  AND start_time <= $1
		ORDER BY start_time
	`

	rows, err := r.db.QueryContext(ctx, query, before)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get windows requiring activation")
	}
	defer closeResource(rows, &err, "failed to close activation rows")

	return r.scanRows(ctx, rows)
}

func (r *maintenanceWindowRepository) GetWindowsRequiringCompletion(ctx context.Context, before time.Time) (_ []*interfaces.MaintenanceWindow, err error) {
	query := `
		SELECT id, user_id, name, start_time, end_time, status, recurrence,
		       is_global, pause_monitoring, suppress_alerts, safe_mode,
		       created_at, updated_at, activated_at, completed_at, version
		FROM maintenance_windows
		WHERE status = 'ACTIVE'
		  AND end_time <= $1
		ORDER BY end_time
	`

	rows, err := r.db.QueryContext(ctx, query, before)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get windows requiring completion")
	}
	defer closeResource(rows, &err, "failed to close completion rows")

	return r.scanRows(ctx, rows)
}

func (r *maintenanceWindowRepository) AddMonitorsToWindow(ctx context.Context, windowID uuid.UUID, monitorIDs []uuid.UUID) error {
	query := `
		INSERT INTO maintenance_window_monitors (maintenance_window_id, monitor_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

	now := time.Now()
	for _, monitorID := range monitorIDs {
		_, err := r.db.ExecContext(ctx, query, windowID, monitorID, now)
		if err != nil {
			return errors.Wrap(err, "failed to add monitors to window")
		}
	}

	return nil
}

func (r *maintenanceWindowRepository) RemoveMonitorsFromWindow(ctx context.Context, windowID uuid.UUID, monitorIDs []uuid.UUID) error {
	query := `DELETE FROM maintenance_window_monitors WHERE maintenance_window_id = $1 AND monitor_id = ANY($2)`

	_, err := r.db.ExecContext(ctx, query, windowID, monitorIDs)
	if err != nil {
		return errors.Wrap(err, "failed to remove monitors from window")
	}

	return nil
}

func (r *maintenanceWindowRepository) GetWindowMonitors(ctx context.Context, windowID uuid.UUID) (_ []uuid.UUID, err error) {
	query := `SELECT monitor_id FROM maintenance_window_monitors WHERE maintenance_window_id = $1`

	rows, err := r.db.QueryContext(ctx, query, windowID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get window monitors")
	}
	defer closeResource(rows, &err, "failed to close maintenance window monitor rows")

	var monitorIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, "failed to scan monitor id")
		}
		monitorIDs = append(monitorIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating monitor ids")
	}

	return monitorIDs, nil
}

func (r *maintenanceWindowRepository) GetHistory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, monitorIDs []uuid.UUID, limit, offset int) (_ []*interfaces.MaintenanceWindow, err error) {
	query := `
		SELECT DISTINCT w.id, w.user_id, w.name, w.start_time, w.end_time, w.status,
		       w.recurrence, w.is_global, w.pause_monitoring, w.suppress_alerts, w.safe_mode,
		       w.created_at, w.updated_at, w.activated_at, w.completed_at, w.version
		FROM maintenance_windows w
		LEFT JOIN maintenance_window_monitors mwm ON w.id = mwm.maintenance_window_id
		WHERE w.user_id = $1
		  AND w.status IN ('COMPLETED', 'CANCELLED')
		  AND w.created_at >= $2
		  AND w.created_at <= $3
	`

	args := []any{userID, startDate, endDate}
	argPos := 4

	if len(monitorIDs) > 0 {
		query += fmt.Sprintf(" AND mwm.monitor_id = ANY($%d)", argPos)
		args = append(args, monitorIDs)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY w.created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get maintenance window history")
	}
	defer closeResource(rows, &err, "failed to close maintenance window history rows")

	return r.scanRows(ctx, rows)
}

// addMonitorsToTx добавляет мониторы к окну в транзакции.
func (r *maintenanceWindowRepository) addMonitorsToTx(ctx context.Context, tx *sqlx.Tx, windowID uuid.UUID, monitorIDs []uuid.UUID) error {
	query := `
		INSERT INTO maintenance_window_monitors (maintenance_window_id, monitor_id, created_at)
		VALUES ($1, $2, $3)
	`

	now := time.Now()
	for _, monitorID := range monitorIDs {
		_, err := tx.ExecContext(ctx, query, windowID, monitorID, now)
		if err != nil {
			return errors.Wrap(err, "failed to add monitors to window")
		}
	}

	return nil
}

// scanRows сканирует строки в []*interfaces.MaintenanceWindow.
func (r *maintenanceWindowRepository) scanRows(ctx context.Context, rows *sql.Rows) ([]*interfaces.MaintenanceWindow, error) {
	var windows []*interfaces.MaintenanceWindow

	for rows.Next() {
		window := &interfaces.MaintenanceWindow{}
		err := rows.Scan(
			&window.ID, &window.UserID, &window.Name, &window.StartTime, &window.EndTime,
			&window.Status, &window.Recurrence, &window.IsGlobal, &window.PauseMonitoring,
			&window.SuppressAlerts, &window.SafeMode, &window.CreatedAt, &window.UpdatedAt,
			&window.ActivatedAt, &window.CompletedAt, &window.Version,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan maintenance window row")
		}
		windows = append(windows, window)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating maintenance window rows")
	}

	return windows, nil
}
