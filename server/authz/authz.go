// Package authz implements the authorization checking logic via Go and OPA's
// Rego.
//
// Policy is defined in policy.rego. Policy is evaluated by Authorizer, defined
// in authz.go.
//
// See https://www.openpolicyagent.org/ for more details on OPA and Rego.
package authz

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/open-policy-agent/opa/rego"
)

// Authorizer stores the compiled policy and performs authorization checks.
type Authorizer struct {
	query rego.PreparedEvalQuery
}

// Load the policy from policy.rego in this directory.
//go:embed policy.rego
var policy string

// NewAuthorizer creates a new authorizer by compiling the policy embedded in
// policy.rego.
func NewAuthorizer() (*Authorizer, error) {
	ctx := context.Background()
	query, err := rego.New(
		rego.Query("allowed = data.authz.allow"),
		rego.Module("policy.rego", policy),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("prepare query: %w", err)
	}

	return &Authorizer{query: query}, nil
}

// Must returns a new authorizer, or panics if there is an error.
func Must() *Authorizer {
	auth, err := NewAuthorizer()
	if err != nil {
		panic(err)
	}
	return auth
}

// SkipAuthorization must be used by service methods that do not need an
// authorization check.
//
// Please be sure it is appropriate to skip authorization when using this. You
// MUST leave a comment above the use of this function explaining why
// authorization is skipped, starting with `skipauth:`
//
// This will mark the authorization context (if any) as checked without
// performing any authorization check.
func (a *Authorizer) SkipAuthorization(ctx context.Context) {
	// Mark the authorization context as checked (otherwise middleware will
	// error).
	if authctx, ok := authz_ctx.FromContext(ctx); ok {
		authctx.SetChecked()
	}
}

// Authorize checks authorization for the provided object, and action,
// retrieving the subject from the context.
//
// Object type may be dynamic. This method also marks the request authorization
// context as checked, so that we don't return an error at the end of the
// request.
func (a *Authorizer) Authorize(ctx context.Context, object, action interface{}) error {
	// Mark the authorization context as checked (otherwise middleware will
	// error).
	if authctx, ok := authz_ctx.FromContext(ctx); ok {
		authctx.SetChecked()
	}

	subject := UserFromContext(ctx)
	if subject == nil {
		return ForbiddenWithInternal("nil subject always forbidden", subject, object, action)
	}

	// Map subject and object to map[string]interface{} for use in policy evaluation.
	subjectInterface, err := jsonToInterface(subject)
	if err != nil {
		return ForbiddenWithInternal("subject to interface: "+err.Error(), subject, object, action)
	}
	objectInterface, err := jsonToInterface(object)
	if err != nil {
		return ForbiddenWithInternal("object to interface: "+err.Error(), subject, object, action)
	}

	// Perform the check via Rego.
	input := map[string]interface{}{
		"subject": subjectInterface,
		"object":  objectInterface,
		"action":  action,
	}
	results, err := a.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return ForbiddenWithInternal("policy evaluation failed: "+err.Error(), subject, object, action)
	}
	if len(results) != 1 {
		return ForbiddenWithInternal(fmt.Sprintf("expected 1 policy result, got %d", len(results)), subject, object, action)
	}
	if results[0].Bindings["allowed"] != true {
		return ForbiddenWithInternal("policy disallows request", subject, object, action)
	}

	return nil
}

// AuthzTyper is the interface that may be implemented to get a `type`
// property added during marshaling for authorization. Any struct that will be
// used as a subject or object in authorization should implement this interface.
type AuthzTyper interface {
	// AuthzType returns the type as a snake_case string.
	AuthzType() string
}

// jsonToInterface turns any type that can be JSON (un)marshaled into an
// map[string]interface{} for evaluation by the OPA engine. Nil is returned as nil.
func jsonToInterface(in interface{}) (interface{}, error) {
	// Special cases for nil and string.
	if in == nil {
		return nil, nil
	}
	if _, ok := in.(string); ok {
		return in, nil
	}

	// Anything that makes it to here should be encodeable as a
	// map[string]interface{} (structs, maps, etc.)
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(in); err != nil {
		return nil, fmt.Errorf("encode input: %w", err)
	}

	d := json.NewDecoder(&buf)
	// Note input numbers must be represented with json.Number according to
	// https://pkg.go.dev/github.com/open-policy-agent/opa/rego#example-Rego.Eval-Input
	d.UseNumber()
	var out map[string]interface{}
	if err := d.Decode(&out); err != nil {
		return nil, fmt.Errorf("decode input: %w", err)
	}

	// Add the `type` property if the AuthzTyper interface is implemented.
	if typer, ok := in.(AuthzTyper); ok {
		out["type"] = typer.AuthzType()
	}

	return out, nil
}

// UserFromContext retrieves a user from the viewer context, returning nil if
// there is no user.
func UserFromContext(ctx context.Context) *fleet.User {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil
	}
	return vc.User
}
