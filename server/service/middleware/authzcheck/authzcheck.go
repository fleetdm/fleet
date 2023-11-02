// Package authzcheck implements a middleware that ensures that an authorization
// check was performed. This does not ensure that the correct authorization
// check was performed, but offers a backstop in case that a developer misses a
// check.
package authzcheck

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

// Middleware is the authzcheck middleware type.
type Middleware struct{}

// NewMiddleware returns a new authzcheck middleware.
func NewMiddleware() *Middleware {
	return &Middleware{}
}

func (m *Middleware) AuthzCheck() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			authzctx := &authz_ctx.AuthorizationContext{}
			ctx = authz_ctx.NewContext(ctx, authzctx)

			response, err := next(ctx, req)

			// If authentication check failed, return that error (so that we log
			// appropriately).
			var authFailedError *fleet.AuthFailedError
			var authRequiredError *fleet.AuthRequiredError
			var authHeaderRequiredError *fleet.AuthHeaderRequiredError
			if errors.As(err, &authFailedError) ||
				errors.As(err, &authRequiredError) ||
				errors.As(err, &authHeaderRequiredError) ||
				errors.Is(err, fleet.ErrPasswordResetRequired) {
				return nil, err
			}

			// TODO(mna): currently, any error detected before an authorization check gets
			// lost and the response is always Unauthorized because of the following condition.
			// I _think_ it would be safe to check here of response.error() returns a non-nil
			// error and if so, leave that error go through instead of returning a check missing
			// authorization error. To look into when addressing #4406.

			// If authorization was not checked, return a response that will
			// marshal to a generic error and log that the check was missed.
			if !authzctx.Checked() {
				// Getting to here means there is an authorization-related bug in our code.
				return nil, authz.CheckMissingWithResponse(response)
			}

			return response, err
		}
	}
}
