package service

import (
	"context"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/go-kit/kit/endpoint"
)

// authenticatedHost wraps an endpoint, checks the validity of the node_key
// provided in the request, and attaches the corresponding osquery host to the
// context for the request
func authenticatedHost(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		nodeKey, err := getNodeKey(request)
		if err != nil {
			return nil, err
		}

		host, err := svc.AuthenticateHost(ctx, nodeKey)
		if err != nil {
			return nil, err
		}

		ctx = hostctx.NewContext(ctx, *host)
		return next(ctx, request)
	}
}

func getNodeKey(r interface{}) (string, error) {
	// Retrieve node key by reflection (note that our options here
	// are limited by the fact that request is an interface{})
	v := reflect.ValueOf(r)
	if v.Kind() != reflect.Struct {
		return "", osqueryError{
			message: "request type is not struct. This is likely a Fleet programmer error.",
		}
	}
	nodeKeyField := v.FieldByName("NodeKey")
	if !nodeKeyField.IsValid() {
		return "", osqueryError{
			message: "request struct missing NodeKey. This is likely a Fleet programmer error.",
		}
	}
	if nodeKeyField.Kind() != reflect.String {
		return "", osqueryError{
			message: "NodeKey is not a string. This is likely a Fleet programmer error.",
		}
	}
	return nodeKeyField.String(), nil
}

// authenticatedUser wraps an endpoint, requires that the Fleet user is
// authenticated, and populates the context with a Viewer struct for that user.
//
// If auth fails or the user must reset their password, an error is returned.
func authenticatedUser(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
	authUserFunc := func(ctx context.Context, request interface{}) (interface{}, error) {
		// first check if already successfully set
		if v, ok := viewer.FromContext(ctx); ok {
			if v.User.AdminForcedPasswordReset {
				return nil, fleet.ErrPasswordResetRequired
			}

			return next(ctx, request)
		}

		// if not succesful, try again this time with errors
		sessionKey, ok := token.FromContext(ctx)
		if !ok {
			return nil, fleet.NewAuthHeaderRequiredError("no auth token")
		}

		v, err := authViewer(ctx, string(sessionKey), svc)
		if err != nil {
			return nil, err
		}

		if v.User.AdminForcedPasswordReset {
			return nil, fleet.ErrPasswordResetRequired
		}

		ctx = viewer.NewContext(ctx, *v)
		return next(ctx, request)
	}

	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		res, err := authUserFunc(ctx, request)
		if err != nil {
			logging.WithErr(ctx, err)
			return nil, err
		}
		return res, nil
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

func canPerformPasswordReset(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, fleet.ErrNoContext
		}
		if !vc.CanPerformPasswordReset() {
			return nil, fleet.NewPermissionError("cannot reset password")
		}
		return next(ctx, request)
	}
}
