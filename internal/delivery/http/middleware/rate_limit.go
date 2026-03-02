package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// L9: Lifecycle-managed rate limiter for graceful shutdown
type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter manages per-IP rate limiting with lifecycle control
type RateLimiter struct {
	clients       map[string]*client
	mu            sync.Mutex
	limit         rate.Limit
	burst         int
	cleanupPeriod time.Duration
	clientTTL     time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewRateLimiter creates a new RateLimiter with background cleanup
// limit: requests per second
// burst: maximum burst size
// cleanupPeriod: how often to remove stale clients
// clientTTL: how long before a client is considered stale
func NewRateLimiter(ctx context.Context, limit rate.Limit, burst int, cleanupPeriod, clientTTL time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:       make(map[string]*client),
		limit:         limit,
		burst:         burst,
		cleanupPeriod: cleanupPeriod,
		clientTTL:     clientTTL,
	}
	rl.ctx, rl.cancel = context.WithCancel(ctx)
	go rl.cleanupLoop()
	return rl
}

// Middleware returns the HTTP middleware handler
func (rl *RateLimiter) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			limiter := rl.getVisitor(ip)
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.clients[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.limit, rl.burst)
		rl.clients[ip] = &client{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupLoop runs periodic cleanup with context cancellation support
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.ctx.Done():
			return // Graceful shutdown
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, v := range rl.clients {
		if time.Since(v.lastSeen) > rl.clientTTL {
			delete(rl.clients, ip)
		}
	}
}

// Shutdown gracefully stops the cleanup goroutine
func (rl *RateLimiter) Shutdown() {
	rl.cancel()
}
