package main

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// rateLimiter tracks failed authentication attempts per IP address.
type rateLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

var authLimiter = &rateLimiter{
	failures: make(map[string][]time.Time),
}

const (
	rateLimitWindow = 5 * time.Minute
	rateLimitMax    = 10 // max failures per IP within the window
)

// isRateLimited returns true if the given IP has exceeded the failure threshold.
func (rl *rateLimiter) isRateLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)

	// Evict stale entries.
	recent := rl.failures[ip][:0]
	for _, t := range rl.failures[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) == 0 {
		delete(rl.failures, ip)
		return false
	}
	rl.failures[ip] = recent

	return len(recent) >= rateLimitMax
}

// recordFailure records a failed auth attempt for the given IP.
func (rl *rateLimiter) recordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.failures[ip] = append(rl.failures[ip], time.Now())
}

// bearerAuthMiddleware rejects requests whose Authorization header does not
// match "Bearer <token>", returning 401 Unauthorized. The comparison uses
// crypto/subtle.ConstantTimeCompare to prevent timing side-channel attacks.
// Failed attempts are logged and rate-limited per source IP.
func bearerAuthMiddleware(token string, next http.Handler) http.Handler {
	expected := []byte("Bearer " + token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		if authLimiter.isRateLimited(ip) {
			logrus.WithField("ip", ip).Warn("auth rate limit exceeded, rejecting request")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, expected) != 1 {
			authLimiter.recordFailure(ip)
			logrus.WithField("ip", ip).Warn("authentication failed: invalid bearer token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the client IP address from the request, preferring
// X-Forwarded-For or X-Real-IP when present (for deployments behind a reverse
// proxy), and stripping the port from RemoteAddr as a fallback.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain a comma-separated chain; the first
		// entry is the original client.
		first, _, _ := strings.Cut(xff, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
