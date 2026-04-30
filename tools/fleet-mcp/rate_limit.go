package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimitDefaults — chosen so a single client doing normal MCP work (a few
// tools/call requests per second) never trips the limiter, while a flooder
// gets throttled to a bounded request rate. Tunable via env if needed later.
const (
	defaultPerIPRatePerSec = 20 // tokens refilled per second per IP
	defaultPerIPBurst      = 60 // initial bucket size — short bursts allowed
	visitorTTL             = 10 * time.Minute
	visitorSweepInterval   = 1 * time.Minute
)

// visitor tracks a single client IP's limiter and last-seen time.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ipRateLimiter is a per-IP token-bucket throttle for the SSE transport.
// Backed by a map of net.IP → *rate.Limiter so each client gets its own
// bucket. The map is swept periodically to evict stale entries — without
// this, an attacker rotating source IPs would grow the map unbounded.
//
// Routes that need throttling wrap their handler with Middleware().
type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
}

// newIPRateLimiter constructs a limiter with the given per-second rate and
// burst, and starts the background sweeper.
func newIPRateLimiter(rps rate.Limit, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rps,
		burst:    burst,
	}
	go rl.sweepLoop()
	return rl
}

// getLimiter returns the limiter for ip, creating one on first sight.
func (rl *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// sweepLoop evicts visitor entries that haven't issued a request in
// visitorTTL. Runs forever — process death is the lifecycle.
func (rl *ipRateLimiter) sweepLoop() {
	ticker := time.NewTicker(visitorSweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-visitorTTL)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware throttles incoming requests by client IP. When a client exceeds
// the bucket, the request is rejected with 429 Too Many Requests rather than
// blocked — MCP clients should treat 429 as a retry signal.
func (rl *ipRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.getLimiter(ip).Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isTrustedProxy reports whether ip belongs to a network range we treat as a
// trusted proxy: loopback (local dev) or private/link-local (typical
// deployment topology where the app sits behind a sidecar or load balancer
// in the same VPC, e.g. Render). Public source IPs are never trusted —
// XFF from a public peer is attacker-controlled.
func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// clientIP returns the request's client IP. When the immediate peer
// (r.RemoteAddr) is a trusted proxy per isTrustedProxy, honors the first
// entry in X-Forwarded-For (per RFC 7239). Otherwise XFF is ignored and
// the peer address is used directly — preventing rate-limit bypass via a
// spoofed XFF header on directly-exposed deployments.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !isTrustedProxy(net.ParseIP(host)) {
		return host
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First entry is the original client per RFC 7239; subsequent entries
		// are appended by intermediate proxies. Trim because RFC 7230 allows
		// optional whitespace after the comma (and before it, in the wild) —
		// without trimming "1.2.3.4" and " 1.2.3.4" become distinct map keys
		// and weaken per-IP throttling.
		first, _, _ := strings.Cut(xff, ",")
		if first = strings.TrimSpace(first); first != "" {
			return first
		}
	}
	return host
}
