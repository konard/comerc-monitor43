package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
)

type IncidentRepository struct {
	db *DB
}

func NewIncidentRepository(db *DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

func (r *IncidentRepository) Upsert(ctx context.Context, incident *model.Incident) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "IncidentRepository.Upsert")
		defer span.End()
		span.SetAttributes(
			attribute.String("incident_id", incident.ID.String()),
			attribute.String("monitor_id", incident.MonitorID.String()),
			attribute.String("status", string(incident.Status)),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO incidents (id, monitor_id, started_at, ended_at, duration_seconds, check_count, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			ended_at = EXCLUDED.ended_at,
			duration_seconds = EXCLUDED.duration_seconds,
			check_count = EXCLUDED.check_count,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		incident.ID, incident.MonitorID, incident.StartedAt, incident.EndedAt,
		incident.DurationSeconds, incident.CheckCount, incident.Status,
		incident.CreatedAt, incident.UpdatedAt,
	)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return errors.Wrap(err, "failed to upsert incident")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "upsert", "incidents", float64(duration))
	}

	return nil
}

func (r *IncidentRepository) ListByMonitorID(ctx context.Context, monitorID string, filter model.IncidentFilter) ([]*model.Incident, int, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "IncidentRepository.ListByMonitorID")
		defer span.End()
		span.SetAttributes(
			attribute.String("monitor_id", monitorID),
			attribute.Int("page", filter.Page),
			attribute.Int("page_size", filter.PageSize),
		)
	}

	startTime := time.Now()

	args := []any{monitorID}
	argNum := 2

	var whereParts []string

	if filter.StartDate != nil {
		whereParts = append(whereParts, fmt.Sprintf("started_at >= $%d", argNum))
		args = append(args, *filter.StartDate)
		argNum++
	}

	if filter.EndDate != nil {
		whereParts = append(whereParts, fmt.Sprintf("started_at <= $%d", argNum))
		args = append(args, *filter.EndDate)
		argNum++
	}

	whereClause := "WHERE monitor_id = $1"
	if len(whereParts) > 0 {
		whereClause += " AND " + strings.Join(whereParts, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM incidents " + whereClause
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, errors.Wrap(err, "failed to count incidents")
	}

	dataQuery := `
		SELECT id, monitor_id, started_at, ended_at, duration_seconds, check_count, status, created_at, updated_at
		FROM incidents
	` + whereClause + " ORDER BY started_at DESC"

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	page := filter.Page
	if page <= 0 {
		page = model.DefaultPage
	}

	dataQuery += fmt.Sprintf(" LIMIT $%d", argNum)
	args = append(args, pageSize)
	argNum++

	offset := (page - 1) * pageSize
	dataQuery += fmt.Sprintf(" OFFSET $%d", argNum)
	args = append(args, offset)

	var incidents []*model.Incident
	err = r.db.SelectContext(ctx, &incidents, dataQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, errors.Wrap(err, "failed to list incidents")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "list", "incidents", float64(duration))
	}

	return incidents, total, nil
}

func (r *IncidentRepository) GetByID(ctx context.Context, id string) (*model.Incident, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "IncidentRepository.GetByID")
		defer span.End()
		span.SetAttributes(attribute.String("incident_id", id))
	}

	startTime := time.Now()

	query := `
		SELECT id, monitor_id, started_at, ended_at, duration_seconds, check_count, status, created_at, updated_at
		FROM incidents
		WHERE id = $1
	`

	var incident model.Incident
	err := r.db.GetContext(ctx, &incident, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, errors.Wrap(err, "failed to get incident by id")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getbyid", "incidents", float64(duration))
	}

	return &incident, nil
}

func (r *IncidentRepository) Update(ctx context.Context, incident *model.Incident) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "IncidentRepository.Update")
		defer span.End()
		span.SetAttributes(
			attribute.String("incident_id", incident.ID.String()),
			attribute.String("monitor_id", incident.MonitorID.String()),
			attribute.String("status", string(incident.Status)),
		)
	}

	startTime := time.Now()

	query := `
		UPDATE incidents SET
			ended_at = $1,
			duration_seconds = $2,
			check_count = $3,
			status = $4,
			updated_at = $5
		WHERE id = $6
	`

	result, err := r.db.ExecContext(ctx, query,
		incident.EndedAt, incident.DurationSeconds, incident.CheckCount,
		incident.Status, incident.UpdatedAt, incident.ID,
	)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return errors.Wrap(err, "failed to update incident")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return model.ErrIncidentNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "update", "incidents", float64(duration))
	}

	return nil
}
