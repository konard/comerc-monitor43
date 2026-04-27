package model

import "errors"

var (
	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when a token is expired
	ErrExpiredToken = errors.New("token expired")

	// ErrMissingToken is returned when a token is missing
	ErrMissingToken = errors.New("missing token")
)

// TokenValidationRequest represents a token validation request
type TokenValidationRequest struct {
	AccessToken string
}

// TokenValidationResult represents the result of token validation
type TokenValidationResult struct {
	Valid  bool
	UserID string
	Email  string
	Tier   string
	Error  error
}

// NewTokenValidationResult creates a new validation result
func NewTokenValidationResult(valid bool, userID, email, tier string) *TokenValidationResult {
	return &TokenValidationResult{
		Valid:  valid,
		UserID: userID,
		Email:  email,
		Tier:   tier,
	}
}

// NewTokenValidationError creates a validation result with an error
func NewTokenValidationError(err error) *TokenValidationResult {
	return &TokenValidationResult{
		Valid: false,
		Error: err,
	}
}
