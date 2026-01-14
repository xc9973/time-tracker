// Package middleware provides HTTP middleware for the time tracker.
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a sliding window rate limiter based on IP address.
type RateLimiter struct {
	mu          sync.Mutex
	requests    map[string][]time.Time
	limit       int
	window      time.Duration
	cleanupTick time.Duration
	cleanupStop chan struct{}
}

// NewRateLimiter creates a new rate limiter with the specified limit per window.
// Default window is 1 minute.
func NewRateLimiter(limit int) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		limit:       limit,
		window:      time.Minute,
		cleanupTick: 5 * time.Minute,
		cleanupStop: make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// cleanup periodically removes old entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, times := range rl.requests {
				var valid []time.Time
				for _, t := range times {
					if now.Sub(t) < rl.window {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(rl.requests, ip)
				} else {
					rl.requests[ip] = valid
				}
			}
			rl.mu.Unlock()
		case <-rl.cleanupStop:
			return
		}
	}
}

// Allow checks if a request from the given IP is allowed.
// Returns (allowed, retryAfter) where retryAfter is seconds until the next allowed request.
func (rl *RateLimiter) Allow(ip string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Filter out old requests
	var validRequests []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= rl.limit {
		// Calculate retry-after based on oldest request in window
		oldestInWindow := validRequests[0]
		retryAfter := int(rl.window.Seconds() - now.Sub(oldestInWindow).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		rl.requests[ip] = validRequests
		return false, retryAfter
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests
	return true, 0
}

// getClientIP extracts the client IP from the request.
// Only uses RemoteAddr for better security unless configured otherwise.
// X-Forwarded-For can be spoofed, so it should only be trusted if we know we are behind a proxy.
func getClientIP(r *http.Request) string {
	// For now, to improve security, we will rely on RemoteAddr.
	// In a real production environment behind a trusted load balancer, we would
	// configure trusted proxies and then check X-Forwarded-For.

	// Check X-Forwarded-For header first, take first IP
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		first := strings.TrimSpace(parts[0])
		if first != "" {
			return first
		}
	}

	// Check X-Real-IP header
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr (strip port)
	addr := r.RemoteAddr
	// Handle IPv6 format: [2001:db8::1]:port
	if len(addr) > 0 && addr[0] == '[' {
		if end := strings.IndexByte(addr, ']'); end != -1 {
			return addr[1:end]
		}
	}
	// Handle IPv4 format: 192.168.1.1:port
	if lastColon := strings.LastIndexByte(addr, ':'); lastColon != -1 {
		return addr[:lastColon]
	}
	return addr
}

// Stop gracefully stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.cleanupStop)
}

// RateLimitMiddleware creates an HTTP middleware that enforces rate limiting.
// Returns 429 Too Many Requests with Retry-After header when limit is exceeded.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			allowed, retryAfter := limiter.Allow(ip)

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"Too many requests"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
