package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/openapi"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/throttled/throttled/v2"
)

// errMiddlware stores errors and responds with json
// TODO: should accept a errorstore.Handler!
func errorMiddleware(eh *errorstore.Handler, logger kitlog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			ctx := context.WithValue(r.Context(), openapi.ErrorContextKey{}, &err)

			// TODO: log the path, see handler.go:33

			next.ServeHTTP(w, r.WithContext(ctx))
			if err != nil {
				eh.Store(err)

				// TODO: log the error with extra fields, see handler.go:50
				level.Error(logger).Log("err", err)

				service.EncodeError(w, err)
			}
		})
	}
}

// TODO: need a better name
func authMiddleware(svc fleet.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			bearer := token.FromHTTPRequest(r)
			ctx = token.NewContext(ctx, bearer) // TODO: even if bearer is ""?
			if bearer != "" {
				v, err := authViewer(ctx, string(bearer), svc)
				if err == nil {
					ctx = viewer.NewContext(ctx, *v)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

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

func authUserMiddleware(svc fleet.Service) openapi.MiddlewareFunc {
	return func(next openapi.HandlerFunc) openapi.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx := r.Context()

			// first check if already successfully set
			if v, ok := viewer.FromContext(ctx); ok {
				if v.User.IsAdminForcedPasswordReset() {
					return fleet.ErrPasswordResetRequired
				}

				return next(w, r)
			}

			// if not succesful, try again this time with errors
			sessionKey, ok := token.FromContext(ctx)
			if !ok {
				return fleet.NewAuthHeaderRequiredError("no auth token")
			}

			v, err := authViewer(ctx, string(sessionKey), svc)
			if err != nil {
				return err
			}

			if v.User.IsAdminForcedPasswordReset() {
				return fleet.ErrPasswordResetRequired
			}

			ctx = viewer.NewContext(ctx, *v)
			if ac, ok := authz_ctx.FromContext(ctx); ok {
				ac.SetAuthnMethod(authz_ctx.AuthnUserToken)
			}

			return next(w, r)
		}
	}
}

// authViewer creates an authenticated viewer by validating the session key.
func authViewer(ctx context.Context, sessionKey string, svc fleet.Service) (*viewer.Viewer, error) {
	session, err := svc.GetSessionByKey(ctx, sessionKey)
	if err != nil {
		return nil, fleet.NewAuthRequiredError(err.Error())
	}
	user, err := svc.UserUnauthorized(ctx, session.UserID)
	if err != nil {
		return nil, fleet.NewAuthRequiredError(err.Error())
	}
	return &viewer.Viewer{User: user, Session: session}, nil
}

func authDeviceMiddleware(svc fleet.Service) openapi.MiddlewareFunc {
	return func(next openapi.HandlerFunc) openapi.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			// TODO: implement
			return next(w, r)
		}
	}
}
