package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/service/dto"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	ValidateToken(ctx context.Context, req *dto.ValidateTokenRequest) (*dto.ValidateTokenResponse, error)
	GetUserContext(ctx context.Context, token string) (*model.UserContext, error)
}

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// ValidateToken handles POST /api/v1/gateway/validate
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.ValidateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			return
		}
	}()

	if req.AccessToken == "" {
		h.respondError(w, "access_token is required", http.StatusBadRequest)
		return
	}

	resp, err := h.authService.ValidateToken(r.Context(), &req)
	if err != nil {
		h.respondError(w, "Token validation failed", http.StatusBadGateway)
		return
	}

	h.respondJSON(w, resp, http.StatusOK)
}

// respondJSON sends a JSON response
func (h *AuthHandler) respondJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// respondError sends an error response
func (h *AuthHandler) respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
