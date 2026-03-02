package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger is the global logger instance
var log zerolog.Logger

// ContextKey for storing logger in context
type ctxKey struct{}

// Init initializes the global logger
func Init(env string, logLevel string) {
	// Set time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Default output
	var output io.Writer = os.Stdout

	// Pretty console output for development
	if env == "development" || env == "dev" || env == "" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
	}

	// Parse log level
	var level zerolog.Level
	switch logLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn", "warning":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	log = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()
}

// Get returns the global logger
func Get() *zerolog.Logger {
	return &log
}

// WithContext returns a logger with context
func WithContext(ctx context.Context) *zerolog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zerolog.Logger); ok {
		return l
	}
	return &log
}

// NewContext creates a new context with the logger
func NewContext(ctx context.Context, l *zerolog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// WithRequestID adds a request ID to the logger
func WithRequestID(requestID string) zerolog.Logger {
	return log.With().Str("request_id", requestID).Logger()
}

// WithUserID adds a user ID to the logger
func WithUserID(l zerolog.Logger, userID string) zerolog.Logger {
	return l.With().Str("user_id", userID).Logger()
}

// --- Convenience Methods ---

// Debug logs a debug message
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info logs an info message
func Info() *zerolog.Event {
	return log.Info()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs an error message
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal logs a fatal message and exits
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// --- Structured Logging Helpers ---

// HTTPRequest logs an HTTP request
func HTTPRequest(method, path string, statusCode int, duration time.Duration, userID string) {
	event := log.Info().
		Str("method", method).
		Str("path", path).
		Int("status", statusCode).
		Dur("duration_ms", duration)

	if userID != "" {
		event = event.Str("user_id", userID)
	}

	event.Msg("HTTP Request")
}

// DBQuery logs a database query
func DBQuery(query string, duration time.Duration, err error) {
	event := log.Debug().
		Str("query", query).
		Dur("duration_ms", duration)

	if err != nil {
		event.Err(err).Msg("DB Query Failed")
	} else {
		event.Msg("DB Query")
	}
}

// ServiceStart logs service startup
func ServiceStart(name, version, port string) {
	log.Info().
		Str("service", name).
		Str("version", version).
		Str("port", port).
		Msg("Service Started")
}

// ServiceStop logs service shutdown
func ServiceStop(name string) {
	log.Info().
		Str("service", name).
		Msg("Service Stopped")
}
