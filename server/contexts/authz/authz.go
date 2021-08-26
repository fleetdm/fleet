// Package authz defines the "authorization context", used to check that a
// request has had an authorization check performed before returning results.
package authz

import "context"

type key int

const authzKey key = 0

// NewContext creates a new context.Context with  an AuthorizationContext.
func NewContext(ctx context.Context, authz *AuthorizationContext) context.Context {
	return context.WithValue(ctx, authzKey, authz)
}

// FromContext returns a pointer to the AuthorizationContext.
func FromContext(ctx context.Context) (*AuthorizationContext, bool) {
	v, ok := ctx.Value(authzKey).(*AuthorizationContext)
	return v, ok
}

// AuthorizationContext contains the context information used for the
// authorization check.
type AuthorizationContext struct {
	// Checked indicates whether a call was made to check authorization for the request.
	Checked bool
}
