package handler

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/raul/monitor/gateway/internal/model"
)

// Context key for user context
type contextKey string

const (
	userContextKey contextKey = "userContext"
)

// UserContextService defines the interface for user context operations
type UserContextService interface {
	GetUserContext(ctx context.Context, token string) (*model.UserContext, error)
}

// AuthMiddleware validates Bearer tokens and adds user context
type AuthMiddleware struct {
	authService UserContextService
	publicPaths map[string]bool
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService UserContextService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		publicPaths: map[string]bool{
			"/health":                     true,
			"/ready":                      true,
			"/live":                       true,
			"/api/v1/auth/register":       true,
			"/api/v1/auth/login":          true,
			"/api/v1/auth/oauth/login":    true,
			"/api/v1/auth/oauth/callback": true,
			"/api/v1/auth/token/refresh":  true,
		},
	}
}

// Middleware returns the middleware function
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public paths
		if m.isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := extractBearerToken(authHeader)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token
		userCtx, err := m.authService.GetUserContext(r.Context(), token)
		if err != nil || !userCtx.IsValid() {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), userContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isPublicPath checks if the path is public (doesn't require auth)
func (m *AuthMiddleware) isPublicPath(path string) bool {
	return m.publicPaths[path]
}

// LoggingMiddleware logs HTTP requests
type LoggingMiddleware struct {
	logger *log.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *log.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Middleware returns the middleware function
func (m *LoggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		m.logger.Printf("%s %s %s (%dms)",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			duration.Milliseconds(),
		)
	})
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) string {
	if userCtx, ok := ctx.Value(userContextKey).(*model.UserContext); ok {
		return userCtx.UserID
	}
	return ""
}

// GetUserContext extracts full user context from context
func GetUserContext(ctx context.Context) *model.UserContext {
	if userCtx, ok := ctx.Value(userContextKey).(*model.UserContext); ok {
		return userCtx
	}
	return nil
}

// extractBearerToken extracts token from Authorization header
func extractBearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.HasPrefix(header, prefix) {
		return header[len(prefix):]
	}
	return ""
}
