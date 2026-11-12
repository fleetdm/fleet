package ratelimit

import (
	"fmt"
	"net/http"

	"github.com/realclientip/realclientip-go"
	"github.com/throttled/throttled/v2"
)

// clientIPVaryBy implements throttled.VaryBy using the real client IP extracted via a configured
// realclientip strategy. This correctly identifies clients behind load balancers/reverse proxies
// instead of rate-limiting by the proxy's IP address.
type clientIPVaryBy struct {
	strategy realclientip.Strategy
}

func (v *clientIPVaryBy) Key(r *http.Request) string {
	ip := v.strategy.ClientIP(r.Header, r.RemoteAddr)
	if ip == "" {
		return r.RemoteAddr
	}
	return ip
}

// DefaultHTTPRateQuota returns a reasonable default for HTTP-level rate limiting:
// 10 requests per minute per IP with burst of 9 (allows 10 initial requests before throttling).
func DefaultHTTPRateQuota() throttled.RateQuota {
	return throttled.RateQuota{
		MaxRate:  throttled.PerMin(10),
		MaxBurst: 9,
	}
}

// NewHTTPRateLimiter creates an HTTP-level rate limiter that varies by real client IP.
// The ipStrategy determines how to extract the real client IP from requests (e.g. from
// X-Forwarded-For headers when behind a load balancer). Use endpointer.NewClientIPStrategy
// to create a strategy from the server's trusted_proxies configuration.
func NewHTTPRateLimiter(store throttled.GCRAStore, quota throttled.RateQuota, ipStrategy realclientip.Strategy) (*throttled.HTTPRateLimiter, error) {
	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		return nil, fmt.Errorf("create rate limiter: %w", err)
	}

	return &throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      &clientIPVaryBy{strategy: ipStrategy},
	}, nil
}
