// Package authz provides authorization interfaces for bounded contexts.
// This package contains only interfaces with no dependencies on fleet packages,
// allowing bounded contexts to use authorization without coupling to legacy code.
package authz

import "context"

// Authorizer is the interface for authorization checks.
type Authorizer interface {
	// Authorize checks if the current user (from context) can perform the action on the subject.
	// subject must implement AuthzTyper interface.
	// action is typically "read", "write", "list", etc.
	Authorize(ctx context.Context, subject AuthzTyper, action string) error
}

// AuthzTyper is implemented by types that can be authorized.
// Each bounded context defines its own authorization subjects that implement this interface.
type AuthzTyper interface {
	AuthzType() string
}
