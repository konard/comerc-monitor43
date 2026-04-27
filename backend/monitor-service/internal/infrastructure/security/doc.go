// Package security предоставляет middleware для HTTP security headers и CORS.
//
// Пакет реализует:
// - SecurityHeadersMiddleware: устанавливает security headers для защиты от XSS, clickjacking, и других атак
// - CORSMiddleware: управляет Cross-Origin Resource Sharing на основе whitelist origins
//
// Использование:
//
//	mux := http.NewServeMux()
//	secured := security.Chain(mux, security.SecurityHeadersMiddleware, security.CORSMiddleware(cfg))
//
// Security headers:
//   - X-Frame-Options: DENY
//   - X-Content-Type-Options: nosniff
//   - X-XSS-Protection: 1; mode=block
//   - Strict-Transport-Security: max-age=31536000
//   - Content-Security-Policy: default-src 'self'
//
// CORS headers:
//   - Access-Control-Allow-Origin (на основе whitelist)
//   - Access-Control-Allow-Methods: GET, POST, OPTIONS
//   - Access-Control-Allow-Headers: Content-Type, Authorization
//   - Access-Control-Max-Age: 86400
package security
