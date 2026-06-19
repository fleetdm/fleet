package main

import (
	"net/http"

	"golang.org/x/time/rate"
)

// SSE rate limit — a local flood backstop, NOT a security control.
//
// The MCP is single-tenant (one shared MCP_AUTH_TOKEN for a handful of
// operators), so there is no per-client fairness to provide. This single GLOBAL
// token bucket just protects the MCP process from a request flood (including
// failed-auth requests, which are rejected here before ever reaching Fleet, so
// Fleet's own rate limiting doesn't cover them) and keeps the MCP from
// amplifying load onto Fleet.
const (
	defaultGlobalRatePerSec = 25  // tokens refilled per second (sustained ceiling)
	defaultGlobalBurst      = 100 // bucket size — well above interactive use, well below a flood
)

// globalRateLimiter is a single shared token-bucket throttle for the SSE
// transport. Sized so normal MCP traffic (a few operators issuing bursts of
// tools/call requests) never trips it, while a flooder gets 429-throttled.
type globalRateLimiter struct {
	lim *rate.Limiter
}

func newGlobalRateLimiter(rps rate.Limit, burst int) *globalRateLimiter {
	return &globalRateLimiter{lim: rate.NewLimiter(rps, burst)}
}

// Middleware rejects requests with 429 Too Many Requests once the shared bucket
// is empty (MCP clients should treat 429 as a retry signal).
func (rl *globalRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.lim.Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
