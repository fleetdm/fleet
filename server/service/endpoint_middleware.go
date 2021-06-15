package service

import (
	"context"
	"reflect"

	"github.com/fleetdm/fleet/server/fleet"

	hostctx "github.com/fleetdm/fleet/server/contexts/host"
	"github.com/fleetdm/fleet/server/contexts/token"
	"github.com/fleetdm/fleet/server/contexts/viewer"
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
func authenticatedUser(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// first check if already successfully set
		if _, ok := viewer.FromContext(ctx); ok {
			return next(ctx, request)
		}

		// if not succesful, try again this time with errors
		sessionKey, ok := token.FromContext(ctx)
		if !ok {
			return nil, fleet.NewAuthRequiredError("no auth token")
		}

		v, err := authViewer(ctx, string(sessionKey), svc)
		if err != nil {
			return nil, err
		}

		ctx = viewer.NewContext(ctx, *v)
		return next(ctx, request)
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

func canPerformActions(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, fleet.ErrNoContext
		}
		if !vc.CanPerformActions() {
			return nil, fleet.NewPermissionError("no read permissions")
		}
		return next(ctx, request)
	}
}

func canReadUser(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, fleet.ErrNoContext
		}
		uid := requestUserIDFromContext(ctx)
		if !vc.CanPerformReadActionOnUser(uid) {
			return nil, fleet.NewPermissionError("no read permissions on user")
		}
		return next(ctx, request)
	}
}

func canModifyUser(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, fleet.ErrNoContext
		}
		uid := requestUserIDFromContext(ctx)
		if !vc.CanPerformWriteActionOnUser(uid) {
			return nil, fleet.NewPermissionError("no write permissions on user")
		}
		return next(ctx, request)
	}
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

func requestUserIDFromContext(ctx context.Context) uint {
	userID, ok := ctx.Value("request-id").(uint)
	if !ok {
		return 0
	}
	return userID
}
