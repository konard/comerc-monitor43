package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

var (
	ErrTemplateNotFound      = errors.New("template not found")
	ErrTemplateAlreadyExists = errors.New("template already exists")
)

// PostgresTemplateRepository реализует TemplateRepository для PostgreSQL.
type PostgresTemplateRepository struct {
	db *sqlx.DB
}

// NewPostgresTemplateRepository создаёт новый репозиторий шаблонов.
func NewPostgresTemplateRepository(db *sqlx.DB) *PostgresTemplateRepository {
	return &PostgresTemplateRepository{db: db}
}

// Create создаёт новый шаблон.
func (r *PostgresTemplateRepository) Create(ctx context.Context, tmpl *model.Template) (err error) {
	tmpl.BeforeCreate()
	query := `
		INSERT INTO templates (
			user_id, name, description, channel, type, engine,
			subject, body, format, is_default, is_system, version, parent_id
		) VALUES (
			:user_id, :name, :description, :channel, :type, :engine,
			:subject, :body, :format, :is_default, :is_system, :version, :parent_id
		) RETURNING id, created_at, updated_at
	`

	rows, err := r.db.NamedQueryContext(ctx, query, tmpl)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}
	defer closeResource(rows, &err, "failed to close template rows")

	if rows.Next() {
		if err := rows.Scan(&tmpl.ID, &tmpl.CreatedAt, &tmpl.UpdatedAt); err != nil {
			return fmt.Errorf("failed to scan template id: %w", err)
		}
	}

	return nil
}

// GetByID получает шаблон по ID.
func (r *PostgresTemplateRepository) GetByID(ctx context.Context, id string) (*model.Template, error) {
	query := `
		SELECT id, user_id, name, COALESCE(description,'') as description, channel, type, engine,
		       COALESCE(subject,'') as subject, body, format, is_default, is_system, version, parent_id,
		       created_at, updated_at
		FROM templates
		WHERE id = $1
	`

	var tmpl model.Template
	if err := r.db.GetContext(ctx, &tmpl, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &tmpl, nil
}

// GetDefault получает дефолтный шаблон для пользователя, канала и типа.
func (r *PostgresTemplateRepository) GetDefault(
	ctx context.Context,
	userID string,
	channel model.TemplateChannel,
	tmplType model.TemplateType,
) (*model.Template, error) {
	query := `
		SELECT id, user_id, name, COALESCE(description,'') as description, channel, type, engine,
		       COALESCE(subject,'') as subject, body, format, is_default, is_system, version, parent_id,
		       created_at, updated_at
		FROM templates
		WHERE user_id = $1 AND channel = $2 AND type = $3 AND is_default = true
		LIMIT 1
	`

	var tmpl model.Template
	if err := r.db.GetContext(ctx, &tmpl, query, userID, channel, tmplType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get default template: %w", err)
	}

	return &tmpl, nil
}

// List получает список шаблонов с пагинацией и фильтрами.
func (r *PostgresTemplateRepository) List(ctx context.Context, filter service.ListFilter) ([]*model.Template, int, error) {
	baseQuery := `SELECT id, user_id, name, COALESCE(description,'') as description, channel, type, engine, COALESCE(subject,'') as subject, body, format, is_default, is_system, version, parent_id, created_at, updated_at FROM templates WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM templates WHERE 1=1`
	args := make(map[string]any)

	if filter.UserID != "" {
		baseQuery += " AND user_id = :user_id"
		countQuery += " AND user_id = :user_id"
		args["user_id"] = filter.UserID
	}

	if filter.Channel != "" {
		baseQuery += " AND channel = :channel"
		countQuery += " AND channel = :channel"
		args["channel"] = filter.Channel
	}

	if filter.Type != "" {
		baseQuery += " AND type = :type"
		countQuery += " AND type = :type"
		args["type"] = filter.Type
	}

	if filter.IsDefault != nil {
		baseQuery += " AND is_default = :is_default"
		countQuery += " AND is_default = :is_default"
		args["is_default"] = *filter.IsDefault
	}

	if filter.IsSystem != nil {
		baseQuery += " AND is_system = :is_system"
		countQuery += " AND is_system = :is_system"
		args["is_system"] = *filter.IsSystem
	}

	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	sortOrder := "ASC"
	if filter.SortDesc {
		sortOrder = "DESC"
	}
	baseQuery += fmt.Sprintf(" ORDER BY %s %s", filter.SortBy, sortOrder)

	if filter.PageSize > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", filter.PageSize)
		if filter.Page > 0 {
			offset := (filter.Page - 1) * filter.PageSize
			baseQuery += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	countQ, countArgs, err := r.db.BindNamed(countQuery, args)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to bind count query: %w", err)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQ, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	listQ, listArgs, err := r.db.BindNamed(baseQuery, args)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to bind query: %w", err)
	}

	var templates []*model.Template
	if err := r.db.SelectContext(ctx, &templates, listQ, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, total, nil
}

// Update обновляет шаблон.
func (r *PostgresTemplateRepository) Update(ctx context.Context, tmpl *model.Template) error {
	query := `
		UPDATE templates SET
			name = :name,
			description = :description,
			subject = :subject,
			body = :body,
			format = :format,
			version = :version,
			updated_at = NOW()
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, tmpl)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// Delete удаляет шаблон.
func (r *PostgresTemplateRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM templates WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// SetDefault устанавливает шаблон как дефолтный.
func (r *PostgresTemplateRepository) SetDefault(ctx context.Context, id, userID string) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer rollbackOnError(tx, &err, "failed to rollback default template transaction")

	// Сначала получаем шаблон, чтобы узнать канал и тип
	var tmpl model.Template
	query := `SELECT channel, type FROM templates WHERE id = $1 AND user_id = $2`
	if err := tx.Get(&tmpl, query, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTemplateNotFound
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Сбрасываем предыдущий дефолтный шаблон
	resetQuery := `
		UPDATE templates
		SET is_default = false
		WHERE user_id = $1 AND channel = $2 AND type = $3 AND is_default = true
	`
	if _, err := tx.Exec(resetQuery, userID, tmpl.Channel, tmpl.Type); err != nil {
		return fmt.Errorf("failed to reset default template: %w", err)
	}

	// Устанавливаем новый дефолтный шаблон
	setQuery := `
		UPDATE templates
		SET is_default = true
		WHERE id = $1 AND user_id = $2
	`
	if _, err := tx.Exec(setQuery, id, userID); err != nil {
		return fmt.Errorf("failed to set default template: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Exists проверяет существование шаблона по ID.
func (r *PostgresTemplateRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM templates WHERE id = $1)`

	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, id); err != nil {
		return false, fmt.Errorf("failed to check template existence: %w", err)
	}

	return exists, nil
}
