package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

type monitorRepository struct {
	db *sqlx.DB
}

// NewMonitorRepository создаёт новый репозиторий мониторов.
func NewMonitorRepository(db *sql.DB) interfaces.MonitorRepository {
	return &monitorRepository{
		db: sqlx.NewDb(db, "postgres"),
	}
}

func (r *monitorRepository) Create(ctx context.Context, monitor *domain.Monitor) error {
	query := `
		INSERT INTO monitors (
			id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		) VALUES (
			:id, :user_id, :name, :url, :check_type, :interval_seconds, :timeout_seconds,
			:status, :working_hours_start, :working_hours_end, :working_days,
			:degraded_response_time_threshold, :degraded_failure_rate_threshold,
			:last_check_at, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, r.monitorToDB(monitor))
	if err != nil {
		return errors.Wrap(err, "failed to create monitor")
	}

	return nil
}

func (r *monitorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE id = $1
	`

	var dbMonitor dbMonitor
	err := r.db.GetContext(ctx, &dbMonitor, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(interfaces.ErrMonitorNotFound, "monitor not found")
		}
		return nil, errors.Wrap(err, "failed to get monitor by id")
	}

	return r.dbToMonitor(&dbMonitor)
}

func (r *monitorRepository) GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1 AND name = $2
	`

	var dbMonitor dbMonitor
	err := r.db.GetContext(ctx, &dbMonitor, query, userID, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(interfaces.ErrMonitorNotFound, "monitor not found")
		}
		return nil, errors.Wrap(err, "failed to get monitor by user_id and name")
	}

	return r.dbToMonitor(&dbMonitor)
}

func (r *monitorRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var dbMonitors []dbMonitor
	err := r.db.SelectContext(ctx, &dbMonitors, query, userID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list monitors by user_id")
	}

	return r.dbListToMonitors(dbMonitors)
}

func (r *monitorRepository) ListByUserIDAndStatus(ctx context.Context, userID uuid.UUID, status domain.MonitorStatus) ([]*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
	`

	var dbMonitors []dbMonitor
	err := r.db.SelectContext(ctx, &dbMonitors, query, userID, status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list monitors by user_id and status")
	}

	return r.dbListToMonitors(dbMonitors)
}

func (r *monitorRepository) ListActive(ctx context.Context) ([]*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE status != 'PAUSED'
		ORDER BY created_at DESC
	`

	var dbMonitors []dbMonitor
	err := r.db.SelectContext(ctx, &dbMonitors, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list active monitors")
	}

	return r.dbListToMonitors(dbMonitors)
}

func (r *monitorRepository) ListDueForCheck(ctx context.Context, limit int) ([]*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE status != 'PAUSED'
			AND (last_check_at IS NULL OR last_check_at + (interval_seconds || ' seconds')::interval < NOW())
		ORDER BY last_check_at ASC NULLS FIRST
		LIMIT $1
	`

	var dbMonitors []dbMonitor
	err := r.db.SelectContext(ctx, &dbMonitors, query, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list monitors due for check")
	}

	return r.dbListToMonitors(dbMonitors)
}

func (r *monitorRepository) Update(ctx context.Context, monitor *domain.Monitor) error {
	query := `
		UPDATE monitors SET
			name = :name,
			url = :url,
			check_type = :check_type,
			interval_seconds = :interval_seconds,
			timeout_seconds = :timeout_seconds,
			status = :status,
			working_hours_start = :working_hours_start,
			working_hours_end = :working_hours_end,
			working_days = :working_days,
			degraded_response_time_threshold = :degraded_response_time_threshold,
			degraded_failure_rate_threshold = :degraded_failure_rate_threshold,
			last_check_at = :last_check_at,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, r.monitorToDB(monitor))
	if err != nil {
		return errors.Wrap(err, "failed to update monitor")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(interfaces.ErrMonitorNotFound, "monitor not found")
	}

	return nil
}

func (r *monitorRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error {
	query := `
		UPDATE monitors SET
			status = $1,
			updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, "failed to update monitor status")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(interfaces.ErrMonitorNotFound, "monitor not found")
	}

	return nil
}

