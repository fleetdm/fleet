package authz

import "net/http"

// Forbidden is the error type for authorization errors
type Forbidden struct {
	internal string
}

// ForbiddenWithInternal creates a new error that will return a simple
// "forbidden" to the client, logging internally the more detailed message
// provided.
func ForbiddenWithInternal(internal string) *Forbidden {
	return &Forbidden{internal: internal}
}

// Error implements the error interface.
func (e *Forbidden) Error() string {
	return "forbidden"
}

// StatusCode implements the service.ErrWithStatusCode interface.
func (e *Forbidden) StatusCode() int {
	return http.StatusForbidden
}

func (e *Forbidden) Internal() string {
	return e.internal
}
