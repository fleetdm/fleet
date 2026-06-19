package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SSE rate limiting — a local flood backstop, NOT the brute-force defense.
//
// Brute force of MCP_AUTH_TOKEN is defended by the token-strength check in
// main.go (>= 32 chars). The limiter only sheds floods (including failed-auth
// requests, which are rejected here before ever reaching Fleet, so Fleet's own
// rate limiting doesn't cover them) and keeps the MCP from amplifying load onto
// Fleet.
//
// Two modes, selected by MCP_RATE_LIMIT_MODE:
//
//   - "global" (default): one shared token bucket. Right for a small, trusted
//     operator group, and for ANY deployment behind a reverse proxy / WAF —
//     there, per-client throttling belongs at the edge, not here.
//
//   - "ip": one bucket per client IP, keyed on the TCP peer
//     (http.Request.RemoteAddr). Intended for a DIRECT-connect internal-network
//     deployment, where it isolates a single noisy/compromised client without
//     throttling everyone else. It deliberately does NOT read X-Forwarded-For:
//     the TCP source address of an established connection can't be spoofed,
//     whereas XFF is client-supplied and can be spoofed. Trade-off:
//     behind a reverse proxy every request's RemoteAddr is the proxy's IP, so
//     all clients collapse into one bucket — this fails SAFE (over-throttles,
//     never under-throttles). Use "global" there instead.
const (
	defaultGlobalRatePerSec = 25  // tokens/sec refilled (sustained ceiling, all clients)
	defaultGlobalBurst      = 100 // shared bucket size — above interactive use, below a flood

	defaultPerIPRatePerSec = 10 // tokens/sec refilled per client IP
	defaultPerIPBurst      = 40 // per-IP bucket size

	perIPSweepInterval = 5 * time.Minute  // how often idle IP buckets are evicted
	perIPIdleTTL       = 10 * time.Minute // evict an IP not seen for at least this long
)

// Rate-limit mode values for MCP_RATE_LIMIT_MODE.
const (
	RateLimitModeGlobal = "global"
	RateLimitModeIP     = "ip"
)

type rateLimiter interface {
	Middleware(next http.Handler) http.Handler
}

func newRateLimiter(ctx context.Context, mode string) (rateLimiter, error) {
	switch mode {
	case RateLimitModeGlobal:
		return newGlobalRateLimiter(defaultGlobalRatePerSec, defaultGlobalBurst), nil
	case RateLimitModeIP:
		return newPerIPRateLimiter(ctx, defaultPerIPRatePerSec, defaultPerIPBurst), nil
	default:
		return nil, fmt.Errorf("invalid MCP_RATE_LIMIT_MODE %q (want %q or %q)", mode, RateLimitModeGlobal, RateLimitModeIP)
	}
}

// --- global mode --------------------------------------------------------------

type globalRateLimiter struct {
	lim *rate.Limiter
}

func newGlobalRateLimiter(rps rate.Limit, burst int) *globalRateLimiter {
	return &globalRateLimiter{lim: rate.NewLimiter(rps, burst)}
}

// Middleware rejects requests with 429 once the shared bucket is empty. It reads
// no client-controlled value, so a spoofed/rotated header can't reset it.
func (rl *globalRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.lim.Allow() {
			rejectRateLimited(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- per-IP mode --------------------------------------------------------------

type ipVisitor struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// perIPRateLimiter holds one token bucket per client IP. Buckets are created on
// first use and evicted once idle (see sweepLoop) so the map stays bounded.
type perIPRateLimiter struct {
	rps   rate.Limit
	burst int

	mu       sync.Mutex
	visitors map[string]*ipVisitor
}

func newPerIPRateLimiter(ctx context.Context, rps rate.Limit, burst int) *perIPRateLimiter {
	rl := &perIPRateLimiter{
		rps:      rps,
		burst:    burst,
		visitors: make(map[string]*ipVisitor),
	}
	go rl.sweepLoop(ctx)
	return rl
}

// Middleware throttles per client IP (the TCP peer). X-Forwarded-For is ignored
// on purpose — see clientIP and the file header.
func (rl *perIPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiterFor(clientIP(r)).Allow() {
			rejectRateLimited(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// limiterFor returns the bucket for ip, creating it on first use and refreshing
// its lastSeen so active clients aren't evicted.
func (rl *perIPRateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[ip]
	if !ok {
		v = &ipVisitor{lim: rate.NewLimiter(rl.rps, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.lim
}

// sweepLoop periodically evicts idle per-IP buckets to bound memory.
func (rl *perIPRateLimiter) sweepLoop(ctx context.Context) {
	t := time.NewTicker(perIPSweepInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done(): // shut down with the server (and lets tests stop it cleanly)
			return
		case <-t.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > perIPIdleTTL {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// clientIP is the per-IP key: the host portion of the TCP peer address. It
// deliberately ignores X-Forwarded-For — that header is client-supplied and
// spoofable, whereas the TCP source of an established connection is not.
// Per-IP mode is therefore only meaningful for direct client connections;
// behind a proxy every RemoteAddr is the proxy's, collapsing all
// clients into one bucket (fails safe — use "global" mode there).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // no port (or malformed) — use as-is
	}
	return host
}

func rejectRateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "1")
	http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
}
