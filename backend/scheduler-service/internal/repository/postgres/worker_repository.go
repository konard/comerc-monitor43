package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

type WorkerRepository struct {
	db *sql.DB
}

func NewWorkerRepository(db *sql.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

func (r *WorkerRepository) Create(ctx context.Context, worker *model.Worker) error {
	query := `
		INSERT INTO scheduler_workers (id, name, zone, status, last_heartbeat, checks_completed, checks_failed, avg_check_duration_ms, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	metadataJSON := metadataToJSON(worker.Metadata)
	_, err := r.db.ExecContext(ctx, query,
		worker.ID, worker.Name, worker.Zone, string(worker.Status),
		worker.LastHeartbeat, worker.ChecksCompleted, worker.ChecksFailed,
		worker.AvgCheckDurationMs, metadataJSON,
		worker.CreatedAt, worker.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return fmt.Errorf("worker name already exists: %w", err)
		}
		return fmt.Errorf("failed to create worker: %w", err)
	}
	return nil
}

func (r *WorkerRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Worker, error) {
	query := `
		SELECT id, name, zone, status, last_heartbeat, checks_completed, checks_failed,
		       avg_check_duration_ms, metadata, created_at, updated_at
		FROM scheduler_workers WHERE id = $1
	`
	worker := &model.Worker{}
	var metadataStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&worker.ID, &worker.Name, &worker.Zone, &worker.Status,
		&worker.LastHeartbeat, &worker.ChecksCompleted, &worker.ChecksFailed,
		&worker.AvgCheckDurationMs, &metadataStr,
		&worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worker not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}
	worker.Metadata = jsonToMetadata(metadataStr)
	return worker, nil
}

func (r *WorkerRepository) GetByName(ctx context.Context, name string) (*model.Worker, error) {
	query := `
		SELECT id, name, zone, status, last_heartbeat, checks_completed, checks_failed,
		       avg_check_duration_ms, metadata, created_at, updated_at
		FROM scheduler_workers WHERE name = $1
	`
	worker := &model.Worker{}
	var metadataStr string
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&worker.ID, &worker.Name, &worker.Zone, &worker.Status,
		&worker.LastHeartbeat, &worker.ChecksCompleted, &worker.ChecksFailed,
		&worker.AvgCheckDurationMs, &metadataStr,
		&worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worker not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get worker by name: %w", err)
	}
	worker.Metadata = jsonToMetadata(metadataStr)
	return worker, nil
}

func (r *WorkerRepository) List(ctx context.Context, status, zone string, page, pageSize int) (_ []*model.Worker, total int, err error) {
	args := []any{}
	argIdx := 1
	where := "WHERE 1=1"

	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if zone != "" {
		where += fmt.Sprintf(" AND zone = $%d", argIdx)
		args = append(args, zone)
		argIdx++
	}

	countQuery := "SELECT COUNT(*) FROM scheduler_workers " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count workers: %w", err)
	}

	offset := (page - 1) * pageSize
	listQuery := "SELECT id, name, zone, status, last_heartbeat, checks_completed, checks_failed, avg_check_duration_ms, metadata, created_at, updated_at FROM scheduler_workers " + where + " ORDER BY created_at DESC"
	listQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1) //nolint:gosec // G202: параметры пагинации используют placeholders, не user input
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workers: %w", err)
	}
	defer closeResource(rows, &err, "failed to close worker rows")

	workers := []*model.Worker{}
	for rows.Next() {
		w := &model.Worker{}
		var metadataStr string
		if err := rows.Scan(
			&w.ID, &w.Name, &w.Zone, &w.Status,
			&w.LastHeartbeat, &w.ChecksCompleted, &w.ChecksFailed,
			&w.AvgCheckDurationMs, &metadataStr,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan worker: %w", err)
		}
		w.Metadata = jsonToMetadata(metadataStr)
		workers = append(workers, w)
	}

	return workers, total, nil
}

func (r *WorkerRepository) Update(ctx context.Context, worker *model.Worker) error {
	query := `
		UPDATE scheduler_workers
		SET status = $2, last_heartbeat = $3, checks_completed = $4,
		    checks_failed = $5, avg_check_duration_ms = $6, updated_at = $7
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		worker.ID, string(worker.Status), worker.LastHeartbeat,
		worker.ChecksCompleted, worker.ChecksFailed,
		worker.AvgCheckDurationMs, worker.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update worker: %w", err)
	}
	return nil
}

