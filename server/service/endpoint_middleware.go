package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kithttp "github.com/go-kit/kit/transport/http"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/go-kit/kit/endpoint"
)

func logJSON(logger log.Logger, v interface{}, key string) {
	jsonV, err := json.Marshal(v)
	if err != nil {
		level.Debug(logger).Log("err", fmt.Errorf("marshaling %s for debug: %w", key, err))
		return
	}
	level.Debug(logger).Log(key, string(jsonV))
}

// instrumentHostLogger adds host IP information and extras to the context logger.
func instrumentHostLogger(ctx context.Context, extras ...interface{}) {
	remoteAddr, _ := ctx.Value(kithttp.ContextKeyRequestRemoteAddr).(string)
	xForwardedFor, _ := ctx.Value(kithttp.ContextKeyRequestXForwardedFor).(string)
	logging.WithExtras(
		logging.WithNoUser(ctx),
		append(extras, "ip_addr", remoteAddr, "x_for_ip_addr", xForwardedFor)...,
	)
}

// authenticatedHost wraps an endpoint, checks the validity of the node_key
// provided in the request, and attaches the corresponding osquery host to the
// context for the request
func authenticatedHost(svc fleet.Service, logger log.Logger, next endpoint.Endpoint) endpoint.Endpoint {
	authHostFunc := func(ctx context.Context, request interface{}) (interface{}, error) {
		nodeKey, err := getNodeKey(request)
		if err != nil {
			return nil, err
		}

		host, debug, err := svc.AuthenticateHost(ctx, nodeKey)
		if err != nil {
			logging.WithErr(ctx, err)
			return nil, err
		}

		hlogger := log.With(logger, "host-id", host.ID)
		if debug {
			logJSON(hlogger, request, "request")
		}

		ctx = hostctx.NewContext(ctx, *host)
		instrumentHostLogger(ctx)

		resp, err := next(ctx, request)
		if err != nil {
			return nil, err
		}

		if debug {
			logJSON(hlogger, request, "response")
		}
		return resp, nil
	}
	return logged(authHostFunc)
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

		v, err := authViewer(ctx, string(sessionKey), svc)
		if err != nil {
			return nil, err
		}

		if v.User.IsAdminForcedPasswordReset() {
			return nil, fleet.ErrPasswordResetRequired
		}

		ctx = viewer.NewContext(ctx, *v)
		return next(ctx, request)
	}

	return logged(authUserFunc)
}

// logged wraps an endpoint and adds the error if the context supports it
func logged(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		res, err := next(ctx, request)
		if err != nil {
			logging.WithErr(ctx, err)
			return nil, err
		}
		if errResp, ok := res.(errorer); ok {
			err = errResp.error()
			if err != nil {
				logging.WithErr(ctx, err)
			}
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
