package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

type incidentRepository struct {
	db *sqlx.DB
}

// NewIncidentRepository создаёт новый репозиторий инцидентов.
func NewIncidentRepository(db *sql.DB) interfaces.IncidentRepository {
	return &incidentRepository{
		db: sqlx.NewDb(db, "postgres"),
	}
}

func (r *incidentRepository) Create(ctx context.Context, incident *domain.Incident) error {
	query := `
		INSERT INTO incidents (
			id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		) VALUES (
			:id, :monitor_id, :start_time, :end_time, :duration_seconds, :status, :created_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, r.incidentToDB(incident))
	if err != nil {
		return errors.Wrap(err, "failed to create incident")
	}

	return nil
}

func (r *incidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	query := `
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE id = $1
	`

	var dbIncident dbIncident
	err := r.db.GetContext(ctx, &dbIncident, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(interfaces.ErrIncidentNotFound, "incident not found")
		}
		return nil, errors.Wrap(err, "failed to get incident by id")
	}

	return r.dbToIncident(&dbIncident)
}

func (r *incidentRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*domain.Incident, error) {
	query := `
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1
		ORDER BY start_time DESC
		LIMIT $2 OFFSET $3
	`

	var dbIncidents []dbIncident
	err := r.db.SelectContext(ctx, &dbIncidents, query, monitorID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get incidents by monitor_id")
	}

	return r.dbListToIncidents(dbIncidents)
}

func (r *incidentRepository) GetByMonitorIDAndPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Incident, error) {
	query := `
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1 AND start_time >= $2 AND start_time <= $3
		ORDER BY start_time DESC
	`

	var dbIncidents []dbIncident
	err := r.db.SelectContext(ctx, &dbIncidents, query, monitorID, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get incidents by monitor_id and period")
	}

	return r.dbListToIncidents(dbIncidents)
}

func (r *incidentRepository) GetActiveByMonitorID(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error) {
	query := `
		SELECT id, monitor_id, start_time, end_time, duration_seconds, status, created_at
		FROM incidents
		WHERE monitor_id = $1 AND status = 'ACTIVE'
		ORDER BY start_time DESC
	`

	var dbIncidents []dbIncident
	err := r.db.SelectContext(ctx, &dbIncidents, query, monitorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active incidents")
	}

	return r.dbListToIncidents(dbIncidents)
}

func (r *incidentRepository) Update(ctx context.Context, incident *domain.Incident) error {
	query := `
		UPDATE incidents SET
			end_time = :end_time,
			duration_seconds = :duration_seconds,
			status = :status
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, r.incidentToDB(incident))
	if err != nil {
		return errors.Wrap(err, "failed to update incident")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(interfaces.ErrIncidentNotFound, "incident not found")
	}

	return nil
}

func (r *incidentRepository) Resolve(ctx context.Context, id uuid.UUID, endTime time.Time) error {
	query := `
		UPDATE incidents SET
			end_time = $1,
			duration_seconds = EXTRACT(EPOCH FROM ($1 - start_time))::INTEGER,
			status = 'RESOLVED'
		WHERE id = $2 AND status = 'ACTIVE'
	`

	result, err := r.db.ExecContext(ctx, query, endTime, id)
	if err != nil {
		return errors.Wrap(err, "failed to resolve incident")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(interfaces.ErrIncidentNotFound, "incident not found or already resolved")
	}

	return nil
}

func (r *incidentRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM incidents WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete old incidents")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get rows affected")
	}

	return rows, nil
}

func (r *incidentRepository) CountByMonitorID(ctx context.Context, monitorID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM incidents WHERE monitor_id = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, monitorID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count incidents")
	}

	return count, nil
}

// dbIncident представляет структуру инцидента в БД.
type dbIncident struct {
	ID              uuid.UUID     `db:"id"`
	MonitorID       uuid.UUID     `db:"monitor_id"`
	StartTime       time.Time     `db:"start_time"`
	EndTime         sql.NullTime  `db:"end_time"`
	DurationSeconds sql.NullInt64 `db:"duration_seconds"`
	Status          string        `db:"status"`
	CreatedAt       time.Time     `db:"created_at"`
}

// incidentToDB конвертирует domain.Incident в dbIncident.
func (r *incidentRepository) incidentToDB(i *domain.Incident) dbIncident {
	db := dbIncident{
		ID:        i.ID,
		MonitorID: i.MonitorID,
		StartTime: i.StartTime,
		Status:    string(i.Status),
		CreatedAt: i.CreatedAt,
	}

	if i.EndTime != nil {
		db.EndTime = sql.NullTime{Time: *i.EndTime, Valid: true}
	}
	if i.DurationSeconds != nil {
		db.DurationSeconds = sql.NullInt64{Int64: int64(*i.DurationSeconds), Valid: true}
	}

	return db
}

// dbToIncident конвертирует dbIncident в domain.Incident.
func (r *incidentRepository) dbToIncident(db *dbIncident) (*domain.Incident, error) {
	status := domain.IncidentStatus(db.Status)
	if status != domain.IncidentActive && status != domain.IncidentResolved {
		return nil, errors.Wrapf(errors.New("invalid status"), "invalid incident status: %s", db.Status)
	}

	i := &domain.Incident{
		ID:        db.ID,
		MonitorID: db.MonitorID,
		StartTime: db.StartTime,
		Status:    status,
		CreatedAt: db.CreatedAt,
	}

	if db.EndTime.Valid {
		i.EndTime = &db.EndTime.Time
	}
	if db.DurationSeconds.Valid {
		v := int(db.DurationSeconds.Int64)
		i.DurationSeconds = &v
	}

	return i, nil
}

// dbListToIncidents конвертирует []dbIncident в []*domain.Incident.
func (r *incidentRepository) dbListToIncidents(dbIncidents []dbIncident) ([]*domain.Incident, error) {
	incidents := make([]*domain.Incident, len(dbIncidents))
	for i, db := range dbIncidents {
		incident, err := r.dbToIncident(&db)
		if err != nil {
			return nil, err
		}
		incidents[i] = incident
	}
	return incidents, nil
}
