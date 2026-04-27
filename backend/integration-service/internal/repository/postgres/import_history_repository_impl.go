package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
)

const (
	importHistoryTable = "import_history"
)

// ImportHistoryRepositoryImpl реализует ImportHistoryRepository для PostgreSQL.
type ImportHistoryRepositoryImpl struct {
	db *sqlx.DB
}

// NewImportHistoryRepository создаёт новый ImportHistoryRepositoryImpl.
func NewImportHistoryRepository(db *sqlx.DB) interfaces.ImportHistoryRepository {
	return &ImportHistoryRepositoryImpl{db: db}
}

// Create создаёт новую запись истории импорта.
func (r *ImportHistoryRepositoryImpl) Create(ctx context.Context, history *model.ImportHistory) error {
	// Подготавливаем JSONB данные
	validationErrorsJSON, err := history.GetValidationErrorsJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal validation errors")
	}

	importedMonitorIDsJSON, err := history.GetImportedMonitorIDsJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal imported monitor IDs")
	}

	conflictResolutionJSON, err := history.GetConflictResolutionJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal conflict resolution")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, user_id, source, status,
			total_monitors, successful_imports, failed_imports, skipped_imports,
			overwrite_existing,
			imported_monitor_ids, validation_errors, conflict_resolution,
			file_name, file_size_bytes,
			started_at, completed_at, duration_ms, error_message
		) VALUES (
			:id, :user_id, :source, :status,
			:total_monitors, :successful_imports, :failed_imports, :skipped_imports,
			:overwrite_existing,
			:imported_monitor_ids, :validation_errors, :conflict_resolution,
			:file_name, :file_size_bytes,
			:started_at, :completed_at, :duration_ms, :error_message
		)
	`, importHistoryTable)

	args := map[string]any{
		"id":                   history.ID,
		"user_id":              history.UserID,
		"source":               history.Source,
		"status":               history.Status,
		"total_monitors":       history.TotalMonitors,
		"successful_imports":   history.SuccessfulImports,
		"failed_imports":       history.FailedImports,
		"skipped_imports":      history.SkippedImports,
		"overwrite_existing":   history.OverwriteExisting,
		"imported_monitor_ids": importedMonitorIDsJSON,
		"validation_errors":    validationErrorsJSON,
		"conflict_resolution":  conflictResolutionJSON,
		"file_name":            history.FileName,
		"file_size_bytes":      history.FileSizeBytes,
		"started_at":           history.StartedAt,
		"completed_at":         history.CompletedAt,
		"duration_ms":          history.DurationMs,
		"error_message":        history.ErrorMessage,
	}

	_, err = sqlx.NamedExecContext(ctx, r.db, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to create import history")
	}

	return nil
}

// GetByID получает историю импорта по ID.
func (r *ImportHistoryRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.ImportHistory, error) {
	query := fmt.Sprintf(`
		SELECT
			id, user_id, source, status,
			total_monitors, successful_imports, failed_imports, skipped_imports,
			overwrite_existing,
			imported_monitor_ids, validation_errors, conflict_resolution,
			file_name, file_size_bytes,
			started_at, completed_at, duration_ms, error_message
		FROM %s
		WHERE id = $1
	`, importHistoryTable)

	var dbHistory dbImportHistory
	if err := r.db.GetContext(ctx, &dbHistory, query, id); err != nil {
		return nil, errors.Wrap(err, "failed to get import history by ID")
	}

	return r.toDomain(dbHistory)
}

// ListByUserID получает список истории импорта для пользователя.
func (r *ImportHistoryRepositoryImpl) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.ImportHistory, error) {
	query := fmt.Sprintf(`
		SELECT
			id, user_id, source, status,
			total_monitors, successful_imports, failed_imports, skipped_imports,
			overwrite_existing,
			imported_monitor_ids, validation_errors, conflict_resolution,
			file_name, file_size_bytes,
			started_at, completed_at, duration_ms, error_message
		FROM %s
		WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`, importHistoryTable)

	var dbHistories []dbImportHistory
	if err := r.db.SelectContext(ctx, &dbHistories, query, userID, limit, offset); err != nil {
		return nil, errors.Wrap(err, "failed to list import history")
	}

	histories := make([]*model.ImportHistory, len(dbHistories))
	for idx, dbHistory := range dbHistories {
		history, err := r.toDomain(dbHistory)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert import history at index %d", idx)
		}
		histories[idx] = history
	}

	return histories, nil
}

// Update обновляет историю импорта.
func (r *ImportHistoryRepositoryImpl) Update(ctx context.Context, history *model.ImportHistory) error {
	// Подготавливаем JSONB данные
	validationErrorsJSON, err := history.GetValidationErrorsJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal validation errors")
	}

	importedMonitorIDsJSON, err := history.GetImportedMonitorIDsJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal imported monitor IDs")
	}

	conflictResolutionJSON, err := history.GetConflictResolutionJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal conflict resolution")
	}

	query := fmt.Sprintf(`
		UPDATE %s SET
			status = :status,
			total_monitors = :total_monitors,
			successful_imports = :successful_imports,
			failed_imports = :failed_imports,
			skipped_imports = :skipped_imports,
			imported_monitor_ids = :imported_monitor_ids,
			validation_errors = :validation_errors,
			conflict_resolution = :conflict_resolution,
			completed_at = :completed_at,
			duration_ms = :duration_ms,
			error_message = :error_message
		WHERE id = :id
	`, importHistoryTable)

	args := map[string]any{
		"id":                   history.ID,
		"status":               history.Status,
		"total_monitors":       history.TotalMonitors,
		"successful_imports":   history.SuccessfulImports,
		"failed_imports":       history.FailedImports,
		"skipped_imports":      history.SkippedImports,
		"imported_monitor_ids": importedMonitorIDsJSON,
		"validation_errors":    validationErrorsJSON,
		"conflict_resolution":  conflictResolutionJSON,
		"completed_at":         history.CompletedAt,
		"duration_ms":          history.DurationMs,
		"error_message":        history.ErrorMessage,
	}

	_, err = sqlx.NamedExecContext(ctx, r.db, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to update import history")
	}

	return nil
}

// UpdateStatus обновляет статус импорта.
func (r *ImportHistoryRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ImportStatus) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET status = $1
		WHERE id = $2
	`, importHistoryTable)

	_, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return errors.Wrap(err, "failed to update import status")
	}

	return nil
}

