package authz

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	authz_ctx "github.com/fleetdm/fleet/server/contexts/authz"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
)

// Authorizer stores the compiled policy and performs authorization checks.
type Authorizer struct {
	query rego.PreparedEvalQuery
}

// Load the policy from authz.rego in this directory.
//go:embed authz.rego
var module string

// NewAuthorizer creates a new authorizer by compiling the policy embedded in
// authz.rego.
func NewAuthorizer() (*Authorizer, error) {
	ctx := context.Background()
	query, err := rego.New(
		rego.Query("allowed = data.authz.allow"),
		rego.Module("authz.rego", module),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "prepare query")
	}

	return &Authorizer{query: query}, nil
}

// SkipAuthorization must be used by service methods that do not need an
// authorization check. Please be sure it is appropriate to skip authorization
// when using this. You MUST leave a comment above the use of this function
// explaining why authorization is skipped.
//
// This will mark the authorization context (if any) as checked without
// performing any authorization check.
func (a *Authorizer) SkipAuthorization(ctx context.Context) {
	// Mark the authorization context as checked (otherwise middleware will
	// error).
	if authctx, ok := authz_ctx.FromContext(ctx); ok {
		authctx.Checked = true
	}
}

// Authorize checks authorization for the provided subject, object, and action.
//
// Object and action types may be dynamic, while the subject must be a kolide.User.
func (a *Authorizer) Authorize(ctx context.Context, subject *kolide.User, object interface{}, action string) error {
	// Mark the authorization context as checked (otherwise middleware will
	// error).
	if authctx, ok := authz_ctx.FromContext(ctx); ok {
		authctx.Checked = true
	}

	if subject == nil {
		return ForbiddenWithInternal("nil subject always forbidden")
	}

	// Map subject and object to map[string]interface{} for use in policy evaluation.
	subjectInterface, err := jsonToInterface(subject)
	if err != nil {
		return ForbiddenWithInternal("subject to interface: " + err.Error())
	}
	objectInterface, err := jsonToInterface(object)
	if err != nil {
		return ForbiddenWithInternal("object to interface: " + err.Error())
	}

	// Perform the check via Rego.
	input := map[string]interface{}{
		"subject": subjectInterface,
		"object":  objectInterface,
		"action":  action,
	}
	results, err := a.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return ForbiddenWithInternal("policy evaluation failed: " + err.Error())
	}
	if len(results) != 1 {
		return ForbiddenWithInternal(fmt.Sprintf("expected 1 policy result, got %d", len(results)))
	}
	if results[0].Bindings["allowed"] != true {
		return ForbiddenWithInternal("policy disallows request")
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
		return nil, errors.Wrap(err, "encode input")
	}

	d := json.NewDecoder(&buf)
	// Note input numbers must be represented with json.Number according to
	// https://pkg.go.dev/github.com/open-policy-agent/opa/rego#example-Rego.Eval-Input
	d.UseNumber()
	var out map[string]interface{}
	if err := d.Decode(&out); err != nil {
		return nil, errors.Wrap(err, "decode input")
	}

	// Add the `type` property if the AuthzTyper interface is implemented.
	if typer, ok := in.(AuthzTyper); ok {
		out["type"] = typer.AuthzType()
	}

	return out, nil
}
