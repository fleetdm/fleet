package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/certserial"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	middleware_log "github.com/fleetdm/fleet/v4/server/service/middleware/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/go-kit/kit/endpoint"
)

// extractCertSerialFromHeader extracts certificate serial from X-Client-Cert-Serial
// header (set by load balancer during mTLS) for iOS/iPadOS device authentication.
func extractCertSerialFromHeader(ctx context.Context, r *http.Request) context.Context {
	serialStr := r.Header.Get("X-Client-Cert-Serial")
	if serialStr == "" {
		return ctx
	}

	serial, err := strconv.ParseUint(serialStr, 10, 64)
	if err != nil {
		// Force cert auth on parse error instead of falling back to token auth.
		return certserial.NewContext(ctx, 0)
	}

	return certserial.NewContext(ctx, serial)
}

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

// authenticatedDevice checks the validity of the device auth token
// provided in the request, and attaches the corresponding host to the
// context for the request.
func authenticatedDevice(svc fleet.Service, logger log.Logger, next endpoint.Endpoint) endpoint.Endpoint {
	authDeviceFunc := func(ctx context.Context, request interface{}) (interface{}, error) {
		identifier, err := getDeviceAuthToken(request)
		if err != nil {
			return nil, err
		}

		var host *fleet.Host
		var debug bool
		var authnMethod authz_ctx.AuthenticationMethod

		if certSerial, ok := certserial.FromContext(ctx); ok {
			// Header presence signals cert auth intent, even if serial is invalid.
			host, debug, err = svc.AuthenticateDeviceByCertificate(ctx, certSerial, identifier)
			authnMethod = authz_ctx.AuthnDeviceCertificate
		} else {
			// Try token auth first (hot path for Fleet Desktop).
			host, debug, err = svc.AuthenticateDevice(ctx, identifier)
			if err == nil {
				authnMethod = authz_ctx.AuthnDeviceToken
			} else {
				// Fallback to UUID auth for iOS/iPadOS self-service via URL.
				// The identifier (from {token}) is treated as the device UUID.
				host, debug, err = svc.AuthenticateIDeviceByURL(ctx, identifier)
				authnMethod = authz_ctx.AuthnDeviceURL
			}
		}

		if err != nil {
			logging.WithErr(ctx, err)
			return nil, err
		}

		hlogger := log.With(logger, "host_id", host.ID)
		if debug {
			logJSON(hlogger, request, "request")
		}

		ctx = hostctx.NewContext(ctx, host)
		// Register host as error context provider for ctxerr enrichment
		hostProvider := &hostctx.HostAttributeProvider{Host: host}
		ctx = ctxerr.AddErrorContextProvider(ctx, hostProvider)

		instrumentHostLogger(ctx, host.ID)
		if ac, ok := authz_ctx.FromContext(ctx); ok {
			ac.SetAuthnMethod(authnMethod)
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
	return middleware_log.Logged(authDeviceFunc)
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
		// Register host as error context provider for ctxerr enrichment
		hostProvider := &hostctx.HostAttributeProvider{Host: host}
		ctx = ctxerr.AddErrorContextProvider(ctx, hostProvider)

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
	return middleware_log.Logged(authHostFunc)
}

func authenticatedOrbitHost(
	svc fleet.Service,
	logger log.Logger,
	next endpoint.Endpoint,
	orbitNodeKeyGetter func(context.Context, interface{}) (string, error),
) endpoint.Endpoint {
	authHostFunc := func(ctx context.Context, request interface{}) (interface{}, error) {
		nodeKey, err := orbitNodeKeyGetter(ctx, request)
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
		// Register host as error context provider for ctxerr enrichment
		hostProvider := &hostctx.HostAttributeProvider{Host: host}
		ctx = ctxerr.AddErrorContextProvider(ctx, hostProvider)

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
	return middleware_log.Logged(authHostFunc)
}

func getOrbitNodeKey(ctx context.Context, r interface{}) (string, error) {
	if onk, err := r.(interface{ orbitHostNodeKey() string }); err {
		return onk.orbitHostNodeKey(), nil
	}
	return "", errors.New("error getting orbit node key")
}

func authHeaderValue(prefix string) func(ctx context.Context, r interface{}) (string, error) {
	return func(ctx context.Context, r interface{}) (string, error) {
		if authHeader, ok := ctx.Value(kithttp.ContextKeyRequestAuthorization).(string); ok {
			return strings.TrimPrefix(authHeader, prefix), nil
		}
		return "", nil
	}
}

func getNodeKey(r interface{}) (string, error) {
	if hnk, ok := r.(interface{ hostNodeKey() string }); ok {
		return hnk.hostNodeKey(), nil
	}
	return "", newOsqueryError("request type does not implement hostNodeKey method. This is likely a Fleet programmer error.")
}
