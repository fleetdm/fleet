package authz

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// ForbiddenErrorMessage is the error message that should be returned to
	// clients when an action is forbidden. It is intentionally vague to prevent
	// disclosing information that a client should not have access to.
	ForbiddenErrorMessage = "forbidden"
)

// Forbidden is the error type for authorization errors
type Forbidden struct {
	internal string
	subject  *fleet.User
	object   interface{}
	action   interface{}
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

// Error implements the error interface.
func (e *Forbidden) Error() string {
	return ForbiddenErrorMessage
}

// StatusCode implements the go-kit http StatusCoder interface.
func (e *Forbidden) StatusCode() int {
	return http.StatusForbidden
}

// Internal allows the internal error message to be logged.
func (e *Forbidden) Internal() string {
	return e.internal
}

// LogFields allows this error to be logged with subject, object, and action.
func (e *Forbidden) LogFields() []interface{} {
	return []interface{}{
		"subject", e.subject,
		"object", e.object,
		"action", e.action,
	}
}

// CheckMissing is the error to return when no authorization check was performed
// by the service.
type CheckMissing struct {
	response interface{}
}

// CheckMissingWithResponse creats a new error indicating the authorization
// check was missed, and including the response for further anaylis by the error
// encoder.
func CheckMissingWithResponse(response interface{}) *CheckMissing {
	return &CheckMissing{response: response}
}

func (e *CheckMissing) Error() string {
	return ForbiddenErrorMessage
}

func (e *CheckMissing) Internal() string {
	return "Missing authorization check"
}

func (e *CheckMissing) Response() interface{} {
	return e.response
}
