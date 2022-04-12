package main

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/openapi"
	"github.com/throttled/throttled/v2"
)

func limitMiddleware(limitStore throttled.GCRAStore, keyName string, quota throttled.RateQuota) openapi.MiddlewareFunc {
	return func(next openapi.HandlerFunc) openapi.HandlerFunc {
		limiter, err := throttled.NewGCRARateLimiter(limitStore, quota)
		if err != nil {
			// TODO: don't panic
			panic(err)
		}

		return func(w http.ResponseWriter, r *http.Request) error {
			limited, result, err := limiter.RateLimit(keyName, 1)
			if err != nil {
				// TODO: wrap errors properly?
				return fmt.Errorf("check rate limit: %w", err)
			}
			if limited {
				return &ratelimitError{result: result}
			}

			return next(w, r)
		}
	}
}

// TODO: don't copy this from ratelimit.go
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
