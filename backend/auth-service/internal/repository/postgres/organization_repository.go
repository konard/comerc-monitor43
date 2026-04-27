package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/repository/interfaces"
)

// ErrOrganizationNotFound возвращается, когда организация не найдена.
var ErrOrganizationNotFound = errors.New("organization not found")

// OrganizationRepository implements organization repository using PostgreSQL.
type OrganizationRepository struct {
	db *DB
}

// NewOrganizationRepository creates a new OrganizationRepository.
func NewOrganizationRepository(db *DB) interfaces.OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create creates a new organization.
func (r *OrganizationRepository) Create(ctx context.Context, org *model.Organization) (*model.Organization, error) {
	const query = `
		INSERT INTO organizations (id, name, owner_id, tier, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, owner_id, tier, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		org.ID, org.Name, org.OwnerID, org.Tier, org.CreatedAt, org.UpdatedAt,
	).Scan(
		&org.ID, &org.Name, &org.OwnerID, &org.Tier, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return org, nil
}

// GetByID retrieves an organization by ID.
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	const query = `
		SELECT id, name, owner_id, tier, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	org := &model.Organization{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.OwnerID, &org.Tier, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// UpdateOwner changes owner_id of an organization.
func (r *OrganizationRepository) UpdateOwner(ctx context.Context, orgID, newOwnerID uuid.UUID) error {
	const query = `
		UPDATE organizations
		SET owner_id = $2, updated_at = NOW()
		WHERE id = $1
	`

	res, err := r.db.ExecContext(ctx, query, orgID, newOwnerID)
	if err != nil {
		return fmt.Errorf("failed to update organization owner: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if rows == 0 {
		return ErrOrganizationNotFound
	}

	return nil
}
