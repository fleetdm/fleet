// Package authz provides authorization interfaces for bounded contexts.
// This package contains only interfaces with no dependencies on fleet packages,
// allowing bounded contexts to use authorization without coupling to legacy code.
package authz

import (
	"context"
	"errors"
)

// Action represents an authorization action.
type Action string

const (
	ActionRead Action = "read"
	ActionList Action = "list"
)

// Authorizer is the interface for authorization checks.
type Authorizer interface {
	// Authorize checks if the current user (from context) can perform the action on the subject.
	// subject must implement AuthzTyper interface.
	Authorize(ctx context.Context, subject AuthzTyper, action Action) error
}

// AuthzTyper is implemented by types that can be authorized.
// Each bounded context defines its own authorization subjects that implement this interface.
type AuthzTyper interface {
	AuthzType() string
}

// Forbidden is an interface for authorization errors.
// Errors implementing this interface indicate that the requested action was forbidden.
type Forbidden interface {
	error
	Forbidden()
}

// IsForbidden returns true if the error (or any wrapped error) is a forbidden/authorization error.
func IsForbidden(err error) bool {
	var f Forbidden
	return errors.As(err, &f)
}
