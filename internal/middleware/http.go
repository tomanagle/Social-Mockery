package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging deliberately excludes URLs, query strings, headers and bodies: fixture
// tokens can appear in any of them. Logs remain useful without recording secrets.
func Logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		method := r.Method
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		default:
			method = "OTHER"
		}
		log.InfoContext(r.Context(), "request completed", "method", method, "duration_ms", time.Since(start).Milliseconds())
	})
}
