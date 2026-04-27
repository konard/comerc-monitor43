package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JSONB represents a JSONB value for PostgreSQL
type JSONB map[string]any

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal JSONB value: %v (type %T)", value, value)
	}
	return json.Unmarshal(bytes, j)
}

// OAuthAccount represents an OAuth-linked account
type OAuthAccount struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Provider       string    `json:"provider"` // google, github, yandex, vk, mailru, telegram
	ProviderUserID string    `json:"provider_user_id"`
	Email          string    `json:"email"`
	ProfileData    JSONB     `json:"profile_data,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// NewOAuthAccount creates a new OAuth account
func NewOAuthAccount(userID uuid.UUID, provider, providerUserID, email string, profileData JSONB) *OAuthAccount {
	return &OAuthAccount{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		ProfileData:    profileData,
		CreatedAt:      time.Now(),
	}
}

// ProfileDataValue safely retrieves a value from profile data
func (oa *OAuthAccount) ProfileDataValue(key string) (any, bool) {
	if oa.ProfileData == nil {
		return nil, false
	}
	val, ok := oa.ProfileData[key]
	return val, ok
}
