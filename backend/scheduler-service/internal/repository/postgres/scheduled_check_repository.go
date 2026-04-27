package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

type ScheduledCheckRepository struct {
	db *sql.DB
}

func NewScheduledCheckRepository(db *sql.DB) *ScheduledCheckRepository {
	return &ScheduledCheckRepository{db: db}
}

func (r *ScheduledCheckRepository) Create(ctx context.Context, check *model.ScheduledCheck) error {
	query := `
		INSERT INTO scheduled_checks (id, monitor_id, worker_id, priority, scheduled_at, executed_at, completed_at, status, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	var workerID any
	if check.WorkerID != uuid.Nil {
		workerID = check.WorkerID
	}
	_, err := r.db.ExecContext(ctx, query,
		check.ID, check.MonitorID, workerID, string(check.Priority),
		check.ScheduledAt, check.ExecutedAt, check.CompletedAt,
		string(check.Status), check.ErrorMessage,
		check.CreatedAt, check.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create scheduled check: %w", err)
	}
	return nil
}

func (r *ScheduledCheckRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ScheduledCheck, error) {
	query := `
		SELECT id, monitor_id, worker_id, priority, scheduled_at, executed_at, completed_at, status, error_message, created_at, updated_at
		FROM scheduled_checks WHERE id = $1
	`
	return r.scanCheck(ctx, query, id)
}

func (r *ScheduledCheckRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID) (*model.ScheduledCheck, error) {
	query := `
		SELECT id, monitor_id, worker_id, priority, scheduled_at, executed_at, completed_at, status, error_message, created_at, updated_at
		FROM scheduled_checks
		WHERE monitor_id = $1 AND status IN ('PENDING', 'IN_PROGRESS')
		ORDER BY scheduled_at DESC LIMIT 1
	`
	return r.scanCheck(ctx, query, monitorID)
}

func (r *ScheduledCheckRepository) Update(ctx context.Context, check *model.ScheduledCheck) error {
	query := `
		UPDATE scheduled_checks
		SET worker_id = $2, status = $3, executed_at = $4, completed_at = $5,
		    error_message = $6, updated_at = $7
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		check.ID, check.WorkerID, string(check.Status),
		check.ExecutedAt, check.CompletedAt, check.ErrorMessage,
		check.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update scheduled check: %w", err)
	}
	return nil
}

func (r *ScheduledCheckRepository) ListPendingByWorkerID(ctx context.Context, workerID uuid.UUID) ([]*model.ScheduledCheck, error) {
	query := `
		SELECT id, monitor_id, worker_id, priority, scheduled_at, executed_at, completed_at, status, error_message, created_at, updated_at
		FROM scheduled_checks
		WHERE worker_id = $1 AND status = 'PENDING'
		ORDER BY priority DESC, scheduled_at ASC
	`
	return r.scanChecks(ctx, query, workerID)
}

func (r *ScheduledCheckRepository) ListByTimeRange(ctx context.Context, from, to time.Time, status string, page, pageSize int) (_ []*model.ScheduledCheck, total int, err error) {
	args := []any{from, to}
	argIdx := 3
	where := "WHERE scheduled_at BETWEEN $1 AND $2"

	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	countQuery := "SELECT COUNT(*) FROM scheduled_checks " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count scheduled checks: %w", err)
	}

	offset := (page - 1) * pageSize
	listQuery := "SELECT id, monitor_id, worker_id, priority, scheduled_at, executed_at, completed_at, status, error_message, created_at, updated_at FROM scheduled_checks " + where + " ORDER BY scheduled_at ASC"
	listQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1) //nolint:gosec // G202: параметры пагинации используют placeholders, не user input
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list scheduled checks: %w", err)
	}
	defer closeResource(rows, &err, "failed to close scheduled check rows")

	checks := []*model.ScheduledCheck{}
	for rows.Next() {
		c := &model.ScheduledCheck{}
		if err := r.scanRow(rows, c); err != nil {
			return nil, 0, err
		}
		checks = append(checks, c)
	}
	return checks, total, nil
}

func (r *ScheduledCheckRepository) ListOverdue(ctx context.Context, threshold time.Duration) (_ []*model.ScheduledCheck, err error) {
	query := `
		SELECT id, monitor_id, worker_id, priority, scheduled_at, executed_at, completed_at, status, error_message, created_at, updated_at
		FROM scheduled_checks
		WHERE status = 'PENDING' AND scheduled_at < NOW() - $1::interval
		ORDER BY scheduled_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, fmt.Sprintf("%d seconds", int(threshold.Seconds())))
	if err != nil {
		return nil, fmt.Errorf("failed to list overdue checks: %w", err)
	}
	defer closeResource(rows, &err, "failed to close overdue scheduled check rows")

	checks := []*model.ScheduledCheck{}
	for rows.Next() {
		c := &model.ScheduledCheck{}
		if err := r.scanRow(rows, c); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, nil
}

func (r *ScheduledCheckRepository) ReassignByWorkerID(ctx context.Context, oldWorkerID uuid.UUID) (int, error) {
	query := `
		UPDATE scheduled_checks
		SET worker_id = NULL, status = 'PENDING', executed_at = NULL, updated_at = NOW()
		WHERE worker_id = $1 AND status IN ('PENDING', 'IN_PROGRESS')
	`
	result, err := r.db.ExecContext(ctx, query, oldWorkerID)
	if err != nil {
		return 0, fmt.Errorf("failed to reassign checks: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get reassigned rows count: %w", err)
	}
	return int(count), nil
}

func (r *ScheduledCheckRepository) DeleteCompletedBefore(ctx context.Context, before time.Time) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM scheduled_checks WHERE status = 'COMPLETED' AND completed_at < $1", before)
	if err != nil {
		return fmt.Errorf("failed to delete completed checks: %w", err)
	}
	return nil
}

func (r *ScheduledCheckRepository) HasPendingCheck(ctx context.Context, monitorID uuid.UUID) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM scheduled_checks WHERE monitor_id = $1 AND status IN ('PENDING', 'IN_PROGRESS'))"
	if err := r.db.QueryRowContext(ctx, query, monitorID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check pending: %w", err)
	}
	return exists, nil
}

func (r *ScheduledCheckRepository) scanCheck(ctx context.Context, query string, args ...any) (*model.ScheduledCheck, error) {
	c := &model.ScheduledCheck{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&c.ID, &c.MonitorID, &c.WorkerID, &c.Priority,
		&c.ScheduledAt, &c.ExecutedAt, &c.CompletedAt,
		&c.Status, &c.ErrorMessage,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scheduled check not found")
		}
		return nil, fmt.Errorf("failed to get scheduled check: %w", err)
	}
	return c, nil
}

func (r *ScheduledCheckRepository) scanChecks(ctx context.Context, query string, args ...any) (_ []*model.ScheduledCheck, err error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled checks: %w", err)
	}
	defer closeResource(rows, &err, "failed to close scheduled check rows")

	checks := []*model.ScheduledCheck{}
	for rows.Next() {
		c := &model.ScheduledCheck{}
		if err := r.scanRow(rows, c); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, nil
}

func (r *ScheduledCheckRepository) scanRow(rows *sql.Rows, c *model.ScheduledCheck) error {
	return rows.Scan(
		&c.ID, &c.MonitorID, &c.WorkerID, &c.Priority,
		&c.ScheduledAt, &c.ExecutedAt, &c.CompletedAt,
		&c.Status, &c.ErrorMessage,
		&c.CreatedAt, &c.UpdatedAt,
	)
}
