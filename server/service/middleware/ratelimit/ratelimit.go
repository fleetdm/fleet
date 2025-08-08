package ratelimit

import (
	"context"
	"fmt"
	"net/http"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log/level"
	kitlog "github.com/go-kit/log"
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

// ErrorMiddleware is a rate limiter that performs limits only when there is an error in the request
type ErrorMiddleware struct {
	store throttled.GCRAStore
}

// NewErrorMiddleware creates a new instance of ErrorMiddleware
func NewErrorMiddleware(store throttled.GCRAStore) *ErrorMiddleware {
	if store == nil {
		panic("nil store")
	}

	return &ErrorMiddleware{store: store}
}

// Limit returns a new middleware function enforcing the provided quota only when errors occur in the next middleware
func (m *ErrorMiddleware) Limit(keyName string, quota throttled.RateQuota, logger kitlog.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		limiter, err := throttled.NewGCRARateLimiter(m.store, quota)
		if err != nil {
			panic(err)
		}

		return func(ctx context.Context, req interface{}) (response interface{}, err error) {
			publicIP := publicip.FromContext(ctx)
			if publicIP == "" {
				level.Warn(logger).Log("msg", "missing public_ip, skipping rate limit")
				return next(ctx, req)
			}
			ipKeyName := fmt.Sprintf("%s-%s", keyName, publicIP)

			// RateLimit with quantity 0 will never get limited=true, so we check result.Remaining instead
			_, result, err := limiter.RateLimit(ipKeyName, 0)
			if err != nil {
				// This can happen if the limit store (e.g. Redis) is unavailable.
				//
				// We need to set authentication as checked, otherwise we end up returning HTTP 500 errors.
				if az, ok := authz_ctx.FromContext(ctx); ok {
					az.SetChecked()
				}
				return nil, ctxerr.Wrap(ctx, err, "rate limit ErrorMiddleware: failed to check rate limit")
			}
			if result.Remaining == 0 {
				// We need to set authentication as checked, otherwise we end up returning HTTP 500 errors.
				if az, ok := authz_ctx.FromContext(ctx); ok {
					az.SetChecked()
				}
				level.Warn(logger).Log(
					"ip", publicIP,
					"msg", "limit exceeded",
				)
				return nil, ctxerr.Wrap(ctx, &rateLimitError{result: result})
			}

			resp, err := next(ctx, req)
			if err != nil {
				_, _, rateErr := limiter.RateLimit(ipKeyName, 1)
				if rateErr != nil {
					// This can happen if the limit store (e.g. Redis) is unavailable.
					//
					// We need to set authentication as checked, otherwise we end up returning HTTP 500 errors.
					if az, ok := authz_ctx.FromContext(ctx); ok {
						az.SetChecked()
					}
					return nil, ctxerr.Wrap(ctx, err, "rate limit ErrorMiddleware: failed to increase rate limit")
				}
			}
			return resp, err
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
	// github.com/throttled/throttled has a bug where "peeking" with RateLimit(key, 0)
	// always returns a RetryAfter=-1. So we just return "limit exceeded" to prevent confusing
	// errors with "limit exceeded, retry after: 0s".
	ra := int(r.result.RetryAfter.Seconds())
	if ra > 0 {
		return fmt.Sprintf("limit exceeded, retry after: %ds", ra)
	}
	return "limit exceeded"
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
