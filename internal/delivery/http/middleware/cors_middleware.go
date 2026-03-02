package middleware

import (
	"net/http"
	"strings"

	"valancis-backend/config"
)

// NewCORSMiddleware creates a CORS middleware with config injection
// L9: Avoid duplication - inject config instead of reading env directly
func NewCORSMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowedOrigins := strings.Split(cfg.AllowedOrigin, ",")

			for _, o := range allowedOrigins {
				o = strings.TrimSpace(o)
				if o == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
					break
				}
				if o == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			// If no match found but we usually want to allow something for dev sanity or strictness?
			// If not allowed, we just don't set the header, effectively blocking CORS for browsers.

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle Preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
