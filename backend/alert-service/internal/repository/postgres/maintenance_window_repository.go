package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// MaintenanceWindowRepository реализует интерфейс MaintenanceWindowRepository для PostgreSQL
type MaintenanceWindowRepository struct {
	db *DB
}

// NewMaintenanceWindowRepository создаёт новый экземпляр MaintenanceWindowRepository
func NewMaintenanceWindowRepository(db *DB) *MaintenanceWindowRepository {
	return &MaintenanceWindowRepository{db: db}
}

// Create создаёт окно обслуживания
func (r *MaintenanceWindowRepository) Create(ctx context.Context, mw *model.MaintenanceWindow) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("user_id", mw.UserID.String()),
			attribute.String("window_id", mw.ID.String()),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO maintenance_windows (
			id, user_id, monitor_id, name, status, recurrence,
			is_global, pause_monitoring, suppress_alerts, safe_mode,
			monitor_ids, starts_at, ends_at, activated_at, completed_at,
			version, reason, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19
		)
	`

	// monitorID используется для обратной совместимости со старой схемой
	var monitorID any
	if len(mw.MonitorIDs) > 0 {
		monitorID = mw.MonitorIDs[0]
	}

	_, err := r.db.ExecContext(ctx, query,
		mw.ID, mw.UserID, monitorID, mw.Name, string(mw.Status), string(mw.Recurrence),
		mw.IsGlobal, mw.PauseMonitoring, mw.SuppressAlerts, mw.SafeMode,
		pq.Array(mw.MonitorIDs), mw.StartsAt, mw.EndsAt, mw.ActivatedAt, mw.CompletedAt,
		mw.Version, mw.Reason, mw.CreatedAt, mw.UpdatedAt,
	)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create maintenance window: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "maintenance_windows", float64(duration))
	}

	return nil
}

// GetByID возвращает окно обслуживания по ID
func (r *MaintenanceWindowRepository) GetByID(ctx context.Context, id string) (*model.MaintenanceWindow, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.GetByID")
		defer span.End()
		span.SetAttributes(attribute.String("window_id", id))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, name, status, recurrence,
		       is_global, pause_monitoring, suppress_alerts, safe_mode,
		       monitor_ids, starts_at, ends_at, activated_at, completed_at,
		       version, reason, created_at, updated_at
		FROM maintenance_windows
		WHERE id = $1
	`

	mw, err := r.scanRow(ctx, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrMaintenanceWindowNotFound
		}
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get maintenance window by id: %v", err)
	}

	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "get_by_id", "maintenance_windows", float64(time.Since(startTime).Milliseconds()))
	}

	return mw, nil
}

// GetActiveByMonitorID возвращает активное окно обслуживания для монитора
func (r *MaintenanceWindowRepository) GetActiveByMonitorID(ctx context.Context, monitorID string) (*model.MaintenanceWindow, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.GetActiveByMonitorID")
		defer span.End()
		span.SetAttributes(attribute.String("monitor_id", monitorID))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, name, status, recurrence,
		       is_global, pause_monitoring, suppress_alerts, safe_mode,
		       monitor_ids, starts_at, ends_at, activated_at, completed_at,
		       version, reason, created_at, updated_at
		FROM maintenance_windows
		WHERE (is_global = true OR $1 = ANY(monitor_ids))
		  AND starts_at <= NOW() AND ends_at >= NOW()
		  AND status IN ('scheduled', 'active')
		ORDER BY created_at DESC
		LIMIT 1
	`

	mw, err := r.scanRow(ctx, query, monitorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get active maintenance window: %v", err)
	}

	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "get_active", "maintenance_windows", float64(time.Since(startTime).Milliseconds()))
	}

	return mw, nil
}

// IsUnderMaintenance проверяет, находится ли монитор в окне обслуживания
func (r *MaintenanceWindowRepository) IsUnderMaintenance(ctx context.Context, monitorID string) (bool, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.IsUnderMaintenance")
		defer span.End()
		span.SetAttributes(attribute.String("monitor_id", monitorID))
	}

	startTime := time.Now()

	query := `
		SELECT COUNT(*) FROM maintenance_windows
		WHERE (is_global = true OR $1 = ANY(monitor_ids))
		  AND starts_at <= NOW() AND ends_at >= NOW()
		  AND status IN ('scheduled', 'active')
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return false, fmt.Errorf("failed to check maintenance status: %v", err)
	}

	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "is_under_maintenance", "maintenance_windows", float64(time.Since(startTime).Milliseconds()))
	}

	return count > 0, nil
}

// Update обновляет окно обслуживания с оптимистичной блокировкой
func (r *MaintenanceWindowRepository) Update(ctx context.Context, mw *model.MaintenanceWindow) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.Update")
		defer span.End()
		span.SetAttributes(
			attribute.String("window_id", mw.ID.String()),
			attribute.Int("version", mw.Version),
		)
	}

	startTime := time.Now()

	query := `
		UPDATE maintenance_windows SET
			name = $1,
			status = $2,
			starts_at = $3,
			ends_at = $4,
			activated_at = $5,
			completed_at = $6,
			version = version + 1,
			updated_at = $7
		WHERE id = $8 AND version = $9
	`

	result, err := r.db.ExecContext(ctx, query,
		mw.Name, string(mw.Status),
		mw.StartsAt, mw.EndsAt,
		mw.ActivatedAt, mw.CompletedAt,
		mw.UpdatedAt,
		mw.ID, mw.Version,
	)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to update maintenance window: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return model.ErrMaintenanceWindowConflict
	}

	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "update", "maintenance_windows", float64(time.Since(startTime).Milliseconds()))
	}

	return nil
}

