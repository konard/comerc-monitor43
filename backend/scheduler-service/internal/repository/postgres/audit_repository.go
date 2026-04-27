package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, action string, workerID, checkID, monitorID uuid.UUID, details map[string]any) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO scheduler_audit_log (id, action, worker_id, check_id, monitor_id, details, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New(), action, nullableUUID(workerID), nullableUUID(checkID),
		nullableUUID(monitorID), detailsJSON, time.Now(),
	)
	return err
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
