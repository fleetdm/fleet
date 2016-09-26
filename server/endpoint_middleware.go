package server

import (
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

var errNoContext = errors.New("no viewer context set")

func authenticated(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, err := viewerContextFromContext(ctx)
		if err != nil {
			return nil, err
		}
		if !vc.IsLoggedIn() {
			return nil, authError{reason: "must be logged in", clientReason: "must be logged in"}
		}
		return next(ctx, request)
	}
}

func mustBeAdmin(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, err := viewerContextFromContext(ctx)
		if err != nil {
			return nil, err
		}
		if !vc.IsAdmin() {
			return nil, permissionError{message: "must be an admin"}
		}
		return next(ctx, request)
	}
}

func canPerformActions(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, err := viewerContextFromContext(ctx)
		if err != nil {
			return nil, err
		}
		if !vc.CanPerformActions() {
			return nil, permissionError{message: "no read permissions"}
		}
		return next(ctx, request)
	}
}

func canReadUser(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, err := viewerContextFromContext(ctx)
		if err != nil {
			return nil, err
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
		vc, err := viewerContextFromContext(ctx)
		if err != nil {
			return nil, err
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
		vc, err := viewerContextFromContext(ctx)
		if err != nil {
			return nil, err
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
