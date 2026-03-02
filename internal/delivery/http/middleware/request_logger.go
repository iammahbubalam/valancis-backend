package middleware

import (
	"net/http"
	"valancis-backend/pkg/logger"
	"valancis-backend/pkg/utils"
	"time"

	"github.com/google/uuid"
)

// RequestLogger logs all HTTP requests with timing and status
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()[:8]

		// Create logger with request ID
		reqLogger := logger.WithRequestID(requestID)

		// Add logger to context
		ctx := logger.NewContext(r.Context(), &reqLogger)
		r = r.WithContext(ctx)

		// Add request ID to response header
		w.Header().Set("X-Request-ID", requestID)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Get user ID if authenticated
		userID := ""
		if claims, err := utils.ExtractClaims(r); err == nil && claims != nil {
			userID = claims.UserID
		}

		// Log all requests
		logEvent := reqLogger.Info()
		if wrapped.statusCode >= 500 {
			logEvent = reqLogger.Error()
		} else if wrapped.statusCode >= 400 {
			logEvent = reqLogger.Warn()
		}

		logEvent.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Int("status", wrapped.statusCode).
			Dur("duration_ms", duration).
			Str("ip", getClientIP(r)).
			Str("origin", r.Header.Get("Origin")).
			Str("user_agent", r.UserAgent()).
			Str("user_id", userID).
			Msg("HTTP")
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
