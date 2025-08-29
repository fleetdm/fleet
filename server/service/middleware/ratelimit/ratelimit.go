package ratelimit

import (
	"context"
	"fmt"
	"net/http"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/kit/endpoint"
	"github.com/throttled/throttled/v2"
)

// Middleware is a rate limiting middleware using the provided store. Each
// function wrapped by the rate limiter receives a separate quota.
type Middleware struct {
	store throttled.GCRAStore
}

// NewMiddleware initializes the middleware with the provided store.
func NewMiddleware(store throttled.GCRAStore) *Middleware {
	if store == nil {
		panic("nil store")
	}

	return &Middleware{store: store}
}

// Limit returns a new middleware function enforcing the provided quota.
func (m *Middleware) Limit(keyName string, quota throttled.RateQuota) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		limiter, err := throttled.NewGCRARateLimiter(m.store, quota)
		if err != nil {
			panic(err)
		}

		return func(ctx context.Context, req interface{}) (response interface{}, err error) {
			limited, result, err := limiter.RateLimit(keyName, 1)
			if err != nil {
				// This can happen if the limit store (e.g. Redis) is unavailable.
				//
				// We need to set authentication as checked, otherwise we end up returning HTTP 500
				// errors.
				if az, ok := authz_ctx.FromContext(ctx); ok {
					az.SetChecked()
				}
				return nil, ctxerr.Wrap(ctx, err, "rate limit Middleware: failed to increase rate limit")
			}

			if limited {
				// We need to set authentication as checked, otherwise we end up returning HTTP 500
				// errors.
				if az, ok := authz_ctx.FromContext(ctx); ok {
					az.SetChecked()
				}
				return nil, ctxerr.Wrap(ctx, &rateLimitError{result: result})
			}

			return next(ctx, req)
		}
	}
}

// Error is the interface for rate limiting errors.
type Error interface {
	error
	Result() throttled.RateLimitResult
}

type rateLimitError struct {
	result throttled.RateLimitResult
}

func (r rateLimitError) Error() string {
	return fmt.Sprintf("limit exceeded, retry after: %ds", r.RetryAfter())
}

func (r rateLimitError) StatusCode() int {
	return http.StatusTooManyRequests
}

func (r rateLimitError) RetryAfter() int {
	return int(r.result.RetryAfter.Seconds())
}

func (r rateLimitError) Result() throttled.RateLimitResult {
	return r.result
}
