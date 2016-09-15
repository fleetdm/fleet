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
			return nil, forbiddenError{message: "must be logged in"}
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
			return nil, forbiddenError{message: "must be an admin"}
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
			return nil, forbiddenError{message: "no read permissions"}
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
		// TODO discuss the semantics of this check
		if !vc.CanPerformReadActionOnUser(uid) {
			return nil, forbiddenError{message: "no read permissions on user"}
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
			return nil, forbiddenError{message: "no write permissions on user"}
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

		// check for admin required fields
		if fields, ok := must[admin]; ok {
			if !vc.IsAdmin() {
				return nil, forbiddenError{message: "must be an admin", fields: fields}
			}
		}

		// check if any fields which the user can update themselves were set
		if fields, ok := must[self]; ok {
			if !vc.CanPerformWriteActionOnUser(uid) {
				return nil, forbiddenError{message: "no write permission on user", fields: fields}
			}
		}

		// check password reset permissions
		// must be either self or an admin
		if p.Password != nil {
			if vc.IsUserID(uid) || vc.IsAdmin() {
				return nil, forbiddenError{
					message: "must be your own account or an admin",
					fields:  []string{"password"},
				}
			}
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
