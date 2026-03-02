package middleware

import (
	"net/http"
	"valancis-backend/internal/domain"
)

// AdminMiddleware ensures the authenticated user has the 'admin' role.
// MUST be used AFTER AuthMiddleware.
func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
		if !ok || user == nil {
			http.Error(w, "Unauthorized: No user found in context", http.StatusUnauthorized)
			return
		}

		if user.Role != "admin" {
			http.Error(w, "Forbidden: Admins only", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