// Delete удаляет историю импорта.
func (r *ImportHistoryRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, importHistoryTable)

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete import history")
	}

	return nil
}

// toDomain конвертирует dbImportHistory в model.ImportHistory.
func (r *ImportHistoryRepositoryImpl) toDomain(dbHistory dbImportHistory) (*model.ImportHistory, error) {
	validationErrors, err := model.ParseValidationErrorsFromJSON(dbHistory.ValidationErrors)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse validation errors")
	}

	importedMonitorIDs, err := model.ParseImportedMonitorIDsFromJSON(dbHistory.ImportedMonitorIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse imported monitor IDs")
	}

	conflictResolution, err := model.ParseConflictResolutionFromJSON(dbHistory.ConflictResolution)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse conflict resolution")
	}

	// Парсим timestamps
	startedAt, err := time.Parse(time.RFC3339, dbHistory.StartedAt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse started_at")
	}

	var completedAt *time.Time
	if dbHistory.CompletedAt != nil {
		parsed, err := time.Parse(time.RFC3339, *dbHistory.CompletedAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse completed_at")
		}
		completedAt = &parsed
	}

	return &model.ImportHistory{
		ID:                 dbHistory.ID,
		UserID:             dbHistory.UserID,
		Source:             model.ImportSource(dbHistory.Source),
		Status:             model.ImportStatus(dbHistory.Status),
		TotalMonitors:      dbHistory.TotalMonitors,
		SuccessfulImports:  dbHistory.SuccessfulImports,
		FailedImports:      dbHistory.FailedImports,
		SkippedImports:     dbHistory.SkippedImports,
		OverwriteExisting:  dbHistory.OverwriteExisting,
		ImportedMonitorIDs: importedMonitorIDs,
		ValidationErrors:   validationErrors,
		ConflictResolution: conflictResolution,
		FileName:           dbHistory.FileName,
		FileSizeBytes:      dbHistory.FileSizeBytes,
		StartedAt:          startedAt,
		CompletedAt:        completedAt,
		DurationMs:         dbHistory.DurationMs,
		ErrorMessage:       dbHistory.ErrorMessage,
	}, nil
}

// dbImportHistory представляет структуру в БД.
type dbImportHistory struct {
	ID                 uuid.UUID `db:"id"`
	UserID             uuid.UUID `db:"user_id"`
	Source             string    `db:"source"`
	Status             string    `db:"status"`
	TotalMonitors      int       `db:"total_monitors"`
	SuccessfulImports  int       `db:"successful_imports"`
	FailedImports      int       `db:"failed_imports"`
	SkippedImports     int       `db:"skipped_imports"`
	OverwriteExisting  bool      `db:"overwrite_existing"`
	ImportedMonitorIDs []byte    `db:"imported_monitor_ids"`
	ValidationErrors   []byte    `db:"validation_errors"`
	ConflictResolution []byte    `db:"conflict_resolution"`
	FileName           *string   `db:"file_name"`
	FileSizeBytes      *int      `db:"file_size_bytes"`
	StartedAt          string    `db:"started_at"`
	CompletedAt        *string   `db:"completed_at"`
	DurationMs         *int      `db:"duration_ms"`
	ErrorMessage       *string   `db:"error_message"`
}
