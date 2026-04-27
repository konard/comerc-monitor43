package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

type auditRepository struct {
	db *sqlx.DB
}

// NewAuditRepository создаёт новый репозиторий audit log.
func NewAuditRepository(db *sql.DB) interfaces.AuditRepository {
	return &auditRepository{
		db: sqlx.NewDb(db, "postgres"),
	}
}

func (r *auditRepository) Create(ctx context.Context, entry *interfaces.AuditLogEntry) error {
	oldValuesJSON, err := json.Marshal(entry.OldValues)
	if err != nil {
		return errors.Wrap(err, "failed to marshal old_values")
	}

	newValuesJSON, err := json.Marshal(entry.NewValues)
	if err != nil {
		return errors.Wrap(err, "failed to marshal new_values")
	}

	query := `
		INSERT INTO monitor_audit_log (
			id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.ExecContext(ctx, query,
		entry.ID, entry.MonitorID, entry.UserID, entry.Action,
		oldValuesJSON, newValuesJSON, entry.IPAddress, entry.UserAgent, entry.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create audit log entry")
	}

	return nil
}

func (r *auditRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) (_ []*interfaces.AuditLogEntry, err error) {
	query := `
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, monitorID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit log by monitor_id")
	}
	defer closeResource(rows, &err, "failed to close audit log rows")

	return r.scanAuditLogRows(rows)
}

func (r *auditRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) (_ []*interfaces.AuditLogEntry, err error) {
	query := `
		SELECT id, monitor_id, user_id, action, old_values, new_values, ip_address, user_agent, created_at
		FROM monitor_audit_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit log by user_id")
	}
	defer closeResource(rows, &err, "failed to close audit log rows")

	return r.scanAuditLogRows(rows)
}

func (r *auditRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM monitor_audit_log WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete old audit log entries")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get rows affected")
	}

	return rows, nil
}

// scanAuditLogRows сканирует строки в []*interfaces.AuditLogEntry.
func (r *auditRepository) scanAuditLogRows(rows *sql.Rows) ([]*interfaces.AuditLogEntry, error) {
	var entries []*interfaces.AuditLogEntry

	for rows.Next() {
		entry := &interfaces.AuditLogEntry{
			OldValues: make(map[string]any),
			NewValues: make(map[string]any),
		}

		var oldValuesJSON, newValuesJSON []byte
		err := rows.Scan(
			&entry.ID, &entry.MonitorID, &entry.UserID, &entry.Action,
			&oldValuesJSON, &newValuesJSON,
			&entry.IPAddress, &entry.UserAgent, &entry.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan audit log row")
		}

		if len(oldValuesJSON) > 0 {
			if err := json.Unmarshal(oldValuesJSON, &entry.OldValues); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal old_values")
			}
		}

		if len(newValuesJSON) > 0 {
			if err := json.Unmarshal(newValuesJSON, &entry.NewValues); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal new_values")
			}
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating audit log rows")
	}

	return entries, nil
}
