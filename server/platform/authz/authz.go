// Package authz provides authorization interfaces for bounded contexts.
// This package contains only interfaces with no dependencies on fleet packages,
// allowing bounded contexts to use authorization without coupling to legacy code.
package authz

import "context"

// Action represents an authorization action.
type Action string

const (
	ActionRead Action = "read"
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
