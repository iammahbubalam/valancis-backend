package middleware

import (
	"context"
	"net/http"
	"valancis-backend/internal/domain"
	"valancis-backend/pkg/utils"
	"strings"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get Token from Header or Cookie
		tokenString := ""
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			cookie, err := r.Cookie("accessToken")
			if err == nil {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
			return
		}

		// 2. Validate Token
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// 3. Set Context
		// We construct a partial user from the token claims to avoid a DB hit on every request.
		// If strict role checking against DB is needed (e.g. if role changes mid-session),
		// we would query the DB here. For now, token claims are sufficient.
		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		user := &domain.User{
			ID:    sub,
			Email: email,
			Role:  role,
		}

		ctx := context.WithValue(r.Context(), domain.UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
