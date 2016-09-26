package server

import (
	"errors"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/kolide/kolide-ose/server/contexts/token"
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"golang.org/x/net/context"
)

var errNoContext = errors.New("context key not set")

func authenticated(jwtKey string, svc kolide.Service, next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// first check if already succesfuly set
		if _, ok := viewer.FromContext(ctx); ok {
			return next(ctx, request)
		}
		// if not succesful, try again this time with errors
		bearer, ok := token.FromContext(ctx)
		if !ok {
			return nil, authError{reason: "no auth token"}
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
func authViewer(ctx context.Context, jwtKey string, bearerToken string, svc kolide.Service) (*viewer.Viewer, error) {
	jwtToken, err := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtKey), nil
	})
	if err != nil {
		return nil, authError{reason: err.Error(), clientReason: "bad credentials"}
	}
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, authError{reason: "no jwt claims", clientReason: "bad credentials"}
	}
	sessionKeyClaim, ok := claims["session_key"]
	if !ok {
		return nil, authError{reason: "no session_key in JWT claims", clientReason: "bad credentials"}
	}
	sessionKey, ok := sessionKeyClaim.(string)
	if !ok {
		return nil, authError{reason: "non-string key in sessionClaim", clientReason: "bad credentials"}
	}
	session, err := svc.GetSessionByKey(ctx, sessionKey)
	if err != nil {
		return nil, authError{reason: err.Error(), clientReason: "bad credentials"}
	}
	user, err := svc.User(ctx, session.UserID)
	if err != nil {
		return nil, authError{reason: err.Error(), clientReason: "bad credentials"}
	}
	return &viewer.Viewer{User: user, Session: session}, nil
}

func mustBeAdmin(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errNoContext
		}
		if !vc.IsAdmin() {
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

type permission int

const (
	anyone permission = iota
	self
	admin
)

func validateModifyUserRequest(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(modifyUserRequest)
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errNoContext
		}
		uid := requestUserIDFromContext(ctx)
		p := r.payload
		must := requireRoleForUserModification(p)

		var badArgs []invalidArgument
		if !vc.IsAdmin() {
			for _, field := range must[admin] {
				badArgs = append(badArgs, invalidArgument{name: field, reason: "must be an admin"})
			}
		}
		if !vc.CanPerformWriteActionOnUser(uid) {
			for _, field := range must[self] {
				badArgs = append(badArgs, invalidArgument{name: field, reason: "no write permissions on user"})
			}
		}
		if len(badArgs) != 0 {
			return nil, permissionError{badArgs: badArgs}
		}
		return next(ctx, request)
	}
}

// checks if fields were set in a user payload
// returns a map of updated fields for each role required
func requireRoleForUserModification(p kolide.UserPayload) map[permission][]string {
	must := make(map[permission][]string)
	adminFields := []string{}
	if p.Enabled != nil {
		adminFields = append(adminFields, "enabled")
	}
	if p.Admin != nil {
		adminFields = append(adminFields, "admin")
	}
	if p.AdminForcedPasswordReset != nil {
		adminFields = append(adminFields, "force_password_reset")
	}
	if len(adminFields) != 0 {
		must[admin] = adminFields
	}

	selfFields := []string{}
	if p.Username != nil {
		selfFields = append(selfFields, "username")
	}
	if p.GravatarURL != nil {
		selfFields = append(selfFields, "gravatar_url")
	}
	if p.Position != nil {
		selfFields = append(selfFields, "position")
	}
	if p.Email != nil {
		selfFields = append(selfFields, "email")
	}
	if p.Password != nil {
		selfFields = append(selfFields, "password")
	}
	// self is always a must, otherwise
	// anyone can edit the field, and we don't have that requirement
	must[self] = selfFields
	return must
}

func requestUserIDFromContext(ctx context.Context) uint {
	userID, ok := ctx.Value("request-id").(uint)
	if !ok {
		return 0
	}
	return userID
}
