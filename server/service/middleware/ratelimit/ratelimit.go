package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"runtime"

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
func (m *Middleware) Limit(quota throttled.RateQuota) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		// Get function name to use as a key for rate limiting (each wrapped function
		// gets a separate quota)
		funcName := runtime.FuncForPC(reflect.ValueOf(next).Pointer()).Name()

		limiter, err := throttled.NewGCRARateLimiter(m.store, quota)
		if err != nil {
			panic(err)
		}

		return func(ctx context.Context, req interface{}) (response interface{}, err error) {
			limited, result, err := limiter.RateLimit(funcName, 1)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "check rate limit")
			}
			if limited {
				return nil, ctxerr.Wrap(ctx, &ratelimitError{result: result})
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

type ratelimitError struct {
	result throttled.RateLimitResult
}

func (r ratelimitError) Error() string {
	return fmt.Sprintf("limit exceeded, retry after: %ds", int(r.result.RetryAfter.Seconds()))
}

func (r ratelimitError) StatusCode() int {
	return http.StatusTooManyRequests
}

func (r ratelimitError) RetryAfter() int {
	return int(r.result.RetryAfter.Seconds())
}

func (r ratelimitError) Result() throttled.RateLimitResult {
	return r.result
}