func (r *WorkerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM scheduler_workers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}
	return nil
}

func (r *WorkerRepository) ListExpired(ctx context.Context, timeout time.Duration) (_ []*model.Worker, err error) {
	query := `
		SELECT id, name, zone, status, last_heartbeat, checks_completed, checks_failed,
		       avg_check_duration_ms, metadata, created_at, updated_at
		FROM scheduler_workers
		WHERE status != 'OFFLINE' AND last_heartbeat < $1
	`
	rows, err := r.db.QueryContext(ctx, query, time.Now().Add(-timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to list expired workers: %w", err)
	}
	defer closeResource(rows, &err, "failed to close expired worker rows")

	workers := []*model.Worker{}
	for rows.Next() {
		w := &model.Worker{}
		var metadataStr string
		if err := rows.Scan(
			&w.ID, &w.Name, &w.Zone, &w.Status,
			&w.LastHeartbeat, &w.ChecksCompleted, &w.ChecksFailed,
			&w.AvgCheckDurationMs, &metadataStr,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expired worker: %w", err)
		}
		w.Metadata = jsonToMetadata(metadataStr)
		workers = append(workers, w)
	}
	return workers, nil
}

func (r *WorkerRepository) ListIdleByZone(ctx context.Context, zone string) (_ []*model.Worker, err error) {
	query := `
		SELECT id, name, zone, status, last_heartbeat, checks_completed, checks_failed,
		       avg_check_duration_ms, metadata, created_at, updated_at
		FROM scheduler_workers
		WHERE status = 'IDLE'
	`
	args := []any{}
	if zone != "" {
		query += " AND zone = $1"
		args = append(args, zone)
	}
	query += " ORDER BY checks_completed ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list idle workers: %w", err)
	}
	defer closeResource(rows, &err, "failed to close idle worker rows")

	workers := []*model.Worker{}
	for rows.Next() {
		w := &model.Worker{}
		var metadataStr string
		if err := rows.Scan(
			&w.ID, &w.Name, &w.Zone, &w.Status,
			&w.LastHeartbeat, &w.ChecksCompleted, &w.ChecksFailed,
			&w.AvgCheckDurationMs, &metadataStr,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan idle worker: %w", err)
		}
		w.Metadata = jsonToMetadata(metadataStr)
		workers = append(workers, w)
	}
	return workers, nil
}

func (r *WorkerRepository) ListOfflineForCleanup(ctx context.Context, cutoffTime time.Time) (_ []*model.Worker, err error) {
	query := `
		SELECT id, name, zone, status, last_heartbeat, checks_completed, checks_failed,
		       avg_check_duration_ms, metadata, created_at, updated_at
		FROM scheduler_workers
		WHERE status = 'OFFLINE' AND updated_at < $1
	`
	rows, err := r.db.QueryContext(ctx, query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to list offline workers for cleanup: %w", err)
	}
	defer closeResource(rows, &err, "failed to close offline worker rows")

	workers := []*model.Worker{}
	for rows.Next() {
		w := &model.Worker{}
		var metadataStr string
		if err := rows.Scan(
			&w.ID, &w.Name, &w.Zone, &w.Status,
			&w.LastHeartbeat, &w.ChecksCompleted, &w.ChecksFailed,
			&w.AvgCheckDurationMs, &metadataStr,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan offline worker: %w", err)
		}
		w.Metadata = jsonToMetadata(metadataStr)
		workers = append(workers, w)
	}
	return workers, nil
}

func isDuplicateKey(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func metadataToJSON(m map[string]string) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func jsonToMetadata(s string) map[string]string {
	m := make(map[string]string)
	if s == "" || s == "{}" {
		return m
	}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return make(map[string]string)
	}
	return m
}
