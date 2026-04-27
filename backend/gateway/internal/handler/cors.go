package handler

import (
	"net/http"
	"strings"
)

// corsMiddleware добавляет CORS-заголовки и обрабатывает preflight-запросы.
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}

	isAllowed := func(origin string) bool {
		if len(originSet) == 0 {
			return true
		}
		_, ok := originSet[origin]
		return ok
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && isAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
				w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
					"Authorization",
					"Content-Type",
					"Accept",
					"X-Request-ID",
					"X-Trace-ID",
				}, ", "))
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Preflight — сразу 204, дальше не идём.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
