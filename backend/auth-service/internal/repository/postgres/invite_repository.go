package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/repository/interfaces"
)

// ErrInviteNotFound возвращается, когда приглашение не найдено.
var ErrInviteNotFound = errors.New("invite not found")

// InviteRepository implements invite repository using PostgreSQL.
type InviteRepository struct {
	db *DB
}

// NewInviteRepository creates a new InviteRepository.
func NewInviteRepository(db *DB) interfaces.InviteRepository {
	return &InviteRepository{db: db}
}

// Create creates a new invite with PENDING status.
func (r *InviteRepository) Create(ctx context.Context, inv *model.Invite) (*model.Invite, error) {
	const query = `
		INSERT INTO invites (id, organization_id, inviter_id, email, role, token, status, created_at, expires_at, accepted_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, organization_id, inviter_id, email, role, token, status, created_at, expires_at, accepted_at, revoked_at
	`

	err := r.db.QueryRowContext(ctx, query,
		inv.ID, inv.OrganizationID, inv.InviterID, inv.Email, inv.Role,
		inv.Token, inv.Status, inv.CreatedAt, inv.ExpiresAt, inv.AcceptedAt, inv.RevokedAt,
	).Scan(
		&inv.ID, &inv.OrganizationID, &inv.InviterID, &inv.Email, &inv.Role,
		&inv.Token, &inv.Status, &inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	return inv, nil
}

// GetByID retrieves an invite by ID.
func (r *InviteRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Invite, error) {
	const query = `
		SELECT id, organization_id, inviter_id, email, role, token, status, created_at, expires_at, accepted_at, revoked_at
		FROM invites
		WHERE id = $1
	`

	inv := &model.Invite{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID, &inv.OrganizationID, &inv.InviterID, &inv.Email, &inv.Role,
		&inv.Token, &inv.Status, &inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInviteNotFound
		}
		return nil, fmt.Errorf("failed to get invite: %w", err)
	}

	return inv, nil
}

// GetByToken retrieves an invite by token.
func (r *InviteRepository) GetByToken(ctx context.Context, token uuid.UUID) (*model.Invite, error) {
	const query = `
		SELECT id, organization_id, inviter_id, email, role, token, status, created_at, expires_at, accepted_at, revoked_at
		FROM invites
		WHERE token = $1
	`

	inv := &model.Invite{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID, &inv.OrganizationID, &inv.InviterID, &inv.Email, &inv.Role,
		&inv.Token, &inv.Status, &inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInviteNotFound
		}
		return nil, fmt.Errorf("failed to get invite by token: %w", err)
	}

	return inv, nil
}

// FindPendingByOrgEmail returns a PENDING invite for (org, email) or nil if missing.
func (r *InviteRepository) FindPendingByOrgEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Invite, error) {
	const query = `
		SELECT id, organization_id, inviter_id, email, role, token, status, created_at, expires_at, accepted_at, revoked_at
		FROM invites
		WHERE organization_id = $1 AND email = $2 AND status = 'PENDING'
	`

	inv := &model.Invite{}
	err := r.db.QueryRowContext(ctx, query, orgID, email).Scan(
		&inv.ID, &inv.OrganizationID, &inv.InviterID, &inv.Email, &inv.Role,
		&inv.Token, &inv.Status, &inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find pending invite: %w", err)
	}

	return inv, nil
}

// CountPendingByOrg returns the number of PENDING invites in an organization.
func (r *InviteRepository) CountPendingByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	const query = `SELECT COUNT(*) FROM invites WHERE organization_id = $1 AND status = 'PENDING'`

	var count int
	if err := r.db.QueryRowContext(ctx, query, orgID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count pending invites: %w", err)
	}

	return count, nil
}

// UpdateStatus меняет статус приглашения и проставляет соответствующий timestamp.
// ACCEPTED -> accepted_at, REVOKED/EXPIRED/DECLINED -> revoked_at.
func (r *InviteRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InviteStatus, at time.Time) error {
	const query = `
		UPDATE invites
		SET status = $2::invite_status,
		    accepted_at = CASE WHEN $2::text = 'ACCEPTED' THEN $3 ELSE accepted_at END,
		    revoked_at  = CASE WHEN $2::text IN ('REVOKED', 'EXPIRED', 'DECLINED') THEN $3 ELSE revoked_at END
		WHERE id = $1
	`

	res, err := r.db.ExecContext(ctx, query, id, status, at)
	if err != nil {
		return fmt.Errorf("failed to update invite status: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if rows == 0 {
		return ErrInviteNotFound
	}

	return nil
}
