package authz

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

const (
	// ForbiddenErrorMessage is the error message that should be returned to
	// clients when an action is forbidden. It is intentionally vague to prevent
	// disclosing information that a client should not have access to.
	ForbiddenErrorMessage = "forbidden"

	gitOpsMDMCommandErrorMessage = "Users with the GitOps role can no longer run MDM commands. " +
		"What to use instead: https://fleetdm.com/learn-more-about/fleet-5-migration"
)

// Forbidden is the error type for authorization errors
type Forbidden struct {
	message  string
	internal string
	subject  *fleet.User
	object   interface{}
	action   interface{}

	fleet.ErrorWithUUID
}

// ForbiddenWithInternal creates a new error that will return a simple
// "forbidden" to the client, logging internally the more detailed message
// provided.
func ForbiddenWithInternal(internal string, subject *fleet.User, object, action interface{}) *Forbidden {
	return &Forbidden{
		internal: internal,
		subject:  subject,
		object:   object,
		action:   action,
	}
}

// GitOpsMDMCommandError replaces err, a failed MDM command write authorization, with a Fleet 5 migration message when the
// viewer has the GitOps role globally or on the fleet of teamID. Callers gate on selective list (not list) so GitOps users
// reach this check.
func GitOpsMDMCommandError(ctx context.Context, teamID *uint, err error) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok || vc.User == nil {
		return err
	}
	isGitOps := vc.User.GlobalRole != nil && *vc.User.GlobalRole == fleet.RoleGitOps
	for _, team := range vc.User.Teams {
		if teamID != nil && team.ID == *teamID && team.Role == fleet.RoleGitOps {
			isGitOps = true
		}
	}
	if !isGitOps {
		return err
	}
	return &Forbidden{message: gitOpsMDMCommandErrorMessage, internal: err.Error(), subject: vc.User}
}

// Error implements the error interface.
func (e *Forbidden) Error() string {
	if e.message != "" {
		return e.message
	}
	return ForbiddenErrorMessage
}

// StatusCode implements the go-kit http StatusCoder interface.
func (e *Forbidden) StatusCode() int {
	return http.StatusForbidden
}

func (e *Forbidden) IsClientError() bool {
	return true
}

// Forbidden implements platform_authz.Forbidden interface.
func (e *Forbidden) Forbidden() {}

// Internal allows the internal error message to be logged.
func (e *Forbidden) Internal() string {
	return e.internal
}

// LogFields allows this error to be logged with subject, object, and action.
func (e *Forbidden) LogFields() []interface{} {
	// Only logging User's email, and not other details such as Password and Salt.
	email := "nil"
	if e.subject != nil {
		email = e.subject.Email
	}
	return []interface{}{
		"subject", email,
		"object", e.object,
		"action", e.action,
	}
}

// CheckMissing is the error to return when no authorization check was performed
// by the service.
//
// Deprecated: Use platform_http.CheckMissing instead. This alias is kept for
// backward compatibility.
type CheckMissing = platform_http.CheckMissing

// CheckMissingWithResponse creates a new error indicating the authorization
// check was missed, and including the response for further analysis by the error
// encoder.
//
// Deprecated: Use platform_http.CheckMissingWithResponse instead. This alias is
// kept for backward compatibility.
var CheckMissingWithResponse = platform_http.CheckMissingWithResponse
