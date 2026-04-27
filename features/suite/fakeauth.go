//go:build bdd

package suite

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
)

// FakeOAuthServer — локальный HTTP сервер, имитирующий Google OAuth2.
// Всегда возвращает одного фиксированного пользователя.
// Endpoints:
//
//	POST /token     — обменивает code на access_token
//	GET  /userinfo  — возвращает данные пользователя
type FakeOAuthServer struct {
	srv   *http.Server
	addr  string
	Email string // возвращаемый email, можно менять между сценариями
	Name  string
}

// NewFakeOAuthServer создаёт и запускает fake OAuth сервер на случайном порту.
func NewFakeOAuthServer() (*FakeOAuthServer, error) {
	s := &FakeOAuthServer{
		Email: "bdd-user@test.example.com",
		Name:  "BDD Test User",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/token", s.handleToken)
	mux.HandleFunc("/userinfo", s.handleUserInfo)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s.addr = ln.Addr().String()
	s.srv = &http.Server{Handler: mux}

	go s.srv.Serve(ln) //nolint:errcheck // сервер в фоне

	return s, nil
}

// TokenURL возвращает URL для обмена кода на токен.
func (s *FakeOAuthServer) TokenURL() string { return "http://" + s.addr + "/token" }

// UserInfoURL возвращает URL для получения данных пользователя.
func (s *FakeOAuthServer) UserInfoURL() string { return "http://" + s.addr + "/userinfo" }

// Stop останавливает сервер.
func (s *FakeOAuthServer) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

// handleToken обменивает code на access_token.
func (s *FakeOAuthServer) handleToken(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//nolint:errcheck // write в http.ResponseWriter
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": "fake-bdd-token",
		"token_type":   "bearer",
		"expires_in":   3600,
	})
}

// handleUserInfo возвращает JSON с данными пользователя.
func (s *FakeOAuthServer) handleUserInfo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//nolint:errcheck // write в http.ResponseWriter
	_ = json.NewEncoder(w).Encode(map[string]any{
		"sub":            "fake-bdd-sub-001",
		"email":          s.Email,
		"name":           s.Name,
		"email_verified": true,
	})
}
