package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// redirectTransport перенаправляет все запросы на заданный тестовый сервер.
type redirectTransport struct {
	base   http.RoundTripper
	srvURL string
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = rt.srvURL[len("http://"):]
	return rt.base.RoundTrip(req)
}

// newFakeOAuthServer поднимает тестовый сервер, имитирующий OAuth token + userinfo endpoints.
func newFakeOAuthServer(t *testing.T, tokenStatus int, tokenPayload string, userInfoStatus int, userInfoPayload string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tokenStatus)
			fmt.Fprint(w, tokenPayload)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(userInfoStatus)
			fmt.Fprint(w, userInfoPayload)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newFakeOAuthProviderClient возвращает http.Client, направляющий все запросы на srv.
func newFakeOAuthProviderClient(srv *httptest.Server) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{
			base:   http.DefaultTransport,
			srvURL: srv.URL,
		},
	}
}

func TestNewGoogleProvider(t *testing.T) {
	t.Parallel()

	p := NewGoogleProvider("client-id", "client-secret", "https://oauth2.googleapis.com/token", "https://www.googleapis.com/oauth2/v3/userinfo")

	require.NotNil(t, p)
	assert.Equal(t, "client-id", p.config.ClientID)
	assert.Equal(t, "client-secret", p.config.ClientSecret)
}

func TestGoogleProvider_ProviderName(t *testing.T) {
	t.Parallel()

	p := NewGoogleProvider("id", "secret", "https://oauth2.googleapis.com/token", "https://www.googleapis.com/oauth2/v3/userinfo")
	assert.Equal(t, "google", p.ProviderName())
}

func TestGoogleProvider_GetAuthURL(t *testing.T) {
	t.Parallel()

	p := NewGoogleProvider("my-client-id", "my-secret", "https://oauth2.googleapis.com/token", "https://www.googleapis.com/oauth2/v3/userinfo")
	url := p.GetAuthURL("state123", "https://example.com/callback")

	assert.Contains(t, url, "state123")
	assert.Contains(t, url, "my-client-id")
}

func TestGoogleProvider_Exchange_success(t *testing.T) {
	t.Parallel()

	tokenPayload := `{"access_token":"fake-token","token_type":"Bearer","expires_in":3600}`
	userInfoPayload := `{"sub":"user-123","email":"test@gmail.com","email_verified":true,"name":"Test User","picture":""}`

	srv := newFakeOAuthServer(t, http.StatusOK, tokenPayload, http.StatusOK, userInfoPayload)

	p := NewGoogleProvider("client-id", "client-secret", srv.URL+"/token", srv.URL+"/userinfo")
	p.config.Endpoint = oauth2.Endpoint{
		TokenURL:  srv.URL + "/token",
		AuthURL:   srv.URL + "/auth",
		AuthStyle: oauth2.AuthStyleInParams,
	}
	p.client = newFakeOAuthProviderClient(srv)

	info, err := p.Exchange(context.Background(), "valid-code", srv.URL+"/cb")
	require.NoError(t, err)
	assert.Equal(t, "user-123", info.ID)
	assert.Equal(t, "test@gmail.com", info.Email)
	assert.Equal(t, "google", info.Provider)
}

func TestGoogleProvider_Exchange_userinfo_non_200(t *testing.T) {
	t.Parallel()

	tokenPayload := `{"access_token":"fake-token","token_type":"Bearer","expires_in":3600}`

	srv := newFakeOAuthServer(t, http.StatusOK, tokenPayload, http.StatusUnauthorized, `{"error":"unauthorized"}`)

	p := NewGoogleProvider("client-id", "client-secret", srv.URL+"/token", srv.URL+"/userinfo")
	p.config.Endpoint = oauth2.Endpoint{
		TokenURL:  srv.URL + "/token",
		AuthURL:   srv.URL + "/auth",
		AuthStyle: oauth2.AuthStyleInParams,
	}
	p.client = newFakeOAuthProviderClient(srv)

	_, err := p.Exchange(context.Background(), "code", srv.URL+"/cb")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user info request failed")
}

func TestGoogleProvider_Exchange_userinfo_invalid_json(t *testing.T) {
	t.Parallel()

	tokenPayload := `{"access_token":"fake-token","token_type":"Bearer","expires_in":3600}`

	srv := newFakeOAuthServer(t, http.StatusOK, tokenPayload, http.StatusOK, `not-json`)

	p := NewGoogleProvider("client-id", "client-secret", srv.URL+"/token", srv.URL+"/userinfo")
	p.config.Endpoint = oauth2.Endpoint{
		TokenURL:  srv.URL + "/token",
		AuthURL:   srv.URL + "/auth",
		AuthStyle: oauth2.AuthStyleInParams,
	}
	p.client = newFakeOAuthProviderClient(srv)

	_, err := p.Exchange(context.Background(), "code", srv.URL+"/cb")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse user info")
}

func TestGoogleProvider_Exchange_unverified_email(t *testing.T) {
	t.Parallel()

	tokenPayload := `{"access_token":"fake-token","token_type":"Bearer","expires_in":3600}`
	userInfoPayload := `{"sub":"u-1","email":"unverified@gmail.com","email_verified":false,"name":"No Verify"}`

	srv := newFakeOAuthServer(t, http.StatusOK, tokenPayload, http.StatusOK, userInfoPayload)

	p := NewGoogleProvider("client-id", "client-secret", srv.URL+"/token", srv.URL+"/userinfo")
	p.config.Endpoint = oauth2.Endpoint{
		TokenURL:  srv.URL + "/token",
		AuthURL:   srv.URL + "/auth",
		AuthStyle: oauth2.AuthStyleInParams,
	}
	p.client = newFakeOAuthProviderClient(srv)

	_, err := p.Exchange(context.Background(), "code", srv.URL+"/cb")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email is not verified")
}

func TestGoogleProvider_Exchange_http_error(t *testing.T) {
	t.Parallel()

	// Сервер немедленно закрывается, чтобы вызвать ошибку при обращении к token endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srvURL := srv.URL
	srv.Close() // закрыт до вызова Exchange

	p := NewGoogleProvider("id", "secret", srvURL+"/token", srvURL+"/userinfo")
	p.config.Endpoint = oauth2.Endpoint{
		TokenURL:  srvURL + "/token",
		AuthURL:   srvURL + "/auth",
		AuthStyle: oauth2.AuthStyleInParams,
	}

	_, err := p.Exchange(context.Background(), "invalid-code", srvURL+"/cb")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to exchange code")
}
