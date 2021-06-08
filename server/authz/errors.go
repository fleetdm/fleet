package authz

import (
	"net/http"

	"github.com/fleetdm/fleet/server/fleet"
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
	return "forbidden"
}

// StatusCode implements the service.ErrWithStatusCode interface.
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
