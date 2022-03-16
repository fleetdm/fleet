// Package authz defines the "authorization context", used to check that a
// request has had an authorization check performed before returning results.
package authz

import (
	"context"
	"sync"
)

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

// AuthenticationMethod identifies the method used to authenticate.
type AuthenticationMethod int

// List of supported authentication methods.
const (
	AuthnUserToken AuthenticationMethod = iota
	AuthnHostToken
	AuthnDeviceToken
)

// AuthorizationContext contains the context information used for the
// authorization check.
type AuthorizationContext struct {
	l sync.Mutex
	// checked indicates whether a call was made to check authorization for the request.
	checked bool
	// store the authentication method, as some methods cannot have granular authorizations.
	authnMethod AuthenticationMethod
}

func (a *AuthorizationContext) Checked() bool {
	a.l.Lock()
	defer a.l.Unlock()
	return a.checked
}

func (a *AuthorizationContext) SetChecked() {
	a.l.Lock()
	defer a.l.Unlock()
	a.checked = true
}

func (a *AuthorizationContext) AuthnMethod() AuthenticationMethod {
	a.l.Lock()
	defer a.l.Unlock()
	return a.authnMethod
}

func (a *AuthorizationContext) SetAuthnMethod(method AuthenticationMethod) {
	a.l.Lock()
	defer a.l.Unlock()
	a.authnMethod = method
}
