package service

import (
	"context"
	"reflect"

	jwt "github.com/dgrijalva/jwt-go"
	hostctx "github.com/fleetdm/fleet/server/contexts/host"
	"github.com/fleetdm/fleet/server/contexts/token"
	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/endpoint"
	"github.com/pkg/errors"
)

var errNoContext = errors.New("context key not set")

// authenticatedHost wraps an endpoint, checks the validity of the node_key
// provided in the request, and attaches the corresponding osquery host to the
// context for the request
func authenticatedHost(svc kolide.Service, next endpoint.Endpoint) endpoint.Endpoint {
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
func authenticatedUser(jwtKey string, svc kolide.Service, next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// first check if already successfully set
		if _, ok := viewer.FromContext(ctx); ok {
			return next(ctx, request)
		}

		// if not succesful, try again this time with errors
		bearer, ok := token.FromContext(ctx)
		if !ok {
			return nil, authRequiredError{internal: "no auth token"}
		}

		v, err := authViewer(ctx, jwtKey, bearer, svc)
		if err != nil {
			return nil, err
		}

		ctx = viewer.NewContext(ctx, *v)
		return next(ctx, request)
	}
}

// authViewer creates an authenticated viewer by validating a JWT token.
func authViewer(ctx context.Context, jwtKey string, bearerToken token.Token, svc kolide.Service) (*viewer.Viewer, error) {
	jwtToken, err := jwt.Parse(string(bearerToken), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtKey), nil
	})
	if err != nil {
		return nil, authRequiredError{internal: err.Error()}
	}
	if !jwtToken.Valid {
		return nil, authRequiredError{internal: "invalid jwt token"}
	}
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, authRequiredError{internal: "no jwt claims"}
	}
	sessionKeyClaim, ok := claims["session_key"]
	if !ok {
		return nil, authRequiredError{internal: "no session_key in JWT claims"}
	}
	sessionKey, ok := sessionKeyClaim.(string)
	if !ok {
		return nil, authRequiredError{internal: "non-string key in sessionClaim"}
	}
	session, err := svc.GetSessionByKey(ctx, sessionKey)
	if err != nil {
		return nil, authRequiredError{internal: err.Error()}
	}
	user, err := svc.User(ctx, session.UserID)
	if err != nil {
		return nil, authRequiredError{internal: err.Error()}
	}
	return &viewer.Viewer{User: user, Session: session}, nil
}

func mustBeAdmin(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errNoContext
		}
		if !vc.CanPerformAdminActions() {
			return nil, permissionError{message: "must be an admin"}
		}
		return next(ctx, request)
	}
}

func canPerformActions(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errNoContext
		}
		if !vc.CanPerformActions() {
			return nil, permissionError{message: "no read permissions"}
		}
		return next(ctx, request)
	}
}

func canReadUser(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errNoContext
		}
		uid := requestUserIDFromContext(ctx)
		if !vc.CanPerformReadActionOnUser(uid) {
			return nil, permissionError{message: "no read permissions on user"}
		}
		return next(ctx, request)
	}
}

func canModifyUser(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errNoContext
		}
		uid := requestUserIDFromContext(ctx)
		if !vc.CanPerformWriteActionOnUser(uid) {
			return nil, permissionError{message: "no write permissions on user"}
		}
		return next(ctx, request)
	}
}

func canPerformPasswordReset(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errNoContext
		}
		if !vc.CanPerformPasswordReset() {
			return nil, permissionError{message: "cannot reset password"}
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
