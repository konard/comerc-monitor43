package oauth

import "context"

// Provider defines the interface for OAuth providers
type Provider interface {
	// GetAuthURL returns the OAuth authorization URL
	GetAuthURL(state, redirectURI string) string

	// Exchange exchanges the authorization code for user info
	Exchange(ctx context.Context, code, redirectURI string) (*UserInfo, error)

	// ProviderName returns the provider name
	ProviderName() string
}

// UserInfo represents user information from OAuth provider
type UserInfo struct {
	ID             string
	Email          string
	Name           string
	Picture        string
	Provider       string // google, github, etc.
	ProviderUserID string
}