// Delete удаляет окно обслуживания по ID
func (r *MaintenanceWindowRepository) Delete(ctx context.Context, id string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.Delete")
		defer span.End()
		span.SetAttributes(attribute.String("window_id", id))
	}

	startTime := time.Now()

	query := `DELETE FROM maintenance_windows WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete maintenance window: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return model.ErrMaintenanceWindowNotFound
	}

	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "delete", "maintenance_windows", float64(time.Since(startTime).Milliseconds()))
	}

	return nil
}

// List возвращает список окон обслуживания с фильтрацией и пагинацией
func (r *MaintenanceWindowRepository) List(ctx context.Context, userID string, filter model.MaintenanceWindowFilter) (windows []*model.MaintenanceWindow, total int, err error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.List")
		defer span.End()
		span.SetAttributes(attribute.String("user_id", userID))
	}

	startTime := time.Now()

	conditions := []string{"user_id = $1"}
	args := []any{userID}
	argIdx := 2

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(filter.Status))
		argIdx++
	}
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("starts_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("ends_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM maintenance_windows WHERE %s`, where)
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, fmt.Errorf("failed to count maintenance windows: %v", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	listQuery := fmt.Sprintf(`
		SELECT id, user_id, monitor_id, name, status, recurrence,
		       is_global, pause_monitoring, suppress_alerts, safe_mode,
		       monitor_ids, starts_at, ends_at, activated_at, completed_at,
		       version, reason, created_at, updated_at
		FROM maintenance_windows
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryxContext(ctx, listQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, fmt.Errorf("failed to list maintenance windows: %v", err)
	}
	defer closeResource(rows, &err, "failed to close maintenance windows rows")

	for rows.Next() {
		mw, scanErr := r.scanRowFromRows(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("failed to scan maintenance window: %v", scanErr)
		}
		windows = append(windows, mw)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate maintenance windows: %v", err)
	}

	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "list", "maintenance_windows", float64(time.Since(startTime).Milliseconds()))
	}

	return windows, total, nil
}

// CheckOverlapping проверяет, пересекается ли новое окно с существующими для тех же мониторов
func (r *MaintenanceWindowRepository) CheckOverlapping(ctx context.Context, monitorIDs []string, startsAt, endsAt time.Time, excludeID string) (bool, *model.MaintenanceWindow, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MaintenanceWindowRepository.CheckOverlapping")
		defer span.End()
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, name, status, recurrence,
		       is_global, pause_monitoring, suppress_alerts, safe_mode,
		       monitor_ids, starts_at, ends_at, activated_at, completed_at,
		       version, reason, created_at, updated_at
		FROM maintenance_windows
		WHERE status IN ('scheduled', 'active')
		  AND ($1::text[] && monitor_ids)
		  AND starts_at < $2
		  AND ends_at > $3
		  AND ($4 = '' OR id::text != $4)
		LIMIT 1
	`

	mw, err := r.scanRow(ctx, query, pq.Array(monitorIDs), endsAt, startsAt, excludeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, nil
		}
		if span != nil {
			span.RecordError(err)
		}
		return false, nil, fmt.Errorf("failed to check overlapping windows: %v", err)
	}

	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "check_overlapping", "maintenance_windows", float64(time.Since(startTime).Milliseconds()))
	}

	return true, mw, nil
}

// rowScanner интерфейс для сканирования строк (sqlx.Rows)
type rowScanner interface {
	Scan(dest ...any) error
}

// scanRow сканирует одну строку из запроса
func (r *MaintenanceWindowRepository) scanRow(ctx context.Context, query string, args ...any) (*model.MaintenanceWindow, error) {
	row := r.db.QueryRowxContext(ctx, query, args...)
	return r.scanRowFromRows(row)
}

// scanRowFromRows сканирует строку из rowScanner (sqlx.Row или sqlx.Rows)
func (r *MaintenanceWindowRepository) scanRowFromRows(rows rowScanner) (*model.MaintenanceWindow, error) {
	var mw model.MaintenanceWindow
	var monitorIDs pq.StringArray
	var statusStr, recurrenceStr string
	var legacyMonitorID *string // для обратной совместимости со старой схемой

	err := rows.Scan(
		&mw.ID, &mw.UserID, &legacyMonitorID, &mw.Name, &statusStr, &recurrenceStr,
		&mw.IsGlobal, &mw.PauseMonitoring, &mw.SuppressAlerts, &mw.SafeMode,
		&monitorIDs, &mw.StartsAt, &mw.EndsAt, &mw.ActivatedAt, &mw.CompletedAt,
		&mw.Version, &mw.Reason, &mw.CreatedAt, &mw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	mw.Status = model.MaintenanceWindowStatus(statusStr)
	mw.Recurrence = model.RecurrenceType(recurrenceStr)
	mw.MonitorIDs = []string(monitorIDs)

	return &mw, nil
}
