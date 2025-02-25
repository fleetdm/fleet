package auth

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	"github.com/go-kit/kit/endpoint"
)

// AuthViewer creates an authenticated viewer by validating the session key.
func AuthViewer(ctx context.Context, sessionKey string, svc fleet.Service) (*viewer.Viewer, error) {
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

// AuthenticatedUser wraps an endpoint, requires that the Fleet user is
// authenticated, and populates the context with a Viewer struct for that user.
//
// If auth fails or the user must reset their password, an error is returned.
func AuthenticatedUser(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
	authUserFunc := func(ctx context.Context, request interface{}) (interface{}, error) {
		// first check if already successfully set
		if v, ok := viewer.FromContext(ctx); ok {
			if v.User.IsAdminForcedPasswordReset() {
				return nil, fleet.ErrPasswordResetRequired
			}

			return next(ctx, request)
		}

		// if not succesful, try again this time with errors
		sessionKey, ok := token.FromContext(ctx)
		if !ok {
			return nil, fleet.NewAuthHeaderRequiredError("no auth token")
		}

		v, err := AuthViewer(ctx, string(sessionKey), svc)
		if err != nil {
			return nil, err
		}

		if v.User.IsAdminForcedPasswordReset() {
			return nil, fleet.ErrPasswordResetRequired
		}

		ctx = viewer.NewContext(ctx, *v)
		if ac, ok := authz.FromContext(ctx); ok {
			ac.SetAuthnMethod(authz.AuthnUserToken)
		}
		return next(ctx, request)
	}

	return log.Logged(authUserFunc)
}

func UnauthenticatedRequest(_ fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
	return log.Logged(next)
}
