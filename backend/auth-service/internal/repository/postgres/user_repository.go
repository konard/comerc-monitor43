package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/repository/interfaces"
)

// UserRepository implements user repository using PostgreSQL
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *DB) interfaces.UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, full_name, tier, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, password_hash, full_name, tier, created_at, updated_at, last_login_at, login_attempts, locked_until
	`

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FullName, user.Tier,
		user.CreatedAt, user.UpdatedAt,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Tier,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.LoginAttempts, &user.LockedUntil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	const query = `
		SELECT id, email, password_hash, full_name, tier, created_at, updated_at, last_login_at, login_attempts, locked_until
		FROM users
		WHERE id = $1
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Tier,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.LoginAttempts, &user.LockedUntil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	const query = `
		SELECT id, email, password_hash, full_name, tier, created_at, updated_at, last_login_at, login_attempts, locked_until
		FROM users
		WHERE email = $1
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Tier,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.LoginAttempts, &user.LockedUntil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %v", err)
	}

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	const query = `
		UPDATE users
		SET email = $2, password_hash = $3, full_name = $4, tier = $5, updated_at = $6,
		    last_login_at = $7, login_attempts = $8, locked_until = $9
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FullName, user.Tier, user.UpdatedAt,
		user.LastLoginAt, user.LoginAttempts, user.LockedUntil,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	return nil
}

// UpdateLastLogin updates last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE users
		SET last_login_at = NOW(), updated_at = NOW(), login_attempts = 0, locked_until = NULL
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %v", err)
	}

	return nil
}

// IncrementLoginAttempts increments failed login counter
func (r *UserRepository) IncrementLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE users
		SET login_attempts = login_attempts + 1, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to increment login attempts: %v", err)
	}

	return nil
}

// ResetLoginAttempts resets failed login counter
func (r *UserRepository) ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE users
		SET login_attempts = 0, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to reset login attempts: %v", err)
	}

	return nil
}

// LockAccount locks a user account until specified time
func (r *UserRepository) LockAccount(ctx context.Context, userID uuid.UUID, lockedUntil any) error {
	const query = `
		UPDATE users
		SET locked_until = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID, lockedUntil)
	if err != nil {
		return fmt.Errorf("failed to lock account: %v", err)
	}

	return nil
}

// UnlockAccount unlocks a user account
func (r *UserRepository) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE users
		SET locked_until = NULL, login_attempts =0, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unlock account: %v", err)
	}

	return nil
}

// ExistsByEmail checks if a user with given email exists
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %v", err)
	}

	return exists, nil
}
