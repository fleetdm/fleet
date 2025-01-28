package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
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

// instrumentHostLogger adds host ID, IP information, and extras to the context logger.
func instrumentHostLogger(ctx context.Context, hostID uint, extras ...interface{}) {
	remoteAddr, _ := ctx.Value(kithttp.ContextKeyRequestRemoteAddr).(string)
	xForwardedFor, _ := ctx.Value(kithttp.ContextKeyRequestXForwardedFor).(string)
	logging.WithExtras(
		logging.WithNoUser(ctx),
		append(extras,
			"host_id", hostID,
			"ip_addr", remoteAddr,
			"x_for_ip_addr", xForwardedFor,
		)...,
	)
}

func authenticatedDevice(svc fleet.Service, logger log.Logger, next endpoint.Endpoint) endpoint.Endpoint {
	authDeviceFunc := func(ctx context.Context, request interface{}) (interface{}, error) {
		token, err := getDeviceAuthToken(request)
		if err != nil {
			return nil, err
		}

		host, debug, err := svc.AuthenticateDevice(ctx, token)
		if err != nil {
			logging.WithErr(ctx, err)
			return nil, err
		}

		hlogger := log.With(logger, "host_id", host.ID)
		if debug {
			logJSON(hlogger, request, "request")
		}

		ctx = hostctx.NewContext(ctx, host)
		instrumentHostLogger(ctx, host.ID)
		if ac, ok := authz_ctx.FromContext(ctx); ok {
			ac.SetAuthnMethod(authz_ctx.AuthnDeviceToken)
		}

		resp, err := next(ctx, request)
		if err != nil {
			return nil, err
		}

		if debug {
			logJSON(hlogger, request, "response")
		}
		return resp, nil
	}
	return logged(authDeviceFunc)
}

func getDeviceAuthToken(r interface{}) (string, error) {
	if dat, ok := r.(interface{ deviceAuthToken() string }); ok {
		return dat.deviceAuthToken(), nil
	}
	return "", fleet.NewAuthRequiredError("request type does not implement deviceAuthToken method. This is likely a Fleet programmer error.")
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

		hlogger := log.With(logger, "host_id", host.ID)
		if debug {
			logJSON(hlogger, request, "request")
		}

		ctx = hostctx.NewContext(ctx, host)
		instrumentHostLogger(ctx, host.ID)
		if ac, ok := authz_ctx.FromContext(ctx); ok {
			ac.SetAuthnMethod(authz_ctx.AuthnHostToken)
		}

		resp, err := next(ctx, request)
		if err != nil {
			return nil, err
		}

		if debug {
			logJSON(hlogger, resp, "response")
		}
		return resp, nil
	}
	return logged(authHostFunc)
}

func authenticatedOrbitHost(svc fleet.Service, logger log.Logger, next endpoint.Endpoint) endpoint.Endpoint {
	authHostFunc := func(ctx context.Context, request interface{}) (interface{}, error) {
		nodeKey, err := getOrbitNodeKey(request)
		if err != nil {
			return nil, err
		}

		host, debug, err := svc.AuthenticateOrbitHost(ctx, nodeKey)
		if err != nil {
			logging.WithErr(ctx, err)
			return nil, err
		}

		hlogger := log.With(logger, "host_id", host.ID)
		if debug {
			logJSON(hlogger, request, "request")
		}

		ctx = hostctx.NewContext(ctx, host)
		instrumentHostLogger(ctx, host.ID)
		if ac, ok := authz_ctx.FromContext(ctx); ok {
			ac.SetAuthnMethod(authz_ctx.AuthnOrbitToken)
		}

		resp, err := next(ctx, request)
		if err != nil {
			return nil, err
		}

		if debug {
			logJSON(hlogger, resp, "response")
		}
		return resp, nil
	}
	return logged(authHostFunc)
}

func getOrbitNodeKey(r interface{}) (string, error) {
	if onk, err := r.(interface{ orbitHostNodeKey() string }); err {
		return onk.orbitHostNodeKey(), nil
	}
	return "", errors.New("error getting orbit node key")
}

func getNodeKey(r interface{}) (string, error) {
	if hnk, ok := r.(interface{ hostNodeKey() string }); ok {
		return hnk.hostNodeKey(), nil
	}
	return "", newOsqueryError("request type does not implement hostNodeKey method. This is likely a Fleet programmer error.")
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
		if ac, ok := authz_ctx.FromContext(ctx); ok {
			ac.SetAuthnMethod(authz_ctx.AuthnUserToken)
		}
		return next(ctx, request)
	}

	return logged(authUserFunc)
}

func unauthenticatedRequest(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
	return logged(next)
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
