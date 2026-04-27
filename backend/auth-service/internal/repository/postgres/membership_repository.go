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

// ErrMembershipNotFound возвращается, когда membership не найден.
var ErrMembershipNotFound = errors.New("membership not found")

// MembershipRepository implements membership repository using PostgreSQL.
type MembershipRepository struct {
	db *DB
}

// NewMembershipRepository creates a new MembershipRepository.
func NewMembershipRepository(db *DB) interfaces.MembershipRepository {
	return &MembershipRepository{db: db}
}

// Create adds a member to organization.
func (r *MembershipRepository) Create(ctx context.Context, m *model.Membership) (*model.Membership, error) {
	const query = `
		INSERT INTO memberships (id, organization_id, user_id, role, joined_at, last_active_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, organization_id, user_id, role, joined_at, last_active_at
	`

	err := r.db.QueryRowContext(ctx, query,
		m.ID, m.OrganizationID, m.UserID, m.Role, m.JoinedAt, m.LastActiveAt,
	).Scan(
		&m.ID, &m.OrganizationID, &m.UserID, &m.Role, &m.JoinedAt, &m.LastActiveAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create membership: %w", err)
	}

	return m, nil
}

// GetByOrgAndUser retrieves a membership by (organization_id, user_id).
func (r *MembershipRepository) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*model.Membership, error) {
	const query = `
		SELECT id, organization_id, user_id, role, joined_at, last_active_at
		FROM memberships
		WHERE organization_id = $1 AND user_id = $2
	`

	m := &model.Membership{}
	err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(
		&m.ID, &m.OrganizationID, &m.UserID, &m.Role, &m.JoinedAt, &m.LastActiveAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Контракт: not-found возвращается как (nil, nil), чтобы
			// service-слой не импортировал postgres-пакет ради sentinel.
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get membership: %w", err)
	}

	return m, nil
}

// ListByOrg returns members of an organization with pagination.
func (r *MembershipRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, page, pageSize int) ([]*model.Membership, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	const countQuery = `SELECT COUNT(*) FROM memberships WHERE organization_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count memberships: %w", err)
	}

	const listQuery = `
		SELECT id, organization_id, user_id, role, joined_at, last_active_at
		FROM memberships
		WHERE organization_id = $1
		ORDER BY joined_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, listQuery, orgID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list memberships: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			return
		}
	}()

	var members []*model.Membership
	for rows.Next() {
		m := &model.Membership{}
		if err := rows.Scan(
			&m.ID, &m.OrganizationID, &m.UserID, &m.Role, &m.JoinedAt, &m.LastActiveAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan membership: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate memberships: %w", err)
	}

	return members, total, nil
}

// CountByOrg returns the number of active members in an organization.
func (r *MembershipRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	const query = `SELECT COUNT(*) FROM memberships WHERE organization_id = $1`

	var count int
	if err := r.db.QueryRowContext(ctx, query, orgID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count memberships: %w", err)
	}

	return count, nil
}

// UpdateRole changes the role of a member.
func (r *MembershipRepository) UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role model.Role) error {
	const query = `
		UPDATE memberships
		SET role = $3
		WHERE organization_id = $1 AND user_id = $2
	`

	res, err := r.db.ExecContext(ctx, query, orgID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to update membership role: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if rows == 0 {
		return ErrMembershipNotFound
	}

	return nil
}

// Delete removes a member from an organization.
func (r *MembershipRepository) Delete(ctx context.Context, orgID, userID uuid.UUID) error {
	const query = `DELETE FROM memberships WHERE organization_id = $1 AND user_id = $2`

	res, err := r.db.ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete membership: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if rows == 0 {
		return ErrMembershipNotFound
	}

	return nil
}

// FindByUserEmail returns a membership by (organization_id, user email).
func (r *MembershipRepository) FindByUserEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Membership, error) {
	const query = `
		SELECT m.id, m.organization_id, m.user_id, m.role, m.joined_at, m.last_active_at
		FROM memberships m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1 AND u.email = $2
	`

	m := &model.Membership{}
	err := r.db.QueryRowContext(ctx, query, orgID, email).Scan(
		&m.ID, &m.OrganizationID, &m.UserID, &m.Role, &m.JoinedAt, &m.LastActiveAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find membership by user email: %w", err)
	}

	return m, nil
}
