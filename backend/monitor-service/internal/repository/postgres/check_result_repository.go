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

type checkResultRepository struct {
	db *sqlx.DB
}

// NewCheckResultRepository создаёт новый репозиторий результатов проверок.
func NewCheckResultRepository(db *sql.DB) interfaces.CheckResultRepository {
	return &checkResultRepository{
		db: sqlx.NewDb(db, "postgres"),
	}
}

func (r *checkResultRepository) Create(ctx context.Context, result *domain.CheckResult) error {
	query := `
		INSERT INTO check_results (
			id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		) VALUES (
			:id, :monitor_id, :status, :response_time_ms, :status_code,
			:error_message, :error_code, :checked_at, :created_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, r.checkResultToDB(result))
	if err != nil {
		return errors.Wrap(err, "failed to create check result")
	}

	return nil
}

func (r *checkResultRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CheckResult, error) {
	query := `
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE id = $1
	`

	var dbResult dbCheckResult
	err := r.db.GetContext(ctx, &dbResult, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(interfaces.ErrCheckResultNotFound, "check result not found")
		}
		return nil, errors.Wrap(err, "failed to get check result by id")
	}

	return r.dbToCheckResult(&dbResult)
}

func (r *checkResultRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*domain.CheckResult, error) {
	query := `
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2 OFFSET $3
	`

	var dbResults []dbCheckResult
	err := r.db.SelectContext(ctx, &dbResults, query, monitorID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get check results by monitor_id")
	}

	return r.dbListToCheckResults(dbResults)
}

func (r *checkResultRepository) GetByMonitorIDAndPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.CheckResult, error) {
	query := `
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
	`

	var dbResults []dbCheckResult
	err := r.db.SelectContext(ctx, &dbResults, query, monitorID, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get check results by monitor_id and period")
	}

	return r.dbListToCheckResults(dbResults)
}

func (r *checkResultRepository) GetByMonitorIDAndPeriodPaginated(ctx context.Context, monitorID uuid.UUID, from, to time.Time, limit, offset int) ([]*domain.CheckResult, error) {
	query := `
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
		LIMIT $4 OFFSET $5
	`

	var dbResults []dbCheckResult
	err := r.db.SelectContext(ctx, &dbResults, query, monitorID, from, to, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get check results by monitor_id and period paginated")
	}

	return r.dbListToCheckResults(dbResults)
}

func (r *checkResultRepository) GetByMonitorIDAndPeriodAndStatus(ctx context.Context, monitorID uuid.UUID, from, to time.Time, status domain.MonitorStatus, limit, offset int) ([]*domain.CheckResult, error) {
	query := `
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3 AND status = $4
		ORDER BY checked_at DESC
		LIMIT $5 OFFSET $6
	`

	var dbResults []dbCheckResult
	err := r.db.SelectContext(ctx, &dbResults, query, monitorID, from, to, string(status), limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get check results by monitor_id, period and status")
	}

	return r.dbListToCheckResults(dbResults)
}

func (r *checkResultRepository) GetLatestByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]*domain.CheckResult, error) {
	query := `
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`

	var dbResults []dbCheckResult
	err := r.db.SelectContext(ctx, &dbResults, query, monitorID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest check results")
	}

	return r.dbListToCheckResults(dbResults)
}

func (r *checkResultRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM check_results WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete old check results")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get rows affected")
	}

	return rows, nil
}

func (r *checkResultRepository) CountByMonitorID(ctx context.Context, monitorID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM check_results WHERE monitor_id = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, monitorID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count check results")
	}

	return count, nil
}

// dbCheckResult представляет структуру результата проверки в БД.
type dbCheckResult struct {
	ID             uuid.UUID      `db:"id"`
	MonitorID      uuid.UUID      `db:"monitor_id"`
	Status         string         `db:"status"`
	ResponseTimeMs sql.NullInt64  `db:"response_time_ms"`
	StatusCode     sql.NullInt64  `db:"status_code"`
	ErrorMessage   sql.NullString `db:"error_message"`
	ErrorCode      sql.NullString `db:"error_code"`
	CheckedAt      time.Time      `db:"checked_at"`
	CreatedAt      time.Time      `db:"created_at"`
}

// checkResultToDB конвертирует domain.CheckResult в dbCheckResult.
func (r *checkResultRepository) checkResultToDB(cr *domain.CheckResult) dbCheckResult {
	db := dbCheckResult{
		ID:        cr.ID,
		MonitorID: cr.MonitorID,
		Status:    string(cr.Status),
		CheckedAt: cr.CheckedAt,
		CreatedAt: cr.CreatedAt,
	}

	if cr.ResponseTimeMs != nil {
		db.ResponseTimeMs = sql.NullInt64{Int64: int64(*cr.ResponseTimeMs), Valid: true}
	}
	if cr.StatusCode != nil {
		db.StatusCode = sql.NullInt64{Int64: int64(*cr.StatusCode), Valid: true}
	}
	if cr.ErrorMessage != nil {
		db.ErrorMessage = sql.NullString{String: *cr.ErrorMessage, Valid: true}
	}
	if cr.ErrorCode != nil {
		db.ErrorCode = sql.NullString{String: string(*cr.ErrorCode), Valid: true}
	}

	return db
}

// dbToCheckResult конвертирует dbCheckResult в domain.CheckResult.
func (r *checkResultRepository) dbToCheckResult(db *dbCheckResult) (*domain.CheckResult, error) {
	status := domain.MonitorStatus(db.Status)
	if !status.IsValid() {
		return nil, errors.Wrapf(errors.New("invalid status"), "invalid check result status: %s", db.Status)
	}

	cr := &domain.CheckResult{
		ID:        db.ID,
		MonitorID: db.MonitorID,
		Status:    status,
		CheckedAt: db.CheckedAt,
		CreatedAt: db.CreatedAt,
	}

	if db.ResponseTimeMs.Valid {
		v := int(db.ResponseTimeMs.Int64)
		cr.ResponseTimeMs = &v
	}
	if db.StatusCode.Valid {
		v := int(db.StatusCode.Int64)
		cr.StatusCode = &v
	}
	if db.ErrorMessage.Valid {
		cr.ErrorMessage = &db.ErrorMessage.String
	}
	if db.ErrorCode.Valid {
		code := domain.CheckErrorCode(db.ErrorCode.String)
		cr.ErrorCode = &code
	}

	return cr, nil
}

// dbListToCheckResults конвертирует []dbCheckResult в []*domain.CheckResult.
func (r *checkResultRepository) dbListToCheckResults(dbResults []dbCheckResult) ([]*domain.CheckResult, error) {
	results := make([]*domain.CheckResult, len(dbResults))
	for i, db := range dbResults {
		cr, err := r.dbToCheckResult(&db)
		if err != nil {
			return nil, err
		}
		results[i] = cr
	}
	return results, nil
}