func (r *monitorRepository) UpdateLastCheck(ctx context.Context, id uuid.UUID, lastCheckAt time.Time) error {
	query := `
		UPDATE monitors SET
			last_check_at = $1,
			updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, lastCheckAt, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, "failed to update monitor last_check_at")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(interfaces.ErrMonitorNotFound, "monitor not found")
	}

	return nil
}

func (r *monitorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM monitors WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete monitor")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(interfaces.ErrMonitorNotFound, "monitor not found")
	}

	return nil
}

func (r *monitorRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM monitors WHERE user_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count monitors by user_id")
	}

	return count, nil
}

func (r *monitorRepository) ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM monitors WHERE user_id = $1 AND name = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, userID, name)
	if err != nil {
		return false, errors.Wrap(err, "failed to check monitor existence")
	}

	return exists, nil
}

// dbMonitor представляет структуру монитора в БД.
type dbMonitor struct {
	ID                            uuid.UUID      `db:"id"`
	UserID                        uuid.UUID      `db:"user_id"`
	Name                          string         `db:"name"`
	URL                           string         `db:"url"`
	CheckType                     string         `db:"check_type"`
	IntervalSeconds               int            `db:"interval_seconds"`
	TimeoutSeconds                int            `db:"timeout_seconds"`
	Status                        string         `db:"status"`
	WorkingHoursStart             sql.NullTime   `db:"working_hours_start"`
	WorkingHoursEnd               sql.NullTime   `db:"working_hours_end"`
	WorkingDays                   pq.StringArray `db:"working_days"`
	DegradedResponseTimeThreshold sql.NullInt64  `db:"degraded_response_time_threshold"`
	DegradedFailureRateThreshold  sql.NullInt64  `db:"degraded_failure_rate_threshold"`
	LastCheckAt                   sql.NullTime   `db:"last_check_at"`
	CreatedAt                     time.Time      `db:"created_at"`
	UpdatedAt                     time.Time      `db:"updated_at"`
}

// monitorToDB конвертирует domain.Monitor в dbMonitor.
func (r *monitorRepository) monitorToDB(m *domain.Monitor) dbMonitor {
	db := dbMonitor{
		ID:              m.ID,
		UserID:          m.UserID,
		Name:            m.Name,
		URL:             m.URL,
		CheckType:       m.CheckType,
		IntervalSeconds: m.IntervalSeconds,
		TimeoutSeconds:  m.TimeoutSeconds,
		Status:          string(m.Status),
		WorkingDays:     pq.StringArray(workingDaysToString(m.WorkingDays)),
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	if m.WorkingHoursStart != nil {
		db.WorkingHoursStart = sql.NullTime{Time: *m.WorkingHoursStart, Valid: true}
	}
	if m.WorkingHoursEnd != nil {
		db.WorkingHoursEnd = sql.NullTime{Time: *m.WorkingHoursEnd, Valid: true}
	}
	if m.DegradedResponseTimeThreshold != nil {
		db.DegradedResponseTimeThreshold = sql.NullInt64{Int64: int64(*m.DegradedResponseTimeThreshold), Valid: true}
	}
	if m.DegradedFailureRateThreshold != nil {
		db.DegradedFailureRateThreshold = sql.NullInt64{Int64: int64(*m.DegradedFailureRateThreshold), Valid: true}
	}
	if m.LastCheckAt != nil {
		db.LastCheckAt = sql.NullTime{Time: *m.LastCheckAt, Valid: true}
	}

	return db
}

// dbToMonitor конвертирует dbMonitor в domain.Monitor.
func (r *monitorRepository) dbToMonitor(db *dbMonitor) (*domain.Monitor, error) {
	status := domain.MonitorStatus(db.Status)
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid monitor status: %s", db.Status)
	}

	m := &domain.Monitor{
		ID:              db.ID,
		UserID:          db.UserID,
		Name:            db.Name,
		URL:             db.URL,
		CheckType:       db.CheckType,
		IntervalSeconds: db.IntervalSeconds,
		TimeoutSeconds:  db.TimeoutSeconds,
		Status:          status,
		WorkingDays:     stringToWorkingDays([]string(db.WorkingDays)),
		CreatedAt:       db.CreatedAt,
		UpdatedAt:       db.UpdatedAt,
	}

	if db.WorkingHoursStart.Valid {
		t := db.WorkingHoursStart.Time
		m.WorkingHoursStart = &t
	}
	if db.WorkingHoursEnd.Valid {
		t := db.WorkingHoursEnd.Time
		m.WorkingHoursEnd = &t
	}
	if db.DegradedResponseTimeThreshold.Valid {
		v := int(db.DegradedResponseTimeThreshold.Int64)
		m.DegradedResponseTimeThreshold = &v
	}
	if db.DegradedFailureRateThreshold.Valid {
		v := int(db.DegradedFailureRateThreshold.Int64)
		m.DegradedFailureRateThreshold = &v
	}
	if db.LastCheckAt.Valid {
		t := db.LastCheckAt.Time
		m.LastCheckAt = &t
	}

	return m, nil
}

// dbListToMonitors конвертирует []dbMonitor в []*domain.Monitor.
func (r *monitorRepository) dbListToMonitors(dbMonitors []dbMonitor) ([]*domain.Monitor, error) {
	monitors := make([]*domain.Monitor, len(dbMonitors))
	for i, db := range dbMonitors {
		m, err := r.dbToMonitor(&db)
		if err != nil {
			return nil, err
		}
		monitors[i] = m
	}
	return monitors, nil
}

// workingDaysToString конвертирует []time.Weekday в []string.
func workingDaysToString(days []time.Weekday) []string {
	if len(days) == 0 {
		return nil
	}

	result := make([]string, len(days))
	for i, d := range days {
		result[i] = d.String()
	}
	return result
}

// stringToWorkingDays конвертирует []string в []time.Weekday.
func stringToWorkingDays(days []string) []time.Weekday {
	if len(days) == 0 {
		return nil
	}

	dayMap := map[string]time.Weekday{
		"Sunday":    time.Sunday,
		"Monday":    time.Monday,
		"Tuesday":   time.Tuesday,
		"Wednesday": time.Wednesday,
		"Thursday":  time.Thursday,
		"Friday":    time.Friday,
		"Saturday":  time.Saturday,
	}

	result := make([]time.Weekday, len(days))
	for i, d := range days {
		result[i] = dayMap[d]
	}
	return result
}
