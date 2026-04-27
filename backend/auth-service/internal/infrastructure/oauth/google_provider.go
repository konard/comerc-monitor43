package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// GoogleProvider implements OAuth for Google
type GoogleProvider struct {
	config      *oauth2.Config
	client      *http.Client
	userInfoURL string
}

// googleUserInfo represents the response from Google's userinfo endpoint
type googleUserInfo struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// NewGoogleProvider creates a new Google OAuth provider.
// tokenURL и userInfoURL позволяют переопределить endpoint-ы для тестов.
func NewGoogleProvider(clientID, clientSecret, tokenURL, userInfoURL string) *GoogleProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "", // Will be set per request
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: tokenURL,
		},
	}

	return &GoogleProvider{
		config:      config,
		client:      &http.Client{Timeout: 30 * time.Second},
		userInfoURL: userInfoURL,
	}
}

// GetAuthURL returns the Google OAuth authorization URL
func (p *GoogleProvider) GetAuthURL(state, redirectURI string) string {
	config := p.config
	config.RedirectURL = redirectURI

	return config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)
}

// Exchange exchanges the authorization code for user info
func (p *GoogleProvider) Exchange(ctx context.Context, code, redirectURI string) (*UserInfo, error) {
	config := p.config
	config.RedirectURL = redirectURI

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %v", err)
	}

	// Get user info
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.userInfoURL+"?access_token="+token.AccessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build userinfo request: %v", err)
	}
	response, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			return
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status %d: %s", response.StatusCode, string(body))
	}

	var googleUser googleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %v", err)
	}

	if !googleUser.VerifiedEmail {
		return nil, fmt.Errorf("email is not verified")
	}

	return &UserInfo{
		ID:             googleUser.ID,
		Email:          googleUser.Email,
		Name:           googleUser.Name,
		Picture:        googleUser.Picture,
		Provider:       "google",
		ProviderUserID: googleUser.ID,
		// Include locale and other fields in profile data if needed
	}, nil
}

// ProviderName returns the provider name
func (p *GoogleProvider) ProviderName() string {
	return "google"
}
