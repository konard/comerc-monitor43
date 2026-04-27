package model

import "time"

// UserContext represents authenticated user information
type UserContext struct {
	UserID    string
	Email     string
	Tier      string
	Valid     bool
	CreatedAt time.Time
}

// NewUserContext creates a new UserContext
func NewUserContext(userID, email, tier string) *UserContext {
	return &UserContext{
		UserID:    userID,
		Email:     email,
		Tier:      tier,
		Valid:     true,
		CreatedAt: time.Now(),
	}
}

// IsValid returns whether the user context is valid
func (u *UserContext) IsValid() bool {
	return u.Valid && u.UserID != ""
}

// HasTier checks if user has a specific tier
func (u *UserContext) HasTier(tier string) bool {
	return u.Tier == tier
}

// IsPremium checks if user has premium tier
func (u *UserContext) IsPremium() bool {
	return u.Tier == "premium"
}
