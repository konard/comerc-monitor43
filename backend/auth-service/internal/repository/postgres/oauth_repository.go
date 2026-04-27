package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/repository/interfaces"
)

// OAuthRepository implements OAuth account repository using PostgreSQL
type OAuthRepository struct {
	db *DB
}

// NewOAuthRepository creates a new OAuthRepository
func NewOAuthRepository(db *DB) interfaces.OAuthRepository {
	return &OAuthRepository{db: db}
}

// Create creates a new OAuth account
func (r *OAuthRepository) Create(ctx context.Context, account *model.OAuthAccount) (*model.OAuthAccount, error) {
	const query = `
		INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id, email, profile_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, provider, provider_user_id, email, profile_data, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		account.ID, account.UserID, account.Provider, account.ProviderUserID,
		account.Email, account.ProfileData, account.CreatedAt,
	).Scan(
		&account.ID, &account.UserID, &account.Provider, &account.ProviderUserID,
		&account.Email, &account.ProfileData, &account.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth account: %v", err)
	}

	return account, nil
}

// GetByID retrieves an OAuth account by ID
func (r *OAuthRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.OAuthAccount, error) {
	const query = `
		SELECT id, user_id, provider, provider_user_id, email, profile_data, created_at
		FROM oauth_accounts
		WHERE id = $1
	`

	account := &model.OAuthAccount{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.UserID, &account.Provider, &account.ProviderUserID,
		&account.Email, &account.ProfileData, &account.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth account: %v", err)
	}

	return account, nil
}

// GetByProviderUserID retrieves an OAuth account by provider and provider user ID
func (r *OAuthRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*model.OAuthAccount, error) {
	const query = `
		SELECT id, user_id, provider, provider_user_id, email, profile_data, created_at
		FROM oauth_accounts
		WHERE provider = $1 AND provider_user_id = $2
	`

	account := &model.OAuthAccount{}
	err := r.db.QueryRowContext(ctx, query, provider, providerUserID).Scan(
		&account.ID, &account.UserID, &account.Provider, &account.ProviderUserID,
		&account.Email, &account.ProfileData, &account.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth account by provider user ID: %v", err)
	}

	return account, nil
}

// GetByUserID retrieves all OAuth accounts for a user
func (r *OAuthRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.OAuthAccount, error) {
	const query = `
		SELECT id, user_id, provider, provider_user_id, email, profile_data, created_at
		FROM oauth_accounts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth accounts: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			return
		}
	}()

	var accounts []*model.OAuthAccount
	for rows.Next() {
		account := &model.OAuthAccount{}
		err := rows.Scan(
			&account.ID, &account.UserID, &account.Provider, &account.ProviderUserID,
			&account.Email, &account.ProfileData, &account.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OAuth account: %v", err)
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// Delete deletes an OAuth account
func (r *OAuthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM oauth_accounts WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete OAuth account: %v", err)
	}

	return nil
}
